#!/usr/bin/env bash
#
# Omoikane Phase 37 — full-component recheck.
#
# A repeatable, black-box verification of every component built through Phase 36.
# It speaks only to the public gateway (plus, in compose mode, the services'
# published ports for /health and /metrics) and exercises each layer end to end:
#
#   infra        gateway + frontend reachability, service health, deployment state
#   routing      every gateway route group lands on its owning service; fail-loud
#                404s for unmapped /api/*; /metrics never exposed
#   auth         setup/login/logout, users CRUD, register+role gate, API tokens
#   content      pages (draft->publish, reorder) + blog (tags/categories/likes)
#   media        upload -> thumbnail -> public file (ETag/304/Cache-Control) -> alt
#   messaging    contact submit -> contacts list; settings X-Cache miss->hit
#   aggregators  dashboard facade shape; trash fan-out restore/hard-delete
#   events       post.published -> outbox -> Kafka -> audit row, and the event
#                appears on the wire EXACTLY once (no relay fan-out)
#   webhooks     allow-list, delivery, HMAC signature, retry->expired, test ping
#   observability/metrics pod-direct, route normalisation, gateway 404,
#                JSON-structured service logs (every line carries `service`)
#
# Usage:
#   scripts/verify-components.sh [BASE_URL] [--only a,b] [--json] [--keep] [--list]
#   make verify                                    # compose (http://localhost)
#   make verify VERIFY_URL=$(K8S_GATEWAY_URL)      # minikube NodePort
#
# Environment knobs:
#   VERIFY_URL                    base URL (default http://localhost)
#   VERIFY_ONLY                   comma-separated section list
#   VERIFY_DIRECT                 auto (default) | 1 | 0 — direct-port checks
#   OMOKANE_ADMIN_EMAIL/PASSWORD  admin credentials (SecurePass123! / NewPass123!)
#   OMOKANE_SINK_URL              demo sink base, e.g. http://localhost:8091
#   OMOKANE_WEBHOOK_URL           delivery target used by the delivery subscription
#   OMOKANE_SINK_PORT             demo sink port published on localhost (compose)
#
# Exit code 0 = every check passed (SKIPs allowed). Every failure is printed.
set -o pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BASE_URL="${VERIFY_URL:-http://localhost}"
ADMIN_EMAIL="${OMOKANE_ADMIN_EMAIL:-admin@example.com}"
ADMIN_PASSWORD="${OMOKANE_ADMIN_PASSWORD:-SecurePass123!}"
ALT_PASSWORD="${OMOKANE_ALT_ADMIN_PASSWORD:-NewPass123!}"
SINK_BASE="${OMOKANE_SINK_URL:-http://localhost:8091}"
SINK_PORT="${OMOKANE_SINK_PORT:-8091}"
DELIVERY_URL="${OMOKANE_WEBHOOK_URL:-http://webhook-sink:8091/hook}"
DEAD_URL="${OMOKANE_DEAD_WEBHOOK_URL:-http://webhook-sink:8099/hook}"
POST_EVENT_TYPE="${OMOKANE_WEBHOOK_EVENT_TYPE:-org.omoikane.content.post.published.v1}"
ONLY="${VERIFY_ONLY:-}"
DIRECT="${VERIFY_DIRECT:-auto}"

JSON_OUT=0
KEEP=0
LIST=0

while [ $# -gt 0 ]; do
  case "$1" in
    --json) JSON_OUT=1; shift ;;
    --keep) KEEP=1; shift ;;
    --list) LIST=1; shift ;;
    --only) ONLY="${2:-}"; shift 2 ;;
    --only=*) ONLY="${1#*=}"; shift ;;
    -h|--help) sed -n '2,45p' "$0"; exit 0 ;;
    http://*|https://*) BASE_URL="${1%/}"; shift ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
BASE_URL="${BASE_URL%/}"

ALL_SECTIONS="infra routing health auth content media messaging aggregators events webhooks observability"

if [ "$LIST" = "1" ]; then
  echo "sections: $ALL_SECTIONS"
  exit 0
fi

# ---------------------------------------------------------------------------
# Reporting
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YELLOW=$'\033[33m'
  C_BOLD=$'\033[1m'; C_DIM=$'\033[2m'; C_RESET=$'\033[0m'
else
  C_RED=""; C_GREEN=""; C_YELLOW=""; C_BOLD=""; C_DIM=""; C_RESET=""
fi

CUR=""
PASS=0; FAIL=0; SKIP=0
TMPD="$(mktemp -d)"
COOKIE="$TMPD/cookies.txt"
COOKIE2="$TMPD/cookies2.txt"
RESULTS="$TMPD/results.txt"
: > "$RESULTS"
FAILURES="$TMPD/failures.txt"
: > "$FAILURES"

cleanup() { [ "$KEEP" = "1" ] || rm -rf "$TMPD"; }
trap cleanup EXIT

pass() { PASS=$((PASS + 1)); printf '%s|%s|%s\n' PASS "$CUR" "$1" >> "$RESULTS"; printf '  %s✓%s %s\n' "$C_GREEN" "$C_RESET" "$1"; }
fail() { FAIL=$((FAIL + 1)); printf '%s|%s|%s\n' FAIL "$CUR" "$1" >> "$RESULTS"; printf '%s|%s|%s\n' FAIL "$CUR" "$1" >> "$FAILURES"; printf '  %s✗%s %s\n' "$C_RED" "$C_RESET" "$1"; }
skip() { SKIP=$((SKIP + 1)); printf '%s|%s|%s\n' SKIP "$CUR" "$1" >> "$RESULTS"; printf '  %s-%s %s\n' "$C_YELLOW" "$C_RESET" "$1"; }
info() { printf '  %s%s%s\n' "$C_DIM" "$*" "$C_RESET"; }

section() { CUR="$1"; printf '\n%s[%s]%s %s\n' "$C_BOLD" "$1" "$C_RESET" "$2"; }

enabled() {
  [ -z "$ONLY" ] && return 0
  case ",$ONLY," in *",$1,"*) return 0 ;; *) return 1 ;; esac
}

# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------
CURL=(curl -sS --max-time 25)

LAST_CODE=""

