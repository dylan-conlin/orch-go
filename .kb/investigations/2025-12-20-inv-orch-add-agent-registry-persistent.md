**TLDR:** Question: How to port Python's AgentRegistry to Go for persistent agent tracking? Answer: Implemented pkg/registry with Agent struct, file-based persistence (~/.orch/agent-registry.json), file locking for concurrency, and merge logic for conflict resolution. Integrated with spawn command. High confidence (95%) - all 22 tests pass including concurrent access tests.

---

# Investigation: Agent Registry for Persistent Tracking

**Question:** How to port Python's AgentRegistry (~/.orch/registry.json) to Go for orch-go?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Claude agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Python registry has well-defined state machine

**Evidence:** Python AgentRegistry in orch-cli uses:
- Four states: active, completed, abandoned, deleted
- Tombstone pattern for deletions (agent stays in registry with status=deleted)
- Window ID reuse abandons old agent
- File locking with fcntl for concurrent access
- Merge logic using updated_at timestamps

**Source:** `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/registry.py` (444 lines)

**Significance:** Clear spec to port. The state machine is minimal and focused on tmux operations.

---

### Finding 2: Beads is source of truth for metadata

**Evidence:** Python registry deliberately stores minimal data:
- Core fields: id, window_id, beads_id, status
- Metadata (task, project_dir) stored in beads, not registry
- Registry only needs tmux window mapping for operations

**Source:** Python registry register() method (lines 283-375) and model (171 lines)

**Significance:** Go implementation can follow same minimal pattern - don't duplicate beads data.

---

### Finding 3: Concurrent access requires careful handling

**Evidence:** Python uses:
- fcntl.LOCK_SH for reads, fcntl.LOCK_EX for writes
- Merge on save: re-read disk content, compare updated_at, newer wins
- skip_merge option for delete operations to prevent resurrection

**Source:** Python save() method with merge logic, test_concurrent_access tests

**Significance:** Go implementation must handle same scenarios - implemented using syscall.Flock with timeout.

---

## Synthesis

**Key Insights:**

1. **Minimal state machine** - Only 4 states with clear transitions. Implementation focused on tmux window tracking, not full agent metadata.

2. **File locking + merge = concurrent safety** - Acquire exclusive lock before write, re-read current state, merge using timestamps, write result.

3. **Timestamps need sub-second precision** - RFC3339 only has second precision, insufficient for concurrent tests. Using RFC3339Nano.

**Answer to Investigation Question:**

Successfully ported Python's AgentRegistry to Go in pkg/registry with:
- Agent struct with core fields (ID, BeadsID, WindowID, Status)
- AgentState type for the 4 states
- File-based persistence to ~/.orch/agent-registry.json
- File locking via syscall.Flock with timeout
- Merge logic comparing UpdatedAt timestamps
- 22 comprehensive tests including concurrent access

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All 22 tests pass including:
- State transition tests (register, abandon, delete, reconcile)
- Find preference tests (agent ID over beads_id)
- Concurrent access tests (20 goroutines registering simultaneously)
- Merge logic tests (stale writes don't overwrite newer data)

**What's certain:**

- ✅ Core registry operations work (register, find, list, abandon, remove)
- ✅ File locking prevents corruption under concurrent access
- ✅ Merge logic preserves newer data when conflicts occur
- ✅ Integration with spawn command registers agents

**What's uncertain:**

- ⚠️ Not tested in production with real tmux sessions yet
- ⚠️ Cross-project beads lookup not implemented (needs beads_db_path)

**What would increase confidence to 100%:**

- End-to-end testing with actual agent spawns
- Integration testing with orch status and orch complete commands

---

## Implementation Summary

**Implemented:**

1. `pkg/registry/registry.go` - Core registry implementation (450 lines)
   - Agent struct with all fields from Python
   - AgentState type (active/completed/abandoned/deleted)
   - New(), Register(), Find(), ListAgents(), ListActive()
   - Abandon(), Remove(), Reconcile()
   - Save() with file locking and merge logic

2. `pkg/registry/registry_test.go` - Comprehensive tests (620 lines)
   - 22 test cases covering all functionality
   - Concurrent access tests
   - Merge logic tests
   - Tombstone pattern tests

3. Integration with `cmd/orch/main.go`
   - runSpawnInTmux registers agent with window_id
   - runSpawnInline registers agent without window_id

**Files Changed:**
- `pkg/registry/registry.go` (new)
- `pkg/registry/registry_test.go` (new)
- `cmd/orch/main.go` (modified - added registry import and registration)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/registry.py` - Python implementation
- `/Users/dylanconlin/Documents/personal/orch-cli/src/orch/models/agent_model.py` - State machine model
- `/Users/dylanconlin/Documents/personal/orch-cli/tests/test_registry_model.py` - Test patterns

**Commands Run:**
```bash
# Build and test
go build ./pkg/registry/...
go test ./pkg/registry/... -v

# Full test suite
go test ./...

# Build binary
go build -o orch-go .
```

---

## Investigation History

**2025-12-20 10:00:** Investigation started
- Initial question: Port Python AgentRegistry to Go
- Context: orch-go is stateless, needs registry for clean/abandon/wait/status --global

**2025-12-20 10:30:** Core implementation complete
- Registry with file locking and merge logic
- 22 tests passing

**2025-12-20 10:45:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: pkg/registry package ready for use, integrated with spawn command
