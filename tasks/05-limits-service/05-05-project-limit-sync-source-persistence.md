# Заголовок
Project limits: хранение `sync_source` и обратная совместимость

## Контекст
После добавления budget/token sync нужно хранить источник синхронизации в БД, чтобы автопересчет цен работал детерминированно без ручных действий пользователя.

## Что нужно сделать
- Добавить `sync_source` в таблицу `limits` для project scope.
- В `PUT /limits/projects/{id}` сохранять `sync_source` вместе с лимитом.
- В `GET /limits/projects/{id}` возвращать `sync_source`.
- Для старых записей по умолчанию ставить `sync_source='tokens'`.

## Связанный сервис / модуль
`limits-service`, `db/init.sql`, `db/migrations`.

## Зависимости
- `05-03-project-budget-sync.md`
- `07-07-catalog-pricing-admin-update.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `sync_source` хранится и читается через project limits API.
- Старые записи не ломаются и интерпретируются как `tokens`.
- Runtime check остается совместимым и использует эффективный `token_limit`.
