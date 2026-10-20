# Full-component verification (Phase 37)

A repeatable, evidence-based recheck of every component Omoikane ships. It is
**black-box**: it speaks only to the public gateway (plus, in compose mode, the
services' published ports for `/health` and `/metrics`) and exercises each layer
end to end. The goal is that any phase boundary can be re-verified in minutes
instead of trusting that "the tests passed last time".

- Tooling: `scripts/verify-components.sh` + `make verify`
- Gate: `make verify` green on **compose** and **minikube** (plus `make go-test`,
  `make test`, `make k8s-test`, `helm lint`/`template`, and CI)

## Running it

```sh
# compose (default gateway http://localhost)
make verify

# minikube / kind gateway (NodePort) — read the URL off the cluster:
make verify VERIFY_URL=http://$(minikube -p minikube ip):30080
make verify VERIFY_URL=http://localhost:30080            # kind (port-mapped)

# narrower runs
scripts/verify-components.sh --list
scripts/verify-components.sh --only auth,content
scripts/verify-components.sh --json > /tmp/verify.json

# keep the temp workspace (debugging)
scripts/verify-components.sh --keep
```

Environment knobs: `VERIFY_URL`, `VERIFY_ONLY`, `VERIFY_DIRECT`
(`auto`|`1`|`0`), `OMOKANE_ADMIN_EMAIL`/`OMOKANE_ADMIN_PASSWORD`,
`OMOKANE_SINK_URL` (demo sink base, default `http://localhost:8091`),
`OMOKANE_WEBHOOK_URL` (delivery target, default `http://webhook-sink:8091/hook`),
`OMOKANE_DEAD_WEBHOOK_URL`.

Exit code `0` means every check passed (SKIPs are allowed and reported). Any
failure prints a `Failures:` list and exits `1`.

## What it checks

| Section | Components | Representative assertions |
|---|---|---|
| `infra` | gateway, frontend | `GET /api/setup/check` 200 + `setupRequired`; `GET /` 200/3xx |
| `routing` | nginx route map | every guarded route 401 anonymously; public routes 200; unmapped `/api/*` → 404; `/metrics` → 404; **`/api/internal/*` → 404** (the internal aggregation API is never gateway-exposed); `/api/admin/blog/` reaches content; both swagger UIs; public media 404-not-401 |
| `health` | 11 services + 16 deployments | compose: each `:80xx/health` = `{"status":"ok"}`; k8s: all deployments Ready + 7 migration Jobs `1/1` |
| `auth` | auth-service | setup/login; users CRUD; register + 403 role gate; API-token create → Bearer 200 → revoke; logout |
| `content` | content-service | page draft→publish, public slug, reorder; tag + category + published post, public slug/list, admin list, like |
| `media` | media-service | multipart upload → thumbnail; public file 200 + `Cache-Control: immutable` + ETag → conditional 304; alt update |
| `messaging` | messages/settings | public contact submit → admin list; settings PUT flushes, GET `X-Cache: miss` then `hit` |
| `aggregators` | trash/dashboard | dashboard + stats shapes; page soft-delete → trash list/count → restore → hard-delete; unknown entity 400 |
| `events` | outbox→relay→Kafka→audit | publish a post → poll `/api/audit-logs?entity=post&search=<slug>` for `action=publish`; **exactly-once on the wire** (a topic-tailer joins a per-run group *before* the publish and counts occurrences of the run's unique slug — `1` = pass, `>1` = duplicate relay, `0` = watcher raced and is reported SKIP) |
| `webhooks` | webhooks-service | unknown eventType 400; delivery → `delivered` (http 200); dead endpoint → backoff → `expired`; test ping; **HMAC recomputed** against the sink's recorded signature; delivery-log filter |
| `observability` | all services | `/metrics` reachable pod-direct with `http_requests_total` + `route="/users"` normalisation; never gateway-exposed; **service logs are JSON with a `service` attribute** (GORM/`net/http` stderr lines are excluded from the pipeline, so unstructured text is a failure); `/internal/*` never gateway-exposed (404) |

## Component matrix

For each component: the automatic check (above) and the manual check a human
should still eyeball. Manual checks are the judgment calls a script cannot make
(visual correctness, usability, copy, runbook accuracy).

| Component | Automatic | Manual |
|---|---|---|
| nginx gateway | route map + fail-loud | confirm `X-Cache`, redirects and `/media/` in a browser |
| postgres / redis / kafka | health + topics + pipeline events | inspect Kafka topic via `kafka-topics.sh`; Redis `X-Cache` |
| auth-service | setup/login/users/tokens/role gate | login/logout/forgot-password screens, avatar refresh |
| content-service | pages/blog CRUD + events | editor toolbar, pointer-drag reorder, publish badges, draft visibility |
| media-service | upload/thumb/file/alt | grid, multi-upload, bulk delete, alt dialog, image insert in editor |
| messages-service | contact + contacts list | contact form validation + reCAPTCHA, messages spinner |
| settings-service | GET/PUT + cache | site settings form, favicon, email-template editor |
| trash aggregator | fan-out restore/hard-delete | badge refresh (`trash-changed`), entity chips |
| dashboard aggregator | stats shape | dashboard cards + charts, key warnings absent |
| audit-service | Kafka consumer → audit row | audit-logs page: filters, chips, action colours |
| webhooks-service | allow-list/delivery/HMAC/retry/ping | subscription UI, one-time secret copy, delivery-log filters |
| docs-service | swagger UI 200 | browse endpoints + auth scheme |
| frontend | `GET /` | every admin + public page, responsive/mobile, axe a11y |
| observability | `/metrics`, JSON logs | `kubectl logs`/compose logs readability |
| CI/CD | (re-run on `main`) | Actions run green, GHCR tags, `cloud-smoke` |

## Manual checklist (run at 37c)

Setup → login → logout; all 11 admin pages (`/admin`, users, pages, blog,
media, messages, contacts, settings, api-tokens, trash, audit-logs, webhooks);
all public/auth routes (home, page, blog list + detail, contact, settings,
preview, login, register, setup, forgot/reset password). Cross-cutting: rich-text
editor, pointer-drag page reorder, multi-upload, favicon, avatar refresh, trash
badge, responsive/mobile layout, axe a11y scan, both swagger UIs, JSON-log
readability, HMAC/webhook UI. Run in full on compose; spot-check on minikube.

## Notes

- **Direct mode** is auto-detected: `compose` when `localhost:8082/health`
  answers, `k8s` when the `omoikane` namespace is reachable, otherwise the
  direct checks are skipped (the gateway checks still run).
- **reCAPTCHA**: when a secret is configured, contact/register return 400 and the
  script reports SKIP rather than FAIL (the gateway path is still verified).
- **HMAC**: the demo sink records the `X-Omoikane-Signature` header and the raw
  body; the script recomputes HMAC-SHA256 over the delivered body and compares.
- The script never mutates anything it does not clean up: created users, pages,
  posts, tags, categories, media, contacts and webhook subscriptions are deleted
  at the end (unless `--keep`).

## Phase 37 findings (all fixed at source)

Every finding below was reproduced, fixed, **and given a regression test** — no
gate was weakened to make the result green.

| # | Finding | Root cause | Fix | Regression test |
|---|---|---|---|---|
| F1 | `make up` restarted nginx *before* the health waits and never probed the gateway — a dead/stale upstream could hide until Playwright failed mysteriously. `audit-service` and `webhook-sink` were not waited on at all. | Copy-pasted wait loops; ordering drift. | `wait_health`/`wait_http` Makefile macros; waits for all 11 services incl. audit (`:8081`) + webhook-sink (`:8091`); nginx restarted **after** the healths; then two real gateway assertions (`/api/setup/check` → auth, `/api/swagger/` → docs) that fail `make up` on 502. | `make up` itself (fails loudly) |
| F2 | A single transient container start failure (module download, DNS blip, broker race) left a service dead forever and cascaded into gateway 502s. | No restart policy in compose. | `restart: unless-stopped` on all 16 services. | compose config validation |
| F3 | **Duplicate events on the wire.** Five services each poll the ONE shared `outbox_events` table with `WHERE status='pending'`; two relays could claim the same row in the gap before `MarkSent`, publishing it twice (observed: 38 messages, 28 distinct ids, 10 ids twice). | Uncoordinated concurrent `Pending()` reads. | `Pending()` now **atomically claims** rows inside a transaction (`SELECT … FOR UPDATE SKIP LOCKED`, `ORDER BY id ASC`, `LIMIT n`) and flips them to `inflight`; stale `inflight` rows (> `OutboxInflightTimeout` = 2 min) are reclaimed. Correct under 5 relays *and* multi-replica HPA. | `TestOutboxStore_PendingClaimsRows`, `TestOutboxStore_ReclaimsStaleInflightRows`, `TestOutboxStore_PendingRespectsLimit`, `TestRelay_ConcurrentRelaysPublishEachEventOnce` (3 relays × 5 runs, 20 events → each published exactly once), `TestRelay_FailedPublishReleasesClaimForRetry` |
| F4 | The two Kafka consumers (audit, webhooks) did `Create`-then-check-exists for idempotency — a check-then-act TOCTOU under redelivery. | Non-atomic dedupe. | `clause.OnConflict{DoNothing: true}` on the create; the unique index is the arbiter. Dead `alreadyStored`/`alreadyEnqueued` helpers removed. | existing redelivery-idempotency tests (audit + webhooks) |
| F5 | **DB errors escaped the JSON pipeline as raw ANSI text** (GORM's stock logger owns a separate stdlib logger; 62 of 66 audit-service lines were unstructured). Ordinary "record not found" control flow was logged at ERROR. | `logger.Default.LogMode(…)`. | `observability.GormLogger(level)` → GORM v1.31's upstream `logger.NewSlogLogger(slog.Default(), …)` with `IgnoreRecordNotFoundError: true`; wired at all 8 `gorm.Open` sites. | `internal/observability/gorm_test.go` (7 tests), plus the verify sweep's JSON-logs assertion |
| F6 | `e2e/12-user-status.spec.ts` sort-by-status flaked: `aria-sort` was null. | Documented click-before-hydration race (the click landed on server-rendered markup). | `waitForHydration(page, "table thead th")` after each `/admin/users` navigation (a `gotoUsers` helper). Assertion unchanged. | the spec itself |
| F7 | `http: superfluous response.WriteHeader call` polluted the JSON logs. | The Phase 34 `statusRecorder` forwarded **every** `WriteHeader`, so a second status reached `net/http`. | The recorder swallows repeats (only the first status passes through) and records the effective status; `Write`/`Flush` handled. | `TestStatusRecorder_SwallowsRepeatedWriteHeader`, `TestStatusRecorder_ImplicitOK` |
| F8 | The verify sweep itself had two latent bugs: `printf | grep -q` under `set -o pipefail` reported failure whenever `grep` matched and exited before `printf` finished (SIGPIPE), and the `0/TOTAL` success string was parsed as a failure. | Shell pipeline semantics. | `expect_contains` uses bash `[[ … == *…* ]]` (no pipe); all other `curl|grep -q` comparisons converted; the count split fixed. The Kafka topic-tailer now waits until it has **joined its consumer group** (per-run group name ⇒ always starts at the tail) instead of sleeping a guessed 4s. | three consecutive 118/118 runs |
| F9 | **`make verify` timed out on the dead-endpoint retry ladder on minikube** ("dead endpoint retried then expired") although the same check passed on compose. The delivery row was mid-backoff (`failed`, attempts 3–4) when the 150s budget expired. | In compose `http://webhook-sink:8099` is refused instantly, so the 1+2+4+8+16+32s ladder runs in ~90s. In k8s the hostname resolves through kube-proxy but the connection then **hangs**, so every one of the 6 attempts burns the delivery client's full 10s timeout: worst case ≈ 6×10s + 63s backoff + poll jitter ≈ 130s — past the budget, particularly on a cold cluster. | The `expired` poll budget is computed from the worst-case ladder, not guessed: raised to **300s** (and the `delivered` poll to 90s for headroom). The assertion is unchanged — the ladder is correct, only the wait was too tight. | the sweep's webhooks section on minikube (green) |
| F10 | **Webhook delivery head-of-line blocking** — on minikube a *healthy* delivery intermittently timed out ("webhook delivered to sink") alongside the dead-endpoint check, and even a delivered row showed `created_at → updated_at` gaps of ~70s. | `deliveryWorker.RunOnce` delivered its batch **sequentially** (`for i := range due { w.deliver(ctx, &due[i]) }`). In compose a dead endpoint is refused in ~0s; on k8s it resolves-then-stalls for the full 10s client timeout, so each accumulated dead subscription pushed every other due row in the same batch behind it (N×10s). | `RunOnce` now fans the batch out to a **bounded worker pool** (`concurrency`, default 8, `WEBHOOKS_DELIVERY_CONCURRENCY`) and awaits all deliveries before returning — one slow/dead subscriber can no longer delay the rest, while the pump step stays complete and deterministic. | `TestDeliveryPump_SlowSubscriberDoesNotBlockOthers` (a blocked slow endpoint + a healthy one in one batch; fails with the old sequential delivery, passes with the pool) |
| F11 | The verify sweep's **HMAC signature check silently SKIPped on minikube** ("sink has not recorded this delivery yet") while the very same delivery was in the sink — the sweep was reading the **compose** sink. | `section_webhooks` never called `detect_direct`, so `DIRECT_MODE` was empty and `sink_fetch` fell through to `curl "$SINK_BASE/requests"` = `http://localhost:8091`. On a machine running compose **and** the cluster at once, that is the compose sink (stale, 30 KB, frozen). Compounding it, `detect_direct` probed `localhost:8082/health` first, so it would have chosen compose even had it run. | `section_webhooks` now calls `detect_direct`; `detect_direct` decides from the **gateway being verified** (`BASE_URL`): a non-local host or a `:30080` NodePort ⇒ k8s, a plain localhost root ⇒ compose (probe kept only for that local case, without a `curl|grep -q` pipe). The fetch also retries until the run's slug is visible. HMAC now verifies on both stacks (12/12, 0 skips). | the sweep's webhooks section on minikube (HMAC verified instead of SKIP) |
| F12 | Once F11 let the HMAC check reach the **right** sink, it reported **`HMAC signature mismatch`** on minikube although our delivery was correctly signed — a false FAIL on a real, correct delivery. | The sink received the post slug from *several* subscriptions: our run's (secret `$dsecret`) **plus leftover verify subscriptions from earlier runs**, each signing with a *different* secret. The verifier walked records newest-first and stopped at the **first** slug match, then asserted it against *our* secret — but that first record had been signed by a leftover subscription, so our secret could never reproduce it. A slug match does not imply *we* delivered it. | The verifier now scans **all** slug-matching records and passes as soon as **any** validates against our secret; it reports `mismatch` only when a *signed* matching record exists that our secret cannot validate, and `not-found` when the only matches are unsigned or absent. (Leftover subscriptions are an accumulation nuisance the sweep's normal cleanup keeps in check; a run interrupted with Ctrl-C can leave some behind — harmless now that the verifier no longer trusts the first match.) | the sweep's webhooks section on minikube (green, 12/12, 0 skips) |

> **Operational note:** `make verify` against the **k8s** gateway must not run while `make k8s-test` is in flight — that target runs `k8s-db-reset` twice (dropping the databases and restarting the DB-owning deployments), so a sweep overlapping it sees transient 5xx/empty-DB failures. The one "106 passed / 1 failed" minikube run recorded during this phase was exactly such a collision; the clean re-run is 107/0/0.

## Results

| Gate | Where | Result |
|---|---|---|
| `make verify` | compose | **118 checks, 0 failed, 0 skipped** — three consecutive green runs |
| `make go-test` | host (Kafka up) | **234/234 pass** (was 220; +14 Phase 37 tests) |
| `make test` | compose (desktop + mobile) | **desktop 284 passed / 8 skipped / 0 failed; mobile 276 passed / 16 skipped / 0 failed** |
| `make verify` | minikube (NodePort) | **107 checks, 0 failed, 0 skipped** (compose runs 118: 11 checks are compose-host-only — published service ports / direct `curl` — and have in-cluster equivalents on k8s) |
| `make k8s-test` | minikube | **green** — desktop then mobile through the NodePort (mobile 276 passed / 16 skipped; desktop phase precedes it in the same target, so a desktop failure would abort before mobile) |
| `helm lint` / `helm template` | — | **green** — lint 0 failed; 45 kinds default + minikube/kind/eks/gke/aks overlays; 7 migration Jobs render |
| CI (`deploy.yml`) | GitHub Actions | Phase 36 pipeline unchanged (go-test → kind e2e → GHCR build/push → helm deploy); not re-run in this phase |
| `scripts/cloud-smoke.sh` | live gateway | Phase 36 script unchanged; exercised end-to-end against the minikube gateway (create sub → publish → delivered → audit row) as the `cloud-smoke` flow equivalent |
