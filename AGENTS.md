# Omoikane — Project Context for AI Agents

## Goal
- Phase 30 (Wave 2 content + media services; gateway flip; page.published/post.published/media.uploaded outbox) — DONE
- Phase 29 (Wave 1 auth service; gateway flip + user.registered outbox) — DONE
- Phase 28 (event SDK & outbox infrastructure) — DONE
- Phase 27 (platform blueprint & contract freeze) — DONE
- Phase 26 (fixes from manual review — 15 issues) — DONE
- Phase 25 (manual system review) — DONE
- Phase 24 (accessibility) — DONE
- Phase 23 (CDN-ready media delivery) — DONE
- Phase 22 (image optimization) — DONE
- Phase 21 (Redis cache) — DONE
- Phase 20 (API tokens / headless CMS) — DONE
- Phase 19 (OpenAPI docs + public Swagger UI) — DONE

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (138 tests: 110 handler + 10 events + 9 middleware + 2 mailer + 3 database + 4 auth-service; need running PostgreSQL; Kafka integration tests skip cleanly when the broker is not reachable)
- `make swagger` regenerates both OpenAPI doc sets via swag (main + audit; run before committing if handler annotations changed)
- Public Swagger UI: `/api/swagger/` (main API) and `/api/audit/swagger/` (audit microservice); nginx `proxy_redirect /swagger/` rewrites the trailing-slash redirect so prefixed URLs resolve
- `make test` for full Playwright suite (desktop + mobile); DB reset twice: before desktop, between desktop and mobile
- `make db-reset` (depends on `up`) for clean DB reset
- `psql -c` needs separate flags per statement (DROP/CREATE in one call fails in transaction)
- DB reset requires `pg_terminate_backend()` before DROP DATABASE (active connections)
- DB reset commands must not be silenced (`2>/dev/null || true` removed) — errors must surface
- `db-reset` also stops/restarts **auth-service** (shares `omoikane` DB) and waits for `http://localhost:8082/health`; `make up` waits for auth-service readiness too
- nginx proxies `/api/*` → Go:8080
- Database must be reset before each clean Playwright run
- `.next-root-owned/` added to frontend `.gitignore` (Next.js 16 cache)
- Mobile Playwright has `actionTimeout: 15000` in config
- `loginAsAdmin` uses `domcontentloaded` instead of `networkidle`, deletes all existing pages via API before re-creating 2 test pages — this populates trash with soft-deleted pages
- Docker backend uses Air hot-reload; Go source changes are picked up automatically in the running container
- Docker frontend requires restart + `npm install` when adding new npm packages
- Full `npm install` in Makefile (was selective — caused missing `react-google-recaptcha`)

## Progress
### Done
- **Phase 10**: All Go API response shapes aligned, 27 dead API routes deleted
- **Phase 11**: Breadcrumb + draft visibility — 231/231 desktop pass
- **Phase 12**: Email integration (SMTP + password reset) + ReCAPTCHA v2
- **Phase 13a**: Email templates (customizable via admin UI)
- **Phase 13b**: Rate limiting on forgot-password (3 req/15min per IP)
- **Phase 13c**: Contact form with ReCAPTCHA — public POST + admin CRUD
- **Mobile fix round 1**: 186→29 failures (201 pass), desktop still 231/231
- **Phase 14**: All 230 mobile tests pass — 29 pre-existing failures eliminated
- **Phase 15**: Trash system + bulk actions — 249/249 Playwright + 82/82 Go tests pass
- **Phase 16**: Manual testing session completed — all items verified
- **Phase 17**: Bug fixes + feature completion — 249/249 Playwright + 88/88 Go tests pass
  - Rich text editor enhanced: toolbar (16 buttons), ProseMirror CSS, `minimal` prop, 300px height
  - `@tiptap/extension-placeholder` installed
  - Public header: removed Blog/Contact duplication
  - Contact form: client-side email validation
  - Avatar refresh: `avatar-changed` custom event
  - Pages admin: status/menu badges, indentation reduced
  - Blog: tag/category selectors in post form, sidebar sub-items removed, standalone pages deleted
  - `DELETE /blog/categories/{id}` route wired
  - `UpdatePost` now handles tags + categoryId
  - `BlogPost` model: added `Tags []Tag` with `many2many`
  - Dashboard: `recentRegistrations` + `recentMessages` return real data
  - Media: multiupload
  - Trash: badge refresh after restore/hard-delete via `trash-changed` event
  - Messages: loading spinner
  - Blog: like/unlike button with heart icon
- **Phase 20**: API tokens / headless CMS — 107/107 Go tests pass (committed `a38e65c`)
  - `ApiToken` model (sha256-hashed at rest), `Authorization: Bearer` auth via middleware, `/api-tokens` CRUD, admin UI page, 7 Go tests
- **Phase 21**: Redis cache — 111/111 Go tests pass (committed `a312894`)
  - `internal/cache` (redis + noop fallback), `CacheRead` middleware with `X-Cache` headers + auth bypass, 5 public GETs cached, `flushCache()` on 22 mutating handlers, redis service in compose
