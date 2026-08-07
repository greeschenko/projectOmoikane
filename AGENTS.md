# Omoikane — Project Context for AI Agents

## Goal
- Phase 19 (platform & performance) — in progress; item 1 (OpenAPI docs + public Swagger UI) DONE
- Phase 18 (audit log microservice) — DONE
- Phase 17 (bug fixes + feature completion) — DONE

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (100 tests: 95 handler + 2 mailer + 3 middleware)
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

## Next Steps
1. Phase 19, item 2: API tokens / headless CMS mode
2. (Optional) Wire UndoSnackbar into delete flows for undo-toast UX

## Critical Context
- **Go tests**: 100/100 pass (95 handler + 2 mailer + 3 middleware; need running PostgreSQL)
- **Desktop Playwright**: 242/242 pass, 8 skipped — 0 failures
- **Mobile Playwright**: 249/249 pass, 9 skipped — 0 failures
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

### Documentation
- `AGENTS.md`: This file
- `TODO.md`: Phase 17 completed, Phase 18 (audit log) pending
