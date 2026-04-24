# MVP System Contracts

## Unified Error JSON
```json
{
  "code": "limit_exceeded",
  "message": "project_limit_exceeded"
}
```

## RBAC Roles
- `Admin`: полный доступ в рамках организации.
- `PM`: проекты/ключи/лимиты/аналитика без управления org-owner аспектами.
- `Dev`: доступ к интеграции и технической диагностике, без изменения лимитов и ролей.
- `Finance`: read-only аналитика/отчеты, без изменения настроек.

## Limit Model
- Единица: токены.
- Уровни: `project`, `key`.
- Периоды: `day`, `week`, `month`.
- Поведение: при превышении новый запрос блокируется.
- Для `project` в MVP также поддерживается бюджетный лимит `budget_limit_usd` с пересчетом в эффективный `token_limit`.

## Kafka Topics
- `project.created`
- `api_key.created`
- `api_key.revoked`
- `limit.updated`
- `request.accepted`
- `request.rejected`
- `request.completed`
- `fallback.used`
- `audit.event.created`
- `usage.recorded`

## Core Internal REST Contracts
- `GET /internal/keys/resolve?api_key=...` -> `{ key_id, project_id, org_id, status }`
- `POST /internal/limits/check` -> `{ allowed, reason }`
- `GET /internal/route?model=...&project_id=...` -> `{ primary_provider, fallback_provider, fallback_model, timeout_ms, retry_count, primary_url, fallback_url }`
- `POST /internal/gemini/completions` (OpenAI-compatible adapter for Gemini provider)
- `POST /internal/audit/event`
  - payload (MVP+): `{ org_id, action, object_type, object_id, actor_user_id?, project_id?, api_key_id?, details? }`
- `POST /internal/usage/record`
  - payload (MVP+): `{ org_id, project_id, api_key_id, requested_model?, effective_model?, model?, input_tokens, output_tokens, total_tokens, estimated_cost }`
  - совместимость: `model` сохраняется как alias `effective_model`.

## Public Project-Key REST Contracts (MVP+)
- `POST /projects/{id}/keys` -> `{ id, api_key, status, prefix }`
  - Полный `api_key` возвращается только в ответе create.
- `GET /projects/{id}/keys` -> `{ items: [{ id, status, prefix, created_at, revoked_at }] }`
- `POST /projects/{id}/keys/{key_id}/revoke` -> `{ status }`
- `GET /projects/{id}/routing` -> `{ project_id, fallback_model_id }`
- `PUT /projects/{id}/routing` -> `{ project_id, fallback_model_id }`
- Legacy `/projects/{id}/key` и `/projects/{id}/key/revoke` считаются deprecated и не используются UI.

## Public Limits REST Contracts (MVP+)
- `GET /limits/projects/{id}` -> `{ scope_type, scope_id, token_limit, budget_limit_usd, billing_model, usd_per_token, period, sync_source }`
- `PUT /limits/projects/{id}`:
  - backward-compatible payload: `{ token_limit, period }`
  - extended payload: `{ token_limit?, budget_limit_usd?, billing_model, period, sync_source }`

## Public Analytics REST Contracts (MVP+)
- `GET /analytics/usage?group_by=project|model|day&scope=org|project|key&project_id=&api_key_id=`
  - семантика: `group_by=model` агрегирует по фактической модели ответа (`effective_model`, fallback на legacy `model`).
- `GET /analytics/timeseries?metric=tokens|input_tokens|output_tokens|cost|error_rate|fallback_rate&bucket=hour|day|week|month|year|all&from=&to=&scope=org|project|key&project_id=&api_key_id=`
- `GET /audit?scope=org|project|key&project_id=&api_key_id=`
- `GET /logs/technical?scope=org|project|key&project_id=&api_key_id=`
- `GET /reports/csv/usage?scope=org|project|key&project_id=&api_key_id=`
  - CSV включает `model` (legacy), `requested_model`, `effective_model`.
- `GET /reports/csv/audit?scope=org|project|key&project_id=&api_key_id=`
- `GET /reports/csv/logs?scope=org|project|key&project_id=&api_key_id=`
- Legacy `GET /reports/csv` -> usage CSV alias (deprecated).

## Public Catalog REST Contracts (MVP+)
- `GET /catalog/models` -> `{ items: [{ id, provider, status, input_cost, output_cost, pricing_source, pricing_updated_at }] }`
- `PUT /catalog/models/{id}/pricing` (Admin-only) -> `{ id, provider, status, input_cost, output_cost, pricing_source, pricing_updated_at, affected_project_limits }`
- `GET /catalog/providers/status?refresh=1` -> `{ items: [{ provider, status, checked_at, latency_ms, error }] }`
