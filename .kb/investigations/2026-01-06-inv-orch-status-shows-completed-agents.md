---
linked_issues:
  - orch-go-ij1pl
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch complete` does not delete the OpenCode session, causing completed agents to appear in `orch status --all` until the 30-minute idle window expires.

**Evidence:** Searched `complete_cmd.go` for `DeleteSession` - no matches. Found 136 persisted OpenCode sessions with orch-go beads IDs. `orch abandon` correctly deletes sessions (line 169).

**Knowledge:** Agent state exists in 4 layers (OpenCode memory, OpenCode disk, registry, tmux). `orch complete` cleans tmux but not OpenCode. Fix should copy deletion pattern from `orch abandon`.

**Next:** Implement fix: Add `client.DeleteSession(sessionID)` to `complete_cmd.go` after tmux cleanup (around line 537).

**Promote to Decision:** recommend-no (tactical fix, pattern already established in abandon_cmd.go)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Orch Status Shows Completed Agents

**Question:** Why does `orch status --all` show completed agents (beads issue closed) after `orch complete` runs? Is it stale state in the registry or OpenCode session tracking?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Status command correctly checks beads issue status for IsCompleted flag

**Evidence:** In `status_cmd.go:310-313` and `status_cmd.go:362-365`, the code fetches issue status via `verify.GetIssuesBatch()` and sets `isCompleted = strings.EqualFold(issue.Status, "closed")`. This is called for every agent in both tmux and OpenCode agent lists.

**Source:** 
- `cmd/orch/status_cmd.go:310-313` (tmux agents)
- `cmd/orch/status_cmd.go:362-365` (OpenCode agents)
- `pkg/verify/beads_api.go:298-350` (GetIssuesBatch implementation)

**Significance:** The code correctly determines completion status from beads, so the bug is not in status calculation logic itself.

---

### Finding 2: Filtering logic correctly filters completed agents by default

**Evidence:** In `status_cmd.go:389-404`, completed agents are filtered out unless `--all` flag is set:
```go
// Filter completed agents (beads issue closed) unless --all is set
if agent.IsCompleted && !statusAll {
    continue
}
```

**Source:** `cmd/orch/status_cmd.go:389-404`

**Significance:** The filtering logic is correct - completed agents should only appear with `--all` flag.

---

### Finding 3: Agent collection happens from two sources with 30-minute idle threshold

**Evidence:** 
1. **Tmux windows** (lines 159-195): All windows in `workers-*` sessions are collected as potential agents
2. **OpenCode sessions** (lines 206-232): Sessions updated within 30 minutes (`maxIdleTime = 30 * time.Minute`) are matched to beads IDs

**Source:** `cmd/orch/status_cmd.go:130-144` (session filtering), `cmd/orch/status_cmd.go:159-232` (agent collection)

**Significance:** OpenCode sessions persist even after agent completes. If the beads ID is extracted from session title, the agent will appear in the list. The 30-minute window means recently completed agents will still be collected.

---

### Finding 4: `orch complete` does NOT delete the OpenCode session

**Evidence:** Searched `complete_cmd.go` for `DeleteSession` - no matches. The command only:
1. Closes beads issue (line 480)
2. Closes tmux window (lines 531-537)
3. Invalidates serve cache (line 627)

**Source:** 
- `cmd/orch/complete_cmd.go` (full file search)
- `cmd/orch/abandon_cmd.go:165-174` (shows how session deletion SHOULD be done)

**Significance:** This is the root cause. After completion, the OpenCode session persists and is matched to the closed beads issue within the 30-minute window, causing stale "completed" agents to appear in `orch status --all`.

---

### Finding 5: `orch abandon` correctly deletes OpenCode sessions

**Evidence:** In `abandon_cmd.go:165-174`:
```go
// Delete the OpenCode session if it exists
// This prevents abandoned agents from appearing in `orch status`
if sessionID != "" {
    fmt.Printf("Deleting OpenCode session: %s\n", sessionID[:12])
    if err := client.DeleteSession(sessionID); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to delete OpenCode session: %v\n", err)
    } else {
        fmt.Printf("Deleted OpenCode session\n")
    }
}
```

**Source:** `cmd/orch/abandon_cmd.go:165-174`

**Significance:** The fix is clear - `orch complete` should replicate this session deletion logic.

---

### Finding 6: Test confirmed 136 persisted OpenCode sessions with orch-go beads IDs

**Evidence:** 
```bash
$ curl -s localhost:4096/session | jq '[.[] | select(.title | test("orch-go-"))] | length'
136
```

**Source:** Local testing against OpenCode API

**Significance:** This demonstrates the scope of session accumulation. Without cleanup, sessions persist indefinitely until `orch clean` is run manually.

---

## Synthesis

**Key Insights:**

1. **Root cause: Missing OpenCode session cleanup in `orch complete`** - When `orch complete` runs, it closes the beads issue and tmux window, but does NOT delete the OpenCode session. The session persists and can be matched to the closed beads ID.

2. **Detection mechanism: 30-minute idle window in `orch status`** - Sessions updated within 30 minutes are matched to beads IDs. Combined with the persistence bug, this causes recently completed agents to appear as "Completed" in status output.

3. **Precedent exists: `orch abandon` correctly cleans up sessions** - The fix is straightforward: copy the session deletion logic from `abandon_cmd.go:165-174` to `complete_cmd.go`.

**Answer to Investigation Question:**

The bug is caused by **missing OpenCode session deletion in `orch complete`**, not registry state issues. When `orch complete` runs:
1. It closes the beads issue ✓
2. It closes the tmux window ✓  
3. It does NOT delete the OpenCode session ✗

The persisted session is then matched by `orch status` (within 30-minute window), and since the beads issue is closed, marked as `IsCompleted = true`. This causes the agent to appear in `--all` output and be counted as "Completed" in the header.

The fix is to add OpenCode session deletion to `complete_cmd.go`, similar to how `abandon_cmd.go` does it.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch complete` does not call `client.DeleteSession()` (verified: searched `complete_cmd.go` for `DeleteSession`, no matches)
- ✅ `orch abandon` does call `client.DeleteSession()` (verified: read `abandon_cmd.go:165-174`, found session deletion code)
- ✅ 136 OpenCode sessions exist with orch-go beads IDs (verified: `curl localhost:4096/session | jq '[.[] | select(.title | test("orch-go-"))] | length'`)
- ✅ Filtering logic correctly filters completed agents with default flag (verified: code review of `status_cmd.go:389-404`)

