# Заголовок
Projects page slim mode: fallback + limits + delete

## Контекст
После выделения API Keys в отдельный раздел страница проектов должна фокусироваться на проектных настройках, а не на ключах.

## Что нужно сделать
- Удалить из `/projects` UI-блоки управления ключами и связанные упоминания.
- Оставить на странице:
  - fallback routing,
  - limits/budget sync,
  - delete project.
- Привести удаление проекта к одному action-кнопке с открытием существующего confirm flow.
- Сохранить текущие RBAC-ограничения и API-вызовы project/limits/routing.

## Связанный сервис / модуль
`frontend/app/projects/page.tsx`

## Зависимости
- `tasks/08-frontend/08-29-api-keys-page-modal-grouping-and-key-scope-jump.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- На Projects странице нет управления API ключами.
- Лимиты, fallback и удаление проекта работают как раньше.
- Delete UX остался понятным и подтверждаемым.
