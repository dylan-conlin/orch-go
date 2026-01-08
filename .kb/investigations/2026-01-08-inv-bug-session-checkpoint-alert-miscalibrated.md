## Summary (D.E.K.N.)

**Delta:** Session checkpoint thresholds now support type-aware durations - orchestrator sessions (4h/6h/8h) vs agent sessions (2h/3h/4h).

**Evidence:** Tests pass showing orchestrator at 3h is "ok" while agent at 3h is "strong"; configurable via ~/.orch/config.yaml session settings.

**Knowledge:** Orchestrators coordinate work (spawn/complete), not accumulate implementation context, so longer sessions are safe before quality degradation.

**Next:** close - implementation complete with tests, backward-compatible with existing code.

**Promote to Decision:** recommend-yes - Establishes a key principle: session type determines checkpoint thresholds.

---

# Investigation: Bug Session Checkpoint Alert Miscalibrated

**Question:** How should session checkpoint alerts differ between orchestrator and agent sessions?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Verification:** Reproduction verified - 5h39m session now shows warning (not exceeded)

---

## Findings

### Finding 1: Current implementation uses hardcoded agent-centric thresholds

**Evidence:** pkg/session/session.go had hardcoded constants:
- CheckpointWarningDuration = 2h
- CheckpointStrongDuration = 3h  
- CheckpointMaxDuration = 4h

**Source:** pkg/session/session.go:27-36

**Significance:** These thresholds assume context degradation from implementation work, which doesn't apply to orchestrator sessions that primarily coordinate via spawn/complete.

---

### Finding 2: Orchestrator sessions have different context degradation patterns

**Evidence:** From spawn context analysis:
- Orchestrators don't accumulate implementation context
- Each spawn/complete is relatively independent
- Coordination state doesn't degrade like code understanding
- Orchestrators delegate to agents, so their work is naturally checkpointed

**Source:** SPAWN_CONTEXT.md analysis

**Significance:** Orchestrator sessions can safely run longer before quality degrades. The recommended ratio is 2x agent thresholds.

---

### Finding 3: Config system already supports typed settings

**Evidence:** pkg/userconfig/userconfig.go already handles:
- DaemonConfig with multiple typed sub-settings
- ReflectConfig with interval and enabled flags
- Clean getter methods with defaults

**Source:** pkg/userconfig/userconfig.go

**Significance:** Adding SessionConfig follows established patterns and maintains consistency.

---

## Synthesis

**Key Insights:**

1. **Session type determines checkpoint thresholds** - Agents accumulate implementation context (shorter thresholds), orchestrators coordinate (longer thresholds).

2. **Defaults should be sensible but configurable** - Orchestrator: 4h/6h/8h, Agent: 2h/3h/4h, both customizable via ~/.orch/config.yaml.

3. **Backward compatibility is essential** - Existing GetCheckpointStatus() method uses agent thresholds for backward compatibility.

**Answer to Investigation Question:**

Orchestrator sessions should use longer checkpoint thresholds (4h warning, 6h strong, 8h max) compared to agent sessions (2h/3h/4h) because orchestrators coordinate work rather than accumulate implementation context. The implementation adds:
- SessionType enum (agent/orchestrator)
- GetCheckpointStatusWithType() for type-aware thresholds
- GetCheckpointStatusWithThresholds() for custom thresholds
- SessionConfig in ~/.orch/config.yaml for user customization

---

## Structured Uncertainty

**What's tested:**

- Tests verify orchestrator at 3h is "ok" while agent at 3h is "strong"
- Tests verify threshold ordering (warning < strong < max)
- Tests verify config loading from YAML
- Tests verify defaults when section is missing

**What's untested:**

- Real-world validation of 8h orchestrator max being appropriate
- Performance impact of longer sessions on coordination quality

**What would change this:**

- If orchestrators showed quality degradation at shorter durations in practice
- If coordination state was found to degrade faster than expected

---

## Implementation Recommendations

### Recommended Approach: Implemented

**Tier-aware thresholds with config support** - Session type determines default thresholds, user can customize via config.

**Why this approach:**
- Directly addresses the root cause (agent vs orchestrator context patterns)
- Configurable to handle edge cases
- Backward compatible with existing code

**Trade-offs accepted:**
- Slightly more complexity in session package
- Two sets of thresholds to maintain

**Implementation sequence:**
1. Add SessionConfig and CheckpointThresholds to userconfig
2. Add SessionType and threshold functions to session package
3. Add GetCheckpointStatusWithType() and GetCheckpointStatusWithThresholds()
4. Update session.go CLI to use orchestrator thresholds

---

## References

**Files Modified:**
- pkg/session/session.go - Added SessionType, CheckpointThresholds, type-aware methods
- pkg/session/session_test.go - Added tests for new functionality
- pkg/userconfig/userconfig.go - Added SessionConfig with checkpoint settings
- pkg/userconfig/userconfig_test.go - Added tests for session config
- cmd/orch/session.go - Updated to use orchestrator thresholds

**Commands Run:**
```bash
go build ./...
go test ./pkg/session/... -v
go test ./pkg/userconfig/... -v
go test ./... -timeout 5m
```

---

## Investigation History

**2026-01-08 HH:MM:** Investigation started
- Initial question: How to differentiate checkpoint alerts for orchestrator vs agent sessions
- Context: Orchestrator sessions hitting 4h max felt premature for coordination work

**2026-01-08 HH:MM:** Investigation completed
- Status: Complete
- Key outcome: Implemented tier-aware checkpoint thresholds with config support