**What's untested:**

- ⚠️ Live reproduction of the exact bug scenario (agent was already completed before investigation started)
- ⚠️ Whether session deletion in complete would break any existing workflows
- ⚠️ Whether there's a reason sessions were intentionally NOT deleted in complete (no comments explaining the omission)

**What would change this:**

- Finding would be wrong if there's intentional design reason for keeping sessions after completion
- Finding would be incomplete if there are other sources of stale state beyond OpenCode sessions (e.g., another registry file)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add OpenCode session deletion to `orch complete`** - Copy the session deletion logic from `abandon_cmd.go:165-174` to `complete_cmd.go` after tmux window cleanup.

**Why this approach:**
- Direct fix for root cause (session persistence)
- Proven pattern already exists in `orch abandon`
- Minimal code change, low risk

**Trade-offs accepted:**
- Sessions are permanently deleted (no "undo" after completion)
- This is acceptable because `orch complete` is intentional - user has verified work is done

**Implementation sequence:**
1. Find the OpenCode session ID for the completed agent (from `.session_id` file in workspace or from beads-to-session mapping)
2. Add `client.DeleteSession(sessionID)` call after tmux window cleanup (lines 531-537)
3. Add warning message on failure (non-fatal, like tmux cleanup)

### Alternative Approaches Considered

**Option B: Reduce 30-minute idle window in status**
- **Pros:** Faster stale state eviction without code changes to complete
- **Cons:** Doesn't address root cause, sessions still accumulate, could hide legitimately idle agents
- **When to use instead:** If there's a reason to preserve sessions for debugging/history

