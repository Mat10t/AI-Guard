# Заголовок
Analytics UI: разделы, навигация и графики

## Контекст
Текущая страница `/analytics` перегружена и содержит кнопки, которые не образуют явную навигацию по разделам данных.

## Что нужно сделать
- Разделить analytics на страницы:
  - `/analytics/usage`
  - `/analytics/audit`
  - `/analytics/logs`
  - `/analytics/providers`
- Сделать рабочую навигацию-кнопки между разделами.
- В каждой секции добавить отдельный CSV-экспорт (где разрешено ролью).
- Добавить графики (Recharts):
  - tokens over time;
  - cost over time;
  - error_rate over time;
  - fallback_rate over time;
  - top groups (project/model).
- Добавить общий scale control: `hour/day/week/month/year/all`.
- Сохранить mobile-first layout и desktop adaptation без отдельной логики.

## Связанный сервис / модуль
`frontend` (`/analytics*`, shared UI/components).

## Зависимости
- `06-03-analytics-sections-timeseries-and-csv.md`
- `08-06-role-aware-projects-analytics.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Переходы по разделам аналитики работают как маршрутизация, а не как «тихие» кнопки.
- В разделах usage/audit/logs доступны корректные CSV-экспорты.
- Графики рендерятся и переключаются по временному масштабу.
- На мобильной ширине интерфейс остаётся читаемым и рабочим.
