# Заголовок
Резолв маршрута с project-level fallback

## Контекст
Текущий `/internal/route` принимает только `model` и возвращает глобальный fallback из `routing_rules`. Для project-level настройки нужен optional `project_id`.

## Что нужно сделать
- Расширить `GET /internal/route` параметром `project_id` (optional).
- Если `project_id` не передан, оставить текущее поведение.
- Если `project_id` передан и у проекта задан `fallback_model_id`, брать fallback-провайдера и fallback-url из routing-rule выбранной fallback-модели.
- Добавить в ответ поле `fallback_model`.
- Сохранить backward compatibility для старых internal-вызовов.

## Связанный сервис / модуль
`provider-catalog-service`.

## Зависимости
- `04-05-project-fallback-routing-settings.md`
- `07-01-provider-catalog.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `/internal/route?model=...` работает как раньше.
- `/internal/route?model=...&project_id=...` учитывает проектный fallback, если он задан.
- Ответ содержит `fallback_model` для диагностики fallback-ветки.