# api METHOD PATH [JSON]  — authenticated (admin cookie jar), body -> body.json
api() {
  local m="$1" p="$2" d="${3:-}"
  if [ -n "$d" ]; then
    LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' -X "$m" \
      -H 'Content-Type: application/json' -c "$COOKIE" -b "$COOKIE" --data-binary "$d" "$BASE_URL$p")
  else
    LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' -X "$m" \
      -c "$COOKIE" -b "$COOKIE" "$BASE_URL$p")
  fi
}

# raw METHOD PATH — no cookies, body -> raw.json
raw() {
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/raw.json" -w '%{http_code}' -X "$1" "$BASE_URL$2")
}

# raw_json METHOD PATH JSON — no cookies, JSON body -> raw.json
raw_json() {
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/raw.json" -w '%{http_code}' -X "$1" \
    -H 'Content-Type: application/json' --data-binary "$3" "$BASE_URL$2")
}

# bearer PATH TOKEN — Authorization: Bearer, shell out to a fresh jar, body -> body.json
bearer() {
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' \
    -H "Authorization: Bearer $2" "$BASE_URL$1")
}

# request capturing response headers (authenticated) -> headers.txt
api_headers() {
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -D "$TMPD/headers.txt" -w '%{http_code}' \
    -X "$1" -c "$COOKIE" -b "$COOKIE" "$BASE_URL$2")
}

# request capturing response headers (anonymous) -> headers.txt
raw_headers() {
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -D "$TMPD/headers.txt" -w '%{http_code}' "$BASE_URL$1")
}

header_val() { grep -i "^$1:" "$TMPD/headers.txt" 2>/dev/null | tail -1 | sed 's/^[^:]*: *//' | tr -d '\r'; }

file_headers() { # URL  (anonymous HEAD-ish GET, body discarded)
  LAST_CODE=$("${CURL[@]}" -o /dev/null -D "$TMPD/headers.txt" -w '%{http_code}' "$1")
}

# json_get FILE dotted.path — minimal lookup without jq (arrays: [0], logs[0].x)
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

jf() { json_get "$TMPD/body.json" "$1"; }
rf() { json_get "$TMPD/raw.json" "$1"; }

# ---------------------------------------------------------------------------
# Assertions
# ---------------------------------------------------------------------------
expect_code() { local l="$1" w="$2" g="${3:-$LAST_CODE}"; [ "$g" = "$w" ] && pass "$l [$w]" || fail "$l (want $w, got $g)"; }
expect_code_in() {
  local l="$1" want="$2" g="${3:-$LAST_CODE}" w
  for w in $want; do [ "$g" = "$w" ] && { pass "$l [$g]"; return; }; done
  fail "$l (want one of [$want], got $g)"
}
expect_eq() { local l="$1" g="$2" w="$3"; [ "$g" = "$w" ] && pass "$l" || fail "$l (want '$w', got '$g')"; }
expect_nonempty() { local l="$1" g="$2"; [ -n "$g" ] && [ "$g" != "null" ] && pass "$l" || fail "$l (empty)"; }
# expect_contains LABEL NEEDLE HAYSTACK
expect_contains() {
  local l="$1" needle="$2" hay="$3"
  # No pipeline here on purpose: with `set -o pipefail`, `printf | grep -q`
  # reports failure whenever grep matches and exits before printf finishes
  # (printf takes SIGPIPE) — which made this check flake as the haystack grew.
  if [[ "$hay" == *"$needle"* ]]; then pass "$l"; else fail "$l (missing '$needle')"; fi
}

# poll_field PATH EXPR WANT TIMEOUT [api|raw] -> body.json holds the last response
POLL_CODE=""
poll_field() {
  local path="$1" expr="$2" want="$3" timeout="${4:-60}" mode="${5:-api}"
  local deadline=$(( $(date +%s) + timeout )) got
  while :; do
    if [ "$mode" = "raw" ]; then raw GET "$path"; else api GET "$path"; fi
    got="$(json_get "$TMPD/body.json" "$expr")"
    POLL_CODE="$LAST_CODE"
    [ "$got" = "$want" ] && return 0
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 2
  done
}

# ---------------------------------------------------------------------------
# Bootstrap: admin session
# ---------------------------------------------------------------------------
ADMIN_READY=""   # "" unknown, "1" ready, "0" failed
ensure_admin() {
  [ "$ADMIN_READY" = "1" ] && return 0
  [ "$ADMIN_READY" = "0" ] && return 1

  raw GET /api/setup/check
  local required; required="$(rf setupRequired)"
  if [ "$required" = "True" ]; then
    info "setupRequired=true -> creating initial admin"
    api POST /api/setup "$(printf '{"email":"%s","password":"%s"}' "$ADMIN_EMAIL" "$ADMIN_PASSWORD")"
    expect_code "POST /api/setup" 200
  fi

  local pw
  for pw in "$ADMIN_PASSWORD" "$ALT_PASSWORD"; do
    api POST /api/auth/login "$(printf '{"email":"%s","password":"%s"}' "$ADMIN_EMAIL" "$pw")"
    if [ "$LAST_CODE" = "200" ]; then
      info "admin session established ($ADMIN_EMAIL)"
      ADMIN_READY=1; return 0
    fi
  done
  info "admin login failed (last HTTP $LAST_CODE)"
  ADMIN_READY=0; return 1
}

# guard authed sections
authed() {
  if ensure_admin; then return 0; fi
  skip "admin session unavailable — $1 checks skipped"
  return 1
}

