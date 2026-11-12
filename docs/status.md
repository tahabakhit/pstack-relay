# Current status and continuation

The latest runtime milestone is a reproducible installed multi-model `interrogate` workflow. A subsequent [shell capability investigation](shell-capability.md) defines a proposed Claude profile and [acceptance tests](shell-capability-tests.md). The [disposable prototype](claude-sandbox-prototype.md) rejected its direct-CLI variant on malformed-policy admission. Shell execution remains disabled for Claude and Pi. The wider goal remains an upstream-tracking pstack integration with configurable harnesses, models, efforts and Rex execution.

## Completed milestone

The runtime implementation and validation baseline is commit `56de7706aedbf521863a6ef0f3a871f1267c0cc9`. Later changes record the shell investigation and align the Go module identity for GitHub publication. No release archives have been published.

- [The interrogate example](../examples/codex-interrogate/main.go) installs the bundle, prepares a detached read-only worktree per configured reviewer, and launches a Codex parent through the local MCP server.
- [The verifier](../examples/codex-interrogate/verify.go) reconciles every configured seat, requests, receipts, actual probe commands, fixture identity and the parent's later rerun. Its public structured finding is the authoritative verified report. It does not classify arbitrary narrative claims.
- [The regression suite](../examples/codex-interrogate/verify_test.go) rejects missing or extra attempts, changed identities, invalid terminal states, fabricated execution evidence and missing parent reruns.
- [Review evidence](validation/interrogate-evidence.json) records the observed two-seat run. [Writer evidence](validation/writer-evidence.json) records two controlled concurrent isolated writers that left the source checkout clean.

[Validation](validation.md) describes the checks, independent reviews and limits. Requested model diversity is verified; provider-served model identities and effective effort remain unknown.

## Retention and reproducibility

The temporary live receipts, raw traces, archives, design candidates and fresh clone used for validation are no longer present on the handoff host. Public evidence and local checkpoint copies of the audit reports survive. Do not describe the original run as currently replayable without checking its artifact directory.

The source can rebuild archives using the documented contributor commands. A new authenticated live example run makes new model calls. Neither rebuilding nor rerunning was necessary for the subsequent documentation-only handoff. Older archive hashes identify the earlier tested bytes; they do not identify a rebuild.

## Next bounded milestone

The capability contract and negative-test plan are drafted from installed Claude Code 2.1.322, Pi 1.5.2, current official documentation, and adapter source. The recommendation is a Claude Bash-only native-sandbox experiment. Pi's native bash has no intrinsic OS boundary. Existing source-only profiles remain available.

Selected offline and actual Claude tool-path probes ran without provider inference. The well-formed policy blocked selected forbidden accesses, but a malformed settings file permitted them. A separate agent independently reproduced both variants. [CL-001](issues/claude-invalid-policy.md) blocks promotion. Next, prototype strict prelaunch policy validation and repeat S03 before integrating a profile or completing the remaining matrix. Linux results cannot certify macOS or other platforms.

A direct Pi loader canary also proved that disabling context files does not disable system-prompt discovery. [PI-001](issues/pi-startup-inputs.md) tracks the startup-profile gap. The canary does not demonstrate a tool escape. The latest [observation record](validation/shell-controls.json) records its scope. Do not repeat historical model tasks to reestablish prior milestones.

## Parked capabilities and external checks

These remain explicit future work, rather than requirements silently added to the next milestone.

- Certify semantic delegation coverage beyond the source hashes and broad lexical inventory.
- Add native execution, steering, resume, deliberate crash recovery and worker MCP with their own contracts.
- Add enforced funding selection and remote file staging.
- Exercise macOS and graphical Rex clients directly. Cross-builds do not establish runtime behavior.
- Exercise hosted CI and real update proposals after an authorized remote exists. Local fixture tests do not establish hosted results.

Unknown attempts retain capacity and are never automatically replayed. Preserve the existing Git-program protections and permission boundaries. Publication, remote creation and cleanup are separate operations.
