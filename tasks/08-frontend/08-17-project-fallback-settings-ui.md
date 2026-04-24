# Заголовок
UI проекта: настройка fallback-модели

## Контекст
В интерфейсе проектов нет управления fallback-маршрутом, хотя backend поддерживает resilience и требуется проектная настройка fallback.

## Что нужно сделать
- В `/projects` добавить блок `Fallback routing` в карточке проекта.
- Подгружать текущее значение через `GET /projects/{id}/routing`.
- Добавить select: `По умолчанию` + модели из `/catalog/models`.
- Добавить действие сохранения через `PUT /projects/{id}/routing`.
- Для `Admin|PM` блок editable; для остальных — скрыт или read-only по текущей RBAC-матрице.
- Добавить явные success/error notice.

## Связанный сервис / модуль
`frontend` (`/projects`).

## Зависимости
- `04-05-project-fallback-routing-settings.md`
- `02-06-fallback-model-override-by-project.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- В карточке проекта можно выбрать и сохранить fallback-модель.
- В списке есть значение `По умолчанию` (без override).
- Неавторизованные/неподходящие роли не получают доступ к изменению.
- UI показывает явный результат действия (success/error).