DIRECT_MODE=""   # compose | k8s | none
detect_direct() {
  [ -n "$DIRECT_MODE" ] && return 0
  if [ "$DIRECT" = "0" ]; then DIRECT_MODE=none; return 0; fi
  if [ "$DIRECT" = "1" ]; then
    DIRECT_MODE=compose; return 0
  fi
  # Decide from the gateway we are actually verifying, not from whichever
  # stack happens to be listening: a machine can run compose AND a k8s cluster
  # at once, and probing localhost:8082 would then wrongly pick compose while
  # the sweep targets the cluster (and vice-versa). The NodePort gateway
  # (…:30080) or a non-local host means k8s; a plain localhost/127.0.0.1 root
  # means compose.
  local hostport="${BASE_URL#http://}"; hostport="${hostport#https://}"
  case "$hostport" in
    localhost*|127.0.0.1*)
      # localhost with a NodePort (kind maps 30080 to the host) is k8s.
      case "$hostport" in
        *:30080*|*:3008*) DIRECT_MODE=k8s; return 0 ;;
      esac
      if curl -s --max-time 3 "http://localhost:8082/health" 2>/dev/null > "$TMPD/__direct.json"; then
        if [[ "$(cat "$TMPD/__direct.json" 2>/dev/null)" == *'"status":"ok"'* ]]; then
          DIRECT_MODE=compose; return 0
        fi
      fi
      ;;
    *)
      DIRECT_MODE=k8s; return 0
      ;;
  esac
  if command -v kubectl >/dev/null 2>&1 && kubectl get ns omoikane >/dev/null 2>&1; then
    DIRECT_MODE=k8s; return 0
  fi
  DIRECT_MODE=none; return 0
}

SVC_PORTS="auth:8082 content:8083 media:8084 messages:8085 settings:8086 trash:8087 dashboard:8088 docs:8089 audit:8081 webhooks:8090 webhook-sink:8091"

# kafka_bootstrap — broker address as seen from inside the broker itself
# (compose container / in-cluster pod). Empty when there is no direct mode.
kafka_bootstrap() {
  case "$DIRECT_MODE" in
    compose) echo "localhost:9092" ;;
    k8s)     echo "kafka:29092" ;;
    *)       echo "" ;;
  esac
}

# compose_container PATTERN — name of a RUNNING compose container whose name
# matches the regex (project name varies), empty when docker is unavailable.
compose_container() {
  command -v docker >/dev/null 2>&1 || { echo ""; return 0; }
  docker ps --format '{{.Names}}' 2>/dev/null | grep -E -- "$1" | head -1
}

# sink_fetch OUTFILE — write the demo sink's recorded requests (JSON) to OUTFILE.
# Compose publishes :8091 on the host so a plain curl works; in k8s the sink has
# no host port, so we fetch it in-cluster through the nginx pod (which already
# talks to every service). Returns non-zero when the sink cannot be reached —
# callers report SKIP, never FAIL (the HMAC math is covered by Go unit tests).
sink_fetch() {
  local out="$1"
  case "$DIRECT_MODE" in
    k8s)
      kubectl -n omoikane exec deploy/nginx -- wget -qO- "http://webhook-sink:8091/requests" > "$out" 2>/dev/null \
        && [ -s "$out" ]
      ;;
    *)
      curl -sS --max-time 5 "$SINK_BASE/requests" -o "$out" 2>/dev/null && [ -s "$out" ]
      ;;
  esac
}

# kafka_tail TOPIC GROUP SECONDS OUTFILE — tail the topic in the background for
# up to SECONDS, writing the raw CloudEvent lines to OUTFILE. Prints the watcher
# PID (empty when Kafka is not reachable directly, i.e. gateway-only mode).
# The watcher attaches at the latest offset, so callers must start it BEFORE
# producing the event they want to observe.
kafka_tail() {
  local topic="$1" group="$2" secs="$3" out="$4" bs pid=""
  bs="$(kafka_bootstrap)"; [ -n "$bs" ] || { echo ""; return 0; }
  local -a cmd=(/opt/kafka/bin/kafka-console-consumer.sh
    --bootstrap-server "$bs" --topic "$topic" --group "$group")
  case "$DIRECT_MODE" in
    compose)
      local kc; kc="$(compose_container '(^|[-_])kafka([-_]|$)')"
      [ -n "$kc" ] || { echo ""; return 0; }
      timeout "$secs" docker exec "$kc" "${cmd[@]}" > "$out" 2>/dev/null &
      pid=$! ;;
    k8s)
      timeout "$secs" kubectl -n omoikane exec deploy/kafka -- "${cmd[@]}" > "$out" 2>/dev/null &
      pid=$! ;;
  esac
  echo "$pid"
}

# kafka_tail_stop PID — stop a watcher started by kafka_tail and wait for it.
kafka_tail_stop() {
  [ -n "$1" ] || return 0
  kill "$1" 2>/dev/null
  wait "$1" 2>/dev/null
  return 0
}

# kafka_tail_ready PID GROUP SECS — block until the watcher has joined its
# consumer group and reached the tail of the topic, so an event published after
# this returns is guaranteed to be observed. Polls the group's lag rather than
# sleeping a guessed interval: `--group` means the console consumer starts at
# the group's committed offset, and a FRESH group (the per-run name) starts at
# `auto.offset.reset=latest` — i.e. the tail. Returns 0 once the group is
# visible to the broker with zero lag, 1 on timeout.
kafka_tail_ready() {
  local pid="$1" group="$2" secs="${3:-20}" i
  [ -n "$pid" ] || return 1
  [ "$DIRECT_MODE" = "compose" ] || { sleep 4; return 0; }   # k8s fallback: no shell in the pod to poll
  local kc; kc="$(compose_container '(^|[-_])kafka([-_]|$)')"
  [ -n "$kc" ] || { sleep 4; return 0; }
  for i in $(seq 1 "$secs"); do
    # The group shows up in --list as soon as the consumer joins. That is the
    # moment its fetch position is the topic end; publishing after this cannot
    # be missed by the `--since` window.
    local groups=""
    groups="$(timeout 5 docker exec "$kc" /opt/kafka/bin/kafka-consumer-groups.sh \
         --bootstrap-server localhost:9092 --list 2>/dev/null || true)"
    if [[ $'\n'"$groups"$'\n' == *$'\n'"$group"$'\n'* ]]; then
      return 0
    fi
    kill -0 "$pid" 2>/dev/null || return 1   # watcher died (timeout wrapper etc.)
    sleep 1
  done
  return 1
}

TS="$(date +%s)"
PFX="verify-$TS"
CLEANUP_URLS="$TMPD/cleanup.txt"
: > "$CLEANUP_URLS"
track() { printf '%s\n' "$1" >> "$CLEANUP_URLS"; }

# ===========================================================================
# Sections
# ===========================================================================
section_infra() {
  section infra "Infrastructure & gateway reachability"
  raw GET /api/setup/check
  expect_code "GET /api/setup/check" 200
  expect_nonempty "setup/check returns setupRequired" "$(rf setupRequired)"
  raw GET /
  expect_code_in "GET / (frontend)" "200 301 302 307 308"
}

