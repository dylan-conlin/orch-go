<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrators should use workspace-based tracking with light session registry - beads tracking is fundamentally misaligned with interactive session semantics.

**Evidence:** Analyzed spawn/complete flows - orchestrators already skip beads phase reporting, use .orchestrator markers, and complete via workspace name lookup. Beads adds friction without value for sessions.

**Knowledge:** Key insight: beads tracks work items (spawn→task→complete), sessions track conversations (start→interact→end). Different lifecycles require different tracking mechanisms.

**Next:** Implement Option A: Workspace-Based Session Registry - extend existing .session_id/.workspace_name files into lightweight registry at ~/.orch/sessions.json.

---

# Investigation: Orchestrator Session Lifecycle Without Beads Tracking

**Question:** If we remove beads tracking for orchestrators, how do we identify sessions for `orch complete`, show status in `orch status`, know when ready for completion, and maintain transcript export?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - decision record needed
**Status:** Complete

**Related-From:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md`
**Related-From:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md`

---

## Problem Framing

### Design Question

If we remove beads tracking for orchestrators, how do we:
1. Identify orchestrator sessions for `orch complete`
2. Show orchestrator status in `orch status`
3. Know when an orchestrator is ready to be completed
4. Maintain transcript export on complete

### Key Insight from Dylan

> "Orchestrators aren't issues being worked on - they're interactive sessions with Dylan. Beads is for tracking work items, not collaborative sessions."

This insight reveals a fundamental **semantic mismatch**: Beads tracks work items with a spawn→task→complete lifecycle, while orchestrator sessions have a start→interact→end lifecycle. They're different things.

### Success Criteria

A good solution should:
1. **Enable identification** - `orch complete <workspace>` can find the session
2. **Enable status visibility** - `orch status` shows orchestrator sessions alongside workers
3. **Signal completion readiness** - Detectable when SESSION_HANDOFF.md exists
4. **Preserve transcript export** - Works with tmux or HTTP session

---

## Findings

### Finding 1: Current State is Contradictory

**Evidence:** The current implementation already has partial orchestrator support that's inconsistent:

| Aspect | Worker | Orchestrator (Current) | Issue |
|--------|--------|------------------------|-------|
| Beads Issue | Created on spawn | Created on spawn | ❌ Orchestrators have issues they don't use |
| Phase Reporting | `bd comment "Phase: X"` | Skipped | ✅ Already different |
| Completion Signal | Phase: Complete | SESSION_HANDOFF.md | ✅ Already different |
| Workspace Lookup | .beads_id file | .orchestrator marker | ✅ Already different |
| `orch complete` | By beads ID | By workspace name | ✅ Already supported |

**Source:** 
- `pkg/spawn/orchestrator_context.go:213` - "Orchestrators do NOT write .beads_id"
- `cmd/orch/complete_cmd.go:111` - `isOrchestratorWorkspace()` check
- `cmd/orch/shared.go:299-312` - Orchestrator detection via marker files

**Significance:** The codebase already treats orchestrators differently. The beads issue creation is vestigial - it's created but then ignored. Removing it would simplify the model.

---

### Finding 2: Workspace Files Already Provide Session Identity

**Evidence:** Orchestrator workspaces already write identity files:

```
.orch/workspace/{workspace-name}/
├── ORCHESTRATOR_CONTEXT.md    # Session context (like SPAWN_CONTEXT.md)
├── .orchestrator              # Marker file (orchestrator vs worker)
├── .session_id                # OpenCode session ID
├── .workspace_name            # Workspace name for lookup
├── .spawn_time                # Spawn timestamp
└── SESSION_HANDOFF.md         # Completion artifact (when ready)
```

**Source:** `pkg/spawn/orchestrator_context.go:200-211`

**Significance:** All the identity information needed for lifecycle management is already in the workspace. No beads ID is required - the workspace name IS the identity.

---

### Finding 3: `orch status` Already Has Multi-Source Detection

**Evidence:** Status command collects agents from multiple sources:

1. **tmux windows** - Primary source, extracts beads ID from window name
2. **OpenCode sessions** - Secondary, extracts beads ID from session title
3. **Workspace files** - Used for cross-project context

The code pattern:
```go
// Phase 1: Collect agents from tmux windows
for _, w := range windows {
    beadsID := extractBeadsIDFromWindowName(w.Name)
    // ...
}
```

