# Заголовок
Invite link + accept flow + auto-login по роли

## Контекст
Текущий invite-поток создает token, но не дает приглашенному пользователю завершить вход в систему без ручных шагов.

## Что нужно сделать
- На `/auth` после успешного `invite` показывать готовую ссылку вида `/auth?invite_token=...`.
- Реализовать guest-режим `accept invite` при наличии `invite_token` в query.
- Добавить форму установки пароля (без полей login/register в этом режиме).
- Изменить backend `POST /org/members/accept`: возвращать `access_token` + `user_id/org_id/role` и выставлять `refresh_token` cookie.
- После `accept` выполнять auto-login и редирект в кабинет.
- Если `accept` не вернул `access_token`, показывать явную ошибку и действие перехода к обычному входу.

## Связанный сервис / модуль
`auth-org-service`, `frontend`.

## Зависимости
- `03-02-members-and-roles.md`
- `08-04-auth-session-guard-shell.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Приглашенный пользователь может открыть invite-link, задать пароль и сразу попасть в кабинет.
- Ответ `POST /org/members/accept` содержит `access_token` и сохраняет совместимость по `user_id/org_id/role`.
- После перезагрузки страницы сессия продолжается через refresh-cookie path.
