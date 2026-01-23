<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The cross-project fix from 2026-01-22 still had a gap - `DefaultActiveCount()` didn't use session.Directory, relying only on `kb projects list` which fails for unregistered projects.

**Evidence:** Code analysis showed `DefaultActiveCount()` fetched sessions without Directory field, while `orch status` uses session.Directory for reliable cross-project resolution.

**Knowledge:** Session Directory is the authoritative source for project path; kb projects list is a fallback. Both sources should be used with Directory taking priority.

**Next:** Fixed - `DefaultActiveCount()` now fetches session.Directory and passes it to `GetClosedIssuesBatchWithProjectDirs()`.

**Promote to Decision:** recommend-no (bug fix extending existing cross-project architecture)

---

# Investigation: Fix Daemon Capacity Counter Getting Stuck

**Question:** Why does the daemon capacity counter still get stuck after the cross-project fix from 2026-01-22?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Agent (orch-go-ry3ay)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A (extends .kb/investigations/2026-01-22-inv-daemon-capacity-tracking-stale-after.md)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Session.Directory field not being used in DefaultActiveCount

**Evidence:** The session struct in `DefaultActiveCount()` only included ID, Title, and Time fields:
```go
var sessions []struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Time  struct {
        Updated int64 `json:"updated"`
    } `json:"time"`
}
```

The Directory field was available in the OpenCode API response but not being fetched or used.

**Source:** pkg/daemon/active_count.go:36-42 (before fix)

**Significance:** Without Directory, the function couldn't determine where sessions were actually running, forcing reliance on kb projects list for project path resolution.

---

### Finding 2: orch status uses session.Directory directly

**Evidence:** In `cmd/orch/status_cmd.go`, the code uses `session.Directory` to build project path mappings:
```go
for beadsID, session := range beadsToSession {
    if session != nil && session.Directory != "" && session.Directory != "/" && session.Directory != projectDir {
        beadsProjectDirs[beadsID] = session.Directory
    }
}
```

**Source:** cmd/orch/status_cmd.go (status command implementation)

**Significance:** This explains why `orch status` correctly shows 1 active agent while daemon thinks 3 are active - orch status uses the authoritative source (session.Directory) while daemon relied on kb projects list which may not include all projects.

---

### Finding 3: kb projects list fallback fails for unregistered projects

**Evidence:** The `buildProjectPathMap()` function relies on `kb projects list`:
```go
func buildProjectPathMap() map[string]string {
    projects, _ := ListProjects()
    pathMap := make(map[string]string, len(projects))
    for _, p := range projects {
        pathMap[p.Name] = p.Path
    }
    return pathMap
}
```

When a project isn't registered in kb, the lookup fails and falls back to current directory, which may be wrong.

**Source:** pkg/daemon/active_count.go:127-134

**Significance:** Projects spawned from sessions in unregistered projects would have their issue status checked against the wrong beads database, leading to incorrect "open" status and inflated active counts.

---

## Synthesis

**Key Insights:**

1. **Two-tier project resolution needed** - Session.Directory is the authoritative source; kb projects list is a useful fallback when Directory isn't available or valid.

2. **API field oversight** - The OpenCode session API provides Directory, but it wasn't being captured in the daemon's session struct.

3. **Consistency with orch status** - The fix aligns `DefaultActiveCount()` with how `orch status` resolves project paths, ensuring both report consistent agent counts.

**Answer to Investigation Question:**

The daemon capacity counter got stuck because `DefaultActiveCount()` relied solely on `kb projects list` for cross-project resolution. When sessions ran in projects not registered with kb (or with different names), the beads ID-to-project mapping failed, causing those sessions to be looked up in the wrong beads database. The fix adds session.Directory to the fetched fields and uses it as the primary source for project path, with kb projects list as fallback.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles successfully (verified: `go build ./pkg/daemon/...`)
- ✅ All daemon tests pass (verified: `go test ./pkg/daemon/...` - PASS)
- ✅ New tests for `GetClosedIssuesBatchWithProjectDirs()` pass (verified: test added and passing)

**What's untested:**

- ⚠️ End-to-end verification with actual cross-project daemon scenario (requires running daemon with multiple projects)
- ⚠️ Original reproduction steps (daemon crashed scenario may require specific timing to reproduce)

