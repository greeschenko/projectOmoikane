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

## ✅ Phase 21: Cache Layer (Redis)

- [x] `redis` container added to docker-compose + go-redis client in backend config
- [x] TTL cache for public GET endpoints (pages, blog list/detail, settings)
- [x] Cache invalidation on writes (create/update/delete page, post, settings, tags/categories, users, trash)
- [x] Admin/authenticated requests bypass cache (no draft leakage)
- [x] Graceful degradation — backend works if Redis is down (noop cache fallback)
- [x] Go tests for cache hit/miss, query-key variance, auth bypass, flush + integration test (miniredis)

**Test status:** Go tests 111/111 pass (0 fail)

## ✅ Phase 22: Image Optimization

- [x] Go-native `github.com/disintegration/imaging` thumbnails at upload (auto-orient, 640px Lanczos, JPEG q80, `_thumb` suffix) — `sharp` NOT used (per decision)
- [x] `MediaItem` gains `ThumbPath` + `Alt`; media list returns `url`/`thumbUrl`/`alt` alongside legacy `data`
- [x] Plain `<img>` + thumbnail delivery on admin media grid + editor embed (NOT `next/image` — per decision)
- [x] Editor image embed uses optimized `url`/`thumbUrl`; alt text prompt on insert (aligns with Phase 24)
- [x] `PUT /media/{id}` update route (alt text edit dialog in admin UI) + swagger regenerated
- [x] Go tests for real-PNG thumbnail generation + alt update

**Test status:** Go tests 119/119 pass (0 fail)

## ✅ Phase 23: CDN-ready Media Delivery

- [x] Public media serving route `GET /media/file/{filename}` (path-traversal safe, no auth) behind nginx `/media/` location
- [x] Cache headers: `Cache-Control: public, max-age=31536000, immutable` + ETag/Last-Modified with conditional 304 responses
- [x] Configurable `MEDIA_BASE_URL` (env / compose) — media URLs become absolute CDN URLs when set; backend + nginx fall back to self-served `/media/file/`
- [ ] HMAC-signed URLs — deferred: stored rich-text `<img src>` URLs would expire and break rendered content; revisit if private-media is needed
- [x] nginx `location /media/` block with long cache header for CDN offload

**Test status:** Go tests 123/123 pass (0 fail) — also fixed test DB connection exhaustion (pool cap + teardown close in `setupTestDB`)

## ✅ Phase 24: Accessibility Audit & Improvements

- [x] Baseline: `@axe-core/playwright` added; new `frontend/e2e/29-accessibility.spec.ts` scans home, page detail, blog, contact, login, admin dashboard, users, blog admin, media, settings, trash, audit-logs — runs in both desktop + mobile projects (auto-wired via `make test`)
- [x] 0 critical/serious violations on all scanned routes (both projects)
- [x] Navigation & semantics — skip-to-content link, `main` landmarks (public, admin, login/register/forgot-password), `aria-expanded`/`aria-controls` on mobile menu toggle, visible `:focus-visible` rings
- [x] Content a11y — `alt` on `MediaItem` (model + admin dialog + editor alt UI) done in Phase 22; fallback alt = filename
- [x] Interactions — keyboard move up/down buttons as alternative to page drag-and-drop reorder (WCAG 2.1.1), `aria-label` on select-all + row checkboxes, named loading spinners (`aria-label="Loading"` on CircularProgress), `role="alert"` via MUI Alert
- [x] Visual/perception — `prefers-reduced-motion` global CSS, status chips (color badges); 44px touch targets deferred (MUI defaults 40px)
- [x] Regression — full `make test` + `make go-test` ran clean as pre-Phase-25 baseline (committed `a9a9e40`)

## ✅ Phase 25: Manual System Review

Completed the full regression baseline and manually reviewed the running app (admin + public surfaces), cataloguing 15 issues. The fixes are scheduled in Phase 26.

- [x] Pre-review regression baseline — Go tests **123/123 pass**; desktop Playwright **272/272 pass** (8 skip); mobile Playwright **272/272 pass** (9 skip) — committed `a9a9e40`
  - e2e flake fixes included: media alt/caption selectors, API-token extraction, `waitForHydration` helper in `frontend/e2e/helpers.ts`
