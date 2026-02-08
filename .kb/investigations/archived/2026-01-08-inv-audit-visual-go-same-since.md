<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** visual.go has the same bug pattern - `hasWebChangesSinceTime()` checks ALL commits since spawn time, not just workspace-specific commits.

**Evidence:** Code at visual.go:172-186 uses `git log --since=` without workspace filtering, identical to the bug pattern in test_evidence.go that was fixed with `HasCodeChangesSinceSpawnForWorkspace()`.

**Knowledge:** When multiple agents run concurrently, using `--since` without workspace scoping creates false positives - agents detect each other's commits as their own changes.

**Next:** Implement `HasWebChangesForWorkspace()` following the same pattern as the test_evidence.go fix.

**Promote to Decision:** recommend-no - Tactical bug fix following established pattern

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

# Investigation: Audit Visual Go Same Since

**Question:** Does visual.go have the same `--since` bug pattern as test_evidence.go where it checks ALL commits since spawn time instead of workspace-specific commits?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent (feature-impl)
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** Discovered during orch-go-vsdz3 completion
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: visual.go uses --since without workspace scoping

**Evidence:** The `hasWebChangesSinceTime()` function at lines 172-186 uses:
```go
cmd := exec.Command("git", "log", "--since="+sinceStr, "--name-only", "--format=")
cmd.Dir = projectDir
```
This gets ALL commits since spawn time, regardless of which agent/workspace made them.

**Source:** `pkg/verify/visual.go:172-186`

**Significance:** This is the exact same bug pattern that was fixed in test_evidence.go. When multiple agents run concurrently and have similar spawn times, each agent's visual verification gate will see the other agents' web/ commits, causing false positives.

---

### Finding 2: test_evidence.go fix pattern exists

**Evidence:** The test_evidence.go fix added `HasCodeChangesSinceSpawnForWorkspace()` which:
1. Gets commit hashes that touch the workspace directory using `git log --since=... --format=%H -- <workspace>`
2. For those commits, gets all changed files
3. Only reports code changes from workspace-touching commits

**Source:** `pkg/verify/test_evidence.go:208-285`

**Significance:** This pattern can be directly applied to visual.go. The key insight is filtering commits to only those that touch the workspace directory, not just time-based filtering.

---

### Finding 3: HasWebChangesForAgent already receives workspacePath

**Evidence:** The function signature `HasWebChangesForAgent(projectDir, workspacePath string)` already receives the workspace path but only uses it to read spawn time:
```go
func HasWebChangesForAgent(projectDir, workspacePath string) bool {
    spawnTime := spawn.ReadSpawnTime(workspacePath)
    // ... only uses spawnTime, ignores workspacePath for git filtering
    return hasWebChangesSinceTime(projectDir, spawnTime)
}
```

**Source:** `pkg/verify/visual.go:159-169`

**Significance:** The workspace path is available - we just need to pass it through and use it for git filtering like test_evidence.go does.

---

## Synthesis

**Key Insights:**

1. **Bug pattern confirmed** - visual.go has the exact same `--since` bug as test_evidence.go. The `hasWebChangesSinceTime()` function checks ALL commits since spawn time, not workspace-specific commits.

2. **Fix pattern established** - test_evidence.go already has the fix pattern: `hasCodeChangesInWorkspaceCommits()` filters commits to only those touching the workspace directory before checking for code changes.

3. **Minimal change required** - The workspace path is already passed to `HasWebChangesForAgent()`, it just isn't being used for filtering. The fix requires adding a similar `hasWebChangesInWorkspaceCommits()` function.

**Answer to Investigation Question:**

Yes, visual.go has the same bug. The `hasWebChangesSinceTime()` function (lines 172-186) uses `git log --since=` without workspace filtering. When multiple agents run concurrently with similar spawn times, each agent's visual verification gate sees all agents' web/ commits, causing false positives where agents are incorrectly flagged as needing visual verification for web/ changes they didn't make.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles successfully (verified: `go build ./...` passed)
- ✅ All visual verification tests pass (verified: `go test ./pkg/verify/...` all pass)
- ✅ New workspace-scoped function follows same pattern as test_evidence.go fix

**What's untested:**

- ⚠️ Not tested with concurrent agents in real scenario (requires multi-agent test setup)
- ⚠️ Performance impact of multiple git commands per check (not benchmarked)

**What would change this:**

- Finding would be wrong if workspace path filtering doesn't correctly scope to agent's commits
- Finding would be wrong if git log behavior differs on non-standard setups

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Apply same fix pattern as test_evidence.go** - Add workspace-scoped filtering to visual.go's web changes detection.

**Why this approach:**
- Proven pattern: test_evidence.go fix (orch-go-vsdz3) already validated this approach
- Minimal change: Only need to add workspace path parameter and filtering logic
- Maintains backward compatibility: Falls back to unscoped check when workspace path is empty

**Trade-offs accepted:**
- Additional git commands per check (one to get workspace-touching commits, one per commit for changed files)
- Acceptable because visual verification check runs once per completion, not in hot path

**Implementation sequence:**
1. Add `hasWebChangesSinceTimeForWorkspace()` function following `hasCodeChangesInWorkspaceCommits()` pattern
2. Update `HasWebChangesForAgent()` to call workspace-scoped version
3. Mark `hasWebChangesSinceTime()` as deprecated for documentation
4. Add tests for new workspace-scoped behavior

### Implementation Status: COMPLETE

Fix has been implemented and all tests pass.

**Files modified:**
- `pkg/verify/visual.go` - Added `hasWebChangesSinceTimeForWorkspace()`, updated `HasWebChangesForAgent()`
- `pkg/verify/visual_test.go` - Added tests for new workspace-scoped behavior

**Success criteria:**
- ✅ Code compiles: `go build ./...` passes
- ✅ All tests pass: `go test ./pkg/verify/...` passes
- ✅ Follows established pattern from test_evidence.go fix

---

## References

**Files Examined:**
- `pkg/verify/visual.go` - Main file with the bug, needed to understand current implementation
- `pkg/verify/test_evidence.go` - Reference for the fix pattern (HasCodeChangesSinceSpawnForWorkspace)
- `pkg/verify/visual_test.go` - Added tests for new behavior

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification  
go test ./pkg/verify/... -v -run "Visual|WebChange" -count=1
```

**Related Artifacts:**
- **Prior fix:** orch-go-vsdz3 - The test_evidence.go fix that established the pattern
- **Related bug:** The same pattern bug was discovered during completion of orch-go-vsdz3

---

## Investigation History

**2026-01-08:** Investigation started
- Initial question: Does visual.go have the same --since bug pattern as test_evidence.go?
- Context: Discovered during orch-go-vsdz3 completion review

**2026-01-08:** Bug confirmed
- Found `hasWebChangesSinceTime()` uses `--since` without workspace filtering at line 172-186
- Identical pattern to the bug that was fixed in test_evidence.go

**2026-01-08:** Fix implemented
- Added `hasWebChangesSinceTimeForWorkspace()` following established pattern
- Updated `HasWebChangesForAgent()` to use workspace-scoped version
- All tests pass

**2026-01-08:** Investigation completed
- Status: Complete
- Key outcome: Bug confirmed and fixed, visual.go now scopes web changes detection to workspace-specific commits
