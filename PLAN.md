# Omoikane — Modular Event-Driven Platform on Kubernetes

**Master plan** — evolve the completed CMS into a modular, event-driven platform
that fast-deploys on any cloud Kubernetes via a single `helm install`.

Status: approved direction. Decomposition to microservices, Kafka event backbone,
cloud-agnostic Helm chart, GitHub Actions CI/CD, minikube (local) + kind (CI) testing.
Flagship demo = webhook module. First result = working platform on a fresh cluster.

---

## 1. Locked decisions

| # | Decision | Choice | Why |
|---|---|---|---|
| D1 | Platform direction | Refactor CMS into microservices — split the monolith `cmd/api` into domain services | The existing codebase becomes the platform's first module |
| D2 | Event backbone | Kafka (single-node KRaft by default; managed MSK/Confluent overridable) | Industry-standard event streaming; chart default keeps the demo cheap |
| D3 | Event format | CloudEvents 1.0 (JSON envelope: id, source, type, subject, time, data) | Vendor-neutral, spec-compliant, self-describing |
| D4 | Reliability | Outbox pattern per service → relay → Kafka; consumer groups + DLQ | No dual-write loss; replay-able events |
| D5 | Deployment | Cloud-agnostic umbrella Helm chart `charts/omoikane`; provider values files (eks/gke/aks/kind/minikube) | One chart, any cloud |
| D6 | Local testing | minikube (manual, real cluster, ingress addon) + kind (CI, fast) | Real-cluster fidelity locally; cheap + fast CI |
| D7 | CI/CD | GitHub Actions: test → build → push → helm upgrade | Repo is already git; one workflow file |
| D8 | Waves | Wave 1: everything in Docker (prove architecture). Wave 2: K8s (pure deployment move). Wave 3: cloud + CI (fast-deploy story) | Never mix architecture change with deployment change |
| D9 | Flagship demo | Webhook delivery module (event type → URL, retries/backoff/DLQ) | Demonstrates the entire event chain end-to-end |
| D10 | Repo layout | Monorepo, one Go module, one binary per service (`backend/cmd/<service>`), shared `internal/` = platform SDK | `cmd/audit` already proves the pattern; zero new module plumbing |

## 2. Target architecture (end state of Wave 3)

```
                        ┌───────────────┐
   Browser / Next.js    │   Gateway     │  (nginx: /api/* route split)
                        └───────┬───────┘
      ┌──────────┬───────────┬───┴────────┬───────────┬──────────┬─────────┐
      ▼          ▼           ▼            ▼           ▼          ▼         ▼
 cmd/auth   cmd/content  cmd/media   cmd/messages cmd/settings cmd/webhooks cmd/audit
 (users,    (pages,      (upload,    (broadcasts,  (site        (subs,      (consumes
  roles,     blog,        thumbs,     contact,      settings,    delivery,   events)
  tokens,    tags,        file serve) notifs)       email tmpl)  retries)
  setup,     categories,
  login)     trash, sitemap)

   each service: own Postgres schema   ┐
   outbox table + relay                ┼──► Kafka (KRaft) ◄── consumers + DLQ
   CloudEvents producer/consumer SDK   ┘    (shared internal/events)
        ▲
        └─── dashboard = aggregator (internal fetches, no direct DB)
```

- **Data:** one Postgres instance, one schema per service (fast, cloud-agnostic);
  overridable to managed DB per service later via chart values.
- **Dashboard:** becomes an aggregator — fetches stats from services via internal API.

## 3. Phase pipeline (3 waves)

### Wave 1 — Docker: prove the architecture (K8s untouched)

#### Phase 27 — Blueprint & contract freeze
- Service boundary map — exact route→service table (`backend/docs/service-boundaries.md`):
  - **auth**: `/api/setup*`, `/api/auth*`, `/api/users*`, `/api-tokens*`, `/api/settings/profile`, `/api/settings/password`
  - **content**: `/api/pages*`, `/api/blog*`
  - **media**: `/api/media*`, `/api/media/file/*`
  - **messages**: `/api/messages*`, `/api/contact*`, `/api/contacts*`
  - **settings**: `/api/settings*` (site settings + email templates)
  - **audit**: `/api/audit*` (existing microservice; later event-driven), `/api/audit-logs`
  - **trash**: cross-cutting — `/api/trash*` spans every entity type (entity in the path, so the gateway cannot prefix-split it) → dedicated trash aggregator service (see service-boundaries.md §2)
- Monorepo layout: `backend/cmd/<service>`; new `backend/internal/events`
- CloudEvents schema catalog: `backend/internal/events/schemas/*.json`
- docker-compose: add Kafka (single-node KRaft); add route-split gateway config in nginx
- **Gate:** all Go + Playwright tests still green (contract unchanged)

#### Phase 28 — Event SDK & outbox infrastructure
- `internal/events`: CloudEvents envelope, producer, consumer (consumer groups),
  DLQ handling, outbox table + relay worker (interval, batch), config
- **Gate:** integration test — event round-trips Kafka in compose; relay publishes, consumer acks

#### Phase 29 — Wave 1 services: auth
- `cmd/auth`: setup, login/register/logout, forgot/reset password, users CRUD,
  roles, API tokens, profile — own DB schema
- Gateway routes `/api/auth*`, `/api/users*`, `/api-tokens*` → auth
- Emits `user.registered` (behind outbox; consumer exists but nobody subscribes yet)
- **Gate:** auth/users Go + Playwright specs pass against the gateway

#### Phase 30 — Wave 2 services: content + media
- `cmd/content`: pages (incl. reorder, trash, preview token), blog posts,
  tags/categories, sitemap, RSS — own schema
