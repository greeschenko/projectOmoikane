# Omoikane — Frontend

Next.js 16 App Router CMS frontend.

## Structure

```
app/
  (withHeader)/    Public pages (home, pages, preview, settings)
  admin/           Admin panel (dashboard, users, pages, media, messages, settings)
  api/             API routes (auth, pages, media, messages, users, settings, setup)
  layout.tsx       Root layout (MUI ThemeRegistry, OG/Twitter metadata)
  sitemap.ts       /sitemap.xml generation
  robots.ts        /robots.txt generation
components/        Shared React components
lib/
  store.ts         InMemoryStore (singleton on globalThis)
  auth.ts          Session management (cookie-based)
e2e/               Playwright tests (26 spec files, 199 test items)
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

- **In-memory store** — data lives on `globalThis.__omoikane_store__`, lost on restart
- **Docker node_modules** — anonymous volume; install new packages via `docker exec`
- **Auth** — session cookie set by `/api/auth/login`, checked via `getSession()` server-side
- **Settings** — `SiteSettings` object with siteName, tagline, logo, favicon; fetched by branding components
- **Profile** — User object with name, email, avatar (base64); edited via `/settings`
