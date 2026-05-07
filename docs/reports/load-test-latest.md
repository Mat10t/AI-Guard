# Load Test Report (latest)

- Run at (UTC): 2026-05-07T14:56:59Z
- Environment: local docker-compose
- Threshold policy: p95 < 3500 ms, error_rate < 5%

## Results

| Scenario | p95 (ms) | error_rate |
|---|---:|---:|
| gateway chat completions | 3025.14190595 | 0 |
| auth + project flow | 78.48776085 | 0 |

## Artifacts

- `docs/reports/artifacts/k6_gateway_summary.json`
- `docs/reports/artifacts/k6_auth_summary.json`
- `docs/reports/artifacts/load-test.env`
