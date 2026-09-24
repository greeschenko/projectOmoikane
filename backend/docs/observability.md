# Observability & Ops Hardening (Phase 34)

Phase 34 made every Omoikane service observable and the Helm deploy restart-safe
and explicit. Scope: structured JSON logs, Prometheus metrics, migration Jobs,
probes, resources, secrets via Secret, and HPA manifests.

## Structured JSON logs

Every service binary calls `observability.Setup("auth")` (or the matching
service name) as its first statement. `backend/internal/observability/logging.go`
installs a stdlib `log/slog` JSON handler as the slog default **and** redirects
the stdlib `log` package through it (`logBridge`), so every existing
`log.Printf`/`Println`/`Fatalf` call site emits a JSON record with a `service`
attribute — zero call-site rewrites. Lines prefixed `WARNING: ` map to the
`WARN` level.

Example (audit migration Job):

```json
{"time":"2026-09-24T20:29:10.321399242Z","level":"INFO","msg":"Audit service connected and migrated","service":"audit"}
{"time":"2026-09-24T20:29:10.321457952Z","level":"INFO","msg":"audit-service: migrations complete, exiting (MIGRATE_ONLY=1)","service":"audit"}
```

## Prometheus metrics

`backend/internal/observability/metrics.go` provides a self-contained registry
per process (one service binary = one process). Middleware records every request
into three families:

| Metric | Labels |
|---|---|
| `http_requests_total` | `service, method, route, status` |
| `http_request_duration_seconds` (histogram, `prometheus.DefBuckets`) | `service, method, route` |
| `http_in_flight_requests` (gauge) | `service` |

**Route cardinality bounding** (`routeLabel`): a path is truncated to at most
two segments and a trailing numeric segment is dropped, so item routes collapse
onto their resource group while collection + subresource pairs stay distinct:

- `/users/5` → `/users`
- `/blog/posts/2` → `/blog/posts`
- `/internal/trash/page/5` → `/internal/trash`
- `/media/file/a.png` → `/media/file`

Every service mux registers `GET /metrics` (inside `newAuthMux`/`newContentMux`
… so it is reachable in cmd tests without booting a server); the production
`http.Server` wraps the mux in `observability.Middleware`. The gateway (nginx)
intentionally has **no** `/metrics` route — `/metrics` is per-pod and never
exposed through the gateway; scrape it directly from the pods.

### Scraping

The chart annotates every backend Pod with `prometheus.io/scrape=true`,
`prometheus.io/port=<svc port>`, `prometheus.io/path=/metrics`
(`observability.prometheusAnnotations`, default `true`) so
kube-prometheus-stack discovers them out of the box. To scrape one directly:

```sh
kubectl port-forward -n omoikane deployment/auth-service 8082:8082
curl -s localhost:8082/metrics
```

`http_requests_total` label order in the text exposition is **alphabetical**
(`method,route,service,status`) — Prometheus sorts the label pairs, so
assertions must use that order.

## Migration Jobs

Each DB-backed service binary (auth, content, media, messages, settings, audit)
supports `MIGRATE_ONLY=1`: connect, `AutoMigrate` + `MigrateOutbox`, print, exit
0. The chart renders one helm hook Job per `backend.databases` entry
(`templates/migration-jobs.yaml`) with that env var, an initContainer that waits
for Postgres with `pg_isready` (the backend runtime image is busybox-only), and
`helm.sh/hook: post-install,post-upgrade` + `before-hook-creation`. Deploy-time
schema changes become explicit and hermetic — a failing migration fails the
release (fail-loud) because hook Jobs complete before `helm … --wait` returns.
Startup auto-migration stays in every Deployment (restart-safe; `k8s-db-reset`
relies on it re-running migrations on scale-back-to-1).

**Helm hook gotcha (verified against Helm v4.3.0):** every `range`-emitted
document in a hook template file MUST start with a `---` YAML document
separator. Without it Helm parses the concatenated documents as a single YAML
stream and only the LAST resource is ever created — the first five migration
Jobs silently never run. The separator is the fix; `helm.sh/hook-weight` is not
required.

```sh
kubectl get jobs -n omoikane        # expect audit/auth/content/media/messages/settings-migrate
kubectl logs -n omoikane job/auth-migrate
```

## Probes

All 14 Deployments carry `readinessProbe` + `livenessProbe` on `/health`
(HTTP GET on the container's `http` port; frontend uses tcpSocket liveness +
`httpGet /` readiness with a high failureThreshold). The frontend readiness
tolerance absorbs SSR cold-start so `helm … --wait` doesn't fail the release.

## Resources

`values.yaml` `resources.*` sets requests/limits for every container, sized to
the 6 CPU / 6 GiB minikube node used by `make k8s-up`:

- backend services: requests 50m/64Mi, limits 500m/512Mi
- frontend: requests 100m/128Mi, limits 1000m/768Mi
- gateway: requests 20m/32Mi, limits 250m/256Mi
- postgres: requests 250m/512Mi, limits 1000m/1536Mi
- redis: requests 50m/64Mi, limits 200m/256Mi
- kafka: requests 500m/768Mi, limits 1500m/1536Mi

## Secrets via Secret

`templates/secrets.yaml` renders an `omoikane-secrets` Secret (unless
`.Values.secrets.existingSecret` is set) with keys `jwt-secret`,
`internal-token`, `smtp-pass`, `recaptcha-secret`. Auth and messages
Deployments consume SMTP_PASS / RECAPTCHA_SECRET via `secretKeyRef` — these are
**never** plaintext env values anymore. JWT_SECRET / INTERNAL_TOKEN also come
from `secretKeyRef`. Postgres password stays in values (dev convenience; point
`secrets.existingSecret` at a real Secret for production).

## HPA

`templates/hpa.yaml` renders one `autoscaling/v2` HorizontalPodAutoscaler per
backend service + the frontend — but **only** when `.Values.autoscaling.enabled`
is true (default **false**). Caveat: an HPA with `minReplicas > 0` fights
`make k8s-db-reset`'s scale-to-0 (HPA scales the DB-owning Deployments back up
and blocks the DROP), so the local k8s gate tooling requires HPAs off. Enable
per-target only in environments that don't run the gate tooling. Validate the
render with:

```sh
helm template omoikane charts/omoikane -f charts/omoikane/values-minikube.yaml \
  --set autoscaling.enabled=true | grep -c 'kind: HorizontalPodAutoscaler'   # 10
```

## Service coverage

| Service | JSON logs | /metrics | probes | MIGRATE_ONLY Job |
|---|---|---|---|---|
| auth (8082) | ✓ | ✓ | ✓ | ✓ (omoikane) |
| content (8083) | ✓ | ✓ | ✓ | ✓ (omoikane) |
| media (8084) | ✓ | ✓ | ✓ | ✓ (omoikane) |
| messages (8085) | ✓ | ✓ | ✓ | ✓ (omoikane) |
| settings (8086) | ✓ | ✓ | ✓ | ✓ (omoikane) |
| audit (8081) | ✓ | ✓ | ✓ | ✓ (omoikane_audit) |
| trash (8087) | ✓ | ✓ | ✓ | n/a (no DB) |
| dashboard (8088) | ✓ | ✓ | ✓ | n/a (no DB) |
| docs (8089) | ✓ | ✓ | ✓ | n/a (no DB) |
| frontend | — | — | ✓ | n/a |
| nginx gateway | — | — | ✓ | n/a |