section_routing() {
  section routing "Gateway route map & fail-loud"

  raw GET /api/does-not-exist
  expect_code "unmapped /api/* -> 404 (fail-loud)" 404

  raw GET /api/metrics
  expect_code "/metrics NOT exposed via gateway -> 404" 404

  # protected routes reject anonymous callers (proves they hit a guarded owner)
  local p
  for p in /api/users /api/trash /api/dashboard /api/webhooks /api/audit-logs /api/api-tokens /api/messages /api/settings/profile; do
    raw GET "$p"
    expect_code "anonymous $p -> 401" 401
  done
  # /api/admin/blog/ has its own location (the /api/blog/ prefix does not match it)
  raw GET /api/admin/blog/posts
  expect_code "anonymous /api/admin/blog/posts -> 401" 401

  # internal aggregation endpoints (X-Internal-Token) are NEVER routed by the
  # gateway — they must fall through to the fail-loud 404, not reach an owner
  local ip
  for ip in /api/internal/stats /api/internal/trash /api/internal/trash/count; do
    raw GET "$ip"
    expect_code_in "$ip -> 404 (internal, never gateway-exposed)" "404"
  done

  # public routes answer anonymously
  for p in /api/pages /api/blog/posts /api/blog/tags /api/blog/categories /api/settings; do
    raw GET "$p"
    expect_code_in "public $p -> 200" "200"
  done

  # media file serving is public: a missing file is 404, never 401
  raw GET /api/media/file/__verify_missing__.png
  expect_code "public media file missing -> 404 (not 401)" 404

  # both swagger UIs (trailing-slash redirect rewritten by nginx)
  raw GET /api/swagger/
  expect_code_in "GET /api/swagger/ (redirect)" "200 301 302"
  raw GET /api/swagger/index.html
  expect_code "GET /api/swagger/index.html" 200
  raw GET /api/audit/swagger/
  expect_code_in "GET /api/audit/swagger/" "200 301 302"
}

section_health() {
  section health "Service health / deployment readiness"
  detect_direct
  local svc port body
  case "$DIRECT_MODE" in
    compose)
      for entry in $SVC_PORTS; do
        svc="${entry%%:*}"; port="${entry##*:}"
        body="$(curl -sS --max-time 5 "http://localhost:$port/health" 2>/dev/null)"
        if [[ "$body" == *'"status":"ok"'* ]]; then
          pass "$svc /health [:$port]"
        else
          fail "$svc /health [:$port]"
        fi
      done
      ;;
    k8s)
      local total ready
      total="$(kubectl get deploy -n omoikane --no-headers 2>/dev/null | wc -l | tr -d ' ')"
      ready="$(kubectl get deploy -n omoikane --no-headers 2>/dev/null | awk '$2 ~ /^[1-9]/ && $2 ~ /\//{c++} END{print c+0}')"
      expect_eq "all k8s deployments ready ($ready/$total)" "$ready" "$total"
      local jobs done_jobs
      jobs="$(kubectl get jobs -n omoikane --no-headers 2>/dev/null | wc -l | tr -d ' ')"
      done_jobs="$(kubectl get jobs -n omoikane --no-headers 2>/dev/null | awk '$2=="1/1"{c++} END{print c+0}')"
      expect_eq "migration Jobs complete ($done_jobs/$jobs)" "$done_jobs" "$jobs"
      ;;
    *)
      skip "direct-port/k8s checks (no local stack detected)"
      ;;
  esac
}

section_auth() {
  section auth "Auth: setup/login, users, register + role gate, API tokens"
  if ! authed "auth"; then return; fi

  api GET /api/settings/profile
  expect_code "GET /api/settings/profile (session)" 200

  api GET /api/users
  expect_code "GET /api/users (admin)" 200
  expect_nonempty "users list has an id" "$(jf '[0].id')"

  # users CRUD (create a throwaway user)
  local uemail="$PFX-user@example.com"
  api POST /api/users "$(printf '{"name":"Verify User","email":"%s","password":"SecurePass123!","role":"user","status":"active"}' "$uemail")"
  expect_code "POST /api/users -> 201" 201
  local uid; uid="$(jf id)"
  if [ -n "$uid" ]; then
    api PUT "/api/users/$uid" '{"name":"Verify User Renamed"}'
    expect_code "PUT /api/users/{id}" 200
    track "DELETE /api/users/$uid"
  else
    fail "POST /api/users returned no id"
  fi

  # register a normal user, log in with a SEPARATE jar, assert role gate 403
  local remail="$PFX-reg@example.com"
  raw_json POST /api/auth/register "$(printf '{"name":"Verify Reg","email":"%s","password":"SecurePass123!"}' "$remail")"
  case "$LAST_CODE" in
    201)
      LAST_CODE=$("${CURL[@]}" -o "$TMPD/body2.json" -w '%{http_code}' -X POST -H 'Content-Type: application/json' \
        -c "$COOKIE2" -b "$COOKIE2" --data-binary "$(printf '{"email":"%s","password":"SecurePass123!"}' "$remail")" "$BASE_URL/api/auth/login")
      expect_code "login as registered non-admin" 200
      LAST_CODE=$("${CURL[@]}" -o "$TMPD/body2.json" -w '%{http_code}' -c "$COOKIE2" -b "$COOKIE2" "$BASE_URL/api/users")
      expect_code "non-admin GET /api/users -> 403" 403
      # logout that throwaway session
      LAST_CODE=$("${CURL[@]}" -o /dev/null -w '%{http_code}' -X POST -c "$COOKIE2" -b "$COOKIE2" "$BASE_URL/api/auth/logout")
      expect_code "POST /api/auth/logout" 200
      if [ -f "$TMPD/body2.json" ]; then
        local rhuid; rhuid="$(python3 -c "import json;print(json.load(open('$TMPD/body2.json')).get('user',{}).get('id',''))" 2>/dev/null)"
        [ -n "$rhuid" ] && track "DELETE /api/users/$rhuid"
      fi
      ;;
    400)
      skip "POST /api/auth/register (reCAPTCHA configured or email exists)"
      ;;
    *)
      fail "POST /api/auth/register (got $LAST_CODE)"
      ;;
  esac

  # API token: create, use as Bearer, revoke
  api POST /api/api-tokens '{"name":"verify-token","role":"admin"}'
  expect_code "POST /api/api-tokens -> 201" 201
  local tid tok; tid="$(jf id)"; tok="$(jf token)"
  expect_nonempty "api-token secret returned once" "$tok"
  if [ -n "$tok" ]; then
    bearer /api/users "$tok"
    expect_code "Bearer token GET /api/users -> 200" 200
  fi
  if [ -n "$tid" ]; then
    api DELETE "/api/api-tokens/$tid"
    expect_code "DELETE /api/api-tokens/{id}" 200
  fi
}

