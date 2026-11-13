# Troubleshooting

The runtime is the installed plugin's `bin/pstack-relay`. In commands below, replace `<plugin root>` with the absolute directory that holds `delegation.md`.

## A skill name clashes

Use `/poteto-mode`, `/setup-pstack`, and `/interrogate` in Claude Code, or `$name` in Codex. If another command owns a bare name in Claude Code, use the plugin prefix, for example `/pstack:interrogate`. See [how Claude Code names skills](https://code.claude.com/docs/en/skills#how-a-skill-gets-its-command-name).

Uninstall pstack-claude's `pstack` plugin before installing this plugin, because they share the plugin name. The [migration steps](../README.md#migrate-from-pstack-rex-10) include removal commands.

## Codex asks to review hooks

The current Codex plugin ships no hook or `hooks/` directory. The Claude hook lives under `claude-hooks/`. A stale cached install can still contain an older hook. Refresh the marketplace and reinstall the plugin.

```zsh
codex plugin marketplace upgrade pstack-relay
codex plugin remove pstack@pstack-relay
codex plugin add pstack@pstack-relay
```

Restart the harness after reinstalling.

## A worker waits for approval

Orca, Rex, and hc workers use the harness's normal approval settings. Open the worker terminal, inspect the requested action, and answer there. A waiting approval is not a completed result. For hc, the parent may use `hc keys <name> ...` only for the startup trust dialog.

## A skill refuses the Skill tool

`Skill ... cannot be used with Skill tool due to disable-model-invocation` means upstream reserves that skill for user invocation. Type `/interrogate` yourself in Claude Code, or `$interrogate` in Codex. Asking the agent to call the Skill tool again does not change that restriction.

## An Orca child starts from the wrong branch

Use `"<plugin root>/bin/pstack-relay" resolve --role '<role>' --own-worktree` and follow its returned command. The Orca command includes the current branch as `--base-branch`. A detached checkout cannot supply that branch.

Uncommitted changes need copying to isolated worktrees on every backend. Follow [the delegation worktree steps](../overlay/delegation.md#worktrees).

## A worker never reports

For Orca, follow `orca skills get orchestration`. Check the terminal for startup or approval prompts. Orca wakes the parent when the worker reports completion.

For Rex, run `"<plugin root>/bin/pstack-relay" rex wait --id <id>`. `running` means no report or exit was observed before the wait ended. `exited` means the process exited without a report. `reported` means a result file exists. The parent must verify the report.

For hc, run `"<plugin root>/bin/pstack-relay" hc wait --id <id>`. Inspect the returned state and terminal output. The worker reports into the parent terminal with `hc send`. Use the saved `result.md` when the worker supplies one.

Confirm that the worker received its full brief and reporting instructions. Parent and worker must agree on `PSTACK_RELAY_STATE` and the temporary directory. Do not delete active state or launch a duplicate because a wait returned `running`.

## Inspect doctor output

```zsh
"<plugin root>/bin/pstack-relay" doctor
"<plugin root>/bin/pstack-relay" doctor --json
```

Doctor reports the runtime and upstream revision, sheet parsing, detected parent and backend, harness model catalogs, and backend connectivity. A failed required check produces a nonzero exit. Missing optional harnesses and inactive backends are informational.

An environment failure outside a harness means no parent was detected. A model catalog is discovery evidence, not proof of account access. Doctor does not run a model task or certify plugin installation or independent review.

Inspect `"<plugin root>/bin/pstack-relay" models` and `"<plugin root>/bin/pstack-relay" resolve --role '<role>'` for unexpected placement. A `fallback` seat names the reason. Pi on Orca falls back because Orca has no Pi agent.
