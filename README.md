# MVP B2B LLM Gateway

Микросервисный MVP единого LLM-шлюза для компаний.

## Сервисы
- `auth-org-service`: регистрация организации, login/logout/refresh, участники и роли.
- `project-key-service`: проекты и API-ключи проекта (multi-key, отзыв по `key_id`).
- `limits-service`: лимиты токенов на проект/ключ и runtime-проверка.
- `provider-catalog-service`: каталог моделей, статусы провайдеров, routing profile, internal Gemini adapter.
- `api-gateway`: OpenAI-compatible endpoint, retry/timeout/fallback.
- `audit-analytics-service`: аудит, usage аналитика, timeseries, экспорт CSV по разделам.
- `frontend` (Next.js): mobile-first интерфейсы для auth/projects/logs/analytics.

## Локальный запуск
1. `make tidy`
2. `make up`
3. Smoke-проверка критического backend-пути: `make smoke`
4. UI: `http://localhost:3001`
5. PostgreSQL с хоста: `localhost:5433` (в контейнере остаётся `5432`)
6. Остановка: `make down`

Для реальных провайдеров при необходимости задайте переменные окружения:
- `OPENAI_API_KEY` для маршрутов `gpt-5.4-mini` / `gpt-5.4`;
- `GEMINI_API_KEY` для маршрута `gemini-2.5-flash`.

## Demo runbook (backend, без ручных curl)
1. Поднять стек: `make up`
2. Прогнать API smoke: `make smoke`
3. Интеграционные тесты: `make integration`
4. E2E critical flow (+ Kafka events): `make e2e`
5. Полный тестовый прогон: `make test`

## Локальные команды
- `make test` — все Go-тесты
- `make unit` — unit-тесты (`internal` + `services`)
- `make integration` — integration smoke/API-тесты
- `make e2e` — критические e2e-сценарии MVP
- `make smoke` — shell smoke runner (register -> project -> key -> gateway -> limit -> revoke)
- `make quality` — локальный quality/security проход (`go vet` + `govulncheck` + `go test -coverprofile`)
- `make load-test` — базовые нагрузочные тесты публичного API (k6)
- `make postman` — запуск Postman-коллекции через Newman (если установлен `newman`)
- `make build` — сборка сервисов
- `make up` — запуск окружения
- `make down` — остановка окружения
- `make rebuild` — пересборка контейнеров

## Стек
- Backend: Go
- Data: PostgreSQL, Redis, Kafka
- Observability: Prometheus, Grafana
- Containerization: Docker, docker-compose

## Frontend (mobile-first, real API)
- Страницы подключены к backend через Next.js proxy: `frontend/app/api/proxy/[...path]/route.ts`.
- Сначала реализованы мобильные экраны (`/auth`, `/projects`, `/analytics/*`), desktop — через media-query адаптацию тех же экранов.
- Токен хранится в `localStorage` для рабочих вызовов API; refresh/logout идут через HttpOnly cookie.
- В разделе Projects API-ключ всегда отображается в маске (`начало...конец`), копирование полного значения доступно только через кнопки, полный ключ возвращается только при create.
- Аналитика разделена на секции: `/analytics/usage`, `/analytics/audit`, `/analytics/logs`, `/analytics/providers`.
- В `Analytics -> Usage` график токенов показывает две линии: `input_tokens` и `output_tokens`.
- В `Analytics -> Providers` статус провайдеров проверяется live (с TTL-кэшем 30s), а Admin может обновлять цены моделей вручную.
- После обновления цены модели project-лимиты для этой `billing_model` автопересчитываются на backend без ручного пересохранения.
- Модели MVP в каталоге: `gpt-5.4-mini`, `gpt-5.4`, `gemini-2.5-flash`, `mock-fast` (fallback/demo).
- Политика цен и дата последней актуализации: `docs/pricing.md`.

## Test env overrides
- `RUN_INTEGRATION=1` и `RUN_E2E=1` выставляются через `make integration` и `make e2e`.
- Для проверки Kafka в e2e: `KAFKA_BROKERS=localhost:19092` (по умолчанию уже так).
- Если нужно временно выключить Kafka-ассерты в e2e: `VERIFY_KAFKA_EVENTS=0 make e2e`.

## Postman collections
- Коллекции для API-задач: `postman/`.
- Документация по импорту/запуску: `postman/README.md`.
- Локальный запуск через Newman: `make postman`.

## Python examples
- Примеры клиентской интеграции находятся в `examples/python`.
- Быстрый старт:
  - `cd examples/python`
  - `python3 -m venv .venv && source .venv/bin/activate`
  - `pip install -r requirements.txt`
  - `python bootstrap_and_chat.py`
