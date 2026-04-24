# Заголовок
Pricing update: автопересчет project limits по `billing_model`

## Контекст
Обновление цены модели должно автоматически отражаться в лимитах проектов, иначе пользователю приходится вручную пересохранять настройки.

## Что нужно сделать
- В `PUT /catalog/models/{id}/pricing` после обновления цены выполнять автопересчет project limits с `billing_model=<id>`.
- Правила пересчета:
  - `sync_source=tokens`: `token_limit` неизменен, пересчитывается `budget_limit_usd`.
  - `sync_source=budget`: `budget_limit_usd` неизменен, пересчитывается `token_limit=floor(budget/usd_per_token)`.
- Обновлять `usd_per_token` и `updated_at` для затронутых лимитов.
- Вернуть в ответ `affected_project_limits`.

## Связанный сервис / модуль
`provider-catalog-service`, `limits` table.

## Зависимости
- `05-05-project-limit-sync-source-persistence.md`
- `07-07-catalog-pricing-admin-update.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- После pricing update связанные project limits пересчитываются автоматически.
- Ответ pricing endpoint содержит число пересчитанных project limits.
- Пользователь не делает ручной re-save лимита.
