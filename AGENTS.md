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
- `loginAsAdmin` uses `domcontentloaded` instead of `networkidle`

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

### In Progress
- **Phase 15 frontend**: Not started yet (trash page, bulk action checkboxes, undo snackbar, contacts confirm, delete text)

## Key Decisions
- **Trash system architecture**: Unified `GET /api/trash` endpoint returns soft-deleted items from all entity types with `entityType` discriminator; per-entity restore/hard-delete routes; `DELETE /api/trash` with optional `?entity=` filter for emptying
- **Media file lifecycle**: Soft-delete keeps file on disk; only hard-delete from trash calls `os.Remove` — prevents accidental data loss
- **Batch endpoints**: Per-entity (`POST /api/users/batch`, `POST /api/pages/batch`, etc.) rather than a single generic endpoint — idiomatic Go, clear routing
- **Bulk action scope**: Users: delete/ban/activate; Pages: delete/publish/draft; Posts: delete/publish/draft; Media: delete only
- **Go testability**: `var osRemove` as package-level variable allows mocking `os.Remove` in tests

## Next Steps
1. Build frontend trash page (`frontend/app/admin/trash/page.tsx`)
2. Add trash nav item with badge count to `AdminLayout.tsx`
3. Add bulk action checkboxes + toolbars to users/pages/blog/media pages
4. Add undo snackbar, contacts delete confirmation, delete dialog text updates
5. Run `make test` to verify all pass

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

## Relevant Files
### Phase 15
- `backend/internal/handlers/trash.go`: **New** — all trash endpoints, unified listing + per-entity restore/hard-delete/empty
- `backend/internal/handlers/media.go`: Modified — `DeleteMedia` soft-deletes only (no `os.Remove`); file purged on hard-delete via trash handler
- `backend/internal/handlers/media_test.go`: Modified — test verifies file stays on disk after soft-delete
- `backend/internal/handlers/users.go`: Added `BatchUsers` handler
- `backend/internal/handlers/pages.go`: Added `BatchPages` handler
- `backend/internal/handlers/blog.go`: Added `BatchPosts` handler
- `backend/internal/handlers/media.go`: Added `BatchMedia` handler
- `backend/cmd/api/main.go`: 5 trash routes + 4 batch routes wired
- `frontend/app/admin/trash/page.tsx`: **To be created** — unified trash list UI
- `frontend/components/AdminLayout.tsx`: **To be modified** — add "Trash" nav item with badge
- `frontend/app/admin/users/page.tsx`: **To be modified** — add checkbox column + bulk toolbar
- `frontend/app/admin/pages/page.tsx`: **To be modified** — add checkbox column + bulk toolbar
- `frontend/app/admin/blog/page.tsx`: **To be modified** — add checkbox column + bulk toolbar
- `frontend/app/admin/media/page.tsx`: **To be modified** — add checkbox column + bulk toolbar
- `frontend/app/admin/contacts/page.tsx`: **To be modified** — add delete confirmation dialog
- `TODO.md`: Updated with Phase 15 breakdown and deferred phases
