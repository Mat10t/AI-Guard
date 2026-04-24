# Заголовок
Regression: active key copy + hide revoked

## Контекст
После расширения list endpoint и UI-фильтрации revoked нужны регрессионные проверки, чтобы не сломать revoke/resolve и scoped analytics.

## Что нужно сделать
- Integration:
  - проверить, что `GET /projects/{id}/keys` возвращает `api_key` у active;
  - после revoke проверить `api_key: null` у revoked;
  - подтвердить, что revoked ключ по gateway дает `401 revoked_api_key`.
- Frontend regression checklist:
  - copy работает после reload для active ключа;
  - revoked не видно в `API Keys`;
  - revoked не видно в key-селекторах `usage/logs/audit`;
  - если URL содержит revoked `api_key_id`, UI сбрасывает выбор и показывает notice.
- Техпроверка:
  - `npm --prefix frontend run build`
  - `make integration`
  - `make e2e`

## Связанный сервис / модуль
`tests/integration`, `frontend`

## Зависимости
- `tasks/08-frontend/08-36-copy-anytime-and-hide-revoked-everywhere.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Поведение copy/revoked соответствует контракту.
- Основные integration/e2e проверки остаются зелеными.
