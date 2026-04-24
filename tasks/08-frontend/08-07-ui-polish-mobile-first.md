# Заголовок
UI polish: mobile-first quality, reusable primitives, developer mode

## Контекст
Нужен единый, чистый и предсказуемый интерфейс B2B-консоли для демонстрации MVP.

## Что нужно сделать
- Ввести переиспользуемые UI-примитивы (Card, SectionHeader, Notice, EmptyState, controls).
- Убрать token textarea из основного UX.
- Добавить `Developer mode` для отображения токена и ручной отладки.
- Добавить явные loading/empty/error состояния на страницах.
- Привести spacing/typography/цвета к единому mobile-first стилю.

## Связанный сервис / модуль
`frontend`.

## Зависимости
- `08-04-auth-session-guard-shell.md`
- `08-05-landing-auth-first.md`
- `08-06-role-aware-projects-analytics.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- UI выглядит единообразно на всех страницах.
- Основные сценарии выполняются без "ручного" копирования токенов.
- Developer mode доступен, но не мешает обычному пользовательскому потоку.
