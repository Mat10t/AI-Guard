# Заголовок
Регрессия project-level fallback routing

## Контекст
Добавление project-level fallback меняет внутреннюю маршрутизацию gateway и API проектов; без тестов высок риск сломать resilience-путь и RBAC.

## Что нужно сделать
- Добавить unit-тесты:
  - `project-key-service`: валидация `fallback_model_id`, org-boundary, RBAC.
  - `provider-catalog-service`: route-resolve с `project_id` и без него.
  - `api-gateway`: fallback-вызов с `fallback_model`.
- Добавить integration-проверки:
  - `GET/PUT /projects/{id}/routing`.
  - Различное fallback-поведение для проекта с override и без override.
- Добавить e2e-проверку:
  - два проекта с разными fallback-настройками и разным результатом fallback.
- Проверить отсутствие регрессий: `make test`, `make integration`, `make e2e`.

## Связанный сервис / модуль
`tests/unit`, `tests/integration`, `tests/e2e`, `services/*`, `frontend`.

## Зависимости
- `04-05-project-fallback-routing-settings.md`
- `07-05-route-resolution-with-project-fallback.md`
- `02-06-fallback-model-override-by-project.md`
- `08-17-project-fallback-settings-ui.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Покрыт сценарий project-level fallback в unit/integration/e2e.
- Проверена RBAC-матрица для управления fallback.
- Все основные тестовые команды остаются зелеными.
