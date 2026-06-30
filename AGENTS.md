# Omoikane — Project Context for AI Agents

## Goal
- **Phase 13 complete**: Next — investigate mobile failures

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (~87 tests: 82 handler, 2 mailer, 3 middleware)
- `make test` for full Playwright suite (desktop + mobile); must reset DB between runs
- nginx proxies `/api/*` → Go:8080
- Database must be reset (`DROP/CREATE` + restart backend) before each clean Playwright run
- `.next-root-owned/` added to frontend `.gitignore` (Next.js 16 cache)

## Progress
### Done
- **Phase 10**: All Go API response shapes aligned, 27 dead API routes deleted, 229/231 desktop tests pass
- **Phase 11**: Breadcrumb + draft visibility — **231/231 desktop pass**, 82 Go tests pass
- **Phase 12**: Email integration (SMTP + password reset) + ReCAPTCHA v2 — 90 Go tests
- **Phase 13a**: Email templates (customizable via admin UI)
- **Phase 13b**: Rate limiting on forgot-password (3 req/15min per IP)
- **Phase 13c**: Contact form with ReCAPTCHA — public POST + admin CRUD

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

### Mobile Failures (pre-existing)
- 11 failures — pre-existing mobile viewport/timeout issues (unchanged since Phase 11)

## Key Decisions
- **Fix Go, not frontend** — Go is the permanent backend; shape changes localized per handler
- **Bare arrays for lists** — `GetPosts`, `GetPages`, `GetTags`, `GetCategories`, `GetMedia` return `[...]`
- **ReCAPTCHA v2 (checkbox)** — more transparent than v3; keys in env vars, not DB
- **SMTP keys in env vars** — not in SiteSettings DB (security anti-pattern)
- **Reset link URL derived from request Host** — no extra config needed
- **Mailer logs to stdout** when SMTP not configured — transparent dev fallback
- **ForgotPassword always returns success** — prevents email enumeration even if email doesn't exist

## Next Steps
1. Investigate 11 pre-existing mobile Playwright failures (viewport/timeout)
2. Verify desktop Playwright still passes (231/231)

## Critical Context
- **Go tests**: all compile (~87 tests: 82 handler, 2 mailer, 3 middleware; need running PostgreSQL)
- **Desktop Playwright**: 231/231 pass (0 failures) — last verified Phase 11
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