section_content() {
  section content "Content: pages (publish/reorder) + blog (tags/categories/likes)"
  if ! authed "content"; then return; fi

  local pslug="$PFX-page"
  api POST /api/pages "$(printf '{"title":"Verify Page","slug":"%s","content":"<p>v</p>","status":"draft"}' "$pslug")"
  expect_code "POST /api/pages (draft) -> 201" 201
  local pid; pid="$(jf id)"
  expect_nonempty "page id" "$pid"
  if [ -n "$pid" ]; then
    api GET "/api/pages/$pid"
    expect_code "GET /api/pages/{id}" 200
    api PUT "/api/pages/$pid" '{"status":"published"}'
    expect_code "PUT /api/pages/{id} draft->published" 200
    raw GET "/api/pages/slug/$pslug"
    expect_code "public GET /api/pages/slug/{slug}" 200
    api PUT /api/pages/reorder "$(printf '{"pageIds":[%s]}' "$pid")"
    expect_code "PUT /api/pages/reorder" 200
    track "DELETE /api/pages/$pid"
  fi

  # tag + category + post
  local tslug="$PFX-tag" tagid cattid
  api POST /api/blog/tags "$(printf '{"name":"VerifyTag","slug":"%s"}' "$tslug")"
  expect_code "POST /api/blog/tags -> 201" 201
  tagid="$(jf id)"
  api POST /api/blog/categories "$(printf '{"name":"VerifyCat","slug":"%s","description":"v"}' "$PFX-cat")"
  expect_code "POST /api/blog/categories -> 201" 201
  cattid="$(jf id)"

  local postslug="$PFX-post"
  api POST /api/blog/posts "$(printf '{"title":"Verify Post","slug":"%s","content":"<p>v</p>","excerpt":"v","status":"published","tags":["VerifyTag"],"categoryId":%s}' "$postslug" "${cattid:-0}")"
  expect_code "POST /api/blog/posts (published) -> 201" 201
  local bid; bid="$(jf id)"
  expect_nonempty "post id" "$bid"
  if [ -n "$bid" ]; then
    raw GET "/api/blog/posts/slug/$postslug"
    expect_code "public GET /api/blog/posts/slug/{slug}" 200
    raw GET /api/blog/posts
    expect_code "public GET /api/blog/posts" 200
    api GET /api/admin/blog/posts
    expect_code "GET /api/admin/blog/posts (admin)" 200
    api POST "/api/blog/posts/$bid/like"
    expect_code_in "POST /api/blog/posts/{id}/like" "200 201"
    track "DELETE /api/blog/posts/$bid"
  fi
  [ -n "$tagid" ] && { api DELETE "/api/blog/tags/$tagid"; expect_code "DELETE /api/blog/tags/{id}" 200; }
  [ -n "$cattid" ] && { api DELETE "/api/blog/categories/$cattid"; expect_code "DELETE /api/blog/categories/{id}" 200; }
}

section_media() {
  section media "Media: upload -> thumbnail -> public file -> alt"
  if ! authed "media"; then return; fi

  printf 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC' | base64 -d > "$TMPD/verify.png" 2>/dev/null
  LAST_CODE=$("${CURL[@]}" -o "$TMPD/body.json" -w '%{http_code}' -X POST -c "$COOKIE" -b "$COOKIE" \
    -F "file=@$TMPD/verify.png;type=image/png" "$BASE_URL/api/media")
  expect_code "POST /api/media (multipart) -> 201" 201
  local mid murl mthumb
  mid="$(jf media.id)"; murl="$(jf media.url)"; mthumb="$(jf media.thumbUrl)"
  expect_nonempty "media id" "$mid"
  expect_nonempty "media url" "$murl"
  if [ -z "$mid" ]; then return; fi

  api GET /api/media
  expect_code "GET /api/media" 200
  api GET "/api/media/$mid"
  expect_code "GET /api/media/{id}" 200
  api PUT "/api/media/$mid" '{"alt":"verify alt text"}'
  expect_code "PUT /api/media/{id} (alt)" 200

  # public file: immutable cache + ETag/304
  file_headers "$BASE_URL$murl"
  expect_code "GET media file (public) -> 200" 200
  local cc etag; cc="$(header_val cache-control)"; etag="$(header_val etag)"
  expect_contains "media Cache-Control immutable" "immutable" "$cc"
  expect_nonempty "media ETag present" "$etag"
  if [ -n "$etag" ]; then
    LAST_CODE=$("${CURL[@]}" -o /dev/null -D "$TMPD/headers.txt" -w '%{http_code}' -H "If-None-Match: $etag" "$BASE_URL$murl")
    expect_code "media conditional GET -> 304" 304
  fi
  if [ -n "$mthumb" ] && [ "$mthumb" != "null" ]; then
    file_headers "$BASE_URL$mthumb"
    expect_code "GET media thumbnail (public) -> 200" 200
  fi

  api DELETE "/api/media/$mid"
  expect_code "DELETE /api/media/{id}" 200
}

