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
# Validate trusted baseline before importing any third-party source as data.
go test -race ./...
go vet ./...
go run ./cmd/upstream check
jq -Sc '.files | sort_by(.path)' upstream.lock.json > "$work/before.json"
gh repo clone cursor/plugins "$work/cursor" -- --depth=1
revision=$(git -C "$work/cursor" rev-parse HEAD)
[[ "$revision" =~ ^[0-9a-f]{40}$ ]]
go run ./cmd/upstream update --source "$work/cursor" --revision "$revision"
jq -Sc '.files | sort_by(.path)' upstream.lock.json > "$work/after.json"
if cmp -s "$work/before.json" "$work/after.json"; then
  git restore -- upstream.lock.json internal/bundle/manifest.json internal/bundle/review-report.json internal/bundle/_upstream
  echo 'Cursor pstack paths, bytes and modes are unchanged. No update PR.' >> "$GITHUB_STEP_SUMMARY"
  exit 0
fi
# An update must never silently carry compatibility approval forward.
if go run ./cmd/upstream check > "$work/check.log" 2>&1; then
  echo 'Changed source unexpectedly passed compatibility approval' >&2
  exit 1
fi
cat "$work/check.log"
if ! rg -q 'upstream compatibility review pending' "$work/check.log"; then
  echo 'Candidate failed for a reason other than pending compatibility' >&2
  exit 1
fi
git diff --exit-code -- internal/bundle/compatibility.json
go vet ./...
go build -trimpath -o "$work/pstack-rex" ./cmd/pstack-rex
sha256sum "$work/after.json" > "$work/digest"
read -r digest _ < "$work/digest"
branch="automation/cursor-pstack-$digest"
body="$work/body.md"
{
  printf '## Why\n\nKeep the imported Cursor pstack source current for maintainer review.\n\n'
  printf 'Source commit: https://github.com/cursor/plugins/commit/%s\n\n' "$revision"
  printf 'Canonical path/byte/mode inventory SHA-256: %s. Unrelated Cursor commits do not create proposals.\n\n' "$digest"
  printf '## Source changes and role inventory\n\n```json\n'
  jq '{revision, changed_files, changed_delegation_files, new_terms, role_candidates, new_role_candidates, limit}' internal/bundle/review-report.json
  printf '\n```\n\nThe PR Files changed tab contains the complete source diff. The lexical inventory is a review aid, not semantic verification.\n\n'
  printf '## Validation and review\n\n'
  printf 'Baseline Go race tests, vet and accepted inventory passed before import. Candidate vet and build passed. Candidate upstream check intentionally reports compatibility review pending. Candidate tests and installer acceptance remain unverified until review.\n\n'
  printf 'PRs opened using GITHUB_TOKEN do not automatically trigger ordinary push/pull_request CI. The updater validation above is not a substitute for candidate CI. After review, push the compatibility acceptance commit with maintainer credentials, or manually dispatch CI on this branch.\n\n'
  printf '%s\n' '- [ ] Review complete source diff and changed/new/deleted files.' '- [ ] Review delegation terms and role inventory, including runtime adapter compatibility.' '- [ ] Run upstream helper tests when relevant using their documented Bun/Node dependencies.' '- [ ] Explicitly accept compatibility only after review, and commit the baseline change.' '- [ ] Obtain passing Ubuntu and macOS CI on the reviewed candidate.'
} > "$body"
existing=$(gh pr list --repo "$GITHUB_REPOSITORY" --head "$branch" --state open --json number --jq '.[0].number // empty')
if [[ -n "$existing" ]]; then
  draft=$(gh pr view "$existing" --repo "$GITHUB_REPOSITORY" --json isDraft --jq '.isDraft')
  if [[ "$draft" != true ]]; then
    echo 'Existing proposal has left draft review; refusing to change it' >&2
    exit 1
  fi
  gh pr edit "$existing" --repo "$GITHUB_REPOSITORY" --body-file "$body"
else
  git switch -c "$branch"
  git add upstream.lock.json internal/bundle/manifest.json internal/bundle/review-report.json internal/bundle/_upstream
  gitleaks git --staged --redact
  git -c user.name='github-actions[bot]' -c user.email='41898282+github-actions[bot]@users.noreply.github.com' commit -m 'Propose Cursor pstack source update'
  git push origin "$branch"
  gh pr create --repo "$GITHUB_REPOSITORY" --head "$branch" --draft --title 'Review Cursor pstack update' --body-file "$body"
fi
printf 'Draft update proposed for Cursor revision %s. Compatibility approval remains pending.\n' "$revision" >> "$GITHUB_STEP_SUMMARY"
