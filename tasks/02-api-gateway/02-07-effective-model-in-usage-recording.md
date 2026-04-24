# Заголовок
Gateway: запись usage по фактической модели (fallback-aware)

## Контекст
Сейчас usage и cost в gateway считаются по исходной `req.model`, из-за чего fallback искажает аналитику и стоимость.

## Что нужно сделать
- Вернуть из resilience-вызова `effective_model`.
- Для primary-success: `effective_model = requested_model`.
- Для fallback-success: `effective_model = fallback model`.
- В `usage.recorded` и `/internal/usage/record` отправлять `requested_model`, `effective_model`, а `model` заполнять как alias `effective_model`.
- Считать `estimated_cost` по `effective_model`.
- В `request.completed` и `fallback.used` добавить `effective_model` без удаления legacy `model`.

## Связанный сервис / модуль
`services/api-gateway/main.go`

## Зависимости
- `tasks/00-planning/00-07-effective-model-accounting-contract.md`
- `tasks/02-api-gateway/02-06-fallback-model-override-by-project.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Usage содержит `requested_model` и `effective_model`.
- `estimated_cost` рассчитывается по фактической модели.
- Kafka события completion/fallback содержат `effective_model`.
