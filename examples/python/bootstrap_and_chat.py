#!/usr/bin/env python3
import argparse
import json
import os
import random
import string
import sys
import time
from typing import Any, Dict, Tuple

import requests


def random_email(prefix: str) -> str:
    suffix = "".join(random.choices(string.ascii_lowercase + string.digits, k=8))
    return f"{prefix}-{int(time.time())}-{suffix}@example.local"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Bootstrap org/project/key and send first chat request."
    )
    parser.add_argument(
        "--auth-url",
        default=os.getenv("AUTH_URL", "http://localhost:8081"),
        help="Auth service URL.",
    )
    parser.add_argument(
        "--project-url",
        default=os.getenv("PROJECT_URL", "http://localhost:8082"),
        help="Project service URL.",
    )
    parser.add_argument(
        "--gateway-url",
        default=os.getenv("GATEWAY_URL", "http://localhost:8080"),
        help="Gateway service URL.",
    )
    parser.add_argument(
        "--org-name",
        default=os.getenv("DEMO_ORG_NAME", "Python Demo Org"),
        help="Organization name.",
    )
    parser.add_argument(
        "--email",
        default=os.getenv("DEMO_EMAIL", ""),
        help="User email for register/login. Random when empty.",
    )
    parser.add_argument(
        "--password",
        default=os.getenv("DEMO_PASSWORD", "password123"),
        help="User password.",
    )
    parser.add_argument(
        "--project-name",
        default=os.getenv("DEMO_PROJECT_NAME", "Python Demo Project"),
        help="Project name.",
    )
    parser.add_argument(
        "--model",
        default=os.getenv("DEMO_MODEL", "mock-fast"),
        help="Model name.",
    )
    parser.add_argument(
        "--prompt",
        default="Скажи, что интеграция работает.",
        help="User prompt.",
    )
    return parser.parse_args()


def request_json(
    session: requests.Session,
    method: str,
    url: str,
    token: str = "",
    payload: Dict[str, Any] | None = None,
) -> Tuple[int, Dict[str, Any], str]:
    headers: Dict[str, str] = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    response = session.request(
        method=method,
        url=url,
        json=payload,
        headers=headers,
        timeout=20,
    )
    text = response.text
    try:
        body = response.json()
        if not isinstance(body, dict):
            body = {"data": body}
    except ValueError:
        body = {"raw": text}
    return response.status_code, body, text


def required_str(data: Dict[str, Any], key: str) -> str:
    value = data.get(key)
    if not isinstance(value, str) or not value:
        raise ValueError(f"missing or invalid field: {key}")
    return value


def register_or_login(
    session: requests.Session,
    auth_url: str,
    org_name: str,
    email: str,
    password: str,
) -> str:
    register_payload = {"org_name": org_name, "email": email, "password": password}
    status, body, text = request_json(
        session, "POST", auth_url.rstrip("/") + "/auth/register", payload=register_payload
    )
    if status == 201:
        return required_str(body, "access_token")
    if status != 409:
        raise RuntimeError(f"register failed: {status} {text}")

    login_payload = {"email": email, "password": password}
    status, body, text = request_json(
        session, "POST", auth_url.rstrip("/") + "/auth/login", payload=login_payload
    )
    if status != 200:
        raise RuntimeError(f"login after conflict failed: {status} {text}")
    return required_str(body, "access_token")


def main() -> int:
    args = parse_args()
    email = args.email or random_email("python-demo")

    with requests.Session() as session:
        try:
            access_token = register_or_login(
                session=session,
                auth_url=args.auth_url,
                org_name=args.org_name,
                email=email,
                password=args.password,
            )

            status, project_body, text = request_json(
                session=session,
                method="POST",
                url=args.project_url.rstrip("/") + "/projects",
                token=access_token,
                payload={"name": args.project_name},
            )
            if status != 201:
                raise RuntimeError(f"create project failed: {status} {text}")
            project_id = required_str(project_body, "id")

            status, key_body, text = request_json(
                session=session,
                method="POST",
                url=args.project_url.rstrip("/") + f"/projects/{project_id}/keys",
                token=access_token,
            )
            if status != 201:
                raise RuntimeError(f"issue key failed: {status} {text}")
            api_key = required_str(key_body, "api_key")

            status, chat_body, text = request_json(
                session=session,
                method="POST",
                url=args.gateway_url.rstrip("/") + "/v1/chat/completions",
                token=api_key,
                payload={
                    "model": args.model,
                    "messages": [{"role": "user", "content": args.prompt}],
                },
            )
            if status != 200:
                raise RuntimeError(f"gateway chat failed: {status} {text}")

            print("Bootstrap completed successfully.\n")
            print(f"email={email}")
            print(f"project_id={project_id}")
            print(f"api_key={api_key}")
            print("\nGateway response:")
            print(json.dumps(chat_body, ensure_ascii=False, indent=2))
            print("\nUse this API key in your app:")
            print(f"LLM_GATEWAY_API_KEY={api_key}")
            return 0
        except Exception as exc:
            print(f"Error: {exc}", file=sys.stderr)
            return 1


if __name__ == "__main__":
    raise SystemExit(main())
