#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
AUTH_URL="${AUTH_URL:-http://localhost:8081}"
PROJECT_URL="${PROJECT_URL:-http://localhost:8082}"
LIMITS_URL="${LIMITS_URL:-http://localhost:8083}"
ANALYTICS_URL="${ANALYTICS_URL:-http://localhost:8085}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

request_json() {
  local method="$1"; shift
  local url="$1"; shift
  local token="$1"; shift
  local body="${1:-}"

  local out="$TMP_DIR/resp.json"
  local code
  if [[ -n "$body" ]]; then
    code=$(curl -sS -o "$out" -w '%{http_code}' -X "$method" "$url" \
      -H "Content-Type: application/json" \
      ${token:+-H "Authorization: Bearer $token"} \
      -d "$body")
  else
    code=$(curl -sS -o "$out" -w '%{http_code}' -X "$method" "$url" \
      ${token:+-H "Authorization: Bearer $token"})
  fi
  echo "$code"
}

assert_code() {
  local got="$1"; local want="$2"; local msg="$3"
  if [[ "$got" != "$want" ]]; then
    echo "ERROR: $msg: expected $want, got $got"
    cat "$TMP_DIR/resp.json" || true
    exit 1
  fi
}

echo "== Health checks =="
for u in "$GATEWAY_URL/healthz" "$AUTH_URL/healthz" "$PROJECT_URL/healthz" "$LIMITS_URL/healthz" "$ANALYTICS_URL/healthz"; do
  code=$(curl -sS -o /dev/null -w '%{http_code}' "$u")
  assert_code "$code" "200" "health check failed for $u"
  echo "OK $u"
done

EMAIL="smoke-$(date +%s)@example.local"
PASSWORD="password123"

echo "== Register =="
code=$(request_json POST "$AUTH_URL/auth/register" "" "{\"org_name\":\"Smoke Org\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
assert_code "$code" "201" "register"
TOKEN=$(jq -r '.access_token' "$TMP_DIR/resp.json")

code=$(request_json POST "$PROJECT_URL/projects" "$TOKEN" '{"name":"Smoke Project"}')
assert_code "$code" "201" "create project"
PROJECT_ID=$(jq -r '.id' "$TMP_DIR/resp.json")

code=$(request_json POST "$PROJECT_URL/projects/$PROJECT_ID/keys" "$TOKEN" "")
assert_code "$code" "201" "issue key"
API_KEY=$(jq -r '.api_key' "$TMP_DIR/resp.json")
KEY_ID=$(jq -r '.id' "$TMP_DIR/resp.json")

code=$(request_json GET "$PROJECT_URL/projects/$PROJECT_ID/keys" "$TOKEN" "")
assert_code "$code" "200" "list keys"
LISTED_KEY_ID=$(jq -r '.items[0].id // ""' "$TMP_DIR/resp.json")
if [[ -z "$LISTED_KEY_ID" || "$LISTED_KEY_ID" != "$KEY_ID" ]]; then
  echo "ERROR: expected list keys to include issued key id"
  cat "$TMP_DIR/resp.json" || true
  exit 1
fi

code=$(request_json POST "$GATEWAY_URL/v1/chat/completions" "$API_KEY" '{"model":"mock-fast","messages":[{"role":"user","content":"hello"}]}')
assert_code "$code" "200" "gateway completion with active key"

code=$(request_json PUT "$LIMITS_URL/limits/projects/$PROJECT_ID" "$TOKEN" '{"token_limit":1,"period":"day"}')
assert_code "$code" "200" "set project limit"

code=$(request_json POST "$GATEWAY_URL/v1/chat/completions" "$API_KEY" '{"model":"mock-fast","messages":[{"role":"user","content":"this should exceed the limit"}]}')
assert_code "$code" "429" "limit reject"

code=$(request_json POST "$PROJECT_URL/projects/$PROJECT_ID/keys/$KEY_ID/revoke" "$TOKEN" "")
assert_code "$code" "200" "revoke key"

code=$(request_json POST "$GATEWAY_URL/v1/chat/completions" "$API_KEY" '{"model":"mock-fast","messages":[{"role":"user","content":"must fail"}]}')
assert_code "$code" "401" "revoked key reject"

echo "OK: Smoke flow passed"