section_messaging() {
  section messaging "Messages/Settings: contact submit + settings cache"
  if ! authed "messaging"; then return; fi

  # contact (public): reCAPTCHA may be configured -> treat 400 as SKIP
  raw_json POST /api/contact "$(printf '{"name":"Verify Contact","email":"%s","subject":"verify","message":"hello from verify"}' "$PFX@example.com")"
  case "$LAST_CODE" in
    201)
      expect_code "POST /api/contact (public) -> 201" 201
      api GET /api/contacts
      expect_code "GET /api/contacts (admin)" 200
      local cid; cid="$(jf 'contacts[0].id')"
      if [ -n "$cid" ]; then track "DELETE /api/contacts/$cid"; fi
      ;;
    400) skip "POST /api/contact (reCAPTCHA configured or validation)";;
    *) fail "POST /api/contact (got $LAST_CODE)";;
  esac

  # settings cache: PUT flushes, GET miss then hit
  api GET /api/settings
  expect_code "GET /api/settings" 200
  api PUT /api/settings '{"siteName":"Omoikane"}'
  expect_code "PUT /api/settings (admin)" 200
  raw_headers /api/settings
  local xc1; xc1="$(header_val x-cache)"
  expect_contains "settings first GET after write -> miss" "miss" "$xc1"
  raw_headers /api/settings
  local xc2; xc2="$(header_val x-cache)"
  expect_contains "settings second GET -> hit" "hit" "$xc2"
}

section_aggregators() {
  section aggregators "Aggregators: dashboard facade + trash fan-out"
  if ! authed "aggregators"; then return; fi

  api GET /api/dashboard
  expect_code "GET /api/dashboard" 200
  api GET /api/dashboard/stats
  expect_code "GET /api/dashboard/stats" 200
  local k
  for k in userCount pageCount blogCount mediaCount recentMessages recentRegistrations; do
    if [ -n "$(jf "$k")" ]; then pass "dashboard/stats.$k present"; else fail "dashboard/stats.$k missing"; fi
  done

  # trash round trip: create page -> delete -> list -> restore -> delete -> hard delete
  local tslug="$PFX-trash"
  api POST /api/pages "$(printf '{"title":"Verify Trash","slug":"%s","content":"<p>t</p>","status":"draft"}' "$tslug")"
  local tpid; tpid="$(jf id)"
  if [ -z "$tpid" ]; then fail "trash fixture page not created"; return; fi
  api DELETE "/api/pages/$tpid"
  expect_code_in "soft-delete page -> trash" "200 204"
  api GET /api/trash
  expect_code "GET /api/trash" 200
  api GET /api/trash/count
  expect_code "GET /api/trash/count" 200
  expect_nonempty "trash count" "$(jf count)"
  api POST "/api/trash/page/$tpid/restore"
  expect_code "POST /api/trash/page/{id}/restore" 200
  api DELETE "/api/pages/$tpid"
  api DELETE "/api/trash/page/$tpid"
  expect_code "DELETE /api/trash/page/{id} (hard)" 200

  api POST /api/trash/bogus/1/restore
  expect_code "unknown trash entity -> 400" 400
}

section_events() {
  section events "Event pipeline: outbox -> Kafka -> audit consumer"
  if ! authed "events"; then return; fi
  detect_direct

  local pslug="$PFX-event"

  # Watch the topic tail BEFORE publishing: the watcher attaches at the latest
  # offset while our event does not exist yet, so the unique slug below should
  # come back from the wire exactly once. Two copies would mean the event was
  # relayed more than once — five services each poll the ONE shared outbox
  # table, which was the Phase 37 finding (no coordination between relays).
  local tailf="$TMPD/events-tail.jsonl" tailpid="" tailgroup="omoikane-verify-tail-$TS"
  tailpid="$(kafka_tail "${OMOKANE_EVENTS_TOPIC:-omoikane.events}" "$tailgroup" 90 "$tailf")"
  # Wait until the watcher has actually joined the group (it is at the tail
  # then); a fixed sleep raced and made this check skip intermittently. The
  # group name is per-run, so it always starts at the topic end — a committed
  # offset from an earlier sweep can never put the watcher mid-history.
  if [ -n "$tailpid" ] && ! kafka_tail_ready "$tailpid" "$tailgroup" 20; then
    skip "wire watcher never joined the consumer group"
    tailpid=""
  fi

  api POST /api/blog/posts "$(printf '{"title":"Verify Event","slug":"%s","content":"<p>e</p>","status":"published"}' "$pslug")"
  expect_code "publish post (event source) -> 201" 201
  local bid; bid="$(jf id)"

  if poll_field "/api/audit-logs?entity=post&search=$pslug" "logs[0].action" "publish" 60; then
    pass "post.published -> audit row action=publish"
  else
    fail "post.published -> audit row action=publish (timeout; last HTTP $POLL_CODE)"
  fi

  api GET /api/audit-logs
  expect_code "GET /api/audit-logs (admin)" 200

  # exactly-once on the wire (relative to our own marker, so unrelated traffic
  # and pre-existing history never affect the result)
  if [ -n "$tailpid" ]; then
    sleep 2                      # a republish lands within relay interval + batch timeout
    kafka_tail_stop "$tailpid"
    local hits; hits="$(grep -c -- "$pslug" "$tailf" 2>/dev/null || true)"
    hits="${hits:-0}"
    case "$hits" in
      1) pass "post.published appears exactly once on the wire" ;;
      0) skip "post.published appears once on the wire (event not observed by the watcher)" ;;
      *) fail "post.published appears $hits times on the wire (outbox relayed more than once)" ;;
    esac
  else
    skip "wire-level exactly-once check (no direct Kafka access in this mode)"
  fi

  [ -n "$bid" ] && track "DELETE /api/blog/posts/$bid"
}

