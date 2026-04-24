# Заголовок
API Keys page: modal create + grouping + jump to key usage

## Контекст
Управление ключами сейчас смешано с проектами, что перегружает UI. Нужен отдельный API Keys hub в стиле AI Studio.

## Что нужно сделать
- Добавить страницу `/api-keys`:
  - табличный список ключей,
  - группировка `By API key` / `By Project`,
  - действия `copy`, `revoke`, `open usage`.
- Добавить modal создания ключа:
  - поля `Key name`, `Project`,
  - inline создание проекта из модалки и автовыбор нового проекта.
- Реализовать переход из строки ключа в analytics key-scope:
  - `/analytics/usage?scope=key&project_id=...&api_key_id=...`.
- Поддержать явные success/error состояния, без “тихих” кликов.

## Связанный сервис / модуль
`frontend/app/api-keys/page.tsx`, `frontend/app/lib/session.tsx`, `frontend/app/components/ui.tsx`

## Зависимости
- `tasks/04-project-key-service/04-07-api-key-name-support.md`
- `tasks/08-frontend/08-28-sidebar-fix-and-ai-guard-brand.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Создание ключа выполняется через modal.
- Группировка по key/project работает.
- Переход в usage key-scope из строки ключа работает.
