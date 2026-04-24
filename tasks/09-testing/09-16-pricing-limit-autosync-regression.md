# Заголовок
Регрессия: autosync цен и project limits

## Контекст
Автопересчет затрагивает pricing API, limits persistence и runtime-блокировку по токенам; без тестов высок риск скрытых регрессий.

## Что нужно сделать
- Добавить integration-покрытие для autosync:
  - `sync_source=tokens`: меняется бюджет, token_limit сохраняется.
  - `sync_source=budget`: сохраняется бюджет, меняется token_limit.
  - pricing endpoint возвращает `affected_project_limits`.
- Проверить runtime `/internal/limits/check` после autosync без ручного пересохранения.
- Обновить e2e/integration ожидания при расширении payload `sync_source`.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `limits-service`, `provider-catalog-service`.

## Зависимости
- `05-05-project-limit-sync-source-persistence.md`
- `07-08-pricing-auto-sync-project-limits.md`
- `08-20-pricing-update-autosync-feedback.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Интеграционные проверки autosync проходят стабильно.
- `make test`, `make integration`, `make e2e` остаются зелеными.
