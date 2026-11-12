# Implementation contracts

These are the selected shared interfaces for the first milestone. The Go module path is `github.com/tahabakhit/pstack-rex`.

## Configuration

`internal/config` owns `Target`, `Config`, `Load(path, cwd)`, `Validate`, and `Resolve(role, index, parent)`.

```go
type Target struct {
 Harness string `json:"harness"`
 Model string `json:"model"`
 Effort string `json:"effort,omitempty"`
 Backend string `json:"backend"`
 Auth string `json:"auth"`
}
type Config struct {
 Version int `json:"version"`
 Session string `json:"session"`
 MaxConcurrent int `json:"max_concurrent"`
 TimeoutSeconds int `json:"timeout_seconds"`
 Roles map[string][]Target `json:"roles"`
}
```

Global path defaults to `~/.config/pstack-rex/config.json`, respecting XDG_CONFIG_HOME. Project `.pstack-rex.json` overrides roles by whole role list. Explicit `--config` selects only that file. Project configuration cannot alter server, session, auth/spending or permission ceilings. It supplies only `version` and `roles`. User must explicitly pass `--project-config` to load project policy, so repository content cannot choose a paid provider by itself.

An empty model or effort inheritance without a supplied concrete parent fails. Native execution always fails in version 0.1. `auto` resolves Rex. Role lists preserve order and seat cardinality. Configuration contains no executable, shell commands or credential material.

## Harness

`internal/harness` owns a shell-free plan and strict parser. It imports config only.

```go
type Plan struct { Executable string; Args []string; Stdin string; Environment []string; Enforcement string }
type Outcome struct {
 Text string `json:"text"`
 ReportedModel string `json:"reported_model,omitempty"`
 ModelEvidence string `json:"model_evidence"`
 SessionID string `json:"session_id,omitempty"`
}
func Build(target config.Target, access, cwd, prompt string) (Plan, error)
func Parse(target config.Target, stdout []byte) (Outcome, error)
```

Build resolves executable with `exec.LookPath`. Default access `read-only`. Only Codex supports `isolated-write`. Pi and Claude use explicit read-only built-in tool sets without native delegation tools. Their flags suppress specific customization mechanisms, not every possible startup input. Claude managed policy still applies, and Pi system-prompt discovery remains eligible. See [the shell investigation](shell-capability.md) and [PI-001](issues/pi-startup-inputs.md). Auth selection `existing`, `subscription`, `api` describes intended credential source. Unsupported enforced billing modes fail, and existing auth is explicitly unknown funding rather than falsely verified subscription. In v0.1 only `existing` is accepted until a harness-specific billing check is implemented. Plan Environment is the complete child environment, sanitized of parent session identity and unrelated provider keys, retaining only selected harness auth keys. No shell interpolation.

Parse requires terminal success and final text. A zero exit alone is never sufficient. Error, aborted and truncated streams fail. Model absence stays unknown and explicit selection is `pinned-argv`, not proof of served identity. Effective effort is unknown unless actually reported.

## Engine

`internal/engine` owns request reservation, local Rex transport, supervisor and results. It uses config and harness.

```go
type Request struct {
 ID string `json:"id"`
 Role string `json:"role"`
 Seat int `json:"seat"`
 Cwd string `json:"cwd"`
 Prompt string `json:"prompt"`
 Access string `json:"access"`
 Target config.Target `json:"target"`
 TimeoutSeconds int `json:"timeout_seconds"`
}
```

Root and worker run on the local Rex server host. Each attempt owns `request.json`, `launch.json`, `started.json`, `result.json`, streams and optional detached worktree. A local OS advisory lock serializes reservations and concurrency admission. Distinct processes never mutate the same receipt. Interrupted launch remains unknown until inspected. The supervisor claims the attempt exclusively and only one child may start. Cancellation marker is independent. The engine never deletes worktrees or discards partial changes automatically.

`Service{Root string, Config config.Config}` exposes `Dispatch(ctx, Request)`, `Inspect(ctx,id)`, `List(ctx)`, `Cancel(ctx,id)`, `Wait(ctx,id,duration)`. Dispatch returns `Record`; result is optional until terminal. It resolves Target from role and seat itself, not caller-supplied launch flags. Dispatch requires user-supplied ID. Identical ID+canonical request returns existing record. Different request with same ID fails.

`RunWorker(root,id)` is invoked by the absolute running binary in a Rex block. It uses the immutable resolved request. It keeps a process group for bounded cancellation, writes atomic terminal JSON and emits OSC 7501 working/done/error/idle. Status uses Rex process exit when result is absent. Failed result parsing, missing completion, deadline, transport failure and cancellation stay distinct from completion.

## MCP and CLI

`internal/mcp.Serve(in io.Reader,out io.Writer, service *engine.Service) error` supports line-delimited JSON-RPC stdio MCP, initialize, ping, tools/list and tools/call. Tools are dispatch, inspect, list, wait, cancel, capabilities. Same engine policy as CLI. Wait bounds at 60 seconds. Unsupported steer/resume/send fail through capabilities. No generic shell execution tool.

The CLI supports setup, doctor, models, dispatch, inspect, list, wait, cancel, worker, mcp, install, upstream check and version. setup creates an empty config without overwriting. Installation writes one complete skill bundle into an explicitly chosen empty destination and prints MCP registration instructions. It does not edit unrelated user settings or replace an existing pstack install.

## Upstream bundle

`internal/bundle` embeds an explicitly imported pristine source snapshot, a lock and runtime adapter. An offline check validates exact input paths, bytes and executable bits. Generation injects one adapter pointer into all skill and agent entrypoints, including nested skills if packaged. Original content remains unchanged in the imported snapshot. The same transformation applies consistently, avoiding manually maintained skill forks.

The sync command fetches with gh/git into an external temporary directory, validates the commit, copies only explicitly scoped pstack content, and produces changed-file/delegation reports. Any changed/new/deleted content makes compatibility review pending until the compatibility baseline is deliberately accepted. CI update PR is a proposal, not a compatibility approval.
