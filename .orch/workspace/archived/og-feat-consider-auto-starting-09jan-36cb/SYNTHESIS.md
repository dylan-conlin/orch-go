# Session Synthesis

**Agent:** og-feat-consider-auto-starting-09jan-36cb
**Issue:** orch-go-i8w57
**Duration:** 2026-01-09
**Outcome:** success

---

## TLDR

Reviewed the question "Should orch serve automatically start the beads daemon?" and validated the prior investigation's conclusion. **Recommendation: No auto-start needed.** Caching (30s TTL) already solves the cold-start performance penalty, and the per-project daemon architecture makes centralized daemon management inappropriate.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-feat-consider-auto-starting-09jan-36cb/SYNTHESIS.md` - This synthesis document

### Files Modified
- None (investigation only, no implementation needed)

### Commits
- No code changes required

---

## Evidence (What Was Observed)

### Current System State

**bd daemons are running per-project:**
```bash
$ pgrep -fl "bd.*daemon"
31917 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
88366 /Users/dylanconlin/.bun/bin/bd daemon --start --interval 5s
```
*Source: System verification 2026-01-09*

**Caching is implemented and effective:**
- Stats cache: 30s TTL (`serve_beads.go:35`)
- Ready issues cache: 15s TTL (`serve_beads.go:36`)
- Socket existence check prevents slow RPC timeouts (`serve_beads.go:92-98`)
- RPC with CLI fallback architecture (`serve_beads.go:104-112`)

**Prior investigation showed:**
- First request (cold): 6.5s
- Cached request: 15ms
- ~450x improvement with caching

**Intentional architectural separation:**
- `BEADS_NO_DAEMON=1` set in `~/Library/LaunchAgents/com.orch.daemon.plist` (line 42-43)
- orch daemon uses CLI mode (`--no-daemon`) to avoid daemon coordination complexity
- `orch-go.serve` plist is minimal (lines 9-11) - no PATH or daemon management

*Source: Verified in launchd plist files 2026-01-09*

---

## Knowledge (What Was Learned)

### Prior Investigation Validated

The investigation at `.kb/investigations/2026-01-07-inv-consider-auto-starting-beads-daemon.md` reached the correct conclusion. All findings remain valid as of 2026-01-09:

1. **Caching is the correct solution** - Not daemon management
2. **Beads daemons are per-project** - No single daemon to start
3. **Separation is intentional** - `BEADS_NO_DAEMON=1` shows architectural intent

### Architectural Insights

**Why auto-start would be inappropriate:**
1. **Per-project architecture** - Starting one daemon only helps one project (the one orch serve runs from), not other projects being monitored
2. **Complexity without benefit** - Process management adds code complexity when caching already solves the user-facing problem
3. **Conflicts with existing design** - `BEADS_NO_DAEMON=1` pattern shows intent to keep systems separate
4. **Redundancy risk** - Would duplicate work if bd daemon already running for that project

**Current architecture works correctly:**
- orch serve uses RPC when available
- Falls back to CLI when daemon unavailable
- Caches aggressively (30s/15s TTLs)
- Socket check prevents slow timeout loops

### Trade-offs Accepted

**What users experience:**
- First dashboard load after server restart: May be slow (up to 6.5s)
- Subsequent loads: Fast (<15ms from cache)

**Why this is acceptable:**
- First-load slowness is rare (server restarts are infrequent)
- Cache TTL keeps data fresh while maintaining speed
- bd daemons auto-start when bd commands run (happens naturally during work)

### Decisions Made

**Decision: Do not implement auto-start**

**Rationale:**
1. Caching already solves the user-facing performance problem
2. Per-project daemon model makes centralized management inappropriate
3. Would add process management complexity without benefit
4. Existing architecture is intentionally designed this way

**Not a decision to revisit:** This is a confirmed architectural choice, not a temporary workaround. The caching + fallback + per-project pattern is the correct design.

---

## Next (What Should Happen)

**Recommendation:** close

### Close Criteria Met
- [x] All deliverables complete (SYNTHESIS.md created)
- [x] No implementation needed (investigation confirms existing design is correct)
- [x] Investigation validated
- [x] Ready for `orch complete orch-go-i8w57`

### Potential Future Enhancement (Low Priority)

If first-load performance becomes a user-reported issue, consider **cache prewarming** at startup:

```go
// In runServe(), after cache initialization:
go func() {
    // Prewarm caches in background
    globalBeadsStatsCache.getStats("")
    globalBeadsStatsCache.getReadyIssues("")
}()
```

**When to implement:**
- Only if users report frustration with first-load delay
- Low effort (5-10 lines)
- Adds 1-2s to server startup in background
- May fail silently if bd not available (acceptable)

**Not implementing now because:**
- No evidence of user pain
- First request delay is rare (server restarts infrequent)
- Can add later if needed

---

## Unexplored Questions

**Health visibility:**
Could add a `/api/beads/health` endpoint showing daemon status for debugging. Useful for diagnosing slow responses, but adds complexity around "which project's daemon?"

**Cross-project daemon coordination:**
Dashboard follows orchestrator working directory (`project_dir` query param). Could explore better UX for showing multiple projects' daemons status, but this is a dashboard concern, not a serve concern.

**Cache prewarming effectiveness:**
Haven't tested whether prewarming at startup would reliably eliminate first-request latency. Would need to measure impact on startup time and handle failure cases gracefully.

---

## Session Metadata

**Skill:** feature-impl
**Model:** Opus
**Workspace:** `.orch/workspace/og-feat-consider-auto-starting-09jan-36cb/`
**Investigation:** `.kb/investigations/2026-01-07-inv-consider-auto-starting-beads-daemon.md` (prior work, validated)
**Beads:** `bd show orch-go-i8w57`
