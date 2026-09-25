# Omoikane FIRST RESULT — cloud verification (Phase 36)

The FIRST RESULT for a fresh cloud deploy is:

> fresh cloud cluster → `helm install omoikane` → log in → publish a post →
> the `post.published` event travels outbox → Kafka → the webhooks consumer →
> the delivery pump POSTs it to an in-cluster demo sink → **visible in the
> webhook delivery log (status `delivered`) and in the audit log
> (`action=publish`)** — with every local gate green before the deploy.

`scripts/cloud-smoke.sh` automates exactly that chain against any running
gateway (compose, minikube, kind, or a cloud LoadBalancer/Ingress):

```
setup/check → (setup admin if needed) → login
  → create webhook subscription (org.omoikane.content.post.published.v1 → http://webhook-sink:8091/hook)
  → publish a post (status=published)
  → poll /api/webhooks/deliveries until a row for the subscription is
    status=delivered, httpStatus=200 (outbox → Kafka → webhooks consumer → pump → sink)
  → poll /api/audit-logs?entity=post&search=<slug> until logs[0].action=publish
  → cleanup (delete subscription + test post)
```

## The gate is layered

1. **Before any cloud activity** the pipeline runs the same recipes as the local
   gates: `make go-test`, `make k8s-test K8S_DRIVER=kind` (desktop + mobile
   Playwright against the chart on kind), then builds/pushes GHCR images.
2. **The deploy job** installs with `values-<provider>.yaml` + the secrets
   overlay and force-rolls the app Deployments.
3. **`cloud-smoke.sh "$GATEWAY_URL"`** is the final assertion — it is the FIRST
   RESULT check, and it exits non-zero (the workflow fails) on any of:

   - gateway unreachable / `setup/check` not 200
   - setup or login failure (both known admin passwords tried)
   - subscription rejected (unknown event type → 400)
   - post publish rejected
   - delivery not `delivered` within 60 s, or `httpStatus != 200`
   - audit row not `action=publish` within 60 s

## Local dry-run before a cloud deploy

Run the identical script against your minikube/kind gateway (this always runs
before a cloud smoke because the workflow gates on `make k8s-test` first):

```bash
make k8s-up
scripts/cloud-smoke.sh "http://$(minikube ip -p minikube):30080"   # minikube
scripts/cloud-smoke.sh "http://localhost:30080"                    # kind
```

Expect the final line:

```
FIRST RESULT verified — setup -> login -> post.published -> Kafka -> webhook delivered -> audit publish row.
```

## Optional extras

- `OMOKANE_SINK_URL` — point at the sink (e.g. `kubectl port-forward
  svc/webhook-sink 8091:8091` then `http://localhost:8091`) to also fingerprint
  the sink's `GET /requests` list (last 50 received POSTs).
- `OMOKANE_WEBHOOK_URL` — deliver to any reachable URL instead of the sink.
- `OMOKANE_ADMIN_EMAIL` / `OMOKANE_ADMIN_PASSWORD` /
  `OMOKANE_ALT_ADMIN_PASSWORD` — override the admin identity tried by the smoke.

## What "all gates green" means for a deploy

```
make go-test      : 220 Go tests (217 without the Kafka broker) — seed the same
                    run the workflow's go-test job performs
make k8s-test     : full Playwright desktop + mobile gate against the chart on
                    minikube or kind — the workflow's e2e job
helm lint/template: 0 errors for default, minikube, kind AND each
                    values-<provider>.yaml with external.* enabled (the managed
                    render must show no in-cluster postgres/kafka/redis and the
                    external DSNs/bootstrap in place)
cloud-smoke       : FIRST RESULT verified (above)
```