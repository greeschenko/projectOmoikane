.PHONY: up down reset db-reset test go-test go-build swagger

SWAG := $(shell command -v swag 2>/dev/null || echo "$(HOME)/prodev/go/bin/swag")

swagger:
	cd backend && $(SWAG) init -g cmd/docs/main.go --output ./docs --parseDependency --parseInternal --exclude "cmd/audit,docs,cmd/audit/docs"
	cd backend && $(SWAG) init -g cmd/audit/main.go --output ./cmd/audit/docs --parseDependency --parseInternal --exclude "internal,docs"
	@echo "Swagger docs generated"

up:
	docker compose -f docker/docker-compose.yml up -d --remove-orphans
	# nginx resolves upstream hostnames once at startup; compose recreates
	# containers (new IPs) whenever their config changes, so restart nginx
	# after the stack is created to re-resolve them.
	docker compose -f docker/docker-compose.yml restart nginx
	@echo "Waiting for auth service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8082/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Auth service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Auth service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for content service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8083/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Content service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Content service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for media service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8084/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Media service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Media service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for messages service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8085/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Messages service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Messages service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for settings service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8086/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Settings service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Settings service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for trash service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8087/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Trash service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Trash service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for dashboard service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8088/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Dashboard service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Dashboard service not ready after 30s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for docs service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8089/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Docs service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Docs service not ready after 30s"; exit 1; fi; \
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
	docker compose -f docker/docker-compose.yml stop auth-service
	docker compose -f docker/docker-compose.yml stop content-service
	docker compose -f docker/docker-compose.yml stop media-service
	docker compose -f docker/docker-compose.yml stop messages-service
	docker compose -f docker/docker-compose.yml stop settings-service
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='omoikane' AND pid<>pg_backend_pid();"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "CREATE DATABASE omoikane;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane_audit;"
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -d postgres -c "CREATE DATABASE omoikane_audit;"
	docker compose -f docker/docker-compose.yml start audit-service
	docker compose -f docker/docker-compose.yml restart auth-service
	docker compose -f docker/docker-compose.yml restart content-service
	docker compose -f docker/docker-compose.yml restart media-service
	docker compose -f docker/docker-compose.yml restart messages-service
	docker compose -f docker/docker-compose.yml restart settings-service
	@echo "Waiting for auth service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8082/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Auth service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Auth service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for content service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8083/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Content service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Content service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for media service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8084/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Media service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Media service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for messages service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8085/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Messages service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Messages service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for settings service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8086/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Settings service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Settings service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Waiting for audit service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8081/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Audit service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Audit service not ready after 15s"; exit 1; fi; \
		sleep 2; \
	done
	@echo "Restarting frontend for clean state..."
	docker compose -f docker/docker-compose.yml restart frontend
	# Re-resolve upstream IPs (nginx caches hostnames at startup; a container
	# recreation between phases can invalidate them).
	docker compose -f docker/docker-compose.yml restart nginx
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
	cd backend && go test -p 1 ./internal/... ./cmd/auth/... ./cmd/content/... ./cmd/media/... ./cmd/messages/... ./cmd/settings/... ./cmd/trash/... ./cmd/dashboard/... ./cmd/docs/...

go-build:
	cd backend && go build -o bin/auth ./cmd/auth
	cd backend && go build -o bin/content ./cmd/content
	cd backend && go build -o bin/media ./cmd/media
	cd backend && go build -o bin/messages ./cmd/messages
	cd backend && go build -o bin/settings ./cmd/settings
	cd backend && go build -o bin/trash ./cmd/trash
	cd backend && go build -o bin/dashboard ./cmd/dashboard
	cd backend && go build -o bin/docs ./cmd/docs

test: up
	@echo "Creating test database..."
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "DROP DATABASE IF EXISTS omoikane_test;" 2>/dev/null || true
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "CREATE DATABASE omoikane_test;" 2>/dev/null || true
	@echo "Running Go backend tests..."
	cd backend && TEST_DATABASE_URL="host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable" go test -p 1 ./internal/... ./cmd/auth/... ./cmd/content/... ./cmd/media/... ./cmd/messages/... ./cmd/settings/... ./cmd/trash/... ./cmd/dashboard/... ./cmd/docs/...
	@echo "Resetting main database for desktop Playwright run..."
	$(MAKE) db-reset
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=desktop
	@echo "Resetting database for mobile run..."
	$(MAKE) db-reset
	cd frontend && PLAYWRIGHT_EXECUTABLE_PATH=/usr/bin/chromium npx playwright test --config=e2e/playwright.config.ts --project=mobile