- [x] Manual review of the running application — 15 issues found (see Phase 26)

**Test status:** Go 123/123 pass; desktop 272/272 pass (8 skip); mobile 272/272 pass (9 skip)

## ✅ Phase 26: Fixes from Manual Review

All 15 issues from the Phase 25 manual review fixed (backend + frontend) and full regression executed.

### Page & blog editors / forms (issues 1–9)
- [x] **1. Dashboard React key warning** — `GetDashboardStats` now returns zero-filled last-7-days daily counts for `recentRegistrations` + `recentMessages` (`DATE(created_at)` group, zero-filled) → chart renders without duplicate-`key` warnings / NaN; `dashboard_test.go` updated
- [x] **2. Page create form dialog width** — `/admin/pages` dialog → `maxWidth="lg"` (matches new 2-column layout)
- [x] **3. Page form 2-column layout** — editor on the left 70% (`Grid md:8.4`), all other fields in the right column (`md:3.6`)
- [x] **4. RichTextEditor alignment buttons** — `@tiptap/extension-text-align@^3.29.0` installed; Align Left/Center/Right + Justify toolbar buttons (full mode; excluded in `minimal` email-template mode); e2e asserts the four buttons
- [x] **5. Editor media dialog upload + auto-insert** — upload button in the image dialog (`POST /api/media`, multipart); after upload the grid refreshes and the returned image auto-inserts into the editor with an alt prompt
- [x] **6. Pages list child indentation** — tree `<ul>` margins zeroed, per-level indent `depth*12`, tighter row padding
- [x] **7. Drag-and-drop page reorder — no requests fire** — HTML5 DnD replaced with a pointer-event drag from the drag handle: window `pointermove/pointerup` listeners attach synchronously in `pointerdown` (no render race), rows are hit-tested via `data-page-id` with `document.elementFromPoint`, and the string-vs-number id mismatch was fixed (`page.id` is a JSON number; the DOM attribute is a string → parsed with `Number()` — this was why `dropIndex` was always `-1`). Keyboard move up/down kept. e2e: "drag handle reorders pages via pointer drag"
- [x] **8. Blog create/edit form same as page form** — `/admin/blog` dialog → `maxWidth="lg"` + same 70/30 editor-left layout
- [x] **9. Frontend edit form similar to backend** — public blog "New Post" dialog and `PostDetailClient` edit dialog upgraded to the admin form layout (RichTextEditor + status/tags/category/slug) — one consistent post editor everywhere

### Blog content & visibility (issues 10–11)
- [x] **10. Tags/categories not visible in blog** — render category label + tag chips on public list cards and post detail; added category filter dropdown on `/blog`; e2e coverage added (chips + filter + detail)
- [x] **11. Richer blog posts admin list** — author, created/publish date, like count, category chip, tag chips (all from `/api/admin/blog/posts` payload)

### Admin UI polish (issues 12–15)
- [x] **12. Distinct admin menu icons** — Messages keeps `MailIcon`; Contacts → `ContactMailIcon` in `AdminLayout.tsx`
- [x] **13. Audit logs page empty** — `GetAuditLogs` now proxies to the audit microservice (`AuditServiceURL + /logs`, forwarding query filters) with local-DB fallback; `audit_test.go` rewritten with a fake audit service (new `TestGetAuditLogs_ProxiesToAuditService`)
- [x] **14. API tokens page unclear** — added "What are API tokens?" info panel (headless/programmatic access, `Authorization: Bearer <token>`, token shown only once, roles, expiry, revocation, Swagger UI link)
- [x] **15. Favicon not applied** — new client `FaviconLoader` in the root layout fetches `/api/settings` and injects `<link rel="icon" data-favicon="true">`

### Verification
- [x] `make go-test` — full suite green (dashboard 7-day + audit proxy tests included; all packages `ok`)
- [x] `make test` — full Playwright desktop + mobile regression incl. new coverage (blog tags/category chips + filter, richer admin rows, pointer-drag reorder, editor align buttons, media upload-and-insert, favicon link)
- [x] Update `AGENTS.md` / `TODO.md` phase status

**Test status:** Go suite green (incl. new audit-proxy test); desktop + mobile Playwright regression passes with new Phase 26 coverage — counts tracked in `AGENTS.md`

---

