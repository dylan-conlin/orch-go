<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Untracked agents show inconsistent status because CLI uses beads comments for phase detection (which fail for untracked agents), while dashboard only uses session activity time.

**Evidence:** CLI shows "stalled" when NoComments=true AND session > 1 min old; dashboard has no equivalent check and shows "active" based solely on OpenCode session being alive.

**Knowledge:** Untracked agents need an alternative phase reporting mechanism since bd comment fails for fake beads IDs - workspace `.phase` files provide this mechanism.

**Next:** Implementation complete. Close issue. Monitor adoption of .phase file pattern by untracked agents.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Untracked Agents No Phase Visibility

**Question:** Why do untracked agents show "stalled" in CLI but "active" in dashboard, and how can we provide consistent phase visibility?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: CLI detects stalled via NoComments + time threshold

**Evidence:** In `cmd/orch/main.go:3394-3396`, the CLI uses a `NoComments` flag to mark agents as "⚠️ stalled":

```go
if agent.NoComments {
    // Agent has session but no beads comments after >1 min - potential failed-to-start
    return "⚠️ stalled"
}
```

The `NoComments` flag is set at lines 2822-2830 when:
1. No beads comments exist for the agent
2. Session is older than 1 minute

**Source:** `cmd/orch/main.go:2822-2830, 3394-3396`

**Significance:** CLI has stalled detection for tracked agents because it uses beads comments as a progress indicator. For untracked agents with fake beads IDs (e.g., `orch-go-untracked-1767118797`), the `bd comment` calls fail silently, so `NoComments` is always true → always shows as stalled.

---

### Finding 2: Dashboard lacks stalled detection for untracked agents

**Evidence:** In `cmd/orch/serve.go:787-792`, dashboard determines status purely from activity time:

```go
status := "active"
if timeSinceUpdate > deadThreshold {  // deadThreshold = 3 minutes
    status = "dead"
}
```

There's no equivalent of CLI's `NoComments` check. The only status options are "active", "dead", or "completed" (set later if Phase: Complete is found).

For untracked agents:
- They have fake beads IDs that don't exist in beads
- `bd comment` calls fail, so no Phase comments exist
- Dashboard shows "active" because session is alive
- No stalled warning because dashboard doesn't check for missing phase reports

**Source:** `cmd/orch/serve.go:787-792, 1070-1082`

**Significance:** Dashboard relies entirely on Phase comments from beads for work progress. Untracked agents can't report Phase (no real beads issue), so they appear healthy when they might be stuck.

---

### Finding 3: Untracked agents write fake beads IDs that always fail lookups

**Evidence:** In `cmd/orch/main.go:2220`, untracked agents get fake IDs:

```go
return fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix()), nil
```

When `bd comment <fake-id> "Phase: X"` runs, it fails silently because no beads issue exists. The dashboard's `commentsMap[agent.BeadsID]` returns empty, so Phase is never set.

Detection of untracked IDs exists in `cmd/orch/review.go:451-453`:
```go
func isUntrackedBeadsID(beadsID string) bool {
    return strings.Contains(beadsID, "-untracked-")
}
```

**Source:** `cmd/orch/main.go:2220`, `cmd/orch/review.go:451-453`

**Significance:** There's already logic to detect untracked agents by beads ID pattern. We can use this to apply different visibility heuristics for untracked agents.

---

## Synthesis

**Key Insights:**

1. **Phase reporting is beads-dependent** - Both CLI and dashboard rely on beads comments for phase visibility, but untracked agents have fake beads IDs that can't accept comments.

2. **Status detection differs between CLI and dashboard** - CLI has `NoComments` flag + time threshold for stalled detection; dashboard only checks session activity time without phase awareness.

3. **Workspace files provide alternative channel** - Untracked agents can write phase to `.phase` file in their workspace directory, bypassing the beads system entirely.

**Answer to Investigation Question:**

Untracked agents show inconsistent status because CLI explicitly checks for "no beads comments after >1 minute" to show "stalled", while dashboard has no equivalent check and defaults to "active" based solely on OpenCode session being alive. The fix involves two parts: (1) Dashboard now detects stalled untracked agents using the same time threshold heuristic, and (2) Spawn context instructs untracked agents to write phase to workspace `.phase` file as an alternative to beads comments.

---

## Structured Uncertainty

**What's tested:**

- ✅ `isUntrackedBeadsIDServe()` correctly identifies untracked beads IDs (unit tests pass)
- ✅ `readWorkspacePhase()` reads and trims phase from .phase file (unit tests pass)
- ✅ Dashboard stalled detection logic compiles and unit tests pass
- ✅ Spawn context template includes .phase file instructions for --no-track spawns (unit test verified)

**What's untested:**

- ⚠️ End-to-end behavior: spawn untracked agent → writes .phase → dashboard shows phase (not tested in CI)
- ⚠️ Agent compliance: whether agents actually write to .phase file when instructed (requires live spawn)
- ⚠️ Dashboard UI handling of "stalled" status for untracked agents (frontend changes may be needed)

