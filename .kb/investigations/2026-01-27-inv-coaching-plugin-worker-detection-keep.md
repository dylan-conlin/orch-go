<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin worker detection has been fixed 13+ times but keeps failing because of two fundamental architectural problems: (1) session.created event's directory field contains PROJECT root, not worker's workspace directory, and (2) detection relies on fallback heuristics (title patterns, tool arguments) that each have edge cases.

**Evidence:** Verified session.created event shows `"directory": "/Users/dylanconlin/Documents/personal/orch-go"` for ALL sessions including workers. Title-based detection works but requires specific title patterns like `[xxx-yyy]` AND absence of `-orch-`. Tool-argument detection fires too late (same hook as coaching alerts).

**Knowledge:** The "right" fix would be OpenCode exposing session.metadata.role from x-opencode-env-ORCH_WORKER header, but this requires upstream changes. Current title-based detection DOES work for properly-titled workers but has no fallback for edge cases.

**Next:** Recommend moving worker detection to OpenCode core (upstream contribution), with title-based detection as interim solution that already works.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28-coaching-plugin-disabled.md)

---

# Investigation: Why Does Coaching Plugin Worker Detection Keep Failing?

**Question:** Why do coaching plugin worker detection fixes keep breaking despite 13+ attempts since Jan 10? Is there a systemic issue we're missing?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** og-arch-coaching-plugin-worker-27jan-374d
**Phase:** Complete
**Next Step:** None - architectural recommendation complete
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: session.created Event Directory is PROJECT Root, Not Workspace

**Evidence:** Analyzed actual session.created event payloads from ~/.orch/event-test.jsonl:
```json
{
  "event_type": "session.created",
  "properties": {
    "info": {
      "id": "ses_3fe127497ffeuk0tMqqVCFFVvG",
      "directory": "/Users/dylanconlin/Documents/personal/orch-go",
      "title": "og-arch-coaching-plugin-worker-27jan-374d [orch-go-untracked-1769558868]"
    }
  }
}
```

The `directory` field is always the PROJECT root (`/Users/.../orch-go`), NOT the worker's workspace directory (`.orch/workspace/og-arch-*/`).

**Source:** `~/.orch/event-test.jsonl`, verified for 3 different session.created events

**Significance:** This invalidates the Jan 23 investigation's recommendation to use directory-based detection in session.created - the directory is NEVER `.orch/workspace/` because OpenCode tracks the project directory, not the agent's working directory.

---

### Finding 2: Current Detection Logic is Title-Based (And Actually Works)

**Evidence:** Current coaching.ts (lines 2018-2023) uses title-based detection:
```typescript
const isInWorkspace = sessionDirectory && sessionDirectory.includes(".orch/workspace/")  // ALWAYS false!
const isOrchestratorTitle = /-orch-/.test(sessionTitle) || /^meta-/.test(sessionTitle)
const hasBeadsId = /\[[\w-]+-\w+\]/.test(sessionTitle)

const isWorker = (isInWorkspace && !isOrchestratorTitle) || (hasBeadsId && !isOrchestratorTitle)
```

For this session (`og-arch-coaching-plugin-worker-27jan-374d [orch-go-untracked-1769558868]`):
- `isInWorkspace` = false (directory is project root)
- `isOrchestratorTitle` = false (no `-orch-` in title)
- `hasBeadsId` = true (has `[orch-go-untracked-1769558868]`)
- `isWorker` = true (correctly detected!)

**Source:** `plugins/coaching.ts:2018-2023`, session title from event-test.jsonl

**Significance:** The title-based detection WORKS when titles follow the pattern. The problem isn't the current code - it's edge cases where titles don't match.

---

### Finding 3: 13+ Fix Attempts Each Addressed Different Edge Cases

**Evidence:** Git log shows 13+ commits to coaching.ts since Jan 10:
1. `ddca8a36` - Worker filtering (env var approach) - Failed: plugin runs in server process
2. `6e6503ae` - Per-session detection in tool hooks - Failed: fires same time as coaching
3. `05859f38` - Message content detection - Failed: fires after coaching
4. `aea09aaf` - Cache positive results only - Fixed caching bug, didn't solve timing
5. `37b9b0b0` - session.metadata.role - Failed: not reliably exposed by OpenCode
6. `7c01ddef` - File-path based detection - Failed: fires too late
7. `65793caa` - session.created event - Failed: directory is project root
8. `da067161` - Title pattern detection - Works for proper titles
9. `b6301a28` - Directory-based early detection - Failed: wrong directory
10. `389e2041` - Distinguish spawned orchestrators - Works
11. `9ed47c17` - False positive fixes - Works for tuning

