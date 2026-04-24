# Заголовок
Providers pricing controls: скрытие редактирования для не-Admin

## Контекст
В разделе `Analytics -> Providers` кнопки и поля изменения цен должны быть доступны только роли `Admin`; для остальных ролей нужен строго read-only режим без action-кнопок.

## Что нужно сделать
- Вынести право изменения цен моделей в capability (`canUpdatePricing`) в общем RBAC слое.
- На странице `Analytics -> Providers` использовать capability вместо локальной проверки роли.
- Для не-Admin скрыть editable-элементы:
  - `Input cost`,
  - `Output cost`,
  - `Pricing source`,
  - кнопку `Сохранить цену`,
  - warning о запрете редактирования.
- Оставить для не-Admin read-only отображение цен и времени обновления.
- Сохранить защиту в обработчике сохранения (ранний выход при отсутствии прав).

## Связанный сервис / модуль
`frontend/app/lib/rbac.ts`, `frontend/app/analytics/providers/page.tsx`.

## Зависимости
- `08-20-pricing-update-autosync-feedback.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- `Admin` видит и использует controls изменения цены.
- `PM/Dev/Finance` не видят controls редактирования и не видят кнопку сохранения.
- Для всех ролей сохраняется корректный read-only просмотр текущих цен.
