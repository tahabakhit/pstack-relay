# Changelog

## 1.0.0

- Ship pstack as a plugin for Codex and Claude Code.
- Use one shared sheet for each role's harness, model, and effort.
- Resolve delegation to native subagents, Orca workers, or interactive Rex workers according to the parent and detected environment.
- Remove the 0.1 worker sandbox lockdown, MCP server, and Rex supervisor. Workers use their harness's normal configuration, tools, and approval settings.
- Keep Cursor's upstream pstack source pristine and generate host delegation pointers and setup adaptations separately.
- Add scheduled upstream sync with validation and an auto-merge request. The first hosted auto-merge run remains unverified.
- Discover Pi models for the sheet. Pi plugin installation, Pi on Orca, and herdr remain after-1.0 work.
