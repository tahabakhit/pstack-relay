# Configuration reference

The runtime script ships in the installed plugin. In examples below, replace `<plugin root>` with the absolute directory that holds `delegation.md`. Run the script by path, without adding it to `PATH`.

## Shared sheet

The default path is `${XDG_CONFIG_HOME:-$HOME/.config}/pstack-relay/pstack-models.md`. `PSTACK_RELAY_SHEET` overrides it. The `setup`, `models`, `resolve`, and `doctor` commands also accept `--sheet <path>`.

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

`<plugin root>/bin/pstack-relay models` prints the sheet. `<plugin root>/bin/pstack-relay models --json` includes its path, parsed roles, and formatted text. `models --set` replaces a literal line, or creates it if absent:

```zsh
"<plugin root>/bin/pstack-relay" models --set 'interrogate reviewers=claude:opus @xhigh, codex:gpt-6.1-sol @high'
"<plugin root>/bin/pstack-relay" models --set 'default effort=session'
"<plugin root>/bin/pstack-relay" models --set 'session hook=off'
```

Use the full literal name with `--set` to replace a grouped role line. Component lookup applies to resolution, not updates. Setup preserves an existing sheet unless `--force` is given. Changes through `models --set` do not reinstall plugins.

## Environment variables

| Variable | Meaning |
| --- | --- |
| `PSTACK_RELAY_SHEET` | Overrides the shared sheet path for the CLI and Claude hook. |
| `XDG_CONFIG_HOME` | Supplies the default configuration directory, otherwise `$HOME/.config`. |
| `PSTACK_RELAY_STATE` | Overrides the worker state base. Rex and hc use separate subdirectories. Parent and worker must use the same root. |
| `TMPDIR` | Supplies the default state base, otherwise `/tmp`. Workers use `<base>/pstack-relay-<uid>/rex` or `<base>/pstack-relay-<uid>/hc`. State need not survive a reboot. |
| `ORCA_TERMINAL_HANDLE` | Detects Orca when nonempty, ahead of Rex and hc. |
| `REX_SESSION`, `REX_BLOCK` | Detect Rex when both are nonempty and Orca is not detected. |
| `HC_TERMINAL_ID` | Detects hc when nonempty and neither Orca nor Rex was detected. |
| `CLAUDECODE` | Detects a Claude parent when equal to `1`. |
| `CODEX_THREAD_ID` | Detects a Codex parent when nonempty and Claude was not detected. |
| `PI_CODING_AGENT`, `PI_SESSION` | Detect a Pi parent after Claude and Codex, when the first equals `1` or the second is nonempty. Pi parent acceptance remains pending. |

## Resolver modes

`<plugin root>/bin/pstack-relay resolve --role '<role>'` prints a JSON array of seats. It does not start workers. `--parent claude|codex|pi` and `--backend orca|rex|hc|none` override detection for inspection.

| Mode | Condition | Action |
| --- | --- | --- |
| `native` | Bare model, parent's harness, or parent-model alias | Use the parent's native subagent tool. |
| `orca` | Another harness, Orca detected, and target is not Pi | Start the returned Orca command with the brief file. |
| `rex` | Another harness and Rex detected | Start the returned `<plugin root>/bin/pstack-relay rex start` command with the brief file. |
| `hc` | Another harness and hc detected | Run the returned script command with the brief file. |
| `fallback` | Another harness without a backend, or Pi on Orca | Run native on the parent's model and report the substitution. |

An undetected parent also produces `native`, with a diagnostic note. Resolve the missing parent before treating that placement as a cross-harness decision.

`--own-worktree` requests isolation when the upstream skill requires it. `--worktree-own` is an alias. Orca commands then include `--worktree new-child --base-branch <current-branch>`. A detached checkout cannot supply that branch. Native worktrees are created by the parent as described in [the delegation contract](../overlay/delegation.md).

Rex commands accept `<plugin root>/bin/pstack-relay rex start --harness <harness> --model <model> --effort <effort> --brief <file>`, with optional `--own-worktree`. The worker returns text with `<plugin root>/bin/pstack-relay rex report --id <id> --file <file>`, or reads stdin when `--file -`. `<plugin root>/bin/pstack-relay rex wait --id <id> --seconds <n>` returns its state. Its default wait is 540 seconds, after which an unreported active worker remains `running`.

hc commands accept `"<plugin root>/bin/pstack-relay" hc start --harness <harness> --model <model> --brief <file>`, with optional `--effort` and `--own-worktree`. The worker receives instructions to report with `hc send` into the parent's terminal. Long reports can be written to the returned `state_dir`'s `result.md` and sent by absolute path.

Wait with `"<plugin root>/bin/pstack-relay" hc wait --id <id> --seconds <n>`, or let hc notes wake the parent. The script reports hc states and returns a saved result file or recent terminal output. A terminal becoming idle or exiting does not establish successful work. Verify the worker's report. Use `hc keys <name> ...` only to answer the worker's startup trust dialog. See [herdr-core commands](https://github.com/herdrdev/herdr-core/blob/master/docs/commands.md).
