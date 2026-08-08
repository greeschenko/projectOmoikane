# Project Omoikane — Development Roadmap

## ✅ Phase 1: Foundation
- [x] Project setup (Next.js 16, MUI 9, Docker Compose)
- [x] Setup wizard (initial admin account creation)
- [x] Authentication (login, register, forgot-password)
- [x] Public pages (home, dynamic nested pages with breadcrumbs)
- [x] Admin dashboard shell (sidebar layout, auth guard)
- [x] User management (CRUD table, search/filter, sort, roles)

## ✅ Phase 2: Admin Features
- [x] Admin header (AppBar, user avatar dropdown)
- [x] Public header (dynamic menu from pages, login/user state)
- [x] Public footer
- [x] Settings / password change
- [x] User status (active/banned, banned users cannot login)
- [x] Page status (draft/published) & menu toggle
- [x] Message widget (notification bell, unread badge, dropdown)
- [x] Dashboard stats (user count, page count, 7-day registration chart)
- [x] Admin broadcast messages (create, list, mark read)

## ✅ Phase 3: Rich Content
- [x] Rich text editor (TipTap with Bold/Italic)
- [x] Media library (upload, gallery grid, delete)
- [x] Image embed in editor (from media library)
- [x] Page preview (token-based, draft viewing)
- [x] Page reordering (HTML5 drag-and-drop)

---
## ✅ Phase 4: Site Settings & SEO
- [x] Global site settings (site name, tagline, logo, favicon) — `GET/PUT /api/settings`, `/admin/settings` page
- [x] Dynamic branding — PublicHeader/AdminAppBar/PublicFooter fetch settings for live site name, logo, avatar
- [x] User profile editing (name, email, avatar) — `GET/PUT /api/settings/profile`, avatar upload as base64
- [x] `/sitemap.xml` generation — `app/sitemap.ts` includes published pages
- [x] `/robots.txt` generation — `app/robots.ts` with Allow/Disallow/Sitemap
- [x] Structured data (LD+JSON) on public pages — `WebSite` schema in public layout
- [x] OG + Twitter meta tags — via `generateMetadata` in root layout
- [x] Admin sidebar "Settings" link

## ✅ Phase 5: Blog Module

Separate blog entity (not static CMS pages). Tags/categories are blog-only.

**Models:** `BlogPost`, `Tag`, `Category`, `Like` — all CRUD + toggleLike in InMemoryStore.

| Cycle | What |
|-------|------|
| 1 | ✅ Blog Post Model & API — store methods, CRUD routes, like toggle, sitemap inclusion |
| 2 | ✅ Blog Admin UI — sidebar link, post list, create/edit with TipTap, delete |
| 3 | ✅ Categories & Tags Admin — CRUD pages + API routes |
| 4 | ✅ Blog Public Pages — `/blog` list, `/blog/[slug]` detail with author/date/content |
| 5 | ✅ RSS Feed (`/rss`) + Like count on detail page |

**Star/Like:** `POST /api/blog/posts/:id/like` returns `{ liked, count }`.

**Test status: 231 desktop tests pass, 0 fail, 8 mobile-only skipped**

## ✅ Phase 6: Manual QA Checklist
- [x] Setup wizard — fresh container, navigate to `/`, create root admin, verify redirect to `/admin`
- [x] Authentication — login, session survives page refresh, logout clears session
- [x] Admin CRUD — create/edit/delete users, pages, blog posts, tags, categories
- [x] Rich text editor — create a page with TipTap (bold, italic), embed an image from media library
- [x] Media library — upload an image, see it in gallery, delete it
- [x] Drag-and-drop page reordering — rearrange pages in admin, verify order persists on reload
- [x] Page preview — create a draft page, open preview link in incognito, verify draft is visible
- [x] Site settings — change site name/tagline/logo/favicon, verify changes reflect on public pages and admin header
- [x] SEO — verify `/sitemap.xml` lists published pages and blog posts, `/robots.txt` is valid, OG tags appear in page source
- [x] RSS — verify `/rss` returns valid XML with published blog posts, does not include drafts
- [x] Blog — create published + draft posts, verify only published appear on `/blog`, detail page shows content + like count
- [x] User profile — edit name/email/avatar from settings page, verify changes persist
- [x] Responsive — verify mobile sidebar toggle works on admin, public header menu collapses on narrow viewport
- [x] Broadcast messages — create a message as admin, verify badge appears for other users, marking read updates count

