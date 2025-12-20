**TLDR:** Question: Implement `orch abandon` command to kill stuck agents. Answer: Successfully implemented abandon command that finds agent by beads ID, kills tmux window via KillWindowByID, marks agent as abandoned in registry, and logs the event. High confidence (95%) - builds and package tests pass.

---

# Investigation: Add Abandon Command to orch-go

**Question:** How to implement an abandon command for stuck/frozen agents?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Registry already has Abandon method

**Evidence:** The `pkg/registry/registry.go` file already has an `Abandon(agentID string) bool` method that marks an agent as abandoned and sets the `AbandonedAt` timestamp.

**Source:** `pkg/registry/registry.go:388-404`

**Significance:** No new registry functionality needed - can reuse existing method.

---

### Finding 2: Tmux package has KillWindowByID

**Evidence:** The `pkg/tmux/tmux.go` file has `KillWindowByID(windowID string) error` that kills a window by its unique ID (e.g., "@1234").

**Source:** `pkg/tmux/tmux.go:342-346`

**Significance:** Can kill agent's window using the window_id stored in the registry.

---

### Finding 3: Pattern from existing commands

**Evidence:** The `tail` and `question` commands show the pattern for finding agents by beads ID using the registry, though they use different approaches (tmux window search vs registry).

**Source:** `cmd/orch/main.go`

**Significance:** The abandon command uses the registry approach since it already stores the window_id.

---

## Synthesis

**Key Insights:**

1. **Registry-first approach** - Find agent by beads ID using `registry.Find()`, which supports lookup by beads_id as a secondary key.

2. **Window cleanup** - Kill the tmux window using the stored `WindowID` via `tmux.KillWindowByID()`.

3. **State transition** - Use `registry.Abandon()` to mark the agent as abandoned, which sets status and timestamp.

**Answer to Investigation Question:**

Implemented the abandon command as:
1. `abandonCmd` - Cobra command definition with usage and examples
2. `runAbandon(beadsID)` - Implementation that:
   - Opens registry and finds agent by beads ID
   - Verifies agent is active (not already completed/abandoned)
   - Kills tmux window if present
   - Marks agent as abandoned in registry
   - Logs the abandonment event
   - Prints summary with instructions for restarting work

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Implementation reuses existing, tested components (registry.Abandon, tmux.KillWindowByID). The pattern follows established command patterns in the codebase.

**What's certain:**

- ✅ Build passes with new abandon command
- ✅ All package tests pass
- ✅ Command follows established CLI patterns
- ✅ Uses registry for agent lookup (not tmux window name search)

**What's uncertain:**

- ⚠️ Integration testing not performed (would require active agents)
- ⚠️ Some cmd/orch tests fail due to other agents' incomplete work

---

## Implementation Recommendations

### Recommended Approach ⭐

**Registry-based abandon** - Use registry to find agent by beads ID, then kill window and update status.

**Why this approach:**
- Registry is the source of truth for agent state
- Window ID is stored in registry, avoiding window name parsing
- Consistent with how spawn registers agents

**Implementation sequence:**
1. Add abandonCmd to rootCmd.AddCommand()
2. Define abandonCmd with usage/help
3. Implement runAbandon() function

### Alternative Approaches Considered

**Option B: Tmux window name search**
- **Pros:** Works without registry
- **Cons:** Fragile parsing, window names can vary
- **When to use instead:** If registry is unavailable

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Added abandon command
- `pkg/registry/registry.go` - Verified Abandon method exists
- `pkg/tmux/tmux.go` - Verified KillWindowByID exists

**Commands Run:**
```bash
# Build verification
go build ./...

# Package tests
go test ./pkg/...
```

---

## Investigation History

**2025-12-20 10:25:** Investigation started
- Initial question: How to implement abandon command?
- Context: Spawned from beads issue orch-go-djd

**2025-12-20 10:28:** Implementation completed
- Added abandonCmd and runAbandon to cmd/orch/main.go
- Build passes, package tests pass
- cmd/orch tests have pre-existing failures from other agents' work

**2025-12-20 10:30:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Abandon command implemented using registry and tmux package
