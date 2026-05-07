#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "Usage: $0 <tasks/.../<task-file>.md> [feat|fix]" >&2
  exit 1
fi

task_file="$1"
kind="${2:-feat}"

if [[ "$kind" != "feat" && "$kind" != "fix" ]]; then
  echo "ERROR: second argument must be 'feat' or 'fix'" >&2
  exit 1
fi

if [[ ! -f "$task_file" ]]; then
  echo "ERROR: task file not found: $task_file" >&2
  exit 1
fi

if [[ "$task_file" != tasks/* ]]; then
  echo "ERROR: task file must be inside tasks/" >&2
  exit 1
fi

base="$(basename "$task_file")"
if [[ ! "$base" =~ ^([0-9]{2}-[0-9]{2})-(.+)\.md$ ]]; then
  echo "ERROR: task filename must match '<NN-NN>-<slug>.md'" >&2
  exit 1
fi

task_id="${BASH_REMATCH[1]}"
raw_slug="${BASH_REMATCH[2]}"
slug="$(echo "$raw_slug" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//')"
branch="${kind}/${task_id}-${slug}"

git checkout main
git pull --ff-only
git checkout -b "$branch"

echo "Created branch: $branch"
echo "Use commit format: ${task_id}: <message>"
