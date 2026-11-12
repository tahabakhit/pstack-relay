# Bounded shell review capability

Status is proposed, not implemented. The later [disposable prototype](claude-sandbox-prototype.md) rejected the direct-CLI candidate on invalid-policy admission. This investigation used source baseline `c29c5ca3484c903345fab023d6c7e993314acb27` on 12 November 2026. Shell-required Claude and Pi dispatch remains blocked by [the engine](../internal/engine/service.go), lines 104 to 106.

## Recommendation

Evaluate Claude first, with Bash as the only task tool and an explicit native sandbox policy. The direct invocation failed S03, so the next experiment needs a launcher that rejects invalid policy before Claude starts. Keep the existing tool-restricted profiles available for source inspection. Defer Pi shell support until an external OS boundary has its own contract and tests.

This choice gives reviewers a possible route to execute local probes without giving built-in file tools a second, unsandboxed access path. It does not establish confidentiality for the Claude process itself. Managed policy and the harness authentication process remain trusted. If those assumptions cannot be established, retain source-only support or design whole-process isolation separately.

| Alternative | Useful behavior | Limit | Decision |
| --- | --- | --- | --- |
| Current Claude and Pi profiles | Source inspection without command tools | Cannot execute a required reproduction | Retain |
| Claude native sandbox, Bash only | Shell commands and descendants have documented OS controls | Effective policy, credentials, sockets, and failure behavior need execution evidence | Candidate for the next experiment |
| Pi built-in bash | Can execute repository tests | Uses the Pi process's OS permissions | Reject as a bounded profile on its own |
| Whole-process container or VM | Can also contain harness startup and helpers | New filesystem, authentication, network, and lifecycle integration | Separate milestone if native controls fail |

## Observed starting point

Installed versions were Claude Code `2.1.322` and Pi `1.5.2` on Linux x86_64. `bwrap` and `socat` were present. Version/help commands and source inspection do not prove that a sandbox starts. [The observation record](validation/shell-controls.json) distinguishes those observations from the proposed tests.

[The adapter](../internal/harness/harness.go), lines 29 to 103, supplies Claude `Read,Grep,Glob` and Pi `read,grep,find,ls`. It rejects non-Codex isolated writing. It passes selected authentication variables, HOME, PATH, and configuration locations. That filter removes unrelated environment variables, but does not hide files accessible to the worker's OS account.

[The supervisor](../internal/engine/worker.go), lines 111 to 186, creates detached worktrees only for `isolated-write`. Existing read-only requests execute in their supplied cwd. Its process-group cancellation is a lifecycle mechanism, not a hostile-process containment boundary. Terminal parsing proves completion, not execution of a required probe. The candidate needs independent command and artifact checks.

## Documented Claude controls

Claude's [sandbox documentation](https://code.claude.com/docs/en/sandboxing#what-the-sandbox-restricts) describes shell and descendant isolation on Linux and macOS. It excludes built-in file tools, hooks, and MCP servers. Defaults permit workspace and temporary writes, broad reads, and inherited environment. The Linux socket filter is optional. None of these defaults meets the proposed contract unchanged.

