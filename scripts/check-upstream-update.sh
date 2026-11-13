#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
: "${RUNNER_TEMP:?RUNNER_TEMP is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GITHUB_STEP_SUMMARY:?GITHUB_STEP_SUMMARY is required}"
work=$(mktemp -d "$RUNNER_TEMP/pstack-update.XXXXXX")
trap 'rm -rf "$work"' EXIT
if [[ -n $(git status --porcelain) ]]; then
  echo 'Updater requires a clean checkout' >&2
  exit 1
fi
before=$(cat upstream/REVISION)
go run ./cmd/upstream sync
revision=$(cat upstream/REVISION)
if [[ "$before" == "$revision" ]]; then
  echo "Upstream unchanged at $revision" >> "$GITHUB_STEP_SUMMARY"
  exit 0
fi
validation=passed
if go run ./cmd/build-plugin > "$work/plugin.log" 2>&1; then
  cat "$work/plugin.log"
else
  cat "$work/plugin.log"
  validation=failed
fi
if go test ./... > "$work/tests.log" 2>&1; then
  cat "$work/tests.log"
else
  cat "$work/tests.log"
  validation=failed
fi
if go vet ./... > "$work/vet.log" 2>&1; then
  cat "$work/vet.log"
else
  cat "$work/vet.log"
  validation=failed
fi
branch="automation/cursor-pstack-$revision"
body="$work/body.md"
printf 'Sync Cursor pstack from %s to %s.\n\nSource: https://github.com/cursor/plugins/commit/%s\n\nplugin build, go test ./... and go vet ./...: %s.\n' "$before" "$revision" "$revision" "$validation" > "$body"
existing=$(gh pr list --repo "$GITHUB_REPOSITORY" --head "$branch" --state all --json number --jq '.[0].number // empty')
if [[ -n "$existing" ]]; then
  echo "Sync already proposed in PR #$existing" >> "$GITHUB_STEP_SUMMARY"
  exit 0
fi
git switch -c "$branch"
git add -- upstream plugins/pstack-relay .claude-plugin/marketplace.json .agents/plugins/marketplace.json
gitleaks git --staged --no-banner
git -c user.name='github-actions[bot]' -c user.email='41898282+github-actions[bot]@users.noreply.github.com' commit -m "chore: sync Cursor pstack to $revision"
git push origin "$branch"
pr=$(gh pr create --repo "$GITHUB_REPOSITORY" --head "$branch" --title "chore: sync Cursor pstack to ${revision:0:12}" --body-file "$body")
if [[ "$validation" == passed ]]; then
  # The job already ran plugin build, go test and go vet. Auto-merge needs a
  # repository setting and branch protection this repository may not have.
  gh pr merge "$pr" --repo "$GITHUB_REPOSITORY" --squash --delete-branch
else
  gh label create upstream-sync-failed --repo "$GITHUB_REPOSITORY" --color B60205 --description 'Upstream sync failed Go validation' --force
  gh pr edit "$pr" --repo "$GITHUB_REPOSITORY" --add-label upstream-sync-failed
fi
printf 'Upstream %s: %s (validation %s).\n' "$revision" "$pr" "$validation" >> "$GITHUB_STEP_SUMMARY"
