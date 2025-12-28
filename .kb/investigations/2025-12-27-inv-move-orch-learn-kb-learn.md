<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully moved `orch learn` to `kb learn` as Phase 1 consolidation - kb-cli now owns the learning loop.

**Evidence:** Both kb-cli and orch-go build and pass tests; `kb learn --help` shows all subcommands; `orch learn` delegates to `kb learn`.

**Knowledge:** The learning loop conceptually belongs in kb since it's about knowledge gaps, suggests kn/kb artifacts, and aligns with `kb reflect`.

**Next:** Close issue - implementation complete. Future: When Phase 1 completes, update orchestrator skill documentation to use `kb learn` instead of `orch learn`.

---

# Investigation: Move Orch Learn to Kb Learn

**Question:** How should we migrate `orch learn` to `kb learn` as part of Phase 1 consolidation?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Learning loop code is self-contained in pkg/spawn/learning.go

**Evidence:** The learning.go file (923 lines) contains all gap tracking types (GapEvent, GapTracker, LearningSuggestion, etc.) and functions (LoadTracker, Save, FindRecurringGaps, etc.) with no external dependencies on orch-go specific packages.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go`

**Significance:** Clean extraction to kb-cli is feasible without complex refactoring.

---

### Finding 2: Gap types are defined in gap.go but only used by learning.go

**Evidence:** `GapType` (no_context, sparse_context, no_constraints, no_decisions) and `GapSeverity` (info, warning, critical) constants are only referenced by learning.go for suggestion generation.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/gap.go`

**Significance:** These types can be duplicated in kb-cli without causing import cycles; orch-go continues to use its copy for gap detection during spawn.

---

### Finding 3: kb-cli follows same Cobra patterns as orch-go

**Evidence:** kb-cli uses `spf13/cobra` with init() functions registering commands to rootCmd. reflect.go shows pattern of creating subcommands with RunE handlers.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go`, `cmd/kb/main.go`

**Significance:** Direct port of command structure is straightforward.

---

## Implementation

**What was implemented:**

1. **kb-cli: cmd/kb/learn.go** - Full implementation of `kb learn` with all subcommands:
   - `kb learn` / `kb learn suggest` - Show recurring gap suggestions
   - `kb learn patterns` - Analyze gap patterns by topic
   - `kb learn skills` - Show gap rates by skill  
   - `kb learn projects` - Show gap rates by source project
   - `kb learn effects` - Show improvement effectiveness
   - `kb learn act [index]` - Run suggested command
   - `kb learn resolve [index] [type]` - Mark gap as resolved
   - `kb learn clear` - Clear gap history
   - `kb learn external-summary` - Show external project gaps

2. **kb-cli: cmd/kb/learn_test.go** - Comprehensive test coverage

3. **orch-go: pkg/spawn/learning.go** - Updated TrackerPath() to use `~/.kb/gap-tracker.json`

4. **orch-go: cmd/orch/learn.go** - Replaced with wrapper that delegates to `kb learn`

---

## Structured Uncertainty

**What's tested:**

- ✅ kb-cli builds successfully (verified: `go build ./cmd/kb/`)
- ✅ kb-cli tests pass (verified: `go test ./cmd/kb/...` - 0.077s)
- ✅ orch-go builds successfully (verified: `go build ./cmd/orch/`)
- ✅ orch-go tests pass (verified: `go test ./pkg/spawn/...` - 0.034s)
- ✅ `kb learn --help` shows all subcommands

**What's untested:**

- ⚠️ End-to-end flow: orch spawn → gap event → kb learn display (requires running spawn with sparse context)
- ⚠️ Migration of existing ~/.orch/gap-tracker.json data (users would need to copy file)

**What would change this:**

- Finding would be wrong if gap events written by orch spawn are not readable by kb learn

---

## References

**Files Created:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/learn.go` - Main implementation
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/learn_test.go` - Test coverage

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/learning.go` - TrackerPath change
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/learn.go` - Delegation wrapper

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: How to migrate orch learn to kb learn for Phase 1 consolidation
- Context: Learning loop conceptually belongs in kb (knowledge management), not orch (agent coordination)

**2025-12-27:** Implementation complete
- Created kb learn command with all subcommands
- Updated gap-tracker.json location to ~/.kb/
- orch learn now delegates to kb learn
- Both projects build and tests pass