## ✅ Phase 7: Bug Fixes & Quick Polish

| Cycle | What |
|-------|------|
| 1 | ✅ Media upload error — client-side file size (>10MB) validation |
| 2 | ✅ Media embed — `allowBase64: true` in TipTap Image extension |
| 3 | ✅ HTML rendering — `dangerouslySetInnerHTML` on pages & preview |
| 4 | ✅ Login redirect — non-admin user goes to `/` instead of `/admin` |
| 5 | ✅ Dashboard — blog stats, media count, recent messages |
| 6 | ✅ Blog view button — link to public post next to edit button |
| 7 | ✅ Blog filtering & search — by title on admin list |
| 8 | ✅ Child pages offset — reduced container margins |
| 9 | ✅ Admin menu icons — MUI icons on all sidebar nav items |
| 10 | ✅ Loaders/spinners — `CircularProgress` on blog/admin pages/tags/categories |
| 11 | ✅ Blog tags/categories — inline MUI `Tabs` on `/admin/blog` |
| 12 | ✅ Blog form alignment — validation, auto-slug, `content→value` fix |

**Test status: 231 desktop tests pass, 0 fail, 8 mobile-only skipped**

## ✅ Phase 8: Blog for Regular Users + Reworks

- [x] Blog on/off toggle — Switch on `/admin/blog` page, hides public blog + nav when off
- [x] Blog in MainMenu — optional nav item tied to `blogEnabled` in `SiteSettings`
- [x] Regular user blog UI on `/blog` — "My Posts" filter, edit own posts, "New Post" button
- [x] Page form rework — full-width dialog with title, slug, content fields
- [x] User settings rework — vertical MUI Tabs (Profile / Password / Avatar)
- [x] Main page redesign — documentation link, GitHub link, project heading/logo

## ✅ Phase 9: Go Backend + PostgreSQL

Replaced the in-memory store with a Go 1.24 + GORM + PostgreSQL backend, fronted by nginx.

**77 Go tests pass, 0 fail** (3 database + 74 handlers across 8 test files)

- Go project skeleton with Air hot-reload, PostgreSQL in docker-compose
- JWT auth (httpOnly cookie) with bcrypt passwords, admin/user middleware
- 34 handler methods: auth, users, settings, pages, blog, media, messages, dashboard
- 10 GORM models with AutoMigrate at startup
- Media file upload to disk with MIME detection
- nginx routes `/api/*` → Go:8080 (strips prefix)
- 7 SSR components (home, setup, blog/[slug], preview/[id], pages/[...slug], sitemap, robots) fetch from Go via `lib/api.ts`
- Updated auth.ts for JWT decoding, docker-compose API_URL env var

## ✅ Phase 10: Playwright E2E Cleanup

Aligned Go API response shapes with frontend expectations, removed dead Next.js API routes.

- **Go handler changes**: blog (bare arrays, `count`, `tags`, `categoryId`, `authorName`), pages (bare array, auth-aware drafts), media (bare array + base64 data URIs, wrapped upload), messages (`readBy`, `unreadCount`, `success`), dashboard (`/stats`), auth (Register returns user), DeleteTag + DeleteCategory endpoints
- **Frontend fixes**: RichTextEditor media picker, admin media page (data URIs), RSS (Go backend), sitemap (bare arrays), blog slug (authorName), blog API test
- **Dead route cleanup**: 27 files deleted (716 lines), entire `frontend/app/api/` tree removed
- **Go test additions**: 5 new tests (82 total)
- **Results** (clean DB): Go tests 77/77 pass; desktop Playwright 229/231 pass (2 pre-existing failures); mobile 11 failures

## ✅ Phase 11: E2E Fixes

Fixed 2 pre-existing desktop Playwright failures.

- **Breadcrumb** (`04-pages.spec.ts:61`): `GetPageBySlug` returns `parentTitle`/`parentSlug`; frontend renders MUI `<Breadcrumbs>`
- **Draft visibility** (`22-admin-blog.spec.ts:40`): New `GET /admin/blog/posts` endpoint (admin-only, all statuses); admin blog page fetches from it
- **Results** (clean DB): Go tests **82/82 pass**; desktop Playwright **231/231 pass** (0 failures); mobile 11 failures (pre-existing)

## ✅ Phase 12: Public Interactions

Email integration, ReCAPTCHA, email templates, rate limiting, and contact form.

