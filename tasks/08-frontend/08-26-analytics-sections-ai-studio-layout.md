# Заголовок
Analytics pages: AI Studio-like sections, tables и action bars

## Контекст
`/analytics` и дочерние страницы должны выглядеть как единый console-интерфейс: секционная навигация, верхние действия, табличные панели и читаемые графики.

## Что нужно сделать
- Перестроить `/analytics` landing и страницы `/analytics/usage`, `/analytics/logs`, `/analytics/audit`, `/analytics/providers` в едином стиле.
- Сохранить текущие scope/filter/export действия и API-вызовы.
- Представить logs/audit/providers в table/panel-first подаче.
- Для usage оставить текущие графики и controls, улучшив визуальную иерархию и плотность.
- Сохранить RBAC-ограничения и видимость действий.

## Связанный сервис / модуль
`frontend/app/analytics/*`, `frontend/app/globals.css`

## Зависимости
- `tasks/08-frontend/08-23-ai-studio-inspired-design-tokens-and-primitives.md`
- `tasks/08-frontend/08-24-shell-sidebar-drawer-navigation.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Все analytics sections визуально консистентны между собой.
- Scope/filter/export продолжают работать без изменений контрактов.
- Dev/PM/Admin/Finance видят только разрешенные разделы и actions.
