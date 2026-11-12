# CL-001: Claude print mode ignores malformed sandbox settings

Status is open. This blocks promotion of the direct-CLI shell prototype. It is a launcher integration gap, not a demonstrated escape from an active OS sandbox.

## Reproduced behavior

Installed Claude Code `2.1.322` received identical Bash-only controls in two disposable runs. One settings file contained the prototype policy. The other contained a single `{` character. Both runs used safe mode, restricted mode, disabled permission prompts, strict MCP configuration, explicit Bash approval, and subprocess credential scrubbing.

A local synthetic Messages API supplied deterministic Bash tool calls. The actual Claude executable executed the tools and returned their results. No real model inference, real credentials, or production permission changes were involved.

| Observation | Well-formed prototype policy | Malformed policy |
| --- | --- | --- |
| Read source | `SOURCE_CANARY` | `SOURCE_CANARY` |
| Write scratch | `SCRATCH_CANARY` persisted | `SCRATCH_CANARY` persisted |
| Overwrite source | Read-only filesystem error. Source unchanged. | Source became `CHANGED`. |
| Read private fixture | Path inaccessible | Returned `PRIVATE_CANARY` |
| Connect to fixture Unix socket | Socket creation failed with EPERM | Command completed successfully |
| Read explicitly denied synthetic environment values | No values returned | GitHub and unusual-name canaries returned |
| Attempt write with unsandboxed-retry parameter | Read-only filesystem error | Marker file appeared |
| Claude terminal result | Success | Success |

These observations come from `stdout.jsonl` tool results `tool_prototype_0` through `tool_prototype_7` and parent-side file reads in each run's `summary.json`. [The evidence index](../validation/claude-sandbox-prototype.json) records paths and hashes.

The retry parameter alone does not prove that Claude left an active sandbox. The forbidden accesses with ignored policy establish the failure. A namespace attempt in the malformed run failed with ENOSPC, which is not evidence that the candidate policy blocked it.

## Scope and limits

The prototype ran inside an outer bubblewrap envelope. Real home, runtime sockets, and temporary files were masked. Only the scratch experiment was writable. Positive controls and the malformed variant demonstrate that the tested fixture resources remained accessible inside that envelope.

This containment means the experiment does not establish arbitrary host access, complete credential confidentiality, or production managed-policy behavior. The well-formed policy tests selected canaries, not the full proposed host allowlist. It is not a passing result for the complete acceptance matrix.

The installed CLI help, under `--print`, warns that invalid settings may be ignored. The behavior is consistent with that warning. [Claude's CLI reference](https://code.claude.com/docs/en/cli-reference#cli-flags), under `--restricted`, also distinguishes restricting tools from enabling a shell sandbox. Safe mode and restricted mode alone are insufficient once Bash is explicitly available and approved.

## Smallest adequate next step

Prototype a launcher that validates the complete generated policy before starting Claude. Reject malformed JSON, wrong types, unknown keys, omitted restrictions, unsupported versions, and incompatible effective policy. Protect the policy against mutation between validation and use. A syntax check alone is insufficient.

Repeat this exact negative control through that launcher and verify that no Claude process or probe starts. Then test schema-valid weakening, managed overrides, and actual effective controls. A new launcher may close this gap, but this experiment does not prove that it does.

Keep Claude and Pi shell-required dispatch blocked until the acceptance gate passes. A normal terminal success cannot certify that required probes executed under the intended policy.
