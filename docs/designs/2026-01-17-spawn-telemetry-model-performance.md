# Design: Spawn Telemetry for Model Performance Tracking

**Date:** 2026-01-17
**Status:** Proposed
**Author:** orch-go-x67lc

---

## Problem Statement

Model selection for agent spawning is currently opinion-based ("opus is better for architecture"). We need evidence-based selection with data like "sonnet has 89% success rate on surgical feature-impl tasks".

**Gaps:**
- No structured telemetry for model performance across spawn/complete/abandon lifecycle
- Missing data: model used, tokens consumed, duration, outcome classification, retry counts
- No queryable format to answer "which model performs best for which skills?"

---

## Goals

1. **Instrument lifecycle endpoints** (spawn, complete, abandon) to collect model performance data
2. **Capture comprehensive metrics**: model, skill, outcome, tokens, duration, retries, failure reasons
3. **Enable queryable analysis** for evidence-based model selection decisions
4. **Minimal disruption** to existing spawn/complete/abandon workflows

---

## Design Decision

### Extend Existing Events System

**Rationale:**
- Events system (`pkg/events/logger.go`) already tracks lifecycle with JSONL format
- Already logs spawn, completion, error events with metadata
- Proven infrastructure (no need to create parallel telemetry system)
- JSONL format is queryable via jq, grep, or simple scripts

**Alternative Considered:** Create separate `pkg/telemetry` package with dedicated telemetry.jsonl file
- **Rejected because:** Adds complexity, duplicates infrastructure, splits related data across files

---

## Architecture

### Data Collection Points

```
┌────────────────────────────────────────────────────────────┐
│ SPAWN (cmd/orch/spawn_cmd.go)                             │
│ Collect: model, skill, spawn_time, beads_id, session_id   │
│ Event: Enhance session.spawned with model + skill         │
└────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────┐
│ COMPLETE (cmd/orch/complete_cmd.go)                       │
│ Collect: duration, tokens, outcome=success                │
│ Event: Enhance agent.completed with duration + tokens     │
└────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────┐
│ ABANDON (cmd/orch/abandon_cmd.go)                         │
│ Collect: duration, tokens, outcome=abandoned, reason      │
│ Event: NEW agent.abandoned with duration + tokens         │
└────────────────────────────────────────────────────────────┘
```

### Event Schema Changes

#### Enhanced `session.spawned` Event
```json
{
  "type": "session.spawned",
  "session_id": "abc123",
  "timestamp": 1705500000,
  "data": {
    "prompt": "...",
    "title": "...",
    "model": "sonnet-4",        // NEW
    "skill": "feature-impl",    // NEW
    "beads_id": "orch-go-x67lc" // NEW
  }
}
```

#### Enhanced `agent.completed` Event
```json
{
  "type": "agent.completed",
  "session_id": "orch-go-x67lc",
  "timestamp": 1705503600,
  "data": {
    "beads_id": "orch-go-x67lc",
    "workspace": "...",
    "reason": "...",
    "forced": false,
    "verification_passed": true,
    "skill": "feature-impl",
    "duration_seconds": 3600,     // NEW: completion_time - spawn_time
    "tokens_input": 45000,        // NEW: from OpenCode API
    "tokens_output": 12000,       // NEW: from OpenCode API
    "outcome": "success"          // NEW: success|forced|failed
  }
}
```

#### New `agent.abandoned` Event
```json
{
  "type": "agent.abandoned",
  "session_id": "orch-go-x67lc",
  "timestamp": 1705503000,
  "data": {
    "beads_id": "orch-go-x67lc",
    "workspace": "...",
    "reason": "out of context",
    "skill": "feature-impl",
    "duration_seconds": 2400,
    "tokens_input": 120000,
    "tokens_output": 8000,
    "outcome": "abandoned"
  }
}
```

---

## Implementation Plan

### Phase 1: Enhance Spawn Event (20 min)

**Changes:**
1. Update `LogSpawn()` in `pkg/events/logger.go` to accept `model`, `skill`, `beadsID` parameters
2. Modify `cmd/orch/spawn_cmd.go` to pass these values when calling `LogSpawn()`
3. Update tests in `pkg/events/logger_test.go`

**Files:**
- `pkg/events/logger.go:99-110` - LogSpawn signature and implementation
- `cmd/orch/spawn_cmd.go` - spawn command (search for LogSpawn call)

### Phase 2: Add Token Retrieval from OpenCode (30 min)

