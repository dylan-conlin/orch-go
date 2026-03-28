# Brief: orch-go-0d3th

## Frame

The dashboard said zero agents were running while four were visibly working in tmux windows. Dylan couldn't trust the dashboard as his monitoring surface — the number it showed didn't match reality. The suspicion was that the dashboard's data source was disconnected from orch serve's actual state.

## Resolution

The data source was fine — the same `queryTrackedAgents` function feeds both `orch status` and the API. The disconnect was in how "idle" got interpreted at two different layers.

When an OpenCode session is alive but between requests, it reports status "idle." The discovery layer treated this as `SessionDead = true` — a flag that means "session is gone." The API adapter saw that flag and overrode the status to "dead." So agents that were perfectly healthy but momentarily not processing appeared dead in the dashboard. Then the frontend only counted `status === 'active'` agents, which excluded both the falsely-dead ones and any legitimately idle ones.

The fix was surgical: stop marking idle sessions as dead (they're alive, just not busy), and widen the frontend counter to include idle agents alongside active ones. This matches how `orch status` counts — both processing and idle agents are "active" in the swarm.

## Tension

The `SessionDead` flag has a semantic identity crisis. The original decision doc defines it as "session exists but idle/errored" while the code that consumes it treats it as "session is dead." I fixed the specific case (idle ≠ dead), but the no-liveness case (session not found in the status map) still sets `SessionDead = true` with `Status = "idle"` — which is a misleading combination. That path wasn't causing the reported bug, but it's the same kind of semantic confusion waiting to bite again.
