# Omoikane — Project Context for AI Agents

## Goal
- **Phase 10 complete**: Phase 11 — (next milestone)

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (82 tests, ~12s)
- `make test` for full Playwright suite (desktop + mobile); must reset DB between runs
- nginx proxies `/api/*` → Go:8080
- Database must be reset (`DROP/CREATE` + restart backend) before each clean Playwright run

## Progress
### Done
- **Phase 10 complete**: All Go handler response shapes aligned, 27 dead API routes deleted, 229/231 desktop Playwright tests pass (2 pre-existing failures)
- **Go changes**: blog (bare arrays, `count`, `tags []string`, `categoryId`, `authorName`), pages (bare array, auth-aware drafts), media (bare array + base64 data URIs, wrapped upload), messages (`readBy`, `unreadCount`, `success`), dashboard (`/stats`), auth (Register returns user), DeleteTag + DeleteCategory endpoints
- **Frontend fixes**: RichTextEditor media picker (bare array), admin media page (bare array + data URIs), RSS route (Go backend, not in-memory store), sitemap (bare arrays), blog slug detail page (authorName), blog API test (2 published posts)
- **Dead route cleanup**: 27 files deleted (716 lines), entire `frontend/app/api/` tree removed
- **Test results** (clean DB):
  - Desktop: 229 passed, 2 failed (breadcrumb — pre-existing, blog draft status — pre-existing)
  - Mobile: 11 failed (all pre-existing mobile layout/timeout issues)
- **Go tests**: all 82 pass (`ok omoikane-backend/internal/handlers ~12s`)
- **Phase 11 fix — breadcrumb**: `GetPageBySlug` includes `parentTitle`/`parentSlug`; frontend renders MUI `<Breadcrumbs>` — fixes `04-pages.spec.ts:61`
- **Phase 11 fix — draft visibility**: New `GET /admin/blog/posts` endpoint (admin-only, all statuses); admin blog page uses it — fixes `22-admin-blog.spec.ts:40`
- **Desktop Playwright**: now **231/231 pass** (0 failures)

### Mobile Failures (pre-existing)
- 11 failures — pre-existing mobile viewport/timeout issues

## Key Decisions
- **Fix Go, not frontend** — Go is the permanent backend; shape changes localized per handler
- **Bare arrays for lists** — `GetPosts`, `GetPages`, `GetTags`, `GetCategories`, `GetMedia` return `[...]`

## Next Steps (Phase 11)
1. Investigate mobile Playwright failures
2. Phase 11 features (email + password reset, ReCAPTCHA)

## Critical Context
- **All 82 Go tests pass**
- **231/231 desktop Playwright tests pass** (0 failures)
- **Mobile**: 11 pre-existing failures (viewport/timeout)
- **Docker backend** uses `omoikane` database; must reset before each clean Playwright run
- **DB reset**: `docker compose exec postgres psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane; CREATE DATABASE omoikane;"` then restart backend

## Key Files Changed
### Phase 10
- `backend/internal/handlers/*.go`: Response shape alignment, new endpoints
- `backend/cmd/api/main.go`: Wired all new endpoints
- `frontend/app/admin/media/page.tsx`: Bare array + data URI support
- `frontend/app/rss/route.ts`: Uses Go backend instead of in-memory store
- `frontend/app/sitemap.ts`: Handles bare array responses
- `frontend/app/(withHeader)/blog/[slug]/page.tsx`: Passes authorName
- `frontend/components/RichTextEditor.tsx`: Bare array media picker
- `frontend/e2e/21-blog-api.spec.ts`: Creates 2 published posts for list test

### Phase 11 fixes
- `backend/internal/handlers/pages.go`: `GetPageBySlug` returns `parentTitle`/`parentSlug`
- `backend/internal/handlers/pages_test.go`: Test for parent fields in `GetPageBySlug`
- `backend/internal/handlers/blog.go`: New `GetAdminPosts` handler (all statuses)
- `backend/internal/handlers/blog_test.go`: Test for `GetAdminPosts`
- `backend/cmd/api/main.go`: New `GET /admin/blog/posts` route
- `frontend/app/(withHeader)/pages/[...slug]/page.tsx`: MUI `<Breadcrumbs>` rendered
- `frontend/app/admin/blog/page.tsx`: Fetches from `/api/admin/blog/posts`
