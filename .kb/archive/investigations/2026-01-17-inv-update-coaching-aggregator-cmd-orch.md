<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented dashboard observability for worker health metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) emitted by coaching.ts plugin.

**Evidence:** API returns aggregated worker health with health_status (good/warning/critical) - verified: curl test with injected metrics returned correct aggregation.

**Knowledge:** Worker health is keyed by session_id, allowing dashboard to match agents to their health metrics. Thresholds mirror coaching.ts: 3/5 failures, 80/90% context, 15/30m phase, 30/60m commit.

**Next:** Complete - feature ready for production use.

**Promote to Decision:** recommend-no - Tactical dashboard feature, no new architectural pattern.

---

# Investigation: Update Coaching Aggregator Cmd Orch

**Question:** How to surface worker health metrics in the orch dashboard for real-time agent monitoring?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-update-coaching-aggregator-17jan-fd2f
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Worker Health Metrics Already Emitted by coaching.ts

**Evidence:** The coaching.ts plugin (lines 1007-1120) emits 4 worker-specific metrics to `~/.orch/coaching-metrics.jsonl`:
- `tool_failure_rate`: consecutive failures (emits at >=3)
- `context_usage`: estimated token % (emits at >=80% or every 50 tool calls)
- `time_in_phase`: minutes since session start (emits every 30 tool calls when >5m)
- `commit_gap`: minutes since last git commit (emits every 30 tool calls when >10m)

**Source:** `plugins/coaching.ts:1029-1119`, `.kb/investigations/2026-01-17-inv-add-worker-specific-metrics-plugins.md`

**Significance:** Backend emissions are complete; this investigation focuses on aggregation and display.

---

### Finding 2: Existing Aggregator Only Handled Orchestrator Metrics

**Evidence:** serve_coaching.go:79-168 aggregated only `action_ratio` and `analysis_paralysis` metrics. Worker health metrics were present in JSONL but ignored by the API.

**Source:** `cmd/orch/serve_coaching.go:79-168` (original implementation)

**Significance:** Required adding WorkerHealthMetrics struct and separate aggregation logic for worker sessions.

---

### Finding 3: Agent Cards Display Health via session_id Matching

**Evidence:** Agent interface has `session_id` field (agents.ts:32). This matches the `session_id` in worker health metrics, enabling per-agent health display in agent-card.svelte.

**Source:** `web/src/lib/stores/agents.ts:32`, `web/src/lib/components/agent-card/agent-card.svelte`

**Significance:** No schema changes needed; existing session_id provides the join key.

---

## Synthesis

**Key Insights:**

1. **Separation of Concerns** - Worker health metrics are distinct from orchestrator coaching metrics. The aggregator needed separate logic paths to avoid mixing coaching signals meant for orchestrators with health signals meant for workers.

2. **Threshold Consistency** - Warning/critical thresholds in serve_coaching.go (3/5 failures, 80/90% context, 15/30m phase, 30/60m commit) mirror those in coaching.ts to provide consistent user experience.

3. **Graceful Degradation** - When no worker health metrics exist, the API omits the `worker_health` field (via `omitempty`). Dashboard components handle null gracefully.

**Answer to Investigation Question:**

Worker health metrics are surfaced by:
1. Backend aggregation in serve_coaching.go with new WorkerHealthMetrics struct
2. API response includes `worker_health` map keyed by session_id
3. coaching.ts store fetches and exposes worker health data
4. agent-card.svelte displays health indicators when metrics exist and status is warning/critical

---

## Structured Uncertainty

**What's tested:**

- API returns aggregated worker health (verified: curl test with injected metrics)
- Health status calculation (verified: 85% context + 20m phase = "warning")
- Go code compiles (verified: `go build ./cmd/orch/...`)
- Web frontend builds (verified: `npm run build`)

**What's untested:**

- Visual display in production dashboard (no active worker health metrics in JSONL)
- Performance with many sessions (tested only single session)
- Real-time update latency via polling

**What would change this:**

- If coaching.ts metric format changes, aggregation would break
- If session_id format doesn't match between agents and metrics, health won't display

---

## Implementation Recommendations

**Purpose:** N/A - this IS the implementation, following the task definition.

### Approach Taken

**Full-stack integration** - Backend aggregation + TypeScript types + Svelte UI components

**Why this approach:**
- Reuses existing coaching polling infrastructure
- Matches agent-to-health via session_id (no new IDs needed)
- Follows established patterns for tooltip-based indicators

**Trade-offs accepted:**
- Polling instead of push (simpler, matches existing coaching pattern)
- Health shown only when thresholds crossed (mirrors coaching.ts behavior)

**Implementation sequence:**
1. Added WorkerHealthMetrics struct to serve_coaching.go
2. Added aggregateWorkerHealthMetrics() function
3. Updated CoachingResponse to include worker_health map
4. Added TypeScript types to coaching.ts
5. Added healthIndicator to agent-card.svelte with tooltip

---

## References

**Files Modified:**
- `cmd/orch/serve_coaching.go` - Added WorkerHealthMetrics struct and aggregation
- `web/src/lib/stores/coaching.ts` - Added TypeScript types and helper functions
- `web/src/lib/components/agent-card/agent-card.svelte` - Added health indicator display

**Commands Run:**
```bash
# Verify Go builds
go build ./cmd/orch/...

# Verify TypeScript/Svelte
npm run build

# Test API with injected metrics
echo '{"timestamp":"2026-01-17T10:10:00.000Z","session_id":"test","metric_type":"context_usage","value":85}' >> ~/.orch/coaching-metrics.jsonl
curl -sk https://localhost:3348/api/coaching
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-add-worker-specific-metrics-plugins.md` - Implemented the coaching.ts emissions
- **Beads:** `bd show orch-go-yirj1` - This task

---

## Investigation History

**2026-01-17 09:50:** Investigation started
- Initial question: How to surface worker health metrics in dashboard
- Context: coaching.ts now emits 4 worker health metrics; need dashboard display

**2026-01-17 10:05:** Implementation complete
- Status: Complete
- Key outcome: Worker health metrics aggregated and displayed in agent cards