**Source:** `cmd/orch/status_cmd.go:156-183`

**Significance:** The status command could add a third source: scan `.orch/workspace/` for `.orchestrator` markers. This would find orchestrator sessions without beads IDs.

---

### Finding 4: Completion Detection is Already Workspace-Based

**Evidence:** For orchestrators, `orch complete` already:
1. Takes workspace name as identifier (not beads ID)
2. Checks `isOrchestratorWorkspace()` via marker file
3. Verifies SESSION_HANDOFF.md exists
4. Skips beads-dependent verification

```go
if isOrchestratorWorkspace(workspacePath) {
    isOrchestratorSession = true
    fmt.Printf("Orchestrator session: %s\n", agentName)
}
```

**Source:** `cmd/orch/complete_cmd.go:111-113`, `pkg/verify/check.go:87-130`

**Significance:** The completion flow for orchestrators is already separate from beads. The only remaining beads usage is creating an issue on spawn (which is then ignored).

---

### Finding 5: The Four Requirements Can All Be Met Without Beads

| Requirement | Current (with Beads) | Proposed (without Beads) |
|-------------|---------------------|-------------------------|
| **Identify for complete** | Workspace name (already) | Workspace name (no change) |
| **Show in status** | Beads ID in window name | Workspace name or session registry |
| **Know when ready** | SESSION_HANDOFF.md (already) | SESSION_HANDOFF.md (no change) |
| **Transcript export** | tmux window by beads ID | tmux window by workspace name |

**Source:** Analysis of requirements vs existing implementation

**Significance:** Only the status visibility requires new implementation. All other requirements are already met by existing orchestrator infrastructure.

---

## Synthesis

### Key Insights

1. **Semantic mismatch is real** - Beads tracks work items (issues), not sessions. Forcing orchestrator sessions into issue tracking creates friction without benefit.

2. **Most infrastructure already exists** - The orchestrator spawn/complete flow is already beads-independent except for issue creation (which is vestigial).

3. **The missing piece is status visibility** - `orch status` needs a way to discover orchestrator sessions without beads IDs in window names.

4. **Workspace IS the identity** - The workspace directory name (e.g., `og-orch-goal-04jan`) serves as unique identifier. No external registry needed.

### Answer to Investigation Question

**How do we maintain orchestrator lifecycle without beads tracking?**

1. **Identify for `orch complete`**: Already works via workspace name - `orch complete og-orch-goal-04jan`

2. **Show in `orch status`**: Add workspace scanning - look for `.orchestrator` markers in `.orch/workspace/` and include those sessions

3. **Know when ready**: Check for `SESSION_HANDOFF.md` in workspace - already implemented in verification

4. **Transcript export**: Use workspace name to find tmux window (already included in window name via spawn.Config.WorkspaceName)

---

## Structured Uncertainty

**What's tested:**

- ✅ Workspace marker files work for orchestrator detection (verified: `isOrchestratorWorkspace()` tests pass)
- ✅ Completion by workspace name works (verified: `complete_cmd.go` handles workspace names)
- ✅ SESSION_HANDOFF.md verification exists (verified: `pkg/verify/check.go` has `VerifySessionHandoff()`)
- ✅ Workspace files contain session ID for API lookups (verified: `.session_id` file)

**What's untested:**

- ⚠️ Status command workspace scanning for orchestrator sessions (needs implementation)
- ⚠️ tmux window lookup by workspace name without beads ID (may need adjustment)
- ⚠️ Event logging without beads ID (currently logs beads_id field)

**What would change this:**

- If orchestrators need issue-level tracking (dependencies, priority) → keep beads but change issue type
- If workspace scanning proves too slow → add lightweight session registry file
- If tmux window lookup fails without beads ID → adjust window naming convention

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Option A: Workspace-Based Session Registry** - Create a lightweight `~/.orch/sessions.json` registry that tracks active orchestrator sessions without beads.

**Why this approach:**
- Solves status visibility problem efficiently (O(1) lookup vs O(n) workspace scan)
- Provides central source of truth for session state
- Minimal code changes - registry is additive
- Workspace remains authoritative for identity/artifacts