- **Phase 22**: Image optimization — 119/119 Go tests pass (committed `009c157`)
  - Go-native `github.com/disintegration/imaging` thumbnails (640px, JPEG q80, `_thumb` suffix), `MediaItem.ThumbPath`/`Alt`, `PUT /media/{id}` alt edit, editor image insert via `url || data` + alt prompt (plain `<img>`, NOT `next/image` — per decision)
- **Phase 23**: CDN-ready media delivery — 123/123 Go tests pass (committed `46cc014`)
  - Public `GET /media/file/{filename}` (path-traversal safe, `Cache-Control: immutable` + ETag/304), `MEDIA_BASE_URL` config → absolute CDN URLs, nginx `/media/` location with long cache
  - HMAC-signed URLs deferred (stored rich-text `<img>` URLs would expire)
- **Phase 24**: Accessibility — 0 critical/serious axe violations on 12 scanned routes (desktop + mobile) (committed)
  - `@axe-core/playwright` + `e2e/29-accessibility.spec.ts`; skip-to-content link, `main` landmarks, mobile-menu `aria-expanded/controls`, named spinners/checkboxes, keyboard move-up/down for page reorder, `:focus-visible` rings, `prefers-reduced-motion`
- **Phase 25**: Manual system review — complete regression baseline + 15 issues catalogued (fix plan in Phase 26)
  - Pre-review regression: Go 123/123, desktop 272/272, mobile 272/272 (committed `a9a9e40`; e2e flake fixes: media alt/caption selectors, API-token extraction, `waitForHydration` in `frontend/e2e/helpers.ts`)
  - Issues found: dashboard key warning (backend shape mismatch), dialog width/2-col form (pages + blog), editor align buttons, media-dialog upload/auto-insert, tree indentation, HTML5 DnD reorder broken, frontend edit form parity, tags/categories not surfaced, richer blog list, menu icons, audit-logs empty (writes→audit DB, reads←main DB), API-token explainer, favicon never applied
- **Phase 26**: 15 manual-review fixes — DONE (committed)
  - Backend: dashboard `GetDashboardStats` returns zero-filled last-7-days `[{date,count}]` for registrations + messages; `GetAuditLogs` proxies to the audit microservice (`AuditServiceURL + /logs`) with local-DB fallback; new `TestGetAuditLogs_ProxiesToAuditService` (fake service)
  - Frontend: pages + blog create/edit dialogs → `maxWidth="lg"` with 70/30 editor-left 2-col layout; RichTextEditor Align Left/Center/Right/Justify buttons (`@tiptap/extension-text-align`) + media-dialog upload with auto-insert; pages tree tightened (depth*12, zero margins); HTML5 DnD replaced with pointer-event drag from the handle (window listeners attached in `pointerdown`, `data-page-id` hit-testing, numeric-id parse fix); public blog category filter + tag/category chips on list & detail; richer admin blog rows (author/dates/likes/category/tags); Contacts icon → `ContactMailIcon`; api-tokens explainer panel; new `FaviconLoader` renders the settings favicon
  - Verification: `make go-test` green; full `make test` (desktop + mobile) passes with new e2e coverage (blog chips/filter, editor align buttons, pointer-drag reorder, media upload/insert, favicon)
- **Phase 27**: Platform blueprint & contract freeze — DONE (committed `c4ca4c1`)
  - `backend/docs/service-boundaries.md`: authoritative route→service table (auth/content/media/messages/settings/audit/trash/dashboard) + cross-cutting decisions (trash = thin aggregator service since the entity lives in the path; dashboard = aggregator facade)
  - `backend/internal/events/`: CloudEvents 1.0 envelope Go types + Type/Source constants + JSON Schema catalog (`schemas/*.json`, 10 event types incl. user.registered, post.published, media.uploaded, contact.received) + 3 new Go tests (envelope round-trip, catalog JSON validity, catalog completeness vs type constants) — 126 Go tests total
  - docker-compose: `kafka` service (apache/kafka:3.9.0, single-node KRaft, `kafka-data` volume, healthcheck)
  - nginx gateway: route-split blueprint — per-service `upstream` blocks + `location` blocks, ALL still → monolith (contract unchanged); Phases 29–31 flip `proxy_pass` hosts. Nginx prefix→URI semantics preserve `/api`-stripping (`/api/trash/page/5/restore` → `/trash/page/5/restore`)
  - Verification: `make go-test` green (126); spot-checked every route group through the gateway (401 on protected, 200 on public, swagger UI + audit swagger reachable); full `make test` gate green
