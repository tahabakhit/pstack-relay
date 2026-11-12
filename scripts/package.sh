#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
output=${1:-dist}
mkdir -p "$output"
output=$(cd "$output" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
go run ./cmd/upstream check
for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  os=${target%/*}
  arch=${target#*/}
  name="pstack-rex_${os}_${arch}"
  mkdir -p "$stage/$name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -o "$stage/$name/pstack-rex" ./cmd/pstack-rex
  cp LICENSE NOTICE "$stage/$name/"
  cp internal/bundle/upstream/LICENSE "$stage/$name/LICENSE.upstream"
  for doc in README.md docs/architecture.md docs/contracts.md; do
    if [[ -f "$doc" && ! -L "$doc" ]]; then
      mkdir -p "$stage/$name/$(dirname "$doc")"
      cp "$doc" "$stage/$name/$doc"
    fi
  done
  tar -czf "$output/$name.tar.gz" -C "$stage" "$name"
done
(
  cd "$output"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum pstack-rex_{darwin,linux}_{amd64,arm64}.tar.gz > SHA256SUMS
  else
    shasum -a 256 pstack-rex_{darwin,linux}_{amd64,arm64}.tar.gz > SHA256SUMS
  fi
)
bash scripts/check-package.sh "$output"
