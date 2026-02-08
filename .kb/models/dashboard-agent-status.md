# Model: Dashboard Agent Status Calculation

**Domain:** Dashboard / Agent Lifecycle
**Last Updated:** 2026-01-29
**Synthesized From:** 11 investigations (Jan 4-8, 2026) into completion verification, status display, and registry behavior; extended Jan 18, 2026 with tmux visibility architecture

---

## Summary (30 seconds)

The dashboard calculates agent status through a **Priority Cascade model**: check Phase first (highest priority), then registry state, then session existence. Status can be "wrong" at the dashboard level while being "correct" at each individual check - this is a measurement artifact, not a system failure. The root issue was observation infrastructure (missing events, double-counting metrics, state not surfaced), not broken completion logic.

---

## Core Mechanism

### How Status Is Calculated

**Priority Cascade (Jan 8, 2026):**
```
1. Phase check (highest priority)
   - If Phase: Complete → status = completed

2. Registry state check
   - If Status: abandoned → status = abandoned
   - If Status: running → status = running

3. Session existence check (lowest priority)
   - If session doesn't exist → status = dead
   - If session exists → status = running
```

**Why cascade order matters:**
- Phase is canonical (agent's own declaration)
- Registry is authoritative (human-set via `orch abandon`)
- Session existence is fallback (infrastructure check)

### State Transitions

**Normal lifecycle:**
```
spawned (registry + session created)
    ↓
running (working toward Phase: Complete)
    ↓
Phase: Complete reached (agent declares done)
    ↓
orch complete runs (verification + beads close)
    ↓
completed (registry marked, session may remain)
```

**Abandoned path:**
```
spawned → running
    ↓
orch abandon (human judgment)
    ↓
Status: abandoned in registry
    ↓
Dashboard shows abandoned (override Phase)
```

### Critical Invariants

1. **Phase: Complete is agent's declaration** - Only agent can reach this, not orchestrator
2. **Registry is source of truth for abandonment** - Human judgment, can't be inferred
3. **Session may outlive completion** - Session existence ≠ agent still working
4. **Status checks don't mutate state** - Calculation is read-only, no side effects

### Tmux Agent Status (Escape Hatch / Claude CLI)

**Extension to Priority Cascade (Jan 18, 2026):**

Claude CLI agents spawned with `--backend claude --tmux` use the same Priority Cascade but with different data sources:

```
1. Phase check (highest priority)
   - Extract beads ID from tmux window name: `{emoji} {workspace-name} [{beads-id}]`
   - Fetch Phase from beads comments (already implemented in batch fetch)
   - If Phase: Complete → status = completed

2. Registry state check
   - If Status: abandoned → status = abandoned

3. Activity detection (replaces session existence)
   - Monitor Claude Code transcript file mtime: `~/.claude/projects/.../[session_id].jsonl`
   - Fallback: Parse tmux pane content for TUI activity indicators
   - No activity for 3+ minutes → status = dead
   - Otherwise → status = active
```

**Key differences from OpenCode agents:**

| Data Point | OpenCode Agents | Claude CLI (tmux) |
|------------|-----------------|-------------------|
| Session API | Yes (OpenCode HTTP) | No (architectural constraint) |
| Token usage | Available | **Not available** (Max subscription, no per-session API) |
| Activity signal | session.last_updated | Transcript file mtime or pane content |
| Runtime | session.created_at | `.spawn_time` file in workspace |

**Why token usage is unavailable:** Claude CLI (Max subscription) doesn't expose per-session token usage via API. This is an **architectural constraint, not a fixable gap**. Progress tracking via Phase comments is more valuable anyway.

**Reference:** `.kb/investigations/archived/2026-01-18-design-dashboard-add-tmux-session-visibility.md`

---

## Why This Fails

### Failure Mode 1: "Dead" Agents That Completed

**Symptom:** Dashboard shows "dead", but work is done and beads issue closed

**Root cause:** Session cleanup happens async, dashboard checks session existence as fallback

**Why it happens:**
- Agent reaches Phase: Complete
- `orch complete` verifies and closes beads issue
- Session cleanup happens later (or not at all)
- Dashboard cascade reaches session check → sees no session → "dead"

**Fix (Jan 8):** Priority Cascade puts Phase check before session check

### Failure Mode 2: Metrics Show Wrong Completion Rate

**Symptom:** `orch stats` showed 72% completion when reality was 89%

**Root cause:** Metrics counted events (double-counting) instead of deduplicating entities

**Why it happens:**
- `agent.completed` event emitted by both `orch complete` AND beads close hook
- Metrics counted events → some completions counted 2x
- Result: inflated total, deflated completion %

**Fix (Jan 8):** Metrics deduplicate by beads_id before calculating ratios

### Failure Mode 3: Work Completed via Bypass Paths

**Symptom:** Beads issue closed but no completion event, dashboard doesn't update

**Root cause:** `bd close` (direct) doesn't emit events, only `orch complete` does

**Why it happens:**
- Multiple paths to completion: `orch complete`, `bd close`, `bd sync` with commit message
- Only `orch complete` emits events
- Other paths are invisible to observation infrastructure

**Fix (Jan 6):** Beads close hook emits `agent.completed` event

---

## Constraints

### Why Can't We Just Poll Session State?

**Constraint:** OpenCode server doesn't expose session state via HTTP API

**Implication:** Dashboard can't ask "is session actually working?" - can only check existence

**Workaround:** SSE events for real-time updates, but dashboard refresh is pull-based

**This enables:** Simple API design without exposing internal session state
**This constrains:** Dashboard must infer status from existence, not actual execution state

### Why Can't Agents Mark Themselves Complete in Registry?

**Constraint:** Agents don't have write access to `~/.orch/registry.json`

**Implication:** Only `orch complete` (orchestrator tool) can update registry

**Workaround:** Agent declares Phase, orchestrator verifies and records

**This enables:** Separation of concerns (agents declare, orchestrator verifies)
**This constrains:** Agents cannot self-terminate or update their own registry state

### Why Not Single Source of Truth?

**Constraint:** Three systems with different responsibilities:
- **Agent workspace** (Phase in .phase file) - agent's declaration
- **Registry** (~/.orch/registry.json) - orchestrator's record
- **Beads** (.beads/issues.jsonl) - work tracking

**Implication:** Status is computed by combining sources, not stored in one place

**Workaround:** Dashboard reconciles across systems

**This enables:** Each system optimized for its purpose (declaration, tracking, work management)
**This constrains:** Cannot query single location for definitive status

---

## Evolution

### Jan 4-6, 2026: Discovery Phase
- 8 investigations into "why agents show dead when they're done"
- Hypothesis: Completion verification broken
- Reality: Observation infrastructure gaps (no events, wrong metrics)

### Jan 8, 2026: Priority Cascade Model
- Investigation `2026-01-04-design-dashboard-agent-status-model.md` proposed cascade
- Tested against known failures, explained all observed behavior
- Decision: Implement Priority Cascade, fix event emission gaps

### Jan 12, 2026: Model Synthesis
- 11 investigations synthesized into this model
- Recognition: 89% completion rate, not 72% (measurement artifact)
- Principle emerged: Observation Infrastructure ("if system can't observe it, system can't manage it")

---

## References

**Key Investigations:**
- `2026-01-04-design-dashboard-agent-status-model.md` - Priority Cascade design
- `2026-01-08-inv-dashboard-shows-agents-dead-actually.md` - Discovery of 89% actual completion
- `2026-01-08-inv-orch-stats-completion-rate-miscalculated.md` - Metrics double-counting
- `2026-01-06-inv-bd-close-bypass-event-emission.md` - Beads hook gap
- ...and 7 others

**Decisions Informed by This Model:**
- `2026-01-08-observation-infrastructure-principle.md` - New principle from this work
- `2026-01-08-priority-cascade-model.md` - Status calculation approach

**Related Models:**
- (Future) `agent-lifecycle.md` - Broader lifecycle model
- (Future) `event-emission-architecture.md` - How events flow through system

**Primary Evidence (Verify These):**
- `pkg/status/calculate.go` - Status calculation implementation
- `pkg/dashboard/server.go` - Dashboard data serving
- `.beads/hooks/on_close` - Event emission on beads close
