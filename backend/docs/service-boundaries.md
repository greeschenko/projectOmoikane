# Service Boundary Map (Contract)

Source of truth for the Phase 27+ decomposition. Every route the frontend or an
external client can call is listed with its owner (the microservice process
serving it since Phase 31 retired the monolith `cmd/api`). The **external API
contract** (paths, methods, auth, request/response bodies) is frozen — this
document records ownership only.

Owners:
- **auth** — identity: setup, login/register, users, roles, API tokens, profile/password
- **content** — pages, blog, tags, categories, sitemap/RSS data
- **media** — upload, thumbnails, alt, file serving
- **messages** — broadcast messages, contact form, contacts
- **settings** — site settings, email templates
- **audit** — audit log entries (existing microservice, later event-driven)
- **trash** — cross-cutting soft-delete restore/hard-delete aggregator (see §2)
- **dashboard** — aggregator over services, no direct DB (see §3)

## 1. Route → service table

### Auth (identity)
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/setup/check` | public; setup-state check |
| POST | `/setup` | public; only when no users exist |
| POST | `/auth/login`, `/auth/register`, `/auth/logout` | public (logout = authed no-op) |
| POST | `/auth/forgot-password`, `/auth/reset-password` | public; rate-limited (forgot) |
| GET/POST | `/settings/profile`, PUT `/settings/profile` | authed; **user identity** → auth (domain over prefix) |
| POST | `/settings/password` | authed; identity → auth |
| GET/POST/PUT/DELETE | `/users*`, `/users/batch` | admin |
| GET/POST/DELETE | `/api-tokens*` | admin; headless CMS credentials |

### Content
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/pages`, `/pages/slug/{slug}`, `/pages/{id}` | public reads (cache-tier) |
| POST/PUT/DELETE | `/pages*`, `/pages/batch`, `/pages/reorder` | authed writes |
| GET | `/blog/posts`, `/blog/posts/slug/{slug}`, `/blog/posts/{id}`, `/admin/blog/posts` | public + admin reads |
| POST/PUT/DELETE | `/blog/posts*`, `/blog/posts/{id}/like`, `/blog/posts/batch` | authed writes |
| GET/POST/DELETE | `/blog/tags*`, `/blog/categories*` | public reads, admin writes |

### Media
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/media/file/{filename}` | public; CDN-ready immutable + ETag |
| GET/POST | `/media`, `/media/batch` | authed |
| GET/PUT/DELETE | `/media/{id}` | authed |

### Messages
| Method(s) | Path | Notes |
|---|---|---|
| POST | `/contact` | public; ReCAPTCHA |
| GET | `/contacts`, `/contacts/{id}`, POST `/contacts/{id}/read`, DELETE `/contacts/{id}` | admin |
| GET | `/messages`, `/messages/{id}` | authed |
| POST | `/messages`, `/messages/{id}/read`, `/messages/read-all` | admin create / authed read |
| DELETE | `/messages`, `/messages/{id}` | admin |

### Settings (site)
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/settings` | public read (cache-tier); site name/tagline/logo/favicon/email templates |
| PUT | `/settings` | admin |

### Audit
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/audit-logs` | admin; routes straight to the audit microservice (Phase 31) |

### Trash — cross-cutting (§2)
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/trash`, `/trash/count` | admin |
| POST | `/trash/{entity}/{id}/restore` | admin; entity ∈ {page, user, post, media, contact, message, tag, category} |
| DELETE | `/trash/{entity}/{id}`, `/trash` | admin; hard-delete (+ disk cleanup for media) |

### Dashboard — aggregator (§3)
| Method(s) | Path | Notes |
|---|---|---|
| GET | `/dashboard`, `/dashboard/stats` | admin; aggregator over services |

## 2. Trash: cross-cutting ownership decision

Trash spans every entity type while exposing **one path shape** (entity encoded in
the path, not the prefix). A gateway cannot prefix-split it. Chosen design:

- A thin **`trash` service** owns `GET/POST/DELETE /trash*` — it queries each
  owning service's soft-deleted rows via **internal service APIs** (not shared DB).
- Each owning service (auth, content, media, messages) keeps its own
  soft-delete columns and exposes an internal endpoint for trash listing/restore/hard-delete.
- Media hard-delete keeps doing disk cleanup (in media service).

**Status (Phase 31): implemented.** `trash-service` (8087) owns `/trash*`
(admin-gated with its own JWT validation), fans out to each owner's
`/internal/trash*` with the shared internal token, sums counts, routes
restore/hard-delete by entity→owner, and supports `DELETE /trash?entity=` for a
single entity. The per-service internal endpoints are scoped by
`Handler.TrashEntities` (400 on foreign entities). No shared-DB reads remain.

