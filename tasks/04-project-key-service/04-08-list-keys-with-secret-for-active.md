# Заголовок
List keys with secret for active keys

## Контекст
Backend уже хранит `key_value`, но list endpoint не возвращает его, из-за чего копирование полного ключа после reload невозможно.

## Что нужно сделать
- Расширить `GET /projects/{id}/keys`:
  - добавить поле `api_key` в item списка;
  - заполнять `api_key` только для `status=active` и непустого `key_value`;
  - для revoked и прочих случаев возвращать `api_key: null`.
- Revoke-логику не менять (`key_value = NULL` остается).
- Не добавлять новые endpoint и не менять существующие URL.

## Связанный сервис / модуль
`services/project-key-service/main.go`

## Зависимости
- `tasks/00-planning/00-10-active-key-copy-and-revoked-visibility-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- В list-ответе есть `api_key`.
- После revoke `api_key` для этого ключа возвращается как `null`.
