# Заголовок
Copy anytime + hide revoked keys in all UI

## Контекст
На frontend полный ключ недоступен после перезагрузки, а revoked ключи продолжают показываться в API Keys и analytics key scope.

## Что нужно сделать
- `/api-keys`:
  - заполнять `keySecrets` из `GET /projects/{id}/keys` (поле `api_key`);
  - оставить copy доступным для активных ключей без привязки к моменту create;
  - скрыть revoked ключи в обоих режимах (`By API key`, `By Project`).
- Analytics (`usage`, `logs`, `audit`):
  - в `loadScopeKeys` показывать только активные ключи;
  - если выбранный `api_key_id` стал недоступен, сбрасывать выбор и показывать notice.

## Связанный сервис / модуль
`frontend/app/api-keys/page.tsx`, `frontend/app/analytics/*/page.tsx`

## Зависимости
- `tasks/04-project-key-service/04-08-list-keys-with-secret-for-active.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Copy для активного ключа работает после reload.
- Revoked ключи не отображаются в API Keys и key-селекторах analytics.
