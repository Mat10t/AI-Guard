# Заголовок
AI Studio-like UI tuning: colors, buttons, layout density

## Контекст
Темная тема уже внедрена, но цвета, формы кнопок и плотность layout пока не совпадают с целевым визуальным стилем.

## Что нужно сделать
- Подтянуть палитру к более нейтральному dark baseline.
- Привести action-кнопки к rounded/pill форме и согласованным состояниям active/secondary/ghost.
- Унифицировать spacing/height для:
  - section action-bar,
  - segmented controls,
  - table rows,
  - modal headers.
- Сохранить существующую структуру AppShell и mobile-first адаптацию.

## Связанный сервис / модуль
`frontend/app/globals.css`, `frontend/app/components/ui.tsx`

## Зависимости
- `tasks/00-planning/00-09-keys-projects-modal-flow-and-style-contracts.md`

## Приоритет
P1

## MVP
да

## Критерии готовности
- Интерфейс визуально ближе к референсу (без пиксельного копирования).
- Кнопки и таблицы выглядят консистентно на mobile и desktop.
