#!/usr/bin/env python3
import argparse
import json
import os
import sys

import requests


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Call LLM Gateway using existing project API key."
    )
    parser.add_argument(
        "--gateway-url",
        default=os.getenv("GATEWAY_URL", "http://localhost:8080"),
        help="Gateway base URL.",
    )
    parser.add_argument(
        "--api-key",
        default=os.getenv("LLM_GATEWAY_API_KEY", ""),
        help="Project API key (sk_...).",
    )
    parser.add_argument(
        "--model",
        default=os.getenv("DEMO_MODEL", "mock-fast"),
        help="Model name (mock-fast for local demo).",
    )
    parser.add_argument(
        "--prompt",
        default="Привет! Ответь одной строкой.",
        help="User prompt.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.api_key:
        print(
            "Error: api key is required. Use --api-key or LLM_GATEWAY_API_KEY env.",
            file=sys.stderr,
        )
        return 1

    url = args.gateway_url.rstrip("/") + "/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {args.api_key}",
        "Content-Type": "application/json",
    }
    payload = {
        "model": args.model,
        "messages": [{"role": "user", "content": args.prompt}],
    }

    response = requests.post(url, headers=headers, json=payload, timeout=20)
    text = response.text
    try:
        body = response.json()
    except ValueError:
        body = {"raw": text}

    print(f"HTTP {response.status_code}")
    print(json.dumps(body, ensure_ascii=False, indent=2))

    if not response.ok:
        return 1

    choices = body.get("choices") if isinstance(body, dict) else None
    if isinstance(choices, list) and choices:
        first = choices[0] if isinstance(choices[0], dict) else {}
        message = first.get("message", {}) if isinstance(first, dict) else {}
        content = message.get("content", "") if isinstance(message, dict) else ""
        if content:
            print("\nAssistant:")
            print(content)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
