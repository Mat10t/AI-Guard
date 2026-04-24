# Заголовок
Docker Compose стек для локального MVP

## Контекст
Нужен единый локальный запуск всех сервисов и инфраструктуры без внешнего CI/CD.

## Что нужно сделать
- Собрать `docker-compose.yml` для Go-сервисов, PostgreSQL, Redis, Kafka, Prometheus, Grafana.
- Добавить healthchecks, сети и volume для локальной разработки.
- Подготовить `.env.example` для портов и базовых конфигураций.

## Связанный сервис / модуль
Infrastructure / docker-compose.

## Зависимости
- `00-02-system-contracts.md`.

## Приоритет
P0

## MVP
да

## Критерии готовности
- Команда `docker compose up` поднимает весь стек локально.
- Все сервисы доступны по объявленным портам.
- Сервисам доступны PostgreSQL/Redis/Kafka.
