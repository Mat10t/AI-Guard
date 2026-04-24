# Заголовок
Analytics sections: timeseries и CSV по разделам

## Контекст
Сейчас analytics и экспорт CSV реализованы как один агрегированный поток, что ухудшает UX и ограничивает сценарии отчетности.

## Что нужно сделать
- Добавить timeseries endpoint:
  - `GET /analytics/timeseries?metric=tokens|cost|error_rate|fallback_rate&bucket=hour|day|week|month|year|all&from=&to=`.
- Добавить разделенные CSV endpoint’ы:
  - `GET /reports/csv/usage`
  - `GET /reports/csv/audit`
  - `GET /reports/csv/logs`
- Оставить `GET /reports/csv` как deprecated alias на usage CSV.
- Поддержать RBAC для новых endpoint’ов на уровне текущей модели ролей.
- Добавить техничные пустые/ошибочные ответы в едином формате ошибок.

## Связанный сервис / модуль
`audit-analytics-service`.

## Зависимости
- `00-03-requirements-update-multi-key.md`
- `06-01-audit-ingestion.md`
- `06-02-usage-analytics-and-csv.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Timeseries возвращает точки по всем поддержанным метрикам и bucket’ам.
- CSV выгрузки работают раздельно для usage/audit/logs.
- Legacy `/reports/csv` сохраняет обратную совместимость.