**Trade-offs accepted:**
- New state file to manage (but it's local, not cross-system)
- Registry could get stale (mitigated by verification on lookup)

**Implementation sequence:**

1. **Phase 1: Create session registry** (`pkg/session/registry.go`)
   - JSON file at `~/.orch/sessions.json`
   - Fields: workspace_name, session_id, spawn_time, project_dir, status (active/complete)
   - Lock file for concurrent access

2. **Phase 2: Update spawn to register** (`pkg/spawn/orchestrator_context.go`)
   - On orchestrator spawn, register in sessions.json
   - Skip beads issue creation for orchestrator skills

3. **Phase 3: Update status to read registry** (`cmd/orch/status_cmd.go`)
   - Add orchestrator session source from registry
   - Show alongside worker agents with different visualization

4. **Phase 4: Update complete to unregister** (`cmd/orch/complete_cmd.go`)
   - On orchestrator complete, remove from registry
   - Preserve transcript export (already workspace-based)

### Alternative Approaches Considered

**Option B: Workspace Scanning Only**
- **Pros:** No new state file, workspace is single source of truth
- **Cons:** O(n) scan on every status call, slower for large workspaces
- **When to use:** If simplicity outweighs performance, small number of sessions

**Option C: Extend Beads with Session Type**
- **Pros:** Unified tracking, beads already works
- **Cons:** Semantic mismatch remains, beads owns lifecycle
- **When to use:** If dependency tracking needed for orchestrators

**Option D: tmux Session as Registry**
- **Pros:** Already exists, no new file
- **Cons:** Only works for tmux mode, not headless orchestrators
- **When to use:** If orchestrators always use tmux (not headless)

**Rationale for recommendation:** Option A provides fast lookups (important for `orch status` responsiveness) while maintaining workspace as authoritative source. The registry is a cache, not ground truth - can always be rebuilt from workspaces if corrupted.

---

### Implementation Details

**What to implement first:**
- Session registry with basic CRUD operations
- Update spawn to skip beads for orchestrator skills
- Update status to include orchestrator sessions from registry

**Things to watch out for:**
- ⚠️ Lock file contention if multiple orchestrators spawn simultaneously
- ⚠️ Stale entries if orchestrator crashes without completing
- ⚠️ Window naming needs workspace name visible (for tmux lookup)

**Areas needing further investigation:**
- Should registry include goal/focus for status display?
- How to handle orphaned sessions (crash cleanup)?
- Should headless orchestrators be supported?

**Success criteria:**
- ✅ `orch spawn orchestrator "goal"` creates workspace but not beads issue
- ✅ `orch status` shows orchestrator sessions alongside workers
- ✅ `orch complete <workspace>` works without beads ID
- ✅ No beads phase reporting required for orchestrators
- ✅ Transcript export preserved

---

## References

**Files Examined:**
- `pkg/spawn/orchestrator_context.go` - Orchestrator spawn implementation
- `cmd/orch/spawn_cmd.go` - Spawn command, skill routing
- `cmd/orch/complete_cmd.go` - Complete command, orchestrator detection
- `cmd/orch/status_cmd.go` - Status command, agent collection
- `cmd/orch/shared.go` - Shared utilities including `isOrchestratorWorkspace()`
- `pkg/verify/check.go` - Completion verification, tier handling

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Prior decision on orchestrator sessions
- **Investigation:** `.kb/investigations/2026-01-04-inv-spawnable-orchestrator-sessions-infrastructure-changes.md` - Infrastructure for spawnable orchestrators
- **Investigation:** `.kb/investigations/2026-01-04-design-meta-orchestrator-architecture-spawnable-orchestrator.md` - Meta-orchestrator architecture

---

## Investigation History

**2026-01-05 10:48:** Investigation started
- Initial question: How to manage orchestrator lifecycle without beads tracking?
- Context: Dylan's insight that orchestrators are sessions, not issues

**2026-01-05 11:00:** Explored current state
- Found contradictory implementation (beads issues created but unused)
- Identified five key findings about existing orchestrator infrastructure

**2026-01-05 11:15:** Synthesis complete
- Answered four requirements from SPAWN_CONTEXT
- Recommended workspace-based session registry approach

**2026-01-05 11:30:** Investigation complete
- Status: Complete
- Key outcome: Recommend Option A (Workspace-Based Session Registry) to replace vestigial beads tracking for orchestrators
