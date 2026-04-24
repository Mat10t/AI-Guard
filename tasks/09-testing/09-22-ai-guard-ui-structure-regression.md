# Заголовок
AI Guard UI structure regression: sidebar + api-keys + page split

## Контекст
Нужна проверка, что крупная перестройка frontend-навигации и переразделение экранов не сломали существующие сценарии MVP.

## Что нужно сделать
- Проверить backend-интеграцию key-name:
  - create key с `name`,
  - create key без `name`,
  - list keys возвращает `name`,
  - revoke/resolve без регрессий.
- Проверить frontend-регрессию:
  - sidebar геометрия/бренд/навигация,
  - `/api-keys` modal create + grouping + key-scope jump,
  - `/projects` только fallback/limits/delete,
  - `/auth` members + invite modal + grouping,
  - split analytics pages и redirect логика `/analytics` и `/`.
- Техпроверка:
  - `npm --prefix frontend run build`,
  - `make integration`,
  - `make e2e`.

## Связанный сервис / модуль
`frontend`, `project-key-service`, `tests/integration`, `tests/e2e`

## Зависимости
- `tasks/04-project-key-service/04-07-api-key-name-support.md`
- `tasks/08-frontend/08-28-sidebar-fix-and-ai-guard-brand.md`
- `tasks/08-frontend/08-29-api-keys-page-modal-grouping-and-key-scope-jump.md`
- `tasks/08-frontend/08-30-projects-page-slim-fallback-limits-delete.md`
- `tasks/08-frontend/08-31-organization-members-grouped-view-and-invite-modal.md`
- `tasks/08-frontend/08-32-analytics-nav-split-and-console-removal.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новый UI-поток полностью рабочий и соответствует RBAC.
- Key-name контракт стабилен.
- Build/integration/e2e проходят.
