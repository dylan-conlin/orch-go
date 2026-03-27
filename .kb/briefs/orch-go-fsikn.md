# Brief: orch-go-fsikn

## Frame

This looked simple on the surface: there is a file called `VERIFICATION_SPEC.yaml`, workers are told to fill it out, and completion messaging treats it like proof. The question was whether that proof is actually machine-checked, or whether everyone has been treating a ritual artifact as if it were part of the gate.

## Resolution

The turn was realizing there are really two stories in the codebase. Spawn has a very concrete story: `pkg/spawn/verification_spec.go` knows how to generate a typed skeleton with fields for commands, working directory, timeout, and expected output. Completion tells a different story. Once I traced `orch complete` from `pkg/verify/check.go`, the file basically vanished. The actual gate path checks phase comments, synthesis presence, and `COMPLETION.yaml`-style artifact data, and investigation work gets a scan-tier exemption that skips even that artifact gate.

So the file matters, but not in the way its name suggests. Right now it is evidence for a human reader, not a contract the verifier enforces. That explains why multiple schemas can exist in the repo without anything breaking: the system is asking people to supply proof, but it is not yet reading that proof as code.

## Tension

The unresolved question is whether that looseness is intentional or just unfinished migration. If Dylan wants `VERIFICATION_SPEC.yaml` to actually shape completion behavior, the next move is not a small fix - it is choosing one canonical schema and deciding whether `orch complete` or some other tool owns enforcement.
