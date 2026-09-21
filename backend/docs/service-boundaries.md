# Service Boundary Map (Contract)

Source of truth for the Phase 27+ decomposition. Every route the frontend or an
external client can call is listed with its current owner (monolith `cmd/api`)
and its target microservice. The **external API contract** (paths, methods,
auth, request/response bodies) is frozen — this document records ownership only.

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
| GET | `/audit-logs` | admin; **today**: monolith proxies → audit microservice. Target: gateway routes `/api/audit-logs` → audit directly |

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

This is deliberately the *last* boundary wired (Phase 31) since it has the most
dependencies; until then `cmd/api` continues to serve `/trash*` as today.

## 3. Dashboard: aggregator decision

`/dashboard` and `/dashboard/stats` read across **users, content, media, messages**
(existing handlers already aggregate in memory). After decomposition the dashboard
service is a **facade**: it calls each service's internal stats endpoint.
Owned by **dashboard** (own deployment unit) or folded into the gateway service —
decision deferred to Phase 31; contract unchanged meanwhile.

## 4. Cross-service invariants

- Authentication (JWT cookie + `Authorization: Bearer`) is validated **in every
  service** via the shared `internal/middleware` + `internal/auth` packages —
  the gateway never terminates auth.
- Public reads currently carried by the monolith's Redis cache (`CacheRead`,
  30s TTL) move **into each service** (each owns its cache configuration); the
  monolith-level flush calls migrate to per-service flush (Phase 32).
- Audit events are **written by the owning service** (outbox → Kafka → audit)
  starting Phase 32; until then the existing HTTP proxy path stays.

## 5. Gateway routing plan (nginx)

| Location | Target today | Target after decomposition |
|---|---|---|
| `/api/audit/` | audit-service:8081 | quit (audit moves to `/api/audit-logs` route + events) |
| `/api/auth/...`, `/api/users*`, `/api-tokens*`, `/api/settings/profile`, `/api/settings/password` | backend:8080 | auth-service |
| `/api/pages*`, `/api/blog*` | backend:8080 | content-service |
| `/api/media*` | backend:8080 | media-service |
| `/api/contact*`, `/api/contacts*`, `/api/messages*` | backend:8080 | messages-service |
| `/api/settings`, `/api/settings/*` (site) | backend:8080 | settings-service |
| `/api/audit-logs` | backend:8080 (proxy) | audit-service |
| `/api/trash*` (all entities) | backend:8080 | trash-service (Phase 31) — one prefix owner (§2) |
| `/api/dashboard*` | backend:8080 | dashboard (Phase 31) |
| `/api/*` (unmapped) | backend:8080 | 404 or proxy — fail-loud so unmapped routes surface |

**Phase 27 status:** all targets are still `backend:8080` (contract unchanged).
Phases 29–31 flip targets per service, gated by the full Playwright suite.

## 6. Compliance check (Phase 27 gate)

- [ ] Every route in the table is served by today's monolith (no orphan)
- [ ] No route listed more than once under a non-cross-cutting owner
- [ ] Auth invariants (§4) hold after each service extraction
- [ ] Full Go + Playwright suites still green on the frozen contract