- [x] SMTP config (`SMTPHost`, `SMTPPort`, `SMTPUser`, `SMTPPass`, `SMTPFrom`), PasswordResetToken model, `net/smtp` mailer (logs to stdout when unconfigured)
- [x] `POST /auth/forgot-password` (32-byte token, 1h expiry) + `POST /auth/reset-password`, frontend `/reset-password` page
- [x] ReCAPTCHA v2 checkbox — `RECAPTCHA_SECRET` env var, `NEXT_PUBLIC_RECAPTCHA_SITE_KEY` frontend, wired into Register + ForgotPassword
- [x] Email templates — `SiteSetting.resetEmailSubject`/`resetEmailBodyHTML`, admin settings tabs with RichTextEditor, `RenderResetTemplate()` in mailer
- [x] Rate limiting — `NewRateLimiter(rps, burst, window)`, per-IP via `golang.org/x/time/rate` + `sync.Map`, 3 req/15min on forgot-password (3 middleware tests)
- [x] Contact form — `ContactMessage` model, `POST /contact` (public + ReCAPTCHA), admin CRUD, public page + admin list (6 handler tests)

**Test status (clean DB):** Go tests 87/87 pass; desktop Playwright 231/231 pass (0 failures); mobile 201 pass, 29 fail, 9 skip — ALL 29 FIXED in Phase 14

## ✅ Phase 14: Mobile E2E Stability

- [x] **14a — Dialog hydration timing** (23 tests in 06/07/08/13/27): Add `page.waitForResponse` before button clicks; use `getByPlaceholder` instead of `getByRole("table")` for server-rendered elements
- [x] **14b — Media upload file picker** (2 tests in 08): Use `input[type="file"]` directly via `setInputFiles()` instead of clicking "Choose File" button
- [x] **14c — Public header menu selectors** (3 tests in 09/13): Click hamburger toggle before asserting menu link visibility on mobile
- [x] **14d — Strict mode heading** (1 test in 27): Use `{ exact: true }` on blog heading selector
- [x] **14e — GetPages inMenu filter** (1 test in 13): Backend `GetPages` filters `in_menu = true` when `?menu=true` query param present; desktop renders menu as `<Button>` (not `<Link>`), so `getByRole("link")` missed, but mobile uses `<ListItemButton component={Link}>`

**Test status (clean DB):** Go tests 87/87 pass; desktop Playwright 231/231 pass (0 failures); mobile 230/230 pass (0 failures), 9 skip

## ✅ Phase 15: Trash System & Bulk Actions

### Feature A — 🗑️ Trash System (unified trash page + restore/hard-delete)

**Backend:**
- [x] `GET /api/trash` — list all soft-deleted items (unified, with `entityType` discriminator)
- [x] `POST /api/trash/{entity}/{id}/restore` — restore item
- [x] `DELETE /api/trash/{entity}/{id}` — hard-delete permanently
- [x] `DELETE /api/trash` — empty entire trash
- [x] Media: move `os.Remove` from `DeleteMedia` → hard-purge only (soft delete keeps file)
- [x] New `backend/internal/handlers/trash.go`

**Frontend:**
- [x] New page `frontend/app/admin/trash/page.tsx` — entity tabs, table with title/type/deleted date, Restore + Delete Forever
- [x] "Trash" nav item in `AdminLayout.tsx` sidebar with `DeleteSweepIcon` + badge count (polled every 30s)
- [x] Entity type filter tabs on trash page (All / Pages / Users / Posts / etc.)

### Feature B — ☑️ Bulk Actions (checkbox selection + batch endpoints)

**Backend (per-entity batch endpoints):**
- [x] `POST /api/users/batch` — actions: `delete`, `ban`, `activate`
- [x] `POST /api/pages/batch` — actions: `delete`, `publish`, `draft`
- [x] `POST /api/blog/posts/batch` — actions: `delete`, `publish`, `draft`
- [x] `POST /api/media/batch` — actions: `delete`

**Frontend (checkbox UI on each admin page):**
- [x] Users page — checkbox column + bulk toolbar (Ban, Activate, Delete)
- [x] Pages page — checkbox on each tree item + bulk toolbar (Publish, Draft, Delete)
- [x] Blog posts page — checkbox per post + bulk toolbar (Publish, Draft, Delete)
- [x] Media page — checkbox overlay on cards + bulk toolbar (Delete Selected)

