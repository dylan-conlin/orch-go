<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Spawn tracking in orchestrator sessions was already implemented in pkg/sessions/orchestrator.go - only needed integration with spawn commands.

**Evidence:** Found OrchestratorStore.RecordSpawn() existing, added recordSpawnInSession() calls to all three spawn modes (inline, headless, tmux), all tests pass.

**Knowledge:** The orchestrator session model (orch-go-amfa epic) was implemented in parallel - session.go and orchestrator.go already provide full session lifecycle management.

**Next:** Close - implementation complete. Session tracking works when `orch session start` is used, silently skips when no active session.

---

# Investigation: Track Spawns in Session State

**Question:** How do we record spawn information in the orchestrator session state when agents are spawned?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** og-feat-track-spawns-session-29dec
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

---

## Findings

### Finding 1: Orchestrator Session Infrastructure Already Exists

**Evidence:** The `pkg/sessions/orchestrator.go` file contains:
- `OrchestratorSession` struct with `Spawns []SpawnRecord` field
- `SpawnRecord` struct with BeadsID, Skill, SpawnedAt, SessionID fields
- `OrchestratorStore.RecordSpawn(beadsID, skill, sessionID string)` method

**Source:** `pkg/sessions/orchestrator.go:16-207`, `pkg/sessions/orchestrator_test.go:142-184`

**Significance:** No need to design new data structures - the infrastructure was implemented as part of the sibling task (orch-go-amfa.1).

---

### Finding 2: Session Commands Already Implemented

**Evidence:** `cmd/orch/session.go` implements:
- `orch session start [goal]` - creates session with ID, sets focus
- `orch session status` - shows session including spawned agents
- `orch session end` - ends session, shows summary

**Source:** `cmd/orch/session.go:1-274`

**Significance:** Session lifecycle is complete. Only spawn tracking integration was missing.

---

### Finding 3: Three Spawn Modes Need Integration

**Evidence:** Three functions in `cmd/orch/main.go` handle spawning:
- `runSpawnInline()` - blocking spawn (line 1566)
- `runSpawnHeadless()` - HTTP API spawn (line 1655)
- `runSpawnTmux()` - tmux window spawn (line 1910)

Each already logs events to events.jsonl and writes session IDs to workspace.

**Source:** `cmd/orch/main.go:1566, 1655, 1910`

**Significance:** Adding RecordSpawn calls to each mode ensures consistent tracking regardless of spawn method.

---

## Synthesis

**Key Insights:**

1. **Parallel Implementation** - The orchestrator session epic (orch-go-amfa) split into multiple tasks, with session.go/orchestrator.go already implementing the session model. This task only needed to wire spawn tracking into the spawn commands.

2. **Best-Effort Tracking** - RecordSpawn silently succeeds when no session is active. This ensures spawns work independently of session state (acceptance criteria #2: "Spawning works normally without active session").

3. **formatDuration Collision** - Both session.go and wait.go defined formatDuration. The session.go version was removed (comment added) since wait.go's version was already used throughout the codebase.

**Answer to Investigation Question:**

Spawn information is recorded in orchestrator session state by calling `recordSpawnInSession(beadsID, skillName, sessionID)` in each spawn mode function after the agent is successfully spawned. This helper function:
1. Creates an OrchestratorStore instance
2. Checks if a session is active
3. Calls RecordSpawn to append the spawn info to session.json
4. Silently ignores errors (best-effort, non-blocking)

---

## Structured Uncertainty

**What's tested:**

- ✅ OrchestratorStore.RecordSpawn() works with and without active session (verified: TestOrchestratorStore_RecordSpawn)
- ✅ Code compiles with all three spawn modes integrated (verified: go build ./cmd/orch/...)
- ✅ Existing tests pass (verified: go test ./pkg/sessions/... and session-related cmd/orch tests)

**What's untested:**

- ⚠️ End-to-end spawn tracking (would require starting a real session and spawning an agent)
- ⚠️ Performance impact of adding session store read/write on each spawn (likely negligible)

**What would change this:**

- If session.json path changes from ~/.orch/session.json, DefaultOrchestratorPath() would need updating
- If spawns need additional metadata, SpawnRecord struct would need extension

---

## Implementation Summary

**Changes Made:**

1. Added `github.com/dylan-conlin/orch-go/pkg/sessions` import to main.go
2. Added `recordSpawnInSession()` helper function (lines 1547-1562)
3. Added calls to `recordSpawnInSession()` in:
   - `runSpawnInline()` (after event logging)
   - `runSpawnHeadless()` (after event logging)
   - `runSpawnTmux()` (after window focus)

**Files Modified:**
- `cmd/orch/main.go` - Added import, helper function, and three integration calls
- `cmd/orch/wait.go` - Minor: renamed internal function to avoid collision (reverted - used session.go's approach instead)
- `cmd/orch/session.go` - Already updated by parallel task (comment about formatDuration being in wait.go)

---

## References

**Files Examined:**
- `pkg/sessions/orchestrator.go` - Session state management
- `pkg/sessions/orchestrator_test.go` - Existing tests for RecordSpawn
- `cmd/orch/main.go` - Spawn command implementation
- `cmd/orch/session.go` - Session commands (start/status/end)
- `.kb/investigations/2025-12-29-inv-unified-session-model-design.md` - Design investigation

**Commands Run:**
```bash
# Verify build
go build ./cmd/orch/...

# Run orchestrator session tests
go test ./pkg/sessions/... -v -run Orchestrator

# Run session-related tests
go test ./cmd/orch/... -v -run Session
```

**Related Artifacts:**
- **Epic:** orch-go-amfa - Unified Orchestrator Session Model
- **Sibling Task:** orch-go-amfa.1 - Implement orch session start command (implemented session infrastructure)
- **Investigation:** .kb/investigations/2025-12-29-inv-unified-session-model-design.md - Design rationale

---

## Investigation History

**2025-12-29 10:46:** Investigation started
- Initial question: How to track spawns in orchestrator session state?
- Context: Part of orch-go-amfa epic for unified session model

**2025-12-29 10:50:** Found existing infrastructure
- Discovered pkg/sessions/orchestrator.go with full session model
- Discovered cmd/orch/session.go with session commands

**2025-12-29 11:05:** Implementation complete
- Added recordSpawnInSession helper
- Integrated into all three spawn modes
- Resolved formatDuration collision
- All tests pass

**2025-12-29 11:10:** Investigation completed
- Status: Complete
- Key outcome: Spawn tracking integrated into orchestrator sessions via recordSpawnInSession() helper called in all spawn modes