# 🚀 Platform Roadmap — Modular Event-Driven Platform on Kubernetes

The CMS (Phases 1–26) is complete. The project now evolves into a **modular, event-driven platform** that fast-deploys to any cloud Kubernetes with a single `helm install`. Full design, decisions (D1–D10), target architecture, and gate criteria: see [PLAN.md](./PLAN.md).

**Direction (locked):** decompose the monolith into microservices · Kafka event backbone (KRaft default, managed overridable) · CloudEvents 1.0 · outbox pattern · cloud-agnostic Helm chart · GitHub Actions CI · minikube (local) + kind (CI) · flagship demo = webhook module.

**Waves:** Wave 1 — everything in Docker (prove the architecture). Wave 2 — Kubernetes (pure deployment move). Wave 3 — cloud + CI (fast-deploy story).

## ⬜ Wave 1 — Docker: Decomposition & Event-Driven Core

## ✅ Phase 27: Blueprint & Contract Freeze
- [x] Service boundary map — route→service table (auth / content / media / messages / settings / audit / trash / dashboard) → `backend/docs/service-boundaries.md`
- [x] Monorepo layout: `backend/cmd/<service>` convention documented in boundary map; new `backend/internal/events`
- [x] CloudEvents schema catalog (`internal/events/schemas/*.json`, 10 event types) + envelope Go types + 3 tests
- [x] docker-compose: add Kafka (single-node KRaft, apache/kafka:3.9.0); nginx route-split gateway config (per-service upstreams, all → monolith today)
- [x] **Gate:** all Go + Playwright tests still green (126 Go; desktop 276/276; mobile 275/275)

## ✅ Phase 28: Event SDK & Outbox Infrastructure
- [x] `internal/events` SDK: `Config`/`ConfigFromEnv` (KAFKA_* env), `Producer`/`KafkaProducer` (CloudEvents → Kafka, key=subject, `RequireAll`), `Consumer` (consumer groups, retry+backoff, DLQ routing), `Handler`/`HandlerFunc`
- [x] Transactional outbox: `OutboxEvent` model + `GormOutboxStore` (`Enqueue`/`Pending`/`MarkSent`/`MarkAttempt`) + `MigrateOutbox`
- [x] `Relay` worker — interval + batch (`RelayBatchSize`/`RelayInterval`), marks rows sent/failed; `RunOnce` for tests
- [x] `EnsureTopics`/`EnsureTopic` admin provisioning (idempotent) — services call at startup so first publish never races broker topic creation
- [x] **Gate:** integration tests round-trip Kafka in compose — producer→consumer-group ack, outbox→relay→consumer, DLQ on handler failure (skip cleanly when Kafka is down)
- [x] Docs/AGENTS/README updated; Go suite 129 tests green (106 handler + 9 events + 9 middleware + 3 database + 2 mailer)

## ✅ Phase 29: Wave 1 Services — Auth
- [x] `cmd/auth`: setup, auth, users CRUD, roles, API tokens, profile, forgot/reset password — **process split, shared store** (same `omoikane` Postgres until Phase 31 aggregator work; physical schema partition deferred)
- [x] Gateway routes `/api/auth*`, `/api/users*`, `/api/api-tokens*`, `/api/setup*`, `/api/settings/profile|password` → auth-service (`auth_service` upstream flipped to `auth-service:8082`)
- [x] Emits `user.registered` (behind outbox; single-writer — monolith outbox stays nil) from Setup/Register/CreateUser
- [x] **Gate:** auth/users Go + Playwright specs pass against the gateway (full `make go-test` + `make test`)

## ✅ Phase 30: Wave 2 Services — Content + Media
- [x] `cmd/content`: pages, blog, posts, tags, categories — **process split, shared store**
- [x] `cmd/media`: upload, thumbnails, alt edit, file serving — **process split, shared store**
- [x] Emits `page.published`, `post.published`, `media.uploaded` (behind outbox; single-writer — monolith outbox stays nil)
- [x] Gateway: `/api/pages*`, `/api/blog*`, `/api/admin/blog/` → content-service:8083; `/api/media*`, `/media/` → media-service:8084
- [x] **First result #1:** pages/blog/media Playwright specs green against the gateway (full `make go-test` + `make test`)

