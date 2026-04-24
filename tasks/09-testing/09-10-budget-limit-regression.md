# Заголовок
Тесты budget-limit и синхронизации токенов

## Контекст
После добавления budget_limit нужен регрессионный набор, который гарантирует корректную синхронизацию и runtime-блокировки.

## Что нужно сделать
- Добавить unit-тесты конвертации `tokens <-> budget` (формула 50/50, floor, нулевая цена).
- Добавить integration-тесты:
  - `PUT /limits/projects/{id}` с `sync_source=tokens`.
  - `PUT /limits/projects/{id}` с `sync_source=budget`.
  - `GET /limits/projects/{id}`.
  - runtime-блокировка после budget-based сохранения.
- Обновить e2e-критический сценарий проверкой budget-sync пути.
- Добавить frontend regression checklist для новых полей лимитов.

## Связанный сервис / модуль
`services/limits-service`, `tests/integration`, `tests/e2e`, `frontend`.

## Зависимости
- `05-03-project-budget-sync.md`
- `08-14-project-budget-sync-ui.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Unit и integration проверки budget-sync проходят локально.
- E2E подтверждает блокировку по пересчитанному token_limit.
- Существующие сценарии token-only не ломаются.