## 3. Dashboard: aggregator decision

`/dashboard` and `/dashboard/stats` read across **users, content, media, messages**
(existing handlers already aggregate in memory). After decomposition the dashboard
service is a **facade**: it calls each service's internal stats endpoint.
Owned by **dashboard** (own deployment unit).

**Status (Phase 31): implemented.** `dashboard-service` (8088) owns `/dashboard*`
(admin-gated), fetches each owner's `GET /internal/stats` with the shared internal
token, and merges them into the exact Phase 26 JSON shapes
(`users/pages/posts/media/messages` counts; `recentRegistrations` = 7-day
zero-filled chart from auth; `recentMessages` = last-5 from messages; dashboard
keys `userCount/pageCount/blogCount/mediaCount/recentMessages/recentRegistrations`).
Unreachable owners fail loud (502). No shared-DB reads remain.

## 4. Cross-service invariants

- Authentication (JWT cookie + `Authorization: Bearer`) is validated **in every
  service** via the shared `internal/middleware` + `internal/auth` packages —
  the gateway never terminates auth. (Trash + dashboard facades validate the
  admin JWT locally; Bearer API tokens are NOT accepted there — the accepted
  trade-off of a fully stateless facade.)
- Service-to-service calls (trash aggregation, dashboard stats, audit-logs
  strictness) authenticate with the shared `INTERNAL_TOKEN` via the
  `X-Internal-Token` header and `middleware.InternalAuth`; these `/internal/*`
  endpoints are never exposed through the gateway.
- Public reads are cached per service through the **shared Redis**
  (`CacheRead`, 30s TTL): content, auth, and settings mounted it; a `flushCache()`
  (FlushDB) in any service invalidates every other service's SSR/cache tier.
- Audit events are **written by the owning service**: each service appends
  domain events to its transactional outbox, the relay publishes them to the
  Kafka backbone (`omoikane.events`), and the **audit-service consumes them**
  (group `audit`) into its `AuditLog` table — no HTTP audit path exists since
  Phase 32. `GET /api/audit-logs` hits the audit-service directly (Phase 31).
  See [`events.md`](./events.md) for the full event catalog.

## 5. Gateway routing plan (nginx)

| Location | Target after decomposition (all live as of Phase 31) |
|---|---|
| `/api/audit/` | audit-service:8081 (kept for the public Swagger UI) |
| `/api/auth/...`, `/api/users*`, `/api/api-tokens*`, `/api/setup*`, `/api/settings/profile`, `/api/settings/password` | **auth-service:8082** (Phase 29) |
| `/api/pages*`, `/api/admin/blog/`, `/api/blog*` | **content-service:8083** (Phase 30) |
| `/api/media*`, `/media/` (file serving) | **media-service:8084** (Phase 30) |
| `/api/contact*`, `/api/contacts*`, `/api/messages*` | **messages-service:8085** (Phase 31) |
| `/api/settings`, `/api/settings/*` (site) | **settings-service:8086** (Phase 31) — exact-match `location = /api/settings` so `/api/settings/profile` stays with auth |
| `/api/trash*` (all entities) | **trash-service:8087** (Phase 31) — one prefix owner (§2) |
| `/api/dashboard*` | **dashboard-service:8088** (Phase 31) |
| `/api/swagger/` | **docs-service:8089** (Phase 31, monolith docs UI) |
| `GET /api/audit-logs` | **audit-service:8081** directly (Phase 31; was monolith proxy) |
| `/api/*` (unmapped) | **`return 404` fail-loud** — unmapped routes surface instead of hitting a dead monolith |

**Phase 31 status (Wave 3 — decomposition complete):** the monolith `cmd/api`
is **retired**. Every route flows gateway → its owning service. New services:
`messages-service` (8085, messages + contacts + outbox), `settings-service`
(8086, site settings + shared-Redis cache),
`trash-service` (8087, aggregator, NO DB),
`dashboard-service` (8088, facade, NO DB),
`docs-service` (8089, main Swagger UI after `cmd/api` removal).
All of auth/content/media/messages/settings wire the events outbox
(producer + relay + `EnsureTopics`, log-only on failure); no new emissions in
this phase. `npm install` note: the frontend container is `API_URL=http://nginx`
and enriches with `/api` — SSR now goes through the gateway too.

**Internal-API layer (Phase 31):** owning services expose token-gated internal
endpoints consumed by the trash + dashboard aggregators — `X-Internal-Token`
header (shared `INTERNAL_TOKEN` env from the platform compose), validated by
`middleware.InternalAuth`, NEVER exposed via the gateway:
- `GET /internal/stats` — per-service dashboard slice (auth: users +
  recentRegistrations; content: pages/posts; media: media; messages: messages +
  recentMessages)
