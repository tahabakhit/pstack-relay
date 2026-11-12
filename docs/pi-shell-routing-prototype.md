# Pi shell routing experiment

Status is a verified RPC routing observation, not an admitted coding profile. This disposable experiment supports the [1.0.0 execution-boundary comparison](release-1.0.md). It does not close [PI-001](issues/pi-startup-inputs.md) or establish confinement.

## Question

Does the actual installed Pi CLI honor a generated `shellPath` wrapper, and can project settings replace it?

The experiment ran Pi 1.5.2 in RPC mode with a fresh home and agent directory per variant, no real credentials, no provider requests and offline resource discovery. Generated agent settings selected a wrapper that records its arguments and then executes Bash. Project settings selected `/bin/bash` directly. An outer bubblewrap envelope masked real home, runtime sockets and temporary files. The wrapper itself was an observation instrument, not a sandbox.

The driver kept stdin open, correlated the `get_state` response, then submitted a `bash` request and waited for its correlated response. A parent-side check read the command marker and the wrapper marker. The selected provider/model was `anthropic/claude-haiku-5-5`, with requested and RPC-reported effort `low`. No model prompt was submitted.

## Observations

| Startup | Bash command executed | Generated wrapper used |
| --- | --- | --- |
| Fresh untrusted project with conflicting project setting | Yes | Yes |
| Same fixture shape with explicit project approval | Yes | No |
| Neutral control cwd, generated agent setting | Yes | Yes |

Every command returned exit 0 and wrote the exact `PI_RPC_CONTROL` marker. Explicit project approval allowed the project shell setting to take effect. This is settings precedence, not an escape from an active sandbox. A production wrapper design cannot assume its generated agent setting always wins.

The initial driver closed stdin immediately after sending requests. Pi returned only `get_state`, and no bash marker appeared. That observation was inconclusive. The corrected driver waits for each response before closing input. The [evidence index](validation/pi-shell-routing.json) records the corrected observations, file hashes and local artifact locations.

## Design implications and limits

A neutral per-attempt control cwd is a candidate way to avoid task-project settings while a trusted wrapper enters the actual worktree. This remains a hypothesis for the model-directed Bash path. RPC `bash` calls `session.executeBash`; it does not prove that model tool execution, startup discovery, authentication, refresh and selection preflight all use an equivalent controlled environment.

The installed source identifies both routes. `dist/modes/rpc/rpc-mode.js`, lines 461 to 478, calls session execution. `dist/core/agent-session.js`, lines 3061 to 3070, constructs the model Bash tool with the settings-derived shell path. `dist/core/tools/bash.js`, lines 62 to 83, starts the configured shell. These are source observations, not execution equivalence certification. The installed package is under `/home/taha/.pi/agent/install/releases/1.5.2/node_modules/@earendil-works/pi-coding-agent`.

Next test the model-directed tool path under synthetic provider responses, controlled startup inputs and an actual command boundary. Positive local execution and denied filesystem, credential, network and descendant operations need separate observations. Keep production shell and writer admission closed.

## Model-directed follow-up

A later synthetic Messages experiment reached the actual Pi model-directed Bash path from a neutral control cwd. It selected the generated wrapper twice. Correlated `tool_execution_start` and `tool_execution_end` events recorded the intentional assertion failure with structured exit code 1 and a passing marker command with structured exit code 0. The parent read the exact `PI_TOOL_CONTROL` bytes and matched each wrapper invocation to its tool command.

With the configured wrapper removed, both tool calls returned `Custom shell path not found` and neither command marker appeared. Pi still exited zero after the synthetic endpoint supplied final text. These errors are not genuine exit-1 reproduction evidence. A workflow must inspect correlated tool results and parent-side observations rather than terminal success alone.

Explicit system and append overrides excluded the selected agent-directory and task-project prompt canaries from the recorded provider requests. This tests those inputs in this fixture, not every startup mechanism. The wire request contained Bash plus a deferred provider placeholder. Installed `@earendil-works/pi-ai/dist/api/anthropic-messages.js`, lines 131 to 142, describes that placeholder as never activated. It was not invoked or independently execution-tested here.

The [model-tool evidence index](validation/pi-tool-routing.json) records the two variants and verifier result. The wrapper still only records and executes Bash. No confinement, real authentication, model reasoning or Rex integration follows. The next experiment is an actual command boundary with the same model-directed controls.

## Retention

Scratch source and results are at `/tmp/pstack-rex-pi-shell.5c7r0kvn`. The driver and corrected raw fixtures are preserved in `.git/pstack/experiments/pi-shell-routing-2026-11-12`. No installed third-party package source was copied into the repository. These local artifacts do not arrive in a GitHub clone.

The model-directed follow-up is at `/tmp/pstack-rex-pi-tools.hh2ro0qa`, preserved under `.git/pstack/experiments/pi-tool-routing-2026-11-12`. Its `verify.py` checks both retained variants offline. The synthetic endpoint uses fake credentials and returns fixed tool calls. It makes no real provider calls.
