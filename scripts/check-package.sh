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
for doc in docs/design.md docs/configuration.md docs/troubleshooting.md docs/upstream.md docs/research.md CONTRIBUTING.md; do
  test -f "$stage/pstack-rex_${os}_${arch}/$doc"
done
for file in .agents/plugins/marketplace.json .claude-plugin/marketplace.json plugins/pstack-rex/delegation.md plugins/pstack-rex/claude-hooks/session-start.sh plugins/pstack-rex/skills/poteto-mode/SKILL.md; do
  cmp "$file" "$stage/pstack-rex_${os}_${arch}/$file"
done
test -x "$stage/pstack-rex_${os}_${arch}/plugins/pstack-rex/claude-hooks/session-start.sh"
expected_revision=$(cat upstream/REVISION)
actual=$(cd "$stage" && PATH='' "$binary" version)
[[ "$actual" == "pstack-rex 1.0.0"$'\n'"upstream $expected_revision" ]]
printf '%s\n' "$actual"
