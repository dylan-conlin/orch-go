<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch session` command does not exist - prior agent claimed completion but code was never committed.

**Evidence:** No session.go in cmd/orch/, `./build/orch session --help` returns "Command not found", session.json only contains `{"session": null}`.

**Knowledge:** Session state tracking currently exists in fragmented form (agent-registry.json, current-session.json, focus.json) but no unified session command exposes it.

**Next:** Implement `orch session start/status/end` commands with spawn reconciliation via GetLiveness().

---

# Investigation: Orch Session Status Reconcile Spawn

**Question:** Why does `orch session status` show stale spawn states, and how should it reconcile against actual agent status?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** og-debug-orch-session-status-02jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Session Command Does Not Exist

**Evidence:** 
- `./build/orch session --help` returns "Command not found"
- `ls cmd/orch/` shows no session.go file
- `grep -r "sessionCmd" cmd/orch/*.go` returns no results (except unrelated session struct in tokens.go)

**Source:** 
- cmd/orch/*.go files
- CLI output from `./build/orch --help`

**Significance:** The prior agent (based on beads comment "Phase: Complete - Implemented orch session command...") claimed to implement this but the code was never committed. This is why the issue was reopened.

---

### Finding 2: Session State Files Are Fragmented

**Evidence:** Multiple session-related files exist with different purposes:
- `~/.orch/session.json` - Contains `{"session": null}` - appears unused
- `~/.orch/current-session.json` - Contains `{"tmux_session": "workers", "started_at": "2025-12-09T09:49:46"}` - stale
- `~/.orch/focus.json` - Contains current focus goal
- `~/.orch/agent-registry.json` - Contains spawn history with status tracking

**Source:** File system inspection of ~/.orch/*.json

**Significance:** No unified session model exists. The "Focus-Based Session Model" described in the orchestrator skill requires implementing session state management that ties these concepts together.

---

### Finding 3: State Reconciliation Package Already Exists

**Evidence:** `pkg/state/reconcile.go` provides:
- `GetLiveness(beadsID, serverURL, projectDir)` - Returns LivenessResult with TmuxLive, OpencodeLive, BeadsOpen, etc.
- `IsLive()` / `IsPhantom()` methods for determining agent status
- Cross-references tmux windows, OpenCode sessions, beads issues, and workspaces

**Source:** pkg/state/reconcile.go:64-101

**Significance:** The infrastructure for state reconciliation exists. Session status command should use GetLiveness() to derive spawn states at query time rather than trusting stored state.

---

## Synthesis

**Key Insights:**

1. **Missing Implementation** - The session command infrastructure was never built. The prior agent may have planned it but didn't commit the code.

2. **Derive State, Don't Duplicate** - The issue description's key insight is correct: session status should derive spawn state from actual sources (OpenCode API, tmux, beads) using GetLiveness() rather than maintaining a separate state file that can get stale.

3. **Session Model is Focus + Spawns** - Based on the orchestrator skill's "Focus-Based Session Model", a session should track:
   - Goal (from focus)
   - Start time
   - Spawns during session (can be derived from spawn history)
   - Active agents (derived via GetLiveness at query time)

**Answer to Investigation Question:**

The `orch session status` command doesn't exist yet - that's the root cause. When implemented, it should:
1. Show session goal and duration
2. List spawns made during the session
3. Categorize spawns as Active/Completed/Phantom by calling GetLiveness() for each

---

## Implementation Recommendations

### Recommended Approach: pkg/session with Query-Time Reconciliation

Create a new pkg/session package that:
1. Manages session state (goal, start time, spawn history)
2. Uses GetLiveness() at query time to determine spawn status (not stored status)
3. Exposes session start/status/end commands

**Why this approach:**
- Follows "derive state, don't duplicate" principle from issue description
- Leverages existing GetLiveness() infrastructure in pkg/state
- Matches the Focus-Based Session Model from orchestrator skill

**Implementation sequence:**
1. Create pkg/session with Session struct and Store
2. Add session start command that records goal + start time
3. Add session status command that queries GetLiveness() for each spawn
4. Add session end command that clears session state
5. Wire commands into cmd/orch/main.go

### Data Model

```go
type Session struct {
    Goal      string          `json:"goal"`
    StartedAt time.Time       `json:"started_at"`
    Spawns    []SpawnRecord   `json:"spawns"` // Recorded at spawn time
}

type SpawnRecord struct {
    BeadsID     string    `json:"beads_id"`
    Skill       string    `json:"skill"`
    SpawnedAt   time.Time `json:"spawned_at"`
    // Status derived at query time via GetLiveness(), not stored
}

type SpawnStatus struct {
    SpawnRecord
    State string // "active", "completed", "phantom" - derived via GetLiveness()
}
```

---

## References

**Files Examined:**
- pkg/state/reconcile.go - Existing liveness check infrastructure
- cmd/orch/focus.go - Focus command for reference pattern
- cmd/orch/handoff.go - Session handoff for reference
- ~/.orch/session.json - Current (empty) session state file
- ~/.orch/agent-registry.json - Spawn history tracking

**Commands Run:**
```bash
# Check if session command exists
./build/orch session --help  # Returns "Command not found"

# Find session-related files
ls ~/.orch/*.json

# Check session.json contents
cat ~/.orch/session.json  # {"session": null}
```

**Related Artifacts:**
- **Skill:** ~/.claude/skills/meta/orchestrator/SKILL.md - Focus-Based Session Model section
- **Package:** pkg/state/reconcile.go - GetLiveness() function for state reconciliation

---

## Investigation History

**2026-01-02 22:30:** Investigation started
- Initial question: Why does orch session status show stale spawn states?
- Context: Beads issue orch-go-gba4 describes session.json tracking spawns but not updating status

**2026-01-02 22:45:** Root cause identified
- The orch session command does not exist
- Prior agent claimed completion but code was never committed
- Session state files exist but are fragmented and stale

**2026-01-02 22:50:** Implementation plan defined
- Will create pkg/session package
- Will use GetLiveness() for query-time state derivation
- Will implement session start/status/end commands
