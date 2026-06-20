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
**Test status: 163 desktop tests pass, 0 fail, 8 mobile-only skipped**

## ✅ Phase 4: Site Settings & SEO
- [x] Global site settings (site name, tagline, logo, favicon) — `GET/PUT /api/settings`, `/admin/settings` page
- [x] Dynamic branding — PublicHeader/AdminAppBar/PublicFooter fetch settings for live site name, logo, avatar
- [x] User profile editing (name, email, avatar) — `GET/PUT /api/settings/profile`, avatar upload as base64
- [x] `/sitemap.xml` generation — `app/sitemap.ts` includes published pages
- [x] `/robots.txt` generation — `app/robots.ts` with Allow/Disallow/Sitemap
- [x] Structured data (LD+JSON) on public pages — `WebSite` schema in public layout
- [x] OG + Twitter meta tags — via `generateMetadata` in root layout
- [x] Admin sidebar "Settings" link

## 🔲 Phase 5: Blog & Content Extensions
- [ ] Tags & categories for pages
- [ ] Author attribution on pages
- [ ] Content scheduling (publish date)
- [ ] Page revision history
- [ ] RSS feed generation
- [ ] Full-text search across pages and media

## 🔲 Phase 6: Persistence Layer
- [ ] Database integration (SQLite or PostgreSQL via Prisma/Drizzle)
- [ ] Migrate InMemoryStore to database operations
- [ ] File-based media storage (local filesystem or S3)
- [ ] Data migration / seeding scripts

## 🔲 Phase 7: Public Interactions
- [ ] Contact form builder (admin creates forms, public submits)
- [ ] Page comments (with moderation)
- [ ] Email integration (make forgot-password work, form notifications)
- [ ] ReCAPTCHA/spam protection on public forms

## 🔲 Phase 8: Admin Polish & UX
- [ ] WYSIWYG page builder (drag-and-drop sections, widgets, blocks)
- [ ] Theme customizer (colors, typography, layout options via admin UI)
- [ ] Bulk actions (delete, publish, unpublish multiple pages/users)
- [ ] Soft delete / trash system
- [ ] Audit log (who did what, when)
- [ ] Import/export (CSV/JSON for pages, users, media)

## 🔲 Phase 9: Platform & Performance
- [ ] API tokens / headless CMS mode
- [ ] OpenAPI documentation for all routes
- [ ] Cache layer (Redis or in-memory)
- [ ] Image optimization (sharp, next/image, thumbnails)
- [ ] CDN-ready media delivery
- [ ] Accessibility audit & improvements
- [ ] i18n / multi-language support
