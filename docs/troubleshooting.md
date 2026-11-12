# Troubleshoot delegation

## Check prerequisites without inference

Run `pstack-rex doctor`. It checks binary presence, Rex connectivity and auth-status command results for configured targets. It does not print tokens. A successful status check does not prove account access to a selected model. Use `doctor --probe --role ROLE --cwd /repository` for that live check.

The initial tested versions are Codex 0.167.2, Claude Code 2.1.322, Pi 1.5.2 and Rex API 12. Newer or older versions may change flags or event schemas. Inspect retained stderr and report the versions and sanitized failure, not credentials.

## A Codex parent refuses MCP calls

With approval policy `never`, Codex can refuse MCP calls even when `default_tools_approval_mode` is `auto`. `auto` does not mean preapproved. In the tested version, explicit scoped `approve` for the trusted local pstack-rex server permits the call while keeping the parent read-only sandbox. Interactive users can retain their normal approval prompts instead.

Do not disable the entire parent sandbox to solve this. See the scoped configuration in the README and the [official settings](https://learn.chatgpt.com/docs/extend/mcp?surface=cli#configuration-reference). Two early live examples correctly blocked without dispatch when approval was absent. The example checker exits nonzero in that situation.

## A request is rejected before launch

Check the configured role and seat using `models`. Configure every role the workflow needs. There are no upstream-model defaults after adaptation. `native`, inheritance without a concrete parent, non-existing auth modes, invalid IDs and unsupported writer harnesses fail explicitly.

Interrogate reviewers require shell execution and therefore use Codex in the initial release. Claude and Pi controlled profiles cannot run proofs. Other command-dependent tasks must set `require_shell`.

`why` and `reflect` worker roles need MCP under the current upstream workflows. The one-shot worker profiles disable MCP and executable customization, so those roles are blocked. Keep the missing capability visible. Do not strip it from the brief to make the task launch.

## The Rex server is on another machine

Run pstack-rex and the coding harnesses on that machine against its local Rex Unix socket. The project and state directory must also exist there. Connect the macOS client or Rex Rune to that same server to view workers. A graphical client's local path is not a path on the server.

The initial engine rejects remote control endpoints rather than pretending local prompt/result files are shared. Cross-host file transfer and remote runner transport remain unimplemented.

## A launch or worker is unknown

Inspect its ID. Read `request.json`, `launch.json`, `started.json`, `child.json`, stderr and the structured stdout in the returned artifact directory. Use the full Rex session and block IDs to inspect the owned terminal.

```sh
rex block call -s SESSION_ID -b BLOCK_ID process
rex wait -s SESSION_ID BLOCK_ID --for 10s --json
```

A Rex command may have succeeded before its acknowledgement was lost. The supervisor writes its own Rex identities, which can recover an ambiguous launch when available. A supervisor exit without a terminal receipt stays unknown because its child might still be running. That attempt retains a concurrency slot.

Do not replay the request or kill a stale PID from a receipt. PIDs can be reused. Confirm the current owned process and descendants before manually stopping them. SIGTERM, SIGINT and SIGHUP are handled by the supervisor. SIGKILL, server crashes and processes deliberately escaping their process group require human diagnosis. There is no automatic crash-recovery engine or safe “forget this unknown attempt” shortcut in version 0.1.

## Cancellation is not confirmed

`cancel` writes a durable request and waits briefly. A confirmed supervisor result is cancelled, completed if completion won the race, or another terminal failure. Unknown or running status means termination has not been confirmed. Keep its writer worktree and ownership reserved.

A parent disconnection does not cancel a Rex worker. Its supervisor remains in Rex. Reconnect and inspect the same ID. Deadline cancellation is independent of the parent and remains active in the supervisor.

## A worker exits successfully but its result fails

Read stdout as JSON events and inspect stderr. Process exit zero is insufficient. Missing terminal success, a reported model mismatch, aborted output, error events or missing final text fail the task. Partial output is retained for diagnosis, not accepted as a passing independent review.

An initial Codex smoke run rejected an experimental-feature warning emitted as an error item. The launch profile now suppresses that known startup warning while retaining strict error parsing. Pi receives its prompt on stdin so an initial `@` is literal text rather than an attachment path.

## A writer refuses a dirty checkout

Commit the intended input or create an explicit clean snapshot before dispatching an isolated writer. The engine does not guess which uncommitted edits belong to a task. It creates a detached worktree at the recorded revision and preserves a nested working directory within it.

Review the result's worktree before integrating changes. Keep failed or cancelled worktrees until you decide what to preserve. Use `git worktree list` and remove only an owned, reviewed worktree. There is no automatic worktree pruning or force reset.

## Upstream installation reports pending compatibility

Review `internal/bundle/review-report.json`, the complete source diff, roles and `adaptations.json`. Then follow the [upstream update procedure](upstream.md). Do not run `accept` just to make CI green. Full input hashing detects drift; it does not certify arbitrary natural-language delegation semantics.

## A repository depends on Git filters or hooks

Owned Git operations disable hooks, filesystem-monitor programs, clean/smudge/process filters and external diff/text conversion. These can otherwise execute repository-selected programs before the harness sandbox starts. Worktree checkout uses the raw recorded Git tree. Repositories requiring filter-based hydration, including some LFS setups, are not fully supported for isolated writers. Read-only investigation can inspect already materialized files, but the runner does not execute hydration programs outside the sandbox.
