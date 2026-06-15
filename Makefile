.PHONY: up test

up:
	docker compose -f docker/docker-compose.yml up

test:
	docker compose -f docker/docker-compose.yml restart frontend
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=desktop
	docker compose -f docker/docker-compose.yml restart frontend
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=mobile
