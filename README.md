# pstack-relay

pstack-relay ships Cursor's pstack workflows as a Claude Code and Codex plugin. One shared model sheet assigns each role to a native subagent, an Orca worker, a Rex worker, or a herdr-core (`hc`) worker.

## Requirements

- Claude Code, Codex, or both, installed and authenticated.
- POSIX sh, awk, and standard Unix tools. Go is needed only for repository tools and tests.
- An optional backend for workers on another harness.

Workers use the harness's normal configuration, tools, and approval settings.

## Install through a marketplace

For Claude Code, run these commands in your terminal.

```zsh
claude plugin marketplace add tahabakhit/pstack-relay
claude plugin install pstack@pstack-relay
```

Then type `/setup-pstack` in Claude Code.

For Codex, run these commands in your terminal.

```zsh
codex plugin marketplace add tahabakhit/pstack-relay
codex plugin add pstack@pstack-relay
```

Then type `$setup-pstack` in Codex.

The setup skill shows `bin/pstack-relay setup --dry-run`, asks you to confirm, then runs `setup --yes`. Setup discovers harnesses and models, writes the shared sheet if absent, and installs the plugin into the other detected harness. Existing sheets are preserved. Pi models are discovered, but Pi plugin installation is deferred.

The plugin keeps upstream's name, `pstack`. The repository, marketplace, and runtime script are named `pstack-relay`. The script ships inside the plugin. No separate CLI installation or Go build is needed.

Disable other pstack copies before using these skills. If you installed pstack-claude, uninstall its `pstack` plugin first so the plugin names do not clash. See [migration](#migrate-from-pstack-rex-10).

Codex native subagents need `multi_agent` enabled.

```zsh
codex features enable multi_agent
```

## Choose models and a backend

The default sheet is `~/.config/pstack-relay/pstack-models.md`. Run `/setup-pstack` in Claude Code or `$setup-pstack` in Codex to change assignments. These lines show the syntax and do not guarantee account access.

```text
feature, refactoring: sonnet
interrogate reviewers: claude:opus @xhigh, codex:gpt-6.1-sol @high
reflect judgment, divergent, synthesizer: inherit-parent
default effort: session
session hook: on
```

A bare model or a model on the parent's harness runs native. A model on another harness uses the detected backend. Without a backend, that seat falls back to the parent's model. `inherit-parent` and `auto` also use the parent's model.

| Backend | Detection | Worker completion |
| --- | --- | --- |
| Native | Bare model, same harness, or parent-model alias | Parent's subagent tool |
| Orca | `ORCA_TERMINAL_HANDLE` | Orca wakes the parent on `worker_done` |
| Rex | `REX_SESSION` and `REX_BLOCK` | Bundled script's `rex report` and `rex wait` |
| [herdr-core](https://github.com/herdrdev/herdr-core) (`hc`) | `HC_TERMINAL_ID` | Worker sends a report with `hc send`; parent uses `hc wait` through the bundled script or receives hc notes |

Detection checks Orca, then Rex, then hc. See the [configuration reference](docs/configuration.md) and herdr-core's [command reference](https://github.com/herdrdev/herdr-core/blob/master/docs/commands.md).

## Use the upstream skills

Type `/poteto-mode` or `/interrogate` in Claude Code, or `$poteto-mode` or `$interrogate` in Codex, followed by your request. Skills marked user-invocable must be typed by you. See the [upstream pstack README](upstream/README.md).

The Claude `SessionStart` hook reminds the agent to read `delegation.md`. Set `session hook: off` to disable the reminder. Codex ships no hook. Its skill entrypoints point to the delegation reference.

Upstream stays unchanged in `upstream/`. Generated entrypoints add delegation pointers. Setup and the agent bootstrap have source-hash-checked host adaptations. See [upstream maintenance](docs/upstream.md).

## Migrate from pstack-rex 1.0

Remove the old plugin and marketplace in each harness where they are installed.

```zsh
claude plugin uninstall pstack-rex@pstack-rex
claude plugin marketplace remove pstack-rex
codex plugin remove pstack-rex@pstack-rex
codex plugin marketplace remove pstack-rex
rm -f ~/.local/bin/pstack-rex
```

If pstack-claude is installed, remove its `pstack` plugin before installing the new plugin.

```zsh
claude plugin uninstall pstack@pstack-claude
codex plugin remove pstack@pstack-claude
```

Run only the removal commands for installed plugins. Remove any manually copied pstack skills from `~/.codex/skills` as well. Follow the marketplace installation steps above, then run the setup skill.

Setup automatically copies assignments from `${PSTACK_REX_SHEET:-${XDG_CONFIG_HOME:-$HOME/.config}/pstack-rex/pstack-models.md}` when the new sheet is absent. It keeps the old sheet. `PSTACK_RELAY_SHEET` now overrides the sheet path. Update any local configuration that used `PSTACK_REX_SHEET`.

## Update or uninstall

Refresh Claude Code with these commands, then restart the harness.

```zsh
claude plugin marketplace update pstack-relay
claude plugin update pstack@pstack-relay
```

For Codex, refresh the marketplace, then restart.

```zsh
codex plugin marketplace upgrade pstack-relay
```

To uninstall, run the command for each installed harness.

```zsh
claude plugin uninstall pstack@pstack-relay
codex plugin remove pstack@pstack-relay
```

The shared sheet remains available for later use.

## Reference and attribution

See [troubleshooting](docs/troubleshooting.md), [release status](docs/status.md), [acceptance evidence](docs/acceptance.md), [contributing](CONTRIBUTING.md), and the [changelog](CHANGELOG.md).

Cursor pstack is by Lauren Tan and is distributed under the MIT license. See [NOTICE](NOTICE), [the project license](LICENSE), and [the upstream license](upstream/LICENSE).
