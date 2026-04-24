#!/usr/bin/env python3
import argparse
import json
import os
import sys
from typing import Any

from openai import OpenAI


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Call local LLM Gateway via OpenAI Python SDK compatibility layer."
    )
    parser.add_argument(
        "--base-url",
        default=os.getenv("OPENAI_COMPAT_BASE_URL", "http://localhost:8080/v1"),
        help="OpenAI-compatible base URL.",
    )
    parser.add_argument(
        "--api-key",
        default=os.getenv("LLM_GATEWAY_API_KEY", ""),
        help="Project API key (sk_...).",
    )
    parser.add_argument(
        "--model",
        default=os.getenv("DEMO_MODEL", "mock-fast"),
        help="Model for chat.completions call.",
    )
    parser.add_argument(
        "--prompt",
        default="Проверка OpenAI SDK compatibility.",
        help="User prompt.",
    )
    return parser.parse_args()


def to_jsonable(value: Any) -> Any:
    if hasattr(value, "model_dump"):
        return value.model_dump()
    if hasattr(value, "to_dict"):
        return value.to_dict()
    return value


def main() -> int:
    args = parse_args()
    if not args.api_key:
        print(
            "Error: project API key is required. Set LLM_GATEWAY_API_KEY or --api-key.",
            file=sys.stderr,
        )
        return 1

    client = OpenAI(
        api_key=args.api_key,
        base_url=args.base_url.rstrip("/"),
    )

    try:
        response = client.chat.completions.create(
            model=args.model,
            messages=[{"role": "user", "content": args.prompt}],
        )
    except Exception as exc:
        print(f"Request failed: {exc}", file=sys.stderr)
        return 1

    payload = to_jsonable(response)
    print(json.dumps(payload, ensure_ascii=False, indent=2))

    text = ""
    choices = getattr(response, "choices", None)
    if choices:
        message = getattr(choices[0], "message", None)
        if message is not None:
            text = getattr(message, "content", "") or ""
    if text:
        print("\nAssistant:")
        print(text)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
