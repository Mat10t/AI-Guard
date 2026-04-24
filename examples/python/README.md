# Python API Examples

Минимальные примеры для интеграции с MVP LLM Gateway через API ключ.

В текущем MVP production-модели каталога: `gpt-5.4-mini`, `gpt-5.4`, `gemini-2.5-flash`.
Модель `mock-fast` используется для локального демо и fallback.

## Что внутри
- `bootstrap_and_chat.py`:
создает/логинит пользователя, создает проект, выпускает API-ключ и делает первый запрос в gateway.
- `chat_with_api_key.py`:
делает запрос в gateway по уже готовому `api_key`.
- `openai_sdk_compatible.py`:
делает запрос через Python SDK `openai` с `base_url` на наш gateway.
- `ask_model_about_itself.py`:
минимальный OpenAI-compatible вызов к нашему gateway с запросом "расскажи о себе".
- `gemini_via_gateway_openai_compat.py`:
OpenAI-compatible вызов модели `gemini-2.5-flash` через наш gateway.

## Подготовка
```bash
cd examples/python
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

Опционально подготовьте окружение:
```bash
cp .env.example .env
set -a
source .env
set +a
```

## 1) Полный bootstrap сценарий
```bash
python bootstrap_and_chat.py
```

Скрипт выведет:
- `email`
- `project_id`
- `api_key`
- JSON-ответ от `/v1/chat/completions`

По умолчанию используется модель `mock-fast` (подходит для локального демо без внешнего OpenAI ключа).

Пример вызова production-модели:
```bash
export LLM_GATEWAY_API_KEY="sk_..."
python openai_sdk_compatible.py --base-url http://localhost:8080/v1 --model gpt-5.4-mini
```

## 2) Запрос с существующим API ключом
```bash
export LLM_GATEWAY_API_KEY="sk_..."
python chat_with_api_key.py --model mock-fast --prompt "Привет"
```

## 3) OpenAI SDK-compatible вызов
```bash
export LLM_GATEWAY_API_KEY="sk_..."
python openai_sdk_compatible.py --base-url http://localhost:8080/v1 --model mock-fast
```

## 4) Простой запрос "расскажи о себе"
```bash
export LLM_GATEWAY_API_KEY="sk_..."
python ask_model_about_itself.py --base-url http://localhost:8080/v1 --model mock-fast
```

## 5) Gemini через OpenAI-compatible gateway
```bash
export LLM_GATEWAY_API_KEY="sk_..."
python gemini_via_gateway_openai_compat.py --base-url http://localhost:8080/v1 --model gemini-2.5-flash
```

Если видите `[mock-fallback]`, значит Gemini недоступен и gateway ушел в fallback.

## Полезные параметры
`bootstrap_and_chat.py`:
- `--auth-url` (default `http://localhost:8081`)
- `--project-url` (default `http://localhost:8082`)
- `--gateway-url` (default `http://localhost:8080`)
- `--org-name`
- `--email`
- `--password`
- `--project-name`
- `--model`
- `--prompt`

`chat_with_api_key.py`:
- `--gateway-url` (default `http://localhost:8080`)
- `--api-key`
- `--model`
- `--prompt`

`openai_sdk_compatible.py`:
- `--base-url` (default `http://localhost:8080/v1`)
- `--api-key`
- `--model`
- `--prompt`

`ask_model_about_itself.py`:
- `--base-url` (default `http://localhost:8080/v1`)
- `--api-key`
- `--model`

`gemini_via_gateway_openai_compat.py`:
- `--base-url` (default `http://localhost:8080/v1`)
- `--api-key`
- `--model` (default `gemini-2.5-flash`)
- `--prompt`
