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
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
tar -xzf "$output/pstack-relay-plugin.tar.gz" -C "$stage"
for file in .agents/plugins/marketplace.json .claude-plugin/marketplace.json plugins/pstack/delegation.md plugins/pstack/claude-hooks/session-start.sh plugins/pstack/skills/poteto-mode/SKILL.md plugins/pstack/bin/pstack-relay plugins/pstack/lib/relay.awk plugins/pstack/lib/setup-pstack.md plugins/pstack/UPSTREAM-REVISION; do
  cmp "$file" "$stage/$file"
done
test -x "$stage/plugins/pstack/claude-hooks/session-start.sh"
test -x "$stage/plugins/pstack/bin/pstack-relay"
test ! -e "$stage/plugins/pstack/hooks"
cmp upstream/skills/setup-pstack/SKILL.md "$stage/plugins/pstack/lib/setup-pstack.md"
expected_revision=$(cat upstream/REVISION)
actual=$(cd "$stage" && sh "$stage/plugins/pstack/bin/pstack-relay" version)
[[ "$actual" == "pstack-relay 2.0.0"$'\n'"upstream $expected_revision" ]]
printf '%s\n' "$actual"
