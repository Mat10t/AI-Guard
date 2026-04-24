# Pricing Policy (MVP)

## Что хранится в системе
- Цены моделей хранятся в `provider_models`:
  - `input_cost` (USD за 1K input tokens),
  - `output_cost` (USD за 1K output tokens),
  - `pricing_source` (источник обновления),
  - `pricing_updated_at` (время последней актуализации).

## Как обновляются цены
- В MVP обновление выполняется вручную через endpoint:
  - `PUT /catalog/models/{id}/pricing` (только роль `Admin`).
- UI-блок обновления доступен в разделе `Analytics -> Providers`.

## Источник официальных прайсов
- OpenAI: публичная страница цен OpenAI.
- Google Gemini: публичная страница цен Google AI / Gemini API.

## Зафиксированная дата последней актуализации в репозитории
- April 20, 2026.

## Модели, покрытые в MVP
- `gpt-5.4-mini`
- `gpt-5.4`
- `gemini-2.5-flash`
- `mock-fast` (демо-модель, цена обычно `0/0`)

## Ограничения MVP
- Нет автоматического web-scraping/cron-синхронизации цен.
- Изменение цены действует сразу для новых usage-записей.
