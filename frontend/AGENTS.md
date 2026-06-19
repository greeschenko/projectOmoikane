<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

# Testing

Always run tests via `make test` from the repository root (`/home/olex/prodev/DEV/projectOmoikane`). Do NOT invoke npx playwright directly. The Makefile restarts Docker containers and waits for the frontend health check before running tests.