- `cmd/media`: upload (multipart), thumbnails (imaging), alt edit, file serving — own schema
- Emits `page.published`, `post.published`, `media.uploaded`
- **First result #1:** pages/blog/media Playwright specs green against the gateway

#### Phase 31 — Wave 3 services: messages + settings; monolith retired
- `cmd/messages`: broadcast messages, contact form, notification widgets — own schema
- `cmd/settings`: site settings incl. favicon, email templates — own schema
- `cmd/trash`: cross-cutting aggregator over all entities' soft-delete (Phase 27 §2)
- Dashboard becomes an aggregator (fetches stats from services via internal API)
- Delete `cmd/api` monolith; migrate Redis public-cache layer per service
- **First result #2:** full Go + Playwright green; zero monolith; every request flows
  gateway → microservice

#### Phase 32 — Real events live
- Outbox relays publish real domain events for write operations
- `cmd/audit` switches from HTTP proxy to **Kafka consumer** (`user.registered`,
  `page.published`, `post.published`, `media.uploaded`, `auth.login`, …)
- Event catalog documented in Swagger (`/api/swagger/`) + `docs/events.md`
- **Gate:** e2e — publish post → audit entry arrives via event (not HTTP)

### Wave 2 — Kubernetes: pure deployment move

#### Phase 33 — Helm chart v1 + local cluster
- Umbrella chart `charts/omoikane`: subcharts/values for every service +
  kafka (single-node KRaft default) + postgres + redis + gateway/ingress
- `make k8s-up` (minikube: docker driver, ingress + metrics-server addons)
- `make k8s-test` — Playwright suite against the cluster (base URL override)
- Values files: `values.yaml` (base), `values-minikube.yaml`, `values-kind.yaml`
- **First result #3:** `helm install omoikane` on minikube → CMS fully functional

#### Phase 34 — Observability & ops hardening
- Liveness/readiness probes on all services (health endpoints already exist)
- Prometheus metrics (`/metrics` per service), OTel tracing (optional collector),
  JSON structured logs
- Migration `Jobs` (idempotent per-service schema), secrets via values/Secret,
  HPA manifests (cpu/memory), resource requests/limits in chart
- **Gate:** zero-manual-step deploy; failures visible in metrics/logs; restart-safe

#### Phase 35 — Webhook module (flagship demo)
- `cmd/webhooks`: subscription CRUD (event type → URL, optional HMAC secret),
  admin UI page (`/admin/webhooks`), delivery worker (Kafka consumer),
  **retries with exponential backoff, DLQ for dead endpoints, delivery-log UI**
- **Gate:** e2e — publish post → webhook fires with CloudEvents payload;
  a failing endpoint retries with backoff; delivery log shows attempts

### Wave 3 — Cloud & scale: the fast-deploy story

#### Phase 36 — FIRST RESULT: CI/CD + cloud runbooks + demo
- GitHub Actions workflow `deploy.yml`:
  `go-test → playwright (desktop + mobile) → docker build/push (GHCR) → helm upgrade`
  with versioned image tags; credentials via repo secrets
- Runbooks: EKS, GKE, AKS one-pagers (`docs/cloud/`)
- Values: managed Kafka (MSK/Confluent) + managed Postgres (RDS/CloudSQL/Azure DB)
- **FIRST RESULT:** fresh cloud cluster → `helm install omoikane` → log in →
  publish a post → event → webhook delivers to an in-cluster demo sink →
  visible in delivery log + audit log; all gates green

#### Phase 37 — (optional) Workflow/automation module
- Event → condition → action rules engine (e.g. `post.published` + tag=X → call URL / send email)
- Admin UI for rules; e2e workflow tests
- **Gate:** e2e workflow passes

#### Phase 38 — (optional) Multi-tenancy & scale validation
- Per-tenant schema separation, tenant scoping of events, HPA stress test,
  load test with N tenants stays green

## 4. Local testing toolchain (D6 detail)

| Context | Tool | Why |
|---|---|---|
| Dev (fast iteration) | docker-compose + existing `make test` / Playwright | What we already have; no cluster overhead |
| Manual K8s testing | minikube (docker driver; `--addons ingress,metrics-server,storage-provisioner`) | Real single-node cluster; catches real YAML/ingress/secret errors locally |
| CI (GitHub Actions) | kind | Fastest cluster bring-up in CI; no VM layer; loads chart + runs smoke tests |
| Cloud acceptance | Real EKS/GKE/AKS via runbook | Final fast-deploy validation |

`make k8s-up` / `make k8s-down` / `make k8s-test` wrap minikube;
the CI job uses kind with the same `charts/omoikane` and `make k8s-test`.

## 5. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Decomposition breaks the API contract | Contract frozen in Phase 27; gateway split first; every service extraction gated by a full Playwright run |
| Kafka operational weight | Single-node KRaft default; bundled in the chart; managed override for production |
| Moving the monolith to K8s mid-refactor | Waves: decomposition fully done + green in Docker (Wave 1) before any chart exists (Wave 2) |
| Outbox/event loss or duplication | Outbox relay (at-least-once) + consumer idempotency + DLQ (Phases 28, 32, 35) |
| Test suite becomes slow/multi-backend | Tests stay black-box against the gateway; per-service Go unit tests; Playwright unchanged |
| minikube/CI divergence | Same chart + same `make k8s-test` in both places |

## 6. Gate discipline (applies to every phase)

- All Go tests pass (`make go-test`)
- Desktop + mobile Playwright pass on a clean DB (`make test`) — same contract via gateway
- Any chart change: `helm lint` + `helm template` + `make k8s-test` on minikube/kind
- Docs updated in the same commit (TODO.md, AGENTS.md, README.md, runbooks)