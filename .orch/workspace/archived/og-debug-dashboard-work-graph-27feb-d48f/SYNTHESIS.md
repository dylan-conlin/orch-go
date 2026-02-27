# Session Synthesis

**Agent:** og-debug-dashboard-work-graph-27feb-d48f
**Issue:** orch-go-l1jp
**Outcome:** success

---

## Plain-Language Summary

The dashboard's work-graph page had API responses taking 2-10+ seconds because the `/api/attention` endpoint ran 11 data collectors sequentially (each making CLI/HTTP calls), and the `/api/beads/graph` endpoint rebuilt active agent data from scratch on every request. Fixed by parallelizing the attention collectors with goroutines, adding a 15-second response cache for attention, caching the active agent map with 15-second TTL, and parallelizing the two independent data-fetch steps inside `buildActiveAgentMap()`. Cached responses now return in ~7ms (down from 2-10s). Also silenced noisy console warnings from the stub WIP store that fired on every 30s poll cycle.

## TLDR

Dashboard API responses reduced from 2-10s to ~7ms by parallelizing 11 attention collectors, adding response-level caching (15s TTL) to both `/api/attention` and the active agent map used by `/api/beads/graph`, and parallelizing independent data fetches within `buildActiveAgentMap()`.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification steps and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_attention.go` - Added `attentionCache` with 15s TTL, parallelized 11 collectors using goroutines, added cache check/set in handler
- `cmd/orch/serve_beads.go` - Added `activeAgentMapCache` with 15s TTL, parallelized tracked agents + OpenCode session fetches in `buildActiveAgentMap()`, handler uses cached version
- `web/src/lib/stores/wip.ts` - Removed `console.warn` calls from stub `fetch`, `fetchQueued`, `setRunningAgents` methods

---

## Evidence (What Was Observed)

- `/api/attention` ran 11 collectors SEQUENTIALLY: beads, git, recently-closed, agent (HTTP to self), epic-orphan, verify-failed, unblocked, stuck (HTTP to self), stale, duplicate, competing
- Each collector makes external calls: `bd` CLI (200-500ms each), HTTP to `/api/agents` (500-1000ms), `git log` (200-800ms)
- Total sequential time: ~4-10s depending on load
- `/api/beads/graph` called `buildActiveAgentMap()` on every request, which made 2+ OpenCode HTTP calls + beads comments fetch
- `wip.ts` stubs emitted `console.warn` on every poll cycle (3 warnings every 30s)
- After fix: cached responses return in 7ms, cold cache still ~6s (one-time per 15s window)

### Tests Run
```bash
go test ./cmd/orch/... ./pkg/attention/...
# ok  github.com/dylan-conlin/orch-go/cmd/orch  7.718s
# ok  github.com/dylan-conlin/orch-go/pkg/attention  (cached)

go vet ./cmd/orch/
# (no output - clean)
```

### Smoke Test Results
```
Graph (cold cache):  6.3s  → one-time per 15s window
Graph (warm cache):  0.007s → 900x improvement
Attention (cached):  0.007s → 900x improvement
```

---

## Architectural Choices

### Parallel collectors + response cache (chosen) vs. just response cache
- **What I chose:** Both parallelization AND response-level caching
- **What I rejected:** Cache-only approach
- **Why:** Cache alone reduces cold-call frequency but doesn't reduce cold-call duration. Parallel collectors ensure cold calls take ~max(collectors) instead of sum(collectors). Combined, the user gets fast cached responses (~7ms) AND faster cold starts.
- **Risk accepted:** Parallel collectors may increase momentary resource usage (11 goroutines per cold request), but this is bounded and short-lived.

### 15-second TTL (chosen) vs. 30-second or shorter
- **What I chose:** 15s TTL for both attention and agent map caches
- **What I rejected:** 30s (too stale), 5s (too many cache misses)
- **Why:** Matches existing `globalBeadsStatsCache.graphTTL`. Dashboard polls every 30s, so 15s means at most 2 cold calls per minute. Event-driven refreshes are throttled to 3s minimum on the frontend, so they'll hit cache.

### activeAgentMapCache parallel internals vs. sequential
- **What I chose:** Parallel tracked agents + OpenCode sessions fetch
- **What I rejected:** Keep sequential
- **Why:** Steps 1 (tracked agents from cache) and 2 (OpenCode sessions via HTTP) are independent. Running in parallel reduces wall-clock time from sum to max of the two operations.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `buildActiveAgentMap()` takes 5-6s because `listSessionsAcrossProjects()` makes N HTTP calls to OpenCode (1 per registered kb project)
- The tracked agents cache has only 3s TTL, causing frequent cache misses
- The attention endpoint had NO caching at all before this fix

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (go test, go vet)
- [x] Smoke test passed (7ms cached, 6.3s cold)
- [x] Ready for `orch complete orch-go-l1jp`

---

## Unexplored Questions

- The cold-start graph call still takes ~6s. The dominant cost appears to be in `listSessionsAcrossProjects()` and `queryTrackedAgents()`. A future optimization could cache OpenCode session lists at a higher level or reduce the number of HTTP calls.
- The tracked agents cache has a 3s TTL which may be unnecessarily short. Increasing it could reduce cold-call frequency further.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-dashboard-work-graph-27feb-d48f/`
**Beads:** `bd show orch-go-l1jp`
