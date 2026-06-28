# Omoikane — Project Context for AI Agents

## Goal
- **Phase 10 complete**: Phase 11 — (next milestone)

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (77 tests, ~12s)
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
- **Go tests**: all 77 pass (`ok omoikane-backend/internal/handlers ~12s`)

### Remaining Failures (pre-existing, out of Phase 10 scope)
- `04-pages.spec.ts:61` — breadcrumb not rendered on public page
- `22-admin-blog.spec.ts:40` — `GetPosts` only returns published; admin can't see drafts
- Mobile: 11 failures — pre-existing mobile viewport/timeout issues

## Key Decisions
- **Fix Go, not frontend** — Go is the permanent backend; shape changes localized per handler
- **Bare arrays for lists** — `GetPosts`, `GetPages`, `GetTags`, `GetCategories`, `GetMedia` return `[...]`

## Next Steps (Phase 11)
1. Address the 2 pre-existing desktop Playwright failures
2. Investigate mobile Playwright failures
3. Consider adding admin-only blog posts endpoint for draft visibility

## Critical Context
- **All 77 Go tests pass**
- **2 desktop Playwright failures remain** (pre-existing, not Phase 10 regressions)
- **Docker backend** uses `omoikane` database; reset before each clean run
- **Pre-existing build error**: `frontend/app/(withHeader)/blog/[slug]/page.tsx:42` — unrelated to Phase 10

## Key Files Changed (Phase 10)
- `backend/internal/handlers/*.go`: Response shape alignment, new endpoints
- `backend/cmd/api/main.go`: Wired all new endpoints
- `frontend/app/admin/media/page.tsx`: Bare array + data URI support
- `frontend/app/rss/route.ts`: Uses Go backend instead of in-memory store
- `frontend/app/sitemap.ts`: Handles bare array responses
- `frontend/app/(withHeader)/blog/[slug]/page.tsx`: Passes authorName
- `frontend/components/RichTextEditor.tsx`: Bare array media picker
- `frontend/e2e/21-blog-api.spec.ts`: Creates 2 published posts for list test
