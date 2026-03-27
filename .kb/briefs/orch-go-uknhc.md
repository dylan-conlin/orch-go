# Brief: orch-go-uknhc

## Frame

When you abandon a freshly-spawned agent — one that hasn't had time to report its first phase comment — the system is supposed to warn you: "this agent just started, it might still be booting up." That warning never fires. Instead, `orch abandon` quietly concludes the agent is dead and kills it without hesitation. The prior investigation (orch-go-z1pkh) confirmed the liveness function itself is fine; the bug is that `abandon` never tells the liveness function when the agent was born.

## Resolution

The root cause is a missing argument. `VerifyLiveness` takes a `SpawnTime` parameter that enables a 5-minute grace window for new agents, but `abandon_cmd.go` calls it with `SpawnTime` left at zero. Zero means "unknown," which means "skip the grace period," which means "no phase comment = dead." Meanwhile, `orch complete` already passes spawn time correctly — it reads it from the agent's workspace manifest. The fix is to do the same thing in `abandon`: discover the workspace earlier in the flow (it's just a disk scan), read the spawn time from the manifest, and pass it through. One file changes, following a pattern that's already proven. This is a textbook Class 7 defect (Premature Destruction — killing something based on incomplete information). Implementation issue created as orch-go-ehy5q.

## Tension

The fix only works when the workspace exists and the manifest is readable. If `findWorkspaceByBeadsID` returns nothing (timing edge case, cross-project, pre-manifest legacy), the spawn time stays at zero and the grace period still doesn't fire. The current behavior becomes the fallback rather than the default. Is that acceptable as-is, or should there be a second fallback — like using the beads issue creation timestamp as a proxy for spawn time?
