# Why pstack-rex uses a thin adapter

pstack-rex keeps Cursor pstack as its workflow source and owns a small Go boundary for configuration, Rex lifecycle, and Codex, Claude, and Pi output parsing. Importing another fork's runner would also import a second model policy and maintenance path. This is a design choice based on the source comparison below, not a measured implementation-cost claim.

## Source comparison

The investigation examined these exact revisions. Links identify the source behind each finding.

| Repository and examined commit | Useful pattern | Cost or limit for this integration |
| --- | --- | --- |
| [michael-denyer/pstack-claude](https://github.com/michael-denyer/pstack-claude/tree/36f43364d9198e951f2b739bfb007eb747f834b7), `36f43364d9198e951f2b739bfb007eb747f834b7` | [Three-way sync](https://github.com/michael-denyer/pstack-claude/blob/36f43364d9198e951f2b739bfb007eb747f834b7/tools/sync.mjs#L373-L394) compares old upstream, new upstream, and local content. | [Generation](https://github.com/michael-denyer/pstack-claude/blob/36f43364d9198e951f2b739bfb007eb747f834b7/tools/generate.mjs#L1-L37) stamps its own model policy and runtime mappings. Reusing it entails that fork's packaging and policy. |
| [ericlitman/open-pstack](https://github.com/ericlitman/open-pstack/tree/faf512125b11f4a30f55c4eaf8a709a890668406), `faf512125b11f4a30f55c4eaf8a709a890668406` | [ProviderAdapter](https://github.com/ericlitman/open-pstack/blob/faf512125b11f4a30f55c4eaf8a709a890668406/plugins/pstack/skills/poteto-mode/scripts/runner/registry/contract.ts#L117-L149) groups argv, permissions, parsing, and reported-model proof. | Its "provider" is a CLI harness. Rex execution belongs outside that abstraction. The contract does not itself supply Rex lifecycle or Pi support. |
| [arjitj2/open-pstack](https://github.com/arjitj2/open-pstack/tree/264230bc14d9055d1333804799e7187e8e6e135c), `264230bc14d9055d1333804799e7187e8e6e135c` | [Maintenance ledger](https://github.com/arjitj2/open-pstack/blob/264230bc14d9055d1333804799e7187e8e6e135c/scripts/fork-maintenance.py#L114-L155) requires reason and evidence for completed decisions. | [Model policy](https://github.com/arjitj2/open-pstack/blob/264230bc14d9055d1333804799e7187e8e6e135c/plugins/pstack/skills/poteto-mode/scripts/model-policy/model-policy.ts#L93-L118) adds ordered attempts, seat rows, fallback, and continuation policy. That exceeds the initial routing requirement. |
| [creedants/pstack-portable](https://github.com/creedants/pstack-portable/tree/0172a1e3be9cb60ff237d51b7aca8931263c2d97), `0172a1e3be9cb60ff237d51b7aca8931263c2d97` | [Build overlays](https://github.com/creedants/pstack-portable/blob/0172a1e3be9cb60ff237d51b7aca8931263c2d97/scripts/build.py#L61-L87) separate source from installed output. | [Workflow rewrites](https://github.com/creedants/pstack-portable/blob/0172a1e3be9cb60ff237d51b7aca8931263c2d97/scripts/build.py#L94-L115) replace panel and orchestration instructions. They cannot establish unchanged Cursor workflow semantics. |

Sync decides how source changes enter a fork. Generation derives installable files. Neither operation proves that a delegation request ran, that its model was served, or that its descendants stopped. Those facts require runtime receipts and harness-specific parsing.

## Canonical Cursor source

The baseline is [cursor/plugins at `a5539b15a3d56e04992565aafee2b37051329ce3`](https://github.com/cursor/plugins/tree/a5539b15a3d56e04992565aafee2b37051329ce3/pstack), pstack 0.15.36. Its [setup model sheet](https://github.com/cursor/plugins/blob/a5539b15a3d56e04992565aafee2b37051329ce3/pstack/skills/setup-pstack/SKILL.md#L41-L66) has 17 role rows, including `hardest tasks`. Other forks' role names and model defaults are separate policies.

That sheet distinguishes panel entries from a cross-judge pool. `auto` and `inherit-parent` select the parent model and still occupy panel seats. They do not mean "choose an execution backend." An external worker needs a concrete harness and model selection. Model identifiers in examples describe a configured installation, not universal defaults or proof of entitlement.

The [Cursor harness generator](https://github.com/cursor/plugins/blob/a5539b15a3d56e04992565aafee2b37051329ce3/scripts/generate-pstack-harness-trees.mjs#L118-L176) copies source and checks generated drift. pstack-rex maintains its own pinned import and runtime contract. Its lexical inventory helps review changed delegation instructions, but does not certify every instruction's meaning. See [upstream review](upstream.md).

## Harness and Rex responsibilities

Configuration separates the worker harness, model, effort and execution backend. The initial parent is Codex. Worker permission profiles are validated separately. Rex manages the process and terminal. Each worker harness owns its invocation syntax and output protocol.

[Codex noninteractive mode](https://learn.chatgpt.com/docs/non-interactive-mode#permissions-and-safety) documents explicit sandbox modes. [Claude's result and init types](https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage) supply structured completion and model evidence, but [ordinary init messages omit effort](https://code.claude.com/docs/en/agent-sdk/typescript#sdksystemmessage). Pi's [JSON stream](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/json.md#event-sequence) distinguishes `agent_end` from session-level `agent_settled`. A process exit or an accepted request alone cannot prove successful work.

Pi's [security guidance](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/security.md) says the working directory does not confine filesystem access. Tool restrictions on Pi and Claude must therefore be reported separately from OS sandbox enforcement. Requested model and effort are distinct from provider-reported values. Missing effective values, authentication readiness, and funding evidence remain unknown.

Native parent tools belong to the active host. A generated instruction can route cooperative delegation through pstack-rex, but cannot intercept a host's built-in agent API. The one-shot workers here are headless processes shown in Rex panes. Their visibility does not provide interactive steering or resume.

The graphical client [Rex Rune at `ad99bed2c5138cc32b370bbf69a254fe6e04e3a5`](https://github.com/shuv1337/rex-rune/blob/ad99bed2c5138cc32b370bbf69a254fe6e04e3a5/README.md#L38) documents OSC 7501 status words on tabs and panes. This supports the status-display design. Source inspection and Rex CLI execution do not constitute a graphical Rune validation.

The decision would favor broader runner reuse if direct CLI backends or its wider harness set became requirements. For the current Codex, Claude, and Pi scope, owning the small Rex boundary avoids importing unrelated fallback and model policies. It still leaves output-parser maintenance and semantic upstream review with this project.
