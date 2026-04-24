# Заголовок
Scoped analytics/logs/audit/csv: organization, project, key

## Контекст
Аналитика сейчас доступна только на org-уровне. Нужны project/key scope с серверной RBAC-проверкой.

## Что нужно сделать
- Расширить query-параметры:
  - `scope=org|project|key` (default `org`),
  - `project_id` для `scope=project`,
  - `api_key_id` для `scope=key`.
- Применить scope-фильтрацию в:
  - `GET /analytics/usage`
  - `GET /analytics/timeseries`
  - `GET /logs/technical`
  - `GET /audit`
  - `GET /reports/csv/{usage|audit|logs}`
- Реализовать RBAC по scope:
  - Admin/Finance: org/project/key,
  - PM/Dev: только project/key в назначенных проектах,
  - Dev без audit/csv (как и раньше).

## Связанный сервис / модуль
`audit-analytics-service`

## Зависимости
- `tasks/04-project-key-service/04-06-project-members-management-and-access-filtering.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Scope-параметры работают backward-compatible.
- PM/Dev не могут запросить org-scope данные.
- Запрос key-scope валиден только для доступного ключа доступного проекта.
