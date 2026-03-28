### Observation Infrastructure

If the system can't observe it, the system can't manage it.

**The test:** "Is there a state transition that doesn't emit an event? A metric that counts events instead of entities? A failure mode that appears healthy?"

**What this means:**

- Every state transition should emit an event (spawn, progress, completion, death)
- Metrics should be deduplicated by entity, not counted by event
- Dashboards should be the single source of truth (if you need CLI/logs to understand state, the dashboard failed)
- Default to visible, not hidden (false positives are better than false negatives)
- Observation gaps are P1 bugs (invisible failures erode trust faster than visible ones)

**What this rejects:**

- "The metrics show 72% completion" when reality is 89% (measurement artifact)
- Agents appearing "dead" when actually complete (state not surfaced)
- Work completing via paths that bypass event emission (`bd close` without `orch complete`)
- Silent RPC failures that cause dashboards to show stale state

**The failure mode:** Trust erosion loop. Dashboard shows unexpected state → investigate → find metrics bug → fix → repeat. Each investigation is wasted effort on a measurement artifact, not a real problem. Eventually you stop trusting the dashboard and the observation infrastructure becomes useless.
