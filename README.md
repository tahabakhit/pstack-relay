# pstack-rex

Run upstream pstack workflows from Codex while delegating to Codex, Claude Code or Pi workers in persistent Rex terminals.

This is an initial usable integration, version 0.1.0. Cursor owns the engineering workflows. pstack-rex owns role resolution, permission profiles, Rex execution, receipts and installation. The included pstack version is 0.15.36 at `a5539b15a3d56e04992565aafee2b37051329ce3`.

See [current status and continuation](docs/status.md) for the completed milestone, artifact retention and next bounded work.

The next scoped release is [1.0.0](docs/release-1.0.md), with Codex as the main chat and Claude Code and Pi as coding workers through Rex. That target requires runnable verification and isolated implementation from both worker harnesses. The current integration does not yet meet those requirements.

A real Codex parent has used the installed pstack `how` workflow to dispatch a Claude worker through MCP and Rex, receive its result, and answer from it. An installed `interrogate` workflow has also completed with two explicitly requested Codex models, runnable reviewer probes and a parent rerun. All three worker harnesses have separately completed live Rex smoke tasks. See [validation and limits](docs/validation.md). Full workflow parity, native subagents and live steering are not claimed.

## Build

Requirements are Go 1.23 or later and Git. Runtime workers require a running local Rex server with API revision 12 or later and their authenticated harness CLI. The runner, repository and harnesses execute on the server machine. Graphical clients can be on other machines.

```sh
go build -trimpath -o bin/pstack-rex ./cmd/pstack-rex
./bin/pstack-rex version
```

Put the binary on your PATH. It embeds the complete pinned upstream bundle and needs no runtime package manager or pstack fork. Some upstream specialist scripts have their own Bun, Node or Java requirements. They are preserved, not installed or executed by this build.

`make package` creates checksummed archives for Linux and macOS on amd64 and arm64. The packaging smoke test installs from the native binary without network access. No release archives have been published. Build from source using the commands above.

## Configure workers

Start with an explicit assignment. Substitute a model you can actually use.

```sh
pstack-rex setup
pstack-rex models --discover
pstack-rex models --set-role 'how explainer' --target 'claude:haiku@low'
pstack-rex doctor
pstack-rex doctor --probe --role 'how explainer' --cwd "$PWD"
```

`doctor --probe` makes a real model call. Authentication readiness and catalogs alone do not prove model access. Existing harness authentication is reused. Its API versus subscription funding remains unknown. Credentials are never written to pstack-rex configuration.

Configure a panel by repeating `--target`.

```sh
pstack-rex models --set-role 'interrogate reviewers' \
  --target 'codex:gpt-6.1-sol@high' \
  --target 'codex:gpt-6.1-terra@medium'
```

Interrogate requires runnable proofs, so its initial worker profiles require Codex shell execution. Claude and Pi can be assigned to source-investigation and prose roles.

These descriptors are syntax examples. They are not universal defaults or account-entitlement claims. Each configured seat is resolved independently. There is no model substitution or automatic retry. See [configuration](docs/configuration.md).

## Install into a Codex project

First disable other pstack plugins or skills in the parent session. This project supplies the upstream skills itself.

From the target repository, install into an absent or empty `.agents` directory.

```sh
pstack-rex install --dest "$PWD/.agents"
codex mcp add pstack-rex -- /absolute/path/to/pstack-rex mcp
```

Codex discovers project skills under `.agents/skills`. Installation refuses to overwrite existing content. If `.agents` already contains other tools, generate into a separate empty directory, then explicitly register or link the generated skills you want. Do not replace unrelated configuration. The complete generated tree is also usable as a global skill source through explicit links.

Restart Codex. Use `$how` or `$poteto-mode` with your task. A generated entrypoint loads the shared runtime contract, which routes delegation through the `pstack-rex` MCP tools. The original upstream workflows and helper files remain in the bundle. Installation changes model setup and worker-bootstrap instructions through two checked adaptations.

For a dedicated parent, disable `multi_agent` and `multi_agent_v2` to remove alternate native delegation. The adapter cannot intercept host tools. Keep your normal approval policy and approve the local MCP tools when prompted. For unattended use, the installed Codex version supports scoped preapproval in its user configuration.

```toml
[mcp_servers.pstack-rex]
command = "/absolute/path/to/pstack-rex"
args = ["mcp"]
default_tools_approval_mode = "approve"
startup_timeout_sec = 10
tool_timeout_sec = 90
```

