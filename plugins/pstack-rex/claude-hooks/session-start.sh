#!/bin/sh
set -eu
plugin_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
sheet=${PSTACK_REX_SHEET:-${XDG_CONFIG_HOME:-$HOME/.config}/pstack-rex/pstack-models.md}
if [ -f "$sheet" ]; then
  enabled=$(awk '/^[[:space:]]*session hook[[:space:]]*:/ { sub(/^[^:]*:[[:space:]]*/, ""); sub(/[[:space:]]*$/, ""); value=$0 } END { print value }' "$sheet")
  if [ "$enabled" = off ]; then
    exit 0
  fi
fi
# Encode the installed path as JSON, including quotes and backslashes.
# Filesystem paths cannot contain NUL; awk handles newlines in the path too.
context=$(printf '%s' "pstack-rex is installed. Before following any pstack skill, read $plugin_root/delegation.md." | awk 'BEGIN { ORS=""; print "\"" } { if (NR>1) printf "\\n"; for (i=1;i<=length($0);i++) { c=substr($0,i,1); if (c=="\\") printf "\\\\"; else if (c=="\"") printf "\\\""; else if (c=="\t") printf "\\t"; else if (c=="\r") printf "\\r"; else printf "%s", c } } END { print "\"" }')
printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":%s}}\n' "$context"
