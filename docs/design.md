# Delegation design for pstack on Orca and Rex

Status is proposed, 13 November 2026. This document replaces architecture and the 1.0 release contract. The code still follows those documents until the plan below lands.

## Goal

You run Cursor's pstack from Codex or Claude Code. When pstack delegates, each role runs on the harness, model, and effort you assigned to it. A role runs as the parent's own native subagent, as an Orca worker, or as a Rex worker. pstack itself behaves exactly as upstream wrote it. This project changes how pstack delegates and nothing else.

The repository supports several backends. Orca and Rex come first. herdr may follow later.

## What changed from pstack-rex 0.1

The first implementation treated every worker as untrusted code. It stripped worker configuration, ran Claude and Pi with read-only tools, and blocked 1.0 on sandbox research. No user asked for sandboxing, and upstream pstack runs its subagents as normal, full agents. Each rule in the table was reviewed with the owner on 13 November 2026.

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
| The parent reaches pstack-rex through an MCP server | The parent runs a CLI. On Orca, that CLI is `orca`. |

Upstream's subagent rules apply to every backend unchanged. Every worker runs in the background. Every brief passes file pointers, not inlined context. Every worker gets an explicit model and effort per role. New work goes to a fresh worker. A follow-up message to an existing worker is allowed only when the work needs state that lives in that worker, as poteto-mode's Subagents section says. A worker that reports done has not been verified, so the parent reviews the work. A required independent review stays blocked when no reviewer can run.

## Reference implementations

Behaviour follows Cursor's pstack in `cursor/plugins`. Where Cursor's text names a Cursor tool or model, the mapping follows `michael-denyer/pstack-claude`, which ports the same upstream to Claude Code and Codex. Its `skills/poteto-mode/references/codex-tools.md` maps `Task` to `spawn_agent` and `wait_agent`. It ships a `SessionStart` hook for Claude Code and Codex. It reads role models from a sheet in the `role: model @effort` form. This project reuses those conventions and adds backends.

## How delegation works

### The role sheet

The role sheet is pstack-claude's model sheet with one addition. A value may name a harness before the model. The sheet never names a backend. One sheet serves every parent harness.

```text
feature, refactoring: sonnet
interrogate reviewers: claude:opus @xhigh, codex:gpt-6.1-sol @high
how explorer: pi:openai/gpt-6.1-terra @medium
default effort: session
```

These values show the syntax. They are not defaults. Every other line, including `inherit-parent`, `auto`, `default effort`, and `session hook`, keeps pstack-claude's meaning.

The parent picks how to run each seat with these rules:

1. A value without a harness runs as a native subagent on the parent's harness. If the parent cannot run that model, upstream's fallback order applies.
2. A value whose harness is the parent's harness runs as a native subagent.
3. A value that names another harness starts a worker of that harness on the detected backend.
4. If no backend is detected, the seat follows upstream's fallback order as a native subagent and reports the model that ran.

From Claude Code, `claude:opus` runs native and `codex:gpt-6.1-sol` starts a Codex worker. From Codex, the reverse happens. The sheet stays the same.

### Backend detection

The parent detects the backend from its own environment. No setting chooses it.

| Backend | Detected when |
| --- | --- |
| Orca | `ORCA_TERMINAL_HANDLE` is set |
| Rex | `REX_SESSION` and `REX_BLOCK` are set |
| herdr | To be decided after herdr is examined |
| None | None of the variables above is set |

If more than one is set, the first row that matches wins.

### Setup across harnesses

One installer serves every harness. It finds `codex`, `claude`, and `pi` on `PATH`. It installs the plugin into each harness it finds. It collects the models each harness offers. Codex lists them with `codex debug models --bundled`. Pi lists them with `pi --offline --list-models`. Claude Code has no list command, so setup offers its aliases `opus`, `sonnet`, `haiku`, and `fable`. You assign roles from that combined list. Setup writes the harness before a model that you pick from a harness's list. It writes one sheet at one path, and every harness's plugin reads that path.

### The delegation reference

One file, `delegation.md`, tells the parent how to carry out a pstack subagent call on this host. It maps Cursor's `Task` call and Cursor model slugs to the parent's own tools. It also gives the commands for each backend. A session-start hook in each plugin tells the parent to read it. Upstream files stay unmodified.