section_webhooks() {
  section webhooks "Webhooks: allow-list, delivery, HMAC, retry->expired, ping"
  if ! authed "webhooks"; then return; fi
  # Populate DIRECT_MODE so sink_fetch resolves the sink for THIS stack (compose
  # publishes :8091 on the host; k8s fetches it in-cluster through nginx). Without
  # this the case fell through to curl localhost:8091 — the *compose* sink on a
  # dev machine — and the HMAC check silently compared against stale data.
  detect_direct

  # allow-list rejects unknown event types
  api POST /api/webhooks '{"eventType":"bogus.type.v1","url":"http://example.invalid/hook"}'
  expect_code "unknown eventType -> 400" 400

  # delivery subscription (to the demo sink)
  api POST /api/webhooks "$(printf '{"eventType":"%s","url":"%s","active":true}' "$POST_EVENT_TYPE" "$DELIVERY_URL")"
  expect_code "POST /api/webhooks (delivery sub) -> 201" 201
  local dsub dsecret; dsub="$(jf id)"; dsecret="$(jf secret)"
  expect_nonempty "delivery subscription id" "$dsub"
  expect_nonempty "one-time HMAC secret" "$dsecret"

  # dead subscription -> retry backoff -> terminal expired
  api POST /api/webhooks "$(printf '{"eventType":"%s","url":"%s","active":true}' "$POST_EVENT_TYPE" "$DEAD_URL")"
  expect_code "POST /api/webhooks (dead sub) -> 201" 201
  local xsub; xsub="$(jf id)"

  # test ping flows through the same pump
  if [ -n "$dsub" ]; then
    api POST "/api/webhooks/$dsub/test"
    expect_code_in "POST /api/webhooks/{id}/test -> 201" "201"
  fi

  # publish a post to trigger both subscriptions
  local pslug="$PFX-hook"
  api POST /api/blog/posts "$(printf '{"title":"Verify Hook","slug":"%s","content":"<p>h</p>","status":"published"}' "$pslug")"
  expect_code "publish post (webhook source) -> 201" 201
  local bid; bid="$(jf id)"

  if [ -n "$dsub" ]; then
    if poll_field "/api/webhooks/deliveries?subscriptionId=$dsub&eventType=$POST_EVENT_TYPE" "deliveries[0].status" "delivered" 90; then
      pass "webhook delivered to sink"
      expect_eq "delivery httpStatus=200" "$(jf 'deliveries[0].httpStatus')" "200"
    else
      fail "webhook delivered to sink (timeout; last HTTP $POLL_CODE)"
    fi
    api GET "/api/webhooks/deliveries?subscriptionId=$dsub"
    expect_code "delivery log filtered by subscriptionId" 200
  fi

  # HMAC signature verification against the demo sink — run BEFORE the
  # dead-endpoint ladder: the sink keeps only its last 50 POSTs, and the ~130s
  # spent waiting for retry exhaustion could otherwise roll this delivery out of
  # the window (the check then reports a misleading SKIP).
  if [ -n "$dsecret" ]; then
    # Fetch retries until our delivery is visible (the sink records shortly
    # after it responds, and the buffer may lag a beat) — then a bounded number
    # of extra attempts before we conclude it never arrived.
    local fetched=""
    local try
    for try in $(seq 1 15); do
      if sink_fetch "$TMPD/sink.json"; then
        fetched=1
        # Stop early once our post appears; otherwise keep polling.
        if grep -q -- "$pslug" "$TMPD/sink.json" 2>/dev/null; then break; fi
      fi
      sleep 2
    done
    if [ -n "$fetched" ]; then
      local verdict
      verdict="$(python3 - "$dsecret" "$TMPD/sink.json" "$pslug" <<'PY'
import hashlib, hmac, json, sys
secret, path, needle = sys.argv[1], sys.argv[2], sys.argv[3]
try:
    with open(path) as fh:
        data = json.load(fh)
except Exception:
    print("sink-error"); sys.exit(0)
# Several subscriptions may carry this post slug (our run's plus leftovers from
# earlier runs, each signed with a DIFFERENT secret), so a slug match alone does
# not mean the record was delivered by US. Accept as soon as ANY matching record
# validates against our secret; only report a mismatch when a signed record
# carrying our slug exists that our secret cannot validate, and no-signature when
# every matching record is unsigned.
saw_signed = False
for rec in reversed(data):
    body = rec.get("body")
    if body is None or needle not in json.dumps(body):
        continue
    sig = rec.get("signature", "")
    if not sig.startswith("sha256="):
        continue
    saw_signed = True
    for raw in (json.dumps(body, separators=(",", ":")),
                json.dumps(body, separators=(",", ":"), ensure_ascii=False)):
        if sig == "sha256=" + hmac.new(secret.encode(), raw.encode(), hashlib.sha256).hexdigest():
            print("ok"); sys.exit(0)
if saw_signed:
    print("mismatch")
else:
    print("not-found")
PY
)"
      case "$verdict" in
        ok) pass "HMAC X-Omoikane-Signature verified (sha256)";;
        no-signature) fail "delivery carried no sha256 signature";;
        not-found) skip "sink has not recorded this delivery yet";;
        sink-error) skip "sink not reachable (HMAC check skipped)";;
        *) fail "HMAC signature mismatch (verdict=$verdict)";;
      esac
    else
      skip "sink not reachable (HMAC check skipped)"
    fi
  fi

  # The retry ladder is 1+2+4+8+16+32s of backoff with a 6-attempt cap. In
  # compose the dead endpoint is refused immediately, so the whole ladder runs
  # in ~90s. In k8s the hostname resolves through kube-proxy but the connection
  # then hangs, so every attempt burns the delivery client's full 10s timeout:
  # worst case ~6x10s + 63s of backoff + poll jitter ≈ 130s. Budget 300s so a
  # slow cluster never reports a correct retry ladder as a failure.
  if [ -n "$xsub" ]; then
    if poll_field "/api/webhooks/deliveries?subscriptionId=$xsub&eventType=$POST_EVENT_TYPE" "deliveries[0].status" "expired" 300; then
      pass "dead endpoint retried then expired"
    else
      fail "dead endpoint retried then expired (timeout; last HTTP $POLL_CODE)"
    fi
  fi

  [ -n "$dsub" ] && track "DELETE /api/webhooks/$dsub"
  [ -n "$xsub" ] && track "DELETE /api/webhooks/$xsub"
  [ -n "$bid" ] && track "DELETE /api/blog/posts/$bid"
}

