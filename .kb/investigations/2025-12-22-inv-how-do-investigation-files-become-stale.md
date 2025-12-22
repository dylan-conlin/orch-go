## Summary (D.E.K.N.)

**Delta:** Investigation files become stale when agents update Status: Complete but forget to update the D.E.K.N. summary - the summary claims "Next: Implement..." while the body claims "Status: Complete".

**Evidence:** Issue orch-go-0cjl's investigation (2025-12-22-inv-update-orch-status-use-islive.md) has D.E.K.N. saying "Next: Implement..." but Status: Complete and actual code at main.go:1602-1616 shows state.GetLiveness() IS implemented.

**Knowledge:** The D.E.K.N. summary is written during investigation BEFORE implementation, but agents don't update it AFTER implementation - they only update Status/Phase fields in the body.

**Next:** Add enforcement: require D.E.K.N. "Next:" field to say "Implementation complete" or "Close" when Status: Complete. Add self-review checklist item.

**Confidence:** High (90%) - Root cause confirmed by comparing investigation file vs actual code vs beads comments.

---

# Investigation: How Do Investigation Files Become Stale and Mislead Agents

**Question:** How did the investigation 2025-12-22-inv-update-orch-status-use-islive.md become stale, claiming work was needed when it was already implemented?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** og-inv-how-do-investigation-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: The stale investigation claims implementation is needed

**Evidence:** D.E.K.N. summary at line 14 of the investigation file:
```markdown
**Next:** Implement: Replace ad-hoc checks with `state.GetLiveness()`, add phantom indicator, exclude phantoms from Active count.
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md:14`

**Significance:** The D.E.K.N. summary - the 30-second handoff designed to inform fresh Claude - explicitly says implementation is needed. A future agent reading this would conclude work needs to be done.

---

### Finding 2: The same investigation file claims Status: Complete

**Evidence:** At lines 44-47 of the same file:
```markdown
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md:44-47`

**Significance:** The body of the investigation contradicts the D.E.K.N. summary. Status: Complete and Phase: Complete suggest work IS done, but the summary says "Next: Implement...".

---

### Finding 3: The actual code shows implementation IS complete

**Evidence:** In `cmd/orch/main.go:1602-1616`, `state.GetLiveness()` is actively used:
```go
// Use state.GetLiveness() for accurate liveness check
var liveness state.LivenessResult
if beadsID != "" {
    liveness = state.GetLiveness(beadsID, serverURL, projectDir)
    seenBeadsIDs[beadsID] = true
}

agent := AgentInfo{
    SessionID: liveness.SessionID,
    ...
    IsPhantom: liveness.IsPhantom(),
}
```

**Source:** `cmd/orch/main.go:1602-1616`

**Significance:** The implementation that the investigation claims is "Next:" is already done. The D.E.K.N. summary is simply outdated.

---

### Finding 4: Beads issue confirms implementation was completed

**Evidence:** From `.beads/issues.jsonl`, issue orch-go-0cjl shows:
```json
{"id":"orch-go-0cjl",...
"comments":[
  {"text":"Phase: Complete - Updated orch status to use state.GetLiveness() for accurate liveness checks. Added phantom agent detection with STATUS column showing 'active' vs 'phantom'. Phantoms now excluded from Active count. All tests passing."}
]}
```

**Source:** `.beads/issues.jsonl` (orch-go-0cjl)

**Significance:** The beads comment explicitly states implementation is complete. The agent reported Phase: Complete via beads, but failed to update the D.E.K.N. summary in the investigation file.

---

### Finding 5: Comparison with properly updated investigation

**Evidence:** Investigation `2025-12-21-inv-orch-status-showing-stale-sessions.md` correctly says:
```markdown
**Next:** Implementation complete. Fix committed.
```
at line 13.

**Source:** `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md:13`

**Significance:** This shows the CORRECT pattern - after implementation, the agent updated the D.E.K.N. "Next:" field. The stale investigation failed to do this.

---

## Synthesis

**Key Insights:**

