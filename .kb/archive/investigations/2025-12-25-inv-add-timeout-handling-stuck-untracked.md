<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added timeout/staleness filtering to `orch review` - default output now shows only actionable agents.

**Evidence:** Before: 235 completions displayed (20+ untracked, 8+ stale). After: 134 actionable shown, 101 hidden with clear summary.

**Knowledge:** Agents with `-untracked-` beads IDs or >24h in non-Complete phase are noise, not actionable work.

**Next:** Close - implementation complete with tests.

**Confidence:** High (90%) - tested manually and with unit tests.

---

# Investigation: Add Timeout Handling Stuck Untracked

**Question:** How can we filter out stale/untracked agents from default `orch review` output to show only actionable items?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Untracked agents clutter review output

**Evidence:** Running `orch review` showed 235 completions, with 44 containing "untracked" in their beads ID.

**Source:** `orch review 2>&1 | grep -c "untracked"` returned 44

**Significance:** Untracked agents (spawned with `--no-track`) have fake beads IDs like `orch-go-untracked-1766695797` and will never complete properly since they can't report Phase via beads comments.

---

### Finding 2: Stale agents stuck in non-Complete phases for days

**Evidence:** Many agents from Dec 20-22 still showing in review despite being abandoned.

**Source:** `ls -l .orch/workspace/ | grep "Dec 20"` showed 87 old workspaces

**Significance:** Agents stuck in phases like "Implementing" or "Planning" for >24h are effectively dead and should be filtered from default view.

---

### Finding 3: Existing codebase has detection patterns

**Evidence:** Found `projectName-untracked-timestamp` pattern in cmd/orch/main.go:1685

**Source:** `grep -n "untracked" cmd/orch/*.go`

**Significance:** The untracked ID format is consistent and detectable via string matching.

---

## Implementation

### Changes Made

1. **CompletionInfo struct** - Added `ModTime`, `IsUntracked`, `IsStale` fields
2. **Helper functions** - `isUntrackedBeadsID()` and `isStaleAgent()` 
3. **Command flags** - Added `--stale` and `--all` flags
4. **Filtering logic** - Default filters out stale/untracked, with summary of hidden count
5. **Status display** - Shows `[STALE]` and `[UNTRACKED]` tags
6. **Tests** - 5 new test functions covering detection and filtering logic

### Files Modified

- `cmd/orch/review.go` - Core implementation
- `cmd/orch/review_test.go` - Test coverage

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
- All tests pass including new tests
- Manual testing confirms expected behavior
- Simple string matching for detection is reliable

**What's certain:**

- ✅ Untracked detection works for all `*-untracked-*` beads IDs
- ✅ Staleness detection correctly uses 24h threshold
- ✅ Filtering flags work as documented

**What's uncertain:**

- ⚠️ ModTime uses directory modification time which may not reflect actual agent activity
- ⚠️ 24h threshold is arbitrary but reasonable

---

## References

**Files Examined:**
- `cmd/orch/review.go` - Main implementation
- `cmd/orch/main.go` - Untracked ID generation pattern

**Commands Run:**
```bash
# Count untracked agents
orch review 2>&1 | grep -c "untracked"

# Test new flags
orch review              # Default: 134 actionable
orch review --stale      # 101 stale/untracked
orch review --all        # 235 total

# Run tests
go test ./cmd/orch -run "TestIsUntracked|TestIsStale" -v
```

---

## Investigation History

**2025-12-25 21:58:** Investigation started
- Initial question: How to filter cluttered review output
- Context: 20+ untracked and 8+ stale agents made review output unreadable

**2025-12-25 22:30:** Implementation complete
- Added detection helpers, flags, filtering logic, and tests
- All tests passing
- Manual verification confirms expected behavior