section_observability() {
  section observability "Observability: /metrics families + route normalisation"
  detect_direct
  if [ "$DIRECT_MODE" = "compose" ]; then
    # Drive one authed auth route so the counter exists, then scrape directly.
    # Both are retried: the sweep may run moments after a service restart, and
    # an empty/partial first scrape must not be reported as a missing label.
    api GET "/api/users" >/dev/null 2>&1 || true
    local metrics=""
    local try
    for try in $(seq 1 20); do
      metrics="$(curl -sS --max-time 5 http://localhost:8082/metrics 2>/dev/null)"
      [[ "$metrics" == *'service="auth"'* ]] && break
      sleep 2
    done
    expect_contains "auth /metrics serves http_requests_total" "http_requests_total" "$metrics"
    expect_contains "auth /metrics labels service=auth" 'service="auth"' "$metrics"
    expect_contains "route normalised (/users not /users/5)" 'route="/users"' "$metrics"
  elif [ "$DIRECT_MODE" = "k8s" ]; then
    local out
    out="$(kubectl exec -n omoikane deploy/nginx -- wget -qO- http://auth-service:8082/metrics 2>/dev/null)"
    if [ -n "$out" ]; then
      expect_contains "auth /metrics serves http_requests_total (in-cluster)" "http_requests_total" "$out"
    else
      skip "in-cluster /metrics scrape unavailable"
    fi
  else
    skip "/metrics direct scrape (no local stack detected)"
  fi

  # Phase 34 promises JSON logs for every service. Two shapes of deviation used
  # to escape the pipeline and must not appear:
  #   - GORM's stock logger owns its own stdlib logger (stderr, ANSI), so DB
  #     errors and ordinary "record not found" control flow came out as raw
  #     text (Phase 37 measured 62 of 66 lines in audit-service);
  #   - `http: superfluous response.WriteHeader call` from a second
  #     WriteHeader reaching net/http.
  # The container log legitimately starts with `go run` build chatter
  # ("go: downloading ..."), which is NOT service output — it is excluded
  # explicitly rather than by tolerance. Each service must have produced at
  # least one JSON record with a service attribute.
  local logf="$TMPD/svc-logs.txt" log_svcs=""
  : > "$logf"
  if [ "$DIRECT_MODE" = "compose" ]; then
    local lc s
    for s in auth content audit webhooks; do
      lc="$(compose_container "${s}-service")"
      if [ -n "$lc" ]; then
        echo "### $s" >> "$logf"
        docker logs --since 30m "$lc" 2>&1 | tail -200 >> "$logf" || true
        log_svcs="$log_svcs $s"
      fi
    done
  elif [ "$DIRECT_MODE" = "k8s" ]; then
    local s
    for s in auth content audit webhooks; do
      echo "### $s" >> "$logf"
      if kubectl -n omoikane logs "deploy/$s-service" --since=30m 2>/dev/null | tail -200 >> "$logf"; then
        log_svcs="$log_svcs $s"
      fi
    done
  fi
  if [ -n "$log_svcs" ] && [ -s "$logf" ]; then
    local badline badtotal
    badline="$(python3 - "$logf" <<'PY'
import json, sys

# `go run` build chatter that is not service output. Kept deliberately narrow
# so a real unstructured service line can never hide behind it.
NOISE = (
    "go: downloading ",
    "go: found ",
    "# omoikane-backend",
    "### ",
)

bad = total = 0
sample = ""
for line in open(sys.argv[1], errors="replace"):
    line = line.strip()
    if not line:
        continue
    if line.startswith(NOISE):
        continue
    total += 1
    ok = False
    try:
        obj = json.loads(line)
        ok = isinstance(obj, dict) and "service" in obj
    except Exception:
        ok = False
    if not ok:
        bad += 1
        if not sample:
            sample = line[:110]
if bad:
    print(f"{bad}/{total}::{sample}")
else:
    print(f"0/{total}")
PY
)"
    # badline is "BAD/TOTAL::sample" (or "0/TOTAL"). Split it explicitly — the
    # old `${var%%::*}` compared "0/31" against "0" and inverted the verdict.
    local badcount="${badline%%/*}"
    local badtotal="${badline#*/}"; badtotal="${badtotal%%::*}"
    if [ "$badcount" = "0" ]; then
      pass "service logs are JSON with a service attribute ($badtotal lines:$log_svcs)"
    else
      fail "service logs are JSON with a service attribute ($badcount of $badtotal lines unstructured; first: ${badline#*::})"
    fi
  else
    skip "structured-log check (no recent service logs reachable)"
  fi

  raw GET /api/metrics
  expect_code "/metrics not gateway-exposed -> 404" 404
}

# ===========================================================================
# Run
# ===========================================================================
printf '%sOmoikane component recheck%s — gateway: %s\n' "$C_BOLD" "$C_RESET" "$BASE_URL"

run() { local s="$1"; enabled "$s" && "section_$s"; }

run infra
run routing
run health
run auth
run content
run media
run messaging
run aggregators
run events
run webhooks
run observability

# ---------------------------------------------------------------------------
# Cleanup (best effort)
# ---------------------------------------------------------------------------
if [ "$KEEP" != "1" ] && [ -s "$CLEANUP_URLS" ] && ensure_admin 2>/dev/null; then
  while read -r path; do
    [ -n "$path" ] && api DELETE "$path" >/dev/null 2>&1 || true
  done < "$CLEANUP_URLS"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
TOTAL=$((PASS + FAIL + SKIP))
printf '\n%sSummary%s  %s%d passed%s  %s%d failed%s  %s%d skipped%s  (%d checks)\n' \
  "$C_BOLD" "$C_RESET" "$C_GREEN" "$PASS" "$C_RESET" "$C_RED" "$FAIL" "$C_RESET" "$C_YELLOW" "$SKIP" "$C_RESET" "$TOTAL"

if [ "$FAIL" -gt 0 ]; then
  printf '\n%sFailures:%s\n' "$C_RED" "$C_RESET"
  sed 's/^/  - /' "$FAILURES" >&2
fi

if [ "$JSON_OUT" = "1" ]; then
  python3 - "$RESULTS" "$PASS" "$FAIL" "$SKIP" "$BASE_URL" <<'PY'
import json, sys
path, p, f, s, base = sys.argv[1], int(sys.argv[2]), int(sys.argv[3]), int(sys.argv[4]), sys.argv[5]
results = []
with open(path) as fh:
    for line in fh:
        line = line.rstrip("\n")
        if not line:
            continue
        status, section, label = line.split("|", 2)
        results.append({"status": status, "section": section, "label": label})
print(json.dumps({
    "gateway": base,
    "passed": p, "failed": f, "skipped": s,
    "ok": f == 0,
    "results": results,
}, indent=2))
PY
fi

[ "$FAIL" -eq 0 ] || exit 1
printf '\n%sAll component checks passed.%s\n' "$C_GREEN" "$C_RESET"
