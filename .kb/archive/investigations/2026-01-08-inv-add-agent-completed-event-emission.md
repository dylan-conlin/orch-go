<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Added `orch emit` command and beads on_close hook to emit agent.completed events when issues are closed via `bd close`, closing the tracking gap.

**Evidence:** Full flow tested - created test issue, closed via `bd close`, verified event appeared in `~/.orch/events.jsonl` with correct beads_id and reason.

**Knowledge:** Beads hooks receive `<issue_id> <event_type>` as args and issue JSON on stdin; hooks must exit 0 to not block the close operation.

**Next:** None - implementation complete, tests pass, feature documented.

**Promote to Decision:** recommend-no (feature implementation, not architectural)

---

# Investigation: Add Agent Completed Event Emission

**Question:** How to emit agent.completed events when issues are closed directly via `bd close` bypassing `orch complete`?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent (feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Beads has a hook system

**Evidence:** Beads CLI runs hooks from `.beads/hooks/` directory. The `on_close` hook is executed when issues are closed.

**Source:** 
- `/Users/dylanconlin/Documents/personal/beads/internal/hooks/hooks.go:17-25` - Hook event constants
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/close.go:124-127` - Hook invocation on close

**Significance:** We can leverage the existing hook system to emit events without modifying beads itself.

---

### Finding 2: Hook interface uses args and stdin

**Evidence:** Hooks receive `<issue_id> <event_type>` as command line args and issue JSON on stdin.

**Source:** `/Users/dylanconlin/Documents/personal/beads/internal/hooks/hooks_unix.go:31-32`

**Significance:** The hook script can parse the issue ID from args and close_reason from the JSON stdin.

---

### Finding 3: events.jsonl structure is well-defined

**Evidence:** `agent.completed` events are already emitted by `orch complete` with beads_id in the data field.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:616-623`

**Significance:** The new emit command should match this structure for consistency with stats aggregation.

---

## Synthesis

**Key Insights:**

1. **Hook-based integration** - Using beads hooks avoids modifying beads itself; the hook calls back into orch-go via `orch emit`.

2. **Event consistency** - The emit command produces events in the same format as `orch complete`, ensuring stats aggregation works uniformly.

3. **Non-blocking design** - The hook script exits 0 even if `orch emit` fails, so it never blocks the `bd close` operation.

**Answer to Investigation Question:**

The solution is a two-part implementation:
1. `orch emit agent.completed` command that writes events to `~/.orch/events.jsonl`
2. `.beads/hooks/on_close` script that calls `orch emit` with the issue ID and close reason

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch emit` command emits events correctly (verified: unit tests pass, manual test with `--json` flag)
- ✅ Hook executes on `bd close` (verified: created test issue orch-go-2d7qs, closed it, saw event in events.jsonl)
- ✅ Close reason is captured from issue JSON (verified: event data["reason"] = "Testing hook")

**What's untested:**

- ⚠️ Hook behavior in daemon mode (beads daemon runs hooks async)
- ⚠️ Cross-project hook deployment (each project needs its own hook)
- ⚠️ Performance impact of hook execution (should be negligible)

**What would change this:**

- Finding would be wrong if beads changed hook invocation interface
- Solution assumes orch binary is in PATH when hook runs

---

## Implementation Recommendations

**Purpose:** Implemented, not applicable.

---

## References

**Files Created:**
- `cmd/orch/emit_cmd.go` - Main emit command implementation
- `cmd/orch/emit_test.go` - Unit tests for emit command
- `.beads/hooks/on_close` - Beads hook script

**Files Modified:**
- `CLAUDE.md` - Added Event Tracking documentation section

**Commands Run:**
```bash
# Build and install
make install

# Test emit command
orch emit agent.completed --beads-id test-emit-123 --reason "Manual test" --json

# Test full flow
bd create "Test hook issue for emit" --type task  # Created orch-go-2d7qs
bd close orch-go-2d7qs --reason "Testing hook" --force
tail -1 ~/.orch/events.jsonl | jq .  # Verified event present
```

---

## Investigation History

**2026-01-08 14:05:** Investigation started
- Initial question: How to emit events when bd close bypasses orch complete?
- Context: Stats aggregation misses completions that go through bd close directly

**2026-01-08 14:10:** Found beads hook system
- Beads has `.beads/hooks/on_close` that runs when issues close
- Hook receives issue ID as arg and issue JSON on stdin

**2026-01-08 14:15:** Implemented orch emit command
- Added `cmd/orch/emit_cmd.go` with validation and JSON support
- Added unit tests covering happy path and error cases

**2026-01-08 14:16:** Created beads hook
- Added `.beads/hooks/on_close` script
- Parses close_reason from JSON, calls orch emit

**2026-01-08 14:17:** Full flow validated
- Created test issue, closed with bd close, verified event in events.jsonl
- All tests pass

**2026-01-08 14:18:** Investigation completed
- Status: Complete
- Key outcome: Added orch emit command and beads hook to close tracking gap
