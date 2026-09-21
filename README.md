# Project Omoikane

A **headless-ish CMS** evolving into a **modular, event-driven platform** that
fast-deploys to any cloud Kubernetes (AWS EKS, GCP GKE, Azure AKS) with a single
`helm install`. Built with Next.js 16, MUI 9, Go 1.24, PostgreSQL, Kafka, Docker,
and Kubernetes.

## What it is

- **CMS today (Phases 1–26, complete):** pages, blog, media library, auth &
  users, settings, messages, trash, API tokens (headless) — with a full
  Go + Playwright regression suite.
- **Platform next (Phases 27+):** the monolith is decomposed into modular
  microservices connected by a **Kafka event backbone** (CloudEvents + outbox),
  deployed via a **cloud-agnostic Helm chart**, with webhooks, workflows, and
  multi-tenancy as the platform's flagship modules.
- See [PLAN.md](./PLAN.md) for the full platform roadmap; [TODO.md](./TODO.md)
  for the phase-by-phase checklist.

## Stack

- **Frontend:** Next.js 16 (App Router), MUI 9, TipTap (rich text)
- **Backend:** Go 1.24, GORM, PostgreSQL, JWT (httpOnly cookie auth), Redis cache
- **Events:** Kafka (single-node KRaft dev default; managed MSK/Confluent for production), CloudEvents 1.0, outbox pattern
- **Infrastructure (dev):** Docker Compose (nginx gateway, Next.js, Go + Air hot-reload, PostgreSQL, Redis, Kafka, audit service)
- **Infrastructure (platform):** minikube/kind locally → Helm chart → any cloud K8s
- **Testing:** Go tests (`make go-test`) + Playwright desktop & mobile (`make test`) + K8s smoke tests (`make k8s-test`)

## Architecture (target)

```
Browser / Next.js → Gateway (nginx) → auth | content | media | messages | settings | webhooks | audit
                                          └──────────────► Kafka (CloudEvents) ◄──────────────┘
```

Each service owns a Postgres schema, publishes domain events via an outbox relay,
and the dashboard acts as an aggregator over internal APIs. See [PLAN.md](./PLAN.md).

## Phases

| Phase | Status | Tests |
|-------|--------|-------|
| 1 — Foundation | ✅ | Included |
| 2 — Admin Features | ✅ | Included |
| 3 — Rich Content | ✅ | Included |
| 4 — Site Settings & SEO | ✅ | Included |
| 5 — Blog Module | ✅ | Included |
| 6 — Manual QA | ✅ | — |
| 7 — Bug Fixes & Polish | ✅ | 231 pass, 0 fail, 8 skip |
| 8 — Blog for Users + Reworks | ✅ | 231 pass, 0 fail, 8 skip |
| 9 — Go Backend + PostgreSQL | ✅ | 77 Go pass, 0 fail |
| 10 — E2E Cleanup | ✅ | 229/231 desktop pass, 77 Go pass |
| 11 — E2E Fixes | ✅ | 231/231 desktop pass, 82 Go pass |
| 12 — Public Interactions | ✅ | 82 Go pass (password reset + ReCAPTCHA) |
| 13 — Email Templates, Rate Limiting, Contact Form | ✅ | 82 Go pass, 231/231 desktop, 201/239 mobile |
| 14 — Mobile E2E Stability | ✅ | 82 Go pass, 231/231 desktop, 230/230 mobile |
| 15 — Trash System & Bulk Actions | ✅ | 249/249 pass, 82 Go pass |
| 16 — Manual Testing Session | 🔲 | — |
| 17–26 — Bug fixes, audit microservice, OpenAPI, API tokens, Redis cache, CDN-ready media, a11y, manual review fix pass | ✅ | Go green; desktop 276/276; mobile 275/275 |
| 27 — Blueprint & Contract Freeze | ✅ | Go 126 pass; desktop/mobile green; gateway split, Kafka KRaft, CloudEvents catalog |
| 28 — Event SDK & Outbox | ✅ | Go 129 pass; Kafka round-trip integration tests (producer→consumer, outbox→relay→consumer, DLQ) |
| 29 — Auth Service | 🔲 | — |
| 30 — Content + Media Services | 🔲 | — |
| 31 — Messages + Settings; Monolith Retired | 🔲 | — |
| 32 — Real Events Live (audit via Kafka) | 🔲 | — |
| 33 — Helm Chart + Local K8s | 🔲 | — |
| 34 — Observability & Ops | 🔲 | — |
| 35 — Webhook Module (Flagship) | 🔲 | — |
| 36 — CI/CD + Cloud Runbooks + **First Result** | 🔲 | — |
| 37–38 — Workflow module / Multi-tenancy (optional) | 🔲 | — |

## Quick Start

### Docker (current dev setup)

```bash
make dev      # Start Docker services (nginx + frontend + Go + PostgreSQL + Redis + Kafka + audit)
make go-test  # Run Go backend tests (requires running PostgreSQL)
make test     # Run full Playwright suite
```

On first run, navigate to `/setup` to create the admin account.

### Kubernetes (Phases 33+, once available)

```bash
make k8s-up     # Spin up minikube (or use kind)
helm install omoikane ./charts/omoikane
make k8s-test   # Playwright smoke tests against the cluster
```