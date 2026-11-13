# Rex worker launcher

`Start` launches an interactive Claude, Codex, or Pi command through `rex run`.
The caller supplies a runner (normally `exec.Command(name, args...).CombinedOutput`)
and environment lookup. Rex's default shell mode supplies the login environment;
the launcher does not filter the environment or use a noninteractive harness mode.

`Report` atomically replaces `result.md` and writes the Rex done-status escape to
stdout. `Wait` checks the report before querying process exit, so a report wins
over exit, including a report written during the process query. Timeouts return
`running`; command and malformed-JSON errors remain errors. The timeout controls
polling, so callers should give their runner its own command deadline if needed.

## Response shapes and local verification

On 13 November 2026, `rex run --help` confirmed login-shell environment handling
and the requested flags. `rex block call --help` confirmed method results are JSON
and the session/block selection flags. This Orca worker had no `REX_SESSION`, so
no live `rex run --json` launch was attempted or observed. The parser accepts
both task-specified forms, covered by fake-runner tests:

```json
{"block_id":"block-1"}
```

```json
{"block_ids":["first","second"]}
```

The first ID is used for the plural form. Process polling accepts
`{"last_exit":null}` and `{"last_exit":{"exit_code":0}}`; absence of an exit
code means no observed exit. A live interactive-worker acceptance run inside Rex
remains an integration check for the coordinator.

Failed starts leave their state directory, and any successfully created worktree,
in place for inspection. Worktree removal belongs to the caller's lifecycle.
