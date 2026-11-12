# Review a Cursor pstack update

Keep third-party checkouts outside this repository. Use a clean Cursor plugins checkout at an exact commit. The examples use `/tmp/cursor` and a shell variable containing the full SHA.

```zsh
gh repo clone cursor/plugins /tmp/cursor
revision=$(git -C /tmp/cursor rev-parse HEAD)
go run ./cmd/upstream check
go run ./cmd/upstream update --source /tmp/cursor --revision "$revision"
```

`update` imports tracked `pstack/` files, records hashes and executable modes, and writes the candidate manifest and review report. It leaves compatibility approval unchanged. A changed manifest makes `check` fail until explicit acceptance.

1. Review the complete source diff, including additions, deletions, scripts, nested references, and role changes.
2. Read `internal/bundle/review-report.json`. Review changed delegation candidates and the adapter's treatment of their semantics.
3. Resolve unsupported capabilities and conflicting instructions in the runtime adaptation. Keep the source import pristine.
4. Run relevant upstream helper tests with their declared dependencies when those helpers change.
5. Accept only the reviewed candidate, then run the local checks.

```zsh
go run ./cmd/upstream accept --source /tmp/cursor --revision "$revision"
go run ./cmd/upstream check --source /tmp/cursor --revision "$revision"
go test -race ./...
go vet ./...
bash scripts/package.sh /tmp/pstack-rex-packages
```

Commit the imported source, both manifests, review report, compatibility baseline, and any adapter changes together. `accept` records an explicit inventory baseline. It does not perform semantic review for you.

## Files to inspect

| Path | Purpose |
| --- | --- |
| `upstream.lock.json` | Source repository, exact commit, version, file hashes, and modes |
| `internal/bundle/manifest.json` | Embedded copy of the lock |
| `internal/bundle/_upstream/` | Pristine source data, excluded from Go package discovery |
| `internal/bundle/review-report.json` | Changed files, lexical delegation candidates, and role candidates |
| `internal/bundle/compatibility.json` | Explicitly accepted manifest hash |

The inventory covers imported files and scans UTF-8 text for known terms. It includes false positives and can miss unfamiliar wording. File integrity, lexical classification, and semantic workflow compatibility are different checks. Review unsupported requirements and new delegation forms even when the report has no new terms.

## Automated proposals

[The updater workflow](../.github/workflows/upstream.yml) runs weekly or by manual dispatch on the default branch. It clones into the runner's temporary directory, validates the trusted baseline, imports candidate source as data, and proposes a draft PR when the file inventory changes. It never accepts compatibility or merges a proposal.

The updater checks candidate build and vet after import. Candidate runtime tests and installer acceptance still require review. PRs created with `GITHUB_TOKEN` do not automatically trigger ordinary push or pull-request workflows. Dispatch CI on the candidate branch, or push the reviewed acceptance commit with maintainer credentials, then require passing candidate checks before merging.

Package creation produces local archives and checksums. It does not publish a release, upload assets, or change a package registry. See [contribution checks](../CONTRIBUTING.md).
