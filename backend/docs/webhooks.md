# Webhook Module (Phase 35 — flagship demo)

The webhook module is the flagship demo of Omoikane's event backbone: platform
activity travels through Kafka once, and the `webhooks-service` fans a chosen
event out to external HTTP endpoints — retrying failures with exponential
backoff, HMAC-signing payloads when a secret is set, and keeping a visible
delivery log. It is deliberately **consumer-only**: it subscribes to the
existing backbone and never emits its own events.

Scope: subscription CRUD (`cmd/webhooks`), an admin UI (`/admin/webhooks`), a
Kafka→DB enqueue consumer, a delivery pump with retry/backoff/expiry, an HMAC
signature, a `POST /webhooks/{id}/test` smoke-test route, and a first-party
demo sink (`cmd/webhook-sink`). The module changed **no backbone producer code**
and **no envelope semantics**.

---

## Architecture

```
                    backbone (unchanged)
┌──────────────┐   ┌──────────────────────────────────┐   ┌──────────────────────────────┐
│ auth/content │   │ webhooks-service (:8090)         │   │ external endpoint             │
│ media/…      │──▶│  consumer (group "webhooks")     │──▶│  (demo: webhook-sink :8091)  │
│   outbox→Kafka│   │  matches ACTIVE subs → pending  │   │  CloudEvents JSON POST       │
└──────────────┘   │  rows (idempotent unique index)  │   │  + X-Omoikane-Signature      │
                   │                                   │   └──────────────┬───────────────┘
                   │  delivery pump (own goroutine)    │◀──────────────────┘
                   │  HTTP POST → 2xx? delivered       │   retry w/ backoff on failure
                   │  else failed → retry → expired    │   (maxAttempts → expired = DLQ-equiv)
                   └──────────────────────────────────┘
```

- **Enqueue vs pump split**: the Kafka consumer only maps events to `pending`
  `WebhookDelivery` rows (fast ACK, idempotent), and a **separate pump
  goroutine** does the HTTP fan-out, retries and backoff. Consumer retries can
  therefore never double-deliver; the pump just resumes from the DB.
- **Idempotency**: `WebhookDelivery` has a unique
  `(subscription_id, event_id)` index. A redelivered event that finds existing
  rows is a no-op (success), so the consumer commits instead of DLQ-ing a
  duplicate.
- **Start offset**: the `webhooks` consumer group starts at `kafka.LastOffset`
  — a fresh deploy never replays history into the delivery store. Committed
  offsets (in Kafka) survive DB resets (same pattern as the `audit` group).
- **Which events**: the seven live backbone types
  (`user.registered`, `auth.login`, `page.published`, `post.published`,
  `media.uploaded`, `contact.received`, `settings.updated`) — the
  `webhookEventTypes` allow-list. Any other type is rejected at create/update
  with 400 so a typo never subscribes to the wrong stream.

## Routes (all admin-gated; gateway-facing)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/webhooks` | list subscriptions (secrets never included) |
| POST | `/api/webhooks` | `{eventType, url, secret?, active?}` → 201; raw secret returned **once** |
| GET | `/api/webhooks/{id}` | one subscription |
| PUT | `/api/webhooks/{id}` | partial update (`eventType`, `url`, `active`; non-empty `secret` rotates the key) |
| DELETE | `/api/webhooks/{id}` | soft-delete; past delivery rows are retained |
| POST | `/api/webhooks/{id}/test` | enqueue a synthetic `org.omoikane.webhooks.ping.v1` delivery through the **same pump** |
| GET | `/api/webhooks/deliveries` | delivery log; filters `status`, `eventType`, `subscriptionId`; `limit`(≤500)/`offset` |

The gateway maps `location /api/webhooks` → `webhooks_service/webhooks`
(static URI; the mux registers /webhooks paths without the `/api` prefix,
same contract as every other service). `GET /health` + `GET /metrics`
(Phase 34 conventions) are served by the service itself and never
gateway-exposed.

## Subscription semantics

- One subscription = **one event type → one URL**. Multiplying is horizontal.
- `active:false` pauses delivery without deleting the subscription; the
  consumer skips inactive subs entirely.
- The **secret** is the HMAC-SHA256 key over the exact request body
  (`X-Omoikane-Signature: sha256=<hex>`). Unlike `ApiToken`, it MUST stay
  recoverable at rest — a hash is useless to a signer. It is stored
  **plaintext at rest** (`json:"-"`, never serialized; shown once at creation)
  — documented trade-off, encryption at rest is a follow-up.
- GORM gotcha fixed during the phase: no `gorm:"default:true"` tag on
  `Active` — GORM replaces zero values with the default tag during INSERT,
  which would silently force every subscription to `active=true` and make
  "create inactive" impossible. The default (`true`) is enforced in
  `CreateWebhook` instead.

## Delivery lifecycle

```
pending ──▶ delivered            (HTTP 2xx; attempts, httpStatus, no error)
   │
   └─▶ failed ──▶ failed ──▶ … ──▶ expired    (terminal after maxAttempts;
        (error recorded,          the module's DLQ-equivalent state)
         next attempt scheduled)
```

- **Pump defaults** (env-tunable): `pollInterval 500ms`, `batchSize 50`,
  `maxAttempts 6` (`WEBHOOKS_MAX_ATTEMPTS`), `baseBackoff 1s`
  (`WEBHOOKS_BACKOFF_CAP_SECONDS` caps the doubling at 60s),
  `WEBHOOKS_POLL_INTERVAL_MS`.
