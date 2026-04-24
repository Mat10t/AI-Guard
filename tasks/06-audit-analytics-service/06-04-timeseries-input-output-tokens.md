# Заголовок
Timeseries метрики input_tokens/output_tokens

## Контекст
График токенов в аналитике сейчас показывает только aggregate `tokens`, что скрывает распределение входных и выходных токенов.

## Что нужно сделать
- Расширить `GET /analytics/timeseries` поддержкой метрик `input_tokens` и `output_tokens`.
- Для этих метрик строить точки на базе `usage_records` по выбранному `bucket`.
- Оставить текущую метрику `tokens` без изменений (backward-compatible).

## Связанный сервис / модуль
`audit-analytics-service`.

## Зависимости
- `06-03-analytics-sections-timeseries-and-csv.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Endpoint возвращает корректные точки для `input_tokens` и `output_tokens`.
- Существующие запросы с `metric=tokens|cost|error_rate|fallback_rate` работают как раньше.
