# Заголовок
Регрессия 5.4: каталог, лимиты и динамический pricing

## Контекст
После перехода на модели 5.4 нужно зафиксировать тестами, что каталог, бюджетные лимиты и usage-cost работают согласованно.

## Что нужно сделать
- Обновить integration/e2e ожидания с `gpt-4o-mini` на `gpt-5.4-mini`.
- Добавить unit-тесты gateway на динамический `estimated_cost` из `provider_models`.
- Проверить в integration:
  - `/catalog/models` содержит `gpt-5.4-mini`, `gpt-5.4`, `mock-fast`;
  - `PUT/GET /limits/projects/{id}` с `billing_model=gpt-5.4-mini` работает корректно;
  - usage содержит ненулевой `estimated_cost` для 5.4-модели.
- Убедиться, что `make test` остается зеленым.

## Связанный сервис / модуль
`api-gateway`, `provider-catalog-service`, `limits-service`, `tests`.

## Зависимости
- `07-03-openai-54-model-catalog.md`
- `05-04-default-billing-model-openai-54.md`
- `02-05-dynamic-model-pricing-from-provider-models.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Тесты покрывают новый каталог моделей и динамический pricing.
- Нет регрессии в существующих critical-flow сценариях.
- Аналитика расходов отражает выбранную 5.4-модель.
