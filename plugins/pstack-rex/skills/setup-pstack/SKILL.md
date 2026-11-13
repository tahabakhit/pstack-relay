---
name: setup-pstack
description: Configure pstack harness, model, and reasoning effort per role in the shared sheet. Run pstack-rex setup and pstack-rex models. Use for setup-pstack, model configuration, or role budget changes.
---

> Read [pstack delegation on this host](../../delegation.md) before following this file. It maps Cursor tools and models to this host and says how to start every subagent.


# Setup pstack

Configure the shared pstack model sheet. Run `pstack-rex setup` to discover installed harnesses and create the sheet, and `pstack-rex models` to print it. The delegation reference maps each role to native subagents or a worker on the detected backend.

## Steps

1. Run `pstack-rex setup --dry-run` to inspect the installed harnesses, available model catalogs, starter sheet, and installation commands. Run `pstack-rex setup` to install the plugin and create the sheet if absent. Existing sheets are preserved.
2. Run `pstack-rex models` and read the shared sheet at `${PSTACK_REX_SHEET:-${XDG_CONFIG_HOME:-$HOME/.config}/pstack-rex/pstack-models.md}`. Ask which harness, model, and effort the user wants for each role. Preserve explicit preferences and panel order. A catalog lists models; it does not prove account access.
3. Edit the shared sheet, or run `pstack-rex models --set 'ROLE=HARNESS:MODEL @EFFORT'`. Use a comma-separated value for a panel; each entry is one seat. Bare models run on the parent's harness. `inherit-parent` and `auto` use the parent's model. `default effort: session` omits an explicit effort, and `session hook: off` disables the session reminder.
4. Run `pstack-rex models` to verify the saved choices. Report the sheet path and assignments. Before delegating, run `pstack-rex resolve --role 'ROLE'` and follow each placement.

### 7. Offer a verification skill (optional)

Check whether the project has a way to drive the real app for proof (a `verify-*` skill, or an existing harness). If not, offer once: "want a project-local verification skill, so agents can drive the app the way a user does and prove changes work? I can generate one with /create-verification-skill." On yes, invoke `/create-verification-skill` (resolves wherever pstack is installed: workspace, user, or plugin). On no, move on without pushing.

### 8. Offer a formal verification skill (optional)

Check whether the project already has a formal verification skill (a project skill that runs a model checker, a proof, or another verifier against the code). If not, offer once: "want a formal verification skill, so agents can prove a key property of your code holds in every case in scope, not only the cases tests reach? I can generate one with /create-formal-verification-skill." On yes, invoke `/create-formal-verification-skill` (resolves wherever pstack is installed: workspace, user, or plugin). On no, move on without pushing.
