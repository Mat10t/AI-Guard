# Заголовок
Regression: usage по фактической модели при fallback

## Контекст
После добавления `requested_model/effective_model` нужно подтвердить, что fallback корректно отражается в usage, CSV и событиях без регрессий.

## Что нужно сделать
- Unit:
  - выбор `effective_model` в gateway (primary/fallback),
  - cost по `effective_model`,
  - ingest legacy fallback значений в analytics.
- Integration:
  - fallback-запрос пишет `requested_model != effective_model`,
  - `group_by=model` возвращает `effective_model`,
  - usage CSV содержит новые колонки.
- E2E:
  - fallback-сценарий подтверждает фактический учет модели.
- Регрессия:
  - `go test ./...`, `make integration`, `make e2e`, `npm --prefix frontend run build`.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `services/api-gateway/main_test.go`

## Зависимости
- `tasks/02-api-gateway/02-07-effective-model-in-usage-recording.md`
- `tasks/06-audit-analytics-service/06-07-usage-effective-model-storage-and-queries.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Автотесты подтверждают учет фактической модели для fallback.
- Существующие критические e2e сценарии остаются зелеными.
