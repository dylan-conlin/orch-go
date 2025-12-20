**TLDR:** Question: Add tail command for capturing tmux window output to debug stuck agents. Answer: Implemented CaptureLines, ListWindows, and FindWindowByBeadsID functions in pkg/tmux with full test coverage. High confidence (95%) - all tests passing.

---

# Investigation: Add Tail Command for Agent Debugging

**Question:** How to capture recent output from an agent's tmux window for debugging stuck agents?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Core tmux capture functionality implemented

**Evidence:** Added three functions to pkg/tmux/tmux.go:
- `CaptureLines(windowTarget string, lines int)` - Captures last N lines from pane
- `ListWindows(sessionName string)` - Lists all windows with index, ID, name, target
- `FindWindowByBeadsID(sessionName, beadsID string)` - Finds window by [beads-id] pattern

**Source:** `pkg/tmux/tmux.go:367-447`

**Significance:** Provides the building blocks for the tail command to:
1. Find the correct tmux window by beads ID
2. Capture the recent output from that window
3. Display it to the orchestrator for debugging

### Finding 2: Test coverage complete

**Evidence:** Five tests added and passing:
- `TestCaptureLines` - Verifies line capture with content
- `TestCaptureLinesDefault` - Verifies capture with 0 lines (all visible)
- `TestListWindows` - Verifies window listing
- `TestFindWindowByBeadsID` - Verifies finding window by beads ID
- `TestFindWindowByBeadsIDNotFound` - Verifies nil return for missing window

**Source:** `pkg/tmux/tmux_test.go:301-490`

**Significance:** TDD approach ensures functions work correctly. All tests pass.

---

## Synthesis

**Key Insights:**

1. **Window naming convention leveraged** - The `[beads-id]` pattern in window names (e.g., `🔬 og-inv-test [proj-123]`) enables reliable window lookup.

2. **Flexible line capture** - Using tmux's `-S` flag allows capturing last N lines; 0 captures all visible content.

3. **Clean API** - Functions return structured `WindowInfo` type with all needed fields for downstream use.

**Answer to Investigation Question:**

The tail functionality is implemented via three functions in pkg/tmux:
- Find window: `FindWindowByBeadsID(sessionName, beadsID)` returns `*WindowInfo`
- Capture output: `CaptureLines(windowTarget, lines)` returns `[]string`
- List all: `ListWindows(sessionName)` for discovery

CLI wiring will be done separately.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All core functionality implemented with full test coverage. Tests verify real tmux operations.

**What's certain:**

- ✅ CaptureLines correctly captures last N lines from pane
- ✅ FindWindowByBeadsID correctly finds windows by beads ID pattern
- ✅ ListWindows correctly lists all windows in a session
- ✅ All tests passing

**What's uncertain:**

- ⚠️ CLI wiring not yet complete (out of scope per orchestrator)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use existing functions for CLI tail command**

The pkg/tmux package now provides:
```go
// Get project workers session
sessionName := tmux.GetWorkersSessionName(projectName)

// Find window by beads ID
window, err := tmux.FindWindowByBeadsID(sessionName, beadsID)

// Capture last N lines
lines, err := tmux.CaptureLines(window.Target, 50)
```

**Implementation sequence:**
1. Get project name from current directory
2. Build workers session name
3. Find window by beads ID
4. Capture lines and print

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - Added CaptureLines, ListWindows, FindWindowByBeadsID
- `pkg/tmux/tmux_test.go` - Added tests for new functions

**Commands Run:**
```bash
# Verify tests pass
go test ./pkg/tmux/... -v -run "TestCaptureLines|TestListWindows|TestFindWindowByBeadsID"
```

---

## Investigation History

**2025-12-20 18:21:** Investigation started
- Initial question: How to capture tmux window output for debugging?
- Context: Need to debug stuck agents by viewing their terminal output

**2025-12-20 18:29:** Implementation complete
- Added CaptureLines, ListWindows, FindWindowByBeadsID to pkg/tmux
- Added 5 tests, all passing
- CLI wiring deferred per orchestrator instruction

**2025-12-20 18:30:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Core tail functionality implemented with tests
