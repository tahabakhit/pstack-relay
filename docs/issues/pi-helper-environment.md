# PI-002: Prototype helper retained synthetic environment secrets

Status is corrected for the observed disposable experiment, pending production design and verification. This was found before enabling Pi shell capability. It is not evidence of an exposure in the current production source-only profile.

## Reproduction

The first inner bubblewrap wrapper used `--clearenv` for the task command but inherited the trusted Pi process environment when starting bubblewrap itself. The actual Pi model-directed tool path ran a process-environment probe. The task's own environment was clear, while its scan of visible process environments found the unusual synthetic secret. That original record retains the failing boolean observation, not a PID attribution.

The original evidence is `/tmp/pstack-rex-pi-boundary.slgiZyDg/fixture-v1-proc-leak/stdout.jsonl`, tool result `tool_prototype_2`, and its `scratch/denials.json`. The focused `fixture-leak-repro` result attributes the synthetic value to `/proc/1/environ` and distinguishes it from the task environment. `boundary-v1.py` preserves the initial launcher. [The evidence index](../validation/pi-command-boundary-prototype.json) records hashes and retained locations.

The actual values were synthetic. The outer envelope masked real home, runtime sockets and temporary files. No real credentials or provider inference were involved.

## Correction

`boundary.py`, line 29 in that scratch directory, supplies a separately constructed `PATH` and `LANG` environment before starting bubblewrap. It retains `--clearenv` for the eventual command. The corrected actual Pi run found no synthetic secret in any visible process environment. Clearing only the final command environment was insufficient for the declared boundary.

Retain the initial failing case as a negative control. Production implementation must construct the helper environment before process creation and cover every visible helper, not only the final shell. Do not pass provider credentials, proxy values or parent identities to the boundary process.

## Remaining evidence

The [read-only boundary experiment](../pi-command-boundary-prototype.md) is a selected subset. Descriptor inheritance needs a distinct control because Python's ordinary opened descriptor is noninheritable. Cancellation, supervisor death, real authentication, isolated writing, runtime replacement resistance and the complete matrix remain open. No production capability is admitted by this correction.
