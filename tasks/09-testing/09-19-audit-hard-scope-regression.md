# Заголовок
Regression тесты strict audit scope

## Контекст
После перехода на явные поля audit scope нужно подтвердить, что фильтрация работает строго и не ломает существующие сценарии.

## Что нужно сделать
- Integration:
  - проверить `scope=project` и `scope=key` для `/audit` и `/reports/csv/audit`.
  - проверить, что org-scope поведение не изменилось.
- E2E:
  - создать события в разных проектах/ключах и проверить раздельность выдачи audit по scope.
- Регрессия:
  - `go test ./...`, `make integration`, `make e2e`, `npm --prefix frontend run build`.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`

## Зависимости
- `tasks/06-audit-analytics-service/06-06-audit-hard-scope-columns-implementation.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Scoped audit тесты проходят стабильно.
- Нет деградации существующих тестов и API-контрактов.
