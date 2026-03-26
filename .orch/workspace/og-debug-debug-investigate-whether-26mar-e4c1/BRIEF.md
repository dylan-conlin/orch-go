# Brief: orch-go-o5uih

## Frame

The question looked narrower than it was: can the stall tracker tell the difference between an agent that's just slow and one that's actually hung? That matters because the dashboard and `orch status` turn this judgment into operator trust - if "stalled" is wrong, Dylan either ignores a real hang or loses confidence in the signal entirely.

## Resolution

The turn was that the bug is simpler and more structural than a slow-versus-stalled edge case. `pkg/daemon/stall_tracker.go` rewrites its timestamp every time it sees the same token count, so it never remembers when progress last happened; it only remembers when the previous poll happened. Once I lined that up with the dashboard's 30 second polling cadence, the behavior snapped into focus: a truly stuck agent can be checked forever and still never accumulate the 3 minutes needed to look stalled.

That means the current implementation is not really making a nuanced distinction between slow and stalled agents. It is mostly measuring poll spacing, then accidentally calling that liveness. I stopped at evidence, a KB constraint, and an architect follow-up because this area is marked as a hotspot, so the next move should be to design the right state model and regression coverage before changing behavior.

## Tension

The open design question is whether "slow but alive" should become a first-class state instead of being inferred from the absence of a stall. If Dylan wants that distinction surfaced in the UI, the next design should decide what signals count as meaningful progress beyond raw token increases.
