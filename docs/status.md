# Release status

The current implementation is a Claude Code and Codex marketplace plugin named `pstack`, distributed by pstack-relay. Its runtime is the bundled POSIX sh script. Native, Orca, Rex, and hc seats share one model sheet. Go is used only for repository tools and tests.

## Acceptance remains separate from implementation

The [acceptance record](acceptance.md) separates first-release live runs, current local installation checks, and pending backend runs. Earlier Orca and Rex results do not establish current release acceptance.

Current local checks cover race tests, vet, generated plugin bytes and permissions, shell lint, Claude manifest validation, packaged execution, and isolated local marketplace installation. The acceptance record contains the actual commands and limits.

## Known limits

- Marketplace-only installation from the public repository in both harnesses remains pending for this release.
- Live Orca, Rex, and hc seats remain pending for this release.
- Codex ships no session hook. Skill entrypoints direct agents to the delegation reference.
- Interactive workers can wait for approval under the user's normal harness settings.
- Model discovery does not prove account access, served model identity, or effective reasoning effort.
- Parent verification is required for worker reports. Independent review stays blocked when no reviewer can run.
- Cursor cloud agents, automations, and project stores have no host equivalent.
- Pi plugin installation and Pi parent acceptance remain pending. Pi seats on Orca fall back because Orca has no Pi agent.
- The current hosted upstream sync has no acceptance result recorded here.
- Codex CLI accepts versionless repository marketplace plugins. OpenAI public-directory submission has separate version requirements, as [upstream maintenance](upstream.md#follow-upstream-updates-without-a-version-pin) explains.
