# Заголовок
Реализация strict audit scope через project_id/api_key_id

## Контекст
Для точной фильтрации audit по проекту и ключу нужны явные поля в `audit_logs`, а также заполнение этих полей всеми сервисами, публикующими audit-события.

## Что нужно сделать
- Расширить `audit_logs` колонками `project_id UUID NULL`, `api_key_id UUID NULL`.
- Обновить ingest endpoint `/internal/audit/event` для приема и сохранения этих полей.
- Обновить фильтрацию `/audit` и `/reports/csv/audit`:
  - `scope=project` -> `WHERE project_id = ...`
  - `scope=key` -> `WHERE api_key_id = ...`
- Обновить сервисы-публикаторы audit:
  - `project-key-service` (project/key/member actions)
  - `limits-service` (`limit.updated` для project/key)
- Добавить индексы для scoped-аудита.

## Связанный сервис / модуль
`audit-analytics-service`, `project-key-service`, `limits-service`, `db/init.sql`, `db/migrations`

## Зависимости
- `tasks/00-planning/00-06-audit-hard-scope-columns-contract.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новые audit события содержат корректные `project_id/api_key_id`.
- Запросы `/audit` и `/reports/csv/audit` не используют эвристику по details.
- Фильтрация project/key-scope возвращает только релевантные записи.
