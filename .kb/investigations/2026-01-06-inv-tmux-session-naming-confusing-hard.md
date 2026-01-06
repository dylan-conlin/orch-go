<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Meta-orchestrators and regular orchestrators were sharing the same "orchestrator" tmux session, making it hard to distinguish between them at a glance.

**Evidence:** Running `tmux list-windows -t orchestrator` showed both meta-orch and regular orch windows mixed together.

**Knowledge:** Following the workers-{project} per-type pattern, creating a separate "meta-orchestrator" session provides clear visual separation without requiring careful reading of workspace names.

**Next:** Implementation complete - meta-orchestrators now spawn in "meta-orchestrator" session, orchestrators in "orchestrator" session.

---

# Investigation: Tmux Session Naming Confusing Hard

**Question:** How to distinguish orchestrator vs meta-orchestrator in tmux sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent (og-feat-tmux-session-naming-06jan-5062)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Both orchestrator types used same tmux session

**Evidence:** Running `tmux list-windows -t orchestrator` showed:
```
1:⚙️ meta-orch-resume-meta-orch-06jan-9172 [orch-go-untracked-1767728799]
2:⚙️ pw-orch-resume-price-watch-06jan-bcd7 [price-watch-untracked-1767720722]
```

**Source:** `pkg/tmux/tmux.go:253-278` - OrchestratorSessionName constant and EnsureOrchestratorSession function

**Significance:** The confusion arose because looking at the "orchestrator" tmux session, you couldn't easily tell which windows were meta-orchestrators vs regular orchestrators without reading workspace names carefully.

---

### Finding 2: Workspace naming already provided visual distinction

**Evidence:** Meta-orchestrators use `meta-orch-*` prefix, regular orchestrators use `{project}-orch-*` prefix in workspace names.

**Source:** `pkg/spawn/config.go:193-238` - GenerateWorkspaceName function with WorkspaceNameOptions

**Significance:** The prefix distinction exists but requires reading workspace names. A session-level separation would be more immediately obvious.

---

### Finding 3: Workers pattern provides the precedent

**Evidence:** Workers use per-project sessions: `workers-{project}` (e.g., `workers-orch-go`, `workers-price-watch`)

**Source:** `pkg/tmux/tmux.go:172-175` - GetWorkersSessionName function

**Significance:** The codebase already has a pattern for type-based session separation. Applying the same pattern to meta-orchestrators is consistent.

---

## Synthesis

**Key Insights:**

1. **Session-level separation is more immediate** - You see session names in `tmux ls` before you see window names. Having "orchestrator" vs "meta-orchestrator" sessions makes the hierarchy visible at a glance.

2. **Consistent with existing patterns** - Workers use per-project sessions (workers-{project}). Meta-orchestrators getting their own session follows this per-type pattern.

3. **Minimal code change** - Only needed to add a new constant, new ensure function, and update the spawn logic and window finder functions.

**Answer to Investigation Question:**

Create a separate "meta-orchestrator" tmux session for meta-orchestrator spawns. This provides immediate visual distinction when listing sessions (`tmux ls`) and follows the existing per-type session pattern used for workers.

---

## Structured Uncertainty

**What's tested:**

- ✅ SessionNameConstants test verifies OrchestratorSessionName != MetaOrchestratorSessionName
- ✅ Build succeeds after changes
- ✅ Existing tmux tests pass

**What's untested:**

- ⚠️ Full end-to-end spawn of meta-orchestrator into new session (requires interactive testing)
- ⚠️ Window finder functions correctly search meta-orchestrator session (unit test passes but no integration test)

**What would change this:**

- Finding would be wrong if meta-orchestrators need to see orchestrator windows in same session for coordination
- Might need adjustment if too many sessions become unwieldy

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Separate meta-orchestrator session** - Add `meta-orchestrator` tmux session for meta-orchestrator spawns

**Why this approach:**
- Immediate visual distinction at session level
- Consistent with workers-{project} per-type pattern
- Minimal code changes required

**Trade-offs accepted:**
- One more session to manage
- Slightly more complex session search logic

**Implementation sequence:**
1. Add MetaOrchestratorSessionName constant and EnsureMetaOrchestratorSession ✅
2. Update spawn_cmd.go to route meta-orch to new session ✅
3. Update FindWindowBy* functions to search meta-orchestrator session ✅
4. Add tests ✅

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - Session management and window finder functions
- `cmd/orch/spawn_cmd.go` - Spawn routing logic
- `pkg/spawn/config.go` - Workspace name generation

**Commands Run:**
```bash
# List current tmux sessions
tmux list-sessions -F "#{session_name}"

# List windows in orchestrator session
tmux list-windows -t orchestrator -F "#{window_index}:#{window_name}"

# Run tests
go test ./pkg/tmux/... -v -run "Session"
```

---

## Investigation History

**2026-01-06 18:15:** Investigation started
- Initial question: How to distinguish orch vs meta-orch in tmux?
- Context: Confusion when viewing orchestrator session

**2026-01-06 18:20:** Root cause identified
- Both orchestrator types using same session
- Workspace naming provides distinction but not immediately visible

**2026-01-06 18:30:** Implementation completed
- Added MetaOrchestratorSessionName constant
- Added EnsureMetaOrchestratorSession function
- Updated spawn routing and window finders
- Tests pass

**2026-01-06 18:35:** Investigation completed
- Status: Complete
- Key outcome: Meta-orchestrators now spawn in separate "meta-orchestrator" session
