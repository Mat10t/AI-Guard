# Заголовок
Frontend regression: keys/projects modal flow and styling baseline

## Контекст
После перехода к nested-modal flow и action-modals важно зафиксировать регрессионную проверку UX и RBAC.

## Что нужно сделать
- Проверить API Keys flow:
  - `Создать новый проект` в селекте открывает вложенную модалку;
  - после создания проект выбран в key modal;
  - inline create project блока больше нет.
- Проверить Projects flow:
  - таблица показывает summary (limit/budget/period/fallback);
  - действия `Настроить лимиты`, `Настроить fallback`, `Удалить` открывают корректные модалки;
  - после сохранения/удаления список обновляется.
- Проверить RBAC-видимость кнопок и модалок.
- Выполнить техпроверки:
  - `npm --prefix frontend run build`,
  - `make integration`,
  - `make e2e`.

## Связанный сервис / модуль
`frontend`, `tests/e2e`, `tests/integration`

## Зависимости
- `tasks/08-frontend/08-33-api-keys-nested-project-create-modal.md`
- `tasks/08-frontend/08-34-projects-list-summary-and-action-modals.md`
- `tasks/08-frontend/08-35-ai-studio-colors-buttons-layout-tuning.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новый UX-поток стабилен и не ломает текущие API-сценарии.
- Сборка frontend и backend regression проверки проходят.
