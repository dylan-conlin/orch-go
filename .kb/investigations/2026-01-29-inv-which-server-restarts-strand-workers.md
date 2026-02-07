<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Which Server Restarts Strand Workers

**Question:** Do stranded workers correlate to OpenCode server (:4096) restarts or orch serve (:3348) restarts, and where should restart-aware auto-resume detection hook?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Agent og-feat-investigate-server-restarts-29jan-02c0
**Phase:** Complete
**Next Step:** None - findings documented, auto-resume mechanism already implemented
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenCode Server (:4096) Restarts Are Detected and Logged

**Evidence:** Daemon log shows multiple server restart events:
```
[19:48:16] [DEBUG] ServerRecoveryState.UpdateServerHealth: server restart detected (was down, now up)
[19:49:25] [DEBUG] ShouldRunServerRecovery: returning true - server restart detected
[19:59:54] [DEBUG] ServerRecoveryState.UpdateServerHealth: server restart detected (was down, now up)
[20:00:57] [DEBUG] ShouldRunServerRecovery: returning true - server restart detected
[20:04:50] [DEBUG] ServerRecoveryState.UpdateServerHealth: server restart detected (was down, now up)
[20:05:52] [DEBUG] ServerRecoveryState.UpdateServerHealth: server restart detected (was down, now up)
```

Current OpenCode server process started: Thu Jan 29 20:05:12 2026 (PID 66819)

**Source:** `~/.orch/daemon.log`, `ps -p 66819 -o lstart=`

**Significance:** OpenCode server (:4096) has been restarting multiple times. The daemon's `ServerRecoveryState` is detecting these restarts (down → up transitions) and triggering recovery attempts. This is the server that agents connect to via SSE streams, so restarts here would kill agent sessions.

---

### Finding 2: No Evidence of orch serve (:3348) Connection Issues

**Evidence:** Grep of daemon log for orch serve (port 3348) connection failures returned no results:
```bash
grep -E "connection refused.*3348|Failed to connect.*3348" ~/.orch/daemon.log
# No output - no connection issues found
```

Current orch serve process started: Thu Jan 29 20:34:24 2026 (PID 78577)

**Source:** `~/.orch/daemon.log`, `ps -p 78577 -o lstart=`, daemon log grep

**Significance:** orch serve (:3348) shows NO connection failures in the daemon log. This server handles dashboard/API requests but agents do NOT connect to it - they connect to OpenCode (:4096). orch serve restarts would not strand agent sessions.

---

### Finding 3: Server Restart Detection Already Implemented in recovery.go

**Evidence:** `pkg/daemon/recovery.go` contains complete server restart detection and recovery:

```go
// ServerRecoveryState tracks state for server recovery detection.
type ServerRecoveryState struct {
    serverWasDown     bool  // True if server was unavailable (used to detect restart)
    restartDetected   bool  // True when a restart is detected (down -> up transition)
}

// UpdateServerHealth updates the server availability state and detects restarts.
func (s *ServerRecoveryState) UpdateServerHealth(available bool) {
    if available {
        // Server is up - check if this is a restart (was down, now up)
        if s.serverWasDown {
            fmt.Printf("[DEBUG] ServerRecoveryState.UpdateServerHealth: server restart detected (was down, now up)\n")
            s.restartDetected = true
            s.serverWasDown = false
        }
    }
}
```

**Source:** `pkg/daemon/recovery.go:214-297`

**Significance:** The auto-resume mechanism is ALREADY implemented. The daemon tracks OpenCode server health and detects restarts by monitoring down → up transitions. This answers the "where should detection hook" question - it's already hooked in `recovery.go` via `ServerRecoveryState`.

---

### Finding 4: FindOrphanedSessions Hooks OpenCode Server, Not orch serve

**Evidence:** `FindOrphanedSessions()` explicitly queries OpenCode server:
```go
func FindOrphanedSessions(serverURL string) ([]OrphanedSession, error) {
    fmt.Printf("[DEBUG] FindOrphanedSessions: starting with serverURL=%s\n", serverURL)
    // ...
    // Get current in-memory sessions from OpenCode
    client := opencode.NewClient(serverURL)
    inMemorySessions, err := client.ListSessions(projectDir)
    // ...
}
```

Daemon calls this with `http://127.0.0.1:4096` (OpenCode server):
```
[DEBUG] FindOrphanedSessions: starting with serverURL=http://127.0.0.1:4096
```

**Source:** `pkg/daemon/recovery.go:317-469`, `~/.orch/daemon.log`

**Significance:** The orphaned session detection hooks into OpenCode server (:4096), NOT orch serve (:3348). This is correct because agents connect to OpenCode via `opencode run --attach http://127.0.0.1:4096`. When OpenCode restarts, in-memory sessions are lost and `FindOrphanedSessions` detects the orphans.

---

### Finding 5: orch complete Auto-Rebuild Only Restarts orch serve

**Evidence:** From prior investigation (Jan 26), `orch complete` has auto-rebuild logic:
```go
// Restart orch serve if orch-go was rebuilt
if rebuiltOrchGo {
    if restarted, err := restartOrchServe(orchGoDir); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to restart orch serve: %v\n", err)
    }
}
```

This restarts `orch serve` (port 3348) which is the dashboard API server, not OpenCode server (port 4096).

**Source:** `cmd/orch/complete_cmd.go:1478-1485`, `.kb/investigations/2026-01-26-inv-opencode-server-keeps-crashing-dying.md:136-150`

**Significance:** The concern about "orch complete auto-rebuild triggers restarts that kill sessions" is unfounded. `orch complete` only restarts orch serve (:3348), not OpenCode (:4096). Agents would NOT be stranded by orch complete's auto-rebuild.

---

### Finding 6: Issue orch-go-21032 Is Fixing Root Cause

**Evidence:** Beads issue shows active work on auto-resume:
```
orch-go-21032: Auto-resume agents after OpenCode/server restart (lost sessions stay in_progress)
Status: open
Comments:
  [2026-01-29 20:01] Phase: Planning - Analyzing auto-resume mechanism for agents killed by server restart
  [2026-01-29 20:07] Phase: Implementing - Root cause identified: FindOrphanedSessions only looks for in_progress issues, but agents killed early remain in open status. Also spawned_tracker TTL blocks re-spawning. Fixing both issues.
```

**Source:** `bd show orch-go-21032`

**Significance:** The auto-resume mechanism had a bug: `FindOrphanedSessions` only looked for `in_progress` issues, missing agents killed early (still in `open` status). Issue 21032 is implementing a fix to include both statuses. This explains why some stranded workers weren't being resumed - they were in "open" status when killed.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
