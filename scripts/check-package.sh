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
tar -xzf "$output/pstack-rex-plugin.tar.gz" -C "$stage"
for file in .agents/plugins/marketplace.json .claude-plugin/marketplace.json plugins/pstack-rex/delegation.md plugins/pstack-rex/claude-hooks/session-start.sh plugins/pstack-rex/skills/poteto-mode/SKILL.md plugins/pstack-rex/bin/pstack-relay plugins/pstack-rex/lib/relay.awk plugins/pstack-rex/UPSTREAM-REVISION; do
  cmp "$file" "$stage/$file"
done
test -x "$stage/plugins/pstack-rex/claude-hooks/session-start.sh"
test -x "$stage/plugins/pstack-rex/bin/pstack-relay"
expected_revision=$(cat upstream/REVISION)
actual=$(cd "$stage" && sh "$stage/plugins/pstack-rex/bin/pstack-relay" version)
[[ "$actual" == "pstack-relay 2.0.0"$'\n'"upstream $expected_revision" ]]
printf '%s\n' "$actual"
