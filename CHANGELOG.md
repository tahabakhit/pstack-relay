# Changelog

## 2.0.0

- Rename the repository and marketplace from pstack-rex to pstack-relay. Keep upstream's `pstack` plugin and skill names.
- Install through Claude Code and Codex marketplaces. Bundle a POSIX sh runtime instead of requiring a Go CLI on `PATH`.
- Run setup through `/setup-pstack` in Claude Code or `$setup-pstack` in Codex. Discover models, write one shared sheet, and install into the other detected harness.
- Copy the old model sheet automatically when the new sheet is absent. Use `PSTACK_RELAY_SHEET` and the pstack-relay configuration directory.
- Add herdr-core (`hc`) seats alongside native, Orca, and Rex delegation.
- Keep upstream source unchanged, with generated delegation pointers and explicit host adaptations. Bundle the pristine setup template as runtime data.
- Omit fixed manifest versions so repository marketplace updates follow synced contents without manual version bumps.
- Build Go repository tools and tests only. Package one marketplace tree, with no user-facing Go executable archives.
- Keep Claude hooks under `claude-hooks/`. Codex has no hook directory.
- Leave live backend and public marketplace acceptance pending until recorded in `docs/acceptance.md`.

## 1.0.0

- Ship pstack as a plugin for Codex and Claude Code.
- Use one shared sheet for each role's harness, model, and effort.
- Resolve delegation to native subagents, Orca workers, or interactive Rex workers according to the parent and detected environment.
- Remove the 0.1 worker sandbox lockdown, MCP server, and Rex supervisor. Workers use their harness's normal configuration, tools, and approval settings.
- Keep Cursor's upstream pstack source pristine and generate host delegation pointers and setup adaptations separately.
- Add a daily upstream sync that merges the pull request when the plugin build, tests and vet pass, and labels it `upstream-sync-failed` otherwise. The first hosted run has not happened yet.
- Discover Pi models for the sheet. Pi plugin installation, Pi on Orca, and herdr remain after-1.0 work.
