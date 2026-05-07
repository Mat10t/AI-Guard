# Quality & Security Report (latest)

- Run at (UTC): 2026-05-07T15:38:03Z

## Checks

| Check | Status | Artifact |
|---|---|---|
| go vet ./... | PASS | `docs/reports/artifacts/go-vet-20260507T153803Z.log` |
| govulncheck ./... | WARN (74 findings) | `docs/reports/artifacts/govulncheck-20260507T153803Z.log` |
| go test ./... -coverprofile | PASS | `docs/reports/artifacts/go-test-cover-20260507T153803Z.log` |

## Coverage

- Total coverage: 4.7%

## Note

- Для учебного MVP блокирующим считается только статус FAIL.
- Govulncheck не использует severity-ярлыки (LOW/MEDIUM/CRITICAL), поэтому найденные записи отражаются как WARN и требуют ручного анализа контекста.
