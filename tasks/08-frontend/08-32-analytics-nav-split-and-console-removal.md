# Заголовок
Analytics nav split + console removal from root flow

## Контекст
Промежуточная страница `Analytics` и “консольная” подача на `/` усложняют навигацию и не соответствуют целевому UX.

## Что нужно сделать
- Сделать `/analytics` redirect-роутом в первый доступный раздел (`usage|logs|audit|providers`).
- Обновить аналитику, чтобы она принимала query-параметры `scope/project_id/api_key_id` при прямом входе.
- Упростить `/`:
  - гость: короткий auth-first landing,
  - авторизованный: redirect на `/api-keys`.
- Убрать упоминания “консоли” из интерфейсных текстов.

## Связанный сервис / модуль
`frontend/app/page.tsx`, `frontend/app/analytics/page.tsx`, `frontend/app/analytics/*/page.tsx`

## Зависимости
- `tasks/08-frontend/08-28-sidebar-fix-and-ai-guard-brand.md`
- `tasks/08-frontend/08-29-api-keys-page-modal-grouping-and-key-scope-jump.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Прямые пункты `Usage/Logs/Audit/Providers` работают без промежуточного экрана.
- `/` не перегружает интерфейс и корректно редиректит авторизованного пользователя.
- Query preselect key-scope корректно подхватывается analytics страницами.
