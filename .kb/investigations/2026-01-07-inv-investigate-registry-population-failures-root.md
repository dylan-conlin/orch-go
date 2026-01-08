<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The "registry population issues" gap (7x) is a false positive - the issue was already investigated and resolved as a filename misconception (registry.json vs sessions.json).

**Evidence:** Prior investigation (2026-01-06-inv-registry-population-issues-orch-status.md) concluded "Not a bug"; current sessions.json has 14 sessions, orch status correctly shows 3 active orchestrator sessions.

**Knowledge:** Gap tracker accumulates events when spawns query "registry population issues" - each spawn generates 2-3 gap events. The 7x count represents multiple spawns referencing the same already-resolved issue, not 7 distinct problems.

**Next:** Add constraint to prevent re-spawning for this resolved issue. Clear the recurring gap events.

**Promote to Decision:** recommend-no - This is a gap tracker hygiene issue, not an architectural decision.

---

# Investigation: Investigate Registry Population Failures Root

**Question:** Why does "registry population issues" appear 7 times in orch learn, and is there a real underlying problem?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent og-feat-investigate-registry-population-07jan-4b8d
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Prior Investigation Already Resolved This Issue

**Evidence:** 
The investigation at `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` already concluded:
- The reported issue was "registry.json appears empty while orch status shows sessions"
- Root cause: **filename misconception** - the actual file is `sessions.json`, not `registry.json`
- `~/.orch/sessions.json` is correctly populated with session data
- Status: "Complete" with conclusion "Not a bug"

Key quote from prior investigation:
> "There is no `~/.orch/registry.json` file. The actual orchestrator session registry is `~/.orch/sessions.json`, which IS correctly populated (11 sessions visible) and IS being correctly used by `orch status`."

**Source:** 
- `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` (complete investigation)
- `ls ~/.orch/` - no `registry.json` file exists

**Significance:** The "7x gap" is tracking re-spawns for an already-resolved issue, not 7 distinct problems.

---

### Finding 2: Sessions Registry Is Working Correctly

**Evidence:** 
Current state verification:
```bash
$ cat ~/.orch/sessions.json | jq '.sessions | length'
14

$ orch status --json | jq '.orchestrator_sessions | length'
3
```

The registry contains 14 total sessions (11 completed/abandoned, 3 active). `orch status` correctly filters to show only the 3 active ones via `ListActive()`.

**Source:** 
- `~/.orch/sessions.json` - 14 sessions with proper status tracking
- `pkg/session/registry.go:277-288` - `ListActive()` correctly filters by status == "active"

**Significance:** No registry population issue exists - the system is working as designed.

---

### Finding 3: Gap Tracker Accumulates Events, Not Unique Issues

**Evidence:** 
The gap tracker at `~/.orch/gap-tracker.json` shows 8 events related to "registry population":
- All 8 events reference the same task description about "registry.json appears empty"
- Each spawn that queries "registry population issues" generates 2-3 gap events (sparse_context, no_constraints, no_decisions)
- The "7x" count in `orch learn` represents accumulated events, not 7 distinct problems

Sample events (all identical task):
```json
{
  "timestamp": "2026-01-07T01:53:33.679025Z",
  "query": "registry population issues",
  "gap_type": "sparse_context",
  "skill": "architect",
  "task": "Registry population issues - orch status shows sessions but registry.json appears empty..."
}
```

**Source:** 
- `~/.orch/gap-tracker.json` - 8 events with identical task descriptions
- `orch learn patterns` output showing "7x" for "registry population issues"

**Significance:** The gap count inflates when multiple spawns reference the same unresolved issue. Once an issue is resolved, the gap should be marked as addressed (via constraint or decision) to prevent re-inflation.

---

### Finding 4: Related Fix Was Also Implemented

**Evidence:**
A second investigation (`.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md`) fixed a **real** issue:
- `orch complete` was using `Unregister()` (removal) instead of `Update()` (status change)
- `orch abandon` had no registry update at all
- Fix: Changed to use `Update()` with status "completed" or "abandoned"

This fix is working correctly - sessions.json shows proper status tracking:
- 3 sessions with status "active"
- 3 sessions with status "abandoned"
- 8 sessions with status "completed"

**Source:**
- `.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md`
- `~/.orch/sessions.json` status distribution

**Significance:** Real registry issues were found and fixed. The "registry population issues" gap is separate and was already resolved as "not a bug."

---

## Synthesis

**Key Insights:**

1. **Gap inflation pattern** - When an issue is spawned multiple times without being marked resolved, the gap tracker accumulates events. The 7x count doesn't mean 7 problems - it means the same issue was queried 7 times (possibly across multiple spawn attempts or architect sessions).

2. **Filename confusion is common** - The registry has two files: `sessions.json` (orchestrator sessions, actively used) and `agent-registry.json` (legacy, unused). Agents encountering "registry" in documentation may confuse which file to check.

3. **Prior work was thorough** - Two investigations already covered this area:
   - Population issue: Resolved as filename misconception (not a bug)
   - Status update issue: Fixed (orch complete/abandon now update status correctly)

