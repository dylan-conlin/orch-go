# Session Synthesis

**Agent:** og-inv-debug-coaching-plugin-28jan-b245
**Issue:** orch-go-21001
**Duration:** 2026-01-28 13:39 → 2026-01-28 14:20
**Outcome:** success

---

## TLDR

Investigated why coaching plugin still fires on workers despite metadata.role='worker' being set. Root cause: session.created events do NOT include metadata field, so metadata-based worker detection never registers workers. Recommendation: revert to title-based detection (proven working in prior investigation).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-debug-coaching-plugin-still-fires.md` - Complete investigation with root cause analysis

### Files Modified
- `.opencode/plugins/coaching.ts` - Added enhanced debug logging to event and tool.execute.after handlers (lines ~1708, 1996-2030)

### Commits
- `23bbe0dc` - investigation: Initial checkpoint - traced worker detection logic
- `e827ab62` - investigation: Root cause found - metadata.role not in session.created events  
- `7e3c0e60` - investigation: Complete findings and recommendations

---

## Evidence (What Was Observed)

### Key Findings

1. **Worker detection logic exists but never triggers**
   - Source: coaching.ts:1993-2030 (event handler), 1705-1733 (tool.execute.after check)
   - Event handler checks `sessionMetadata.role === "worker"` at line 2020
   - If true, should set `workerSessions.set(sessionId, true)` at line 2023
   - Tool handler checks `workerSessions.get(sessionId) === true` at line 1705
   - But workerSessions map is never populated

2. **session.created event structure confirmed**
   - Source: `grep '"event_type":"session.created"' ~/.orch/event-test.jsonl | tail -1 | jq '.'`
   - Event contains: properties.info.{id, slug, version, projectID, directory, title, permission, time}
   - **NO metadata field present**
   - Event handler reads `info.metadata || {}` which evaluates to empty object
   - Therefore `sessionMetadata.role === "worker"` is always false

3. **Title-based detection was working**
   - Source: `.kb/investigations/2026-01-28-inv-verify-coaching-plugin-worker-detection.md`
   - Two worker sessions tested: zero coaching alerts fired
   - Pattern: hasBeadsId (e.g., `[orch-go-20993]`) AND NOT orchestrator pattern (`-orch-`)
   - Sessions with this pattern successfully excluded from coaching

4. **Event hook system is functional**
   - Source: event-test.ts:61-101
   - Event hook successfully receives and logs all events including session.created
   - Confirms the problem is not with event system, but with event content

### Tests Run
```bash
# Verified session.created events exist
grep "session.created" ~/.orch/event-test.jsonl | tail -5

# Examined actual event structure
grep '"event_type":"session.created"' ~/.orch/event-test.jsonl | tail -1 | jq '.'

# Confirmed no metadata field exists
# Result: properties.info has id, title, projectID, directory, permission, time - NO metadata
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-debug-coaching-plugin-still-fires.md` - Root cause analysis for coaching plugin worker detection failure

### Decisions Made
- **Decision 1:** Enhanced debug logging NOT needed in production - should be removed after fix is verified
- **Decision 2:** Title-based detection is the correct approach given metadata is unavailable in events

### Constraints Discovered
- **Constraint 1:** OpenCode plugin event hooks do not have access to session metadata (e.g., metadata.role) in session.created events
- **Constraint 2:** Worker detection must rely on observable session properties: id, title, projectID, directory
- **Constraint 3:** Comments claiming "OpenCode now reliably exposes metadata.role" were aspirational, not validated

### Implementation Insight
The plugin attempted to "upgrade" from title-based to metadata-based detection without verifying that metadata is actually available in the event hook. This is a classic example of assuming an API provides data without testing. The prior working implementation was replaced with a broken one.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Spawn Follow-up
**Issue:** "Fix coaching plugin worker detection: revert to title-based pattern"
**Skill:** feature-impl
**Context:**
```
Investigation orch-go-21001 found coaching plugin's metadata-based worker 
detection doesn't work because session.created events lack metadata field.
Revert to title-based detection at coaching.ts:1993-2030. Pattern: 
hasBeadsId (regex: /\[[\w-]+-\d+\]/) && !isOrchestrator (regex: /-orch-/).
Prior investigation 2026-01-28-inv-verify-coaching-plugin-worker-detection.md
proves this pattern works. Remove enhanced debug logging added during investigation.
```

**Acceptance Criteria:**
- Worker sessions (with beads IDs like `[orch-go-20993]`) have zero coaching alerts
- Orchestrator sessions (with `-orch-` pattern) continue receiving coaching
- Enhanced debug logging cleaned up (console.error statements removed)
- Test with real worker spawn to verify

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

1. **Where does metadata.role actually get set?** - The task description says "curl shows sessions have role='worker'" but we never found where this metadata lives. Is it in a different API endpoint? Is it stored but not exposed to plugin events?

2. **Could metadata be accessed via OpenCode client API?** - The plugin receives a `client` parameter. Does client.session.get() or similar provide access to metadata that isn't in events?

3. **Why was the "upgrade" to metadata-based detection done?** - Who made the change and what problem were they trying to solve? Was there an issue with title-based detection that metadata would fix?

**What remains unclear:**

- Whether orch-go actually sends x-opencode-env-ORCH_WORKER header (assumed from comments, not verified)
- Whether other event types (session.updated, etc.) include metadata
- Whether ad-hoc worker sessions (without beads IDs) need different detection

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-5 (via OpenCode)
**Workspace:** `.orch/workspace/og-inv-debug-coaching-plugin-28jan-b245/`
**Investigation:** `.kb/investigations/2026-01-28-inv-debug-coaching-plugin-still-fires.md`
**Beads:** `bd show orch-go-21001`