## ✅ Phase 31: Wave 3 Services — Messages + Settings; Monolith Retired
- [x] `cmd/messages` (:8085): broadcasts, contact form, contacts — migrations + outbox; emits nothing new yet
- [x] `cmd/settings` (:8086): site settings, email templates — migration + outbox + shared-Redis CacheRead (30s)
- [x] `cmd/trash` (:8087): cross-cutting soft-delete aggregator over all entities — **no shared DB**, fans out to each owner's `/internal/trash*` with `X-Internal-Token` (Phase 27 §2)
- [x] Dashboard became an aggregator facade (:8088) — fetches each owner's `/internal/stats`, merges into the exact Phase 26 JSON shapes; **no shared DB**
- [x] Deleted `cmd/api` monolith + `internal/config`; Redis public cache migrated per service (auth + content + settings share one Redis; FlushDB invalidates across services)
- [x] `cmd/docs` (:8089) — Swagger UI service after monolith removal; `location = /api/audit-logs` → audit-service directly
- [x] nginx `/api/` catch-all → `return 404` fail-loud; frontend SSR now goes through the gateway (`lib/api.ts`/rss use `http://nginx` + `/api` prefix)
- [x] **First result #2:** full Go + Playwright green; zero monolith; every request through gateway → microservice

## ✅ Phase 32: Real Events Live — audit is a Kafka consumer
- [x] New emissions live on the backbone: `auth.login` (Login), `contact.received` (SubmitContact), `settings.updated` (UpdateSettings); `user.registered`/`page.published`/`post.published`/`media.uploaded` already live (Phases 29–30)
- [x] Retired the HTTP audit write path entirely: `internal/audit` deleted, all 18 `audit.Emit` call sites + `Handler.AuditServiceURL` + `POST /events` removed
- [x] `cmd/audit` is a Kafka consumer: group `audit`, `ConsumerStartOffset=kafka.LastOffset` (no historical replay), idempotent on `AuditLog.EventID` (unique index), unknown types skipped, malformed mapped payloads → DLQ, reconnect loop + graceful shutdown; `EnsureTopics` log-only
- [x] `cmd/audit/main_test.go`: 9 tests — health, admin gate (401/403/200), listing/filters, per-type mapping (7 catalog types), unknown-type skip, malformed payload → error, redelivery idempotency, detail content
- [x] Handler outbox tests for the new emissions (login/contact/settings); existing outbox tests now filter by event type (loginAs emits auth.login)
- [x] compose: audit-service gets `KAFKA_BROKERS=kafka:29092` + `depends_on kafka`; `AUDIT_SERVICE_URL` dropped from the 5 producers; Makefile `go-test`/`test` scope += `cmd/audit`, `go-build` += `bin/audit`
- [x] Frontend audit-log page: `ACTION_COLORS` += register/publish/upload/contact; entity tabs += settings
- [x] Event catalog: `docs/events.md` (topology, envelope, 7 live types + 3 schema-frozen, consumer behavior); service-boundaries §4/§7 updated
- [x] **Gate:** e2e `25-admin-audit-log.spec.ts` — publish post → poll `/api/audit-logs?entity=post&search=…` for an `action=publish` row (Kafka path, not HTTP)

## ✅ Wave 2 — Kubernetes: Pure Deployment Move

## ✅ Phase 33: Helm Chart v1 + Local Cluster
- [x] Umbrella chart `charts/omoikane` (all services + kafka + postgres + redis + gateway/nginx)
- [x] `make k8s-up` (minikube) + `make k8s-test` (Playwright against cluster, desktop + mobile, db-reset between suites)
- [x] Values files: `values.yaml`, `values-minikube.yaml`, `values-kind.yaml`
- [x] Backend multi-stage image (all 9 binaries); frontend prod standalone image (`next build` type fixes, `sitemap.ts` `force-dynamic`, explicit MUI icon `data-testid`s)
- [x] **First result #3:** `helm install omoikane` on minikube → CMS fully functional (gateway smoke: setup→login→pages/blog→dashboard aggregator→trash cycle→audit Kafka pipeline→Redis cache→swagger UIs through NodePort→fail-loud `/api/` 404; `make go-test` green; desktop + mobile Playwright green)