Only configure scoped preapproval after reviewing the local server and its role policy. It permits pstack-rex to launch configured workers, including permitted isolated writers. It does not make workers unrestricted. An explicit config path belongs before `mcp` in `args`. See the [official Codex MCP settings](https://learn.chatgpt.com/docs/extend/mcp?surface=cli#configuration-reference).

## Run the end-to-end example

Run this from a source checkout. The example source included in binary archives is for inspection and needs the checkout's Go packages to execute. It uses real authenticated models, creates a temporary Git fixture and installs skills there. It retains transcripts, worker receipts and Rex sessions, and changes no global Codex settings.

```sh
go run ./examples/codex-parent \
  --binary "$PWD/bin/pstack-rex" \
  --config "$HOME/.config/pstack-rex/config.json" \
  --parent-model gpt-6.1-sol
```

The configured `how explainer` must use Claude or Pi. The example verifies an actual MCP dispatch/wait pair and the durable worker result. Success produces `evidence.json` and the parent explanation. A failed approval or missing worker result exits nonzero. [Validation](docs/validation.md) describes the observed run and what it proves.

For a controlled multi-model review, configure `interrogate reviewers` with at least two distinct explicit Codex models, concrete efforts and `rex` backends. Every configured seat participates. The example requires Python 3 and an absent output directory outside the source repository.

```sh
go run ./examples/codex-interrogate \
  --binary "$PWD/bin/pstack-rex" \
  --config "$HOME/.config/pstack-rex/config.json" \
  --parent-model gpt-6.1-sol \
  --out /tmp/pstack-rex-review-proof

go run ./examples/codex-interrogate --verify-only /tmp/pstack-rex-review-proof
```

The first command makes real model calls. Replay checks retained artifacts without inference. Each reviewer gets a separate prepared read-only worktree and executes a committed failing probe. The parent reruns it after all terminal waits. The verifier checks every seat, durable receipts, actual command events, fixture bytes and one merged structured finding. It prints sanitized JSON and retains raw artifacts privately. Only the structured finding receives verification; the raw narrative is not semantically checked. This proves a seeded execution path and requested model diversity. Provider-served identities remain unknown.

## Inspect and cancel workers

```sh
pstack-rex list
pstack-rex inspect --id worker-key
pstack-rex wait --id worker-key --seconds 30
pstack-rex cancel --id worker-key
```

A worker's Rex session and block IDs are in its launch receipt. Attach any compatible client to that server and session. Each attempt retains structured stdout, stderr, final text, code revision, snapshot hashes and any worktree. Default execution creates a separate session per attempt. Set a full session ID in configuration to group workers.

`wait` exits zero only for a completed execution. Execution completion is separate from code verification or a passing independent review. An uncertain launch or a supervisor exit without a terminal receipt remains unknown and reserves concurrency capacity. Never reuse its ID with a changed brief or automatically launch a replacement.

## Supported boundary

| Capability | Initial support |
| --- | --- |
| Parent | Codex CLI through skills and stdio MCP |
| Workers | Codex exec, Claude print, Pi JSON print |
| Backend | Rex, `auto` resolves Rex |
| Read-only workers | Codex OS sandbox; Claude and Pi tool restrictions |
| Writers | Codex workspace-write in detached worktrees from a clean checkout |
| Runnable interrogate reviewers | Codex profiles; Claude and Pi lack shell access |
| Parallel roles | Configured seats with global admission limit per state root |
| Model identity | Requested values plus evidence actually emitted by each harness |
| Native, steer, resume, nested workers | Explicitly unsupported |
| Worker MCP, cloud machines, hosted Automations | Explicitly unsupported |
| Remote graphical clients | Server-backed sessions; no client modification |
| Cross-host file staging | Unsupported; run the adapter on the Rex server host |

Claude and Pi tool restrictions are not a filesystem or confidentiality sandbox. A worktree separates writes, not credentials or network access. No workflow gains a missing independent reviewer because a launch failed. MCP-dependent `why` and `reflect` roles are rejected before launch. Other specialist requirements must pass the shared capability check.

Read [architecture](docs/architecture.md), [research](docs/research.md), [upstream updates](docs/upstream.md), [troubleshooting](docs/troubleshooting.md), and [contributing](CONTRIBUTING.md).

## Upgrade and uninstall

Build or unpack the new binary. Check the reported pstack-rex and upstream versions, stop or finish active workers, and install the new bundle into a new empty destination. Switch only the links or project skill tree you own after reviewing it. Retain the old binary while its workers run.

Remove the MCP entry with `codex mcp remove pstack-rex`. Remove only the skill links or generated tree you installed. Configuration, receipts, worktrees and Rex sessions remain for deliberate inspection and cleanup. There is no background service to uninstall.

## Attribution

Original pstack is by Lauren Tan, MIT licensed. The pristine source and its license are under `internal/bundle/_upstream`. Integration code is MIT licensed. [NOTICE](NOTICE) preserves upstream attribution. [Research](docs/research.md) identifies the examined ports. The project is independent of Cursor, Superlogical and the coding-harness vendors.
