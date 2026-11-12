# Pi read-only command-boundary experiment

Status is an independently reviewed subset with explicit limits. This disposable experiment advances the [selected worker design](worker-execution-design.md). It does not admit production Pi shell or writing, certify the full acceptance matrix or establish 1.0.0 readiness.

## Instrument and observations

Actual Pi 1.5.2 received fixed tool calls from a local synthetic Messages endpoint. Pi started in a neutral control cwd with a generated shell wrapper and explicit prompt inputs. The wrapper invoked an inner bubblewrap boundary with read-only source and system runtime mounts, writable owned scratch, private process/network views, a separately constructed environment and a handcrafted x86_64 seccomp filter. The outer envelope masked real home, runtime sockets and temporary files. No real credentials or provider inference were involved.

The ordered actual tool exits were 0 for source-read/scratch-write, 1 for a real assertion failure, 0 for the denial-observation script, 159 for a foreign-architecture syscall probe, and 0 for the descendant setup probe. Native tool IDs and exact sequential wrapper commands correlated. The [evidence index](validation/pi-command-boundary-prototype.json) records raw locators, hashes and the author verifier result.

Selected source writes failed with EROFS. Private, sibling and escaping-symlink reads failed. Loopback, filesystem Unix, abstract Unix and VSOCK socket creation returned EPERM. Parent listeners for the first three accepted allowed positive-control messages and no worker connection. VSOCK had no positive-connectivity service, so only denied creation is established.

The final visible process environments contained no unusual synthetic secret. Process namespace and command identity checks distinguished inner processes from the outer parent. Numeric PID alone was an invalid initial oracle because different namespaces can reuse the same number. Retained descriptor observations found no synthetic secret file, but the wrapper's opened descriptor was noninheritable. This is measured absence, not an independent test of the inheritable-descriptor closure branch.

The x86_64 filter rejects a foreign architecture and x32-tagged syscall. It explicitly denies socket, selected namespace/cross-process and io_uring operations. The retained probes exercise x32, an int-0x80 foreign-architecture call and io_uring calls. `clone3` has a static ENOSYS rule but was not invoked by the task probe. This denylist is not an exhaustive production-reviewed syscall policy.

Missing wrapper, missing boundary executable and invalid seccomp initialization produced tool errors. Pi still exited zero after the synthetic final response. The initialization file marker uses an absolute path unavailable inside a healthy inner namespace, which weakens that marker as an independent fallback oracle. An independent reviewer subsequently ran the actual frozen wrapper with a fixed stdout command. Healthy initialization printed the marker, while invalid seccomp and missing boundary produced their exact setup errors and no marker. That strengthens those two initialization cases. The stronger native Pi missing-wrapper suppression check remains open.

A setsid/double-fork child wrote its start marker and no delayed marker after normal command completion and a 2.5-second parent observation. The same logic without the inner boundary produced the delayed marker. This is a bounded normal-completion observation, not proof of cancellation, parent death or arbitrary lifetime.

## Defect found and corrected

The original wrapper exposed a synthetic secret through a helper's process environment despite clearing the final command environment. The focused negative control reproduced it. Passing an explicit environment before bubblewrap creation corrected the observed leak. [PI-002](issues/pi-helper-environment.md) preserves the cause, correction and production requirement. The failing original evidence remains retained.

## Next checks and retention

An independent reviewer configured as `gpt-6.1-sol` checked the filter, environment correction, oracles and retained evidence. It confirmed the selected subset, independently interpreted the saved BPF instructions, and performed the fresh initialization controls described above. It also demonstrated why the original FD fixture cannot attribute closure to `close_fds`. The report and exact independent check bodies are preserved with the evidence index. These checks are not a second full Pi execution replay.

Next test inherited descriptors with reachable positive controls, cancellation/death, isolated writers, complete startup controls, runtime replacement, authentication and installed Rex execution. Current production gates remain closed.

Scratch source is `/tmp/pstack-rex-pi-boundary.slgiZyDg`. Selected scripts, raw results, failing controls and fixture observations are preserved locally under `.git/pstack/experiments/pi-command-boundary-2026-11-12`. `verify.py` checks retained observations offline. It is not a fresh execution run. These artifacts do not arrive in a clone.
