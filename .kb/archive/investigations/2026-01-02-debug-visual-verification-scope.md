<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Visual verification was checking project git history (last 5 commits) instead of agent-specific commits, causing false positives when prior agents/commits modified web/ files.

**Evidence:** `HasWebChangesInRecentCommits` used `git diff HEAD~5..HEAD` which includes all recent commits regardless of who made them. Tests pass after switching to spawn-time-based filtering.

**Knowledge:** Agent-scoped verification must use spawn time to filter commits, consistent with how constraint verification already scopes file matching.

**Next:** Close - fix implemented and tested.

---

# Investigation: Visual Verification Scope

**Question:** Why does visual verification check project git history (last 5 commits) instead of agent-specific changes?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** og-debug-visual-verification-checks-02jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: HasWebChangesInRecentCommits uses HEAD~5, not spawn time

**Evidence:** In `pkg/verify/visual.go:124-140`, the function checks `git diff --name-only HEAD~5..HEAD`. This includes all recent commits in the project, not just commits made by the current agent.

```go
// Get changed files from last 5 commits
cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
```

**Source:** `pkg/verify/visual.go:124-140`

**Significance:** When agent A commits to web/, then agent B completes (only modifying pkg/verify/), agent B fails visual verification because it sees agent A's web/ changes in the project history. This is the root cause of the false positives.

---

### Finding 2: Spawn time infrastructure already exists

**Evidence:** The spawn system already writes `.spawn_time` to each workspace (`pkg/spawn/session.go:110-155`), and constraint verification already uses this to scope file matching (`pkg/verify/constraint.go:127-131`).

```go
// VerifyConstraintsWithSpawnTime checks if all constraints are satisfied in the project directory.
// If spawnTime is non-zero, only files with mtime >= spawnTime are considered matches.
func VerifyConstraintsWithSpawnTime(constraints []Constraint, projectDir string, spawnTime time.Time)
```

**Source:** `pkg/spawn/session.go:110-155`, `pkg/verify/constraint.go:127-131`

**Significance:** The infrastructure for agent-scoped verification exists. Visual verification just needs to use the same pattern.

---

### Finding 3: Git supports time-based log filtering

**Evidence:** Git's `--since` flag allows filtering commits by time. Using `git log --since=<spawn_time> --name-only --format=` retrieves only files changed in commits after the spawn time.

**Source:** Git documentation, manual testing

**Significance:** We can scope git commit detection to agent-specific commits by using spawn time rather than a fixed commit count.

---

## Synthesis

**Key Insights:**

1. **Agent scope requires spawn time** - Using a fixed number of commits (HEAD~5) is fundamentally wrong for agent-scoped verification because it includes other agents' work.

2. **Pattern consistency** - Constraint verification already uses spawn time to scope file matching. Visual verification should follow the same pattern.

3. **Backward compatibility** - Legacy workspaces without `.spawn_time` should fall back to the old behavior to avoid breaking existing workflows.

**Answer to Investigation Question:**

The visual verification checks project git history because `HasWebChangesInRecentCommits` was implemented using `HEAD~5..HEAD`, which checks the last 5 project commits regardless of who made them. This was a scoping bug - the fix is to use the agent's spawn time to filter commits, only checking commits made since the agent was spawned.

---

## Structured Uncertainty

**What's tested:**

- `HasWebChangesForAgent` correctly reads spawn time and uses time-based filtering (verified: unit tests pass)
- `hasWebChangesSinceTime` correctly parses git log output for web files (verified: unit tests pass)
- Backward compatibility: no spawn time falls back to HEAD~5 behavior (verified: unit tests pass)
- Build compiles successfully (verified: `go build ./...`)

**What's untested:**

- Integration with actual `orch complete` workflow in real agent scenario (not tested in this session)
- Performance with very long git histories (not benchmarked)

**What would change this:**

- If git `--since` flag behaves differently on some systems
- If spawn time file is unreliable (written after commits start)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach (Implemented)

**Spawn-time-based commit filtering** - Use the workspace's `.spawn_time` file to scope git commit detection to agent-specific commits.

**Why this approach:**
- Directly addresses root cause (commit scope)
- Consistent with existing constraint verification pattern
- Uses existing infrastructure (spawn time already written)

**Trade-offs accepted:**
- Adds dependency on spawn package
- Falls back to old behavior for legacy workspaces

**Implementation sequence:**
1. Add `HasWebChangesForAgent(projectDir, workspacePath)` function
2. Add `hasWebChangesSinceTime(projectDir, since)` internal helper
3. Update `VerifyVisualVerification` to use new function
4. Deprecate `HasWebChangesInRecentCommits`

---

## References

**Files Modified:**
- `pkg/verify/visual.go` - Added `HasWebChangesForAgent`, `hasWebChangesSinceTime`, updated `VerifyVisualVerification`
- `pkg/verify/visual_test.go` - Added tests for new functions

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification  
go test -v ./pkg/verify/... -run "WebChanges"
go test ./pkg/verify/...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-debug-orch-review-shows-needs-review.md` - Prior false positive investigation (skill-based)
- **Investigation:** `.kb/investigations/2025-12-26-inv-ui-completion-gate-require-screenshot.md` - Human approval requirement

---

## Investigation History

**2026-01-02:** Investigation started
- Initial question: Why does `orch complete orch-go-bn9y` fail visual verification when agent only modified `pkg/verify/` code?
- Context: Agent was flagged for web/ changes from prior commits, not its own work

**2026-01-02:** Root cause identified
- Found `HasWebChangesInRecentCommits` uses HEAD~5, not spawn time
- Found spawn time infrastructure already exists in `pkg/spawn/session.go`

**2026-01-02:** Fix implemented and tested
- Added `HasWebChangesForAgent` with spawn-time-based filtering
- Updated `VerifyVisualVerification` to use new function
- All tests pass

**2026-01-02:** Investigation completed
- Status: Complete
- Key outcome: Visual verification now scoped to agent-specific commits via spawn time
