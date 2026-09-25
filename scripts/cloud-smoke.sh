#!/usr/bin/env bash
#
# Omoikane Phase 36 — FIRST RESULT cloud smoke.
#
# Verifies the flagship end-to-end chain against any live deployment gateway:
#
#   setup/login -> webhook subscription (post.published) -> publish a post
#     -> post hit the gateway -> outbox -> Kafka relay -> webhooks consumer
#     enqueues a delivery -> delivery pump POSTs it to the in-cluster demo sink
#     -> delivery log row becomes `delivered` (httpStatus 200)
#     -> audit service consumed the SAME event on group `audit` -> audit trail
#        shows action=publish for the post.
#
# Nothing cloud-specific: it speaks only to the public gateway API, so it also
# serves as the compose/minikube acceptance check that runs *before* the
# Playwright gate (the deploy workflow runs it after helm upgrade).
#
# Usage:
#   scripts/cloud-smoke.sh [BASE_URL]                 # default http://localhost
#   # Optional env knobs:
#   #   OMOKANE_ADMIN_EMAIL / OMOKANE_ADMIN_PASSWORD  (defaults admin@example.com / SecurePass123!)
#   #   OMOKANE_ALT_ADMIN_PASSWORD                    (fallback admin password to try)
#   #   OMOKANE_WEBHOOK_URL                           (default http://webhook-sink:8091/hook —
#   #                                                  only valid *inside* the cluster/compose net)
#   #   OMOKANE_SINK_URL                              (set to also finger the sink directly, e.g.
#   #                                                  http://localhost:8091 via kubectl port-forward)
#
# Exit code 0 = FIRST RESULT verified. Every assertion fails loudly.
set -euo pipefail

BASE_URL="${1:-http://localhost}"
BASE_URL="${BASE_URL%/}"
ADMIN_EMAIL="${OMOKANE_ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${OMOKANE_ADMIN_PASSWORD:-SecurePass123!}"
ALT_PASSWORD="${OMOKANE_ALT_ADMIN_PASSWORD:-NewPass123!}"
WEBHOOK_URL="${OMOKANE_WEBHOOK_URL:-http://webhook-sink:8091/hook}"
# Canonical CloudEvents type (backend/internal/events/events.go) — the webhook
# allow-list only accepts these exact strings.
WEBHOOK_EVENT_TYPE="${OMOKANE_WEBHOOK_EVENT_TYPE:-org.omoikane.content.post.published.v1}"
SINK_URL="${OMOKANE_SINK_URL:-}"

TMPD="$(mktemp -d)"
COOKIE="$TMPD/cookies.txt"
trap 'rm -rf "$TMPD"' EXIT

CURL=(curl -sS --max-time 20)

step() { printf '\n==> %s\n' "$*"; }
ok()   { printf '    \033[32mOK\033[0m %s\n' "$*"; }
info() { printf '    %s\n' "$*"; }
die()  { printf '\n\033[31mFAIL\033[0m %s\n' "$*" >&2; exit 1; }

# Expect a specific HTTP code; die with the body otherwise.
expect_status() {
  local want="$1" got="$2" label="$3"
  if [ "$got" != "$want" ]; then
    info "response body was:"
    cat "$TMPD/body.json" >&2 || true
    die "$label: expected HTTP $want, got $got"
  fi
}

# API call helper. Writes the response body to $TMPD/body.json, prints the code.
api() {
  local method="$1" path="$2" body_file="${3:-}"
  if [ -n "$body_file" ]; then
    "${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' -X "$method" \
      -H 'Content-Type: application/json' -c "$COOKIE" -b "$COOKIE" \
      --data-binary @"$body_file" "$BASE_URL$path"
  else
    "${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' -X "$method" \
      -c "$COOKIE" -b "$COOKIE" "$BASE_URL$path"
  fi
}

# Minimal dotted-path JSON lookup (no jq dependency) — prints the VALUE, or
# prints nothing and exits 1 when the path is missing (e.g. empty arrays).
#   json_get FILE 'deliveries[0].status'
json_get() {
  python3 - "$1" "$2" <<'PY' 2>/dev/null || true
import json, sys
doc = json.load(open(sys.argv[1]))
cur = doc
for part in sys.argv[2].split('.'):
    if not part:
        continue
    if '[' in part:
        key, _, rest = part.partition('[')
        if key:
            cur = cur[key]
        cur = cur[int(rest.rstrip(']'))]
    else:
        cur = cur[part]
if isinstance(cur, str):
    print(cur)
else:
    print(json.dumps(cur))
sys.exit(0)
PY
}

# Poll `<path>` until `json_get <file> <expr>` equals `<expected>` (timeout 60s
# default). The body of every poll lands in $TMPD/poll.json.
poll_until() {
  local path="$1" expr="$2" expected="$3" timeout="${4:-60}"
  local deadline=$(( $(date +%s) + timeout ))
  local got
  while :; do
    "${CURL[@]}" -o "$TMPD/poll.json" -w '%{http_code}' -c "$COOKIE" -b "$COOKIE" \
      "$BASE_URL$path" >/dev/null
    got=$(json_get "$TMPD/poll.json" "$expr") || got=""
    if [ "$got" = "$expected" ]; then
      return 0
    fi
    if [ "$(date +%s)" -ge "$deadline" ]; then
      info "last response body:"
      cat "$TMPD/poll.json" >&2 || true
      die "timeout waiting for $path .$expr == '$expected' (got '$got')"
    fi
    sleep 2
  done
}

# shellcheck disable=SC2016
TS="$(date +%s)"
POST_TITLE="Cloud Smoke Post $TS"
POST_SLUG="cloud-smoke-$TS"

