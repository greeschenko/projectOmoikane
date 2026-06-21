.PHONY: up down reset test

up:
	docker compose -f docker/docker-compose.yml up -d --remove-orphans
	docker exec docker-frontend-1 npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-image 2>/dev/null || true

down:
	docker compose -f docker/docker-compose.yml down --remove-orphans

reset:
	docker compose -f docker/docker-compose.yml restart frontend
	docker exec docker-frontend-1 npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-image 2>/dev/null || true

test:
	docker compose -f docker/docker-compose.yml restart frontend
	@echo "Waiting for frontend to be ready..."
	@for i in $$(seq 1 60); do \
		if curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>/dev/null | grep -q "307\|200"; then \
			echo "Frontend ready after $$i seconds"; \
			break; \
		fi; \
		sleep 2; \
	done
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=desktop
	docker compose -f docker/docker-compose.yml restart frontend
	@echo "Waiting for frontend to be ready..."
	@for i in $$(seq 1 60); do \
		if curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>/dev/null | grep -q "307\|200"; then \
			echo "Frontend ready after $$i seconds"; \
			break; \
		fi; \
		sleep 2; \
	done
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=mobile
