# Project Omoikane

A headless-ish CMS built with Next.js 16, MUI 9, Go 1.24, PostgreSQL, Docker Compose, and Playwright.

## Stack

- **Frontend:** Next.js 16 (App Router), MUI 9, TipTap (rich text)
- **Backend:** Go 1.24, GORM, PostgreSQL, JWT (httpOnly cookie auth)
- **Infrastructure:** Docker Compose (nginx reverse proxy, Next.js dev, Go + Air hot-reload, PostgreSQL)
- **Testing:** Go tests (`make go-test`) + Playwright desktop & mobile (`make test`)

## Phases

See [TODO.md](./TODO.md) for the full development roadmap.

| Phase | Status | Tests |
|-------|--------|-------|
| 1 — Foundation | ✅ | Included |
| 2 — Admin Features | ✅ | Included |
| 3 — Rich Content | ✅ | Included |
| 4 — Site Settings & SEO | ✅ | Included |
| 5 — Blog Module | ✅ | Included |
| 6 — Manual QA | ✅ | — |
| 7 — Bug Fixes & Polish | ✅ | 205 pass, 0 fail, 8 skip |
| 8 — Blog for Users + Reworks | ✅ | 231 pass, 0 fail, 8 skip |
| 9 — Go Backend + PostgreSQL | ✅ | 77 Go pass, 0 fail |
| 10 — E2E Cleanup | ✅ | 229/231 desktop pass, 77 Go pass |
| 11 — E2E Fixes | ✅ | 231/231 desktop pass, 82 Go pass |

## Quick Start

```bash
make dev      # Start Docker services (nginx + frontend + Go + PostgreSQL)
make go-test  # Run Go backend tests (requires running PostgreSQL)
make test     # Run full Playwright suite
```

On first run, navigate to `/setup` to create the admin account.
