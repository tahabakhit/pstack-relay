# Delegation design for pstack-relay

pstack-relay ships Cursor's pstack in Claude Code and Codex. It changes how delegation runs. One sheet assigns each role's harness, model, and effort. The parent resolves each seat to its native subagent tool, Orca, Rex, or herdr-core (`hc`). Live evidence and remaining acceptance work are recorded in [acceptance](acceptance.md).

## Decisions retained from the first release

The early implementation stripped worker configuration and restricted tools. The owner reviewed these decisions on 13 November 2026. The current design keeps the decisions below.

| Rule in 0.1 | Decision |
| --- | --- |
| Workers skip AGENTS.md, CLAUDE.md, skills, plugins, hooks, and user config | Workers start like a normal session of their harness. The brief gives them the path to `poteto-mode` `SKILL.md`, as upstream does. |
| Workers get about 15 environment variables | Workers get the full environment. |
| Workers get no MCP, so `why` and `reflect` are blocked | Workers load their configured MCP servers. |
| Only Codex can run commands or write files | Every harness has full tools. |
| Read-only workers run in the source checkout | Follow upstream. A worker shares the parent's checkout, the way a `Task` subagent inherits it. A worker gets its own worktree when the calling skill says so, such as `arena` candidates, `interrogate` adversaries, and parallel writers. |
| Workers are one-shot runs | Workers are interactive sessions you can watch and message, on Orca and on Rex. |
| Workers must not delegate | Workers may delegate, as upstream allows. |
| Native subagents are forbidden, and Codex `multi_agent` should be off | A seat runs native when its model belongs to the parent's harness. A seat on another harness runs on the backend detected from the environment. Codex needs `multi_agent` on for native subagents, as pstack-claude documents. |
| A rejected model blocks the step | Use upstream's fallback order, and report the model that ran. |
| A reported model mismatch fails the worker | Record the requested model and the served model. Never fail on a mismatch. |
| Writers need a clean checkout | Follow upstream. When a skill gives a worker its own worktree, copy the scope's uncommitted changes into it, as `interrogate` says. |
| A 10-minute limit, 60-second polling, at most 4 workers, unknown workers hold a slot forever | On Orca, use Orca's lifecycle. Add no timeout and no worker cap. |
| Upstream is pinned byte for byte, and each update needs an acceptance record | Follow upstream. A scheduled job syncs it and merges when tests pass. |
| Evidence files, decision logs, and negative controls for every milestone | Keep upstream's own process. Add nothing on top. |
| The parent reaches pstack-relay through an MCP server | The parent runs a CLI. On Orca, that CLI is `orca`. |

Upstream's subagent rules apply to every backend. Workers run in the background. Briefs pass file pointers and the installed `skills/poteto-mode/SKILL.md` path. New work goes to a fresh worker. A follow-up is allowed only when the work needs state that lives in that worker. Parents verify reported work. A required independent review stays blocked when no reviewer can run.

## Resolve one shared sheet

The sheet adds an optional harness prefix to upstream role assignments. It never chooses a backend.

```text
feature, refactoring: sonnet
interrogate reviewers: claude:opus @xhigh, codex:gpt-6.1-sol @high
how explorer: pi:openai/gpt-6.1-terra @medium
default effort: session
```

These lines illustrate syntax, not defaults or account access. A bare model or a model on the parent's harness runs native. A model on another harness uses the detected backend. Without a backend, the seat follows upstream fallback and reports the model used. `inherit-parent` and `auto` use the parent's model.

The default sheet is `~/.config/pstack-relay/pstack-models.md`. `PSTACK_RELAY_SHEET` overrides it. `models --set` updates assignments without reinstalling plugins. See [configuration](configuration.md).

## Detect the backend from the parent

The first matching row wins.

| Backend | Detected when |
| --- | --- |
| Orca | `ORCA_TERMINAL_HANDLE` is nonempty |
| Rex | `REX_SESSION` and `REX_BLOCK` are nonempty |
| hc | `HC_TERMINAL_ID` is nonempty |
| None | No backend matches |