### Backends

**Native.** The parent uses its own subagent tool. That tool is `Agent` in Claude Code and `spawn_agent` in Codex.

**Orca.** The parent runs one command per seat. It passes `--worktree current` when the worker shares the parent's checkout, and `--worktree new-child` when the calling skill gives the worker its own worktree.

```sh
orca orchestration worker-start --agent claude --model opus --effort high \
	--worktree current --spec-file brief.md
```

The parent then ends its turn. Orca wakes it when the worker sends `worker_done`. `orca orchestration worker-read` returns the transcript. Stop, abandon, and retry follow Orca's orchestration guide.

**Rex.** Rex runs terminals and has no concept of an agent. A small launcher supplies the missing parts. It creates a worktree when the calling skill asks for one, then starts the harness's interactive session in a Rex block with the brief. The worker runs `pstack-deleg report` when it finishes. That command writes the result file and sets Rex program status. The parent waits on that report, not on process exit.

### Upstream sync

A scheduled CI job copies the latest `pstack/` from `cursor/plugins` into `upstream/`. It runs the tests and merges automatically when they pass. A failing sync opens a pull request instead. Our files live outside `upstream/`, so an upstream change never conflicts with them.

### Packaging

Each harness installs the project as a plugin through its own plugin system. Codex uses `codex plugin marketplace add` and `codex plugin add`. Claude Code uses its plugin marketplace. A build script combines `upstream/` with our files for each harness. Disable other pstack installations before you install this one, because two `poteto-mode` skills compete for the same requests.

## 1.0 acceptance

1. One setup run installs the plugin in Codex and in Claude Code, discovers their models, and writes one shared sheet.
2. With the same sheet, an `interrogate` run inside Orca sends one seat to `claude:opus` and one to a Codex model. From a Claude Code parent, the Claude seat runs native and the Codex seat runs as an Orca worker. From a Codex parent, the reverse happens. The parent reviews both results.
3. Inside Rex, a seat on another harness runs as an interactive Rex worker and returns its report.
4. Outside any backend, a seat on another harness follows upstream's fallback order and reports the model that ran.
5. Workers share the parent's checkout or get their own worktree exactly where the upstream skill says so.
6. The upstream sync job brings in a new upstream commit and merges it when the tests pass.

Pi as a parent, Pi workers, and herdr come after 1.0.

## Plan

Each step ends with a check that you can rerun.

1. **Prototype the unknowns.** Check that a Codex parent inside Orca can run `orca` and get woken. Check that an Orca worker may start its own workers. Check which commit `new-child` branches from. Install a minimal Codex plugin and confirm its `SessionStart` hook runs. pstack-claude's Codex plugin is the template. It declares `.codex-plugin/plugin.json` with a `skills` path and a `hooks/codex-hooks.json` that runs `session-start.sh codex` on startup, resume, clear, and compact. The result is a short note with the commands and their output.
2. **Delete what the decisions removed.** That covers the sandbox documents, issues, and prototypes, plus the validation evidence. It also covers the worker supervisor and receipts, the harness lockdown, the MCP server, and the byte lock. `go test ./...` passes on what remains.
3. **Add upstream sync.** Move the upstream copy to `upstream/` and add the scheduled job. Run it once against the current upstream.
4. **Add the shared sheet, `delegation.md`, backend detection, and the hook.** Build the Claude Code plugin first, because this machine runs Claude Code inside Orca. A native seat and an Orca seat each complete one task.
5. **Build the Codex plugin.** Repeat the step 4 checks from a Codex parent.
6. **Add the cross-harness installer.** One run installs both plugins, discovers models, and writes the shared sheet.
7. **Rebuild the Rex launcher.** Run one interactive Rex worker and return its report.
8. **Run the 1.0 acceptance list.** Then rewrite the README and release.

## Open questions

- Orca has no `pi` agent ID. Orca lists `omp`. Nobody has checked whether `omp` can stand in for Pi. pstack-claude also ships a `pi` port to compare against when Pi support starts.
- No one has looked at herdr yet. It is not installed on this machine.
- The project name. The repository may outgrow `pstack-rex`.