### Feature C — 🧹 Polish & Consistency
- [x] Contacts delete: add confirmation dialog
- [x] Media delete dialog: update text for soft-delete
- [ ] Undo snackbar component created (not yet wired into delete flows — no tests)

**Test status:** Go tests 82/82 pass; desktop 231/231 pass (8 skip); mobile 230/230 pass (1 skip)

## 🔲 Phase 16: Manual Testing Session

- [x] Setup wizard — fresh container, navigate to `/`, create root admin, verify redirect to `/admin`
- [x] Authentication — login, session persists across refresh, logout clears session
- [x] Admin user CRUD — create/edit/delete users, search/filter, sort
- [x] Admin user bulk actions — select checkboxes, Ban/Activate/Delete, confirm dialog
- [x] Admin pages CRUD — create/edit/delete, drag-and-drop reorder, preview draft in incognito
- [x] Admin pages bulk actions — select checkboxes, Publish/Draft/Delete
- [x] Admin blog CRUD — create/edit/delete posts with TipTap, manage tags/categories
- [x] Admin blog bulk actions — select checkboxes, Publish/Draft/Delete
- [x] Media library — upload image, view gallery, delete (soft-delete, moves to trash)
- [x] Media bulk actions — select multiple items, "Delete Selected" with confirmation dialog
- [x] Contacts — view list, delete with confirmation dialog
- [x] Trash page — view tabs (All/Pages/Users/Posts/etc.), restore item, hard-delete, verify badge count updates
- [x] Messages — create broadcast message, verify badge appears, mark read
- [x] Site settings — change name/tagline/logo/favicon, verify reflected on public pages + admin header
- [x] Email templates — customize in admin settings
- [x] Dashboard — verify stats load (user count, page count, chart)
- [x] Public pages — home loads, dynamic pages with breadcrumbs, menu from pages
- [x] Blog public — `/blog` lists published posts only, detail page shows content + likes, `/rss` returns valid XML
- [x] Contact form — submit as public user, verify it appears in admin contacts
- [x] Registration — create new account, ReCAPTCHA works, verify redirect to `/`
- [x] Forgot/reset password — request reset, check email (or logs), use token to set new password
- [x] Responsive — mobile sidebar toggle, public header menu collapses on narrow viewport

## ✅ Phase 17: Bug Fixes & Feature Completion

| Item | What | Status |
|------|------|--------|
| 1 | Rich text editor — enhanced toolbar (Bold/Italic/Underline/Strikethrough/H1-H3/Lists/Blockquote/Code/Link/HR/Image/Undo/Redo), ProseMirror CSS, `minimal` prop, 300px height | ✅ |
| 2 | `@tiptap/extension-placeholder` installed | ✅ |
| 3 | Public header — removed Blog/Contact duplication (kept only in MainMenu) | ✅ |
| 4 | Contact form — added client-side email validation, removed `noValidate` | ✅ |
| 5 | Avatar refresh — AdminAppBar/PublicHeader listen for `avatar-changed` event, SettingsForm dispatches it | ✅ |
| 6 | Pages admin — status badges (Published/Draft chips) + Menu badge, indentation reduced (depth*16) | ✅ |
| 7 | Blog post form — Autocomplete tags + Select category in post form | ✅ |
| 8 | Blog sidebar — removed Tags/Categories sub-items (redundant with tabs) | ✅ |
| 9 | Deleted standalone `/admin/blog/tags` and `/admin/blog/categories` pages | ✅ |
| 10 | `DELETE /blog/categories/{id}` route wired in main.go | ✅ |
| 11 | `UpdatePost` — now handles tags (clear + re-associate) and categoryId | ✅ |
| 12 | `BlogPost` model — added `Tags []Tag` field with `many2many:blog_post_tags` | ✅ |
| 13 | Dashboard — `recentRegistrations` + `recentMessages` return real data (last 5) | ✅ |
| 14 | Media library — multiupload (select multiple files, upload sequentially) | ✅ |
| 15 | Trash page — badge refresh after restore/hard-delete via `trash-changed` event | ✅ |
| 16 | Messages page — loading spinner while fetching | ✅ |
| 17 | Blog public — like/unlike button with heart icon on post detail page | ✅ |
| 18 | Tags/Categories tests — rewritten for blog page tabs | ✅ |
| 19 | Media upload tests — updated for multiupload button labels | ✅ |
| 20 | Pages edit test — fixed title extraction (now uses `p` element) | ✅ |
| 21 | Mobile sidebar test — fixed `nav` selector to avoid hidden error overlay | ✅ |