**What would change this:**

- If agents ignore .phase instructions and never write the file, the alternative mechanism provides no value
- If dashboard frontend doesn't handle "stalled" status, the backend change won't be visible to users
- If performance degrades from reading .phase files for every agent on every poll, we may need caching

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐ (IMPLEMENTED)

**Dual-mechanism phase visibility** - Dashboard detects stalled untracked agents via time threshold, and agents can optionally report phase via workspace .phase file.

**Why this approach:**
- Matches CLI behavior (consistency between tools)
- Doesn't require beads system changes
- Workspace file is agent-controlled and doesn't depend on external services
- Backward compatible (old agents simply won't have .phase file)

**Trade-offs accepted:**
- `.phase` file is not persisted in git (workspace is gitignored)
- Agents must actively adopt new .phase writing behavior
- Dashboard UI may need updates to display "stalled" status visually

**Implementation sequence:**
1. Add `isUntrackedBeadsIDServe()` and `readWorkspacePhase()` to serve.go (done)
2. Add stalled detection logic for untracked agents in dashboard agent loop (done)
3. Update spawn context template to instruct untracked agents about .phase file (done)
4. Add tests for new functions (done)

### Alternative Approaches Considered

**Option B: Registry metadata storage**
- **Pros:** Centralized, doesn't require workspace file access
- **Cons:** Requires registry format changes, more complex state management
- **When to use instead:** If workspace access becomes unreliable

**Option C: Shadow beads issue creation**
- **Pros:** Would enable full beads tracking even for "untracked" agents
- **Cons:** Contradicts the purpose of --no-track flag, pollutes beads database
- **When to use instead:** Never - violates user intent

**Rationale for recommendation:** Workspace file is simple, doesn't require external services, and follows existing patterns (SYNTHESIS.md already uses workspace for completion detection).

---

### Implementation Details

**What was implemented:**

1. **serve.go changes:**
   - Added `isUntrackedBeadsIDServe()` - detects untracked agents by beads ID pattern
   - Added `readWorkspacePhase()` - reads phase from workspace .phase file
   - Updated agent processing loop to read .phase for untracked agents
   - Added stalled detection: untracked + no phase + >1 min old → status="stalled"

2. **pkg/spawn/context.go changes:**
   - Added `WorkspacePath` to contextData struct
   - Updated NoTrack template to include .phase file instructions
   - Agents now instructed to write phase via `echo 'Phase' > workspace/.phase`

3. **Tests added:**
   - `TestIsUntrackedBeadsIDServe` - verifies untracked ID detection
   - `TestReadWorkspacePhase` - verifies .phase file reading
   - `TestCheckWorkspaceSynthesisFunction` - existing function, additional coverage
   - Updated `TestGenerateContext_NoTrack` - verifies .phase instructions in spawn context

**Things to watch out for:**
- ⚠️ Dashboard frontend may need updates to visually indicate "stalled" status
- ⚠️ Agents must actually write to .phase file for visibility to work
- ⚠️ .phase file not committed to git (workspace is gitignored)

**Areas needing further investigation:**
- Dashboard UI changes to display "stalled" indicator for untracked agents
- Agent compliance verification (do agents follow .phase instructions?)

**Success criteria:**
- ✅ Dashboard shows "stalled" for untracked agents with no phase after 1 min (backend implemented)
- ✅ Spawn context includes .phase file instructions for --no-track spawns (implemented)
- ✅ All related tests pass (verified)

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added untracked agent detection, .phase file reading, stalled status logic
- `pkg/spawn/context.go` - Added WorkspacePath to template data, added .phase file instructions
- `cmd/orch/serve_test.go` - Added tests for new functions
- `pkg/spawn/context_test.go` - Updated NoTrack tests to verify .phase instructions

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/go build ./...

# Test execution
/opt/homebrew/bin/go test ./cmd/orch/... -run "Untracked|Phase|Synthesis" -v
/opt/homebrew/bin/go test ./pkg/spawn/... -run "NoTrack" -v
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-inv-fix-dashboard-completion-detection-untracked.md` - Prior work on untracked agent completion detection
- **Investigation:** `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stale-after.md` - Related issue with untracked agents affecting daemon capacity

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: Why do untracked agents show "stalled" in CLI but "active" in dashboard?
- Context: Agent orch-go-untracked-1767118797 had 45 messages but CLI showed stalled, dashboard showed active

**2025-12-30:** Root cause identified
- CLI uses NoComments flag + time threshold for stalled detection
- Dashboard only checks session activity time, has no phase awareness for untracked agents
- Untracked agents have fake beads IDs that can't accept comments

**2025-12-30:** Implementation completed
- Added stalled detection to dashboard for untracked agents (time threshold heuristic)
- Added workspace .phase file mechanism for alternative phase reporting
- Updated spawn context to instruct untracked agents about .phase file
- All tests pass

**2025-12-30:** Investigation completed
- Status: Complete
- Key outcome: Dashboard now shows "stalled" status for untracked agents with no phase after 1 min, and agents can optionally report phase via workspace .phase file
