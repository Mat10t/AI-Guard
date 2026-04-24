# Заголовок
Добавление провайдера Gemini и модели gemini-2.5-flash

## Контекст
MVP сейчас ориентирован на OpenAI + mock fallback. Для сценариев демонстрации multi-provider нужен второй production-провайдер.

## Что нужно сделать
- Добавить в каталог модель `gemini-2.5-flash` с `provider=gemini` и demo pricing.
- Добавить `routing_rules` для `gemini-2.5-flash`:
  - `primary=gemini`, `fallback=mock`.
- Реализовать internal adapter в `provider-catalog-service`:
  - `POST /internal/gemini/completions`
  - вход OpenAI-compatible, выход OpenAI-compatible.
- Добавить конфиг:
  - `GEMINI_API_KEY`
  - `GEMINI_API_URL`.
- Проверить, что `api-gateway` использует новую модель через `GET /internal/route` без изменения внешнего API.

## Связанный сервис / модуль
`provider-catalog-service`, `db/init.sql`, `docker-compose.yml`.

## Зависимости
- `00-03-requirements-update-multi-key.md`
- `07-01-provider-catalog.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `/catalog/models` возвращает `gemini-2.5-flash`.
- Маршрут `gemini-2.5-flash` доступен через `/internal/route`.
- При недоступности Gemini шлюз корректно переключается в mock fallback.
