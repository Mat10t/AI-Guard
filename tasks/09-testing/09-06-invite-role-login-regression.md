# Заголовок
Regression: invite->accept->auto-login и role visibility

## Контекст
После доработки auth/RBAC нужны явные проверки нового пользовательского сценария и ограничений по ролям.

## Что нужно сделать
- Добавить integration-проверки `invite` и `accept` с `access_token` + refresh cookie.
- Добавить e2e-сценарий: Admin приглашает Dev, Dev принимает invite и работает в рамках своей роли.
- Проверить запреты для ролей на mutating endpoints (projects/keys/limits/audit-csv).
- Обновить frontend regression checklist для нового потока.
- Проверить reject по `accept` с невалидным/просроченным token.

## Связанный сервис / модуль
`tests`, `auth-org-service`, `frontend`.

## Зависимости
- `08-08-invite-link-accept-auto-login.md`
- `08-09-role-navigation-visibility-matrix.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Invite/accept/auto-login сценарий стабильно воспроизводится.
- Проверки role-based ограничений проходят на API и UI уровне.
- Новые тесты не ломают текущие `smoke/integration/e2e`.
- `make integration` и `make e2e` проходят в поднятом docker-compose окружении.
