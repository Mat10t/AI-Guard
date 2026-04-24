# Заголовок
Frontend regression checklist после AI Studio-inspired редизайна

## Контекст
Нужна проверка, что визуальный рефактор не сломал существующий функционал и RBAC-поведение.

## Что нужно сделать
- Manual regression:
  - auth/register/login/refresh/logout,
  - invite/accept/auto-login,
  - projects CRUD + keys + limits + routing + members,
  - analytics usage/logs/audit/providers + csv export,
  - RBAC visibility for Admin/PM/Dev/Finance.
- UI responsiveness:
  - mobile drawer navigation,
  - desktop sidebar layout,
  - отсутствие критичных overflow/overlap.
- Техпроверка:
  - `npm --prefix frontend run build`,
  - `make integration`,
  - `make e2e`.

## Связанный сервис / модуль
`frontend`, `tests/integration`, `tests/e2e`

## Зависимости
- `tasks/08-frontend/08-25-projects-table-and-detail-panels.md`
- `tasks/08-frontend/08-26-analytics-sections-ai-studio-layout.md`
- `tasks/08-frontend/08-27-auth-and-landing-dark-ai-studio-flow.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- UI редизайн завершен без функциональных регрессий.
- RBAC и protected routing ведут себя корректно.
- Build/integration/e2e проверки проходят.
