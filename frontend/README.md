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

- **Go API backend** — nginx is the single gateway: every `/api/*` prefix routes to its owning microservice (auth 8082, content 8083, media 8084, messages 8085, settings 8086, trash 8087, dashboard 8088, docs 8089, audit 8081); unmapped `/api/*` → `return 404`. Server components go through the gateway via `lib/api.ts`
- **Docker node_modules** — anonymous volume; install new packages via `docker exec`
- **Auth** — JWT in httpOnly "session" cookie, set by the auth-service login handler, decoded by `getSession()`
- **Settings** — fetched from settings-service `GET /api/settings` (public endpoint, cached via shared Redis)
- **API_URL** — `process.env.API_URL || 'http://nginx'` used by server components (Phase 31: SSR now flows through the gateway with the `/api` prefix)
- **app/api/ deleted** — Phase 10 removed the entire tree (716 lines); nginx is the sole `/api/*` proxy
