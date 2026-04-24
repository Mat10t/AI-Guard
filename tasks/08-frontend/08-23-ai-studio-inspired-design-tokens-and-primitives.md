# Заголовок
AI Studio-inspired dark design tokens и UI primitives

## Контекст
Нужен единый визуальный фундамент в стиле Google AI Studio: темная тема, плотные панели, табличный интерфейс и согласованные элементы управления.

## Что нужно сделать
- Пересобрать `globals.css` под dark-only токены и layout-переменные.
- Обновить базовые UI primitives (`Card`, `Button`, `Field`, `Notice`, `EmptyState`) в сторону panel/table UX.
- Добавить универсальные стили для таблиц, разделителей, action-toolbar и badge/chip.
- Сохранить текущую business-логику и API-взаимодействия без изменений.

## Связанный сервис / модуль
`frontend`

## Зависимости
- `tasks/08-frontend/08-22-analytics-scope-switcher-and-project-member-ui.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- Все основные страницы используют единый dark visual foundation.
- Примитивы UI переиспользуются без дублирования стилей.
- Нет изменений публичных контрактов backend.
