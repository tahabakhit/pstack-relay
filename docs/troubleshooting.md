# Troubleshooting

## Codex asks to review hooks

Older pstack-rex installs shipped a Codex hook that could block unattended startup at a hooks-need-review prompt. The 1.0 Codex plugin ships no hook. Refresh its cached installation:

```zsh
codex plugin remove pstack-rex@pstack-rex
codex plugin add pstack-rex@pstack-rex
```

Restart the session after reinstalling. The Claude hook lives under `claude-hooks/`, outside Codex's conventional hook directory.

## A worker waits for approval

Orca and Rex workers are interactive harness sessions. They use your normal approval settings and can pause at a prompt. Open the worker's terminal, read the requested action, and answer there. A waiting approval is not a completed worker result.

The [live Rex record](acceptance.md#item-3-rex-worker-2026-11-13) includes approval prompts answered in the worker terminal. The Rex report state now defaults to a temporary directory, but other harness actions can still need approval.

## A skill refuses the Skill tool

`Skill ... cannot be used with Skill tool due to disable-model-invocation` means upstream reserves that skill for user invocation. Type the command yourself, for example `/pstack-rex:interrogate` in Claude Code or `$interrogate` in Codex. Asking the agent to call the Skill tool again does not change that restriction.

## An Orca child starts from the wrong branch

Orca `--worktree new-child` starts from the repository base unless `--base-branch` is supplied. Use `pstack-rex resolve --role '<role>' --own-worktree` and follow its returned command. The resolver adds the current branch as `--base-branch`.

Uncommitted changes also need copying to an isolated worktree. Follow [the delegation contract's worktree steps](../overlay/delegation.md#worktrees).

## A worker never reports

For Orca, follow the lifecycle and recovery instructions from `orca skills get orchestration`. Check the worker terminal for a startup or approval prompt. Orca wakes the parent when the worker reports completion.

For Rex, run `pstack-rex rex wait --id <id>`. `running` means no report or exit was observed before the wait ended. `exited` means the process exited without a report. `reported` means a result file exists, which the parent must still verify. Process exit alone does not establish successful work.

Confirm that the worker received its full brief and the `rex report` instructions. Parent and worker must agree on `PSTACK_REX_STATE` and the temporary directory. Do not delete active state or start a duplicate worker merely because a wait returned `running`.

## Inspect doctor output

```zsh
pstack-rex doctor
pstack-rex doctor --json
```

Doctor reports the binary and upstream revision, sheet parsing, detected parent and backend, installed harness model catalogs, and backend connectivity. A failed required check produces a nonzero exit. Missing optional harnesses and inactive backends are informational.

An environment failure outside a harness means no parent was detected. A model catalog is discovery evidence, not proof of account access. Doctor does not run a model task, verify plugin installation, or certify an independent review.

If a seat runs on an unexpected harness, inspect `pstack-rex models` and `pstack-rex resolve --role '<role>'`. A `fallback` seat names the reason in its note. Pi on Orca falls back because Orca has no Pi agent.
