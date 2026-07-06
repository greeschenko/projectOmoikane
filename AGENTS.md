# Omoikane — Project Context for AI Agents

## Goal
- Complete Phase 15 (trash system + bulk actions) backend and frontend, verify with `make test`

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (~82 tests: 77 handler, 2 mailer, 3 middleware)
- `make test` for full Playwright suite (desktop + mobile); DB reset twice: before desktop, between desktop and mobile
- `psql -c` needs separate flags per statement (DROP/CREATE in one call fails in transaction)
- DB reset requires `pg_terminate_backend()` before DROP DATABASE (active connections)
- DB reset commands must not be silenced (`2>/dev/null || true` removed) — errors must surface
- nginx proxies `/api/*` → Go:8080
- Database must be reset before each clean Playwright run
- `.next-root-owned/` added to frontend `.gitignore` (Next.js 16 cache)
- Mobile Playwright has `actionTimeout: 15000` in config
- `loginAsAdmin` uses `domcontentloaded` instead of `networkidle`, deletes all existing pages via API before re-creating 2 test pages — this populates trash with soft-deleted pages
- Docker backend uses Air hot-reload; Go source changes are picked up automatically in the running container
- All Phase 15 changes must keep all tests passing

## Progress
### Done
- **Phase 10**: All Go API response shapes aligned, 27 dead API routes deleted
- **Phase 11**: Breadcrumb + draft visibility — 231/231 desktop pass
- **Phase 12**: Email integration (SMTP + password reset) + ReCAPTCHA v2
- **Phase 13a**: Email templates (customizable via admin UI)
- **Phase 13b**: Rate limiting on forgot-password (3 req/15min per IP)
- **Phase 13c**: Contact form with ReCAPTCHA — public POST + admin CRUD
- **Mobile fix round 1**: 186→29 failures (201 pass), desktop still 231/231
  - DB reset isolation (separate Desktop/Mobile resets, terminate connections)
  - `01-setup` keeps submission test on mobile (creates admin user)
  - `loginAsAdmin` uses `domcontentloaded` instead of `networkidle`, 15s mobile timeout
  - `playwright.config.ts` added `actionTimeout: 15000` for mobile project
  - **Root cause**: Duplicate AppBars on mobile (AdminAppBar z-index 1201 overlapped mobile AppBar hamburger). Fix: merge hamburger into AdminAppBar via `onMenuToggle` prop, eliminate separate mobile AppBar
- **Phase 14**: All 230 mobile tests pass — 29 pre-existing failures eliminated
  - Root cause: clicking buttons before API data fetch completes (`/api/users`, `/api/media`); server-rendered elements (filter input, "no media uploaded") resolve `waitFor` before React hydrates event handlers
  - Fix: `page.waitForResponse(r => r.url().includes('/api/...') && r.status() === 200)` before button clicks ensures data loaded before interaction
  - Last failure: `GetPages` handler ignored `menu=true` query param, returned all published pages regardless of `inMenu`; desktop rendered as `<Button>` (not `<Link>`), so `getByRole("link")` never matched, but mobile rendered as `<a>` via `ListItemButton component={Link}`
- **Phase 15 plan**: Trash system + bulk actions, with polish items (undo snackbar, contacts confirm dialog, delete text updates)
- **Phase 15 backend — Trash handlers**: `backend/internal/handlers/trash.go` created with `GetTrash`, `GetTrashCount`, `RestoreItem`, `HardDeleteItem`, `EmptyTrash` — supports pages, users, posts, media, contacts, messages, tags, categories
- **Phase 15 backend — Media delete split**: `DeleteMedia` now soft-deletes only (file stays on disk); hard-delete from trash calls `os.Remove` via `HardDeleteItem`
- **Phase 15 backend — Routes**: All trash routes added to `main.go`
- **Phase 15 backend — Batch handlers**: `BatchUsers` (users.go), `BatchPages` (pages.go), `BatchPosts` (blog.go), `BatchMedia` (media.go) — wired to `main.go`
- **Go backend**: Builds clean, all 82 tests pass (77 handler + 2 mailer + 3 middleware)
- **Phase 15 frontend**: All features implemented and passing
  - Trash page at `/admin/trash` with entity tabs, table, restore/hard-delete, empty state
  - Trash nav item with `DeleteSweepIcon` + badge polling (`GET /api/trash/count` every 30s)
  - Bulk action checkboxes + toolbars on Users (Delete/Ban/Activate), Pages (Publish/Draft/Delete), Blog (Publish/Draft/Delete), Media (Delete Selected)
  - Delete confirmation dialog on Contacts page
  - UndoSnackbar component created (not yet wired into delete flows — no tests written for it)
  - Fixed Go batch handlers: `IDs []string` → `IDs []uint` to match frontend numeric JSON IDs (previously caused silent 400 errors, refetch never fired)
  - **249/249 Playwright tests pass** (231 desktop + 9 skipped, 230 mobile + 1 skipped), **82/82 Go tests pass**

