# Заголовок
Регрессия: destructive UX, input/output timeseries, provider live-status, pricing update

## Контекст
Изменения затрагивают frontend UX, analytics API и provider-catalog; без регрессии высок риск скрытых ошибок.

## Что нужно сделать
- Добавить integration-проверки:
  - `analytics/timeseries` для `input_tokens` и `output_tokens`;
  - `catalog/providers/status` с полями `checked_at|latency_ms|error`;
  - `PUT /catalog/models/{id}/pricing`.
- Обновить e2e/интеграционные ожидания CSV/logs/usage при необходимости.
- Проверить destructive UX чеклистом (manual frontend regression).

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `frontend`.

## Зависимости
- `08-18-destructive-actions-refresh-feedback.md`
- `06-04-timeseries-input-output-tokens.md`
- `07-06-live-provider-status-checks.md`
- `07-07-catalog-pricing-admin-update.md`
- `08-19-usage-in-out-lines.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новые сценарии покрыты тестами и/или формализованным manual checklist.
- `go test ./...` и `npm --prefix frontend run build` проходят.
