#!/usr/bin/env bash
set -euo pipefail

if ! command -v newman >/dev/null 2>&1; then
  echo "newman is not installed."
  echo "Install: npm i -g newman"
  exit 1
fi

newman run postman/LLM-Gateway-MVP.postman_collection.json \
  -e postman/LLM-Gateway-Local.postman_environment.json
