# Omoikane — CI/CD pipeline & cloud deployment overview (Phase 36)

Phase 36 makes the project deploy to a real cloud cluster from a single button:
the same chart, the same gate, and a FIRST-RESULT smoke that proves the flagship
webhook path end-to-end on the deployed stack.

- Pipeline: `.github/workflows/deploy.yml`
- Kind cluster config for CI: `.github/kind-config.yaml`
- Managed-service values: `charts/omoikane/values-{eks,gke,aks}.yaml`
- Runbooks: `docs/cloud/eks.md`, `docs/cloud/gke.md`, `docs/cloud/aks.md`
- Verification: `docs/cloud/verify-first-result.md` + `scripts/cloud-smoke.sh`

## Pipeline

```
push to main / manual workflow_dispatch
        │
        ▼
┌───────────────┐   ┌─────────────────────────────┐   ┌────────────────────┐
│ go-test       │   │ e2e  (kind cluster)         │   │                    │
│ compose       │──▶│ make k8s-test K8S_DRIVER=   │   │                    │
│ postgres+kafka│   │ kind (desktop + mobile)     │   │ build-push (GHCR)  │
└───────────────┘   └─────────────────────────────┘   └────────────────────┘
                                                              │
                                                              ▼
                                              ┌──────────────────────────────┐
                                              │ deploy (workflow_dispatch,   │
                                              │ environment "production")    │
                                              │ helm upgrade + cloud-smoke   │
                                              └──────────────────────────────┘
```

Every job is the *same* recipe used in local phase gates, so a green CI run
reproduces the repository's own `make go-test` / `make test` / `make k8s-test`
results:

| CI job | Local equivalent | Cluster/tooling |
|---|---|---|
| `go-test` | `make go-test` | `docker compose up -d postgres kafka` on the runner, then create `omoikane_test` |
| `e2e` | `make k8s-test K8S_DRIVER=kind PLAYWRIGHT_BROWSER=` | kind cluster `omoikane` (config in `.github/kind-config.yaml`), helm-installed chart, full desktop + mobile Playwright |
| `build-push` | `make k8s-images` (local build only) | docker build/push to `ghcr.io/<owner>/omoikane-{backend,frontend}:sha-<sha>` + `:phase36` |
| `deploy` | `helm upgrade` + `scripts/cloud-smoke.sh` | kubeconfig + secret values overlay from repo secrets |

### Images

- Dev/minikube: `omoikane/backend:phase36`, `omoikane/frontend:phase36`
  (built + loaded into minikube/kind by `make k8s-images`, never pushed).
- CI/local-push: `ghcr.io/<owner>/omoikane-backend:{sha-<sha>,phase36}` and the
  same for `omoikane-frontend`. The immutable `sha-` tag is the exact build that
  the pipeline tested; `phase36` is the rolling tag the `deploy` job references
  via `--set images.*.tag=phase36`.

### Credentials (repo/environment secrets)

| Secret | Where | What it holds |
|---|---|---|
| `GITHUB_TOKEN` | repo (auto) | GHCR push (`packages: write`), pull |
| `DEPLOY_KUBECONFIG` | environment `production` | base64-encoded kubeconfig for the target cloud cluster |
| `DEPLOY_VALUES` | environment `production` | base64 YAML values overlay — credentials/endpoints that must never be committed (`external.postgres.password`, `secrets.externalKafkaPassword`, `secrets.jwtSecret`, `secrets.internalToken`, real hostnames) |

`deploy` only runs on `workflow_dispatch` (manual; the `provider` and
`gateway_url` inputs pick the values file and the smoke target) and is gated by
a GitHub *environment* so you can add required reviewers. It `helm upgrade`s
with `values-<provider>.yaml` + the overlay, force-rolls the app Deployments
(rebuilt images with an unchanged tag roll no pods otherwise), then runs
`scripts/cloud-smoke.sh "${{ inputs.gateway_url }}"`.

### Managed services (`external.*`)

`charts/omoikane/values.yaml` gains three switches — `external.postgres`,
`external.kafka`, `external.redis`. When a switch is `enabled: true`, the
in-cluster Deployment/PVC is **skipped entirely** and every service reads its
endpoint from the external block:

- `_helpers.tpl` `omoikane.postgresDSN` renders `host=<external> ... dbname=<same
  per-service database map>` — so both `omoikane` and `omoikane_audit` must
  exist on the managed Postgres instance.
- `omoikane.kafkaBootstrap` + topic override select the managed bootstrap; the
  webhook/audit consumers and event relays then dial it.
- TLS/SASL for managed Kafka is wired from the events SDK through to env:
  `KAFKA_SECURITY_PROTOCOL`, `KAFKA_SASL_MECHANISM`, `KAFKA_SASL_USERNAME` are
  rendered from `external.kafka`; `KAFKA_SASL_PASSWORD` always comes from the
  Secret key `external-kafka-password` (`secrets.externalKafkaPassword`), so a
  credential never sits in a values file.
- `external.redis.url` replaces `redis://redis:6379/0` for the shared cache.
- Migration Jobs skip the `wait-postgres` initContainer when `external.postgres`
  is enabled (the managed instance is already up).
- Default rendering is byte-identical to pre-Phase-36 (all switches off).

Runbooks for each provider include the exact override files and `helm` commands.
There is deliberately **no credentials file for cloud provisioning** — the
cluster/MSK/RDS exist before the deploy (runbook steps); the repo secrets only
*mount* that existing infrastructure into the pipeline.