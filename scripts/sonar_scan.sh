#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_FILE="${ROOT_DIR}/docs/reports/static-analysis-latest.md"
ARTIFACTS_DIR="${ROOT_DIR}/docs/reports/artifacts"
SONAR_URL="${SONAR_URL:-http://localhost:9000}"
SONAR_LOGIN="${SONAR_LOGIN:-admin}"
SONAR_PASSWORD="${SONAR_PASSWORD:-admin}"
PROJECT_KEY="${SONAR_PROJECT_KEY:-ai-guard}"

mkdir -p "${ARTIFACTS_DIR}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for sonar_scan.sh" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for sonar_scan.sh" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required for sonar_scan.sh" >&2
  exit 1
fi

echo "Waiting for SonarQube at ${SONAR_URL}..."
for _ in $(seq 1 180); do
  status="$(curl -fsS "${SONAR_URL}/api/system/status" 2>/dev/null | jq -r '.status // empty' || true)"
  if [[ "${status}" == "UP" ]]; then
    break
  fi
  sleep 2
done

if [[ "${status:-}" != "UP" ]]; then
  ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  cat > "${REPORT_FILE}" <<EOF
# Static Analysis Report (Sonar, latest)

- Run at (UTC): ${ts}
- Sonar URL: ${SONAR_URL}
- Project key: ${PROJECT_KEY}
- Status: FAILED (SonarQube is not ready)
- Last observed status: ${status:-unknown}
EOF
  echo "SonarQube is not ready (status=${status:-unknown})" >&2
  exit 1
fi

docker run --rm \
  --network upprpo_default \
  -v "${ROOT_DIR}:/usr/src" \
  -w /usr/src \
  sonarsource/sonar-scanner-cli:5.0 \
  -Dsonar.host.url="http://sonarqube:9000" \
  -Dsonar.login="${SONAR_LOGIN}" \
  -Dsonar.password="${SONAR_PASSWORD}"

gate_status="$(curl -fsS -u "${SONAR_LOGIN}:${SONAR_PASSWORD}" \
  "${SONAR_URL}/api/qualitygates/project_status?projectKey=${PROJECT_KEY}" \
  | jq -r '.projectStatus.status // "UNKNOWN"' || true)"

ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
cat > "${REPORT_FILE}" <<EOF
# Static Analysis Report (Sonar, latest)

- Run at (UTC): ${ts}
- Sonar URL: ${SONAR_URL}
- Project key: ${PROJECT_KEY}
- Quality gate status: ${gate_status}

## Verification

- Sonar UI: ${SONAR_URL}/dashboard?id=${PROJECT_KEY}
- API status endpoint: ${SONAR_URL}/api/qualitygates/project_status?projectKey=${PROJECT_KEY}
EOF

echo "Static analysis report written to ${REPORT_FILE}"
