# Current status and continuation

The latest completed milestone is a reproducible installed multi-model `interrogate` workflow. The wider goal remains an upstream-tracking pstack integration with configurable harnesses, models, efforts and Rex execution. Completion of the controlled fixture does not establish every upstream workflow or future backend capability.

## Completed milestone

The implementation and validation record are committed through local commit `56de7706aedbf521863a6ef0f3a871f1267c0cc9`. Later documentation commits clarify status and handoff without changing runtime code. No repository remote, pull request or release has been created.

- [The interrogate example](../examples/codex-interrogate/main.go) installs the bundle, prepares a detached read-only worktree per configured reviewer, and launches a Codex parent through the local MCP server.
- [The verifier](../examples/codex-interrogate/verify.go) reconciles every configured seat, requests, receipts, actual probe commands, fixture identity and the parent's later rerun. Its public structured finding is the authoritative verified report. It does not classify arbitrary narrative claims.
- [The regression suite](../examples/codex-interrogate/verify_test.go) rejects missing or extra attempts, changed identities, invalid terminal states, fabricated execution evidence and missing parent reruns.
- [Review evidence](validation/interrogate-evidence.json) records the observed two-seat run. [Writer evidence](validation/writer-evidence.json) records two controlled concurrent isolated writers that left the source checkout clean.

[Validation](validation.md) describes the checks, independent reviews and limits. Requested model diversity is verified; provider-served model identities and effective effort remain unknown.

## Retention and reproducibility

The temporary live receipts, raw traces, archives, design candidates and fresh clone used for validation are no longer present on the handoff host. Public evidence and local checkpoint copies of the audit reports survive. Do not describe the original run as currently replayable without checking its artifact directory.

The source can rebuild archives using the documented contributor commands. A new authenticated live example run makes new model calls. Neither rebuilding nor rerunning was necessary for the subsequent documentation-only handoff. Older archive hashes identify the earlier tested bytes; they do not identify a rebuild.

## Next bounded milestone

Define and validate a bounded shell-execution profile for Claude or Pi before claiming cross-harness runnable review. Start with the installed harness controls and [current permission planning](../internal/harness/harness.go). Keep harness, model and backend selection separate.

The first deliverable is an evidence-backed capability contract and negative-test plan. Establish the supported operating systems, permitted subprocess and write behavior, customization loading, credential exposure and failure semantics. Compare the bounded profile with retaining the current read-only boundary. Implement only a profile the available controls can enforce and test. Do not enable unrestricted execution merely to make an example pass.

The next session should read this document and the project checkpoint, verify Git state and available artifacts, then scope that capability milestone. Do not repeat completed model tasks to reestablish history when the preserved reports are sufficient.

## Parked capabilities and external checks

These remain explicit future work, rather than requirements silently added to the next milestone.

- Certify semantic delegation coverage beyond the source hashes and broad lexical inventory.
- Add native execution, steering, resume, deliberate crash recovery and worker MCP with their own contracts.
- Add enforced funding selection and remote file staging.
- Exercise macOS and graphical Rex clients directly. Cross-builds do not establish runtime behavior.
- Exercise hosted CI and real update proposals after an authorized remote exists. Local fixture tests do not establish hosted results.

Unknown attempts retain capacity and are never automatically replayed. Preserve the existing Git-program protections and permission boundaries. Publication, remote creation and cleanup are separate operations.
