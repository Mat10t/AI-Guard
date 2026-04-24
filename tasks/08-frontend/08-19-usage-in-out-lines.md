# Заголовок
Usage UI: отдельные линии input/output токенов

## Контекст
На странице usage график токенов сейчас агрегированный, без разделения на вход и выход.

## Что нужно сделать
- В `frontend /analytics/usage` загрузить `input_tokens` и `output_tokens` timeseries.
- Обновить график “Tokens over time”: две линии `input_tokens` и `output_tokens`.
- Сохранить переключение scale (`hour/day/week/month/year/all`) и текущие графики стоимости/стабильности.

## Связанный сервис / модуль
`frontend/app/analytics/usage/page.tsx`.

## Зависимости
- `06-04-timeseries-input-output-tokens.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- На графике токенов видны две отдельные линии (in/out).
- График корректно обновляется при смене масштаба.
