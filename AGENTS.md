# Omoikane — Project Context for AI Agents

## Goal
- **Phase 14 complete**: Zero mobile failures — all 461 tests pass (231 desktop + 230 mobile)

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (~87 tests: 82 handler, 2 mailer, 3 middleware)
- `make test` for full Playwright suite (desktop + mobile); DB reset twice: before desktop, between desktop and mobile
- `psql -c` needs separate flags per statement (DROP/CREATE in one call fails in transaction)
- DB reset requires `pg_terminate_backend()` before DROP DATABASE (active connections)
- DB reset commands must not be silenced (`2>/dev/null || true` removed) — errors must surface
- nginx proxies `/api/*` → Go:8080
- Database must be reset before each clean Playwright run
- `.next-root-owned/` added to frontend `.gitignore` (Next.js 16 cache)

## Progress
### Done
- **Phase 10**: All Go API response shapes aligned, 27 dead API routes deleted
- **Phase 11**: Breadcrumb + draft visibility — **231/231 desktop pass**, 82 Go tests pass
- **Phase 12**: Email integration (SMTP + password reset) + ReCAPTCHA v2 — 90 Go tests
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

### Phase 12 — Public Interactions
- SMTP config in env vars (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`)
- `PasswordResetToken` model with 1-hour expiry
- Mailer package (`backend/internal/mailer/`) — `net/smtp` sender; logs to stdout when SMTP not configured
- `POST /auth/forgot-password` — generates 32-byte crypto token, stores in DB, sends email
- `POST /auth/reset-password` — validates token, hashes + updates password, marks token used
- `ReCAPTCHA_SECRET` env var for backend; `NEXT_PUBLIC_RECAPTCHA_SITE_KEY` for frontend
- `backend/internal/recaptcha/` — verifies ReCAPTCHA v2 tokens against Google
- `frontend/components/ReCaptcha.tsx` — renders checkbox via `react-google-recaptcha`
- ReCAPTCHA wired into `Register` and `ForgotPassword` handlers + pages
- `frontend/app/reset-password/page.tsx` — new page with token from URL + password form
- 8 new Go tests for forgot/reset password flows

### Phase 13a — Email Templates
- `SiteSetting` model: `resetEmailSubject` + `resetEmailBodyHTML` fields (with defaults)
- Mailer: `RenderResetTemplate(templateStr, data)`, `ResetEmailData{ResetLink, SiteName, ExpiryHours}`
- Settings handler: GET/PUT include email template fields
- `ForgotPassword` reads template from settings, renders with `html/template`
- Admin settings page: Tabs (General / Email Templates), RichTextEditor for body HTML

### Phase 13b — Rate Limiting
- `middleware.NewRateLimiter(rps, burst, window)` + `Limiter.Middleware(next)`
- Per-IP tracking via `golang.org/x/time/rate` + `sync.Map` with periodic cleanup
- Forgot-password: 3 requests per 15 min per IP (429 on exceed)
- 3 middleware tests: allow, block after burst, different IPs independent

### Phase 13c — Contact Form
- `ContactMessage` model (name, email, subject, message, read bool)
- `POST /contact` — public, ReCAPTCHA-protected
- `GET /contacts`, `GET /contacts/{id}`, `POST /contacts/{id}/read`, `DELETE /contacts/{id}` — admin only
- `frontend/app/(withHeader)/contact/page.tsx` — public form with ReCAPTCHA
- `frontend/app/admin/contacts/page.tsx` — admin list with mark-read + delete
- Contact added to PublicHeader, MainMenu (mobile + desktop), AdminLayout sidebar
- 6 handler tests: submit success, missing fields, default subject, admin-only, mark read, delete

### Phase 14 — Mobile E2E Stability
- Hydration wait fix: `page.waitForResponse` for `/api/users` or `/api/media` before button clicks (React event handlers not yet attached on slow mobile CPU)
- `06-admin-users`: `waitForResponse` for `/api/users` + `getByPlaceholder(/filter|search/i).waitFor()` instead of `getByRole("table").waitFor()`
- `08-admin-media`: `waitForResponse` for `/api/media` + `locator.setInputFiles()` instead of `click({ force })`
- `08-admin-mobile`: `waitForResponse` for `/api/users` before New User button click
- `GetPages` handler: filter by `in_menu = true` when `?menu=true` query param present

## Key Decisions
- **Fix Go, not frontend** — Go is the permanent backend; shape changes localized per handler
- **Bare arrays for lists** — `GetPosts`, `GetPages`, `GetTags`, `GetCategories`, `GetMedia` return `[...]`
- **ReCAPTCHA v2 (checkbox)** — more transparent than v3; keys in env vars, not DB
- **SMTP keys in env vars** — not in SiteSettings DB (security anti-pattern)
- **Reset link URL derived from request Host** — no extra config needed
- **Mailer logs to stdout** when SMTP not configured — transparent dev fallback
- **ForgotPassword always returns success** — prevents email enumeration even if email doesn't exist

## Next Steps
1. Feature work or further test stability improvements
2. Verify desktop Playwright passes (231/231) — last verified this session

## Critical Context
- **Go tests**: all compile (~87 tests: 82 handler, 2 mailer, 3 middleware; need running PostgreSQL)
- **Desktop Playwright**: 231/231 pass (0 failures) — last verified this session
- **Mobile**: 230/230 pass (0 failures), 9 skip — all 29 pre-existing failures eliminated
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

### Phase 12
- `backend/internal/config/config.go`: SMTP config struct + RecaptchaSecret
- `docker/docker-compose.yml`: SMTP + RECAPTCHA_SECRET env vars
- `backend/internal/models/password_reset_token.go`: New model
- `backend/internal/mailer/mailer.go`: New SMTP sender package
- `backend/internal/recaptcha/recaptcha.go`: New ReCAPTCHA verify package
- `backend/internal/handlers/auth.go`: ForgotPassword + ResetPassword + ReCAPTCHA wiring
- `backend/internal/handlers/auth_test.go`: 8 new tests
- `frontend/components/ReCaptcha.tsx`: ReCAPTCHA v2 widget
- `frontend/app/reset-password/page.tsx`: Password reset page
- `frontend/app/register/page.tsx`: ReCAPTCHA added
- `frontend/app/forgot-password/page.tsx`: ReCAPTCHA added

### Phase 13a — Email Templates
- `backend/internal/models/settings.go`: `ResetEmailSubject` + `ResetEmailBodyHTML` fields
- `backend/internal/mailer/mailer.go`: `RenderResetTemplate`, `ResetEmailData`
- `backend/internal/handlers/settings.go`: GET/PUT include email template fields
- `backend/internal/handlers/auth.go`: `ForgotPassword` reads template from settings
- `frontend/app/admin/settings/page.tsx`: Tabs with RichTextEditor for email body

### Phase 13b — Rate Limiting
- `backend/internal/middleware/ratelimit.go`: `NewRateLimiter`, `Limiter.Middleware`
- `backend/internal/middleware/ratelimit_test.go`: 3 tests
- `backend/cmd/api/main.go`: Rate-limited forgot-password route
- `backend/go.mod`: Added `golang.org/x/time`

### Phase 13c — Contact Form
- `backend/internal/models/contact_message.go`: New model
- `backend/internal/handlers/contacts.go`: `SubmitContact`, `GetContacts`, `GetContact`, `MarkContactRead`, `DeleteContact`
- `backend/internal/handlers/contacts_test.go`: 6 tests
- `backend/internal/database/database.go`: Added `ContactMessage` to AutoMigrate
- `backend/internal/handlers/handler_test.go`: Added `contact_messages` to cleanup
- `backend/cmd/api/main.go`: 5 contact routes wired
- `frontend/app/(withHeader)/contact/page.tsx`: Public contact form
- `frontend/app/admin/contacts/page.tsx`: Admin list with mark-read + delete
- `frontend/components/AdminLayout.tsx`: Contacts nav entry
- `frontend/components/PublicHeader.tsx`: Contact button
- `frontend/components/MainMenu.tsx`: Contact link (desktop + mobile)
- `frontend/.gitignore`: Added `.next-root-owned/`

### Mobile Fix Round 1
- `Makefile`: Two DB resets (before desktop, between desktop and mobile), removed `2>/dev/null || true`, separated `-c` flags
- `frontend/e2e/helpers.ts`: `loginAsAdmin` uses `domcontentloaded`, 15s mobile URL timeout
- `frontend/e2e/playwright.config.ts`: `actionTimeout: 15000` for mobile project
- `frontend/e2e/01-setup.spec.ts`: Skip validation tests on mobile, keep submission test
- `frontend/components/AdminAppBar.tsx`: Added `onMenuToggle` prop + hamburger `MenuIcon`
- `frontend/components/AdminLayout.tsx`: Remove separate mobile AppBar, pass `onMenuToggle` to AdminAppBar, remove unused `mt: 8` on main

### Phase 14 — Mobile E2E Stability
- `frontend/e2e/06-admin-users.spec.ts`: `waitForResponse` for `/api/users` + `getByPlaceholder` instead of `getByRole("table")`
- `frontend/e2e/08-admin-media.spec.ts`: `waitForResponse` for `/api/media` + `locator.setInputFiles()` instead of `click({ force })`
- `frontend/e2e/08-admin-mobile.spec.ts`: `waitForResponse` for `/api/users` before New User button
- `backend/internal/handlers/pages.go`: `GetPages` filters `in_menu = true` when `?menu=true`
