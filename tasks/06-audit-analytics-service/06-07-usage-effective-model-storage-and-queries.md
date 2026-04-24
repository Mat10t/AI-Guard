# Заголовок
Usage storage/queries: requested_model + effective_model

## Контекст
Для корректной аналитики fallback нужно хранить обе модели в `usage_records` и агрегировать `group_by=model` по фактической модели.

## Что нужно сделать
- Расширить `usage_records` полями `requested_model`, `effective_model`.
- Обновить ingest `/internal/usage/record` под новые поля с legacy fallback.
- В `GET /analytics/usage?group_by=model` агрегировать по `COALESCE(effective_model, model)`.
- В `GET /reports/csv/usage` добавить колонки `requested_model` и `effective_model`, сохранив `model`.
- Добавить migration + backfill старых данных.

## Связанный сервис / модуль
`services/audit-analytics-service/main.go`, `db/init.sql`, `db/migrations`

## Зависимости
- `tasks/00-planning/00-07-effective-model-accounting-contract.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новые поля присутствуют в схеме и заполняются при ingestion.
- `group_by=model` отражает фактическую модель ответа.
- Usage CSV содержит requested/effective модели и остается backward-compatible.
