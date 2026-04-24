# Заголовок
Python пример через OpenAI SDK (base_url на наш gateway)

## Контекст
Сервис заявлен как OpenAI-compatible, поэтому нужен отдельный пример интеграции через официальную Python-библиотеку OpenAI.

## Что нужно сделать
- Добавить Python-скрипт `openai_sdk_compatible.py` в `examples/python`.
- Использовать `OpenAI(base_url=..., api_key=...)` и вызов `client.chat.completions.create(...)`.
- Настроить значения через CLI-аргументы и переменные окружения.
- Обновить `requirements.txt`, `.env.example` и `examples/python/README.md`.

## Связанный сервис / модуль
`examples/python`.

## Зависимости
- `10-01-python-api-usage-examples.md`
- `02-01-gateway-public-api.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- Пример запускается с `base_url=http://localhost:8080/v1`.
- Запрос к `chat.completions` проходит с проектным `api_key` (`sk_...`).
- В README есть отдельный раздел запуска SDK-совместимого примера.
