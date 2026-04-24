# Заголовок
Регрессия MVP+: multi-key, analytics sections, Gemini

## Контекст
Изменения затрагивают критические контракты ключей, analytics API/UI и маршрутизацию провайдеров; без расширенной регрессии высок риск сломать MVP-сценарии.

## Что нужно сделать
- Обновить integration/e2e под новые key endpoint’ы `/keys`.
- Добавить проверки multi-key:
  - два активных ключа в одном проекте;
  - отзыв одного ключа не влияет на второй.
- Добавить проверки analytics endpoint’ов:
  - `/analytics/timeseries` для всех metric/bucket;
  - `/reports/csv/usage|audit|logs`.
- Добавить проверки Gemini-маршрута и fallback на mock при недоступности провайдера.
- Проверить frontend regression checklist по разделам analytics и role-aware visibility.

## Связанный сервис / модуль
`tests/integration`, `tests/e2e`, `services/*`, `frontend`.

## Зависимости
- `04-04-project-multi-key-management.md`
- `06-03-analytics-sections-timeseries-and-csv.md`
- `07-04-gemini-provider-adapter.md`
- `08-16-analytics-sections-nav-and-charts.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `make test`, `make integration`, `make e2e` проходят на обновленных контрактах.
- Multi-key, analytics CSV/timeseries и Gemini fallback покрыты автоматическими тестами.
- Нет регрессий в критическом e2e пути из `Требования.md`.
