# Заголовок
Landing/Auth dark AI Studio flow (auth-first, без новых сценариев)

## Контекст
Публичная часть (`/` и `/auth`) должна соответствовать новому dark UI кабинета и оставаться понятной для пользователя без дизайнера.

## Что нужно сделать
- Перекомпоновать `/` в auth-first landing с понятными CTA.
- Перестроить `/auth` (login/register/invite accept/organization session) в единый dark layout.
- Сохранить текущие режимы и поведение:
  - register/login,
  - invite accept + auto-login,
  - members/invite/session/developer mode для авторизованного пользователя.
- Не добавлять новый функционал и не менять backend-контракты.

## Связанный сервис / модуль
`frontend/app/page.tsx`, `frontend/app/auth/page.tsx`, `frontend/app/globals.css`

## Зависимости
- `tasks/08-frontend/08-23-ai-studio-inspired-design-tokens-and-primitives.md`
- `tasks/08-frontend/08-24-shell-sidebar-drawer-navigation.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Публичный UX визуально консистентен с кабинетом.
- Все auth и invite сценарии работают как ранее.
- Нет добавленных endpoint’ов или бизнес-правил.
