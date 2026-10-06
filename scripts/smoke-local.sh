#!/usr/bin/env bash

set -euo pipefail

workspace_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
api_base=${PAYMINTO_SMOKE_API_BASE:-http://localhost:8090/api/v1}
frontend_origin=${PAYMINTO_SMOKE_FRONTEND_ORIGIN:-http://localhost:3003}
checkout_origin=${PAYMINTO_SMOKE_CHECKOUT_ORIGIN:-http://localhost:3002}
credentials_file=${PAYMINTO_SMOKE_CREDENTIALS:-${workspace_dir}/.dev-credentials.local.json}

pass() { printf 'PASS  %s\n' "$1"; }
fail() { printf 'FAIL  %s\n' "$1" >&2; exit 1; }

expect_status() {
  local label=$1
  local expected=$2
  local url=$3
  local actual
  actual=$(curl -sS -o /dev/null -w '%{http_code}' "$url")
  [[ "$actual" == "$expected" ]] || fail "$label returned HTTP $actual (expected $expected)"
  pass "$label"
}

expect_status "backend health" "200" "${api_base%/api/v1}/healthz"
expect_status "dashboard sign-in page" "200" "${frontend_origin}/signin"
expect_status "checkout public methods" "200" "${checkout_origin}/api/methods"

cors_headers=$(curl -sS -D - -o /dev/null -X OPTIONS "${api_base}/payments" \
  -H "Origin: ${frontend_origin}" \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: authorization,content-type')
if ! printf '%s\n' "$cors_headers" | grep -Fqi "Access-Control-Allow-Origin: ${frontend_origin}"; then
  fail "backend CORS does not allow ${frontend_origin}"
fi
pass "dashboard-to-API CORS preflight"

command -v jq >/dev/null 2>&1 || fail "jq is required for authenticated smoke checks"
[[ -r "$credentials_file" ]] || fail "credential file is not readable: $credentials_file"

for role in admin merchant; do
  email=$(jq -r ".${role}.email // empty" "$credentials_file")
  password=$(jq -r ".${role}.password // empty" "$credentials_file")
  [[ -n "$email" && -n "$password" ]] || fail "$role credentials are incomplete"

  auth_payload=$(jq -nc --arg email "$email" --arg password "$password" \
    '{email:$email,password:$password}')
  auth_response=$(curl -sS -H 'Content-Type: application/json' \
    --data "$auth_payload" "${api_base}/auth/signin")
  access_token=$(printf '%s' "$auth_response" | jq -r '.tokens.accessToken // empty')
  [[ -n "$access_token" ]] || fail "$role sign-in did not return an access token"
  pass "$role sign-in"

  payments_status=$(curl -sS -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer ${access_token}" \
    "${api_base}/payments?limit=1&offset=0")
  [[ "$payments_status" == "200" ]] || fail "$role payments API returned HTTP $payments_status"
  pass "$role payments API"

  core_endpoints=(
    "/analytics/summary"
    "/members/me"
    "/recipients"
    "/wallets"
    "/wallets/hot"
    "/wallets/cold"
    "/withdrawal/merchant"
    "/webhooks"
    "/referrals/stats"
    "/onramper/payments?limit=1"
    "/api-keys"
  )
  for endpoint in "${core_endpoints[@]}"; do
    endpoint_status=$(curl -sS -o /dev/null -w '%{http_code}' \
      -H "Authorization: Bearer ${access_token}" "${api_base}${endpoint}")
    [[ "$endpoint_status" == "200" ]] || fail "$role ${endpoint} returned HTTP $endpoint_status"
  done
  pass "$role core dashboard APIs"

  if [[ "$role" == "admin" ]]; then
    admin_endpoints=(
      "/admin/members"
      "/admin/roles"
      "/admin/permissions"
      "/admin/system/health"
      "/admin/system/workers"
      "/admin/external-platforms"
      "/admin/configurations"
      "/admin/missed-deposits?limit=1"
    )
    for endpoint in "${admin_endpoints[@]}"; do
      endpoint_status=$(curl -sS -o /dev/null -w '%{http_code}' \
        -H "Authorization: Bearer ${access_token}" "${api_base}${endpoint}")
      [[ "$endpoint_status" == "200" ]] || fail "admin ${endpoint} returned HTTP $endpoint_status"
    done
    pass "admin control-plane APIs"
  fi
done

printf '\nLocal Payminto smoke checks passed.\n'