**Changes:**
1. Add `GetSessionTokens(sessionID string) (input, output int, error)` to `pkg/opencode/client.go`
2. Query OpenCode API `/api/sessions/{id}/messages` to sum token counts
3. Add error handling for session not found / API unavailable

**Files:**
- `pkg/opencode/client.go` - add GetSessionTokens method

**API Contract:**
```go
// GetSessionTokens retrieves total token usage for a session
// Returns (inputTokens, outputTokens, error)
func (c *Client) GetSessionTokens(sessionID string) (int, int, error)
```

### Phase 3: Enhance Completion Event (30 min)

**Changes:**
1. Update `AgentCompletedData` struct to include `DurationSeconds`, `TokensInput`, `TokensOutput`, `Outcome`
2. Update `LogAgentCompleted()` to serialize new fields
3. Modify `cmd/orch/complete_cmd.go` to:
   - Read `.spawn_time` from workspace
   - Calculate duration
   - Call `GetSessionTokens()` to get token usage
   - Determine outcome (success/forced/failed)
   - Pass enriched data to `LogAgentCompleted()`

**Files:**
- `pkg/events/logger.go:201-242` - AgentCompletedData and LogAgentCompleted
- `cmd/orch/complete_cmd.go` - complete command

### Phase 4: Add Abandon Event (30 min)

**Changes:**
1. Add `AgentAbandonedData` struct to `pkg/events/logger.go`
2. Add `LogAgentAbandoned()` method
3. Add `EventTypeAgentAbandoned = "agent.abandoned"` constant
4. Modify `cmd/orch/abandon_cmd.go` to log abandonment with telemetry

**Files:**
- `pkg/events/logger.go` - new AgentAbandonedData struct and LogAgentAbandoned method
- `cmd/orch/abandon_cmd.go` - abandon command

### Phase 5: Add Query Tool (40 min - optional)

**Option A:** Shell script using `jq` for quick analysis
```bash
# ~/.orch/bin/orch-telemetry-query
# Examples:
#   orch-telemetry-query model-success-rate sonnet-4
#   orch-telemetry-query skill-performance feature-impl
```

**Option B:** Go subcommand `orch telemetry query`
- Richer query capabilities
- JSON output for programmatic use
- Built-in aggregation

**Recommendation:** Start with Option A (jq script) for MVP, upgrade to Option B if demand exists.

---

## Testing Strategy

### Unit Tests
- `pkg/events/logger_test.go` - test enhanced event logging
- `pkg/opencode/client_test.go` - test GetSessionTokens with mock API

### Integration Tests
- Spawn agent, verify event logged with model/skill
- Complete agent, verify duration/tokens calculated correctly
- Abandon agent, verify abandonment event logged

### Manual Validation
1. Spawn test agent: `orch spawn feature-impl "test task" --bypass-triage`
2. Complete or abandon
3. Inspect `~/.orch/events.jsonl` - verify telemetry fields present

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| OpenCode API unavailable | Missing token data | Log event with `tokens_input: -1` to indicate unavailable, continue operation |
| Session ID not found | Missing token data | Same as above - graceful degradation |
| Large events.jsonl file | Performance degradation | Events system already handles this (append-only, no lock contention) |
| Breaking existing consumers | Dashboard/monitoring breaks | Use optional fields - existing code ignores unknown fields in JSON |

---

## Success Criteria

- ✅ All spawn events include `model` and `skill` fields
- ✅ All completion events include `duration_seconds`, `tokens_input`, `tokens_output`
- ✅ All abandonment events logged with outcome and telemetry
- ✅ Can query `events.jsonl` to answer: "What's the success rate of sonnet-4 on feature-impl tasks?"
- ✅ No regression in spawn/complete/abandon performance (telemetry adds <100ms overhead)

---

## Future Enhancements (Out of Scope)

- **Retry tracking**: Track spawns that are re-attempts of failed agents
- **Cost tracking**: Multiply tokens by model pricing to track costs
- **Dashboard integration**: Real-time model performance visualization
- **Automated model selection**: Use telemetry to auto-select best model for skill
- **A/B testing**: Random model assignment to compare performance

---

## References

- **Investigation:** `.kb/investigations/2026-01-17-inv-add-spawn-telemetry-model-performance.md`
- **Events System:** `pkg/events/logger.go`
- **Spawn Architecture:** `.kb/models/spawn-architecture.md`
- **Completion Gates:** `.kb/guides/completion-gates.md`
