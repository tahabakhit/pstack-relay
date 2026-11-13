---
name: setup-pstack
description: Configure pstack harness, model, and reasoning effort per role in the shared sheet. Run this plugin's bin/pstack-relay setup and models commands. Use for setup-pstack, model configuration, or role budget changes.
---

> Read [pstack delegation on this host](../../delegation.md) before following this file. It maps Cursor tools and models to this host and says how to start every subagent.


# Setup pstack

Configure the shared pstack model sheet. The plugin root is the directory that holds `delegation.md`. Run its `bin/pstack-relay` script by absolute path.

## Steps

1. Run `<plugin root>/bin/pstack-relay setup --dry-run` to inspect installed harnesses, model catalogs, the starter sheet, and installation commands. Show the dry-run output to the user and ask them to confirm the sheet and installation commands. After they confirm, run `<plugin root>/bin/pstack-relay setup --yes` to create the sheet and install the plugin into detected harnesses. Existing sheets are preserved.
2. Run `<plugin root>/bin/pstack-relay models` and read `${PSTACK_RELAY_SHEET:-${XDG_CONFIG_HOME:-$HOME/.config}/pstack-relay/pstack-models.md}`. Ask which harness, model, and effort the user wants for each role. Preserve explicit preferences and panel order. A catalog lists models; it does not prove account access.
3. Make changes with `<plugin root>/bin/pstack-relay models --set 'ROLE=HARNESS:MODEL @EFFORT'`. Use a comma-separated value for a panel. Each entry is one seat. Bare models run on the parent's harness. `inherit-parent` and `auto` use the parent's model. `default effort: session` omits an explicit effort. `session hook: off` disables the Claude session reminder.
4. Run `<plugin root>/bin/pstack-relay models` to verify the saved choices. Report the sheet path and assignments. Before delegating, run `<plugin root>/bin/pstack-relay resolve --role 'ROLE'` and follow each placement.

### 7. Offer a verification skill (optional)

Check whether the project has a way to drive the real app for proof (a `verify-*` skill, or an existing harness). If not, offer once: "want a project-local verification skill, so agents can drive the app the way a user does and prove changes work? I can generate one with /create-verification-skill." On yes, invoke `/create-verification-skill` (resolves wherever pstack is installed: workspace, user, or plugin). On no, move on without pushing.

### 8. Offer a formal verification skill (optional)

Check whether the project already has a formal verification skill (a project skill that runs a model checker, a proof, or another verifier against the code). If not, offer once: "want a formal verification skill, so agents can prove a key property of your code holds in every case in scope, not only the cases tests reach? I can generate one with /create-formal-verification-skill." On yes, invoke `/create-formal-verification-skill` (resolves wherever pstack is installed: workspace, user, or plugin). On no, move on without pushing.
