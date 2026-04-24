# Заголовок
Мультиключи проекта: create/list/revoke

## Контекст
Сервис `project-key-service` и UI сейчас реализованы под single-key контракт. Для MVP+ нужен полноценный сценарий нескольких активных ключей на проект.

## Что нужно сделать
- Обновить БД `api_keys`: убрать `UNIQUE(project_id)`, добавить индексы для списка ключей проекта.
- Реализовать новые endpoint’ы:
  - `POST /projects/{id}/keys`
  - `GET /projects/{id}/keys`
  - `POST /projects/{id}/keys/{key_id}/revoke`
- Отключить legacy endpoint’ы `.../key` в коде и тестах.
- Сохранить `GET /internal/keys/resolve?api_key=...` и адаптировать к multi-key.
- Публиковать `api_key.created` / `api_key.revoked` с полями `key_id`, `project_id`, `org_id`, `actor`.
- Писать аудит по операциям создания/отзыва ключа.

## Связанный сервис / модуль
`project-key-service`, `db/init.sql`, `db/migrations`.

## Зависимости
- `00-03-requirements-update-multi-key.md`
- `04-01-projects-crud.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Можно создать несколько активных ключей в одном проекте.
- Отзыв одного ключа не ломает остальные активные ключи.
- `GET /projects/{id}/keys` не возвращает полный `api_key`, только metadata/prefix/status.
- `GET /internal/keys/resolve` корректно определяет статус конкретного ключа.