Each fix addressed a real problem but introduced new edge cases because the underlying architecture doesn't support clean worker/orchestrator distinction.

**Source:** `git log --oneline -20 -- plugins/coaching.ts`

**Significance:** This isn't a bug that can be fixed with one more patch - it's a fundamental architectural gap. The plugin layer doesn't have reliable access to worker identity.

---

### Finding 4: Root Cause is Missing First-Class Worker Identity in OpenCode

**Evidence:** 
- `x-opencode-env-ORCH_WORKER=1` header IS sent by orch-go (client.go:559-561)
- OpenCode should expose this in `session.metadata.role` but doesn't reliably
- Plugin layer has NO reliable way to know if a session is a worker before tools execute
- All detection approaches are heuristics with edge cases:
  - Title patterns: Depends on naming conventions
  - Directory paths: session.created has wrong directory, tool args fire too late
  - Environment variables: Plugin runs in server process, can't see agent env
  - Metadata: Not reliably exposed by OpenCode

**Source:** 
- `pkg/opencode/client.go:559-561` (header sent)
- Investigation 2026-01-17 (metadata unreliable)
- session.created event structure (directory is project root)

**Significance:** The fix isn't in coaching.ts - it's in OpenCode. Worker identity should be a first-class concept exposed reliably in session metadata.

---

### Finding 5: Title-Based Detection is Currently the Best Available Approach

**Evidence:** Title-based detection at session.created:
- Fires BEFORE any tool calls (eliminates race condition)
- Uses data that IS available (title from info object)
- Works for all properly-titled workers
- Has clear fallback for orchestrators (title contains `-orch-`)

The detection logic `(hasBeadsId && !isOrchestratorTitle)` correctly identifies:
- Workers: `og-feat-*`, `og-inv-*`, `og-arch-*`, `og-debug-*` with beads ID
- Orchestrators: `og-orch-*`, `meta-orch-*`
- Manual sessions: No beads ID, treated as orchestrator (correct)

**Source:** `plugins/coaching.ts:2018-2035`

**Significance:** The current implementation is actually the best we can do without OpenCode changes. Future work should focus on fixing OpenCode, not adding more heuristics.

---

## Synthesis

**Key Insights:**

1. **The "keeps failing" perception is misleading** - Title-based detection works. The repeated fixes were addressing different edge cases (caching bugs, timing issues, false positives) rather than the same bug recurring.

2. **session.created directory is fundamentally wrong for worker detection** - It's the project root, not the workspace. The Jan 23 investigation's recommendation was based on an incorrect assumption about event payload structure.

3. **Plugin layer can't reliably determine worker identity** - All approaches are heuristics. The proper fix is making worker identity a first-class concept in OpenCode core.

4. **Title-based detection is the current best solution** - It fires early (session.created), uses reliable data (title), and correctly handles the common case. Edge cases should be fixed by improving OpenCode, not adding more plugin heuristics.

**Answer to Investigation Question:**

The coaching plugin worker detection "keeps failing" because each fix addressed a different edge case while the underlying architectural gap remained. The systemic issue is that **OpenCode doesn't expose worker identity as a first-class concept**.

The current title-based detection (checking for beads ID pattern and absence of `-orch-`) actually works for properly-named workers. The perception of repeated failures comes from:
1. Different edge cases appearing over time
2. Incorrect assumptions about event payload structure (directory field)
3. Attempting to detect workers through heuristics rather than metadata

**The proper fix is upstream:** OpenCode should reliably expose `session.metadata.role` from the `x-opencode-env-ORCH_WORKER` header that orch-go already sends.

---

## Structured Uncertainty

**What's tested:**

- ✅ session.created event directory is project root (verified: examined 3 events in event-test.jsonl)
- ✅ Title-based detection works for og-arch-* pattern (verified: this session was correctly detected as worker)
- ✅ x-opencode-env-ORCH_WORKER header is sent (verified: client.go:559-561)
- ✅ 13+ commits exist addressing different aspects (verified: git log)

**What's untested:**

- ⚠️ Whether OpenCode intentionally excludes metadata or has a bug
- ⚠️ Edge cases where title patterns fail (ad-hoc spawns without beads tracking)
- ⚠️ Whether the detection actually prevents coaching alerts in production

