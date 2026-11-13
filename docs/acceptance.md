# Acceptance record

## 2.0 acceptance

The current candidate ships a versionless `pstack` plugin in the pstack-relay marketplace. Release acceptance is broader than local fixture checks.

- Marketplace-only install in both harnesses from the public repository: pending.
- Orca seat: pending.
- Rex seat: pending.
- hc seat: pending.

### Local marketplace and repository checks

Local checks on 13 November 2026, with upstream revision `6e359ce0aba9e927914d8a940510aa2ef66a8f23`:

- `go test -race ./...` passed for all packages. `go vet ./...` exited zero with no diagnostics.
- `go run ./cmd/build-plugin --check` printed `plugin tree is up to date`.
- `shellcheck -s sh overlay/bin/pstack-relay plugins/pstack/bin/pstack-relay overlay/hooks/session-start.sh` exited zero with no diagnostics.
- `claude plugin validate plugins/pstack` exited zero and printed `Validation passed with warnings`. The warnings were the intentional missing version and a missing listing icon.
- `bash scripts/package.sh <temporary-output>` produced a plugin-only archive. Checksum validation printed `pstack-relay-plugin.tar.gz: OK`. Its extracted runtime printed `pstack-relay` and the upstream revision.
- With an empty temporary `CODEX_HOME`, `codex plugin marketplace add <checkout>`, `codex plugin add pstack@pstack-relay --json`, and `codex plugin list --json` succeeded. The list showed the plugin installed and enabled with version `local`.
- With an empty temporary `CLAUDE_CONFIG_DIR`, `claude plugin marketplace add <checkout>`, `claude plugin install pstack@pstack-relay`, and `claude plugin list --json` succeeded. The plugin was enabled, with a cache identifier derived from the checkout commit. Orca's session-only inline agent-view plugin also appeared in that list.
- From outside the checkout, each installed copy's `bin/pstack-relay version` printed the runtime name and the recorded upstream revision. With a temporary sheet, `models --set 'smoke=codex:gpt-6.1-sol @high'` saved the assignment. `resolve --role smoke --parent codex --backend none` returned `mode: "native"`. Using `--parent claude --backend hc` returned `mode: "hc"` with the installed script's absolute path in the command.
- Each installed copy's `setup --dry-run --json --source <checkout>` succeeded under a fresh HOME and produced a starter sheet containing `interrogate reviewers`. No model task ran. Installed copies had executable runtime scripts, matching upstream revisions and pristine setup templates, and no `hooks/` directory.

Codex warned that helper PATH aliases are not created under `/tmp`; installation and script execution still succeeded. These runs used the local marketplace source. They do not prove public marketplace availability, live backend inference, or model access. The runtime was exercised through the available `sh`. Dash and macOS were unavailable locally. The test suite includes a dash lane when installed.

### Versionless Codex marketplace updates

Codex CLI 0.167.2 has no `plugin validate` command. An empty temporary `CODEX_HOME` accepted the versionless manifest through `codex plugin marketplace add <checkout>` and `codex plugin add pstack@pstack-relay`. `codex plugin list --json` reported `version: "local"` and an enabled plugin cached under `plugins/cache/pstack-relay/pstack/local`.

A disposable Git marketplace fixture contained the generated plugin without a version. Git's `insteadOf` configuration mapped a synthetic HTTPS source to that local repository. No hosted service was involved. After committing a marker change from `first` to `second`, `codex plugin marketplace upgrade pstack-relay --json` reported no errors. Reading the installed cache marker returned `second` immediately after upgrade. Removing and adding the plugin again also retained `second`. This establishes local CLI repository-marketplace behavior. It does not establish public-directory submission requirements or unattended update timing.

OpenAI's [submission manifest requirements](https://developers.openai.com/plugins/deploy/submission#manifest-fields) require `version` for Codex submissions. That is a different route from the tested repository marketplace. Claude Code's [version field reference](https://code.claude.com/docs/en/plugins/manifest-reference#version) describes why this repository omits the version pin.

## First-release history

The [unmodified first-release acceptance record](https://github.com/tahabakhit/pstack-relay/blob/c855e1f/docs/acceptance.md) preserves the exact commands, dispatch identifiers, failures, and fixes from 13 November 2026. Those runs used the retired Go runtime and do not prove current script or hc behavior.

That record contains successful setup into both harnesses, live Orca review seats from Claude Code and Codex parents, an interactive Rex worker report, resolver-only fallback output, worktree checks, and a local upstream-sync run with stubbed GitHub operations. It also records first-release remote branch installation and hosted Go checks on Linux and macOS. These are historical results, not current release claims.