## ✅ Phase 34: Observability & Ops Hardening
- [x] JSON structured logs (`internal/observability` — slog JSON + stdlib `log` bridge, `service` attribute, `WARNING:`→warn)
- [x] Prometheus `/metrics` per service (`http_requests_total{service,method,route,status}`, duration histogram, in-flight gauge; 2-segment route normalization; pod scrape annotations)
- [x] Liveness/readiness probes on all 14 Deployments (backend `/health`, frontend tcpSocket + high-threshold readiness)
- [x] Migration `Jobs` — MIGRATE_ONLY=1 early-exit in the 6 DB services + chart post-install/post-upgrade hook Jobs (per-service DSN, pg_isready wait initContainer, fail-loud on release wait)
- [x] Secrets via values: SMTP_PASS / RECAPTCHA_SECRET moved out of plaintext env into the Secret (`secretKeyRef`); `secrets.existingSecret` override
- [x] HPA manifests (`autoscaling/v2`, 10 HPAs, default **off** — would fight k8s-db-reset scale-to-0)
- [x] Resource requests/limits for every container (values-driven; sized to minikube 6 CPU/6 GiB)
- [x] OTel tracing: skipped (documented as optional — not needed for the local k8s gate)
- [x] **Gate:** zero-manual-step deploy — `helm install` runs all 6 migration Jobs (fix: `---` document separators per range-emitted hook doc — without them Helm v4 created only the LAST Job); `make go-test` + `make k8s-test` green; `docs/observability.md` added

## ✅ Phase 35: Webhook Module (Flagship Demo)
- [x] `cmd/webhooks` (:8090): subscription CRUD (event-type allow-list = the 7 live backbone types, validation on create/update), admin UI (`/admin/webhooks`)
- [x] Kafka delivery worker: consumer group `webhooks` (`ConsumerStartOffset = kafka.LastOffset`), idempotent enqueue via unique `(subscription_id, event_id)` index, pump outside the consumer loop
- [x] Retries with **exponential backoff** (1s base doubling, 60s cap, `WEBHOOKS_MAX_ATTEMPTS`=6 → terminal `expired` = DLQ-equivalent, visible in the delivery log with attempt/error trail)
- [x] HMAC-SHA256 signing (`X-Omoikane-Signature: sha256=<hex>`), one-time secret reveal, `POST /webhooks/{id}/test` ping through the normal pump, delivery-log UI (filters + pagination)
- [x] Demo sink `cmd/webhook-sink` (:8091, same Service name compose + k8s) + nginx `location /api/webhooks` + chart (7th migration Job `webhooks-migrate`, Deployment, values, `---` separator inherited) + Makefile (waits, scopes, K8S lists, IMAGE_TAG phase35)
- [x] swag regeneration fix: dropped `--parseDependency` from `make swagger` (Go 1.26+/1.27 stdlib `math/rand/v2` generics broke swag v1.16.x; `--parseInternal` still captures handler annotations)
- [x] **Gate:** `make go-test` 213/213 (13 new cmd/webhooks tests); full `make test` desktop + mobile green (new spec 30-webhooks + a11y scan of `/admin/webhooks`); full `make k8s-test` green on minikube; `docs/webhooks.md` added

## ⬜ Wave 3 — Cloud & Scale: Fast-Deploy Story

## 🔲 Phase 36: FIRST RESULT — CI/CD + Cloud Runbooks + Demo
- [ ] GitHub Actions `deploy.yml`: go-test → Playwright → build/push (GHCR) → helm upgrade
- [ ] EKS / GKE / AKS runbooks (`docs/cloud/`)
- [ ] Values for managed Kafka (MSK/Confluent) + managed Postgres (RDS/CloudSQL/Azure DB)
- [ ] **FIRST RESULT:** fresh cloud cluster → `helm install omoikane` → publish post → webhook delivers to demo sink; all gates green

## 🔲 Phase 37: (optional) Workflow/Automation Module
- [ ] Event → condition → action rules engine; admin UI; e2e workflow tests

## 🔲 Phase 38: (optional) Multi-Tenancy & Scale Validation
- [ ] Per-tenant schema separation, tenant-scoped events, HPA stress test

## 📥 Backlog (parked, not scheduled)

- [ ] i18n / multi-language support — currently deferred; UI-only localization (next-intl + language switcher) is the likely scope if picked up
