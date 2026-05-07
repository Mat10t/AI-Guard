# Заголовок
SonarQube в локальном стеке и quality gate

## Контекст
Для закрытия пункта «в процессе сборки используется статический анализ кода (Sonar)» нужно добавить локально воспроизводимый Sonar-процесс в рамках существующего `docker-compose` и команд Makefile, без внешнего CI/CD.

## Что нужно сделать
- Добавить SonarQube в локальный `docker-compose` стек.
- Добавить конфигурацию сканирования проекта (`sonar-project.properties`).
- Добавить скрипт запуска quality-проверок (`go vet`, `govulncheck`, `go test -coverprofile`).
- Добавить скрипт запуска sonar-scanner с ожиданием готовности SonarQube и сохранением отчета.
- Добавить команды `make quality` и `make sonar-scan`.

## Связанный сервис / модуль
`infrastructure`, `docker-compose`, `quality tooling`

## Зависимости
- `tasks/00-planning/00-12-good-grade-closure-matrix.md`

## Приоритет
P0

## MVP
да

## Критерии готовности
- SonarQube поднимается локально вместе с окружением.
- `make quality` выполняется и формирует результат статпроверок.
- `make sonar-scan` публикует анализ в локальный SonarQube и формирует отчет.

## Service Fields (обязательно для новых задач)
- Branch: `feat/01-05-sonarqube-in-local-stack`
- PR: `TBD`
- Merged at: `TBD`
