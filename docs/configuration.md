# Configuration reference

## Shared sheet

The default path is `${XDG_CONFIG_HOME:-$HOME/.config}/pstack-rex/pstack-models.md`. `PSTACK_REX_SHEET` overrides it. The `setup`, `models`, `resolve`, and `doctor` commands also accept `--sheet <path>`.

Each role line has this grammar:

```text
role[, role...]: [harness:]model [@effort][, [harness:]model [@effort]...]
```

Blank lines and lines beginning with `#` are ignored. Harness names are `claude`, `codex`, and `pi`. Model names contain no spaces, commas, or colons. A Pi provider/model name can contain `/`. Effort values are `low`, `medium`, `high`, `xhigh`, `max`, and `session`. Accepted syntax does not prove that a harness or account supports a model or effort.

```text
feature, refactoring: sonnet
interrogate reviewers: claude:opus @xhigh, codex:gpt-6.1-sol @high
reflect judgment, divergent, synthesizer: inherit-parent
how explorer: pi:openai/gpt-6.1-terra @medium
default effort: session
session hook: on
```

A comma-separated role name is one literal line. `reflect judgment, divergent, synthesizer` is preserved as a whole. Lookup accepts that full name or any component, such as `reflect judgment`, `divergent`, or `synthesizer`. An exact line name takes priority. Otherwise the first matching line wins.

A comma-separated value supplies seats. The calling skill decides how to use the list, including the distinction between a panel and a judge pool. `inherit-parent` and `auto` omit an explicit model and run native, even with a harness prefix. A missing role uses the skill's default model.

| Special key | Meaning |
| --- | --- |
| `default effort: <effort>` | Supplies the effort when a seat omits it. `session` omits an explicit effort. |
| `session hook: on` | Enables the Claude session reminder. This is the default. |
| `session hook: off` | Disables the Claude session reminder. Codex ships no hook. |

`pstack-rex models` prints the sheet. `pstack-rex models --json` includes its path, parsed roles, and formatted text. `models --set` replaces a literal line, or creates it if absent:

```zsh
pstack-rex models --set 'interrogate reviewers=claude:opus @xhigh, codex:gpt-6.1-sol @high'
pstack-rex models --set 'default effort=session'
pstack-rex models --set 'session hook=off'
```

Use the full literal name with `--set` to replace a grouped role line. Component lookup applies to resolution, not updates. Setup preserves an existing sheet unless `--force` is given.

## Environment variables

| Variable | Meaning |
| --- | --- |
| `PSTACK_REX_SHEET` | Overrides the shared sheet path for the CLI and Claude hook. |
| `XDG_CONFIG_HOME` | Supplies the default configuration directory, otherwise `$HOME/.config`. |
| `PSTACK_REX_STATE` | Overrides the Rex worker state root. Parent and worker must use the same root. |
| `TMPDIR` | Supplies the default Rex state base, otherwise `/tmp`. The root is `<base>/pstack-rex-<uid>/rex`. State need not survive a reboot. |
| `ORCA_TERMINAL_HANDLE` | Detects Orca when nonempty, ahead of Rex. |
| `REX_SESSION`, `REX_BLOCK` | Detect Rex when both are nonempty and Orca is not detected. |
| `CLAUDECODE` | Detects a Claude parent when equal to `1`. |
| `CODEX_THREAD_ID` | Detects a Codex parent when nonempty and Claude was not detected. |
| `PI_CODING_AGENT`, `PI_SESSION` | Detect a Pi parent after Claude and Codex, when the first equals `1` or the second is nonempty. Pi parent support is after 1.0. |

## Resolver modes

`pstack-rex resolve --role '<role>'` prints a JSON array of seats. It does not start workers. `--parent claude|codex|pi` and `--backend orca|rex|none` override detection for inspection.

| Mode | Condition | Action |
| --- | --- | --- |
| `native` | Bare model, parent's harness, or parent-model alias | Use the parent's native subagent tool. |
| `orca` | Another harness, Orca detected, and target is not Pi | Start the returned Orca command with the brief file. |
| `rex` | Another harness and Rex detected | Start the returned `pstack-rex rex start` command with the brief file. |
| `fallback` | Another harness without a backend, or Pi on Orca | Run native on the parent's model and report the substitution. |

An undetected parent also produces `native`, with a diagnostic note. Resolve the missing parent before treating that placement as a cross-harness decision.

`--own-worktree` requests isolation when the upstream skill requires it. `--worktree-own` is an alias. Orca commands then include `--worktree new-child --base-branch <current-branch>`. A detached checkout cannot supply that branch. Native worktrees are created by the parent as described in [the delegation contract](../overlay/delegation.md).

Rex commands accept `pstack-rex rex start --harness <harness> --model <model> --effort <effort> --brief <file>`, with optional `--own-worktree`. The worker returns text with `pstack-rex rex report --id <id> --file <file>`, or reads stdin when `--file -`. `pstack-rex rex wait --id <id> --seconds <n>` returns its state. Its default wait is 540 seconds, after which an unreported active worker remains `running`.
