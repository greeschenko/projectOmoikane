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

**Test status: 205 desktop tests pass, 0 fail, 8 mobile-only skipped**

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

Replaced the in-memory store with a Go 1.24 backend using GORM + PostgreSQL, nginx routing `/api/*` to Go.

**Go tests: 77 pass, 0 fail** (3 database + 74 handlers across 8 test files)

### 9.1 — Foundation
- [x] PostgreSQL + Go backend services in docker-compose
- [x] nginx routing `/api/*` → Go:8080 (strips `/api` prefix)
- [x] Go project skeleton (cmd/api, internal/handlers, internal/models, internal/config, internal/database)
- [x] GORM AutoMigrate for all 10 models
- [x] Health endpoint (GET /api/health)
- [x] Air hot-reload for Go dev
- [x] Go test infrastructure (test DB, mock HTTP, database tests)
- [x] Root Makefile targets: `go-test`, `go-build`, `go-lint`

### 9.2 — Auth + Users
- [x] GORM User model with bcrypt password
- [x] JWT token issue/validate (httpOnly cookie "session")
- [x] SetupStatus endpoint (GET /api/setup/check) — public
- [x] POST /api/setup — admin creation
- [x] POST /api/auth/login — JWT issuance
- [x] POST /api/auth/register — user registration
- [x] POST /api/auth/logout — clear session
- [x] GET/POST /api/users, GET/PUT/DELETE /api/users/:id — admin CRUD
- [x] AuthRequired + AdminRequired middleware
- [x] 23 Go tests for auth/user flows

### 9.3 — Settings & Profile
- [x] SiteSetting model (single-row table, auto-seeded)
- [x] GET /api/settings — public
- [x] PUT /api/settings — admin only
- [x] GET /api/settings/profile — authenticated
- [x] PUT /api/settings/profile — update name/email/avatar
- [x] POST /api/settings/password — change password
- [x] 10 Go tests

### 9.4 — Pages
- [x] Page model with parent FK (nullable self-reference)
- [x] GET /api/pages, POST /api/pages
- [x] GET /api/pages/:id, PUT /api/pages/:id, DELETE /api/pages/:id
- [x] GET /api/pages/slug/:slug — public resolve
- [x] PUT /api/pages/reorder — batch sort order update
- [x] GET /api/pages/preview/:token — token-based draft preview
- [x] 14 Go tests

### 9.5 — Blog
- [x] BlogPost, Tag, Category, Like models
- [x] Blog posts CRUD with authorId from JWT session
- [x] Like toggle (atomic `gorm.Expr("like_count + 1")`, re-read post)
- [x] Tags + Categories CRUD
- [x] GET /api/blog/posts (filter by status), GET /api/blog/posts/slug/:slug
- [x] 16 Go tests

### 9.6 — Media & File Storage
- [x] MediaItem model + filesystem storage (UploadDir config)
- [x] POST /api/media — multipart upload to disk with MIME detection
- [x] GET /api/media, GET /api/media/:id/file — serve file
- [x] DELETE /api/media/:id
- [x] 5 Go tests

### 9.7 — Messages
- [x] Message model (readBy JSON array field)
- [x] Messages CRUD, mark as read
- [x] 5 Go tests

### 9.8 — Dashboard
- [x] GET /api/dashboard — aggregated counts (users, pages, posts, media, messages)
- [x] 2 Go tests

### 9.9 — Frontend Integration
- [x] nginx routes `/api/*` → Go:8080 (removes `/api` prefix)
- [x] auth.ts decodes JWT cookies (base64 JSON payload parsing)
- [x] lib/api.ts server-side fetch helper using API_URL env var
- [x] Updated SSR components: home, setup, blog/[slug], preview/[id], pages/[...slug], sitemap, robots
- [x] docker-compose: API_URL=http://backend:8080 env var for frontend service
- [x] Verified all endpoints work through nginx (manual curl E2E)
- [x] GET /api/settings made public for SSR access
- [x] SetupStatus handler added for unauthenticated setup check

## 🔲 Phase 10: Public Interactions
- [ ] Email integration (make forgot-password work, form notifications)
- [ ] ReCAPTCHA/spam protection on public forms

## 🔲 Phase 11: Admin Polish & UX
- [ ] WYSIWYG page builder (drag-and-drop sections, widgets, blocks)
- [ ] Theme customizer (colors, typography, layout options via admin UI)
- [ ] Bulk actions (delete, publish, unpublish multiple pages/users)
- [ ] Soft delete / trash system
- [ ] Audit log (who did what, when)
- [ ] Import/export (CSV/JSON for pages, users, media)

## 🔲 Phase 12: Platform & Performance
- [ ] API tokens / headless CMS mode
- [ ] OpenAPI documentation for all routes
- [ ] Cache layer (Redis or in-memory)
- [ ] Image optimization (sharp, next/image, thumbnails)
- [ ] CDN-ready media delivery
- [ ] Accessibility audit & improvements
- [ ] i18n / multi-language support
