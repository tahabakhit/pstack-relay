# Coding-worker execution design

Status is selected for experiments. No production profile is admitted by this decision. The [1.0.0 contract](release-1.0.md) requires Codex to delegate useful coding work to both Claude Code and Pi through Rex.

## Caller behavior

Keep the current dispatch interface. The parent selects a configured role and seat, exact starting revision, task brief, `read-only` or `isolated-write`, and whether execution is required. It does not supply mounts, raw policy, wrapper paths or permission flags. Missing prerequisites or incompatible trusted inputs produce an explicit refusal.

The supervisor supplies a disposable exact-revision source view for verification and a distinct owned worktree for implementation. Codex inspects the complete worker result, independently verifies it and deliberately integrates accepted changes. Workers edit files and do not commit. Completion remains separate from a passing review.

## Selected experimental shape

Use trusted harnesses with Bash as the sole task tool. Keep authentication and inference networking in the trusted harness. Confine task commands and descendants through a declared command boundary. This trusts the harness and selected startup/authentication inputs. It does not claim to contain a compromised harness or hostile host administrator.

For Claude, test the native Bash sandbox with strict policy admission and compatible effective managed settings. The [admission prototype](claude-admission-prototype.md) established only a subset. Managed-policy precedence, the complete read boundary, runtime identity and lifecycle still need proof. Unknown or incompatible effective inputs refuse admission. Do not remove or override organization policy to make a profile pass.

For Pi, start the CLI in an empty private control cwd outside the task source hierarchy. Generate a minimal agent directory with an exact shell wrapper selection, explicit system and append prompt inputs, and the existing resource-disable flags. The wrapper fixes the actual task cwd inside the command boundary. It accepts only the observed `-c <command>` protocol. The task cannot choose a policy, change control settings or access the control directory.

Pi's RPC and inference launch use the same prepared startup paths, runtime identity and tool policy. The [routing experiments](pi-shell-routing-prototype.md) establish selected wrapper routes and a trusted-project precedence hazard, not confinement. The prepared layout must also work for real configured authentication and its refresh behavior.

Test an alias to the existing Pi auth store before considering per-attempt copies. Installed Pi resolves aliases for locking. This is a feasibility hypothesis, not a passing refresh test. Inventory command-valued authentication and custom provider definitions as trusted host inputs. Their helper shell is separate from the task wrapper. Unknown executable helpers must not silently enter the trusted set. Task commands receive neither the auth binding nor its environment.

## Required boundary

| Resource | Verification | Isolated implementation |
| --- | --- | --- |
| Source | Exact disposable source view, read-only. | Only the owned attempt's source files are writable. |
| Other writes | Private scratch and cache. | Private scratch and cache. |
| Git | No writes to the worktree `.git` pointer, administrative metadata, common Git directory or original checkout. | Same. Parent owns capture and integration. |
| Reads | Declared source, runtime/toolchain inputs, owned scratch/cache. | Same. |
| Private inputs | Host home, credentials, parent environments, sibling attempts, receipts and control assets inaccessible. | Same. |
| Task network | No external, DNS, loopback, host Unix-socket or VM-socket access. | Same. |
| Processes | Private process view, closed unrelated descriptors, separately constructed environment and inherited confinement. | Same. |
| Final capture | Descendants settled before a terminal snapshot. | No descendant can continue changing the reported result. |

Dependencies are preinstalled or deliberately prepared by the parent. A worker cannot silently acquire network access to repair a missing dependency. A worktree and a process group are not substitutes for this boundary.

The Pi experiment starts with a minimal Linux namespace/mount tree, not a read-only host-root mount. Test socket-family and nested-namespace restrictions separately. Let actual detached-descendant and parent-death tests determine whether a delegated cgroup is needed. Do not add a daemon or require privileged host reconfiguration without evidence that the simpler lifetime boundary is insufficient.

## Ownership and internal sketch

The public CLI and MCP request shape stays unchanged. This is pseudocode for ownership, not a new implemented API.

```go
type requirement struct {
	access accessMode
	shell  bool
}

type profile struct { definition *profileDefinition }

type preparedWorker struct {
	plan      harness.Plan
	policy    admittedPolicy
	workspace taskWorkspace
}

type commandEvidence struct {
	callID      string
	command     string
	cwd         string
	exitCode    *int
	disposition commandDisposition
	source      eventLocator
}
```

