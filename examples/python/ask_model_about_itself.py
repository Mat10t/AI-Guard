#!/usr/bin/env python3
import argparse
import os
import sys

from openai import OpenAI


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Ask model to describe itself via local LLM Gateway (OpenAI-compatible API)."
    )
    parser.add_argument(
        "--base-url",
        default=os.getenv("OPENAI_COMPAT_BASE_URL", "http://localhost:8080/v1"),
        help="OpenAI-compatible base URL of your gateway.",
    )
    parser.add_argument(
        "--api-key",
        default=os.getenv("LLM_GATEWAY_API_KEY", ""),
        help="Project API key (sk_...).",
    )
    parser.add_argument(
        "--model",
        default=os.getenv("DEMO_MODEL", "mock-fast"),
        help="Model id from your gateway catalog.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.api_key:
        print("Error: set LLM_GATEWAY_API_KEY or pass --api-key", file=sys.stderr)
        return 1

    client = OpenAI(
        api_key=args.api_key,
        base_url=args.base_url.rstrip("/"),
    )

    try:
        response = client.chat.completions.create(
            model=args.model,
            messages=[
                {
                    "role": "user",
                    "content": "Расскажи о себе.Кто тебя создал. Очень коротко",
                }
            ],
        )
    except Exception as exc:
        print(f"Request failed: {exc}", file=sys.stderr)
        return 1

    content = (response.choices[0].message.content or "").strip()
    if not content:
        print("Model returned empty response.")
        return 0

    print(content)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
