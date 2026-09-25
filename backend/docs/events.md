# Omoikane Event Catalog (Phase 32 — live backbone)

The platform communicates domain facts over a **Kafka backbone** using the
[CloudEvents 1.0](https://github.com/cloudevents/spec/blob/v1.0.1/cloudevents/spec.md)
envelope (`internal/events` SDK). Events are **transactionally appended to an
outbox** inside the business DB write, then a per-service **relay** publishes
them to the `omoikane.events` topic. Consumers — the audit-service (group
`audit`) and, since Phase 35, the webhook delivery service (group `webhooks`) —
read the same topic with their own consumer group.

---

## Topology

```
         business tx                         relay                         consumer group
┌─────────────────────────┐   ┌──────────────────────────┐   ┌───────────────────────────────┐
│ service (auth/content…) │   │ events.Relay (per svc)   │   │ events.Consumer "audit"       │
│  DB write               │──▶│  outbox.Pending → Kafka  │──▶│  audit-service (cmd/audit)    │
│  + outbox.Enqueue (tx)  │   │  omoikane.events         │   │  + DLQ on failure             │
└─────────────────────────┘   └──────────────────────────┘   └───────────────────────────────┘
```

- **Topic**: `omoikane.events` (config `KAFKA_EVENTS_TOPIC`; default
  `omoikane.events`).
- **Dead-letter topic**: `omoikane.events.dlq` (`KAFKA_DLQ_TOPIC`) — messages
  that exhaust the consumer retry budget land here with their `ce-type` header
  preserved.
- **Delivery**: at-least-once, keyed by `subject` (so per-entity ordering is
  preserved). **Consumers must be idempotent** — the audit-service dedupes on
  the CloudEvent `id` (unique index on `AuditLog.EventID`); the webhooks
  service dedupes on the unique `(subscription_id, event_id)` delivery index.
  A group's committed offsets persist in Kafka across DB resets, so neither
  consumer re-ingests history after a reset.
- **Brokers**: single-node KRaft. Services inside compose must use
  `KAFKA_BROKERS=kafka:29092` (the internal advertised listener); host tooling
  and integration tests use `localhost:9092`. Topics are provisioned
  idempotently (`events.EnsureTopics`) at service startup — **log-only on
  failure**, so a down broker never fatal a service.

## Envelope

| Field | Value |
|---|---|
| `specversion` | `1.0` |
| `id` | random 128-bit hex (`events.NewEventID`) — duplicate id ⇒ redelivery |
| `source` | one of `auth-service`, `content-service`, `media-service`, `messages-service`, `settings-service` |
| `type` | `org.omoikane.<domain>.<event>.v1` — see catalog below |
| `subject` | entity reference, e.g. `post/42` |
| `time` | RFC 3339 UTC |
| `data` | JSON payload — JSON Schema per type in `internal/events/schemas/*.json` |

## Live events (Phase 32)

| Type (`events.Type…`) | Source | Subject | Payload (schema) | Emitted by | Audit mapping |
|---|---|---|---|---|---|
| `org.omoikane.auth.user.registered.v1` | auth-service | `user/{id}` | {id, email, role} (`user.registered.json`) | Setup / Register / CreateUser | action `register`, entity `user` |
| `org.omoikane.auth.login.v1` | auth-service | `user/{id}` | {id, email, method} (`auth.login.json`) | Login (method `cookie`) | action `login`, entity `user` |
| `org.omoikane.content.page.published.v1` | content-service | `page/{id}` | {id, slug, title, status, publishedAt} (`page.published.json`) | page create-as-published / draft→published / batch publish | action `publish`, entity `page` |
| `org.omoikane.content.post.published.v1` | content-service | `post/{id}` | {id, slug, title, status, categoryId, tagIds, publishedAt} (`post.published.json`) | post create-as-published / draft→published / batch publish | action `publish`, entity `post` |
| `org.omoikane.media.uploaded.v1` | media-service | `media/{id}` | {id, filename, alt, url, thumbUrl, size, uploadedAt} (`media.uploaded.json`) | media upload | action `upload`, entity `media` |
| `org.omoikane.messages.contact.received.v1` | messages-service | `contact/{id}` | {id, name, email, subject, message, receivedAt} (`contact.received.json`) | public contact form | action `contact`, entity `contact` |
| `org.omoikane.settings.updated.v1` | settings-service | `settings/1` | {siteName, tagline, blogEnabled} (`settings.updated.json`) | admin `PUT /settings` | action `update`, entity `settings` |

### Schemas without a live producer yet

`page.updated.json`, `post.updated.json` (double-log with `.published`) and
`message.created.json` are catalogued and schema-frozen but **not emitted** —
fine-grained CRUD events are deferred to a later phase.

## Audit-service consumer (`cmd/audit`)

- Consumer group **`audit`**, reading `omoikane.events`, handler
  `auditEventHandler` in `cmd/audit/audit_consumer.go`.
- **Start offset**: a brand-new group starts at `kafka.LastOffset`
  (`Config.ConsumerStartOffset`) so first deployment does not replay the
  historical backlog; committed offsets (stored in Kafka) are respected
  thereafter and survive DB resets.
- **Mapping** (`mapEventToLog`): a pure type→`AuditLog` conversion.
  - Unknown/out-of-scope types → skipped (no row, not an error).
  - Mapped type with an unparseable payload → **error** (retry → DLQ), so a
    poison message surfaces in the DLQ topic instead of disappearing.
  - `AuditLog.EventID` carries the CloudEvent `id` and is **unique-indexed**;
    a redelivered duplicate is detected (`alreadyStored`) and treated as
    success so the consumer commits the offset.
  - Actor columns: payloads carry entity identity, not the acting user —
    `user_name` is filled from the payload where present (register/login/
    contact) and defaults to `system` otherwise (documented follow-up: actor
    meta in event payloads).
- **Resilience**: `events.EnsureTopics` log-only at startup; `runConsumerLoop`
  reconnects with a 2s backoff on transient errors; graceful shutdown on
  SIGINT/SIGTERM closes the consumer after the HTTP server drains.
- The retired HTTP write path (`POST /events`, `internal/audit`) is gone —
  **Kafka is the single audit write path** since Phase 32.

## Webhooks consumer (`cmd/webhooks`)

- Consumer group **`webhooks`**, reading `omoikane.events`, handler
  `webhookEventHandler` in `cmd/webhooks/consumer.go`.
- **Match**: every **active** subscription whose `event_type` equals the event
  type gets one `pending` `WebhookDelivery` row (no subscribers → fast ACK).
- **Idempotency**: unique index `(subscription_id, event_id)`; a redelivered
  event whose rows already exist is success (consumer commits, no DLQ).
- **No replay**: `ConsumerStartOffset = kafka.LastOffset` for a fresh group;
  committed offsets honored thereafter (survives DB resets — subscriptions
  themselves live in the DB, so the e2e gate re-creates them after a reset and
  publishes after subscribing).
- **Delivery is decoupled**: the consumer only enqueues; the pump (`delivery.go`)
  POSTs with HMAC-SHA256 signing, exponential backoff, and a terminal `expired`
  state (the DLQ-equivalent). See [`webhooks.md`](./webhooks.md).

## Reading the log

`GET /api/audit-logs` (`audit-service:8081` via the gateway, admin JWT) lists
rows with entity/action/userId/search filters and pagination. Rows serialize
with **camelCase JSON keys** (`id`, `createdAt`, `userName`, `action`,
`entityType`, `entityId`, `detail`, …) — `models.AuditLog` declares explicit
json tags for exactly this contract, so the admin UI at `/admin/audit-logs`
renders rows with action chips
(`register`/`publish`/`upload`/`contact`/`update`/`login` + legacy create/delete
etc.) and entity filter tabs (…, `settings`).

## Verifying the pipeline (manual gateway smoke)

1. `make up` (compose brings up Kafka + the services + audit consumer).
2. Publish any page/post via the admin API or UI.
3. Confirm the row arrives through the consumer, not HTTP:
   `curl -s -H 'Cookie: session=<admin jwt>' http://localhost/api/audit-logs?entity=post`
4. Watch the topic directly on the host:
   `kcat -b localhost:9092 -t omoikane.events -C` (or the kafka console tools).