# PI-001: Pi system prompts remain eligible under the controlled profile

Status is open. Scope is customization isolation, not a demonstrated command or OS permission escape. Detected on Pi `1.5.2` while investigating the shell capability milestone on 12 November 2026.

## Observed behavior

[The adapter](../../internal/harness/harness.go), line 70, disables context files, extensions, MCP, skills, and prompt templates. It does not explicitly select system-prompt content or project-trust policy. Describing this as disabling every customization is too broad.

Installed `@earendil-works/pi-coding-agent/dist/core/resource-loader.js`, lines 481 to 505, clears context files independently of system-prompt discovery. Lines 959 to 979 select trusted project prompt files or agent-directory prompt files. The package lives under `/home/taha/.pi/agent/install/releases/1.5.2/node_modules/` on the investigation host.

A direct loader probe used fresh temporary cwd and agent directories. It enabled the same resource-disable options as the adapter and supplied a synthetic SettingsManager trust state. The root agent reran it and read the returned values. No model, authentication store, or inference call was involved.

| Project trust supplied to loader | System prompt | Appended prompt | Context files |
| --- | --- | --- | --- |
| false | `AGENT_SYSTEM_CANARY` | Empty | Empty |
| true | `PROJECT_SYSTEM_CANARY` | `PROJECT_APPEND_CANARY` | Empty |

This proves residual loading at the resource-loader boundary. It does not prove that a real worker on this host selects either trust state, or that any loaded prompt escapes its enabled tool list. [Pi's security guidance](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/security.md#understand-project-trust) distinguishes project trust from tool permissions.

## Reproduction recipe

1. Import `DefaultResourceLoader` and `SettingsManager` from the installed package's `dist/core` modules.
2. Create a fresh cwd with `.pi/SYSTEM.md`, `.pi/APPEND_SYSTEM.md`, and `AGENTS.md`, each containing a distinct canary. Create a separate agent directory containing its own `SYSTEM.md` canary.
3. Construct SettingsManager with those directories and `projectTrusted: false`.
4. Construct the loader with that manager, `noExtensions`, `noSkills`, `noContextFiles`, and `noPromptTemplates` set to true. Set `disabledBuiltinExtensions` to `['mcp']`.
5. Await `reload()` and read `getSystemPrompt()`, `getAppendSystemPrompt()`, and `getAgentsFiles()`.
6. Repeat with `projectTrusted: true`. Compare the actual returned strings with the table above.

The original probe script is `/tmp/pstack-rex-shell-pi/resource-canary.mjs`. Its location is temporary. [The observation record](../validation/shell-controls.json) preserves the outputs and source locators.

## Smallest adequate follow-up

Specify which Pi startup inputs remain trusted. Then explicitly isolate or override prompt discovery and project settings where installed controls permit it. Test this through the actual CLI startup path, including the existing RPC selection preflight, before claiming customization isolation. Keep the current shell gate closed. Fixing this prompt-loading gap alone would not make built-in bash bounded.
