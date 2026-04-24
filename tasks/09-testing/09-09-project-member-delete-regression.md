# Заголовок
Регрессионные тесты удаления проекта и участника

## Контекст
Добавление удаления проекта в UI и soft-delete участника в backend требует фиксированных integration/e2e проверок, чтобы не допустить регрессий в auth/RBAC и критическом MVP-потоке.

## Что нужно сделать
- Добавить integration-проверки `DELETE /org/members/{id}`: success, self-delete запрет, last-admin запрет, login/refresh после удаления.
- Добавить integration-проверку повторного приглашения email после soft-delete.
- Расширить e2e: Admin приглашает Dev, удаляет Dev, Dev теряет доступ; проект удаляется и исчезает из активного списка.
- Добавить frontend regression checklist по видимости delete actions и typed-confirm UX.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `frontend`.

## Зависимости
- `03-03-members-soft-delete.md`
- `08-12-project-delete-typed-confirm.md`
- `08-13-member-delete-admin-flow.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Новые integration/e2e тесты проходят локально.
- Удаленный пользователь не проходит login/refresh.
- Удаленный проект не возвращается как активный в `/projects`.
- Регрессий в текущих smoke/integration/e2e сценариях нет.
