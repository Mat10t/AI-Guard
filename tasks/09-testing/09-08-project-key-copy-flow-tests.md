# Заголовок
Тесты потока получения и копирования API-ключа

## Контекст
После изменения контракта `GET /projects/{id}/key` нужно зафиксировать поведение тестами, чтобы не потерять удобный flow интеграции.

## Что нужно сделать
- Добавить integration-проверку: после `issue key` вызов `GET /projects/{id}/key` возвращает `api_key`.
- Добавить e2e-проверку: возвращаемый `api_key` из `GET /projects/{id}/key` работает в `POST /v1/chat/completions`.
- Убедиться, что revoke продолжает блокировать runtime-запросы.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`.

## Зависимости
- `04-03-persistent-api-key-retrieval.md`
- `08-11-project-key-copy-button-persistent.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- Новые проверки проходят локально в docker-compose окружении.
- Регрессий в текущих smoke/integration/e2e сценариях нет.
