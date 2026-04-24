# Заголовок
Postman-коллекции для API-задач (public + internal contracts)

## Контекст
В чек-листе защиты есть обязательный пункт: задачи, которые включают вызовы внешних API и/или внутренних микросервисов, должны сопровождаться описанием в формате Postman-коллекций.

## Что нужно сделать
- Добавить в репозиторий Postman collection для ключевых API-сценариев MVP:
  - auth/org;
  - projects/keys/routing;
  - limits;
  - gateway;
  - catalog/providers;
  - analytics/reports.
- Добавить в коллекцию internal-contract сценарии:
  - `/internal/keys/resolve`;
  - `/internal/limits/check`;
  - `/internal/route`;
  - `/internal/audit/event`;
  - `/internal/usage/record`.
- Добавить Postman environment для локального запуска.
- Добавить README по импорту и запуску коллекции вручную/через Newman.
- Добавить команду в `Makefile` для локального запуска postman-проверок (если установлен `newman`).

## Связанный сервис / модуль
`tests`, `docs`, cross-service API contracts

## Зависимости
- `tasks/00-planning/00-02-system-contracts.md`
- `tasks/09-testing/09-02-integration-tests.md`
- `tasks/09-testing/09-03-e2e-critical-flows.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- В репозитории есть коллекция и окружение Postman для основных API-задач.
- Коллекция покрывает как минимум один позитивный и один негативный сценарий для критичных API.
- Для внутреннего API есть отдельные запросы с зафиксированными payload/response-checks.
- Есть воспроизводимая инструкция запуска (Postman UI и `newman` CLI).

## Service Fields (обязательно для новых задач)
- Branch: `feat/09-25-postman-collections-api-contracts`
- PR: `TBD (будет заполнено после открытия PR)`
- Merged at: `TBD (будет заполнено после merge)`
