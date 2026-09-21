# Omoikane — Project Context for AI Agents

## Goal
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
- `make go-test` to verify all Go tests pass (129 tests: 106 handler + 9 middleware + 2 mailer + 3 database + 9 events; need running PostgreSQL; Kafka integration tests skip cleanly when the broker is not reachable)
- `make swagger` regenerates both OpenAPI doc sets via swag (main + audit; run before committing if handler annotations changed)
- Public Swagger UI: `/api/swagger/` (main API) and `/api/audit/swagger/` (audit microservice); nginx `proxy_redirect /swagger/` rewrites the trailing-slash redirect so prefixed URLs resolve
- `make test` for full Playwright suite (desktop + mobile); DB reset twice: before desktop, between desktop and mobile
- `make db-reset` (depends on `up`) for clean DB reset
- `psql -c` needs separate flags per statement (DROP/CREATE in one call fails in transaction)
- DB reset requires `pg_terminate_backend()` before DROP DATABASE (active connections)
- DB reset commands must not be silenced (`2>/dev/null || true` removed) — errors must surface
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
- **Phase 28**: Event SDK & outbox infrastructure — 129/129 Go tests pass (committed `7d8ba20`)
  - `backend/internal/events/`: Kafka-backed `Producer` (CloudEvents → topic, key=subject, `RequiredAcks=RequireAll`), `Consumer` (consumer groups, retry + backoff, DLQ routing with `ce-type` header preservation), `Handler`/`HandlerFunc`, `OutboxEvent` model + `GormOutboxStore` (enqueue inside business tx, `Pending`/`MarkSent`/`MarkAttempt`), `Relay` worker (interval + batch), `EnsureTopics`/`EnsureTopic` idempotent topic provisioning, `Config`/`ConfigFromEnv` (`KAFKA_BROKERS`/`KAFKA_EVENTS_TOPIC`/`KAFKA_DLQ_TOPIC`), `MarshalCloudEvent`/`UnmarshalCloudEvent`
  - Outbox DB ops use the main test DB (`omoikane_test`); outbox migrations not yet wired into the monolith (Phase 29+ services provision their own schema + call `EnsureTopics` at startup)
  - Kafka integration tests (`kafka_integration_test.go`): producer→consumer-group ack, outbox→relay→consumer end-to-end, DLQ on handler failure — skip cleanly when broker unreachable (host `localhost:9092`); `go.mod` adds `github.com/segmentio/kafka-go v0.4.51`
  - Verification: `make go-test` green (129: 106 handler + 9 events + 9 middleware + 3 database + 2 mailer)

## Next Steps
1. **Phase 29** — Wave 1 auth service: `cmd/auth` own DB schema, gateway routes `/api/auth*`, `/api/users*`, `/api-tokens*` → auth; emits `user.registered` behind the outbox (see [PLAN.md](./PLAN.md); `TODO.md` for phase checklist)
2. (After Phase 31) Monolith `cmd/api` retired — every request flows gateway → microservice
3. (Optional, from Phase 15) Wire UndoSnackbar into delete flows for undo-toast UX
4. (Backlog) i18n — see TODO.md Backlog

## Critical Context
- **Gateway route-split** (Phase 27): nginx uses per-service `upstream` blocks + prefix `location`s, ALL still → monolith today. Nginx prefix-location semantics REPLACE the matched prefix with the `proxy_pass` URI — keep the trailing-slash/`$is_args$args` shapes from `service-boundaries.md` §5 when flipping hosts in Phases 29–31. `location = /api/audit-logs` is an exact match (no trailing slash greediness).
- **Kafka**: `kafka` service in compose — `apache/kafka:3.9.0`, single-node KRaft (`KAFKA_PROCESS_ROLES: broker,controller`, `CLUSTER_ID`), auto-creates topics, exposed on `localhost:9092`. Inside compose, other services reach it at `kafka:9092`. The events SDK does NOT rely on broker auto-create for correctness: services call `EnsureTopics` at startup (idempotent) so the first publish can never race topic creation.
- **Events SDK (`github.com/segmentio/kafka-go`)**: producer writes sync (`RequireAll`), `BatchTimeout 50ms`; consumer uses `MaxWait 500ms`, manual commits, at-least-once + DLQ (raw message + `ce-type` header preserved) after `ConsumerMaxRetries`. Outbox = transactional append (enqueue inside the business GORM tx); `Relay` polls `RelayBatchSize`/`RelayInterval` and marks rows sent/failed (`OutboxMaxAttempts`). At-least-once everywhere → **handlers must be idempotent**.
- **Go tests**: green via `make go-test` — 129 tests (106 handler + 9 events + 9 middleware + 2 mailer + 3 database). Phase 26 adds `TestGetAuditLogs_ProxiesToAuditService`; Phase 27 adds envelope round-trip + schema catalog; Phase 28 adds outbox store (3) + Kafka integration tests (3: round-trip ack, outbox→relay→consumer, DLQ) that `t.Skip` if the broker is unreachable. Need running PostgreSQL (`omoikane_test`).
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

### Documentation
- `AGENTS.md`: This file
- `TODO.md`: Phase 28 completed; Phase 29 next; i18n in Backlog
