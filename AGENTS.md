# Omoikane — Project Context for AI Agents

## Goal
- **Phase 12 complete**: Phase 13 — (next milestone)

## Constraints & Preferences
- `make go-test` to verify all Go tests pass (82 tests, ~12s)
- `make test` for full Playwright suite (desktop + mobile); must reset DB between runs
- nginx proxies `/api/*` → Go:8080
- Database must be reset (`DROP/CREATE` + restart backend) before each clean Playwright run

## Progress
### Done
- **Phase 10**: All Go API response shapes aligned, 27 dead API routes deleted, 229/231 desktop Playwright tests pass
- **Phase 11**: Fixed breadcrumb + draft visibility — **231/231 desktop pass**, 82 Go tests pass
- **Phase 12 complete**: Email integration (SMTP + password reset) + ReCAPTCHA v2

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

### Mobile Failures (pre-existing)
- 11 failures — pre-existing mobile viewport/timeout issues

## Key Decisions
- **Fix Go, not frontend** — Go is the permanent backend; shape changes localized per handler
- **Bare arrays for lists** — `GetPosts`, `GetPages`, `GetTags`, `GetCategories`, `GetMedia` return `[...]`
- **ReCAPTCHA v2 (checkbox)** — more transparent than v3; keys in env vars, not DB
- **SMTP keys in env vars** — not in SiteSettings DB (security anti-pattern)
- **Reset link URL derived from request Host** — no extra config needed
- **Mailer logs to stdout** when SMTP not configured — transparent dev fallback
- **ForgotPassword always returns success** — prevents email enumeration even if email doesn't exist

## Next Steps (Phase 13)
1. Investigate mobile Playwright failures
2. Email templates (customizable via admin UI)
3. Rate limiting on forgot-password
4. Contact form with ReCAPTCHA

## Critical Context
- **Go tests**: all compile (82 tests, need running PostgreSQL)
- **Desktop Playwright**: 231/231 pass (0 failures)
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

### Phase 12 — Public Interactions
- `backend/internal/config/config.go`: SMTP config struct + RecaptchaSecret
- `docker/docker-compose.yml`: SMTP + RECAPTCHA_SECRET env vars
- `backend/internal/models/password_reset_token.go`: New model
- `backend/internal/database/database.go`: Added PasswordResetToken to AutoMigrate
- `backend/internal/mailer/mailer.go`: New SMTP sender package
- `backend/internal/recaptcha/recaptcha.go`: New ReCAPTCHA verify package
- `backend/internal/handlers/handler.go`: Added SMTP + Recaptcha fields
- `backend/internal/handlers/auth.go`: Rewrote ForgotPassword, added ResetPassword, wired ReCAPTCHA
- `backend/cmd/api/main.go`: Added `POST /auth/reset-password` route
- `backend/internal/handlers/handler_test.go`: Added routes + cleanup for password_reset_tokens
- `backend/internal/handlers/auth_test.go`: 8 new tests for forgot/reset flows
- `frontend/components/ReCaptcha.tsx`: New ReCAPTCHA v2 widget
- `frontend/app/reset-password/page.tsx`: New password reset page
- `frontend/app/register/page.tsx`: Added ReCAPTCHA widget + recaptchaToken in body
- `frontend/app/forgot-password/page.tsx`: Added ReCAPTCHA widget + recaptchaToken in body
