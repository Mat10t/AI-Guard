# Заголовок
UI: scope switcher аналитики и управление участниками проекта

## Контекст
Нужен явный выбор scope аналитики и UI управления project-members, согласованный с RBAC.

## Что нужно сделать
- Добавить общий scope-switcher для analytics pages:
  - Organization (если роль разрешает),
  - Project (select доступных проектов),
  - Key (select проекта, затем ключа).
- Передавать `scope/project_id/api_key_id` во все analytics/csv запросы.
- В `Projects` добавить блок участников проекта:
  - list/add/remove,
  - действия только для разрешенных ролей,
  - явные success/error/loading состояния.
- В invite-форме (`/auth`) для PM/Dev добавить обязательный мультиселект проектов.

## Связанный сервис / модуль
`frontend`

## Зависимости
- `tasks/06-audit-analytics-service/06-05-scoped-analytics-org-project-key.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Переключение scope корректно меняет данные аналитики.
- PM/Dev не видят org-scope в UI.
- Назначение сотрудников на проект работает из UI с явной обратной связью.
