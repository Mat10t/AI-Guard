# Заголовок
Каталог моделей OpenAI 5.4 для MVP

## Контекст
В текущем MVP в каталоге и маршрутизации остаются только `gpt-4o-mini` и `mock-fast`, из-за чего UI и лимиты не отражают целевую линейку моделей 5.4.

## Что нужно сделать
- Обновить seed каталога моделей на `gpt-5.4-mini`, `gpt-5.4`, `mock-fast`.
- Обновить seed routing rules:
  - `gpt-5.4-mini`: `openai -> mock`.
  - `gpt-5.4`: `openai -> mock`.
  - `mock-fast`: `mock -> mock`.
- Оставить retry/timeout для openai-моделей как в текущем MVP-профиле.
- Зафиксировать demo-цены для `gpt-5.4-mini` и `gpt-5.4` в `provider_models`.

## Связанный сервис / модуль
`provider-catalog-service`, `db/init.sql`.

## Зависимости
- `07-01-provider-catalog.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `/catalog/models` возвращает `gpt-5.4-mini`, `gpt-5.4`, `mock-fast`.
- `/internal/route` корректно находит маршруты для обеих 5.4-моделей.
- Fallback на `mock` работает без изменений по контракту.
