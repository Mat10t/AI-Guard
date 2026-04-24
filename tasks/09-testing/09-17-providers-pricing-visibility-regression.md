# Заголовок
Регрессия UI-прав: pricing controls только для Admin

## Контекст
После рефакторинга RBAC и Providers UI нужно зафиксировать, что изменение цен видно и доступно только Admin, а остальные роли остаются в read-only режиме.

## Что нужно сделать
- Проверить фронтенд-матрицу видимости controls:
  - `Admin` видит поля и кнопку сохранения,
  - `PM/Dev/Finance` не видят поля/кнопку.
- Проверить, что не-Admin не может инициировать save через штатный UI-поток.
- Проверить backend-ограничение:
  - `PUT /catalog/models/{id}/pricing` для не-Admin возвращает `403`.
- Прогнать сборку фронтенда и integration-regression.

## Связанный сервис / модуль
`frontend`, `tests/integration`, `provider-catalog-service`.

## Зависимости
- `08-21-providers-pricing-admin-only-controls.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- Визуальная матрица ролей по pricing controls подтверждена.
- Backend RBAC по pricing endpoint не регрессировал.
- `npm --prefix frontend run build` и `make integration` проходят.
