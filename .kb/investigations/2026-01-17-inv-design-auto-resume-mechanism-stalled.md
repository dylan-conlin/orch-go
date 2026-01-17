<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Auto-resume after server recovery should use daemon's existing recovery loop with server-restart-aware detection, gradual resume with server stabilization delay, and explicit context injection telling agents they were interrupted.

**Evidence:** Four-layer state model shows sessions persist on disk (survive restarts). Existing `RunPeriodicRecovery()` already handles idle agents. x-opencode-directory header enables disk session queries. Advisory-first principle (Jan 15) established resume as safe automation.

**Knowledge:** Server restart creates a specific recovery scenario distinct from normal idle detection: (1) all in-memory sessions lost, (2) disk sessions become orphaned, (3) beads Phase comments remain as ground truth. The key insight is that disk persistence + beads tracking already provides the foundation - we need detection and context injection, not new state.

**Next:** Implement in three phases: (1) Server restart detection, (2) Extend recovery loop with restart-aware resume, (3) Add recovery context injection to resume prompts.

**Promote to Decision:** recommend-yes - Establishes "server recovery as distinct failure mode" with specific detection and resume behavior, applicable to any system with persistent storage + ephemeral memory.

---

# Investigation: Design Auto-Resume Mechanism for Stalled OpenCode Agents After Server Recovery

**Question:** How should the orch daemon or OpenCode server detect and resume agents that were in-progress when the server crashed/recovered, using beads Phase comments and OpenCode session state?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-arch-design-auto-resume-17jan-b70a
**Phase:** Complete
**Next Step:** None - design ready for implementation
**Status:** Complete

---

## Problem Statement

When the OpenCode server crashes or restarts:
1. **In-memory sessions are lost** - Server process dies, memory cleared
2. **Disk sessions persist** - `~/.local/share/opencode/storage/` survives
3. **Agents were mid-task** - Had active work, now orphaned
4. **Beads state remains** - Phase comments show where agent was
5. **No automatic recovery** - Agents stay stalled until manual intervention

**The gap:** Existing `RunPeriodicRecovery()` detects idle agents but doesn't distinguish "normal idle" from "orphaned by server crash". Server recovery is a distinct failure mode that needs explicit handling.

---

## Findings

### Finding 1: Four-Layer State Model Provides Recovery Foundation

**Evidence:** From `.kb/models/agent-lifecycle-state-model.md`:

| Layer | Storage | Lifecycle | Survives Restart? |
|-------|---------|-----------|-------------------|
| **Beads comments** | `.beads/issues.jsonl` | Persistent | Yes |
| **OpenCode on-disk** | `~/.local/share/opencode/storage/` | Persistent | Yes |
| **OpenCode in-memory** | Server process | Until restart | **No** |
| **Tmux windows** | Runtime | Until closed | Variable |

**Source:** `.kb/models/agent-lifecycle-state-model.md:17-30`, `.kb/models/opencode-session-lifecycle.md:17-28`

**Significance:** Server restart only loses in-memory state. Disk sessions and beads state persist. Recovery can reconstruct what was happening from: (1) disk sessions (message history, directory), (2) beads Phase comments (where agent was), (3) workspaces (SPAWN_CONTEXT.md still exists). The infrastructure for recovery already exists - we need detection and coordination.

---

### Finding 2: x-opencode-directory Header Enables Disk Session Discovery

**Evidence:** From `pkg/opencode/client.go:286-311`:

```go
// ListSessions fetches all sessions from the OpenCode API.
// If directory is provided, it passes it via x-opencode-directory header.
func (c *Client) ListSessions(directory string) ([]Session, error) {
    req, err := http.NewRequest("GET", c.ServerURL+"/session", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    if directory != "" {
        req.Header.Set("x-opencode-directory", directory)
    }
    // ...
}
```

**Source:** `pkg/opencode/client.go:286-311`

**Significance:** Without the header, only in-memory sessions are returned. With the header, all historical sessions for that directory are returned from disk. After server restart, in-memory sessions = 0, but disk sessions retain full history. Recovery can query disk sessions to find orphaned work.

---

### Finding 3: Existing Recovery Loop Handles Normal Idle Detection

**Evidence:** From `pkg/daemon/daemon.go:1086-1174`:

```go
// RunPeriodicRecovery runs the periodic stuck agent recovery if due.
func (d *Daemon) RunPeriodicRecovery() *RecoveryResult {
    // ...
    for _, agent := range agents {
        // Skip agents that already reported Phase: Complete
        if strings.EqualFold(agent.Phase, "complete") {
            skipped++
            continue
        }
        // Check if agent is idle long enough to trigger recovery
        idleTime := now.Sub(agent.UpdatedAt)
        if idleTime < d.Config.RecoveryIdleThreshold {
            skipped++
            continue
        }
        // Rate limit: 1 resume per hour per agent
        if lastAttempt, exists := d.resumeAttempts[agent.BeadsID]; exists {
            timeSinceLastAttempt := now.Sub(lastAttempt)
            if timeSinceLastAttempt < d.Config.RecoveryRateLimit {
                skipped++
                continue
            }
        }
        // Attempt resume
        if err := ResumeAgentByBeadsID(agent.BeadsID); err != nil {
            // ...
        }
    }
}
```

**Source:** `pkg/daemon/daemon.go:1086-1174`

**Significance:** The recovery infrastructure exists. Current approach:
- Checks idle time (>10min default)
- Rate-limits resume (1/hour per agent)
- Skips Phase: Complete agents

Missing for server recovery:
- Server restart detection
- Bulk resume awareness (many agents at once)
- Server stabilization delay

---

### Finding 4: Beads Phase Comments Are Ground Truth for Agent State

**Evidence:** From `.kb/models/agent-lifecycle-state-model.md`:

| Question | Source | NOT this |
|----------|--------|----------|
| Is agent complete? | Beads issue `status = closed` | OpenCode session exists |
| What phase is agent in? | Beads comments (`Phase: X`) | Dashboard shows "active" |
| Did agent finish? | `Phase: Complete` comment exists | Session went idle |

**Source:** `.kb/models/agent-lifecycle-state-model.md:35-44`

**Significance:** Beads Phase comments are the canonical source for agent progress. After server restart:
- `Phase: Planning` → Agent was early in task
- `Phase: Implementing` → Agent was mid-work
- `Phase: Complete` → Agent finished (no resume needed)
- No Phase comment → Agent may not have started

Recovery should key off beads Phase state, not session state.

---

### Finding 5: Advisory-First Principle Established Resume as Safe Automation

**Evidence:** From `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md`:

> **Use a tiered approach** with advisory-first principle:
>
> | Tier | Action | Condition | Destructive? | Automatic? |
> |------|--------|-----------|--------------|------------|
> | 1 | Resume | Idle >10min, no Phase: Complete | No | Yes (rate-limited) |
> | 2 | Surface | Resume didn't help after 15min | No | Yes (visibility) |
> | 3 | Human decision | Surfaced agent | Varies | No |

**Source:** `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md:146-154`

**Significance:** Resume is non-destructive and safe to automate. Respawn and abandon are destructive and require human decision. Server recovery resume follows the same principle - attempt non-destructive resume, surface if it doesn't help.

---

### Finding 6: Resume Prompts Need Recovery Context

**Evidence:** Current resume prompt from `cmd/orch/resume.go:92-100`:

```go
func GenerateResumePrompt(workspaceName, projectDir, beadsID string) string {
    contextPath := filepath.Join(projectDir, ".orch", "workspace", workspaceName, "SPAWN_CONTEXT.md")
    return fmt.Sprintf(
        "You were paused mid-task. Re-read your spawn context from %s and continue your work. "+
            "Report progress via bd comment %s.",
        contextPath,
        beadsID,
    )
}
```

**Source:** `cmd/orch/resume.go:92-100`

**Significance:** Current prompt says "paused mid-task" but doesn't explain WHY. For server recovery, agents need to know:
1. The server crashed/restarted (not normal pause)
2. Some context may be lost (in-memory state gone)
3. They should validate their state before continuing

This aligns with **Pain as Signal** principle - agents should "feel" the friction of their failure.

---

## Design Forks

### Fork 1: Detection Mechanism

**Options:**
- A: Server startup detection (daemon detects fresh server, triggers recovery scan)
- B: Polling-based detection (extend existing recovery loop)
- C: Session state tracking (track "was in-progress" before crash)

**Substrate says:**
- **Infrastructure Over Instruction**: Detection must be infrastructure, not dependent on agent remembering
- **Graceful Degradation**: Should work without additional tracking state

**RECOMMENDATION:** Option A + B hybrid - Server startup detection triggers immediate scan, then polling handles stragglers.

**Trade-off accepted:** Requires daemon to know server lifecycle (new coupling)
**When this would change:** If OpenCode server itself implements recovery (Option C becomes viable)

---

### Fork 2: Which Sessions to Resume

**Options:**
- A: Resume all in-progress sessions (by beads Phase comment)
- B: Resume only sessions with workspace (can find SPAWN_CONTEXT.md)
- C: Resume only headless sessions (tmux sessions have visual escape hatch)