**What would change this:**

- Finding wrong if session.created directory sometimes IS workspace (would enable directory-based detection)
- Finding wrong if OpenCode already exposes metadata correctly (would be a reading bug in plugin)
- Finding wrong if title-based detection has frequent false positives (would need different approach)

---

## Implementation Recommendations

### Recommended Approach: Keep Current Implementation + Upstream Fix

**Two-part strategy:**
1. Keep current title-based detection (it works)
2. Contribute fix to OpenCode to reliably expose session.metadata.role

**Why this approach:**
- Title-based detection already works for common cases
- Upstream fix addresses root cause, not symptoms
- No more heuristic churn - clean architectural solution

**Trade-offs accepted:**
- Edge cases (ad-hoc spawns without proper titles) may still receive coaching
- Upstream fix requires external contribution timeline

**Implementation sequence:**
1. Verify current detection is working in production (monitor coaching-metrics.jsonl for worker entries)
2. File OpenCode issue/PR for exposing session.metadata.role reliably
3. Once OpenCode fix is merged, simplify coaching.ts to use metadata

### Alternative Approaches Considered

**Option B: Add more title patterns**
- **Pros:** Covers more edge cases
- **Cons:** Heuristic churn continues; each pattern has its own edge cases
- **When to use instead:** Quick fix for specific missed patterns while waiting for upstream

**Option C: Deferred alert queue**
- **Pros:** Would ensure detection always fires before coaching
- **Cons:** Adds complexity; delays legitimate coaching; timing bugs
- **When to use instead:** If title-based detection proves unreliable in production

**Rationale for recommendation:** The current implementation works. Adding more heuristics continues the pattern of fixing one edge case while introducing another. The root cause is architectural - fix it at the source.

---

### Implementation Details

**What to verify first:**
- Check coaching-metrics.jsonl for any worker session entries (should be zero for worker-specific patterns)
- Confirm title-based detection logged "Worker detected" for recent spawns

**Things to watch out for:**
- ⚠️ Ad-hoc spawns (`orch spawn --no-track`) may not have proper titles
- ⚠️ Manual sessions created through OpenCode UI won't have beads IDs
- ⚠️ OpenCode version changes may affect event payload structure

**Areas needing further investigation:**
- What percentage of spawns are detected correctly?
- Are there coaching alerts in metrics for known worker sessions?
- What's involved in contributing to OpenCode?

**Success criteria:**
- ✅ Zero orchestrator coaching alerts fire on worker sessions
- ✅ Workers show in coaching-metrics.jsonl only for worker-specific metrics (tool_failure_rate, context_usage)
- ✅ Orchestrators continue to receive coaching as expected

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-2039` - Full coaching plugin including worker detection
- `~/.orch/event-test.jsonl` - Real session.created event payloads
- `.kb/investigations/2026-01-23-inv-review-coaching-plugin-worker-detection.md` - Prior investigation
- `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Architecture analysis
- `pkg/opencode/client.go:559-561` - ORCH_WORKER header sending

**Commands Run:**
```bash
# Check recent commits
git log --oneline -20 -- plugins/coaching.ts

# Verify session.created event structure
grep '"event_type":"session.created"' ~/.orch/event-test.jsonl | tail -3 | jq '.'

# Check plugin deployment
diff plugins/coaching.ts .opencode/plugins/coaching.ts
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-review-coaching-plugin-worker-detection.md` - Recommended directory-based detection (incorrect assumption)
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive architecture
- **Decision:** Use x-opencode-env-ORCH_WORKER header (kb-e93bc1)

---

## Investigation History

**2026-01-27 16:07:** Investigation started
- Initial question: Why does worker detection keep failing despite 8+ fix attempts?
- Context: Spawned to analyze systemic issues in coaching plugin worker detection

**2026-01-27 16:20:** Analyzed session.created event structure
- Key discovery: directory field is PROJECT root, not workspace
- This invalidates the Jan 23 recommendation for directory-based detection

**2026-01-27 16:30:** Traced 13+ fix commits
- Found each addressed different edge case
- Pattern: heuristic churn without architectural fix

**2026-01-27 16:45:** Verified current detection works
- Title-based detection correctly identified this session as worker
- Current code is actually working for properly-titled workers

**2026-01-27 17:00:** Investigation completed
- Status: Complete
- Key outcome: Root cause is missing first-class worker identity in OpenCode; current title-based detection works; upstream fix is the proper solution
