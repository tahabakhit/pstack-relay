# 1.0.0 status

The 1.0 implementation is a Codex and Claude Code plugin with one shared model sheet and native, Orca, and interactive Rex delegation. [The design](design.md) defines the target. [The delegation reference](../overlay/delegation.md) is the agent-facing contract. The evidence below is narrower than full acceptance of every path.

## Verified live

The [acceptance record](acceptance.md) holds these runs from 13 November 2026:

- `pstack-rex setup` installed the plugin in Codex and Claude Code from a local marketplace and wrote the shared sheet. The Claude `SessionStart` hook injected the delegation pointer in a fresh session.
- A Claude Code parent in Orca ran `interrogate` after the user typed its slash command. Its Claude seat ran native in its own worktree. Its Codex seat ran as an Orca worker in a child worktree from the parent's branch. Both returned, and the parent reran their proofs.
- A Codex parent in Orca ran `interrogate` with a native Codex seat and a Claude Orca worker in a child worktree. Both returned, and the parent reran every proof.
- A Rex seat resolved to an interactive Codex worker in a Rex block. The worker returned its report, and `pstack-rex rex wait` reported `reported`.
- The upstream sync script synced, validated, and reached the squash-merge step against a stubbed `gh` and a local origin.

The no-backend fallback is shown by resolver output only. A fallback seat is an ordinary native subagent on the parent's model, so no separate worker run was made.

## Known limits

- The Codex plugin ships no session hook. Its skill entrypoints direct the agent to read the delegation reference.
- Interactive workers can wait for approval under the user's normal harness settings.
- Model discovery does not prove account access, provider-served model identity, or effective reasoning effort.
- Worker reports require parent verification. A required independent review stays blocked when no reviewer can run.
- Cursor cloud agents, automations, and project stores have no host equivalent here.
- `pstack-rex setup` with the default source and `go install ...@latest` need this release on the repository's default branch. Until then, use `--source <checkout>`.
- The hosted upstream sync has not run yet. A local run with a stubbed `gh` synced, validated, and reached the squash-merge step.

## After 1.0

- Pi plugin installation and Pi parent acceptance. Setup currently discovers Pi models for the sheet.
- Pi workers on Orca. Orca currently has no Pi agent, so those seats fall back to the parent's model.
- A herdr backend.
- The first hosted upstream sync run, with its actual outcome recorded.
