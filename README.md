# Project Omoikane

A headless-ish CMS built with Next.js 16, MUI 9, Docker Compose, and Playwright.

## Stack

- **Frontend:** Next.js 16 (App Router), MUI 9, TipTap (rich text)
- **Backend:** Next.js API routes, in-memory store (singleton on `globalThis`)
- **Infrastructure:** Docker Compose (nginx reverse proxy, Next.js dev server)
- **Testing:** Playwright (desktop + mobile) via `make test`

## Phases

See [TODO.md](./TODO.md) for the full development roadmap.

| Phase | Status | Tests |
|-------|--------|-------|
| 1 — Foundation | ✅ | Included |
| 2 — Admin Features | ✅ | Included |
| 3 — Rich Content | ✅ | Included |
| 4 — Site Settings & SEO | ✅ | Included |
| 5 — Blog Module | ✅ | 192 pass, 0 fail, 8 skip |

## Quick Start

```bash
make dev      # Start Docker services
make test     # Run full Playwright suite (restarts frontend first)
```

The in-memory store resets on container restart — admin setup is required after each `make dev`.
