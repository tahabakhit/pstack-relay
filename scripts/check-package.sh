#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
output=${1:-dist}
output=$(cd "$output" && pwd)
(
  cd "$output"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c SHA256SUMS
  else
    shasum -a 256 -c SHA256SUMS
  fi
)
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in
  x86_64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo 'Unsupported smoke-test architecture' >&2; exit 1 ;;
esac
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
tar -xzf "$output/pstack-rex_${os}_${arch}.tar.gz" -C "$stage"
binary="$stage/pstack-rex_${os}_${arch}/pstack-rex"
"$binary" version
# An empty PATH proves bundle installation does not require network or helper CLIs.
PATH='' "$binary" install --dest "$stage/installed"
test -f "$stage/installed/runtime-contract.md"
test -f "$stage/installed/LICENSE"
if PATH='' "$binary" install --dest "$stage/installed"; then
  echo 'Installation unexpectedly overwrote a populated directory' >&2
  exit 1
fi
