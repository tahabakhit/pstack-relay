# Configuration reference

Configuration version 1 separates `harness`, `model`, `effort`, `backend` and `auth`.

```json
{
  "version": 1,
  "session": "",
  "max_concurrent": 4,
  "timeout_seconds": 600,
  "roles": {
    "how explainer": [
      {"harness": "claude", "model": "haiku", "effort": "low", "backend": "rex", "auth": "existing"}
    ],
    "how explorer": [
      {"harness": "claude", "model": "opus", "effort": "high", "backend": "rex", "auth": "existing"},
      {"harness": "pi", "model": "openai/gpt-6.1-sol", "effort": "medium", "backend": "rex", "auth": "existing"}
    ]
  }
}
```

Model names are illustrative. The runtime does not hardcode default models. Unknown JSON fields, unsupported auth/backend selections and empty role lists fail validation. Effort is requested explicitly when configured. Effective effort remains unknown in the initial receipt format.

## Locations and precedence

The default is `$XDG_CONFIG_HOME/pstack-rex/config.json`, or `~/.config/pstack-rex/config.json`. `--config PATH` selects exactly that file. Global flags precede the command.

```sh
pstack-rex --config /absolute/config.json --state-root /absolute/private-state models
```

Project overrides load only with an explicit `--project-config PATH`. They replace complete role lists and accept only `version` and `roles`. They cannot change the session, deadline, concurrency or supported auth modes. Repository content never automatically selects a paid provider. The initial release has no separate project permission ceiling because only existing auth and fixed harness permission profiles are supported.

The state root defaults to `$XDG_STATE_HOME/pstack-rex`, or `~/.local/state/pstack-rex`. It must be private, mode 0700, on the Rex server machine. Attempts and logs are private files. Use the same root for every parent that must share the concurrency limit.

## Fields

| Field | Meaning |
| --- | --- |
| `version` | Exactly `1` |
| `session` | Empty creates a new server session per worker; an existing full session ID groups workers |
| `max_concurrent` | Maximum unresolved or active attempts under this state root, default 4 |
| `timeout_seconds` | Worker deadline ceiling, default 600; individual requests may shorten it |
| `roles` | Ordered target arrays indexed from zero |
| `harness` | `codex`, `claude`, `pi`, or unresolved `inherit-parent` |
| `model` | Exact CLI model or documented Claude alias; Pi requires `provider/model` |
| `effort` | Harness effort, `default`, empty, or unresolved inheritance |
| `backend` | `rex` or `auto`; native is rejected |
| `auth` | Only `existing`; funding is unknown and no new credentials are stored |

The resolver supports explicit parent inheritance internally, but CLI and MCP do not currently supply parent identity. `auto` or `inherit-parent` as a model, and inherited effort/harness, therefore fail dispatch. They do not mean “let another harness select a default.” Backend `auto` is a separate concept and resolves Rex.

Allowed effort spellings are validated syntactically. Support for a particular model/effort combination is ultimately established by the installed CLI and a real probe. Rejection never causes a different effort, model or provider to run silently.

`models --set-role ROLE --target DESCRIPTOR` replaces that whole role. Repeated `--target` values define the panel. A cross-judge pool is ordered choices for one judge, not a panel. The workflow selects eligible independent reviewers and judge families; the dispatcher does not claim to prove a workflow's diversity from a role name alone.

The authoritative role vocabulary appears in `pstack-rex models` and is checked against the canonical source during upstream review. It includes the combined keys `feature, refactoring` and `reflect judgment, divergent, synthesizer`, and the current `hardest tasks` key.

## Worker requests and outcomes

CLI dispatch requires `--id`, `--role`, `--cwd` and `--prompt-file`. `--seat` defaults to zero. `--access` defaults to `read-only`. `--revision` can require the exact starting Git SHA. `--require-mcp` rejects the request before inference because the initial profiles do not expose worker MCP.

The MCP tools are `dispatch`, `inspect`, `list`, `wait`, `cancel` and `capabilities`. Their input schemas are the interface reference. There is no generic shell tool. A dispatcher cannot execute arbitrary caller-supplied commands or replace a configured target through tool arguments.

IDs contain 1 to 80 letters, digits, underscores or hyphens, beginning with a letter or digit. Same ID and same resolved request returns the existing attempt. Changed input under that ID fails. Automatic retries and quota fallbacks are disabled. A conscious retry requires a new ID and the workflow's approval and independent-review rules.

`completed` requires process exit zero, semantic terminal success and final text. `failed`, `cancelled`, `timed_out` and `unknown` are not successful review verdicts. `wait` returns nonzero for every status except completed. A wait timeout does not cancel the worker.

Receipts record starting/ending Git revisions and snapshot hashes covering tracked diffs and untracked, nonignored files. They are observations, not atomic snapshots of a concurrently edited checkout. A writer starts from a clean checkout and owns a detached worktree. Untracked new files remain in that worktree; `changes.patch` covers tracked changes only. Every result has `verification: not-performed` until a separate workflow supplies verification evidence.

Codex exec provides command-line model evidence only. Claude assistant messages provide the reported model. Pi provides a selected model and, when present, a provider response model. The evidence field distinguishes these sources. Requested effort, effective effort, model identity and funding are separate facts.

`--require-shell` and MCP `require_shell` reject Claude and Pi controlled profiles before launch. The `interrogate reviewers` role always requires a shell because upstream requires each adversary to prove findings by running the code. Use Codex reviewers in this initial release. Claude and Pi remain useful for source investigation, architecture and prose, but their read-only profiles do not execute tests. Pi also performs a free RPC state preflight to reject fuzzy model selection and normalized effort before inference. Each output stream is limited to 32 MiB.