The [CLI reference](https://code.claude.com/docs/en/cli-reference#cli-flags) permits explicitly named command tools under restricted mode. Safe mode suppresses ordinary customizations, but managed policy still applies. Installed help also warns that print mode can ignore invalid settings. The prototype must establish effective policy rather than infer it from accepted arguments.

The [settings reference](https://code.claude.com/docs/en/settings-reference), in its `sandbox.failIfUnavailable` entry, documents startup failure and command refusal controls. Its entries for `sandbox.allowUnsandboxedCommands`, `sandbox.excludedCommands`, and filesystem restrictions describe separate escape and access controls. Source precedence matters, particularly for merged lists and managed overrides. No ready-to-use settings JSON is certified by this investigation.

The same reference warns that `sandbox.allowNestedUserNamespaces` can permit escape. The sandbox documentation's "Disable filesystem isolation" section explains that `sandbox.enabled` can remain true while filesystem protection is off. Both need explicit checks.

The [subprocess environment documentation](https://code.claude.com/docs/en/env-vars#what-the-subprocess-environment-scrub-removes) describes credential scrubbing and Linux PID isolation. It leaves GitHub tokens, proxy values, and unrecognized secrets in some cases. The existing adapter does not inject that control. Scrubbing also disables sandbox auto-approval according to the [`sandbox.autoAllowBashIfSandboxed` entry](https://code.claude.com/docs/en/settings-reference). The candidate needs an explicit permission rule as well as OS restrictions. [Permission modes](https://code.claude.com/docs/en/permissions#permission-modes) distinguish tool approval from containment.

## Candidate contract

These are acceptance requirements for an experiment, not current capability claims.

| Boundary | Required behavior |
| --- | --- |
| Platform | Initially Linux amd64 with the exact CLI, dependencies, namespace support, and socket filter recorded. No macOS, WSL, or native Windows claim follows from Linux results. |
| Tools and startup | Retain safe mode, restricted mode, strict MCP configuration, noninteractive denial, and explicit model selection. Expose only Bash for task actions. Disable nested delegation and extra tools. Managed executable policy must be identified as trusted or the profile must refuse admission. |
| Shell approval | Permit sandboxed Bash through an explicit policy compatible with credential scrubbing. No bypass-permissions mode, command exclusion, or unsandboxed retry. Require effective `sandbox.filesystem.disabled`, `sandbox.allowNestedUserNamespaces`, `sandbox.enableWeakerNestedSandbox`, and `sandbox.enableWeakerNetworkIsolation` to be false. |
| Sandbox availability | Require `sandbox.enabled` and `sandbox.failIfUnavailable`. Missing or unusable enforcement must stop execution. Dependency presence alone is insufficient, especially for Unix sockets. |
| Source and writes | Use a disposable, exact-revision checkout. Deny writes to that checkout, the original source, common Git directory, receipts, harness configuration, and other attempts. Permit writes only to per-attempt temporary and build-cache locations. Tests that require source-tree writes are unsupported in this profile. |
| Reads and credentials | Permit fixture source and declared toolchain inputs. Deny private host files, authentication stores, sibling attempts, and host process environments. Apply read restrictions to resolved symlinks and descendants. Use synthetic credential canaries to establish what the shell can actually reach. |
| Environment | The harness may use its selected authentication. Bash and descendants must not receive provider tokens or other forbidden values through environment, files, descriptors, or host process inspection. A name-based scrub alone is insufficient. |
| Network and sockets | Task subprocesses have no external network access, host-loopback access, or access to SSH, container, Rex, and other host service sockets. The harness's own inference connection is outside this shell boundary. Dependencies are preinstalled. |
| Configuration | Repository content cannot widen access or load executable customization. Effective managed settings must be compatible. Invalid or ignored policy cannot produce an unrestricted command. |
| Results and failure | Preserve deadlines, cancellation, unknown-capacity retention, and no automatic replay. Record policy identity, CLI identity, actual command evidence, and independent fixture results. A denied probe cannot count as an executed reproduction. |

An isolated checkout prevents accidental shared edits. It is not an OS sandbox. Existing Git-program protections cover engine-owned Git calls, not arbitrary Git invoked by repository tests. The candidate therefore denies writes to shared Git state at the OS boundary.

## Pi remains source-only

Pi's [security guidance](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/security.md#choose-how-to-run-pi) states that enabled tools use the account's permissions and cwd does not confine them. Installed `dist/core/tools/bash.js`, lines 60 to 81, directly spawns a shell. Adding `bash` to the tool list supplies execution without an OS boundary.

The current flags also do not eliminate every startup input. Installed `dist/core/resource-loader.js`, lines 481 to 505 and 959 to 979, discovers system-prompt files independently of context files. A direct canary reproduced that distinction without inference. [PI-001](issues/pi-startup-inputs.md) tracks this narrower profile gap. It is not evidence of command execution or a permission escape under the current tool list.

## Promotion gate

The [negative-test plan](shell-capability-tests.md) defines the acceptance gate. The initial investigation ran no Claude sandbox commands. The later [prototype](claude-sandbox-prototype.md) exercised selected controls through the installed Claude tool path using synthetic API responses and failed S03. It did not complete the gate or run real provider inference. No access value, dispatch gate, permission setting, or installed runtime contract changes here.

Promotion requires passing boundary tests on the exact supported host, then a real Rex-launched review with independently verified command evidence. An independent review must assess that implementation before shell capability is advertised. If a required control cannot be observed or enforced, keep the gate closed rather than weaken the contract.
