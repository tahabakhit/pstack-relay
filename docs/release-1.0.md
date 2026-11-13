# 1.0.0 release contract

Superseded by [the delegation design](design.md) on 13 November 2026. This page describes the 0.1 implementation.

Status is the selected delivery target, not a declaration of readiness. The next scoped release is 1.0.0. A restricted source-investigation release does not satisfy this contract.

## User outcome

Use Codex as the main chat and run Claude Code and Pi as coding workers through Rex. Configure harness, model, requested reasoning effort, and role independently. Both worker harnesses must investigate source, run local verification, and implement changes in isolated worktrees. Codex owns integration and independent verification.

The first execution target is a local Linux Rex server. This is the available validation host, not evidence of support for other worker hosts. Additional host support requires its own native execution checks. Remote clients may attach to that server. Cross-host file transport is outside this release.

The release must preserve the upstream workflows and pinned-source update process. A preserved skill file is not proof that its delegation path works. Required specialist capabilities must either work through the parent and supported workers or produce an explicit prelaunch refusal. Documentation must identify those limits before the user starts a workflow.

## Release acceptance

Every row is a release gate. A missing artifact or inconclusive observation leaves the gate open. Completion of a worker process is separate from verification of its work.

| ID | Required behavior | Passing evidence | Current gap |
| --- | --- | --- | --- |
| R01 | A fresh user installs the exact candidate and starts a Codex parent using the generated skills and local MCP server. | Clean-project installation, recorded MCP dispatch and wait, matching installed version and bundle identity. | Existing installed examples cover earlier limited workflows, not this complete contract. |
| R02 | Codex explicitly selects Claude and Pi role seats and delegates through Rex. | Both harnesses complete in one installed parent workflow with correlated request, launch, start and terminal receipts and actual Rex block identities. | Historical smoke tasks and a single external-harness parent example do not prove the combined workflow. |
| R03 | Each harness runs real local verification under its declared read-only execution profile. | A committed failing probe and a passing control execute through the actual tool path. Parent-side checks establish command, revision, exit, and unchanged protected files. | Claude and Pi shell-required dispatch is disabled. The current review verifier extracts Codex command events only. |
| R04 | Each harness implements an isolated change without changing the source checkout or another attempt. | Two overlapping writers, one per harness, modify the same relative filename in different owned worktrees. External checks read both exact results and confirm the source and sibling worktrees remain protected. | Non-Codex isolated writing is disabled. |
| R05 | Codex can inspect, test and deliberately integrate the changes each worker produced. | The parent verifies tracked edits and new files, rejects a deliberately bad result, integrates a passing result, and reruns the fixture tests. | No combined Claude/Pi implementation-and-integration example exists. Automatic merge is not required. |
| R06 | The declared execution boundary fails closed. | Applicable [shell acceptance checks](shell-capability-tests.md) pass for each enabled profile, including actual harness execution and Rex integration. Recorded controls distinguish denial from broken instrumentation. | CL-001 rejects the direct Claude candidate. Pi needs an external execution-boundary contract. No complete shell-matrix row is certified. |
| R07 | Worker startup and authentication use only the inputs trusted by the selected profile. | Synthetic hostile startup inputs do not execute or widen policy. Exact model selection is checked. Actual configured authentication works without leaking real credentials into retained evidence. | Effective Claude managed policy and Pi prompt/settings discovery need actual startup checks. Existing auth does not prove billing selection. |
| R08 | Failures remain observable and do not turn into successful reviews or automatic replay. | Both harnesses exercise parsing failure, model rejection, cancellation, deadline, parent disconnect and ambiguous supervisor outcome. Descendant behavior matches the declared boundary. Unknown attempts retain admission. | Current lifecycle tests and prior Codex evidence need coverage for the new profiles. |
| R09 | Configured seats, revisions and proof commands are reconciled by the workflow verifier. | Mixed-harness review accepts real evidence and rejects missing seats, wrong revisions, prose-only commands, mismatched identities, missing parent reruns and changed fixture bytes. | Existing strict review verification is Codex-shaped. Served identity and effective effort must remain unknown when not emitted. |
| R10 | Setup, diagnosis and capability reporting match the executable. | A fresh project can configure both harnesses, discover unsupported combinations before inference, inspect results, upgrade deliberately and uninstall only owned files. CLI, MCP and docs agree. | Capability reporting and user guidance describe the current restricted profiles. |
| R11 | Upstream compatibility remains auditable. | Pinned-source integrity and declared adaptation checks pass. A workflow capability map identifies supported delegation and explicit specialist refusals. Update proposals preserve independent compatibility review. | Broad lexical inventory is not complete semantic workflow certification. Real hosted update proposals remain untested. |
| R12 | Released artifacts are the exact reviewed candidate. | Race tests, vet, upstream checks, secret scans, native package installation and overwrite refusal pass. Checksummed archives, version output, MCP server version and release notes agree. Independent readiness review has no unresolved material finding. | Application and MCP currently report 0.1.0. No release is published. Current main CI is green, but it is not a 1.0.0 verdict. |

