# Validation record

This record covers the initial usable integration and the later multi-model review milestone on 12 November 2026. It separates execution evidence from workflow verification and unsupported capabilities.

## Latest verification and retention

The multi-model milestone at local commit `56de7706aedbf521863a6ef0f3a871f1267c0cc9` passed the full race suite, the Go 1.23.12 test suite, vet, source-lock checking, Bash syntax checks, ShellCheck and staged gitleaks. A fresh clone passed tests and built four checksummed archives. Native offline installation, overwrite refusal, archive contents and a real model smoke from the packaged Linux binary passed. Independent GPT-6 Astra source/evidence review and a separate artifact/trail audit found no unresolved material finding in their stated scopes.

These are completed historical checks. A later handoff check found the temporary live-run, writer, package and fresh-clone directories absent. The public evidence JSON and reports preserved in the local Git checkpoint remain available, but their hashes cannot reconstruct raw traces or archives. Replay of the original run requires recovering its artifacts. A new live example run makes new model calls and produces new evidence. The existing archive identities belong to `56de770`; later documentation commits do not update those archives.

## Automated checks

`go test -race ./...` passes on Linux amd64. It covers strict configuration, literal argument and stdin handling, model mismatch and terminal-event parsing, durable worker lifecycle, CLI/MCP boundaries, isolated writes, cancellation and deadlines. `go vet ./...` and pinned-source checks are part of the reproducible contributor commands. The source lock contains 573 files and 61 skill entrypoints, including nested automation skills.

The regression suite proves behavior through subprocess liveness, receipts and filesystem changes. Negative controls run against the earlier `a2bfff7` implementation reproduced the reviewed defects. The original supervisor crash freed capacity while its child survived. The fixed implementation reports unknown and retains admission. A further boundary regression verifies that configured Git hooks, filesystem monitors, filters and text conversion do not run during owned status, snapshot and worktree operations. The same test fails against the earlier implementation. Tests also cover SIGTERM group termination, literal Pi prompts beginning with `@`, nested writer working directories, untracked changes and different-commit snapshot identities.

Mocked Rex tests establish transport contracts without credentials. Harness fixtures establish parser contracts. Neither proves account model access. The original upstream Bun/Node helper suites are not run by Go CI.

The package script cross-builds four targets, verifies archive checksums and performs a native offline installation from an archive. Cross-building a macOS binary does not establish macOS runtime behavior. Ubuntu and macOS GitHub jobs are defined but have not run on GitHub in this unpublished checkout.

The updater was exercised with stubbed GitHub operations and real local Git fixtures. An unchanged pstack subtree at another root commit produced no proposal. Changed source retained the old compatibility approval and prepared a draft body. An injected producer failure preserved exit code 7 before any GitHub operation. No real pull request was created.

An independent GPT-6 Astra review checked the runtime, packaging, live claims and decision trail. It independently reran positive and negative Git-boundary tests and the earlier regression negative controls. No material correctness finding remained in the reviewed changes. The retained [negative-control output](validation/negative-controls.txt) shows actual old-code failures. This was an engineering review, not a complete security audit.

The final implementation commit `22eac9d84ef96ab73179ee1b67d7a2be4d83832d` passed tests from a fresh local clone, the Go 1.23.12 baseline, vet, exact upstream checking, actionlint and shellcheck. Its four archives passed checksums and the native offline install check. The packaged Linux binary also passed the real Codex-parent example. Subsequent documentation-only commits record these results.

## Live execution

[The sanitized receipt extract](validation/live-evidence.json) records observed versions and outcomes. Raw transcripts were kept outside the repository and are subject to the retention limits above.

| Test | Observed outcome | Evidence limit |
| --- | --- | --- |
| Claude through Rex | Completed, returned `PSTACK_REX_READY`, reported `claude-haiku-5-5` | Assistant message model, not effective effort or billing proof |
| Pi through Rex | Completed with literal stdin prompt; free RPC state probe selected exact model and effort before inference | Final message reported selected `gpt-6.1-sol`, not a concrete provider response identity |
| Codex through Rex | Completed and returned `PSTACK_REX_READY` | Explicit `gpt-6.1-sol` command-line selection; exec did not report served model |
| Codex parent, pstack `how`, Claude worker | Parent read the installed skill and contract, called MCP dispatch/wait, and returned the worker's explanation of the actual fixture source | One simple workflow, not comprehensive semantic parity |
| Codex parent, pstack `interrogate`, two Codex model seats | Every configured seat executed the fixed failing probe in its own detached read-only worktree; the parent waited for both and reran the probe | Controlled seeded fixture and requested model diversity, not general review quality or served-model identity |

