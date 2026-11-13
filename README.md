# pstack-rex

pstack-rex is a plugin for Codex and Claude Code that ships Cursor's pstack workflows and changes how delegation runs. A shared model sheet assigns each role to a native subagent, an Orca worker, or an interactive Rex worker.

## Requirements

- Go 1.23 or later to build the CLI.
- Codex, Claude Code, or both, installed and authenticated.
- Optional Orca or Rex for delegation to another harness.

Workers use the harness's normal configuration, tools, and approval settings. An Orca or Rex worker can stop at an approval prompt that you answer in its terminal.

## Install

Install the CLI and make sure its install directory is on `PATH`.

```zsh
go install github.com/tahabakhit/pstack-rex/cmd/pstack-rex@latest
pstack-rex setup
```

Setup installs the plugin in every detected Codex and Claude Code harness from the `tahabakhit/pstack-rex` marketplace. It discovers models and creates the shared sheet if absent, preserving an existing sheet. Pi models are discovered for the sheet, but Pi plugin installation is deferred until after 1.0.

To build from a source checkout, run these commands at its root.

```zsh
go build -o /tmp/pstack-rex ./cmd/pstack-rex
/tmp/pstack-rex setup --source "$PWD"
```

Put the built binary on `PATH` before using the skills. `--source <path>` selects a local marketplace checkout. Preview setup with `pstack-rex setup --dry-run`.

Disable other pstack installations so their skills do not compete. For pstack-claude in Claude Code, run:

```zsh
claude plugin disable pstack@pstack-claude
```

Remove old pstack skill copies from `~/.codex/skills`. Codex native subagents need `multi_agent` enabled:

```zsh
codex features enable multi_agent
```

## Choose models

The default sheet is `~/.config/pstack-rex/pstack-models.md`. Print it with `pstack-rex models`. These are syntax examples, not account access guarantees:

```text
feature, refactoring: sonnet
interrogate reviewers: claude:opus @xhigh, codex:gpt-6.1-sol @high
reflect judgment, divergent, synthesizer: inherit-parent
default effort: session
session hook: on
```

The harness prefix is optional. A bare model or a model on the parent's harness runs native. A model on another harness runs on the detected backend. Without a backend, that seat falls back to the parent's model. `inherit-parent` and `auto` also use the parent's model.

Orca is detected when `ORCA_TERMINAL_HANDLE` is set. Otherwise Rex is detected when both `REX_SESSION` and `REX_BLOCK` are set. See the [sheet and resolver reference](docs/configuration.md) for details.

## Use pstack

Use the skills as upstream describes in the [pstack README](upstream/README.md). Skills marked user-invocable must be typed by you. For example, type `/pstack-rex:interrogate` in Claude Code or `$interrogate` in Codex, followed by your request.

The Claude plugin's `SessionStart` hook adds a reminder to read `delegation.md` on startup, resume, clear, and compact. `session hook: off` disables that reminder. The Codex plugin ships no hook. Its skill entrypoints point to the delegation reference.

The upstream source stays unchanged in `upstream/`. Generated entrypoints add delegation pointers, and setup and agent bootstrap paths have explicit host adaptations. See [upstream maintenance](docs/upstream.md).

## Upgrade

Update the CLI with the same `go install` command, then run `pstack-rex setup` again. To refresh a cached Codex plugin, remove and add it:

```zsh
codex plugin remove pstack-rex@pstack-rex
codex plugin add pstack-rex@pstack-rex
```

For Claude Code, run `claude plugin update pstack-rex@pstack-rex`. Restart the harness after updating.

## Uninstall

Run the command for each installed harness:

```zsh
codex plugin remove pstack-rex@pstack-rex
claude plugin uninstall pstack-rex@pstack-rex
```

Remove the `pstack-rex` binary from its install directory. Delete the shared sheet only if you no longer want its assignments.

## Reference and attribution

See [troubleshooting](docs/troubleshooting.md), [1.0 status](docs/status.md), [contributing](CONTRIBUTING.md), and the [changelog](CHANGELOG.md).

Cursor pstack is by Lauren Tan and is distributed under the MIT license. See [NOTICE](NOTICE), [the project license](LICENSE), and [the upstream license](upstream/LICENSE).
