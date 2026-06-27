.PHONY: up down reset test go-test go-build

up:
	docker compose -f docker/docker-compose.yml up -d --remove-orphans
	@echo "Waiting for backend to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8080/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Backend ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Backend not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for frontend to be ready..."
	@for i in $$(seq 1 60); do \
		if curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>/dev/null | grep -q "307\|200"; then \
			echo "Frontend ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 60 ]; then echo "Frontend not ready after 60s"; exit 1; fi; \
		sleep 2; \
	done
	docker exec docker-frontend-1 npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-image 2>/dev/null || true

down:
	docker compose -f docker/docker-compose.yml down --remove-orphans

reset:
	docker compose -f docker/docker-compose.yml restart frontend
	docker exec docker-frontend-1 npm install @tiptap/react @tiptap/starter-kit @tiptap/extension-image 2>/dev/null || true

go-test:
	cd backend && go test -v -p 1 ./internal/...

go-build:
	cd backend && go build -o bin/api ./cmd/api

test: up
	@echo "Creating test database..."
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "DROP DATABASE IF EXISTS omoikane_test;" 2>/dev/null || true
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "CREATE DATABASE omoikane_test;" 2>/dev/null || true
	@echo "Running Go backend tests..."
	cd backend && TEST_DATABASE_URL="host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable" go test -v -p 1 ./internal/...
	@echo "Restarting frontend for clean state..."
	docker compose -f docker/docker-compose.yml restart frontend
	@echo "Waiting for frontend to be ready..."
	@for i in $$(seq 1 60); do \
		if curl -s -o /dev/null -w "%{http_code}" http://localhost/ 2>/dev/null | grep -q "307\|200"; then \
			echo "Frontend ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 60 ]; then echo "Frontend not ready after 60s"; exit 1; fi; \
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
		if [ $$i -eq 60 ]; then echo "Frontend not ready after 60s"; exit 1; fi; \
		sleep 2; \
	done
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=mobile
