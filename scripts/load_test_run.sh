#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${ROOT_DIR}/docs/reports/artifacts"
ENV_FILE="${ARTIFACTS_DIR}/load-test.env"
REPORT_FILE="${ROOT_DIR}/docs/reports/load-test-latest.md"

mkdir -p "${ARTIFACTS_DIR}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "missing ${ENV_FILE}. Run scripts/load_test_prepare.sh first." >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for load_test_run.sh" >&2
  exit 1
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for load_test_run.sh" >&2
  exit 1
fi

source "${ENV_FILE}"

timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
p95_threshold_ms="${K6_P95_THRESHOLD_MS:-3500}"
gateway_summary_rel="docs/reports/artifacts/k6_gateway_summary.json"
auth_summary_rel="docs/reports/artifacts/k6_auth_summary.json"
gateway_summary_host="${ROOT_DIR}/${gateway_summary_rel}"
auth_summary_host="${ROOT_DIR}/${auth_summary_rel}"

docker run --rm \
  --network upprpo_default \
  --user 0:0 \
  -v "${ROOT_DIR}:/workspace" \
  -w /workspace \
  -e K6_GATEWAY_URL="http://api-gateway:8080" \
  -e K6_API_KEY="${K6_API_KEY}" \
  -e K6_MODEL="${K6_MODEL}" \
  -e K6_P95_THRESHOLD_MS="${p95_threshold_ms}" \
  grafana/k6:0.53.0 run \
  --summary-export "/workspace/${gateway_summary_rel}" \
  tests/load/k6_gateway_chat.js

docker run --rm \
  --network upprpo_default \
  --user 0:0 \
  -v "${ROOT_DIR}:/workspace" \
  -w /workspace \
  -e K6_AUTH_URL="http://auth-org-service:8081" \
  -e K6_PROJECT_URL="http://project-key-service:8082" \
  -e K6_EMAIL="${K6_EMAIL}" \
  -e K6_PASSWORD="${K6_PASSWORD}" \
  -e K6_P95_THRESHOLD_MS="${p95_threshold_ms}" \
  grafana/k6:0.53.0 run \
  --summary-export "/workspace/${auth_summary_rel}" \
  tests/load/k6_auth_project_flow.js

gateway_p95="$(jq -r '.metrics.http_req_duration["p(95)"] // .metrics.http_req_duration.values["p(95)"] // 0' "${gateway_summary_host}")"
gateway_err="$(jq -r '.metrics.http_req_failed.rate // .metrics.http_req_failed.values.rate // 0' "${gateway_summary_host}")"
auth_p95="$(jq -r '.metrics.http_req_duration["p(95)"] // .metrics.http_req_duration.values["p(95)"] // 0' "${auth_summary_host}")"
auth_err="$(jq -r '.metrics.http_req_failed.rate // .metrics.http_req_failed.values.rate // 0' "${auth_summary_host}")"

cat > "${REPORT_FILE}" <<EOF
# Load Test Report (latest)

- Run at (UTC): ${timestamp}
- Environment: local docker-compose
- Threshold policy: p95 < ${p95_threshold_ms} ms, error_rate < 5%

## Results

| Scenario | p95 (ms) | error_rate |
|---|---:|---:|
| gateway chat completions | ${gateway_p95} | ${gateway_err} |
| auth + project flow | ${auth_p95} | ${auth_err} |

## Artifacts

- \`${gateway_summary_rel}\`
- \`${auth_summary_rel}\`
- \`docs/reports/artifacts/load-test.env\`
EOF

echo "Load test report written to ${REPORT_FILE}"
