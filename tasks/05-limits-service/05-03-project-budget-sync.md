# Заголовок
Лимиты проекта: бюджет + синхронизация с токенами

## Контекст
В MVP лимит проекта сейчас задается только в токенах. Для удобства управления расходами нужен бюджетный лимит в USD с автоматической синхронизацией с token_limit.

## Что нужно сделать
- Расширить хранение лимита проекта полями `budget_limit_usd`, `billing_model`, `usd_per_token`.
- Добавить `GET /limits/projects/{id}` для чтения текущих настроек лимита проекта.
- Расширить `PUT /limits/projects/{id}`: поддержать `token_limit`, `budget_limit_usd`, `billing_model`, `sync_source`.
- Реализовать синхронизацию по правилу `last edited wins`:
  - `sync_source=tokens` -> пересчет бюджета.
  - `sync_source=budget` -> пересчет token_limit (floor).
- Использовать цены выбранной модели из `provider_models` по формуле 50/50.
- Для нулевой цены модели (`mock-fast`) возвращать `validation_error` при `sync_source=budget`.
- Сохранить совместимость runtime-check: блокировка остается по эффективному `token_limit`.
- Расширить детали `limit.updated` и audit-лог budget-полями.

## Связанный сервис / модуль
`limits-service`, `db/init.sql`.

## Зависимости
- `05-01-limits-config.md`
- `05-02-limits-runtime-check.md`
- `07-01-provider-catalog.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Лимит проекта можно читать и сохранять через `GET/PUT /limits/projects/{id}`.
- Токены и бюджет синхронизируются детерминированно по `sync_source`.
- Старый payload (`token_limit`, `period`) продолжает работать.
- Runtime блокирует запрос при превышении пересчитанного `token_limit`.
