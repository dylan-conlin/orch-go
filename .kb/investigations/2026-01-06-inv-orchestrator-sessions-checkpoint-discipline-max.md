## Summary (D.E.K.N.)

**Delta:** Implemented session checkpoint discipline with 2h/3h/4h thresholds and visual warnings in `orch session status`.

**Evidence:** Tests pass, session status now shows checkpoint level (ok/warning/strong/exceeded) with actionable guidance.

**Knowledge:** Checkpoint discipline is best enforced via visibility (status output) rather than hard blocks - allows orchestrator judgment while surfacing risk.

**Next:** Close issue. Consider future enhancement: automated checkpoint reminders via daemon or event hooks.

**Promote to Decision:** Actioned - patterns in orchestrator skill (session hygiene)

---

# Investigation: Orchestrator Sessions Checkpoint Discipline Max

**Question:** How should orchestrator sessions enforce checkpoint discipline to prevent quality degradation from context exhaustion?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Session infrastructure already supports duration tracking

**Evidence:** 
- `pkg/session/session.go` has `Store.Duration()` method that calculates `time.Since(session.StartedAt)`
- `cmd/orch/session.go` already formats and displays duration in status output
- Session state persists across restarts via `~/.orch/session.json`

**Source:** 
- pkg/session/session.go:257-268
- cmd/orch/session.go:241

**Significance:** Foundation for checkpoint tracking exists - just needed threshold logic and user-facing warnings.

---

### Finding 2: Prior decision established 75-80% context threshold

**Evidence:**
From kb context in SPAWN_CONTEXT.md:
> "Orchestrator sessions should transition at 75-80% context usage"
> Reason: "At 78% (156k tokens), quality still good but approaching risk zone."

**Source:** SPAWN_CONTEXT.md lines 75-76 (kb context output)

**Significance:** Establishes that session duration limits are already an accepted principle. The 2h/3h/4h thresholds map roughly to context usage progression.

---

### Finding 3: Evidence of harm from long sessions

**Evidence:**
From spawn context problem statement:
- "pw-orch-resume-price-watch-06jan-bcd7: 5h session, partial outcome"
- "Long sessions without checkpoints lead to: context exhaustion, degraded output quality, lost work if session dies, no opportunity for meta-orchestrator course correction"

**Source:** SPAWN_CONTEXT.md lines 5-11

**Significance:** Real-world evidence that checkpoint discipline addresses an actual problem, not a theoretical concern.

---

## Synthesis

**Key Insights:**

1. **Duration-based thresholds are a practical proxy** - While context token usage would be ideal, duration is easily measurable and correlates with context consumption.

2. **Visibility over enforcement** - The implementation adds warnings to `orch session status` rather than hard blocks. This respects orchestrator judgment while making risk visible.

3. **Graduated urgency levels** - Three thresholds (warning at 2h, strong at 3h, exceeded at 4h) provide progressive escalation, matching the gradual nature of context degradation.

**Answer to Investigation Question:**

Checkpoint discipline is enforced through:
1. Constants in `pkg/session/session.go` defining thresholds (2h/3h/4h)
2. `GetCheckpointStatus()` method that returns current checkpoint level
3. Visual warnings in `orch session status` output with actionable guidance
4. Summary advice in `orch session end` for sessions that ran long

This approach surfaces risk without blocking execution, allowing orchestrators to make informed decisions while maintaining accountability through visible status.

---

## Structured Uncertainty

**What's tested:**

- Tests pass for all checkpoint status levels (ok, warning, strong, exceeded)
- Tests verify threshold constant ordering (2h < 3h < 4h)
- Manual verification: `orch session status` shows checkpoint info in both text and JSON output

**What's untested:**

- Long-running session behavior (would need to manually test 2h+ session)
- Daemon/event-based automated reminders (not implemented - listed as future enhancement)
- Integration with session-transition skill (orchestrator skill update not in scope)

**What would change this:**

- If context token usage became easily queryable, could switch from duration-based to token-based thresholds
- If automated reminders prove necessary, could add daemon polling or SSE events

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Duration-based checkpoint warnings in session status** - Adds checkpoint discipline without blocking execution.

**Why this approach:**
- Leverages existing session infrastructure
- Non-intrusive - information only, orchestrator decides action
- Graduated warnings match reality of gradual degradation
- JSON output enables scripting/automation

**Trade-offs accepted:**
- Duration is proxy for context usage, not precise measurement
- No automated reminders (requires manual `orch session status` check)

**Implementation completed:**
1. Added checkpoint constants (2h/3h/4h) to pkg/session/session.go
2. Added `GetCheckpointStatus()` method returning level/message/duration/next_threshold
3. Updated `orch session status` text output with visual indicators and actionable guidance
4. Updated `orch session end` to show advice for long sessions
5. Added comprehensive tests

### Alternative Approaches Considered

**Option B: Hard blocks at threshold**
- **Pros:** Enforces discipline strictly
- **Cons:** May block critical work inappropriately
- **When to use instead:** If soft warnings prove insufficient

**Option C: Daemon-based automated reminders**
- **Pros:** Proactive notification
- **Cons:** More complex, requires daemon changes
- **When to use instead:** If orchestrators consistently miss status warnings

---

## Implementation Details

**Files changed:**
- `pkg/session/session.go` - Added constants and GetCheckpointStatus()
- `cmd/orch/session.go` - Updated status and end commands
- `pkg/session/session_test.go` - Added checkpoint tests

**Success criteria:**
- `orch session status` shows checkpoint level inline with duration
- Detailed warning appears for warning/strong/exceeded levels
- JSON output includes checkpoint object
- Session end shows advice for long sessions

---

## References

**Files Examined:**
- pkg/session/session.go - Session state management
- cmd/orch/session.go - Session CLI commands
- pkg/session/session_test.go - Existing tests

**Commands Run:**
```bash
# Build and test
go test ./pkg/session/... -v
make install
orch session start "Test checkpoint feature"
orch session status
orch session status --json
orch session end
```

**Related Artifacts:**
- **Decision:** kb context showed "Orchestrator sessions should transition at 75-80% context usage"
- **Investigation:** SPAWN_CONTEXT.md documented 5h session with partial outcome as evidence

---

## Investigation History

**2026-01-06 18:30:** Investigation started
- Initial question: How to enforce orchestrator session checkpoint discipline?
- Context: 5h session with partial outcome motivated this work

**2026-01-06 18:37:** Implementation completed
- Added checkpoint constants and GetCheckpointStatus() to session package
- Updated session status command with visual warnings
- Added tests for checkpoint logic
- All tests pass, CLI builds and works as expected

**2026-01-06 18:40:** Investigation completed
- Status: Complete
- Key outcome: Session checkpoint discipline now surfaced via `orch session status` with 2h/3h/4h thresholds