Private constructors establish supported profiles and admitted inputs. A missing exit is unavailable, never zero. Command disposition distinguishes execution, denial, failure to start and unknown. The supervisor-time preparation must not treat an earlier dispatch check as permanent authorization for changed runtime or policy bytes.

- `internal/harness` owns version-specific profile definitions, startup plans, native policy encoding, Pi wrapper protocol, completion parsing and command-event normalization. Capability output and doctor explanations derive from those definitions.
- `internal/engine` retains reservations, Rex identities, worktrees, cancellation decisions and receipt ownership. Engine-local Linux code owns the execution handles and settlement used by its supervisor. No separate execution service is introduced.
- The existing binary can gain a private Pi command-wrapper entrypoint. It is not an MCP shell tool. Each invocation owns a separate create-once fact outside task mounts. It never appends to a shared terminal-result file.
- The workflow verifier joins real tool events, attempt identity, expected seat/revision, trusted invocation facts, fixture bytes and the parent's later rerun. It retains raw evidence privately and rejects ambiguous joins.
- Final result capture includes additions, modifications, deletions, modes and symlinks under the owned tree. The current tracked-only patch is a convenience, not a complete transferable result. Parent integration rejects traversal and unexpected file types.

Do not invent a tool-call ID for a wrapper that did not receive it. Repeated identical commands, parallel calls, output truncation and forged marker text must not create false correlation. Task stdout is never trusted as a wrapper receipt.

Runtime identity includes the executable, interpreter, imported modules and trusted helper closure. A path hash followed by reopening that path does not bind execution to admitted bytes. Test protected versioned trees, private immutable snapshots or opened execution handles as appropriate. The host owner remains trusted, but task code must have no mutation path to those inputs.

## Alternatives and synthesis

Four independent sketches compared a trusted-harness command boundary with whole-process alternatives. All recommended command-boundary-first. Agreement selects an experiment, not an enforcement verdict.

The parent and an independent GPT-6 Astra cross-judge selected candidate A as the base. It keeps the existing request and supervisor, separates Pi control/task cwd, and gives command evidence explicit unavailable and denied states. Candidate B contributed the concrete auth-helper inventory, protection of the `.git` pointer, adversarial correlation cases and parent-death race. Candidates C and D added no unique mechanism needed for this unit.

The parent rubric scores were A 18, B 17, C 13 and D 11 out of 20. The independent judge scored A 18, B 16, C 12 and D 11. These are subjective design assessments against five criteria, not measured capability or safety scores. Both selected A. The parent gave B and C slightly more credit for feasible requirements and retained A's smaller ownership boundary.

Reject a generic command-session API, public sandbox backend choices, arbitrary mount/policy bags, a persistent daemon and a provider proxy in the first unit. A whole-process container with credentials and inference networking exposed to task code does not satisfy the chosen task boundary. A credential-separated gateway or a two-level envelope remains a real fallback if startup or native policy fails. It needs its own authentication and task-network contract before implementation.

The frozen sketches, rubric, judge report and parent decision are preserved locally under `.git/pstack/designs/worker-execution-1.0`. They are not included in a clone. This document is the maintained synthesis; do not implement a discarded sketch independently.

## Next verifiable units

1. Add real isolation to the observed Pi wrapper route in a new disposable fixture. Run actual RPC and synthetic model-directed tools with allowed controls, denied access, and missing/failed boundary initialization. Check protected bytes and test-service observations externally.
2. Attack Claude effective managed-policy precedence and the complete read boundary through the actual CLI with synthetic responses. The existing strict admission subset need not be repeated merely to restate it.
3. Establish descendant settlement, runtime replacement resistance, startup-input coverage and real existing-auth behavior. Run credential leakage controls before real authentication. Preserve unknown admission and never replay automatically.
4. Implement read-only profiles as a separate unit, then isolated writers. Update capability reporting, receipts and installed guidance from the same admitted profile definitions.
5. Complete the applicable shell matrix and the installed mixed-harness coding/review fixture, including new files, rejected bad output and parent integration. Continue through all twelve release gates before 1.0.0 publication.

Current production shell and writer gates remain closed. Any experiment can reject the selected mechanism without reducing the required Claude/Pi coding outcome.