1. **D.E.K.N. written too early** - The D.E.K.N. summary is written during the investigation phase, BEFORE implementation. If the agent then implements the fix, they update Status/Phase but forget to update D.E.K.N.

2. **Two sources of truth diverge** - Status/Phase fields in the body and the D.E.K.N. summary can disagree. Future agents reading only D.E.K.N. get wrong information.

3. **No enforcement of consistency** - The skill's Self-Review checklist doesn't require checking that D.E.K.N. "Next:" matches "Status: Complete".

**Answer to Investigation Question:**

The investigation file became stale because:
1. Agent wrote D.E.K.N. during investigation with "Next: Implement..."
2. Agent implemented the fix and updated Status: Complete
3. Agent forgot to update D.E.K.N. "Next:" to say "Implementation complete"
4. Self-Review checklist didn't catch this inconsistency

---

## Test Performed

**Test:** Compared D.E.K.N. summary vs Status field vs actual code implementation vs beads comments.

**Result:** All four sources tell different stories:
- D.E.K.N.: "Next: Implement..." (claims work needed)
- Status: Complete (claims done)
- Code: GetLiveness() used at lines 1602-1616 (actually done)
- Beads: "Phase: Complete - Updated orch status to use state.GetLiveness()..." (actually done)

The D.E.K.N. is the stale source.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Direct evidence from comparing four sources (investigation file D.E.K.N., investigation file Status, actual code, beads comments) that clearly show the D.E.K.N. wasn't updated after implementation.

**What's certain:**

- ✅ D.E.K.N. says "Next: Implement..." - verified by reading file
- ✅ Status/Phase say "Complete" - verified by reading file  
- ✅ Actual code shows GetLiveness() IS implemented - verified by reading main.go:1602-1616
- ✅ Beads confirms implementation complete - verified by reading issues.jsonl

**What's uncertain:**

- ⚠️ How common is this pattern across all investigations? (Only examined 2 files)
- ⚠️ Would adding a checklist item actually fix this, or will agents skip it?

---

## Implementation Recommendations

### Recommended Approach: Add D.E.K.N. consistency check to Self-Review

**Why this approach:**
- Addresses root cause (D.E.K.N. not updated after implementation)
- Low cost (one additional checklist item)
- Self-enforcing (agents already do self-review)

**Trade-offs accepted:**
- Relies on agent compliance (same as all other checklist items)
- Doesn't prevent the error, just catches it before completion

**Implementation sequence:**
1. Add checklist item: "D.E.K.N. 'Next:' matches Status (if Complete, Next should be 'Implementation complete' or 'Close')"
2. Update investigation skill with explicit guidance
3. Consider tooling: `kb lint` command to check D.E.K.N./Status consistency

### Alternative Approaches Considered

**Option B: Auto-update D.E.K.N. when Status changes**
- **Pros:** Prevents divergence automatically
- **Cons:** Complex to implement, may override intentional states
- **When to use instead:** If Self-Review checklist proves ineffective

**Option C: Remove D.E.K.N. summary**
- **Pros:** No divergence possible
- **Cons:** Loses valuable 30-second handoff capability
- **When to use instead:** Never - D.E.K.N. is too valuable

---

## Self-Review

- [x] Real test performed (compared 4 sources: D.E.K.N., Status, code, beads)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered (traced exact cause of staleness)
- [x] File complete

**Self-Review Status:** PASSED

---

## Discovered Work

**Issue to create:**
- Add Self-Review checklist item to investigation skill requiring D.E.K.N./Status consistency check

---

## Leave it Better

`kn constrain "D.E.K.N. 'Next:' field must be updated when marking Status: Complete" --reason "Prevents stale investigations that mislead future agents"`

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md` - The stale investigation
- `.kb/investigations/2025-12-21-inv-orch-status-showing-stale-sessions.md` - Correct example
- `cmd/orch/main.go:1602-1616` - Actual implementation
- `.beads/issues.jsonl` (orch-go-0cjl) - Beads completion record

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-update-orch-status-use-islive.md` - The problematic file