- **Backoff schedule**: attempt 1 (0s) → fail → retry at +1s → fail → retry at
  +2s → +4s → +8s → +16s → attempt 6 fails → **expired** (~31s to terminal
  for a permanently down endpoint). Every retry doubles the delay, capped.
- **Transport errors** (`ec.Do` failure, e.g. ECONNREFUSED) count as failures
  — `httpStatus` stays 0, `error` carries the reason (`request failed: ...`).
- **Non-2xx** responses count as failures too (`error: HTTP 4xx/5xx`).
- **Deleted subscription**: a due row whose subscription is gone terminates as
  `expired` (error `subscription not found`) instead of retrying forever.
- `expired` rows stay visible in the delivery log — the operator sees exactly
  how delivery degraded (attempts, error, backoff trail), which is what
  "DLQ-equivalent" means here: no raw message disappears, the ledger is the log.

## Test ping

`POST /api/webhooks/{id}/test` enqueues a `pending` delivery whose payload is a
synthetic CloudEvent (`org.omoikane.webhooks.ping.v1`, source
`webhooks-service`, subject `subscription/{id}`). It flows through the **same**
signature/retry machinery as a real delivery — the sink e2e treats it as proof
of the whole loop independent of Kafka. The ping's CloudEvent id seeds the
unique `(subscription_id, event_id)` row just like a real event.

## Demo sink (`cmd/webhook-sink`, :8091)

A tiny first-party receiver so gate tests and demos have a hermetic in-cluster
HTTP target: it 200s every POST it receives, keeps the last 50 in memory, and
exposes `GET /requests` (time, method, path, `X-Omoikane-Signature`,
content-type, body) for debugging. `GET /health` + `GET /metrics` included.
It is present in compose (own container) and in the Helm chart (Deployment on
the shared backend image, `/app/bin/webhook-sink`) with the **same Service
name** (`webhook-sink`) in both, so e2e URLs are byte-identical across gates.

## Verification

- **Go tests** (`make go-test`, 213 with Kafka up): 13 `cmd/webhooks` tests —
  mux wiring (all routes registered + admin-gated), CRUD + one-time secret
  reveal, event-type allow-list rejection, delivery-list filters/pagination,
  consumer → pending-row mapping incl. idempotent redelivery, pump step tests
  (deliver/backoff/expiry/transport-failure) with an in-memory sink, and a
  Kafka integration test (enqueue → pump → delivered) that skips cleanly when
  the broker is unreachable.
- **Compose e2e** (spec `30-webhooks.spec.ts`, desktop + mobile): create a
  subscription through the UI; publish a post → a `delivered` row exists with
  `httpStatus 200` and `attempts ≥ 1` (payload contains the title); a dead URL
  subscription retries (`attempts ≥ 2`, status `failed`, transport error
  recorded); test ping → a delivered `webhooks.ping.v1` row visible in the
  Delivery Log tab. The a11y scan suite also picks up `/admin/webhooks`.
- **K8s gate** (`make k8s-test`): the chart runs a **7th migration Job**
  (`webhooks-migrate`, inheriting the mandatory `---` separator between
  range-emitted hook docs) and the gateway/webhook-sink Service names are
  byte-identical to compose, so the same e2e passes against the minikube
  NodePort.
- **Manual HMAC check** (from the phase verification): the signature emitted
  for a payload stored in the sink equals
  `hmac.new(secret, exact_body_bytes, hashlib.sha256).hexdigest()` — the header
  is keyed HMAC over the **exact** delivered bytes, not a hash of the raw body.

## Configuration

| Env | Default | Meaning |
|---|---|---|
| `WEBHOOKS_PORT` | `8090` | HTTP port |
| `WEBHOOKS_DATABASE_URL` | `host=localhost … dbname=omoikane` | shared store (owns `webhook_*` tables) |
| `JWT_SECRET` | dev default | admin-gate signing key (same as platform) |
| `KAFKA_BROKERS` | `kafka:29092` in-network | backbone broker |
| `KAFKA_EVENTS_TOPIC` / `KAFKA_DLQ_TOPIC` | `omoikane.events` / `omoikane.events.dlq` | consumed topic; DLQ is for consumer-level failures |
| `WEBHOOKS_MAX_ATTEMPTS` | `6` | attempts before `expired` |
| `WEBHOOKS_BACKOFF_CAP_SECONDS` | `60` | backoff cap |
| `WEBHOOKS_POLL_INTERVAL_MS` | `500` | pump poll interval |
| `MIGRATE_ONLY` | — | `1` = migrate + exit (Helm migration Job) |
| `SINK_PORT` (sink) | `8091` | webhook-sink port |

## Design notes

- **Consumer-only**: no producer wiring, no outbox — webhooks is the first
  service that reads the backbone without writing domain facts to it.
- **DLQ-equivalence**: the expiry state + visible error/attempt trail replaces
  the raw Kafka DLQ for the delivery problem; the events SDK DLQ still covers
  consumer-level failures (e.g. a subscription query error).
- **Levels**: `delivered` rows are informative; `failed` rows mean the sink is
  down and recovery is automatic; `expired` rows signal a permanent problem —
  surface them in monitoring.