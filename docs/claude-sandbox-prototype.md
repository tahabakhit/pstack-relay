# Claude sandbox prototype outcome

The direct-CLI candidate fails acceptance check S03. Malformed sandbox settings were ignored while Bash remained available and approved. The actual Claude tool path then changed a source canary and read a private canary. [CL-001](issues/claude-invalid-policy.md) records the finding and the required next experiment.

The prototype is throwaway. It changes no production adapter, capability gate, or permission configuration. The recommendation is to evaluate a launcher that refuses invalid policy before Claude starts, rather than integrate the tested invocation directly.

## Variants and observations

The experiment used installed Claude Code `2.1.322` on Linux amd64 and a separate standalone `@anthropic-ai/sandbox-runtime` `0.0.82` instrument. The standalone package is not established as the version embedded in Claude.

| Surface and variant | Observed result | Evidence limit |
| --- | --- | --- |
| Standalone runtime, unrestricted fixture control | Source and private reads, source and scratch writes, socket connections, and user-namespace creation succeeded. | Establishes reachability inside the outer envelope. |
| Standalone runtime, explicit policy | Source read and scratch write succeeded. Source overwrite, private read, Unix socket creation, host-loopback connection, and user-namespace creation failed. | Selected local probes only. |
| Standalone runtime, missing or malformed policy file | Exit 1. Task marker absent. | Standalone startup behavior does not predict Claude startup. |
| Actual Claude, well-formed policy | Source read and scratch write succeeded. Selected forbidden reads, writes, environment retrieval, socket creation, and namespace creation failed. | Synthetic model responses. No real review or Rex launch. |
| Actual Claude, malformed policy | Source changed, private canary returned, Unix socket command succeeded, denied synthetic environment values returned, and write marker appeared. | Establishes invalid-policy admission failure in this prototype. |

[The evidence index](validation/claude-sandbox-prototype.json) records exact artifacts, hashes, and observed values. Both actual Claude variants ended with terminal success. Their different tool results demonstrate why completion alone is insufficient.

## Instrument and isolation

The smallest instrument is a table of named probes, a local synthetic Messages endpoint, and parent-side file observations. `claude_mock.py` sends deterministic tool-use events using the [documented streaming event sequence](https://platform.claude.com/docs/en/build-with-claude/streaming#event-types). Claude itself chooses the execution and permission path for those tool requests. This tests the installed harness behavior without attributing any reasoning, served-model identity, or review quality to the synthetic endpoint.

The outer bubblewrap process masks real home, `/run`, and `/tmp`, binds the experiment directory writable, and provides private PID, proc, and device views. The Claude executable is rebound read-only from its installed path. The API key and environment canaries are synthetic. This protects host data while allowing the same fixture resources in the control and candidate variants.

The initial outer envelope omitted a private device mount. Node subprocess setup then failed with EACCES when opening `/dev/null`, causing false missing-dependency reports. Those runs are inconclusive. Adding `--dev /dev` repaired the instrument. The corrected run allowed both positive controls and actual sandbox execution.

The exact standalone package was installed outside the repository with lifecycle scripts and optional dependencies disabled, an empty environment, and a temporary home. The lockfile records the registry integrity. No production dependency changed.

## Permission behavior observed

With `CLAUDE_CODE_SUBPROCESS_ENV_SCRUB=1`, Claude warned that it forced the requested `dontAsk` mode to `default`. The init event confirmed `permissionMode: default` and `tools: [Bash]`. `--permission-prompts none` remained in the invocation, and Bash was explicitly approved. The well-formed policy still enforced the tested sandbox restrictions.

Consequently, an integration must inspect effective behavior instead of recording the requested mode as enforced. The scrub remains active in the malformed variant, but its documented exceptions are exposed when the policy's explicit environment denies disappear.

## Reproduce the experiment

The original scratch directory is `/tmp/pstack-rex-claude-prototype.0dD7i6`. It contains `offline.py`, `claude_mock.py`, the isolated runtime installation, and raw fixture directories. These paths are machine-local and temporary.

Copies of both scripts, the package manifest and lockfile, and selected raw evidence are preserved under `/home/taha/dev/agents/pstack-rex/.git/pstack/experiments/claude-sandbox-2026-11-12`. The scripts expect a scratch-relative runtime installation and Claude executable mount. No downloaded third-party package source is stored in that checkpoint.

From a reconstructed scratch directory, use the following outer envelope for the mock tool-path experiment. `valid` and `malformed` select the variants. Each run creates a new fixture directory. The empty `claude` file must exist before the bind mount.

```sh
env -i PATH=/usr/bin:/bin HOME=/tmp/pstack-rex-claude-prototype.0dD7i6/home \
	bwrap --ro-bind / / --tmpfs /home/taha --tmpfs /run --tmpfs /tmp \
	--bind /tmp/pstack-rex-claude-prototype.0dD7i6 /tmp/pstack-rex-claude-prototype.0dD7i6 \
	--ro-bind /home/taha/.local/share/claude/versions/2.1.322 /tmp/pstack-rex-claude-prototype.0dD7i6/claude \
	--unshare-pid --proc /proc --dev /dev \
	--chdir /tmp/pstack-rex-claude-prototype.0dD7i6 \
	/usr/bin/python /tmp/pstack-rex-claude-prototype.0dD7i6/claude_mock.py valid
```

The standalone experiment uses the same envelope without the Claude bind and runs `offline.py`. Reinstall its exact locked dependencies outside the repository with scripts disabled if the temporary runtime has disappeared. Do not infer artifact availability from this document.

## Remaining acceptance gaps

S03 has a demonstrated failure. The selected probes cover parts of S04 through S08, S10, and S14. They do not complete those checks. Missing-dependency Claude startup, effective managed policy, complete filesystem and credential isolation, all network transports, cancellation and detached descendants, concurrent attempts, real authenticated inference, and the installed Rex review remain unverified.

Do not run real inference merely to repeat this deterministic failure. First establish that a launcher rejects invalid policy and retains the intended boundary. Reassess the remaining matrix only after that correction passes its own negative control.
