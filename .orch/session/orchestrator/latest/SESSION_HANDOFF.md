# Session Handoff: Trust Recovery & Verification

**Mandate:** STRATEGIC ORCHESTRATOR ONLY
**Goal:** Verify the Nervous System (coaching plugin) after the caching fix.

## State of Play
- Fix for detectWorkerSession caching bug implemented and merged.
- Swarm cleaned (70+ stale sessions deleted).
- Infrastructure rebooted (OpenCode server and daemon restarted).

## Mandatory Verification Protocol
1. **Worker Evidence:** Spawn a worker and verify it emits worker-specific health metrics (tool_failure_rate, context_usage).
2. **Orchestrator Evidence:** Red Team test - deliberately violate frame (edit code) and verify coaching injection fires.

**Constraint:** DO NOT implement. DO NOT read code. Spawn agents for all implementation/investigation.
