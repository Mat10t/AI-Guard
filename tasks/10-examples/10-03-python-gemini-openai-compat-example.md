# Заголовок
Python пример Gemini через OpenAI-compatible gateway

## Контекст
После добавления `gemini-2.5-flash` в каталог нужен отдельный короткий пример для команды, показывающий вызов Gemini через тот же OpenAI-compatible интерфейс gateway.

## Что нужно сделать
- Добавить Python-скрипт в `examples/python` для вызова `model=gemini-2.5-flash`.
- Использовать существующий OpenAI SDK клиент с `base_url=http://localhost:8080/v1`.
- Оставить авторизацию через проектный `LLM_GATEWAY_API_KEY` (`sk_...`).
- Обновить `examples/python/README.md` отдельной командой запуска Gemini-примера.

## Связанный сервис / модуль
`examples/python`, `api-gateway`, `provider-catalog-service`.

## Зависимости
- `10-02-python-openai-sdk-compat-example.md`
- `07-04-gemini-provider-adapter.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- Скрипт отправляет `chat.completions` запрос через gateway с моделью `gemini-2.5-flash`.
- Запуск документирован в `examples/python/README.md`.
- Пример не требует изменения публичных backend API.
