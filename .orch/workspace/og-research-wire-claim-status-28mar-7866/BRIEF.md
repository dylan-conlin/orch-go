# Brief: orch-go-og67s

## Frame

Every session starts with `orch orient`, which shows what's active, what's ready, and what changed. But there was a blind spot: 9 models carry testable claims, some untested, and orient had no aggregate view of claim coverage. You'd see a single "unconfirmed core" edge but no way to know that architectural-enforcement had 3 untested core claims while harness-engineering had 1. The research cycle design (orch-go-47ppm) identified this as step 4: close the visibility loop so orient shows gaps, the orchestrator can trigger research, and the next orient shows updated status.

## Resolution

Three functions added to `pkg/claims/` and wired through orient. `CollectClaimStatus` scans all claims.yaml files and returns per-model summaries — but only for models that have gaps (all-confirmed models are silent). `CollectRecentDisconfirmations` finds claims with `contradicts` evidence in the last 7 days — these are the actively-contested claims that might need attention. Both feed into `FormatClaimSurface`, which produces a unified "Knowledge Edges" section combining the new summaries with existing tension and staleness edges.

The surprise was how little code was needed. The claims infrastructure was already carrying all the data — confidence levels, evidence verdicts with dates, staleness tracking. The gap was purely a formatting layer. The whole change is ~120 lines of logic and 9 tests. No new types on OrientationData, no new CLI flags — just richer content in the existing ClaimEdges string.

## Tension

I deferred "claims with pending probes (spawned but not complete)" — the third section the task asked for. The problem: nothing currently links a spawned probe agent to a specific claim ID. The Evidence struct records completed probes, not in-flight ones. A convention like beads labels (`claim:HE-08`) on probe issues would solve this, but that's a cross-component decision that reaches beyond implementation scope. Whether this gap matters depends on how often research gets duplicated — worth watching over the next few sessions.
