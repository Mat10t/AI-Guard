# Заголовок
AI Guard UI contracts: navigation split + API Keys hub

## Контекст
Текущий UI уже близок к AI Studio, но структура разделов не соответствует целевому сценарию: нужен отдельный API Keys hub, split аналитики по страницам и упрощение Projects/Organization экранов.

## Что нужно сделать
- Зафиксировать новый frontend baseline:
  - бренд `AI Guard`,
  - sidebar: `API Keys`, `Projects`, `Usage`, `Logs`, `Audit`, `Providers`, `Organization` (с учетом RBAC),
  - убрать промежуточную перегруженную `Analytics` страницу.
- Зафиксировать поведение маршрутов:
  - `/` для авторизованного пользователя ведет в `/api-keys`,
  - `/analytics` используется как redirect в первый доступный аналитический подраздел.
- Зафиксировать UX поток API ключей:
  - создание через modal (name + project),
  - grouping по key/project,
  - переход в usage key-scope из строки ключа.
- Зафиксировать минимальное backend-расширение для `api_keys.name`.

## Связанный сервис / модуль
`frontend`, `project-key-service`, `db`

## Зависимости
- `tasks/00-planning/00-07-effective-model-accounting-contract.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новая структура разделов и маршрутов описана однозначно.
- Контракт key-name в API зафиксирован до имплементации.
- Нет противоречий с `docs/Требования.md`.