## Delivery sequence

1. Capture the baseline and keep this acceptance map current. Give every experiment an exact revision, environment, commands, outcomes and evidence location.
2. Resolve execution-boundary uncertainty before production integration. Begin with strict Claude policy admission and the existing malformed-policy negative control. Compare Pi whole-process isolation with a controlled command-execution boundary and test startup-input controls. Do not enable shell or writing to make a demonstration work.
3. Implement supported profiles as independently verifiable units. Separate read-only verification from isolated implementation. Preserve the existing source-only profiles until replacements pass their own gates.
4. Extend proof extraction and fixtures for actual Claude and Pi events. Negative controls must fail for the intended evidence mismatch, not a missing executable or generic exception.
5. Run the installed Codex parent through a mixed-harness coding and review task. Retain private raw receipts and traces with a sanitized public index. Finish with the parent independently checking and integrating the results.
6. Verify lifecycle failures and concurrency through the same installed path. Refresh setup, diagnosis, installation, upgrade and capability guidance against actual behavior.
7. Freeze a candidate revision. Build and review the exact artifacts. Resolve material findings, repeat affected checks, then publish 1.0.0 with checksums, supported-host requirements and explicit limits.

The [worker execution design](worker-execution-design.md) records the selected experiment path and responsibility boundaries. It does not admit a production profile.

## Scope boundaries

Codex is the only parent required for this release. Rex is the execution backend. Claude Code and Pi are required coding workers. Codex workers can remain available, but cannot substitute for either required harness in the acceptance evidence.

Native subagent execution, new parent harnesses, interactive worker steering, live resume, automatic crash recovery, nested workers, direct worker MCP, enforced subscription/API billing selection, remote staging, cloud provisioning and hosted Automations are outside this release. Unsupported requirements must fail explicitly. The parent may perform connected-source investigation itself when a workflow requires capabilities absent from a worker. This does not certify the original worker-MCP path.

No claim of arbitrary hostile-code containment follows from a worktree, tool list, process group or passing task. Each supported execution profile needs an explicit boundary and evidence. Whole-process isolation and authentication design remain engineering decisions to be tested, not settled by this release contract.

## Baseline evidence

- [Current implementation status](status.md) and [validation record](validation.md) describe the earlier controlled workflows and missing historical raw artifacts.
- [Harness plans](../internal/harness/harness.go) reject non-Codex writers and restrict Claude/Pi to source tools. [Dispatch](../internal/engine/service.go) rejects their shell-required tasks. [Capabilities](../internal/mcp/server.go) advertises only Codex shell and writer profiles.
- [CL-001](issues/claude-invalid-policy.md) records the actual Claude malformed-policy failure. [PI-001](issues/pi-startup-inputs.md) records the residual prompt-discovery gap and its limits.
- [Current-main CI](https://github.com/tahabakhit/pstack-rex/actions/runs/43313799064) passed Linux and macOS tests and packaging at `71e169917b44f95e2c3034d494665eddc820a2ed`. Hosted checks do not establish graphical or authenticated live macOS worker behavior.
- [Packaging](../scripts/package.sh) builds four checksummed archives. Binary availability does not certify each target's worker execution profile.

There is no completion-date estimate. The execution-boundary experiments determine the critical path. A failed experiment can change the design, but cannot reduce the required Claude/Pi coding outcome.
