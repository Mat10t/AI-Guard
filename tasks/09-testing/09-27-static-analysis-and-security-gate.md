# Заголовок
Статический анализ и security gate для локального процесса сборки

## Контекст
Нужно формально подтвердить, что статический анализ встроен в процесс сборки и проект не имеет критичных уязвимостей. Для MVP это должно быть воспроизводимо локально и без внешней CI-инфраструктуры.

## Что нужно сделать
- Встроить `go vet` и `govulncheck` в отдельный quality-проход.
- Добавить запуск unit/integration покрытия для отчета качества.
- Сформировать единый отчет по quality/security-проверкам в `docs/reports`.
- Связать результаты quality-прохода с Sonar-сканированием.

## Связанный сервис / модуль
`tests`, `quality`, `docs/reports`

## Зависимости
- `tasks/00-planning/00-12-good-grade-closure-matrix.md`
- `tasks/01-infrastructure/01-05-sonarqube-in-local-stack.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- `make quality` запускает `go vet`, `govulncheck` и coverage-проход.
- По итогам quality-прохода есть артефакт отчета с датой запуска.
- Набор проверок интегрирован в инструкции запуска для защиты.

## Service Fields (обязательно для новых задач)
- Branch: `feat/09-27-static-analysis-and-security-gate`
- PR: `TBD`
- Merged at: `TBD`
