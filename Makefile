.PHONY: up down reset db-reset test go-test go-build swagger k8s-up k8s-images k8s-db-reset k8s-test k8s-down k8s-destroy cloud-smoke

SWAG := $(shell command -v swag 2>/dev/null || echo "$(HOME)/prodev/go/bin/swag")

# NOTE: no `--parseDependency` here on purpose. swag walks the dependency
# graph with that flag and hits Go 1.26+/1.27 stdlib source (math/rand/v2 uses
# generics syntax swag v1.16.x cannot parse) -> "method must have no type
# parameters". `--parseInternal` still captures the handler annotations, which
# are all under backend/internal/.
swagger:
	cd backend && $(SWAG) init -g cmd/docs/main.go --output ./docs --parseInternal --exclude "cmd/audit,docs,cmd/audit/docs"
	cd backend && $(SWAG) init -g cmd/audit/main.go --output ./cmd/audit/docs --parseInternal --exclude "internal,docs"
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
	@echo "Waiting for webhooks service to be ready..."
	@for i in $$(seq 1 30); do \
		if curl -s http://localhost:8090/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Webhooks service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then echo "Webhooks service not ready after 30s"; exit 1; fi; \
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
	docker compose -f docker/docker-compose.yml stop webhooks-service
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
	docker compose -f docker/docker-compose.yml start webhooks-service
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
	@echo "Waiting for webhooks service to be ready..."
	@for i in $$(seq 1 15); do \
		if curl -s http://localhost:8090/health 2>/dev/null | grep -q '"status":"ok"'; then \
			echo "Webhooks service ready after $$i seconds"; \
			break; \
		fi; \
		if [ $$i -eq 15 ]; then echo "Webhooks service not ready after 15s"; exit 1; fi; \
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
	cd backend && go test -p 1 ./internal/... ./cmd/auth/... ./cmd/content/... ./cmd/media/... ./cmd/messages/... ./cmd/settings/... ./cmd/trash/... ./cmd/dashboard/... ./cmd/docs/... ./cmd/audit/... ./cmd/webhooks/...

go-build:
	cd backend && go build -o bin/auth ./cmd/auth
	cd backend && go build -o bin/content ./cmd/content
	cd backend && go build -o bin/media ./cmd/media
	cd backend && go build -o bin/messages ./cmd/messages
	cd backend && go build -o bin/settings ./cmd/settings
	cd backend && go build -o bin/trash ./cmd/trash
	cd backend && go build -o bin/dashboard ./cmd/dashboard
	cd backend && go build -o bin/docs ./cmd/docs
	cd backend && go build -o bin/audit ./cmd/audit
	cd backend && go build -o bin/webhooks ./cmd/webhooks
	cd backend && go build -o bin/webhook-sink ./cmd/webhook-sink

test: up
	@echo "Creating test database..."
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "DROP DATABASE IF EXISTS omoikane_test;" 2>/dev/null || true
	docker compose -f docker/docker-compose.yml exec -T postgres psql -U omoikane -c "CREATE DATABASE omoikane_test;" 2>/dev/null || true
	@echo "Running Go backend tests..."
	cd backend && TEST_DATABASE_URL="host=localhost port=5432 user=omoikane password=omoikane dbname=omoikane_test sslmode=disable" go test -p 1 ./internal/... ./cmd/auth/... ./cmd/content/... ./cmd/media/... ./cmd/messages/... ./cmd/settings/... ./cmd/trash/... ./cmd/dashboard/... ./cmd/docs/... ./cmd/audit/... ./cmd/webhooks/...
	@echo "Resetting main database for desktop Playwright run..."
	$(MAKE) db-reset
	cd frontend && $(BROWSER_ENV) npx playwright test --config=e2e/playwright.config.ts --project=desktop
	@echo "Resetting database for mobile run..."
	$(MAKE) db-reset
	cd frontend && $(BROWSER_ENV) npx playwright test --config=e2e/playwright.config.ts --project=mobile

