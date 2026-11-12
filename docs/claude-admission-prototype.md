# Strict Claude policy admission experiment

Status is admission-subset verified. This is a disposable experiment for the [1.0.0 target](release-1.0.md), not production integration or a passing S03 verdict. [CL-001](issues/claude-invalid-policy.md) remains open.

## Question and method

Can a launcher reject a policy that differs from the complete generated profile before reaching its child-launch path, and pass an immutable validated snapshot to the actual Claude CLI?

The experiment uses one generated policy object, a strict JSON token reader that rejects duplicate keys and trailing input, and exact structural comparison against that object. Unknown keys, omissions, wrong types and flipped security booleans fail comparison. The launcher reads policy once, retains the validated object, and supplies serialized JSON through the settings argument. Claude does not reread the candidate policy file. A trusted test driver supplies fixture roots, command arguments, environment and the expected executable SHA256. Those caller inputs are not a proposed production interface.

The data shape is an approved generated profile plus a submitted candidate and an immutable admitted snapshot. A table of variants drives admission observations. A fake child writes a marker and captures its received snapshot. An allowed control proves that the child path works. It changes the original policy file before inspecting the argument, demonstrating that the received snapshot is independent of that file.

The actual CLI experiment reuses the earlier synthetic Messages endpoint and disposable outer bubblewrap envelope. Real home, runtime sockets and temporary files are masked. The installed Claude executable is mounted read-only. All credentials and secrets are canaries. There is no real provider inference.

## Observed results

The 72-case fake-child matrix passed. The approved policy started the child and preserved the expected snapshot after the original file changed. Every rejected candidate exited nonzero without the child marker. The matrix includes malformed and trailing JSON, duplicate keys, null, oversized input, an unsupported executable hash, omitted fields, wrong types, unknown keys and flipped security booleans. This is a count of tested variants, not complete schema or policy certification.

| Actual launcher variant | Launcher exit | Local Messages requests | Parent-side observation |
| --- | --- | --- | --- |
| Malformed policy containing `{` | 1 | 0 | Source unchanged, no scratch result, no escape marker, empty stdout. |
| Wrong type for `sandbox.enabled` | 1 | 0 | Source unchanged, no scratch result, no escape marker, empty stdout. |
| Approved policy supplied inline | 0 | 9 | Source read and scratch write succeeded. Source remained unchanged and no escape marker appeared. |

The approved run reached actual Claude Bash. Tool results `tool_prototype_0` and `tool_prototype_1` were successful. Results `tool_prototype_2` through `tool_prototype_7` were errors for source overwrite, private read, denied environment retrieval, Unix-socket creation, user-namespace creation and the unsandboxed-retry write. The verifier checked the exact tool IDs, result flags and filesystem bytes. The socket result was EPERM at creation. No listener accept log was recorded.

The init event advertised only Bash and reported permission mode `default`. Credential scrubbing forced that mode despite requested `dontAsk`, as in the original prototype. These observations are from Claude Code 2.1.322 on the local Linux host. The [evidence index](validation/claude-admission-prototype.json) records executable identity, file hashes, exact fixture paths and the verifier outcome.

## Limits and next experiment

This demonstrates the admission mechanism and actual CLI acceptance of an inline snapshot. It does not establish effective managed policy, complete filesystem/environment/network restrictions, lifecycle containment or complete S03 coverage. The launcher checks an executable hash before launching by path, leaving an executable-replacement race. It also trusts command arguments and environment from the test driver. Production admission must own those controls.

Managed settings can outrank command-line settings. The official [settings precedence](https://code.claude.com/docs/en/settings#settings-precedence) and [organization overrides](https://code.claude.com/docs/en/settings#organization-settings-override-everything) documentation describes that distinction. Exact validation of one settings object cannot certify all effective settings. Determine the actual managed/startup sources and prove refusal or the intended effective controls before promoting a native policy design.

Compare that design with a controlled command-execution boundary and whole-process isolation for both required worker harnesses. Authentication, startup inputs, detached descendants and permitted build/cache writes need explicit ownership. Keep production shell and writer gates closed until the selected boundary passes its own acceptance checks.

## Reproduction and retention

Scratch source and raw results are at `/tmp/pstack-rex-admission.4kV4pu`. They are preserved locally under `.git/pstack/experiments/claude-admission-2026-11-12`. They are not included in a clone. No downloaded third-party runtime source was copied into the repository.

The preserved files include `main.go`, `matrix.py`, `claude_mock.py`, `verify.py`, the matrix result, the native fixture directories and `verification.json`. Build `main.go` into `launcher`, run `matrix.py`, and run the synthetic CLI variants in the same outer envelope described by [the original prototype](claude-sandbox-prototype.md#reproduce-the-experiment), substituting the new scratch root. Run `verify.py` against the retained complete fixture set. A fresh run needs a fresh scratch directory because the verifier rejects duplicate native variants.
