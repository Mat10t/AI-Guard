# Заголовок
Catalog pricing: admin update и метаданные актуальности

## Контекст
Для контроля расходов нужны не только цены в seed, но и возможность актуализировать их без релиза кода.

## Что нужно сделать
- Добавить в `provider_models` поля `pricing_source`, `pricing_updated_at`.
- Реализовать `PUT /catalog/models/{id}/pricing` (только `Admin`).
- Расширить `GET /catalog/models` возвратом полей `pricing_source`, `pricing_updated_at`.
- Добавить простой admin-блок на странице Providers для обновления цен модели.
- Добавить `docs/pricing.md` с источником и датой актуализации.

## Связанный сервис / модуль
`provider-catalog-service`, `db/init.sql`, `frontend/analytics/providers`, `docs/pricing.md`.

## Зависимости
- `00-04-provider-status-pricing-and-analytics-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Admin может обновить цены модели через API/UI.
- API моделей показывает источник и дату обновления цен.
- Документация по pricing доступна в проекте.
