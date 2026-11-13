# Contribute to pstack-rex

Use Go 1.23 or later and Git. Read [the design](docs/design.md) and [the delegation contract](overlay/delegation.md) before changing delegation. The CLI uses the Go standard library.

## Check a change

Run these checks from the repository root:

```zsh
go build -o /tmp/pstack-rex ./cmd/pstack-rex
go test ./...
go test -race ./...
go vet ./...
go run ./cmd/upstream sync --check
go run ./cmd/pstack-rex plugin build --check
git diff --check
```

Use the binary built from your checkout when checking changes to the CLI. `sync --check` validates the recorded revision format without network access. `plugin build --check` checks generated file contents, executable bits, and the file list without writing.

Go tests use fixtures and fake processes. They do not establish authenticated model access or live worker behavior. Record live acceptance commands, outcomes, and qualifications in `docs/acceptance.md`. Keep credentials and model calls out of ordinary CI.

## Change the plugin

Keep `upstream/` pristine. Put host behavior in `overlay/` and the Go implementation. Generate the plugin with `go run ./cmd/pstack-rex plugin build`, then inspect the generated diff. The generator checks source hashes before applying host adaptations.

Follow [upstream maintenance](docs/upstream.md) when importing Cursor changes. Upstream helpers may require Bun, Node.js, Python, or other tools. Read each helper's declared dependencies before running its checks. Go tests do not run those suites.

The package script is `bash scripts/package.sh /tmp/pstack-rex-packages`. It builds Linux and macOS archives for amd64 and arm64 and runs a host archive check. Cross-compilation does not prove runtime behavior on every target. Building archives does not publish a release.

## Prepare a contribution

Describe the problem, resulting behavior, checks run, and remaining limits. Preserve copyright and license notices. Scan the staged patch before committing:

```zsh
git diff --cached
gitleaks git --staged --no-banner
```

Do not commit detected secrets. Report an unavailable scanner rather than claiming the scan passed. Publication and hosted workflow execution require their own evidence.