# ---------- Phase 33/36: Kubernetes (minikube default, kind for CI) ----------
# Same chart + same gate as CI. Service names, env contracts and the nginx
# gateway config are byte-identical to docker-compose. `make k8s-test` runs the
# full Playwright gate (desktop + mobile) against the cluster gateway.
#
# Driver selection (Phase 36): `K8S_DRIVER=minikube` (local dev — Phase 33
# stack) or `K8S_DRIVER=kind` (CI — `.github/kind-config.yaml`, wired by
# `.github/workflows/deploy.yml`). The matching values file is picked
# automatically; both drivers expose the nginx gateway on NodePort
# $(GATEWAY_PORT) — the minikube VM IP locally, localhost in kind via the
# extraPortMappings in kind-config.yaml.
K8S_NS ?= omoikane
MINIKUBE_PROFILE ?= minikube
KIND_NAME ?= omoikane
K8S_DRIVER ?= minikube
K8S_CHART := charts/omoikane
IMAGE_TAG ?= phase36
K8S_IMAGE_BACKEND := omoikane/backend:$(IMAGE_TAG)
K8S_IMAGE_FRONTEND := omoikane/frontend:$(IMAGE_TAG)
GATEWAY_PORT ?= 30080
# Playwright browser selection: dev machines run the system chromium at the
# classic path; CI installs Playwright's own managed browser
# (`npx playwright install --with-deps chromium`) and passes
# `PLAYWRIGHT_BROWSER=` (empty) so the env override is omitted.
PLAYWRIGHT_BROWSER ?= /usr/bin/chromium
BROWSER_ENV = $(if $(PLAYWRIGHT_BROWSER),PLAYWRIGHT_EXECUTABLE_PATH=$(PLAYWRIGHT_BROWSER) ,)

ifeq ($(K8S_DRIVER),minikube)
K8S_VALUES := $(K8S_CHART)/values-minikube.yaml
K8S_GATEWAY_URL = http://$(shell minikube ip -p $(MINIKUBE_PROFILE)):$(GATEWAY_PORT)
else ifeq ($(K8S_DRIVER),kind)
K8S_VALUES := $(K8S_CHART)/values-kind.yaml
K8S_GATEWAY_URL = http://localhost:$(GATEWAY_PORT)
else
$(error K8S_DRIVER must be "minikube" or "kind" (got "$(K8S_DRIVER)"))
endif

# Services owning the `omoikane` (+ audit) databases — restart after a reset so
# they re-run their startup migrations on the fresh DBs (mirrors compose).
K8S_DB_SERVICES := auth-service content-service media-service messages-service settings-service audit-service webhooks-service
# App deployments running locally-built images (omoikane/backend:*, omoikane/frontend:*)
# plus the nginx gateway (ConfigMap-mounted nginx.conf). Excludes postgres/redis/
# kafka: stock images, never rebuilt — and restarting kafka risks the KRaft wedge.
K8S_APP_SERVICES := auth-service content-service media-service messages-service settings-service trash-service dashboard-service docs-service audit-service webhooks-service webhook-sink frontend nginx

k8s-up:
ifeq ($(K8S_DRIVER),minikube)
	minikube status -p $(MINIKUBE_PROFILE) >/dev/null 2>&1 || minikube start -p $(MINIKUBE_PROFILE) --driver=docker --cpus=6 --memory=6144 --disk-size=30g --container-runtime=containerd
	minikube -p $(MINIKUBE_PROFILE) addons enable ingress
	minikube -p $(MINIKUBE_PROFILE) addons enable metrics-server
	minikube -p $(MINIKUBE_PROFILE) addons enable storage-provisioner
else
	@kind get clusters 2>/dev/null | grep -q "^$(KIND_NAME)$$" || kind create cluster --name $(KIND_NAME) --config .github/kind-config.yaml
