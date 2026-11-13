# Changelog

## 1.0.0

- Ship pstack as a plugin for Codex and Claude Code.
- Use one shared sheet for each role's harness, model, and effort.
- Resolve delegation to native subagents, Orca workers, or interactive Rex workers according to the parent and detected environment.
- Remove the 0.1 worker sandbox lockdown, MCP server, and Rex supervisor. Workers use their harness's normal configuration, tools, and approval settings.
- Keep Cursor's upstream pstack source pristine and generate host delegation pointers and setup adaptations separately.
- Add a daily upstream sync that merges the pull request when the plugin build, tests and vet pass, and labels it `upstream-sync-failed` otherwise. The first hosted run has not happened yet.
- Discover Pi models for the sheet. Pi plugin installation, Pi on Orca, and herdr remain after-1.0 work.
