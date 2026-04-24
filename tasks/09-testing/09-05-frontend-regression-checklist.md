# Заголовок
Frontend regression checklist для auth-first и protected console

## Контекст
После UX-рефактора нужен воспроизводимый набор проверок, который подтверждает, что поведение UI согласовано с backend.

## Что нужно сделать
- Подготовить deterministic checklist для сценариев неавторизованного и авторизованного пользователя.
- Проверить redirect/guard поведение для `/projects` и `/analytics`.
- Проверить role-aware отображение действий.
- Проверить invite flow в UI: `/auth?invite_token=...` -> accept -> auto-login.
- Проверить refresh/logout flow и восстановление сессии после reload.
- Проверить критичные сценарии pages: projects/keys/limits, logs/audit/usage/csv.

## Связанный сервис / модуль
`frontend`, `tests`.

## Зависимости
- `08-04-auth-session-guard-shell.md`
- `08-05-landing-auth-first.md`
- `08-06-role-aware-projects-analytics.md`
- `08-07-ui-polish-mobile-first.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Есть финальный чеклист, который команда может прогнать перед демонстрацией.
- Все пункты checklist воспроизводимы локально.
- Изменения frontend не ломают `make smoke`, `make integration`, `make e2e`.
