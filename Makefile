.PHONY: up down reset db-reset test go-test go-build swagger

SWAG := $(shell command -v swag 2>/dev/null || echo "$(HOME)/prodev/go/bin/swag")

swagger:
	cd backend && $(SWAG) init -g cmd/api/main.go --output ./docs --parseDependency --parseInternal --exclude "cmd/audit,docs,cmd/audit/docs"
	cd backend && $(SWAG) init -g cmd/audit/main.go --output ./cmd/audit/docs --parseDependency --parseInternal --exclude "internal,cmd/api,docs"
	@echo "Swagger docs generated"

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
	docker exec docker-frontend-1 npm install --legacy-peer-deps 2>/dev/null || true

down:
	docker compose -f docker/docker-compose.yml down --remove-orphans

reset:
	docker compose -f docker/docker-compose.yml restart frontend
	docker exec docker-frontend-1 npm install --legacy-peer-deps 2>/dev/null || true

db-reset: up
	docker compose -f docker/docker-compose.yml stop audit-service
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='omoikane' AND pid<>pg_backend_pid();"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "CREATE DATABASE omoikane;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane_audit;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "CREATE DATABASE omoikane_audit;"
	docker compose -f docker/docker-compose.yml start audit-service
	docker compose -f docker/docker-compose.yml restart backend
	@echo "Waiting for backend to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8080/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Backend ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Backend not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for audit service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost/api/audit/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Audit service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Audit service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
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
	@echo "Resetting main database for desktop Playwright run..."
	$(MAKE) db-reset
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=desktop
	@echo "Resetting database for mobile run..."
	$(MAKE) db-reset
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=mobile
