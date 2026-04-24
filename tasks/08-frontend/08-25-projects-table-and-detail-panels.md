# Заголовок
Projects UI: table + detail panels (без смены логики)

## Контекст
Страница проектов перегружена стеком карточек. Нужна AI Studio-подобная композиция: список/таблица проектов и панель деталей выбранного проекта.

## Что нужно сделать
- Перекомпоновать `/projects`:
  - desktop: список проектов в table/list панели + detail panel выбранного проекта,
  - mobile: тот же функционал в одном потоке с компактными секциями.
- Сохранить все текущие действия:
  - create/delete project,
  - issue/list/revoke/copy keys,
  - limits + budget sync,
  - fallback routing settings,
  - project members assign/unassign.
- Добавить явные состояния loading/empty/error в новом layout.

## Связанный сервис / модуль
`frontend/app/projects/page.tsx`, `frontend/app/globals.css`

## Зависимости
- `tasks/08-frontend/08-23-ai-studio-inspired-design-tokens-and-primitives.md`
- `tasks/08-frontend/08-24-shell-sidebar-drawer-navigation.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Функционально `/projects` эквивалентен текущей реализации.
- На desktop отображается list/table + detail pattern.
- На mobile нет горизонтальных переполнений и критичных поломок layout.
