# Заголовок
Контракт фактической модели в usage (requested/effective)

## Контекст
При fallback запрос может выполняться не на той модели, которую запросил клиент. Для корректной аналитики и стоимости нужно фиксировать обе модели: запрошенную и фактическую.

## Что нужно сделать
- Зафиксировать поля usage-события: `requested_model`, `effective_model`, `model`.
- Зафиксировать семантику совместимости: `model` хранится как alias фактической модели.
- Зафиксировать семантику аналитики: `group_by=model` агрегирует по фактической модели.

## Связанный сервис / модуль
`docs/mvp-system-contracts.md`, `api-gateway`, `audit-analytics-service`

## Зависимости
- `tasks/00-planning/00-05-scoped-analytics-and-project-assignment-contracts.md`
- `tasks/00-planning/00-06-audit-hard-scope-columns-contract.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Контракт usage payload описывает `requested_model` и `effective_model` без двусмысленности.
- Зафиксировано backward-compatible поведение для legacy поля `model`.
- Описано, что UI/analytics `group_by=model` использует фактическую модель ответа.
