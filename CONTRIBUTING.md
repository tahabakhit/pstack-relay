# Contribute to pstack-rex

Use Go 1.23 or later, Git, and a POSIX environment on Linux or macOS. The Go runtime uses the standard library. Rex and the selected worker CLI are needed for live execution, not for the deterministic test suite.

## Check a change

Run these commands from the repository root.

```zsh
go build ./cmd/pstack-rex
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/upstream check
bash scripts/package.sh /tmp/pstack-rex-packages
```

The package script cross-compiles Linux and macOS archives for amd64 and arm64, writes checksums, and tests installation using the host archive. Cross-compilation does not prove execution on every target. The package check installs into a temporary directory with an empty `PATH` and verifies refusal to overwrite an existing installation.

The ordinary Go tests use controlled fixtures and fake processes. They do not require model credentials or make inference calls. Keep credentials and paid model calls out of ordinary CI. Run live Rex and harness checks separately with explicit local configuration, and report requested versus observed model evidence and any unverified permissions or graphical behavior.

## Change workflows and adapters

Read [the architecture](docs/architecture.md) and [runtime contracts](docs/contracts.md) before changing dispatch or lifecycle handling. Keep the shared CLI and MCP behavior consistent. Cover observable failure behavior when changing parsing, cancellation, idempotency, or writable ownership.

Keep `internal/bundle/_upstream/` pristine. Change runtime adaptation in this project's implementation. Follow [upstream review](docs/upstream.md) for source imports and record the source revision and compatibility decision.

The imported workflows include helper scripts with their own Bun, Node, Python, or other dependencies. `go test ./...` does not run those upstream suites. When changing helper behavior or accepting a relevant source update, read the helper's package metadata and run its declared checks separately. Do not add those dependencies to the Go runtime merely because the source bundle contains them.

## Prepare a contribution

Keep the change focused and describe the problem, resulting behavior, checks run, and remaining limits. Preserve source copyright and license notices. Before committing to a public repository, scan the staged patch.

```zsh
git diff --check
git diff --cached
gitleaks git --staged --redact
```

Do not commit detected secrets. Report an unavailable scanner rather than claiming the scan passed. Building archives does not authorize publishing them. Release publication and hosted workflow execution are separate actions.
