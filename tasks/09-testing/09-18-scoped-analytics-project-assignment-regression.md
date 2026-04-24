# Заголовок
Regression: scoped analytics и project assignment

## Контекст
Новые ограничения доступа и scope-режимы требуют обязательной регрессии на API, e2e и UI.

## Что нужно сделать
- Integration tests:
  - invite PM/Dev без `project_ids` -> `400`,
  - invite+accept создают project_members,
  - `GET /projects` для PM/Dev возвращает только назначенные,
  - analytics/logs/audit/csv с `scope=org|project|key` проверяют RBAC.
- E2E tests:
  - Dev видит только назначенный проект,
  - org-scope скрыт/запрещен для Dev/PM,
  - CSV работает в рамках scope-ограничений.
- Frontend checklist:
  - scope-switcher соответствует роли,
  - project-members add/remove имеют явный результат.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `frontend`

## Зависимости
- `tasks/08-frontend/08-22-analytics-scope-switcher-and-project-member-ui.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `go test ./...`, `make integration`, `make e2e`, `npm --prefix frontend run build` успешны.
- Критичные сценарии scoped-access воспроизводимы без ручных правок.