**Answer to Investigation Question:**

The "registry population issues" gap appearing 7 times in `orch learn` is a **false positive** caused by gap tracker accumulation. The underlying issue was already investigated on 2026-01-06 and concluded as "not a bug" - it was a filename misconception (registry.json doesn't exist; the actual file is sessions.json which is working correctly).

The gap tracker doesn't automatically clear entries when issues are resolved. Each time an agent spawns with this task and queries "registry population issues", new gap events are created, inflating the count. The solution is to add a constraint marking this issue as resolved.

---

## Structured Uncertainty

**What's tested:**

- ✅ `sessions.json` exists and has 14 sessions (verified: `cat ~/.orch/sessions.json | jq '.sessions | length'`)
- ✅ `orch status` correctly shows 3 active sessions (verified: `orch status --json`)
- ✅ Prior investigation concluded "not a bug" (verified: read full investigation file)
- ✅ Gap tracker has 8 events for same task (verified: `jq` query on gap-tracker.json)

**What's untested:**

- ⚠️ Whether adding a constraint will prevent future gap accumulation (need to test `kn constrain`)
- ⚠️ Long-term gap tracker hygiene (what happens to old resolved gaps?)

**What would change this:**

- Finding would be wrong if a new, distinct registry population issue emerged
- Finding would be wrong if sessions.json stopped being populated on new spawns

---

## Implementation Recommendations

**Purpose:** Prevent future false positive gap accumulation for this resolved issue.

### Recommended Approach ⭐

**Add constraint to mark issue as resolved** - Use `kn constrain` to document that "registry population issues" was investigated and resolved.

**Why this approach:**
- Constraints are surfaced during `kb context` queries
- Future spawns will see the constraint and know the issue is resolved
- Prevents re-spawning for the same resolved issue

**Trade-offs accepted:**
- Gap tracker still has historical events (acceptable - they're just history)
- Constraint adds to kb size (minimal impact)

**Implementation sequence:**
1. Run `kn constrain "registry population issues resolved" --reason "Investigated 2026-01-06: filename misconception. Actual file is sessions.json which works correctly. See .kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md"`

### Alternative Approaches Considered

**Option B: Clear gap tracker events manually**
- **Pros:** Removes clutter from gap tracker
- **Cons:** Events may be re-created on next query; doesn't prevent future accumulation
- **When to use instead:** When gap tracker file is extremely large

**Option C: Add gap tracker "resolved" marking**
- **Pros:** Would properly track resolved issues at the gap tracker level
- **Cons:** Requires feature development in orch-go
- **When to use instead:** If this pattern (false positives from resolved issues) recurs frequently

**Rationale for recommendation:** Option A (constraint) uses existing tooling and immediately prevents future spawns from treating this as an open issue.

---

### Implementation Details

**What to implement first:**
- Add constraint via `kn constrain`

**Things to watch out for:**
- ⚠️ Gap tracker will still show historical events (expected, not a problem)
- ⚠️ The constraint text should be specific enough to match future "registry population" queries

**Areas needing further investigation:**
- How to handle gap tracker hygiene systematically (not in scope for this investigation)

**Success criteria:**
- ✅ Future `kb context "registry population"` queries surface the constraint
- ✅ No new spawns for this specific resolved issue

---

## References

**Files Examined:**
- `~/.orch/sessions.json` - Verified contains 14 sessions with correct status
- `~/.orch/gap-tracker.json` - Analyzed gap events for "registry population"
- `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` - Prior investigation (complete, not a bug)
- `.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md` - Related fix (status update)
- `pkg/session/registry.go` - Registry implementation (verified correct path)

**Commands Run:**
```bash
# Check sessions count
cat ~/.orch/sessions.json | jq '.sessions | length'
# Result: 14

# Check active sessions in orch status
orch status --json | jq '.orchestrator_sessions | length'
# Result: 3

# Check gap tracker events
cat ~/.orch/gap-tracker.json | jq '[.events[] | select(.query | test("registry"; "i"))] | length'
# Result: 8

# Check orch learn output
orch learn patterns | grep "registry"
# Shows: 7x count for "registry population issues"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` - Original investigation that resolved this as "not a bug"
- **Investigation:** `.kb/investigations/2026-01-06-inv-session-registry-doesnt-update-orchestrator.md` - Fixed real status update issue

---

## Investigation History

**2026-01-07 20:12:** Investigation started
- Initial question: Why does "registry population issues" appear 7x in orch learn?
- Context: Spawned to investigate possible session lookup failures or storage race conditions

**2026-01-07 20:18:** Discovered prior investigation
- Found `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` already resolved this
- Conclusion was "not a bug" - filename misconception

**2026-01-07 20:25:** Verified current state
- sessions.json has 14 sessions, 3 active
- orch status correctly shows active sessions
- Gap tracker has 8 events all referencing same resolved issue

**2026-01-07 20:30:** Investigation completed
- Status: Complete
- Key outcome: The 7x gap is a false positive from gap tracker accumulation, not 7 distinct problems. Issue was already resolved.