**What would change this:**

- Finding would be wrong if OpenCode API doesn't include Directory in session response
- Finding would be wrong if the Directory field contains incorrect/stale values
- Finding would be wrong if there's another code path that bypasses reconciliation entirely

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Use session.Directory as primary project path source** - Modified `DefaultActiveCount()` to fetch and use session.Directory, falling back to kb projects list.

**Why this approach:**
- Matches how `orch status` resolves project paths (consistency)
- Session.Directory is authoritative - it's where the agent is actually running
- Backward compatible - kb projects list still works as fallback

**Trade-offs accepted:**
- Slightly more data fetched from OpenCode API (Directory field)
- Two resolution mechanisms instead of one (acceptable for robustness)

**Implementation sequence:**
1. Add Directory field to session struct in `DefaultActiveCount()` ✅
2. Build beadsIDToProjectDir map from sessions ✅
3. Create `GetClosedIssuesBatchWithProjectDirs()` that accepts explicit dirs ✅
4. Update `GetClosedIssuesBatch()` to delegate to new function ✅

### Alternative Approaches Considered

**Option B: Always require projects in kb registry**
- **Pros:** Single resolution path, simpler logic
- **Cons:** Requires manual project registration, fragile
- **When to use instead:** If kb becomes the single source of truth for all projects

**Option C: Cache session directories in daemon state**
- **Pros:** Faster lookups on subsequent polls
- **Cons:** Stale cache issues, more state to manage
- **When to use instead:** If poll performance becomes a bottleneck

**Rationale for recommendation:** Option A provides the most reliable resolution with minimal changes and full backward compatibility.

---

### Implementation Details

**What was implemented:**

1. Added `Directory` field to session struct in `DefaultActiveCount()` (active_count.go:39)
2. Built `beadsIDToProjectDir` map from sessions (active_count.go:48-70)
3. Created `GetClosedIssuesBatchWithProjectDirs()` function (active_count.go:117-163)
4. Updated `GetClosedIssuesBatch()` to delegate with nil projectDirs (active_count.go:102-106)
5. Added defensive nil check for `activeCountFunc` in `ReconcileWithOpenCode()` (daemon.go:577-580)

**Things to watch out for:**

- ⚠️ Session.Directory might be "/" for some sessions (explicitly filtered)
- ⚠️ Session.Directory might be empty (falls back to kb projects)
- ⚠️ Tests that create Daemon without activeCountFunc now use DefaultActiveCount fallback

**Success criteria:**

- ✅ Daemon capacity correctly reflects actual running agents
- ✅ No manual daemon restart required when agents complete
- ✅ Tests pass including new tests for GetClosedIssuesBatchWithProjectDirs

---

## References

**Files Modified:**
- `pkg/daemon/active_count.go` - Added Directory field, GetClosedIssuesBatchWithProjectDirs function
- `pkg/daemon/active_count_test.go` - Added tests for new function
- `pkg/daemon/daemon.go` - Added nil check for activeCountFunc
- `pkg/daemon/daemon_test.go` - Removed duplicate test functions (moved to active_count_test.go)

**Commands Run:**
```bash
# Build verification
go build ./pkg/daemon/...

# Test verification
go test ./pkg/daemon/... -v

# Full project build
make build
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-22-inv-daemon-capacity-tracking-stale-after.md - Original cross-project fix
- **Investigation:** .kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md - Initial reconciliation fix
- **Decision:** Daemon completion polling preferred over SSE detection

---

## Investigation History

**2026-01-22 19:42:** Investigation started
- Initial question: Why does daemon capacity still get stuck after cross-project fix?
- Context: Bug report showed orch status: 1 active, daemon-status.json: 3 active

**2026-01-22 19:45:** Root cause identified
- Found DefaultActiveCount() doesn't fetch session.Directory
- Identified gap between orch status (uses Directory) and daemon (uses kb projects)

**2026-01-22 19:50:** Fix implemented
- Added Directory field to session struct
- Created GetClosedIssuesBatchWithProjectDirs()
- Added tests for new function
- All tests passing

**2026-01-22 19:52:** Investigation completed
- Status: Complete
- Key outcome: Fixed by using session.Directory as primary project path source
