#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${ROOT_DIR}/docs/reports/artifacts"
REPORT_FILE="${ROOT_DIR}/docs/reports/quality-latest.md"
GO_BIN="${GO_BIN:-go}"

mkdir -p "${ARTIFACTS_DIR}"

ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
ts_file="$(date -u +"%Y%m%dT%H%M%SZ")"

vet_log="${ARTIFACTS_DIR}/go-vet-${ts_file}.log"
vuln_log="${ARTIFACTS_DIR}/govulncheck-${ts_file}.log"
test_log="${ARTIFACTS_DIR}/go-test-cover-${ts_file}.log"

status=0
vet_status="PASS"
vuln_status="PASS"
test_status="PASS"
vuln_findings="0"

pushd "${ROOT_DIR}" >/dev/null

if ! ${GO_BIN} vet ./... >"${vet_log}" 2>&1; then
  vet_status="FAIL"
  status=1
fi

vuln_exit=0
if command -v govulncheck >/dev/null 2>&1; then
  govulncheck -format json ./... >"${vuln_log}" 2>&1 || vuln_exit=$?
else
  ${GO_BIN} run golang.org/x/vuln/cmd/govulncheck@latest -format json ./... >"${vuln_log}" 2>&1 || vuln_exit=$?
fi

vuln_findings="$(grep -c '"finding"' "${vuln_log}" || true)"
if [[ "${vuln_findings}" != "0" ]]; then
  vuln_status="WARN (${vuln_findings} findings)"
elif [[ "${vuln_exit}" != "0" ]]; then
  vuln_status="FAIL"
  status=1
fi

if ! ${GO_BIN} test ./... -coverprofile=coverage.out >"${test_log}" 2>&1; then
  test_status="FAIL"
  status=1
fi

coverage_line="$(${GO_BIN} tool cover -func=coverage.out | tail -n 1 || true)"
coverage_total="$(awk '{print $3}' <<<"${coverage_line}")"

cat > "${REPORT_FILE}" <<EOF
# Quality & Security Report (latest)

- Run at (UTC): ${ts}

## Checks

| Check | Status | Artifact |
|---|---|---|
| go vet ./... | ${vet_status} | \`${vet_log##${ROOT_DIR}/}\` |
| govulncheck ./... | ${vuln_status} | \`${vuln_log##${ROOT_DIR}/}\` |
| go test ./... -coverprofile | ${test_status} | \`${test_log##${ROOT_DIR}/}\` |

## Coverage

- Total coverage: ${coverage_total:-n/a}

## Note

- Для учебного MVP блокирующим считается только статус FAIL.
- Govulncheck не использует severity-ярлыки (LOW/MEDIUM/CRITICAL), поэтому найденные записи отражаются как WARN и требуют ручного анализа контекста.
EOF

popd >/dev/null

echo "Quality report written to ${REPORT_FILE}"
exit ${status}
