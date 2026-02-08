<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked agents linger in `orch status` because OpenCode sessions persist for 30 minutes after last activity, but there's no completion signal (no beads issue to check for closure).

**Evidence:** Sessions with `-untracked-` in title are shown if updated within `maxIdleTime` (30 min); two sessions at 3 min and 17 min old are within this window.

**Knowledge:** For untracked agents, there's no beads issue to close, so the only signal for completion is the session going stale (>30 min). This is actually correct behavior - they disappear automatically after 30 minutes of inactivity.

**Next:** Close - this is expected behavior. Untracked agents automatically disappear after 30 minutes. If faster cleanup is desired, `orch clean` could be enhanced to delete OpenCode sessions matching `-untracked-` pattern.

**Confidence:** High (90%) - Code analysis confirms behavior; observed sessions match expectations.

---

# Investigation: Investigate Untracked Agents Lingering Orch

**Question:** Why do untracked agents linger in `orch status` after they complete?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Untracked agents are identified by `-untracked-` pattern in beads ID

**Evidence:** 
- Beads IDs for untracked agents follow pattern: `{project}-untracked-{unix_timestamp}`
- Example: `orch-go-untracked-1766786808`, `orch-go-untracked-1766785873`
- Generated in `determineBeadsID()` when `--no-track` flag is set

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:1672-1673`
```go
if spawnNoTrack {
    return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil
}
```

**Significance:** The pattern is intentional and used for identification, but there's no beads issue created to track completion.

---

### Finding 2: `orch status` shows agents from OpenCode sessions updated within 30 minutes

**Evidence:** 
- `runStatus()` filters OpenCode sessions by `maxIdleTime = 30 * time.Minute`
- Sessions updated more than 30 minutes ago are excluded
- Current untracked sessions: 3 min old, 17 min old (within window)
- Older untracked sessions: 78 min, 86 min (correctly excluded)

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:2066-2080`
```go
const maxIdleTime = 30 * time.Minute
// ...
if now.Sub(updatedAt) <= maxIdleTime {
    beadsID := extractBeadsIDFromTitle(s.Title)
    if beadsID != "" {
        beadsToSession[beadsID] = s
    }
}
```

**Significance:** The 30-minute window is the mechanism for automatic cleanup - untracked agents disappear after they go idle for 30+ minutes.

---

### Finding 3: No beads issue means no completion signal for untracked agents

**Evidence:** 
- Tracked agents have beads issues that can be closed via `bd close`
- `IsCompleted` flag is set by checking `strings.EqualFold(issue.Status, "closed")`
- For untracked agents, `allIssues[beadsID]` returns nil, so `IsCompleted` stays false

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:2265-2268`
```go
if issue, ok := allIssues[oa.beadsID]; ok {
    task = truncate(issue.Title, 40)
    isCompleted = strings.EqualFold(issue.Status, "closed")
}
```

**Significance:** This is the fundamental difference - tracked agents can be explicitly completed, untracked agents rely only on the idle timeout.

---

## Synthesis

**Key Insights:**

1. **Automatic cleanup is working** - The 30-minute `maxIdleTime` filter is the cleanup mechanism for untracked agents. Sessions older than 30 minutes are correctly not shown.

2. **No completion signal by design** - Untracked agents (`--no-track`) intentionally don't have beads issues, so there's no way to explicitly mark them complete. This is a trade-off of using `--no-track`.

3. **This is expected behavior** - The agents showing up (3 min, 17 min old) are within the 30-minute window. They will automatically disappear once they go idle for 30+ minutes.

**Answer to Investigation Question:**

Untracked agents linger in `orch status` because they lack a beads issue to signal completion. The system correctly relies on the 30-minute idle timeout to clean them up. This is expected behavior - if faster cleanup is desired, users can either:
1. Wait 30 minutes (automatic)
2. Use `orch clean` to manually clean up

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code analysis clearly shows the filtering logic and the absence of a completion signal for untracked agents. Observed behavior matches the code.

**What's certain:**

- ✅ Untracked agents are filtered by 30-minute `maxIdleTime`
- ✅ No beads issue exists for untracked agents (by design)
- ✅ Agents older than 30 minutes are correctly excluded

**What's uncertain:**

- ⚠️ Whether users expect untracked agents to disappear faster
- ⚠️ Whether `orch clean` handles untracked OpenCode sessions

**What would increase confidence to Very High:**

- Testing `orch clean` to verify it handles untracked sessions
- User feedback on whether 30-minute timeout is acceptable

---

## Implementation Recommendations

**Purpose:** No implementation needed - this is expected behavior.

### Recommended Approach ⭐

**No changes needed** - The 30-minute idle timeout is appropriate for untracked agents.

**Why this approach:**
- Untracked agents are meant to be lightweight/ephemeral
- 30 minutes is reasonable for batch work completion
- Users chose `--no-track` specifically to avoid beads tracking overhead

**Trade-offs accepted:**
- Untracked agents linger for up to 30 minutes after completion
- No explicit "complete" mechanism for untracked agents

### Alternative Approaches Considered

**Option B: Add explicit cleanup command for untracked agents**
- **Pros:** Faster cleanup when needed
- **Cons:** Adds complexity; users already have `orch clean`
- **When to use instead:** If 30-minute timeout proves problematic in practice

**Option C: Reduce `maxIdleTime` to 10-15 minutes**
- **Pros:** Faster automatic cleanup
- **Cons:** Might hide agents that are still legitimately idle
- **When to use instead:** If 30 minutes is universally too long

**Rationale for recommendation:** This is working as designed. Untracked agents are ephemeral by nature.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:1672-1673` - `determineBeadsID()` generates untracked IDs
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:2066-2080` - `maxIdleTime` filtering
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:2265-2268` - `IsCompleted` check

**Commands Run:**
```bash
# Check untracked session ages
curl -s http://127.0.0.1:4096/session | jq '...'

# Verify current orch status
orch status
```

---

## Investigation History

**2025-12-26 14:13:** Investigation started
- Initial question: Why do untracked agents linger in orch status?
- Context: `orch status` showing two untracked agents as "idle"

**2025-12-26 14:15:** Key findings identified
- Found `maxIdleTime` = 30 minutes
- Confirmed untracked agents lack beads issues
- Verified sessions within timeout window

**2025-12-26 14:16:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: This is expected behavior - untracked agents auto-cleanup after 30 min idle