## Next Steps
1. (Optional) Wire UndoSnackbar into delete flows for undo-toast UX

## Critical Context
- **Go tests**: 82/82 pass (77 handler + 2 mailer + 3 middleware; need running PostgreSQL) — last verified this session
- **Desktop Playwright**: 231/231 pass, 8 skipped — 0 failures
- **Mobile Playwright**: 230/230 pass, 9 skipped — 0 failures
- **`GetTrashCount`** queries `Unscoped().Where("deleted_at IS NOT NULL").Count()` across all 8 entity models — used for sidebar badge
- **`osRemove` var** in `trash.go` replaces direct `os.Remove` calls for testability (can be swapped in tests)
- Models with `gorm.Model`: Page, User, BlogPost, MediaItem, ContactMessage, Message, Tag, Category — all support GORM soft-delete via `DeletedAt`
- Trash routes: `GET /trash`, `GET /trash/count`, `POST /trash/{entity}/{id}/restore`, `DELETE /trash/{entity}/{id}`, `DELETE /trash` — all admin-only
- Batch routes: `POST /users/batch`, `POST /pages/batch`, `POST /blog/posts/batch`, `POST /media/batch` — admin/auth-protected
- All Phase 15 changes must keep 461/461 tests passing (231 desktop + 230 mobile + 82 Go)
- **Checkbox column** in users table shifts cell indices: `td:nth(2)` is email (was `nth(1)` before checkbox column was added)
- **Bulk toolbar buttons** use exact labels: "Publish", "Draft", "Delete Selected", "Ban", "Activate", "Clear"
- **Contacts page** has both card-level "Delete" button and dialog "Delete" button — use `.first()` or dialog scoping for disambiguation
- **Trash count polling**: AdminLayout fetches `GET /api/trash/count` every 30s (interval on mount, cleared on unmount)
- All models use GORM `DeletedAt` for soft-delete: Page, User, BlogPost, MediaItem, ContactMessage, Message, Tag, Category

## Relevant Files
### Phase 15 backend
- `backend/internal/handlers/trash.go`: Trash endpoints — unified listing + restore/hard-delete/empty
- `backend/internal/handlers/media.go`: Soft-delete only; file purged via trash handler
- `backend/internal/handlers/users.go`: BatchUsers (delete/ban/activate) — `IDs []uint`
- `backend/internal/handlers/pages.go`: BatchPages (delete/publish/draft) — `IDs []uint`
- `backend/internal/handlers/blog.go`: BatchPosts (delete/publish/draft) — `IDs []uint`
- `backend/internal/handlers/media.go`: BatchMedia (delete) — `IDs []uint`
- `backend/cmd/api/main.go`: 5 trash routes + 4 batch routes wired
- `backend/internal/handlers/media_test.go`: Updated test verifies file stays on disk after soft-delete

### Phase 15 frontend — New
- `frontend/components/UndoSnackbar.tsx`: Shared snackbar with undo button (not yet wired)
- `frontend/app/admin/trash/page.tsx`: Trash page with entity tabs, table, restore/hard-delete, empty state
- `frontend/e2e/23-admin-trash.spec.ts`: Trash page + restore/hard-delete E2E tests
- `frontend/e2e/24-admin-contacts.spec.ts`: Contacts confirmation dialog E2E tests

### Phase 15 frontend — Modified
- `frontend/components/AdminLayout.tsx`: Trash nav item with `DeleteSweepIcon` + badge polling on 30s interval
- `frontend/app/admin/users/page.tsx`: Checkbox column, select-all, bulk toolbar (Delete/Ban/Activate)
- `frontend/app/admin/blog/page.tsx`: Checkbox per post, bulk toolbar (Publish/Draft/Delete)
- `frontend/app/admin/media/page.tsx`: Checkbox overlay on cards, bulk toolbar (Delete Selected); delete dialog text updated
- `frontend/app/admin/pages/page.tsx`: Checkbox on each tree item + recursive `PageTreeList`/`PageTreeItem` props; bulk toolbar (Publish/Draft/Delete)
- `frontend/app/admin/contacts/page.tsx`: Delete confirmation dialog (previously immediate delete)
- `frontend/e2e/06-admin-users.spec.ts`: Bulk action tests added; email column index fixed; confirm button scoped to dialog
- `frontend/e2e/07-admin-pages.spec.ts`: Bulk action tests added; confirm button scoped to dialog; count moved after waitFor
- `frontend/e2e/08-admin-media.spec.ts`: Bulk action tests added; confirm button scoped to dialog; syntax fixed; button label "/delete/i" for media dialog
- `frontend/e2e/22-admin-blog.spec.ts`: Bulk action tests added; loginAsAdmin in beforeEach; getByRole("checkbox"); waitForResponse before confirm click
- `frontend/e2e/23-admin-trash.spec.ts`: Rewritten for non-empty state
- `frontend/e2e/24-admin-contacts.spec.ts`: Added .first() on delete button locators

### Documentation
- `AGENTS.md`: This file
- `TODO.md`: Phase 15 breakdown with backend done, frontend in progress