endif
	$(MAKE) k8s-images
	helm upgrade --install omoikane $(K8S_CHART) -f $(K8S_VALUES) --namespace $(K8S_NS) --create-namespace --wait --timeout 10m
	# Image loading (minikube image load / kind load) replaces the image inside
	# the node, but a rebuilt image with an UNCHANGED tag makes helm upgrade
	# roll no pods — without this explicit restart the gate would silently test
	# the PREVIOUS build (this cost Phase 33 run 3 its a11y fixes).
	kubectl rollout restart -n $(K8S_NS) $(addprefix deployment/,$(K8S_APP_SERVICES))
	kubectl rollout status deployment -n $(K8S_NS) --timeout=600s
	@echo "Gateway reachable at $(K8S_GATEWAY_URL)"

k8s-images:
	docker build -t $(K8S_IMAGE_BACKEND) -f backend/Dockerfile backend
	docker build -t $(K8S_IMAGE_FRONTEND) -f frontend/Dockerfile frontend
ifeq ($(K8S_DRIVER),minikube)
	minikube image load -p $(MINIKUBE_PROFILE) $(K8S_IMAGE_BACKEND) $(K8S_IMAGE_FRONTEND)
else
	kind load docker-image $(K8S_IMAGE_BACKEND) $(K8S_IMAGE_FRONTEND) --name $(KIND_NAME)
endif

k8s-db-reset:
	@echo "Scaling DB-owning services to 0 (mirrors compose stop)..."
	@for d in $(K8S_DB_SERVICES); do kubectl scale deployment $$d -n $(K8S_NS) --replicas=0; done
	@for d in $(K8S_DB_SERVICES); do kubectl rollout status deployment $$d -n $(K8S_NS) --timeout=120s; done
	$(eval PG_POD := $(shell kubectl get pod -n $(K8S_NS) -l app=postgres -o jsonpath='{.items[0].metadata.name}'))
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='omoikane' AND pid<>pg_backend_pid();"
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='omoikane_audit' AND pid<>pg_backend_pid();"
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane;"
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "CREATE DATABASE omoikane;"
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "DROP DATABASE IF EXISTS omoikane_audit;"
	kubectl exec -n $(K8S_NS) $(PG_POD) -- psql -U omoikane -d postgres -c "CREATE DATABASE omoikane_audit;"
	@echo "Scaling services back to 1 (fresh processes re-run startup migrations)..."
	@for d in $(K8S_DB_SERVICES); do kubectl scale deployment $$d -n $(K8S_NS) --replicas=1; done
	@for d in $(K8S_DB_SERVICES); do kubectl rollout status deployment $$d -n $(K8S_NS) --timeout=300s; done

k8s-test: k8s-up
	helm lint $(K8S_CHART)
	helm template omoikane $(K8S_CHART) -f $(K8S_VALUES) --namespace $(K8S_NS) > /tmp/omoikane-render.yaml
	$(MAKE) k8s-db-reset
	cd frontend && $(BROWSER_ENV) BASE_URL="$(K8S_GATEWAY_URL)" npx playwright test --config=e2e/playwright.config.ts --project=desktop
	$(MAKE) k8s-db-reset
	cd frontend && $(BROWSER_ENV) BASE_URL="$(K8S_GATEWAY_URL)" npx playwright test --config=e2e/playwright.config.ts --project=mobile

k8s-down:
	helm uninstall omoikane --namespace $(K8S_NS) || true

k8s-destroy: k8s-down
ifeq ($(K8S_DRIVER),minikube)
	minikube delete -p $(MINIKUBE_PROFILE)
else
	kind delete cluster --name $(KIND_NAME)
endif

# ---------- Phase 36: cloud acceptance smoke ----------
# FIRST RESULT verification without a browser: fresh deploy -> login -> create a
# webhook subscription -> publish a post -> the event flows Kafka -> webhook
# delivery -> the in-cluster demo sink -> assert delivery log + audit log rows.
# Point CLOUD_SMOKE_URL at the cloud gateway (LoadBalancer/Ingress host) after
# `helm install` per docs/cloud/*.md.
CLOUD_SMOKE_URL ?= http://localhost
cloud-smoke:
	scripts/cloud-smoke.sh "$(CLOUD_SMOKE_URL)"