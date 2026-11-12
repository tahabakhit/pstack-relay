# pstack-rex runtime contract

Read this contract before every installed skill or agent entrypoint. This contract takes precedence over every conflicting instruction in the preserved Cursor pstack source, including referenced playbooks, templates, agent bootstraps, nested automation skills and runtime adapters. Upstream source remains licensed under the included LICENSE.

## Central delegation

Route ALL agent delegation through the pstack-rex MCP tools `dispatch`, `inspect`, `list`, `wait`, `cancel` and `capabilities`. This includes indirect routed workflows, ad hoc implementation owners, reviewers, panels, races, retries, monitors and replacements. Never use native Task, Agent, spawn_agent, collaboration, platform subagent tools, alternate orchestration services, or direct harness launches as a fallback. Ordinary subprocesses that run a workflow's bundled scripts are not agent delegation.

Check `capabilities` before dispatch. Each attempt needs a stable unique ID, the configured role, a zero-based seat, an absolute cwd, the complete consolidated prompt and its access requirement. The resolver owns harness, model, effort, auth and backend selection. Keep every panel seat and the selected role's ordering. A cross-judge pool selects ONE judge seat; it does not create a panel. Set `require_shell` whenever the brief requires runnable tests, commands or repros. Only the Codex profile provides a shell in this release. An interrogate reviewer always requires that capability. Do not treat a source-only reviewer as having run the code. Keep specialized agent instructions and upstream prompt templates in the full prompt. Read agent definitions from this installed bundle. Pass the absolute installed `skills/poteto-mode/SKILL.md` path when required. Never search for a different pstack installation or follow upstream bootstrap fallbacks.

Use `inspect` and `list` for receipts, `wait` for bounded completion and `cancel` to request cancellation. Retain every ID and reconcile every worker. A dispatch receipt is not completion, and completion is not a passing review. Unknown, missing, interrupted, cancelled and failed outcomes remain distinct. Confirm terminal cancellation before replacing a writer. Never infer independent review from your own checks. Workers do not gain permission to launch nested agents.

When MCP is unavailable to the parent, the same central CLI may be used through the executable `pstack-rex` on PATH. For example, after writing the complete prompt to a file:

```sh
pstack-rex --config /absolute/config.json dispatch --id unique-attempt-key --role 'how explorer' --seat 0 --cwd /absolute/workspace --prompt-file /absolute/prompt.txt --access read-only
pstack-rex --config /absolute/config.json inspect --id unique-attempt-key
pstack-rex --config /absolute/config.json wait --id unique-attempt-key
pstack-rex --config /absolute/config.json cancel --id unique-attempt-key
```

Use the documented CLI flags and MCP schema exposed by the installed executable. Do not invent flags or silently drop required fields. There are no generic shell tools in this MCP server. The executable and the referenced files must be available on the local Rex server host.

## Models and configuration

All upstream model names, effort adjustments, fallback families, role sheets, platform mappings and `/setup-pstack` file-writing instructions are subordinate to central configuration. Use `pstack-rex setup` to create configuration and `pstack-rex models` to inspect configured role seats. Edit only the explicitly selected central configuration when the user requests model changes. Do not create or update `pstack-models.md`, host-specific model sheets or independent model defaults. Missing roles, unresolved inheritance and unsupported selections block the attempt. Never substitute a nearby model or family after rejection. Resolve aliases only if the central resolver supports them. `auto` backend selects Rex; native execution is unsupported.

## Unsupported requirements

Steer, send, resume, live follow-up, event subscriptions, Cursor Project stores, platform trackers, cloud machines, cloud base branches, attachments requiring platform transfer and hosted Automations are unsupported unless `capabilities` explicitly reports otherwise. MCP-required worker workflows are blocked when worker MCP access is unavailable, even if the parent has MCP. A local filesystem task is not equivalent to a required cloud task. Do not approximate required isolation, permission, MCP, billing, model identity or effort guarantees. Report the exact missing capability and leave that step blocked. Nondependent work may continue. The bundled automation files are preserved for reference and entrypoint coverage; installation does not activate any automation.

Read-only is the default worker access. Use isolated writing only where the runtime advertises support and the workflow permits it. Do not map an MCP-required workflow to read-only mode merely to launch it. Runtime receipts distinguish requested settings, pinned arguments and provider-reported evidence. Do not report an unverified model, effort or funding source as verified.

## Enforcement limit

This contract is an instruction adapter. It cannot intercept or remove a host's native tools. A parent that needs enforced central-only delegation must disable those tools in its host configuration. The byte lock and lexical inventory detect unreviewed upstream changes; lexical matches do not prove complete semantic coverage of natural-language workflows. Upstream updates require explicit compatibility acceptance before installation.
