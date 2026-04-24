# Заголовок
App Shell: desktop sidebar + mobile drawer navigation

## Контекст
Текущий topbar не соответствует целевому AI Studio-подобному UX. Нужна структурная навигация: sidebar на desktop и drawer на мобильных устройствах.

## Что нужно сделать
- Перестроить `AppShell`:
  - desktop: фиксированный левый sidebar,
  - mobile: кнопка меню + drawer + overlay.
- Сохранить RBAC-видимость пунктов навигации.
- Оставить существующие маршруты (`/`, `/auth`, `/projects`, `/analytics/*`) без изменений.
- Добавить верхнюю контент-панель (section title + context actions zone).

## Связанный сервис / модуль
`frontend/app/components/app-shell.tsx`, `frontend/app/globals.css`

## Зависимости
- `tasks/08-frontend/08-23-ai-studio-inspired-design-tokens-and-primitives.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Навигация работает на mobile и desktop без потери текущих переходов.
- RBAC-ограничения в nav сохранены.
- Выход из системы доступен из shell и работает как раньше.
