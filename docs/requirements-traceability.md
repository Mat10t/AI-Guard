# Requirements Traceability (MVP)

| BR/US | Реализация | Тестирование |
|---|---|---|
| BR.1, US.1.1, US.1.2 | `auth-org-service` `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | `internal/platform/auth/auth_test.go`, `tests/integration/health_smoke_test.go` |
| BR.1, BR.2, US.2.1-2.4 | `project-key-service` `/projects` CRUD | `tests/e2e/critical_flow_test.go` |
| BR.2, US.3.1-3.4 | `project-key-service` `/projects/{id}/keys`, `/projects/{id}/keys/{key_id}/revoke`, `api-gateway` `/v1/chat/completions` | `services/api-gateway/main_test.go`, `tests/integration/health_smoke_test.go`, `tests/e2e/critical_flow_test.go` |
| BR.3, US.5.1-5.3 | `limits-service` `/limits/projects/{id}`, `/limits/keys/{id}`, `/internal/limits/check` | `internal/platform/limits/limits_test.go`, `services/limits-service/main_test.go` |
| BR.4, US.6.1-6.3 | `auth-org-service` members/invite/role update + JWT RBAC | `internal/platform/auth/auth_test.go` |
| BR.5, US.7.1-7.2, US.8.x, US.9.x | `audit-analytics-service` `/audit`, `/logs/technical`, `/analytics/usage`, `/analytics/timeseries`, `/reports/csv/usage`, `/reports/csv/audit`, `/reports/csv/logs` | `tests/integration/health_smoke_test.go`, `tests/e2e/critical_flow_test.go` |
| BR.6, US.10.x | `api-gateway` timeout/retry/fallback + `provider-catalog-service` route/status (`openai`, `gemini`, `mock`) + `PUT /catalog/models/{id}/pricing` | `services/api-gateway/main_test.go`, `tests/integration/health_smoke_test.go`, `tests/e2e/critical_flow_test.go` |

## Out of scope (MVP)
- Billing/tariffs/payments
- SSO/2FA/password recovery
- Multi-step fallback chain