**Test status:** Go tests 88/88 pass; desktop 242/242 pass (8 skip); mobile 249/249 pass (9 skip)

## ✅ Phase 18: Audit Log (separate microservice)
- [x] New `audit-log` microservice with own DB (`cmd/audit`, `omoikane_audit` DB, port 8081)
- [x] Event emission from main app handlers via HTTP (`internal/audit/audit.go`, async goroutine emitter)
- [x] Admin audit log viewer (`/admin/audit-logs` page + `GET /audit-logs` handler)

**Test status:** Go tests 100/100 pass (0 fail); desktop Playwright 353/353 pass (10 skip); mobile Playwright 353/353 pass (11 skip)

## ✅ Phase 19: OpenAPI Documentation

- [x] OpenAPI documentation for all Go routes (swaggo/swag, public Swagger UI at `/api/swagger/` + audit docs at `/api/audit/swagger/`)

## ✅ Phase 20: API Tokens / Headless CMS Mode

- [x] `ApiToken` model — hashed token, name, role, expiresAt, lastUsedAt
- [x] Token auth middleware — `Authorization: Bearer <token>` accepted alongside the JWT cookie (extends `extractClaims`)
- [x] Admin UI to create/revoke tokens (`/admin/api-tokens`)
- [x] Go tests for token auth paths (valid, invalid, expired, revoked, scopes)
- [x] Document token auth in Swagger (`BearerAuth` scheme) + `make swagger`

**Test status:** Go tests 107/107 pass (0 fail)

## 🔲 Phase 21: Cache Layer (Redis)

- [ ] `redis` container added to docker-compose + go-redis client in backend config
- [ ] TTL cache for public GET endpoints (pages, blog list/detail, settings, sitemap data)
- [ ] Cache invalidation on writes (create/update/delete page, post, settings)
- [ ] Admin/authenticated requests bypass cache (no draft leakage)
- [ ] Graceful degradation — backend works if Redis is down (cache-aside with fallthrough)
- [ ] Go tests for cache hit/miss + invalidation

## 🔲 Phase 22: Image Optimization

- [ ] `sharp` on backend — generate thumbnails at upload (sizes, quality, format)
- [ ] MediaItem stores thumbnail + full-res file paths; media list returns thumbnail + full-res URLs
- [ ] Replace base64 data-URI delivery with real file URLs on admin + public rendering
- [ ] Frontend `next/image` with configured domains/remote patterns
- [ ] Editor image embed uses optimized URLs; alt text support (aligns with Phase 24)

## 🔲 Phase 23: CDN-ready Media Delivery

- [ ] Public media serving route (`/media/{filename}`) behind nginx
- [ ] Cache headers: immutable (`Cache-Control: public, max-age=31536000, immutable`) + ETag/Last-Modified
- [ ] Configurable `MEDIA_BASE_URL` (env / site setting) so files can be served from a CDN
- [ ] Optional HMAC-signed URLs for private/expiring media
- [ ] nginx `location` block for media with long cache + `try_files` for CDN offload

## 🔲 Phase 24: Accessibility Audit & Improvements

- [ ] Baseline: add `@axe-core/playwright`, new `frontend/e2e/28-accessibility.spec.ts` scanning key routes (home, pages, blog, contact, login, admin dashboard, users, blog, media, settings, trash, audit-logs) desktop + mobile; wire into `make test`; fix to 0 critical/serious violations
- [ ] Navigation & semantics — skip-to-content link, `<main>`/`<nav>` landmarks, `aria-expanded`/`aria-controls` on mobile menu toggles, focus restore, visible `:focus-visible` rings
- [ ] Content a11y — `alt` field on `MediaItem` (model + admin dialog + editor alt UI), fallback for legacy items, alt on rendered rich-text images
- [ ] Interactions — keyboard alternative for page drag-and-drop reorder (WCAG 2.1.1), `aria-labelledby` on confirm dialogs, `role="alert"` on form errors, `title` on icon-only buttons
- [ ] Visual/perception — theme contrast pass, icon+badge on status chips (color-blind safety), `prefers-reduced-motion`, 44px touch targets
- [ ] Regression — full `make test` (353/353 desktop + mobile) + `make go-test` (100/100) stay green

## 📥 Backlog (parked, not scheduled)

- [ ] i18n / multi-language support — currently deferred; UI-only localization (next-intl + language switcher) is the likely scope if picked up
