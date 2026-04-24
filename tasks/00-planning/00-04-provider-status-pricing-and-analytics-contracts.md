# Заголовок
Контракты для live-статусов провайдеров, pricing update и input/output timeseries

## Контекст
Для улучшения UX и прозрачности аналитики нужны уточненные контракты: live-статусы провайдеров, ручное обновление цен моделей и метрики input/output токенов.

## Что нужно сделать
- Зафиксировать расширения API:
  - `GET /analytics/timeseries?metric=input_tokens|output_tokens|...`
  - `GET /catalog/providers/status` с полями `checked_at`, `latency_ms`, `error`.
  - `PUT /catalog/models/{id}/pricing` (роль `Admin`).
- Зафиксировать backward compatibility для существующих метрик и endpoint’ов.
- Зафиксировать политику live-check: TTL 30 секунд, форс-обновление по кнопке.
- Зафиксировать политику актуализации цен: вручную через catalog-admin endpoint.

## Связанный сервис / модуль
`docs/mvp-system-contracts.md`, `provider-catalog-service`, `audit-analytics-service`, `frontend/analytics`.

## Зависимости
- `00-02-system-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Расширенные контракты описаны и согласованы с текущими сервисами.
- Нет противоречий с существующим внешним `/v1/chat/completions`.
