# Заголовок
Gateway: fallback-модель с учетом project-level routing

## Контекст
Gateway сейчас использует fallback только по provider/url и отправляет в fallback тот же `model`, что был в исходном запросе.

## Что нужно сделать
- Передавать `project_id` в `provider-catalog` при вызове `/internal/route`.
- Обрабатывать поле `fallback_model` в route-ответе.
- Для fallback-вызова отправлять payload с `model=fallback_model` (если поле задано).
- Сохранить текущее поведение при отсутствии override.
- Добавить `fallback_model` в технические логи и событие `fallback.used`.

## Связанный сервис / модуль
`api-gateway`, `technical_logs`.

## Зависимости
- `07-05-route-resolution-with-project-fallback.md`
- `02-02-gateway-resilience.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Gateway запрашивает маршрут с `project_id`.
- При fallback на проектной override-модели в fallback-вызов уходит `fallback_model`.
- В логах и `fallback.used` фиксируется `fallback_model`.