- `GET /internal/trash`, `GET /internal/trash/count`,
  `POST /internal/trash/{entity}/{id}/restore`,
  `DELETE /internal/trash/{entity}/{id}`, `DELETE /internal/trash[?entity=]` —
  scoped to the entities the service owns (`TrashEntities`):
  auth=`user`, content=`page,post,tag,category`, media=`media`,
  messages=`contact,message`. Foreign entities return 400 locally.

**Phase 29 status (Wave 1):** `auth_service` upstream → `auth-service:8082`.

**Phase 30 status (Wave 2):** `content_service` → `content-service:8083`
(pages + blog), `media_service` → `media-service:8084` (media CRUD + file
serving). The `/media/` rich-text location flips to media-service (full path,
no URI rewrite).

**Gateway note — `/api/admin/blog/` needs its own location:** `/api/admin/blog/posts`
does NOT match the trailing-slash `location /api/blog/` prefix (it fell to
`location /api/` → monolith). Phase 30 adds `location /api/admin/blog/` mapping to
`content_service/admin/blog/`.

**Wave 2 store split — process split, shared store:** like auth-service, both
content-service and media-service are separate processes over the same Postgres
`omoikane` store and the shared uploads disk (`../backend:/app` bind in every
container → `backend/uploads`). Physical partition stays deferred (Phase 32+).
Events stay single-writer: `content-service`
emits `page.published`/`post.published`, `media-service` emits `media.uploaded`
(no other service wires the content/media outbox). Content-service also wires the
shared Redis (`REDIS_URL=redis://redis:6379/0`) for CacheRead + flushCache (FlushDB).

**Wave 1 store split — process split, shared store:** `auth-service` is its own
process but connects to the same Postgres `omoikane` store. Since Phase 31 the
trash/dashboard aggregators read EVERYTHING via internal APIs — the auth store
is only touched by auth-service and the other owners. Events are single-writer:
only `auth-service` wires the outbox (`user.registered`, Phase 31 adds Redis for
CacheRead).

**Gateway rule — static `proxy_pass` URIs only:** every `proxy_pass` in the
gateway must be a STATIC URI. A `proxy_pass` containing a variable (e.g.
`$is_args$args`) is passed upstream verbatim and DROPS the location-remainder
path segments (`/api/users/5` would arrive as `/users`). With a static URI the
prefix is replaced correctly and the query string is forwarded automatically.
This was hit during the Phase 29 gateway flip and fixed for every location.

**Events broker (compose):** Kafka runs a dual-listener setup — external
`PLAINTEXT` advertised as `localhost:9092` (host-side `make go-test`
integration tests + kafka CLI), and internal `PLAINTEXT_INTERNAL` advertised as
`kafka:29092` (services inside the compose network). In-compose services MUST
use `KAFKA_BROKERS=kafka:29092`; `localhost:9092` is unreachable from inside
containers (the advertised address resolves to the container itself).

## 6. Compliance check

- [x] Every route in the table is served by its owning service — monolith `cmd/api` retired (Phase 31)
- [x] No route listed more than once under a non-cross-cutting owner
- [x] Auth invariants (§4) hold after each service extraction
- [x] Full Go + Playwright suites green on the frozen contract (Phase 31 gate)
- [x] Audit rows arrive via the Kafka backbone, not HTTP (Phase 32 gate: post
      publish → `action=publish` row through the audit consumer)

## 7. Event catalog

Live domain events, producers, and consumers are documented in
[`events.md`](./events.md). In short: the seven live event types
(`user.registered`, `auth.login`, `page.published`, `post.published`,
`media.uploaded`, `contact.received`, `settings.updated`) are emitted by their
owning services via the transactional outbox → relay → `omoikane.events`;
`cmd/audit` consumes group `audit` (start `kafka.LastOffset`,
idempotent on `AuditLog.EventID`) and maps them to audit rows. `page.updated`,
`post.updated`, and `message.created` are schema-frozen but not yet emitted.

## 8. Observability & ops (Phase 34)

Structured JSON logs, Prometheus `/metrics` per service (never gateway-exposed),
probes, migration Jobs, secrets via `secretKeyRef`, HPA manifests (default off),
and resource requests/limits are documented in
[`observability.md`](./observability.md). Every service registers `GET /metrics`
inside its `newXxxMux` so cmd tests can assert it without booting a server; the
production server wraps the mux in `observability.Middleware`.