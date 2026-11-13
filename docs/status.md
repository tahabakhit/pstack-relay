# 1.0.0 status

The 1.0 implementation is a Codex and Claude Code plugin with one shared model sheet and native, Orca, and interactive Rex delegation. [The design](design.md) defines the target. [The delegation reference](../overlay/delegation.md) is the agent-facing contract. The evidence below is narrower than full acceptance of every path.

## Verified live

The [acceptance record](../notes/v1-evidence.md) records these runs on 13 November 2026:

- A Rex seat resolved from a Claude parent to an interactive Codex worker. The worker returned a report, and `rex wait` observed `reported`. Two approval prompts were answered in its terminal. See item 3.
- A Codex parent in Orca ran `interrogate` with a native Codex seat and a Claude Orca worker in a child worktree. Both returned, and the parent reran every proof. See item 2b, including its recorded fixes and qualifications.
- A Claude parent invoked `interrogate` after the user typed its slash command. Its native Claude seat returned findings that the parent reran. The Codex Orca seat stopped at the old hooks-review prompt, so the record still requires that seat's rerun. See item 2a.

The record does not yet establish a complete setup acceptance run or a live no-backend fallback run. Resolver behavior is specified in [configuration](configuration.md#resolver-modes), separately from live acceptance.

## Known limits

- The Codex plugin ships no session hook. Its skill entrypoints direct the agent to read the delegation reference.
- Interactive workers can wait for approval under the user's normal harness settings.
- Model discovery does not prove account access, provider-served model identity, or effective reasoning effort.
- Worker reports require parent verification. A required independent review stays blocked when no reviewer can run.
- Cursor cloud agents, automations, and project stores have no host equivalent here.
- The hosted upstream sync has not run yet. A local run with a stubbed `gh` synced, validated, and reached the squash-merge step.

## After 1.0

- Pi plugin installation and Pi parent acceptance. Setup currently discovers Pi models for the sheet.
- Pi workers on Orca. Orca currently has no Pi agent, so those seats fall back to the parent's model.
- A herdr backend.
- The first hosted upstream sync run, with its actual outcome recorded.