**Option C: Mark sessions as "completed" instead of deleting**
- **Pros:** Preserves session history for debugging
- **Cons:** Requires new session state tracking, more complex, doesn't solve accumulation
- **When to use instead:** If session preservation is valuable for post-mortem analysis

**Rationale for recommendation:** Option A directly fixes the root cause with minimal risk. The pattern is proven (`orch abandon` uses it). Sessions are only deleted after explicit `orch complete`, so there's no risk of accidental data loss.

---

### Implementation Details

**What to implement first:**
- Get session ID: Check workspace `.session_id` file first, fall back to finding session by beads ID title match
- Add session deletion after tmux cleanup (around line 537 in complete_cmd.go)
- Print status message: `"Deleted OpenCode session: %s\n", sessionID[:12]`

**Things to watch out for:**
- ⚠️ Session ID might not be available for older workspaces without `.session_id` file
- ⚠️ Session might already be deleted (handle 404 gracefully)
- ⚠️ For untracked agents, there may be no matching session

**Areas needing further investigation:**
- Whether there are workflows that rely on session persistence after completion
- How to handle cross-project agents (session in different project's directory)

**Success criteria:**
- ✅ After `orch complete`, running `orch status --all` should NOT show the completed agent
- ✅ OpenCode API should return 404 for the deleted session ID
- ✅ Verify with test: spawn agent, complete it, check `curl localhost:4096/session/{id}` returns 404

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Main completion logic, searched for session deletion (not found)
- `cmd/orch/status_cmd.go` - Agent collection and filtering logic
- `cmd/orch/abandon_cmd.go:165-174` - Reference implementation for session deletion
- `pkg/opencode/client.go` - OpenCode API client, including DeleteSession method
- `pkg/session/registry.go` - Orchestrator session registry (separate from OpenCode sessions)
- `pkg/verify/beads_api.go` - Beads API for issue status checking

**Commands Run:**
```bash
# Check current agents in status
orch status --all --json | jq '.agents'

# Count OpenCode sessions with orch-go beads IDs
curl -s localhost:4096/session | jq '[.[] | select(.title | test("orch-go-"))] | length'
# Result: 136

# Show recent orch-go sessions
curl -s localhost:4096/session | jq '[.[] | select(.title | test("orch-go-"))] | sort_by(.time.updated) | reverse | .[0:5]'

# Check beads issue status
bd show orch-go-wrrks
# Result: Status: closed

# Search for session deletion in complete command
rg -n "DeleteSession|delete.*session" cmd/orch/ --type go
```

**External Documentation:**
- None

**Related Artifacts:**
- **Prior knowledge:** "orch-go agent state exists in four layers (OpenCode memory, OpenCode disk, registry, tmux)" - From spawn context, confirms multi-layer state tracking
- **Prior knowledge:** "orch status can show phantom agents (tmux windows where OpenCode exited)" - Related but distinct issue (phantom = no session, this bug = session persists)

---

## Investigation History

**2026-01-06 21:30:** Investigation started
- Initial question: Why does `orch status --all` show completed agents after `orch complete`?
- Context: Example of orch-go-wrrks still showing as completed after successful `orch complete --force`

**2026-01-06 21:45:** Found root cause
- `orch complete` does not call `DeleteSession()` on the OpenCode session
- `orch abandon` does correctly delete sessions (reference implementation found)

**2026-01-06 22:00:** Confirmed with testing
- 136 OpenCode sessions persist with orch-go beads IDs
- Status correctly filters completed agents but they accumulate in session list

**2026-01-06 22:10:** Investigation completed
- Status: Complete
- Key outcome: Fix is to add session deletion to `complete_cmd.go`, copying pattern from `abandon_cmd.go`

---

## Self-Review

- [x] Real test performed (not code review) - Ran curl commands to check session counts, searched code for DeleteSession
- [x] Conclusion from evidence (not speculation) - Based on actual code search results and API responses
- [x] Question answered - Explained why completed agents appear (session persistence) and how to fix
- [x] File complete - All sections filled with concrete information
- [x] D.E.K.N. filled - Summary section has Delta, Evidence, Knowledge, Next, Promote to Decision

**Self-Review Status:** PASSED
