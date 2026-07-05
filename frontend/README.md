# Omoikane — Frontend

Next.js 16 App Router CMS frontend.

## Structure

```
app/
  (withHeader)/    Public pages (home, pages, preview, settings)
  admin/           Admin panel (dashboard, users, pages, media, messages, settings)
   api/             (deleted in Phase 10 — nginx proxies /api/* → Go:8080)
  layout.tsx       Root layout (MUI ThemeRegistry, OG/Twitter metadata)
  sitemap.ts       /sitemap.xml generation (fetches pages/blog from Go API)
  robots.ts        /robots.txt generation
components/        Shared React components
lib/
  store.ts         Deprecated in-memory store (no longer used for API)
  api.ts           Server-side fetch helper (calls Go backend directly)
  auth.ts          JWT session management (cookie-based)
e2e/               Playwright tests (27 spec files, 231 desktop + 230 mobile test items)
```

## Testing

```bash
# From repo root (restarts Docker, waits for health, runs tests):
make test

# Run a single file (from this directory):
PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test \
  --config=e2e/playwright.config.ts --project=desktop e2e/20-structured-data.spec.ts
```

## Key Conventions

- **Go API backend** — nginx proxies `/api/*` → Go:8080; server components use `lib/api.ts` for direct Go fetches
- **Docker node_modules** — anonymous volume; install new packages via `docker exec`
- **Auth** — JWT in httpOnly "session" cookie, set by Go login handler, decoded by `getSession()`
- **Settings** — fetched from Go `GET /api/settings` (public endpoint)
- **API_URL** — `process.env.API_URL || 'http://backend:8080'` used by server components
- **app/api/ deleted** — Phase 10 removed the entire tree (716 lines); nginx is the sole `/api/*` proxy
