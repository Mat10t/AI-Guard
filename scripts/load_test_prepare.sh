#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${ROOT_DIR}/docs/reports/artifacts"
ENV_FILE="${ARTIFACTS_DIR}/load-test.env"

AUTH_URL="${AUTH_URL:-http://localhost:8081}"
PROJECT_URL="${PROJECT_URL:-http://localhost:8082}"
MODEL="${K6_MODEL:-mock-fast}"
PASSWORD="${K6_PASSWORD:-LoadTest123!}"

mkdir -p "${ARTIFACTS_DIR}"

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for load_test_prepare.sh" >&2
  exit 1
fi

ts="$(date +%s)"
org_name="Load Test Org ${ts}"
email="load.${ts}@example.com"

reg_payload="$(jq -n \
  --arg org_name "${org_name}" \
  --arg email "${email}" \
  --arg password "${PASSWORD}" \
  '{org_name:$org_name,email:$email,password:$password}')"

reg_body="$(mktemp)"
reg_code="$(curl -sS -o "${reg_body}" -w "%{http_code}" \
  -X POST "${AUTH_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d "${reg_payload}")"

if [[ "${reg_code}" != "201" ]]; then
  echo "register failed (HTTP ${reg_code})" >&2
  cat "${reg_body}" >&2
  rm -f "${reg_body}"
  exit 1
fi

access_token="$(jq -r '.access_token // empty' "${reg_body}")"
rm -f "${reg_body}"
if [[ -z "${access_token}" ]]; then
  echo "register response missing access_token" >&2
  exit 1
fi

project_payload="$(jq -n --arg name "Load Test Project ${ts}" '{name:$name}')"
project_body="$(mktemp)"
project_code="$(curl -sS -o "${project_body}" -w "%{http_code}" \
  -X POST "${PROJECT_URL}/projects" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${access_token}" \
  -d "${project_payload}")"

if [[ "${project_code}" != "201" ]]; then
  echo "project creation failed (HTTP ${project_code})" >&2
  cat "${project_body}" >&2
  rm -f "${project_body}"
  exit 1
fi

project_id="$(jq -r '.id // empty' "${project_body}")"
rm -f "${project_body}"
if [[ -z "${project_id}" ]]; then
  echo "project response missing id" >&2
  exit 1
fi

key_payload='{"name":"Load Test Key"}'
key_body="$(mktemp)"
key_code="$(curl -sS -o "${key_body}" -w "%{http_code}" \
  -X POST "${PROJECT_URL}/projects/${project_id}/keys" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${access_token}" \
  -d "${key_payload}")"

if [[ "${key_code}" != "201" ]]; then
  echo "key creation failed (HTTP ${key_code})" >&2
  cat "${key_body}" >&2
  rm -f "${key_body}"
  exit 1
fi

api_key="$(jq -r '.api_key // empty' "${key_body}")"
rm -f "${key_body}"
if [[ -z "${api_key}" ]]; then
  echo "key response missing api_key" >&2
  exit 1
fi

cat > "${ENV_FILE}" <<EOF
K6_EMAIL=${email}
K6_PASSWORD=${PASSWORD}
K6_MODEL=${MODEL}
K6_API_KEY=${api_key}
K6_PROJECT_ID=${project_id}
EOF

echo "Prepared load-test data: ${ENV_FILE}"
