<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** CLI's `orch status` now reads phase from `.phase` file for untracked agents, matching dashboard behavior.

**Evidence:** Build compiles, all tests pass (18.864s for cmd/orch). Phase detection tests specifically pass including `TestReadWorkspacePhase`.

**Knowledge:** Untracked agents write phase to workspace `.phase` file because they have synthetic beads IDs that don't exist in the database.

**Next:** Close - fix implemented and tested.

---

# Investigation: Untracked Agents Cannot Report Phase

**Question:** Why do untracked agents always show as stalled in `orch status`, and how can we fix phase detection for them?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** og-debug-untracked-agents-cannot-02jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Phase detection relies on beads comments only

**Evidence:** In `cmd/orch/main.go:2914-2939`, phase is extracted from beads comments via `verify.ParsePhaseFromComments`. For untracked agents, `commentsMap[ta.beadsID]` is empty because synthetic beads IDs like `project-untracked-1766695797` don't exist in the beads database.

**Source:** `cmd/orch/main.go:2921-2926`:
```go
comments, hasComments := commentsMap[ta.beadsID]
if hasComments && len(comments) > 0 {
    phaseStatus := verify.ParsePhaseFromComments(comments)
    if phaseStatus.Found {
        phase = phaseStatus.Phase
    }
}
```

**Significance:** This is the root cause - the CLI lacks the fallback mechanism that the dashboard has.

---

### Finding 2: Dashboard already has the solution

**Evidence:** In `cmd/orch/serve.go:1141-1149`, the dashboard uses `readWorkspacePhase()` to read phase from `.phase` file when beads comments are unavailable for untracked agents:

```go
// For untracked agents, try reading phase from workspace .phase file
if agents[i].Phase == "" && isUntrackedBeadsIDServe(agents[i].BeadsID) {
    workspacePath := wsCache.lookupWorkspace(agents[i].BeadsID)
    if wsPhase := readWorkspacePhase(workspacePath); wsPhase != "" {
        agents[i].Phase = wsPhase
    }
}
```

**Source:** `cmd/orch/serve.go:1141-1149`, `cmd/orch/serve.go:2805-2820`

**Significance:** The solution already exists and is tested - we just need to apply the same pattern to the CLI.

---

### Finding 3: Spawn context instructs untracked agents to write .phase file

**Evidence:** In `pkg/spawn/context.go:57`, untracked agents are instructed to write their phase to the `.phase` file:
```go
`echo 'Planning' > {{.WorkspacePath}}/.phase`
```

And completion protocol at line 67:
```go
1. Run: `echo 'Complete' > {{.WorkspacePath}}/.phase` (report phase FIRST - before commit)
```

**Source:** `pkg/spawn/context.go:57, 67, 73, 320, 326`

**Significance:** Untracked agents ARE writing their phase - the CLI just wasn't reading it.

---

## Synthesis

**Key Insights:**

1. **Parallel code paths** - The CLI (`runStatus`) and dashboard (`handleGetAgents`) both process agent phase, but only the dashboard had the untracked fallback.

2. **Existing infrastructure** - All the helper functions (`isUntrackedBeadsID`, `readWorkspacePhase`, `wsCache.lookupWorkspace`) already existed and are in the same package.

3. **Simple parity fix** - The fix required adding ~8 lines of code in two places to match the dashboard's behavior.

**Answer to Investigation Question:**

Untracked agents showed as stalled because the CLI's `runStatus` function only checked beads comments for phase status. Since untracked agents have synthetic beads IDs that don't exist in the database, no comments are ever found. The fix adds the same fallback that the dashboard uses: when phase is empty and the agent is untracked, read the phase from the workspace's `.phase` file.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build compiles without errors (verified: `go build ./cmd/orch/...`)
- ✅ All existing tests pass (verified: `go test ./...` - 18.864s for cmd/orch)
- ✅ Phase reading tests pass (verified: `TestReadWorkspacePhase` passes)
- ✅ Untracked ID detection tests pass (verified: `TestIsUntrackedBeadsID` passes)

**What's untested:**

- ⚠️ End-to-end with a real untracked agent (would require spawning an actual untracked agent and checking `orch status`)
- ⚠️ Interaction with cross-project untracked agents (untested but should work since we use `wsCache.lookupWorkspace`)

**What would change this:**

- If the `.phase` file format changes (currently just phase name with newline)
- If untracked agents stop writing to `.phase` file
- If workspace path lookup fails for untracked agents

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Implemented Approach

**Add .phase file fallback to CLI runStatus** - When phase is empty and agent is untracked, read phase from workspace `.phase` file.

**Changes made:**

1. **For tmux agents** (`cmd/orch/main.go:~2940-2948`):
```go
// For untracked agents, try reading phase from workspace .phase file
if phase == "" && isUntrackedBeadsID(ta.beadsID) {
    workspacePath := wsCache.lookupWorkspace(ta.beadsID)
    if wsPhase := readWorkspacePhase(workspacePath); wsPhase != "" {
        phase = wsPhase
        noComments = false // They did report, just via .phase file
    }
}
```

2. **For OpenCode agents** (`cmd/orch/main.go:~3020-3028`):
Same pattern applied to OpenCode-only agents.

**Why this approach:**
- Matches dashboard behavior exactly
- Uses existing, tested infrastructure
- Minimal code changes (< 20 lines)

---

## References

**Files Examined:**
- `cmd/orch/main.go:2695-3030` - CLI runStatus function
- `cmd/orch/serve.go:1130-1190` - Dashboard handleGetAgents
- `cmd/orch/serve.go:2798-2820` - isUntrackedBeadsIDServe and readWorkspacePhase
- `cmd/orch/review.go:451-455` - isUntrackedBeadsID
- `pkg/spawn/context.go:57-73` - Untracked agent phase reporting instructions
- `pkg/verify/check.go:124-131` - GetPhaseStatus

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/go build ./cmd/orch/...

# Run all tests
/opt/homebrew/bin/go test ./... 
# Result: PASS - 18.864s for cmd/orch

# Run specific phase/untracked tests
/opt/homebrew/bin/go test ./cmd/orch/... -v -run "Untrack|Phase"
# Result: All passed
```

---

## Investigation History

**2026-01-02:** Investigation started
- Initial question: Why do untracked agents show stalled in orch status?
- Context: Phase detection relies on beads comments which don't exist for untracked agents

**2026-01-02:** Root cause identified
- CLI lacks .phase file fallback that dashboard has

**2026-01-02:** Fix implemented
- Added .phase file fallback for both tmux and OpenCode agent paths
- All tests passing

**2026-01-02:** Investigation completed
- Status: Complete
- Key outcome: CLI now reads phase from .phase file for untracked agents, matching dashboard behavior
