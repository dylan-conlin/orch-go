# Brief: orch-go-ehy5q

## Frame

When you run `orch abandon` on an agent that just spawned -- one that hasn't had time to report its first "Phase: Planning" comment yet -- the system was supposed to warn you. It didn't. It classified the agent as dead and let the abandon proceed without a word. The liveness function had a grace period for exactly this case, but the abandon command never told it when the agent was born.

## Resolution

The fix was pure plumbing. `VerifyLiveness` already has a `SpawnTime` field and a 5-minute grace period -- `orch complete` uses it correctly at line 275. `orch abandon` just forgot to pass it. The workspace lookup that reads the agent manifest was happening *after* the activity check instead of before it, so the spawn time was never available when the liveness decision was made.

I hoisted the workspace discovery before the activity check, passed the workspace path into `checkRecentActivity`, and read the spawn time from the manifest. Six lines of real change. The redundant second workspace lookup in Phase 3 got cleaned up as a side effect. Three tests confirm the integration: recent spawn triggers the grace period, old spawn doesn't, empty path falls back safely.

## Tension

The fallback behavior when no workspace exists (pre-manifest agents, cross-project) is that `SpawnTime` stays zero and the grace period doesn't fire. This means a recently-spawned agent whose workspace hasn't been written yet can still be falsely abandoned. The window is small (manifest is written during spawn Phase 1, before the agent starts), but it's there. If this ever matters, a fallback to the beads issue creation timestamp would close the gap.
