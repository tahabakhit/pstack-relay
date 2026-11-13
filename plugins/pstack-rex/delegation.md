# pstack delegation on this host

Read this file before you follow any pstack skill, playbook, or agent definition in this plugin. The skills are Cursor's upstream pstack, unchanged. They name Cursor tools and Cursor model slugs. This file says what those names mean here and how to start each subagent. Everything else in the skills applies exactly as written.

## Cursor names on this host

| Upstream names | Claude Code | Codex |
| --- | --- | --- |
| The `Task` tool | The `Agent` tool | `spawn_agent`, then `wait_agent` for the result |
| `subagent_type: generalPurpose` | `subagent_type: "general-purpose"` | A plain `spawn_agent` |
| `subagent_type: "poteto-agent"` | `subagent_type: "pstack-rex:poteto-agent"` | A plain `spawn_agent` whose brief says to read the `poteto-mode` `SKILL.md` in full first |
| `readonly: true` | Say in the brief that the subagent must not edit files | Same as Claude Code |
| `run_in_background: true` | `run_in_background: true` | `spawn_agent` already runs in the background |
| `AskQuestion` | `AskUserQuestion` | Ask in plain text |
| The todo list tools | `TaskCreate` and `TaskUpdate`, or `TodoWrite` | `update_plan` |
| `pstack-models.mdc` and the `/setup-pstack` rule | The shared model sheet. Run `pstack-rex models` to print it and `pstack-rex setup` to change it | Same as Claude Code |
| A Cursor model slug such as `claude-opus-5-5-xhigh` | The role's line in the shared model sheet | Same as Claude Code |
| A search under `~/.cursor/plugins/` for a skill | The skill in this plugin. Use the absolute path your parent gave you, or this plugin's `skills/` directory | Same as Claude Code |

On Codex, native subagents need `multi_agent` enabled in `~/.codex/config.toml`. A thread-limit error from `spawn_agent` is a capacity error, not a rejected model. Stop spawning until a finished agent is closed.

Cursor cloud agents, Cursor automations, and Cursor project stores do not exist here. When a skill requires one, say which step is blocked and continue the work that does not depend on it.

## Start every subagent through the resolver

Before each subagent or panel, run the resolver with the role name the skill uses:

```sh
pstack-rex resolve --role "interrogate reviewers"
```

It prints one JSON object per seat. Each seat has a `mode` and the exact values to use. Follow the seat exactly. Do not pick a model, harness, or backend yourself.

- `native` means use your own subagent tool from the table above, with the seat's `model` and `effort`.
- `orca` means start an Orca worker. Write the full brief to a file, then run the seat's `command` with that file. End your turn after you start every worker. Orca wakes you when a worker sends `worker_done`. Read the result with `orca orchestration check`, and the transcript with `orca orchestration worker-read --dispatch <id>`. For stop, retry, and abandon, follow `orca skills get orchestration`.
- `rex` means start a Rex worker with the seat's `command`. Wait for its report with `pstack-rex rex wait --id <id>`.
- `fallback` means the seat's harness cannot run here. Run it native with the seat's `model`, which follows upstream's fallback order, and tell the user which model ran instead of the configured one.

A brief for an Orca or Rex worker is a complete prompt. Give it the upstream brief, the absolute path of this plugin's `skills/poteto-mode/SKILL.md` when the skill asks for `poteto-agent`, the absolute paths of the files involved, and the evidence the worker must return.

## Worktrees

A subagent shares your checkout unless the calling skill gives it its own worktree. That is upstream's rule.

When the skill gives an Orca worker its own worktree, add `--worktree-own` to the resolver call. The seat's `command` then creates a child worktree from your current branch. Before you start the worker, save your uncommitted changes with `git diff HEAD --binary > <file>`. Tell the worker to run `git apply --index <file>` first, so it starts from what you see. Untracked files need `git add -N` before the diff.

## Upstream rules still apply

Every worker runs in the background. Every brief passes file pointers, not pasted context. New work goes to a fresh worker. Send a follow-up to an existing worker only when the work needs state that lives in that worker. A worker that reports done is not verified, so review its work yourself. A required independent review stays blocked when no reviewer can run.
