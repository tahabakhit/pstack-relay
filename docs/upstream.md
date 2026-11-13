# Upstream maintenance

## Source and generated files

`upstream/` contains Cursor's pristine pstack source. `upstream/REVISION` records the imported commit. The installed `bin/pstack-relay version` prints the runtime name and that upstream revision.

The plugin generator combines upstream skills and agents with `overlay/delegation.md`, the POSIX runtime, the pristine setup template, the Claude session hook, and harness manifests. Generated entrypoints add a delegation reference pointer. Source-hash-checked adaptations in [`internal/plugin/adaptations.json`](../internal/plugin/adaptations.json) replace Cursor model setup and the agent's fallback bootstrap path. The original files in `upstream/` remain unchanged.

## Sync locally

From the repository root, run:

```zsh
go run ./cmd/upstream sync
go run ./cmd/build-plugin
go test ./...
go vet ./...
go run ./cmd/build-plugin --check
```

`sync` clones `cursor/plugins` into a temporary directory outside the checkout and replaces the upstream snapshot when its revision changes. It defaults to the `main` branch. `--repo <URL>` and `--ref <branch-or-tag>` select another source. `sync --check` validates the recorded revision format without network access. It does not compare the snapshot to remote bytes.

Review the upstream and generated diffs before committing. If a source-hash check fails, review the changed source and update the relevant adaptation deliberately. Read changed helpers' dependency metadata and run their declared checks when applicable. Go tests do not prove every upstream workflow's semantics.

Keep third-party checkouts outside this repository. Preserve the upstream license and attribution in [NOTICE](../NOTICE).

## Scheduled sync

[The workflow](../.github/workflows/upstream.yml) runs daily at 07:23 UTC or by manual dispatch on the default branch. [Its script](../scripts/check-upstream-update.sh) syncs upstream, builds the plugin, runs Go tests and vet, scans staged changes, and opens a pull request for a changed revision.

When those validation steps pass, the script squash-merges the pull request and deletes its branch. The script merges directly rather than arming repository auto-merge. When validation fails, it labels the pull request `upstream-sync-failed` and leaves it open. The first-release local run with a stubbed `gh` reached the merge step. No current hosted run is recorded in the acceptance evidence. See [status](status.md).

## Follow upstream updates without a version pin

The Claude Code and Codex manifests omit `version`. The sync rebuilds the plugin without a manual version bump. Claude Code's [manifest reference](https://code.claude.com/docs/en/plugins/manifest-reference#version) describes how setting a version pins updates.

Codex CLI 0.167.2 accepts a versionless repository marketplace plugin and labels its cache version `local`. In an isolated Git marketplace check, `codex plugin marketplace upgrade pstack-relay` refreshed the installed contents after the fixture repository changed. This is CLI marketplace evidence, not validation for OpenAI's public plugin directory. The [OpenAI submission requirements](https://developers.openai.com/plugins/deploy/submission#manifest-fields) require a version for Codex submissions. See [the acceptance record](acceptance.md) for the tested scope.
