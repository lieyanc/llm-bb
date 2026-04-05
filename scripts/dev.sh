#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export LLM_BB_ADDRESS="${LLM_BB_ADDRESS:-127.0.0.1:8080}"
export LLM_BB_DATABASE_PATH="${LLM_BB_DATABASE_PATH:-data/dev/llm-bb.db}"
export LLM_BB_SEED_DEMO="${LLM_BB_SEED_DEMO:-true}"

mkdir -p "$(dirname "$LLM_BB_DATABASE_PATH")"

printf 'starting llm-bb dev server\n'
printf '  address: %s\n' "$LLM_BB_ADDRESS"
printf '  database: %s\n' "$LLM_BB_DATABASE_PATH"

if [[ ! -d node_modules ]]; then
  printf 'installing frontend dependencies\n'
  npm ci
fi

printf 'building embedded frontend bundle\n'
npm run build:ui

exec go run ./cmd/llm-bb "$@"
