# Матрица закрытия требований на оценку «хорошо»

| Требование | Артефакт | Команда проверки |
|---|---|---|
| Для проекта существует система сборки | `Makefile` | `make build` |
| Все сервисы упакованы в контейнеры | `services/*/Dockerfile`, `docker-compose.yml` | `make up` |
| В проекте присутствуют unit tests, встроенные в процесс сборки | `go test` в `Makefile` | `make test` |
| Для публичного API реализованы нагрузочные тесты | `tests/load/*.js`, `scripts/load_test_*.sh`, `docs/reports/load-test-latest.md` | `make load-test` |
| В процессе сборки используется статический анализ кода и контроль уязвимостей | `scripts/quality_check.sh`, `scripts/sonar_scan.sh`, `sonar-project.properties`, `docs/reports/static-analysis-latest.md` | `make quality && make sonar-scan` |

## Пояснение по Gradle/Maven

Проект реализован на Go. Для Go не используется Gradle/Maven как стандартный инструмент сборки.  
Требование «Gradle, Maven или другая система» закрывается через `Makefile`, который оркестрирует `go build`, `go test`, интеграционные/e2e проверки, нагрузочные тесты и quality/security проверки.
