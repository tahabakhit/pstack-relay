# Architecture decision 001

Status is accepted for the first usable milestone. The independent architecture comparison selected the thin design, with per-attempt supervisors instead of a socket service. See [research](research.md) for source evidence and the comparison rubric.

## Goal

Keep Cursor's engineering workflows while making harness, model, reasoning effort and execution backend independent choices. Ship a working Rex delegation path before adding native execution.

## Constraints observed

The checkout starts empty. Cursor's canonical `pstack/` is version 0.15.36 at `a5539b15a3d56e04992565aafee2b37051329ce3`. Its Claude tree is generated from canonical source. Upstream owns the workflow content.

The installed Rex exposes `run --shell=none --json`, stable block IDs, `block call process`, and `wait` with child exit codes. CLI execution suffices. A separate service or server modification adds no necessary capability.

Native parent tools cannot be replaced by an MCP server. Generated skills can direct delegation, and a dedicated parent session can disable native delegation, but a skill cannot enforce that restriction by itself.

## Responsibility boundaries

- A single Go executable exposes a CLI and a stdio MCP interface. The process exits when its caller exits. There is no independent daemon.
- Configuration resolves role seats into explicit harness/model/effort/auth/backend selections. Missing roles and unsupported native execution fail. `auto` selects Rex until a native capability adapter exists. There are no implicit model fallbacks.
- Harness adapters own command arguments, permission capabilities, structured output and model evidence. They never own Rex sessions.
- The Rex engine owns durable attempt identities, launch reservations, block identities, cancellation, deadlines, result files and Git worktrees. A supervisor runs inside each Rex terminal so a disconnected parent does not end the worker.
- An embedded, pinned copy of upstream content owns the engineering workflow. Installation generates a copy with one shared runtime contract, preserving original prose and scripts. Compatibility checks lock every source byte and mode, plus a broad delegation inventory.

## Durable contract

Each attempt has a private directory. The parent writes an immutable request and Rex launch receipt. The supervisor writes its start receipt and terminal result. Cancellation is a separate marker. This avoids parent and supervisor rewriting the same state document.

A successful result requires process exit zero, a valid harness completion event and final text. It means execution completed. It never means that code or an independent review passed. The receipt records requested model/effort, provider-reported identity when available, evidence kind, exact starting Git commit, final Git commit and a worktree diff hash.

Request IDs are idempotency keys. Reusing one with different input fails. An ambiguous Rex launch is retained as unknown and never automatically retried. Successful partial work and worktrees remain available for inspection. Cancellation and deadlines terminate the child process group before returning a terminal result.

Default workers are read-only. Codex uses its read-only sandbox. Claude and Pi get explicit tool restrictions and flags that suppress specific customizations. Those restrictions are not an OS sandbox or complete startup isolation. Claude managed policy and Pi system-prompt discovery remain relevant, as recorded in [the shell investigation](shell-capability.md). Initial isolated writers are supported only through Codex workspace-write in a dedicated detached worktree. Unsupported writer combinations fail before launch.

The executable, repository, configuration and artifact directory must be on the Rex server host. Remote clients are supported because they attach to server sessions. Cross-machine filesystem staging is outside the first release.

## Alternatives under comparison

Forking Open Pstack supplies tested provider planning and receipts, but brings Bun, ten eagerly imported adapters, fork-specific model defaults and approximately 3,907 lines of production runner code. Its runner launches local processes, so Rex still needs separate durable launch and result ownership.

A native-only plugin is smaller but cannot provide cross-harness execution or Rex persistence. Intercepting built-in Agent/Task tools is not a documented portable plugin capability.

A custom orchestration daemon would add recovery and transport responsibilities that are unnecessary for a bounded first release. Immutable files and a per-attempt Rex supervisor provide the needed persistence.

## Initial milestones

1. Lock the upstream source, architecture and executable contracts.
2. Test configuration, adapters, Rex dispatch and supervisor with deterministic fake executables.
3. Demonstrate real Rex delegation and explicit model evidence.
4. Add the Codex installation, upstream updater, CI, packaging and documentation.
5. Independently review runtime and maintenance code, fix material findings, and record validation limits.

## Synthesis decision

Choose the thin Go design. Keep the upstream byte lock and single shared generated adapter from the thin candidate. Keep the provider-report versus pinned-argv evidence distinction and strict result parsing from the Open Pstack candidate. Reject the thin candidate's local socket service and the reuse candidate's additional runner dependency. Native delegation and MCP-dependent worker workflows are capability-gated rather than approximated. The cross-judge used GPT-6 Astra and independently favored this boundary.

## Principles applied

Foundational Thinking put shared contracts and verification scaffolding before implementation. Model the Domain separated the harness, model and backend and made attempt receipts explicit. Separate Before Serializing Shared State gave the parent and supervisor different receipt files and reserved a lock for admission only. Laziness Protocol removed the proposed daemon and extra provider-runner dependency. Exhaust the Design Space required distinct fork-based and thin-adapter candidates before the independent comparison. Prove It Works required real Rex processes, a real Codex-parent workflow and negative-control regression evidence.