- **Phase 28**: Event SDK & outbox infrastructure — 129/129 Go tests pass (committed `b345a83`)
  - `backend/internal/events/`: Kafka-backed `Producer` (CloudEvents → topic, key=subject, `RequiredAcks=RequireAll`), `Consumer` (consumer groups, retry + backoff, DLQ routing with `ce-type` header preservation), `Handler`/`HandlerFunc`, `OutboxEvent` model + `GormOutboxStore` (enqueue inside business tx, `Pending`/`MarkSent`/`MarkAttempt`), `Relay` worker (interval + batch), `EnsureTopics`/`EnsureTopic` idempotent topic provisioning, `Config`/`ConfigFromEnv` (`KAFKA_BROKERS`/`KAFKA_EVENTS_TOPIC`/`KAFKA_DLQ_TOPIC`), `MarshalCloudEvent`/`UnmarshalCloudEvent`
  - Outbox DB ops use the main test DB (`omoikane_test`); outbox migrations not yet wired into the monolith (Phase 29+ services provision their own schema + call `EnsureTopics` at startup)
  - Kafka integration tests (`kafka_integration_test.go`): producer→consumer-group ack, outbox→relay→consumer end-to-end, DLQ on handler failure — skip cleanly when broker unreachable (host `localhost:9092`); `go.mod` adds `github.com/segmentio/kafka-go v0.4.51`
  - Verification: `make go-test` green (129: 106 handler + 9 events + 9 middleware + 3 database + 2 mailer)
- **Phase 29**: Wave 1 auth service — 138/138 Go tests + gateway flip (committed locally `551835a`; full `make test` gate green: desktop 276/276, mobile 275/275)
  - `backend/cmd/auth/`: own binary on `AUTH_PORT` (8082), **process split, shared store** (same `omoikane` Postgres; physical partition deferred to Phase 31). Migrates User/ApiToken/PasswordResetToken/SiteSetting + `events.MigrateOutbox`; `newAuthMux(h)` registers paths WITHOUT `/api` (nginx strips the prefix); wires the outbox (`Outbox: GormOutboxStore`) + producer + `EnsureTopics` (log-only on failure — Kafka down never fatals the service) + relay goroutine with signal-ctx graceful shutdown
  - Emission: Setup/Register/CreateUser now go through `createUserAndEmit` — plain `Create` when `Handler.Outbox == nil` (monolith, single-writer: no double emission), else DB-tx-wrapped create + `enqueueUserRegistered` (`events.NewEventID()`, SourceAuth/TypeUserRegistered, subject `user/{id}`, payload `{id,email,role}` per `schemas/user.registered.json`)
  - Gateway flip: `upstream auth_service` → `auth-service:8082`; nginx locations for setup/auth/users/api-tokens/settings-profile|password now hit auth-service. **Fixed latent Phase 27 nginx bug: `proxy_pass` URIs with `$is_args$args` variables DROP location-remainder segments (`/api/users/5` → `/users`); all locations rewritten to STATIC URIs (query strings pass through automatically)**
  - Kafka compose: dual-listener (external `localhost:9092` for host tests, internal `kafka:29092` advertised for in-network services; auth-service uses `KAFKA_BROKERS=kafka:29092`)
  - Makefile: `go-test`/`test` scope includes `./cmd/auth/...`; `go-build` emits `bin/auth`; `up`/`db-reset` await `:8082/health`
  - Verification: `make go-test` green (138: 110 handler + 10 events + 4 cmd/auth + 9 middleware + 3 database + 2 mailer); gateway smoke (setup→login→users CRUD→batch→api-token revoke→setup/check), outbox row → relay → Kafka topic confirmed (`subject=user/3`); host Kafka integration tests still pass; full `make test` gate green