**Substrate says:**
- **Resilient Infrastructure Patterns** (CLAUDE.md): "Escape hatch" sessions (--tmux) are for critical work that needs visual monitoring
- **Session Amnesia**: All agents need context injection regardless of spawn mode

**RECOMMENDATION:** Option A - Resume all in-progress by beads Phase. Tmux agents still benefit from resume prompt even if user can see them.

**Trade-off accepted:** May resume agents user is actively watching
**When this would change:** If tmux agents have different recovery needs (none identified)

---

### Fork 3: Resume Timing

**Options:**
- A: Immediate (on server recovery detection)
- B: Delayed (wait for server stability, e.g., 30 seconds)
- C: Gradual (stagger resumes to prevent rate limits)

**Substrate says:**
- **Verification Bottleneck**: Can't resume faster than can verify behavior
- **Advisory-First**: Rate-limit resume attempts (1/hour per agent)

**RECOMMENDATION:** Option B + C - Wait 30 seconds for server stability, then stagger resumes (10 second delay between each).

**Trade-off accepted:** ~30 seconds latency before first resume
**When this would change:** If server stability is reliable immediately after start

---

### Fork 4: Resume Authority

**Options:**
- A: Daemon handles it (extends existing recovery loop)
- B: OpenCode server handles it (server-side recovery)
- C: New recovery command (`orch recover-server`)

**Substrate says:**
- **Compose Over Monolith**: Extend existing capability rather than add new command
- **Local-First**: Keep logic in orch-go, not OpenCode server

**RECOMMENDATION:** Option A - Extend daemon's `RunPeriodicRecovery()` with server-restart awareness.

**Trade-off accepted:** Daemon must be running for recovery (but daemon is the autonomous processing entry point anyway)
**When this would change:** If recovery needed without daemon running

---

## Synthesis

**Key Insights:**

1. **Server restart is a distinct failure mode** - Different from normal idle detection because ALL in-memory sessions are lost simultaneously, disk sessions become orphaned, and bulk resume is needed. The detection and timing are different even though the resume mechanism is the same.

2. **Infrastructure already exists** - Four-layer state model provides: disk sessions (message history), beads Phase (progress tracking), workspaces (SPAWN_CONTEXT.md). We're adding detection and coordination, not new state.

3. **Resume needs recovery context** - Agents should know they were interrupted by server crash, not just "paused". This aligns with Pain as Signal - the friction creates awareness of the failure mode.

**Answer to Investigation Question:**

The auto-resume mechanism should:

1. **Detect server restart** via daemon polling - check if server was recently started (uptime < threshold) and if there are disk sessions without matching in-memory sessions.

2. **Identify resumable sessions** by:
   - Query disk sessions with x-opencode-directory header
   - Cross-reference with beads issues that are `in_progress` or `open`
   - Filter to those with Phase != Complete
   - Find corresponding workspaces

3. **Resume with awareness**:
   - Wait 30 seconds for server stability
   - Stagger resumes (10 second delay)
   - Rate-limit per agent (existing 1/hour limit)
   - Inject recovery-specific context ("server restarted, validate state")

4. **Surface unrecoverable** - If resume doesn't help after 15 minutes (per existing pattern), surface in Needs Attention.

---

## Structured Uncertainty

**What's tested:**

- ✅ Sessions persist on disk (verified: read opencode-session-lifecycle model, confirmed storage at `~/.local/share/opencode/storage/`)
- ✅ x-opencode-directory header returns disk sessions (verified: read client.go implementation)
- ✅ Existing recovery loop handles idle agents (verified: read daemon.go implementation)
- ✅ Resume is non-destructive (verified: stuck agent recovery investigation)

**What's untested:**

- ⚠️ Server restart detection via uptime check (implementation detail)
- ⚠️ 30-second stabilization delay is sufficient (educated guess)
- ⚠️ 10-second stagger between resumes prevents rate limits (untested)
- ⚠️ Recovery context improves agent behavior (hypothesis)

**What would change this:**

- If OpenCode adds server-side session recovery, client-side detection becomes unnecessary
- If disk sessions don't survive certain crash types, need additional persistence
- If resumed agents consistently fail after recovery, need different approach (respawn vs resume)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Server-Restart-Aware Daemon Recovery** - Extend existing `RunPeriodicRecovery()` with server restart detection, stabilization delay, and recovery-specific resume prompts.

**Why this approach:**
- Extends existing infrastructure (daemon recovery loop)
- Non-destructive (resume is safe to automate per advisory-first)
- Uses existing state (disk sessions + beads Phase)
- Aligns with Compose Over Monolith (no new commands)

