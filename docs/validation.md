# Validation record

This record describes the initial usable milestone on 12 November 2026. It separates execution evidence from workflow verification and unsupported capabilities.

## Automated checks

`go test -race ./...` passes on Linux amd64. It covers strict configuration, literal argument and stdin handling, model mismatch and terminal-event parsing, durable worker lifecycle, CLI/MCP boundaries, isolated writes, cancellation and deadlines. `go vet ./...` and pinned-source checks are part of the reproducible contributor commands. The source lock contains 573 files and 61 skill entrypoints, including nested automation skills.

The regression suite proves behavior through subprocess liveness, receipts and filesystem changes. Negative controls run against the earlier `a2bfff7` implementation reproduced the reviewed defects. The original supervisor crash freed capacity while its child survived. The fixed implementation reports unknown and retains admission. A further boundary regression verifies that configured Git hooks, filesystem monitors, filters and text conversion do not run during owned status, snapshot and worktree operations. The same test fails against the earlier implementation. Tests also cover SIGTERM group termination, literal Pi prompts beginning with `@`, nested writer working directories, untracked changes and different-commit snapshot identities.

Mocked Rex tests establish transport contracts without credentials. Harness fixtures establish parser contracts. Neither proves account model access. The original upstream Bun/Node helper suites are not run by Go CI.

The package script cross-builds four targets, verifies archive checksums and performs a native offline installation from an archive. Cross-building a macOS binary does not establish macOS runtime behavior. Ubuntu and macOS GitHub jobs are defined but have not run on GitHub in this unpublished checkout.

The updater was exercised with stubbed GitHub operations and real local Git fixtures. An unchanged pstack subtree at another root commit produced no proposal. Changed source retained the old compatibility approval and prepared a draft body. An injected producer failure preserved exit code 7 before any GitHub operation. No real pull request was created.

## Live execution

[The sanitized receipt extract](validation/live-evidence.json) records observed versions and outcomes. Raw transcripts and private artifact directories remain outside the repository.

| Test | Observed outcome | Evidence limit |
| --- | --- | --- |
| Claude through Rex | Completed, returned `PSTACK_REX_READY`, reported `claude-haiku-5-5` | Assistant message model, not effective effort or billing proof |
| Pi through Rex | Completed with literal stdin prompt; free RPC state probe selected exact model and effort before inference | Final message reported selected `gpt-6.1-sol`, not a concrete provider response identity |
| Codex through Rex | Completed and returned `PSTACK_REX_READY` | Explicit `gpt-6.1-sol` command-line selection; exec did not report served model |
| Codex parent, pstack `how`, Claude worker | Parent read the installed skill and contract, called MCP dispatch/wait, and returned the worker's explanation of the actual fixture source | One simple workflow, not comprehensive semantic parity |

All three initial harness smoke tasks were launched before any finished. The source records include their start/end times. This proves parallel execution through Rex, not a completed multi-model interrogate workflow. A reusable [Codex-parent example](../examples/codex-parent/main.go) independently checks the actual dispatch/wait trace against the retained worker receipt and exits nonzero if the path did not occur.

Initial failures were retained and diagnosed. Codex emitted an experimental-feature warning as an error item; the parser rejected that run despite exit zero. Suppressing that known launch warning produced a valid repeat. Parent MCP calls under approval policy `never` failed until the local server received explicit scoped `approve` policy. Workers were not switched to unrestricted permissions to solve either failure.

Rex creation and process APIs returned persistent session/block identities. The public Rex Rune source implements server-backed sessions and program-status display. Neither Rex Rune's GUI nor the macOS GUI was operated during validation. The implementation emits program-status sequences and uses documented server facilities; graphical compatibility remains a source-backed expectation pending hands-on testing.

## Product target status

| Acceptance target | Initial milestone |
| --- | --- |
| One installation in Codex | Project skills plus one local MCP server; exercised in a temporary Git project |
| Upstream workflows available | Complete pristine import; generated entrypoint contract and two declared adaptations |
| Harness/model role configuration | Implemented with ordered seats, explicit backend and requested effort |
| Claude, Pi and Codex through Rex | All exercised live |
| Multi-model review workflow | Configurable; full interrogate/arena workflow not live-verified |
| Client visibility | Server IDs observed; GUI presentation not exercised |
| Structured identity/status/results | Implemented and exercised |
| Interrupted workers cannot appear successful | Reproduced and tested, unknown remains unresolved |
| Parallel writer isolation | Detached Codex worktrees tested with fake harnesses; real writer inference not exercised |
| Central integration without hand-editing each skill | Deterministic generation from one source tree |
| Reproducible imports and update proposals | Implemented and fixture-tested; hosted Actions not exercised |
| Setup, inspection and diagnosis | CLI and MCP implemented; configuration remains noninteractive |

## Open boundaries

Native execution, live steering, resume, nested delegation, worker MCP, cloud VMs and remote file transport are unsupported. Claude and Pi are controlled read-only tool profiles. Runnable interrogate reviewers require Codex. Cursor-specific trackers, subscriptions and hosted Automations are preserved as reference content, not emulated.

Hash locking and exact patches catch unreviewed input changes. The broad delegation inventory has not been manually certified as a complete semantic classification. The central adapter governs all installed entrypoints, but it cannot hard-intercept arbitrary host tools or prove that every future model follows its instructions. Disabling native tools and retaining actual execution receipts improves the tested path without changing that limit.

There is no automatic recovery after SIGKILL or a server crash, and no automatic cleanup of successful partial work. Unknown attempts deliberately retain concurrency. Ordinary cancellation controls the process group, but cannot claim containment of a process that deliberately escapes it. These limits prevent unsafe replay and must be resolved explicitly by the operator.
