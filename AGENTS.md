# Omoikane — Project Context for AI Agents

## Goal
- Phase 24 (accessibility) — DONE
- Phase 23 (CDN-ready media delivery) — DONE
- Phase 22 (image optimization) — DONE
- Phase 21 (Redis cache) — DONE
- Phase 20 (API tokens / headless CMS) — DONE
- Phase 19 (OpenAPI docs + public Swagger UI) — DONE

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (123 tests: 109 handler + 9 middleware + 2 mailer + 3 database; need running PostgreSQL)
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

## Next Steps
1. Final regression: full `make test` (desktop + mobile) + `make go-test`
2. (Optional) Wire UndoSnackbar into delete flows for undo-toast UX
3. (Backlog) i18n — see TODO.md Backlog

## Critical Context
- **Go tests**: 123/123 pass (109 handler + 9 middleware + 2 mailer + 3 database; need running PostgreSQL)
- **Desktop Playwright**: baseline 242/242 pass, 8 skipped — 0 failures (Phase 24 added a11y spec)
- **Mobile Playwright**: baseline 249/249 pass, 9 skipped — 0 failures (Phase 24 added a11y spec)
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

### Documentation
- `AGENTS.md`: This file
- `TODO.md`: Phases 20–24 completed; i18n in Backlog
