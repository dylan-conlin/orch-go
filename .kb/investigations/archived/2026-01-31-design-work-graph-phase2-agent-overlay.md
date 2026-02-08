# Design: Work Graph Phase 2 - Agent Overlay

**Date:** 2026-01-31
**Status:** Complete
**Owner:** Dylan + Claude
**Parent Issue:** orch-go-21154
**Parent Epic:** orch-go-21121

---

## Summary (D.E.K.N.)

**Delta:** Agent activity should appear as a dynamic overlay on the static work graph, with a "Work in Progress" pinned section showing the full pipeline (running + queued). Issue lifecycle - not just agent lifecycle - is the key observable, with deliverables as a completion checklist.

**Evidence:** Design session walked through visual placement options, information hierarchy, daemon integration, and deliverables tracking. Connected to prior investigations on semantic metadata encoding and disabled gate patterns.

**Knowledge:** The issue is the anchor. Agents come and go. Artifacts accumulate. The lifecycle is the story of how the issue got (or didn't get) resolved. Deliverables should use hybrid approach: schema defines expectations, overrides are logged for later analysis.

**Next:** Implementation of Phase 2 features.

---

## Design Decisions

### 1. Visual Placement: Pinned Section

**Decision:** Active work floats to a pinned "Work in Progress" section at top, rather than re-sorting the main tree.

**Rationale:**
- Creates natural "active zone" - you know where to look
- Maintains spatial stability in main tree (positions don't shift)
- Same issue appears in both places (pinned + tree position with indicator)

**Rejected alternatives:**
- Inline indicator only (limited info at glance)
- Agent column (wastes space, empty for most rows)
- Re-sorting tree by activity (breaks spatial memory)

---

### 2. Information Hierarchy

| Level | What | Content |
|-------|------|---------|
| **L0** | Row | ID (truncated), title, status indicator, expressive status, health indicator (if warning/critical) |
| **L1** | Auto-expanded for running | Attempt #, phase, context %, deliverables checklist (compact: ✓ ✓ ○ ○) |
| **L2** | Side panel (click) | Full issue description, deliverables detail with artifact links, attempt history, lifecycle events |

**Key principle:** L1 auto-expands for running agents. The running section is your focus area - don't make users click to see details.

---

### 3. Work in Progress Section Structure

```
┌─────────────────────────────────────────────────────────────┐
│ WORK IN PROGRESS                                            │
│                                                             │
│ RUNNING (n)                                     [at capacity]│
│  ▶ ...21150  Fix highlight regression     Running Bash...   │
│    │ Attempt: 1 · Phase: Implementation · Context: 45%      │
│    │ Deliverables: ✓ ✓ ○ ○                                  │
│    │                                                        │
│  🚨 ...21160  Implement login              Stuck (3m)        │
│    │ Attempt: 3 · Phase: Implementation · Context: 89% ⚠️   │
│    │ Deliverables: ✓ ✓ ✗ ○                                  │
│    │ Prior: died (context), closed→reopened (verify failed) │
│                                                             │
│ QUEUED (n)                                                  │
│  ◷ ...21155  Design: Artifact Feed        next              │
│  ◷ ...21156  Implement overlay            blocked by 21154  │
│  ◷ ...21163  Update docs                  waiting for slot  │
└─────────────────────────────────────────────────────────────┘
```

---

### 4. Health Indicators

| State | Indicator | Trigger |
|-------|-----------|---------|
| Healthy | (none) | Normal operation |
| Warning | ⚠️ | Long think time (>30s), high context (>80%), no commits (>30m), tool failures (3+) |
| Critical | 🚨 | Stuck (no activity 3m+), tool failure loop (5+), context exhausted (>95%) |

Health indicators appear inline on the row, with details in L1 expansion.

---

### 5. Daemon Integration

**Decision:** Daemon status integrated into Work in Progress section, not separate.

Shows the full pipeline:
- **Running** - agents actively working
- **Queued** - daemon has these ready, will spawn when capacity available

Queue reasons displayed:
- `next` - top of queue
- `waiting for slot` - at max capacity
- `blocked by {id}` - dependency
- `⏸ paused` - daemon paused

Capacity indicator in header: `at capacity` when at max agents.

---

### 6. Issue Lifecycle as First-Class Observable

**Key insight:** The issue is the anchor. Agents come and go. Artifacts accumulate. The lifecycle is the story.

**Observable lifecycle patterns:**

| Pattern | What happened | Visual signal |
|---------|---------------|---------------|
| Clean completion | One agent, verified, done | Attempt: 1, all ✓ |
| Retry | Agent failed/died, new agent spawned | Attempt: 2, 3, ... |
| Resurrection | Closed, but reopened (verification failed) | "closed→reopened" in history |
| Stuck | Multiple attempts, no progress | Attempt 3+ ⚠️, same failure |
| Escalation | Agent couldn't solve, needs human | Escalated badge |

**L1 shows:** Attempt number, prior attempt outcomes (compact)
**L2 shows:** Full attempt history with timestamps, outcomes, artifacts produced

---

### 7. Deliverables Checklist

**Decision:** Hybrid approach - schema defines expected deliverables per issue type, overrides logged.

**Schema per type (examples):**

| Type + Skill | Expected Deliverables |
|--------------|----------------------|
| bug + feature-impl | Code committed, Tests pass, Visual verification (if UI), SYNTHESIS.md |
| task + feature-impl | Code committed, Tests pass, SYNTHESIS.md |
| investigation | Investigation artifact, Recommendation |
| design-session | Design brief, Mockups (if UI), Decision or Epic created |
| architect | Investigation with recommendation, Decision record |

**Display:**
- L1: Compact checklist (✓ ✓ ○ ○)
- L2: Full checklist with labels, artifact links, override reasons

**Override flow:**
```
DELIVERABLES
  ✓ Code committed
  ✓ Tests passing
  ○ Visual verification                         ← expected
  ○ SYNTHESIS.md                                ← expected

[Close anyway]  [Add missing deliverables]

If closing with gaps, prompted for reason per missing item.
Reason logged for later analysis.
```

**Rationale:** Gates that are too rigid get disabled (see: disabled gate investigation). Override-with-logging lets us:
1. Surface gaps visibly
2. Not hard-block edge cases
3. Collect data on override patterns
4. Calibrate schemas based on real usage

---

### 8. Click Behavior

- **Click running/queued row** → Opens side panel with full L2 details
- **Keyboard nav** → j/k moves selection, l/Enter opens side panel, h/Esc closes
- **L1 details** → Auto-expanded for running, collapsed for queued

---

## Out of Scope (Deferred to Phase 3: Artifact Feed)

- Recently completed work display
- Browsing artifacts (investigations, decisions, SYNTHESIS.md)
- Review queue / needs attention for artifacts
- Artifact-centric view (vs issue-centric)

---

## Implementation Notes

### Data Requirements

- Correlate agent sessions with beads IDs (already exists)
- SSE for real-time updates (already exists)
- Daemon queue state (already exists via daemon store)
- Attempt history per issue (may need to add)
- Deliverables schema per issue type (new)
- Override logging (new)

### Components to Build

1. `WorkInProgressSection` - pinned section with running + queued
2. `AgentRow` - row with L0/L1 content, health indicators
3. `DeliverableChecklist` - compact and full views
4. `IssueSidePanel` - L2 details with lifecycle, history, artifacts
5. `AttemptHistory` - timeline of attempts with outcomes

### API Changes

- Add attempt history to issue data
- Add deliverables schema lookup
- Add override logging endpoint

---

## Related Artifacts

- Prior investigation: `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md`
- Semantic metadata encoding: `.kb/investigations/2026-01-31-inv-investigate-semantic-metadata-encoding-points.md`
- Disabled gates: `.kb/investigations/2026-01-31-inv-investigate-disabled-gate-failure-patterns.md`
- Phase 1 implementation: `orch-go-21122`
- Phase 2 issue: `orch-go-21154`
- Phase 3 design: `orch-go-21155`