**Trade-offs accepted:**
- Requires daemon to be running for recovery
- ~30 second latency before first resume
- Assumes server stability after 30 seconds

**Implementation sequence:**

1. **Server restart detection** - Add function to detect if server was recently restarted (uptime < 2 minutes) or if daemon just started and server is running.

2. **Recovery scan** - On restart detection, scan disk sessions for resumable work:
   - Get all disk sessions via x-opencode-directory header
   - Cross-reference with beads issues (in_progress or open status)
   - Filter to Phase != Complete
   - Match to workspaces

3. **Staggered resume** - Resume identified sessions with stabilization delay and stagger:
   - Wait 30 seconds after restart detection
   - Resume one agent every 10 seconds
   - Use existing rate limiting (1/hour per agent)

4. **Recovery context injection** - Update resume prompt for recovery scenarios:
   - Tell agent server crashed/restarted
   - Advise validation of current state
   - Point to workspace for context

### Alternative Approaches Considered

**Option B: OpenCode Server-Side Recovery**
- **Pros:** Cleaner separation, server knows its own state
- **Cons:** Requires OpenCode changes, not orch-go's codebase
- **When to use instead:** If Dylan wants recovery in OpenCode rather than orchestration layer

**Option C: New `orch recover-server` Command**
- **Pros:** Explicit, can be triggered manually
- **Cons:** Adds another command, doesn't leverage existing daemon
- **When to use instead:** If daemon-independent recovery is needed

**Rationale for recommendation:** Option A (extend daemon) follows Compose Over Monolith, leverages existing recovery infrastructure, and keeps recovery logic in orch-go where it belongs.

---

### Implementation Details

**What to implement first:**
1. Server restart detection (foundation for all else)
2. Disk session scanning with beads cross-reference
3. Staggered resume with delay
4. Recovery-specific resume prompt

**Things to watch out for:**
- ⚠️ Don't resume completed agents (Phase: Complete check is critical)
- ⚠️ Cross-project agents may have different directories (use x-opencode-directory correctly)
- ⚠️ Rate limit applies per agent, not globally (prevent bulk resume from exceeding limits)
- ⚠️ Server may not be fully ready immediately after start (hence 30s delay)

**Areas needing further investigation:**
- How to detect server uptime programmatically (may need OpenCode API endpoint)
- Whether 30 seconds is the right stabilization delay
- Whether recovery prompt actually improves agent behavior

**Success criteria:**
- ✅ After server restart, agents resume automatically within 2 minutes
- ✅ Resumed agents receive recovery context in prompt
- ✅ Phase: Complete agents are NOT resumed
- ✅ Rate limiting prevents overwhelming the server
- ✅ Dashboard shows resumed agents as "Recovered" or similar

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - Existing recovery loop implementation
- `pkg/opencode/client.go` - Session API client, x-opencode-directory header
- `pkg/opencode/types.go` - Session structure
- `cmd/orch/resume.go` - Existing resume command and prompt generation
- `.kb/models/agent-lifecycle-state-model.md` - Four-layer state model
- `.kb/models/opencode-session-lifecycle.md` - Session persistence model
- `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Stuck agent recovery design
- `.kb/guides/session-resume-protocol.md` - Session resume protocol
- `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch patterns
- `~/.kb/principles.md` - Foundational principles

**Commands Run:**
```bash
# Create investigation file
kb create investigation design-auto-resume-mechanism-stalled

# Check beads context
bd show orch-go-byxj3
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-15-ghost-visibility-over-cleanup.md` - Filtering over cleanup
- **Investigation:** `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Stuck agent recovery (advisory-first)
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Four-layer state model
- **Model:** `.kb/models/opencode-session-lifecycle.md` - Session persistence

---

## Investigation History

**2026-01-17 14:30:** Investigation started
- Initial question: How to auto-resume agents after server recovery?
- Context: Server crashes lose in-memory sessions, agents become orphaned

**2026-01-17 15:00:** Context gathering complete
- Read models: agent-lifecycle-state-model, opencode-session-lifecycle
- Read guides: session-resume-protocol, resilient-infrastructure-patterns
- Read investigations: stuck agent recovery, session resume design
- Read code: daemon.go, resume.go, client.go

**2026-01-17 15:30:** Design forks identified and navigated
- Detection: Server startup + polling hybrid
- Which sessions: All in-progress by beads Phase
- Timing: 30s delay + 10s stagger
- Authority: Extend daemon recovery loop

**2026-01-17 16:00:** Investigation completed
- Status: Complete
- Key outcome: Design ready - extend daemon recovery with server-restart detection, stabilization delay, recovery-specific prompts