The resolver reads the parent harness and sheet, then returns each seat's placement and command. Inspection flags can override detection. Resolution does not start a worker.

## Run the seat

`delegation.md` maps Cursor's tool and model names to the host. Every generated skill and agent entrypoint points to it. The script `bin/pstack-relay` sits beside that file and is run by absolute path.

**Native.** Claude Code uses `Agent`. Codex uses `spawn_agent` and waits for its result. Codex needs `multi_agent` enabled.

**Orca.** The resolver supplies `orca orchestration worker-start`. It uses `--worktree current`, or `--worktree new-child --base-branch <current-branch>` when the upstream skill requires isolation. The parent ends its turn after starting workers. Orca wakes it on `worker_done`. Recovery follows `orca skills get orchestration`.

**Rex.** The bundled script creates a worktree when requested, starts an interactive harness in a Rex block, and gives the worker a reporting command. The worker uses the script's `rex report` command to save its result. The parent uses `rex wait` to observe the report or exit. Process exit without a report does not establish success.

**hc.** [herdr-core](https://github.com/herdrdev/herdr-core) supplies interactive worker terminals. The script uses `hc run --json --bg` and records the returned terminal. The worker reports into the parent terminal with `hc send`. The parent uses the script's `hc wait --id <id>` or receives hc notes. The script reads a saved `result.md` or recent terminal output. The parent may answer the startup trust dialog with `hc keys <name> ...`. Other approvals need the user's normal terminal workflow. See [hc commands](https://github.com/herdrdev/herdr-core/blob/master/docs/commands.md).

Workers share the checkout unless the calling skill requires a worktree. Copy uncommitted changes into isolated worktrees as [delegation.md](../overlay/delegation.md#worktrees) describes. Workers retain normal harness settings, tools, and approval behavior.

## Install the runtime inside the plugin

Users install `pstack@pstack-relay` through their harness marketplace, then run `/setup-pstack` in Claude Code or `$setup-pstack` in Codex. The plugin keeps upstream's `pstack` name. The repository and marketplace use `pstack-relay`.

Setup runs the installed POSIX sh script. It discovers installed harnesses and model catalogs, creates the shared sheet, and installs into detected Claude Code and Codex harnesses. Existing sheets are preserved. A previous release's sheet is copied when the new sheet is absent. Pi models are discovered, but Pi plugin installation and parent acceptance remain pending.

Go builds repository tools and tests only. No user-facing Go executable is built or installed. `cmd/build-plugin` combines pristine `upstream/` with `overlay/` to produce `plugins/pstack/` and both marketplace manifests. It copies the runtime with mode `0755`, the awk support file, `UPSTREAM-REVISION`, and a pristine setup template for role discovery. Generated skill and agent entrypoints add delegation pointers. Exact source-hash-checked adaptations replace Cursor setup instructions and the agent bootstrap path.

Claude's hook lives in `claude-hooks/` and is declared only by the Claude manifest. Codex has no hooks declaration or conventional `hooks/` directory. Its entrypoint pointers supply the delegation instructions.

Both manifests omit a fixed version so repository marketplace updates can follow synced contents. The release identifier is kept in the changelog and release tag. See [upstream updates](upstream.md#follow-upstream-updates-without-a-version-pin) for the tested Codex behavior and public-directory distinction.

`scripts/package.sh` creates one development archive of the marketplace tree and verifies it after extraction. It produces no architecture-specific Go archives. The supported user installation route is the marketplace.

## Sync upstream without editing it

The daily workflow imports Cursor's latest pstack into `upstream/`, rebuilds the plugin, and runs tests and vet. It scans staged changes, opens a pull request, and squash-merges it when validation passes. A failed validation leaves a labeled pull request open. Source-hash drift blocks unreviewed adaptations. The imported source stays unchanged by the generator.

The [acceptance record](acceptance.md) distinguishes local fixture evidence from live workers and hosted workflow execution. Local checks do not prove every upstream helper or workflow.
