#!/usr/bin/env bash
# Development artifact containing the same marketplace tree, with no runtime binary.
set -euo pipefail
cd "$(dirname "$0")/.."
output=${1:-dist}
mkdir -p "$output"
output=$(cd "$output" && pwd)
go run ./cmd/upstream sync --check
go run ./cmd/build-plugin --check
tar -czf "$output/pstack-relay-plugin.tar.gz" plugins .claude-plugin .agents LICENSE NOTICE
(
  cd "$output"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum pstack-relay-plugin.tar.gz > SHA256SUMS
  else
    shasum -a 256 pstack-relay-plugin.tar.gz > SHA256SUMS
  fi
)
bash scripts/check-package.sh "$output"
