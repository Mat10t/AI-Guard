# Заголовок
Project members и фильтрация доступа к проектам для PM/Dev

## Контекст
Сейчас PM/Dev видят все проекты организации. Нужно сделать project-scoped доступ и явное управление участниками проекта.

## Что нужно сделать
- Добавить `project_members` и использовать его в проверках доступа.
- Изменить `GET /projects`:
  - Admin/Finance: все проекты org,
  - PM/Dev: только назначенные проекты.
- Для project/key/routing endpoints включить единый project-access helper с учетом membership.
- Добавить endpoint'ы:
  - `GET /projects/{id}/members`
  - `POST /projects/{id}/members`
  - `DELETE /projects/{id}/members/{user_id}`
- Записывать аудит `project.member.assigned` / `project.member.unassigned`.

## Связанный сервис / модуль
`project-key-service`

## Зависимости
- `tasks/03-auth-org-service/03-04-invite-with-project-assignment.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- PM/Dev больше не видят чужие проекты.
- Назначение/снятие участника проекта работает для разрешенных ролей.
- Доступ к project/key/routing endpoints корректно ограничен membership.
