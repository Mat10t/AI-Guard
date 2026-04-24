# Заголовок
Audit hard-scope: явные project_id/api_key_id в audit_logs

## Контекст
Scoped analytics уже поддерживает `scope=org|project|key`, но для audit применялась эвристика по `object_type/object_id/details`. Требуется строгая фильтрация по явным колонкам.

## Что нужно сделать
- Зафиксировать контракт audit payload с опциональными полями `project_id` и `api_key_id`.
- Зафиксировать, что фильтрация `scope=project|key` в `/audit` и `/reports/csv/audit` выполняется только по колонкам `audit_logs.project_id/api_key_id`.
- Зафиксировать backward-compatible поведение для `scope=org`.

## Связанный сервис / модуль
`docs/mvp-system-contracts.md`, `audit-analytics-service`

## Зависимости
- `tasks/00-planning/00-05-scoped-analytics-and-project-assignment-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Контракт audit payload и фильтрации scope описан однозначно.
- Нет разночтений между backend и frontend по семантике scoped audit.