- **Phase 30**: Wave 2 content + media services — 158/158 Go tests + gateway flip (committed locally; full `make test` gate green: desktop 276/276, mobile 275/275)
  - `backend/cmd/content/`: own binary on `CONTENT_PORT` (8083), **process split, shared store** (same `omoikane` Postgres as monolith/auth). Migrates Page/BlogPost/Tag/BlogPostTag/Category/Like + outbox; `newContentMux(h)` registers paths WITHOUT `/api` (nginx strips prefix), full content route set incl. `GET /health`, CacheRead on public GETs (pages, blog lists/detail, tags, categories), admin routes, `POST /blog/posts/{id}/like`; wires shared Redis (`REDIS_URL=redis://redis:6379/0`) so `flushCache()` FlushDB invalidates monolith/SRR cache too; producer + `EnsureTopics` (log-only on failure) + relay goroutine w/ graceful shutdown
  - `backend/cmd/media/`: own binary on `MEDIA_PORT` (8084), same shared store + shared uploads disk (`../backend:/app` bind in every container → `backend/uploads`); migrates MediaItem + outbox; `newMediaMux(h)` = `GET /health`, public `GET /media/file/{filename}` (ServeMediaFile), `GET /media`, `POST /media`, `GET/PUT/DELETE /media/{id}`, `POST /media/batch`; `UPLOAD_DIR`/`MEDIA_BASE_URL` passthrough; producer + relay. NO CacheRead needed (all media endpoints Auth, handlers don't flushCache)
  - Emission (content): `createPageAndEmit`/`updatePageAndEmit`/`publishPagesAndEmit` → `enqueuePagePublished` (SourceContent/TypePagePublished, subject `page/{id}`, `{id,slug,title,status:"published",publishedAt:now}`); posts: `createPostAndEmit`/`updatePostAndEmit`/`publishPostsAndEmit` → `enqueuePostPublished` (subject `post/{id}`, `{id,slug,title,status,categoryId,tagIds,publishedAt}`); tags joined from pre-existing names (CreatePost no longer joins tags inline — moved into `associatePostTags` helper called inside the tx). Only on transition to published: create-as-published, draft→published update, batch publish of previously-non-published rows. Emission (media): `createMediaAndEmit` → `enqueueMediaUploaded` (SourceMedia/TypeMediaUploaded, subject `media/{id}`, `{id,filename,alt,url,thumbUrl,size,uploadedAt}`)
  - Gateway flip: `content_service` → `content-service:8083`, `media_service` → `media-service:8084`; locations for pages/blog/media hit the services, `/media/` file-serving location → media-service (static proxy_pass, full path preserved). **NEW `location /api/admin/blog/` → `content_service/admin/blog/` — the trailing-slash `/api/blog/` prefix does NOT match `/api/admin/blog/posts` (it previously fell to `/api/` → monolith)**
  - Makefile: `go-test`/`test` scope += `./cmd/content/... ./cmd/media/...`; `go-build` += `bin/content` + `bin/media`; `up` awaits `:8083/health` + `:8084/health`; `db-reset` stops/starts content-service + media-service (share omoikane DB)
  - Tests (20 new): 11 handler outbox tests (`content_outbox_test.go` — create/update/batch page+post emission, transition-only, already-published no-dup, tags→tagIds payload, media upload emission, nil-outbox no-event) + 5 `cmd/content/main_test.go` (health, public reads unauthed, page+post→outbox→relay→stub-publish, protected routes 401) + 4 `cmd/media/main_test.go` (health, upload→outbox→relay→stub-publish, public file serving 404-not-401, protected routes 401); service tests mint JWTs via `auth.GenerateToken` as `session` cookie (service muxes have no `/auth/login` route)
  - Verification: `make go-test` green (158: 121 handler + 10 events + 4 cmd/auth + 5 cmd/content + 4 cmd/media + 9 middleware + 3 database + 2 mailer); Kafka integration tests run against real compose broker (no skips); gateway smoke: routes verified landing on the right services (content/media 401/403 logs match), `X-Cache: hit` from shared Redis, real outbox→relay→Kafka events captured for user.registered, page.published (`subject=page/459`), media.uploaded (`subject=media/5`), `/media/file/...` served through gateway with CDN headers; full `make test` gate green

## Next Steps
1. **Phase 31** — Wave 3 services: messages + settings (`cmd/messages`, `cmd/settings`); gateway flips `/api/contact*`, `/api/contacts*`, `/api/messages*`, `/api/settings*`; trash aggregator service + dashboard facade; monolith `cmd/api` retired (see [PLAN.md](./PLAN.md); `TODO.md` for phase checklist)
2. (Optional, from Phase 15) Wire UndoSnackbar into delete flows for undo-toast UX
3. (Backlog) i18n — see TODO.md Backlog

## Critical Context
- **Gateway route-split** (Phase 27 blueprint, Phases 29–30 flip): nginx uses per-service `upstream` blocks + prefix `location`s; `auth_service` → `auth-service:8082`, `content_service` → `content-service:8083`, `media_service` → `media-service:8084`, everything else the monolith. **`proxy_pass` URIs MUST be STATIC — variables (`$is_args$args`, `$request_uri`) drop the location-remainder segments (`/api/users/5` → `/users`)**; with a static URI the prefix is replaced correctly and query strings pass through automatically. `location = /api/audit-logs` is an exact match (no trailing slash greediness). **`/api/admin/blog/` needs its own location** — the trailing-slash `/api/blog/` prefix does NOT match `/api/admin/blog/posts` (previously fell to `/api/` → monolith). `/media/` file-serving location flips to `media_service` (static proxy_pass without URI keeps the full path).
- **Kafka**: `kafka` service in compose — `apache/kafka:3.9.0`, single-node KRaft, dual-listener: external `PLAINTEXT` advertised `localhost:9092` (host-side tools + integration tests) and internal `PLAINTEXT_INTERNAL` advertised `kafka:29092` (in-compose services). Services MUST set `KAFKA_BROKERS=kafka:29092` — `localhost:9092` from inside a container resolves to the container itself and is refused. The events SDK does NOT rely on broker auto-create for correctness: services call `EnsureTopics` at startup (idempotent) so the first publish can never race topic creation.
- **Events SDK (`github.com/segmentio/kafka-go`)**: producer writes sync (`RequireAll`), `BatchTimeout 50ms`; consumer uses `MaxWait 500ms`, manual commits, at-least-once + DLQ (raw message + `ce-type` header preserved) after `ConsumerMaxRetries`. Outbox = transactional append (enqueue inside the business GORM tx); `Relay` polls `RelayBatchSize`/`RelayInterval` and marks rows sent/failed (`OutboxMaxAttempts`). At-least-once everywhere → **handlers must be idempotent**.
- **Go tests**: green via `make go-test` — 158 tests (121 handler + 10 events + 4 cmd/auth + 5 cmd/content + 4 cmd/media + 9 middleware + 2 mailer + 3 database). Phase 26 adds `TestGetAuditLogs_ProxiesToAuditService`; Phase 27 adds envelope round-trip + schema catalog; Phase 28 adds outbox store (3) + Kafka integration tests (3: round-trip ack, outbox→relay→consumer, DLQ) that `t.Skip` if the broker is unreachable; Phase 29 adds `TestNewEventID`, 4 handler outbox-enqueue tests and 4 `cmd/auth` service tests; Phase 30 adds 11 handler outbox tests (`content_outbox_test.go`: page/post/media create/update/batch emission + transition-only + nil-outbox no-event) + 5 `cmd/content` service tests (health, public reads, page+post→outbox→relay→stub-publish, protected 401) + 4 `cmd/media` service tests (health, upload→outbox→relay→stub-publish, public file serving, protected 401). Need running PostgreSQL (`omoikane_test`); service tests mint JWTs via `auth.GenerateToken` as the `session` cookie.
- **Desktop Playwright**: 276/276 pass, 8 skipped — 0 failures (Phase 26 adds 4 tests: drag reorder, blog chips/filter, blog detail chips, api-tokens hydration-race guard; Phase 24 added a11y spec)
- **Mobile Playwright**: 275/275 pass, 9 skipped — 0 failures (Phase 26 adds drag reorder, blog chips/filter, blog detail chips; Phase 24 added a11y spec)
- **Test DB connections**: `setupTestDB` caps pool (MaxOpenConns 3) + closes via `t.Cleanup` — prevents "too many clients" with Postgres' default 100-connection limit
- **Media URLs**: `mediaJSON` emits `url`/`thumbUrl` (relative `/media/file/…` or absolute CDN URL when `MEDIA_BASE_URL` set) alongside legacy base64 `data`
- **MUI v9**: `inputProps`/`InputProps` renamed → use `slotProps.input` on Checkbox; top-level `aria-label` lands on the ROOT span, NOT the native input
- **Hydration race in e2e**: clicking submit/buttons before React hydrates causes a native form GET to `/login?` or a dialog that never opens — `waitForHydration(page, selector)` in `e2e/helpers.ts` waits on React's `__reactProps$*` expando; `networkidle` is NOT reliable (dev-mode websockets)
- **Swagger**: swag v1.16.6 lib + swag CLI at `/home/olex/prodev/go/bin/swag` (GOPATH is `/home/olex/prodev/go`); docs generated with `--parseDependency --parseInternal --exclude` (main API excludes `cmd/audit`; audit service excludes `internal,cmd/api`); main API docs at `backend/docs/`, audit docs at `backend/cmd/audit/docs/`; both use relative `doc.json` URL so the UI works behind the nginx prefix; `@BasePath` must appear BEFORE `@securityDefinitions` or swag drops it; Go 1.22+ mux requires `GET /swagger/` (trailing slash wildcard), NOT `/swagger/*`
- **`GetTrashCount`** queries `Unscoped().Where("deleted_at IS NOT NULL").Count()` across all 8 entity models — used for sidebar badge
- Models with `gorm.Model`: Page, User, BlogPost, MediaItem, ContactMessage, Message, Tag, Category — all support GORM soft-delete via `DeletedAt`
- Trash routes: `GET /trash`, `GET /trash/count`, `POST /trash/{entity}/{id}/restore`, `DELETE /trash/{entity}/{id}`, `DELETE /trash` — all admin-only
- Batch routes: `POST /users/batch`, `POST /pages/batch`, `POST /blog/posts/batch`, `POST /media/batch` — admin/auth-protected
- **Checkbox column** in users table shifts cell indices: `td:nth(2)` is email (was `nth(1)` before checkbox column was added)
- **Bulk toolbar buttons** use exact labels: "Publish", "Draft", "Delete Selected", "Ban", "Activate", "Clear"
- **Contacts page** has both card-level "Delete" button and dialog "Delete" button — use `.first()` or dialog scoping for disambiguation
- **Trash count**: AdminLayout listens for `trash-changed` event (fired by trash page after restore/hard-delete) + polls every 30s
- **Avatar refresh**: AdminAppBar/PublicHeader listen for `avatar-changed` event (fired by SettingsForm after save)
- All models use GORM `DeletedAt` for soft-delete: Page, User, BlogPost, MediaItem, ContactMessage, Message, Tag, Category

## Relevant Files
### Phase 17 backend — Modified
- `backend/internal/handlers/blog.go`: `UpdatePost` now handles tags (clear + re-associate) + categoryId
- `backend/internal/handlers/dashboard.go`: `GetDashboardStats` returns real `recentRegistrations` + `recentMessages` (last 5)
- `backend/internal/models/blog_post.go`: Added `Tags []Tag` field with `gorm:"many2many:blog_post_tags;"`
- `backend/cmd/api/main.go`: Wired `DELETE /blog/categories/{id}` route

### Phase 17 backend — New tests
- `backend/internal/handlers/blog_test.go`: `TestDeleteCategory_AdminDeletes`, `TestDeleteCategory_NonAdminRejected`, `TestUpdatePost_UpdatesTags`, `TestUpdatePost_UpdatesCategory`
- `backend/internal/handlers/pages_test.go`: `TestReorderPages_WithParent_ScopesSiblings`
- `backend/internal/handlers/dashboard_test.go`: `TestDashboardStats_ReturnsRecentData`

### Phase 17 frontend — Modified
- `frontend/components/RichTextEditor.tsx`: 16 toolbar buttons, `minimal` prop, `placeholder` prop, 300px height, ProseMirror-ready
- `frontend/app/globals.css`: ProseMirror styles (headings, lists, blockquote, code, hr, links, placeholder)
- `frontend/components/PublicHeader.tsx`: Removed Blog/Contact buttons + blogEnabled state; added `avatar-changed` listener
- `frontend/components/AdminAppBar.tsx`: Added `avatar-changed` event listener for refresh
- `frontend/components/SettingsForm.tsx`: Dispatches `avatar-changed` event after save
- `frontend/app/(withHeader)/contact/page.tsx`: Client-side email regex validation, removed `noValidate`
- `frontend/app/admin/pages/page.tsx`: Status badges (Chip), Menu badge, indentation `depth*16`
- `frontend/app/admin/blog/page.tsx`: Autocomplete tags + Select category in post form
- `frontend/components/AdminLayout.tsx`: Removed Tags/Categories sidebar items + unused imports; listens for `trash-changed` event
- `frontend/app/admin/media/page.tsx`: Multiupload (multiple file selection, sequential upload)
- `frontend/app/admin/trash/page.tsx`: Dispatches `trash-changed` event after restore/hard-delete
- `frontend/app/admin/messages/page.tsx`: Loading spinner
- `frontend/components/PostDetailClient.tsx`: Like/unlike button with heart icon
- `frontend/app/admin/settings/page.tsx`: RichTextEditor `minimal` prop for email templates

### Phase 17 frontend — Deleted
- `frontend/app/admin/blog/tags/`: Removed (redundant with blog page tabs)
- `frontend/app/admin/blog/categories/`: Removed (redundant with blog page tabs)

### Phase 17 frontend — Test updates
- `frontend/e2e/23-admin-tags-categories.spec.ts`: Rewritten for blog page tabs (not standalone pages)
- `frontend/e2e/08-admin-media.spec.ts`: Updated upload button selectors for multiupload labels
- `frontend/e2e/07-admin-pages.spec.ts`: Title extraction uses `p` element (not textContent which includes chips)
- `frontend/e2e/08-admin-mobile.spec.ts`: Sidebar selector avoids hidden Next.js error overlay nav

### Phase 15 files (still relevant)
- `backend/internal/handlers/trash.go`: Trash endpoints
- `frontend/app/admin/trash/page.tsx`: Trash page
- `frontend/components/UndoSnackbar.tsx`: Shared snackbar (not yet wired)

### Phase 20–24 files
- `backend/internal/models/api_token.go`: `ApiToken` (sha256 `TokenHash`, `ExpiresAt`, `LastUsedAt`)
- `backend/internal/handlers/api_tokens.go`: `GET/POST/DELETE /api-tokens` (admin), one-time raw token on create
- `backend/internal/middleware/auth.go`: `Authenticate` (JWT cookie first, then `Authorization: Bearer` via `TokenStore`); `backend/internal/middleware/cache.go`: `CacheRead` + `isPublicRequest`
- `backend/internal/cache/`: `cache.go`, `redis.go`, `noop.go`; `backend/internal/handlers/handler.go`: `flushCache()`, `MediaBaseURL` field
- `backend/internal/handlers/media.go`: thumbnails (`generateThumbnail`, 640px), `mediaJSON` (`url`/`thumbUrl`/`alt` + legacy `data`), `UpdateMedia` (PUT alt), `ServeMediaFile` (immutable cache + ETag/304), `MEDIA_BASE_URL` handling
- `backend/internal/config/config.go`: `RedisURL`, `MediaBaseURL`; `docker/docker-compose.yml`: redis service, `REDIS_URL`, `MEDIA_BASE_URL`; `docker/nginx/nginx.conf`: `/media/` location
- `frontend/app/admin/api-tokens/page.tsx`, `frontend/app/admin/media/page.tsx` (alt dialog + thumbnails), `frontend/components/RichTextEditor.tsx` (url/data image insert + alt prompt)
- `frontend/e2e/29-accessibility.spec.ts`: axe-core scans (public + admin routes); `frontend/e2e/07-admin-pages.spec.ts`: keyboard move-up/down reorder test
- `frontend/app/layout.tsx` + `frontend/app/(withHeader)/layout.tsx` + `frontend/components/AdminLayout.tsx`: skip link + `main` landmark + nav aria-labels; `frontend/app/globals.css`: `.skip-link`, `:focus-visible`, `prefers-reduced-motion`

### Phase 26 files
- `backend/internal/handlers/dashboard.go`/`dashboard_test.go`: zero-filled last-7-days registration + message counts for chart
- `backend/internal/handlers/audit.go`/`audit_test.go`: `GetAuditLogs` proxies to `AuditServiceURL + "/logs"` (forwarding query); new `TestGetAuditLogs_ProxiesToAuditService` (fake service)
- `frontend/app/admin/pages/page.tsx`: lg 2-col dialog, pointer-event drag reorder with `data-page-id` hit-testing + numeric-id parse fix (`Number()`), depth*12 indentation
- `frontend/app/admin/blog/page.tsx`: lg 70/30 dialog; richer post rows (author/dates/likes/chips)
- `frontend/app/(withHeader)/blog/page.tsx`: category filter dropdown, category + tag chips on list cards
- `frontend/app/(withHeader)/blog/[slug]/page.tsx`: passes `tags` + `categoryId` to `PostDetailClient`
- `frontend/components/PostDetailClient.tsx`: upgraded edit dialog + category/tag chips + edit dialog form parity
- `frontend/components/RichTextEditor.tsx`: Align Left/Center/Right + Justify toolbar buttons (`@tiptap/extension-text-align`), media-dialog upload with auto-insert
- `frontend/components/FaviconLoader.tsx` (new): client component fetching `/api/settings` → injects `<link rel="icon" data-favicon>`
- `frontend/app/layout.tsx`: adds `<FaviconLoader />` inside root layout
- `frontend/components/AdminLayout.tsx`: Contacts nav → `ContactMailIcon`
- `frontend/app/admin/api-tokens/page.tsx`: "What are API tokens?" explainer panel
- `frontend/e2e/07-admin-pages.spec.ts`: pointer-drag reorder test + editor align button assertions
- `frontend/e2e/24-blog-public.spec.ts`: blog chips + category filter e2e

### Phase 27 files
- `backend/docs/service-boundaries.md` (new): authoritative route→service table + cross-cutting decisions (trash aggregator, dashboard facade, auth invariants, gateway plan)
- `backend/internal/events/` (new): `events.go` (CloudEvents envelope + Source/Type constants), `schemas/*.json` (10 JSON Schemas), `events_test.go` (round-trip, catalog validity, completeness)
- `docker/docker-compose.yml`: `kafka` service (apache/kafka:3.9.0 KRaft single-node, advertised `localhost:9092`, `kafka-data` volume, topics healthcheck)
- `docker/nginx/nginx.conf`: route-split gateway blueprint — per-service `upstream` blocks (`auth_service`…`trash_service`) + `location` blocks, all targets still `backend:8080` except `/api/audit/` → `audit-service:8081`

### Phase 28 files
- `backend/internal/events/config.go` (new): `Config` + `ConfigFromEnv` (KAFKA_*), `DefaultTopic`/`DefaultDLQTopic`
- `backend/internal/events/producer.go` (new): `Producer` interface + `KafkaProducer` (`MarshalCloudEvent`/`UnmarshalCloudEvent` shared helpers)
- `backend/internal/events/consumer.go` (new): `Consumer` (consumer groups, retry+backoff, DLQ), `Handler`/`HandlerFunc`
- `backend/internal/events/outbox.go` (new): `OutboxEvent` model + `OutboxStore` interface + `GormOutboxStore` + `MigrateOutbox`
- `backend/internal/events/relay.go` (new): `Relay` worker (`Run`/`RunOnce`, interval + batch)
- `backend/internal/events/admin.go` (new): `EnsureTopic`/`EnsureTopics` idempotent provisioning
- `backend/internal/events/outbox_test.go` (new): 3 outbox store tests (enqueue/pending/mark-sent, attempt exhaustion, marshal round-trip)
- `backend/internal/events/kafka_integration_test.go` (new): 3 Kafka integration tests (round-trip ack, outbox→relay→consumer, DLQ)
- `backend/go.mod`/`go.sum`: added `github.com/segmentio/kafka-go v0.4.51`

### Phase 29 files
- `backend/cmd/auth/main.go` (new): auth-service binary (`AUTH_PORT` 8082, shared `omoikane` store, focused migration + `MigrateOutbox`, `newAuthMux(h)` route wiring WITHOUT `/api` prefix, `EnsureTopics` log-only, producer + relay goroutine, graceful shutdown)
- `backend/cmd/auth/main_test.go` (new): service tests (health, setup-check public, register→outbox→relay→stub-publish, protected routes 401)
- `backend/internal/handlers/user_events.go` (new): `createUserAndEmit` (plain Create when `Outbox == nil`; tx-wrapped create + `enqueueUserRegistered` otherwise) + `enqueueUserRegistered` (CloudEvent SourceAuth/TypeUserRegistered, subject `user/{id}`, `{id,email,role}` payload)
- `backend/internal/handlers/outbox_test.go` (new): 4 tests — Register/Setup/CreateUser enqueue user.registered; nil-outbox path emits nothing
- `backend/internal/handlers/handler.go`: `Handler.Outbox events.OutboxStore` field (nil in monolith — single-writer)
- `backend/internal/handlers/auth.go` / `users.go`: Setup/Register/CreateUser create sites swapped to `createUserAndEmit`
- `backend/internal/events/events.go`/`events_test.go`: `NewEventID()` (crypto/rand 16 bytes hex) + test
- `docker/docker-compose.yml`: `auth-service` service (Dockerfile.dev, `go run ./cmd/auth`, `AUTH_PORT=8082`, `JWT_SECRET` shared, `AUDIT_SERVICE_URL=http://audit-service:8081`, `KAFKA_BROKERS=kafka:29092`, port 8082, healthcheck wget `localhost:8082/health`); nginx `depends_on` + auth-service; kafka dual-listener (`PLAINTEXT_INTERNAL` kafka:29092)
- `docker/nginx/nginx.conf`: `upstream auth_service` → `auth-service:8082`; ALL `proxy_pass` URIs made STATIC (fixed `$is_args$args` segment-dropping bug; affects pages/blog/media/contact/messages/settings/trash/dashboard locations too)
- `Makefile`: `go-test`/`test` scope `./internal/... ./cmd/auth/...`; `go-build` emits `bin/auth`; `up`/`db-reset` await `:8082/health` and restart auth-service
- `backend/docs/service-boundaries.md`: §5 table + Phase 29 status, Wave-1 shared-store note, static-proxy_pass rule, Kafka dual-listener note

### Phase 30 files
- `backend/cmd/content/main.go` (new): content-service binary (`CONTENT_PORT` 8083, shared `omoikane` store, content tables migration + `MigrateOutbox`, `newContentMux(h)` full route set WITHOUT `/api` prefix incl. `GET /health` + `GET /admin/blog/posts`, CacheRead on public reads, shared Redis (`REDIS_URL`), `EnsureTopics` log-only, producer + relay goroutine, graceful shutdown, `getEnv` helper)
- `backend/cmd/content/main_test.go` (new): service tests (health, public reads unauthed, page+post→outbox→relay→stub-publish, protected routes 401; `sessionCookie` mints JWTs via `auth.GenerateToken`)
- `backend/cmd/media/main.go` (new): media-service binary (`MEDIA_PORT` 8084, shared store + shared uploads dir, MediaItem migration + `MigrateOutbox`, `newMediaMux(h)` = health + public `GET /media/file/{filename}` + media CRUD/batch WITHOUT `/api` prefix, `UPLOAD_DIR`/`MEDIA_BASE_URL` passthrough, no CacheRead, producer + relay)
- `backend/cmd/media/main_test.go` (new): service tests (health, upload→outbox→relay→stub-publish, public file serving, protected routes 401)
- `backend/internal/handlers/content_events.go` (new): `createPageAndEmit`/`updatePageAndEmit`/`publishPagesAndEmit` + `createPostAndEmit`/`updatePostAndEmit`/`publishPostsAndEmit` + `associatePostTags` + `enqueuePagePublished`/`enqueuePostPublished` (transition-to-published only; nil-outbox = plain create/update)
- `backend/internal/handlers/media_events.go` (new): `createMediaAndEmit` + `enqueueMediaUploaded` (SourceMedia/TypeMediaUploaded, subject `media/{id}`, `{id,filename,alt,url,thumbUrl,size,uploadedAt}`)
- `backend/internal/handlers/content_outbox_test.go` (new): 11 tests — page/post create/update/batch emission, transition-only + already-published no-dup, tags→tagIds payload, media upload emission, nil-outbox no-event
- `backend/internal/handlers/pages.go` / `blog.go` / `media.go`: CreatePage/UpdatePage/BatchPages, CreatePost/UpdatePost/BatchPosts, UploadMedia swapped to emit-variants
- `docker/docker-compose.yml`: `content-service` (8083) + `media-service` (8084) services mirroring auth-service (`go run ./cmd/content|media`, `KAFKA_BROKERS=kafka:29092`, healthchecks wget `:8083|8084/health`; content also `REDIS_URL`; media also `UPLOAD_DIR=/app/uploads` + `MEDIA_BASE_URL`); nginx `depends_on` both
- `docker/nginx/nginx.conf`: `content_service` → `content-service:8083`, `media_service` → `media-service:8084`; NEW `location /api/admin/blog/` → `content_service/admin/blog/`; `/media/` location → `media_service` (no URI rewrite — full path preserved)
- `Makefile`: `go-test`/`test` scope `./internal/... ./cmd/auth/... ./cmd/content/... ./cmd/media/...`; `go-build` emits `bin/auth|content|media`; `up` awaits `:8083/health` + `:8084/health`; `db-reset` stops/starts content-service + media-service
- `backend/docs/service-boundaries.md`: §5 table + Phase 30 status, `/api/admin/blog/` location note, Wave-2 shared-store + shared-Redis note

### Documentation
- `AGENTS.md`: This file
- `TODO.md`: Phase 30 completed; Phase 31 next; i18n in Backlog