printf '\n\033[1mOmoikane FIRST RESULT smoke\033[0m — gateway: %s\n' "$BASE_URL"
printf '    admin: %s | sink: %s | webhook url: %s\n' "$ADMIN_EMAIL" "${SINK_URL:-(not fingerprinted)}" "$WEBHOOK_URL"

# --- 1. Gateway reachable + setup/login -----------------------------------
step "1/7 Gateway reachable"
code=$(api GET /api/setup/check)
expect_status 200 "$code" "GET /api/setup/check"
setup_required=$(json_get "$TMPD/body.json" "setupRequired")
info "setupRequired=$setup_required"
ok "gateway up"

step "2/7 Admin present (setup or login)"
if [ "$setup_required" = "True" ]; then
  printf '{"email":"%s","password":"%s"}' "$ADMIN_EMAIL" "$ADMIN_PASSWORD" > "$TMPD/setup.json"
  code=$(api POST /api/setup "$TMPD/setup.json")
  expect_status 200 "$code" "POST /api/setup"
  info "created initial admin"
fi

for pw in "$ADMIN_PASSWORD" "$ALT_PASSWORD"; do
  printf '{"email":"%s","password":"%s"}' "$ADMIN_EMAIL" "$pw" > "$TMPD/login.json"
  code=$(api POST /api/auth/login "$TMPD/login.json")
  if [ "$code" = "200" ]; then
    info "logged in with password '...${pw: -3}'"
    break
  fi
  if [ "$pw" = "$ALT_PASSWORD" ]; then
    die "login failed with both known admin passwords (HTTP $code)"
  fi
done
ok "authenticated (session cookie saved)"

# --- 3. Webhook subscription ----------------------------------------------
step "3/7 Create webhook subscription ($WEBHOOK_EVENT_TYPE -> sink)"
printf '{"eventType":"%s","url":"%s","active":true}' "$WEBHOOK_EVENT_TYPE" "$WEBHOOK_URL" > "$TMPD/wh.json"
code=$(api POST /api/webhooks "$TMPD/wh.json")
expect_status 201 "$code" "POST /api/webhooks"
SUB_ID=$(json_get "$TMPD/body.json" "id")
[ -n "$SUB_ID" ] || die "webhook create returned no id"
SECRET=$(json_get "$TMPD/body.json" "secret")
[ -n "$SECRET" ] || die "webhook create returned no one-time secret"
info "subscription id=$SUB_ID (secret shown once, $((${#SECRET})) chars)"
ok "subscription created"

# --- 4. Publish a real post ------------------------------------------------
step "4/7 Publish a post (status=published -> post.published on Kafka)"
printf '{"title":"%s","slug":"%s","content":"<p>first result smoke</p>","status":"published"}' \
  "$POST_TITLE" "$POST_SLUG" > "$TMPD/post.json"
code=$(api POST /api/blog/posts "$TMPD/post.json")
expect_status 201 "$code" "POST /api/blog/posts"
POST_ID=$(json_get "$TMPD/body.json" "id")
info "post id=$POST_ID title='$POST_TITLE'"
ok "post published"

# --- 5. Webhook delivery ---------------------------------------------------
step "5/7 Wait for webhook delivery (eventType=$WEBHOOK_EVENT_TYPE, status=delivered)"
poll_until "/api/webhooks/deliveries?subscriptionId=$SUB_ID&eventType=$WEBHOOK_EVENT_TYPE" \
  "deliveries[0].status" "delivered" 60
HTTP_STATUS=$(json_get "$TMPD/poll.json" "deliveries[0].httpStatus")
ATTEMPTS=$(json_get "$TMPD/poll.json" "deliveries[0].attempts")
[ "$HTTP_STATUS" = "200" ] || die "delivery httpStatus expected 200, got $HTTP_STATUS"
info "delivered httpStatus=$HTTP_STATUS attempts=$ATTEMPTS (payload delivered to $WEBHOOK_URL)"
ok "end-to-end webhook delivery (outbox -> Kafka -> consumer -> pump -> sink)"

# --- 6. Audit trail ----------------------------------------------------------
step "6/7 Wait for audit row (action=publish from the SAME event)"
poll_until "/api/audit-logs?entity=post&search=$POST_SLUG" "logs[0].action" "publish" 60
info "audit: logs[0].action=publish for '$POST_SLUG'"
ok "audit consumer mapped post.published -> publish row"

# --- 7. Optional sink fingerprint + cleanup ----------------------------------
step "7/7 Cleanup + sink fingerprint"
if [ -n "$SINK_URL" ]; then
  SINK_BODY=$("${CURL[@]}" "$SINK_URL/requests")
  printf '%s' "$SINK_BODY" > "$TMPD/sink.json"
  NREQ=$(python3 -c "import json;print(len(json.load(open('$TMPD/sink.json'))))" 2>/dev/null || echo "?")
  info "sink recorded $NREQ request(s) total"
  ok "sink reachable at $SINK_URL"
fi
code=$(api DELETE "/api/webhooks/$SUB_ID")
expect_status 204 "$code" "DELETE /api/webhooks/$SUB_ID"
code=$(api DELETE "/api/blog/posts/$POST_ID")
if [ "$code" != "200" ] && [ "$code" != "204" ]; then
  die "DELETE /api/blog/posts/$POST_ID: expected HTTP 200/204, got $code"
fi
ok "cleaned up subscription + test post"

printf '\n\033[1;32mFIRST RESULT verified\033[0m — setup -> login -> post.published -> Kafka -> webhook delivered -> audit publish row.\n'