All three initial harness smoke tasks were launched before any finished. The source records include their start/end times. This proves parallel execution through Rex, not a completed multi-model interrogate workflow. A reusable [Codex-parent example](../examples/codex-parent/main.go) independently checks the actual dispatch/wait trace against the retained worker receipt and exits nonzero if the path did not occur.

The later [interrogate example](../examples/codex-interrogate/main.go) completed with explicitly requested `gpt-6.1-sol` and `gpt-6.1-terra` reviewer seats. Its [sanitized evidence](validation/interrogate-evidence.json) binds both completed workers, actual exit-1 probe commands, the parent's later rerun and one structured finding to retained trace and receipt hashes. Replay verifies the same artifacts without further inference. Negative controls reject missing or extra attempts, wrong role, seat, target, cwd or revision, changed fixture bytes, prose-only execution claims, wrong probe paths and missing or early parent reruns. The public structured finding is the sole authoritative verified report. Raw parent prose is retained privately and is not semantically certified by the verifier. Independent inspection of this run's prose found only the seeded finding.

The first live replay exposed a verifier error. Actual Codex command events label nonzero commands `failed`, while the initial synthetic fixture labelled them `completed`. A failing regression now captures that shape. The corrected parser checks command status against exit code separately from MCP completion. Replaying the retained live run then passed; no model task was repeated to obtain a passing report.

A later [isolated-writer check](validation/writer-evidence.json) ran two real Codex workers through Rex with workspace-write access. Their execution intervals overlapped. Each wrote a different value to the same filename in its own detached worktree. External checks read both exact results, confirmed the seed files were unchanged, and confirmed the original checkout remained clean without the result file. The starting and ending commits matched. This establishes isolation for two controlled concurrent tasks. The engine still labels its own verification `not-performed`, and the additional file checks are separate evidence.

Initial failures were retained and diagnosed. Codex emitted an experimental-feature warning as an error item; the parser rejected that run despite exit zero. Suppressing that known launch warning produced a valid repeat. Parent MCP calls under approval policy `never` failed until the local server received explicit scoped `approve` policy. Workers were not switched to unrestricted permissions to solve either failure.

Rex creation and process APIs returned persistent session/block identities. The public Rex Rune source implements server-backed sessions and program-status display. Neither Rex Rune's GUI nor the macOS GUI was operated during validation. The implementation emits program-status sequences and uses documented server facilities; graphical compatibility remains a source-backed expectation pending hands-on testing.

## Product target status

| Acceptance target | Verified state |
| --- | --- |
| One installation in Codex | Project skills plus one local MCP server; exercised in a temporary Git project |
| Upstream workflows available | Complete pristine import; generated entrypoint contract and two declared adaptations |
| Harness/model role configuration | Implemented with ordered seats, explicit backend and requested effort |
| Claude, Pi and Codex through Rex | All exercised live |
| Multi-model review workflow | Installed interrogate path completed on a seeded fixture with two requested models, every configured seat, executed probes and parent rerun |
| Client visibility | Server IDs observed; GUI presentation not exercised |
| Structured identity/status/results | Implemented and exercised |
| Interrupted workers cannot appear successful | Reproduced and tested, unknown remains unresolved |
| Parallel writer isolation | Two real concurrent Codex writers produced distinct results in separate worktrees and left the source checkout clean |
| Central integration without hand-editing each skill | Deterministic generation from one source tree |
| Reproducible imports and update proposals | Implemented and fixture-tested; hosted Actions not exercised |
| Setup, inspection and diagnosis | CLI and MCP implemented; configuration remains noninteractive |

## Open boundaries

Native execution, live steering, resume, nested delegation, worker MCP, cloud VMs and remote file transport are unsupported. Claude and Pi are controlled read-only tool profiles. Runnable interrogate reviewers require Codex. Cursor-specific trackers, subscriptions and hosted Automations are preserved as reference content, not emulated.

Hash locking and exact patches catch unreviewed input changes. The broad delegation inventory has not been manually certified as a complete semantic classification. The central adapter governs all installed entrypoints, but it cannot hard-intercept arbitrary host tools or prove that every future model follows its instructions. Disabling native tools and retaining actual execution receipts improves the tested path without changing that limit.

There is no automatic recovery after SIGKILL or a server crash, and no automatic cleanup of successful partial work. Unknown attempts deliberately retain concurrency. Ordinary cancellation controls the process group, but cannot claim containment of a process that deliberately escapes it. These limits prevent unsafe replay and must be resolved explicitly by the operator.
