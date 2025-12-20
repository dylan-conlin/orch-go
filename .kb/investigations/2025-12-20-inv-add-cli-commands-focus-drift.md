**TLDR:** Question: How to wire up focus, drift, and next CLI commands in orch-go? Answer: Created cmd/orch/focus.go with three cobra commands (focus, drift, next) that call the existing pkg/focus/ Store methods, following existing patterns from daemon.go and wait.go. High confidence (95%) - all tests pass and commands verified working.

---

# Investigation: Add CLI Commands for Focus, Drift, and Next

**Question:** How to wire up focus, drift, and next CLI commands in orch-go?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: pkg/focus/ has full implementation

**Evidence:** The focus package provides Store with Set(), Get(), Clear(), CheckDrift(), and SuggestNext() methods. All business logic is already implemented.

**Source:** pkg/focus/focus.go:1-266

**Significance:** CLI commands just need to wire up calls to existing Store methods, no new logic needed.

---

### Finding 2: Existing command patterns in daemon.go and wait.go

**Evidence:** Other commands follow a consistent pattern:

- cobra.Command with Use, Short, Long, RunE
- Flags defined in init() blocks
- run\* functions that implement the command logic
- Events logging for significant actions

**Source:** cmd/orch/daemon.go, cmd/orch/wait.go

**Significance:** New commands should follow the same pattern for consistency.

---

### Finding 3: Registry provides active issues list

**Evidence:** registry.New("").ListActive() returns agents with BeadsID, which is needed for drift detection and next suggestions.

**Source:** pkg/registry/ - ListActive() method

**Significance:** The drift and next commands need to get active issues from registry to pass to CheckDrift() and SuggestNext().

---

## Synthesis

**Key Insights:**

1. **Pure wiring task** - All business logic exists in pkg/focus/, CLI commands just need to expose it.

2. **Consistent patterns** - Following existing daemon.go and wait.go patterns ensures maintainability.

3. **Registry integration** - Getting active issues from registry enables drift detection and next suggestions.

**Answer to Investigation Question:**

Created cmd/orch/focus.go with:

- `focus` command: Set/get/clear the north star priority (with --issue and --json flags)
- `drift` command: Check if active work aligns with focus (with --json flag)
- `next` command: Suggest next action based on focus and state (with --json flag)

All commands registered in cmd/orch/main.go init() function.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass, commands verified working via manual testing.

**What's certain:**

- All commands compile and build successfully
- All existing tests pass
- Commands produce expected output for focus set/get/clear, drift, and next

**What's uncertain:**

- Integration with actual beads issues (bd ready) - tested command exists but output parsing may vary

---

## References

**Files Created:**

- cmd/orch/focus.go - CLI commands for focus, drift, next

**Files Modified:**

- cmd/orch/main.go - Added command registrations

**Commands Run:**

```bash
go build ./cmd/orch/...
go test ./...
go run ./cmd/orch focus --help
go run ./cmd/orch focus "Test goal"
go run ./cmd/orch drift
go run ./cmd/orch next
go run ./cmd/orch focus clear
```

---

## Investigation History

**2025-12-20 14:40:** Investigation started

- Initial question: Wire up focus, drift, and next CLI commands
- Context: pkg/focus/ has full implementation, needed CLI exposure

**2025-12-20 14:43:** Implementation complete

- Created cmd/orch/focus.go with all three commands
- Registered in main.go
- All tests passing

**2025-12-20 14:43:** Investigation completed

- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Three new CLI commands (focus, drift, next) wired up and working
