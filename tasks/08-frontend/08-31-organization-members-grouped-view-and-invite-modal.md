# Заголовок
Organization page: grouped members view + invite modal

## Контекст
Раздел организации должен быть ближе к AI Studio-подаче: список участников и действия в шапке, а не длинный линейный поток форм.

## Что нужно сделать
- Переделать `/auth` в режим Organization workspace для авторизованного пользователя:
  - таблица/список участников,
  - переключение группировки `By Member` / `By Project`.
- Перенести invite flow в modal (используя существующие API и валидации).
- Сохранить существующие режимы:
  - login/register/invite-accept для гостя,
  - session controls/developer mode для авторизованного.
- Скрывать/не рендерить запрещенные действия по текущему RBAC.

## Связанный сервис / модуль
`frontend/app/auth/page.tsx`, `frontend/app/components/ui.tsx`

## Зависимости
- `tasks/08-frontend/08-28-sidebar-fix-and-ai-guard-brand.md`
- `tasks/08-frontend/08-29-api-keys-page-modal-grouping-and-key-scope-jump.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Invite запускается через modal и сохраняет текущую бизнес-логику.
- Список сотрудников можно переключать между группировками.
- RBAC-ограничения соблюдены без регрессий.
