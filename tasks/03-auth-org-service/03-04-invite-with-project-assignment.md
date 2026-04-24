# Заголовок
Инвайт с назначением проектов и применение назначений при accept

## Контекст
Для PM/Dev доступ должен быть ограничен назначенными проектами. Текущий invite не содержит проектных назначений и accept не создает project-membership.

## Что нужно сделать
- Расширить `POST /org/members/invite` полем `project_ids`.
- Ввести валидацию:
  - для `PM`/`Dev` `project_ids` обязательны и не пустые,
  - все `project_ids` принадлежат `org_id` приглашателя,
  - `PM` может назначать только проекты, в которые сам назначен.
- Расширить `invitations` хранением `project_ids`.
- При `POST /org/members/accept` для `PM`/`Dev` создавать записи в `project_members`.
- Добавить backfill назначений для уже существующих `PM`/`Dev`.

## Связанный сервис / модуль
`auth-org-service`, `db/init.sql`

## Зависимости
- `tasks/00-planning/00-05-scoped-analytics-and-project-assignment-contracts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Invite с валидными `project_ids` сохраняет назначение в приглашении.
- Accept создает пользователя и project_members для PM/Dev.
- Invite PM/Dev без `project_ids` отклоняется с валидационной ошибкой.
- PM не может назначать проекты вне своей проектной области.
