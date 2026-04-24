# Заголовок
Sidebar fix + AI Guard brand

## Контекст
Левая панель имеет некорректную геометрию и лишний scroll. Также нужно сменить название продукта в shell.

## Что нужно сделать
- Обновить App Shell:
  - заменить бренд на `AI Guard`,
  - убрать пункт `Analytics`,
  - добавить отдельные пункты `API Keys`, `Usage`, `Logs`, `Audit`, `Providers`.
- Исправить layout sidebar:
  - стабильная высота `100dvh`,
  - без отдельного скролла sidebar,
  - скролл только у основного контента.
- Сохранить mobile drawer и desktop sidebar без регрессий по RBAC-видимости.

## Связанный сервис / модуль
`frontend/app/components/app-shell.tsx`, `frontend/app/globals.css`, `frontend/app/lib/rbac.ts`

## Зависимости
- `tasks/00-planning/00-08-ai-guard-navigation-and-keys-page-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Sidebar не прокручивается отдельно от контента.
- В shell отображается `AI Guard`.
- Навигация соответствует новой структуре и RBAC.
