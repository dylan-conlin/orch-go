# Decision: Observation Infrastructure Principle

**Date:** 2026-01-08
**Status:** Accepted
**Context:** Synthesis of 11 investigations from Jan 8, 2026 revealing systematic visibility gaps

## Decision

Adopt the principle: **"If the system can't observe it, the system can't manage it."**

Observation infrastructure (events, metrics, dashboard state) must be treated as load-bearing. Gaps in observation create false signals that erode trust and waste effort investigating "problems" that are actually measurement artifacts.

## Context

A single day's investigations (Jan 8, 2026) revealed a pattern: the system was performing better than it appeared, but observation gaps made it look broken.

| Investigation | Reality | What System Showed |
|--------------|---------|-------------------|
| 25-28% not completing | 89% actually complete | 72% reported |
| Dead agent detection | Agents die silently | Dashboard showed "active" |
| Stalled detection | Agents stuck 19+ min | No visibility |
| Stats deduplication | 272 unique completions | 298 events counted |
| Activity on load | Agents have activity | "Starting up..." forever |
| bd close tracking | Work completed | No events emitted |
| Epic children | Epic labeled ready | Children not spawned |

The meta-problem: **you can't trust your dashboard**, so you investigate, find metrics are wrong, fix them, repeat. This is a trust erosion loop.

## The Observation Gaps Identified

### Gap 1: Events Not Emitted
Work completed via `bd close` bypasses `orch complete` and emits no events. The system literally doesn't know work finished.

**Fix:** Beads `on_close` hook emits `agent.completed` events (implemented Jan 8).

### Gap 2: Events Double-Counted
Stats counted completion EVENTS, not unique completions. Same issue completing multiple times inflated counts.

**Fix:** Stats deduplication by beads_id (implemented Jan 8).

### Gap 3: State Not Surfaced
Dead agents (no heartbeat for 3+ min) appeared as "active". Stalled agents (same phase 15+ min) had no indicator.

**Fix:** Dead detection restored (Jan 8). Stalled detection designed, pending implementation.

### Gap 4: Progress Signals Missing
Dashboard showed "Starting up..." on initial load because API returned activity as string, frontend expected object.

**Fix:** Frontend transformation of API response (Jan 8).

### Gap 5: Inheritance Not Inferred
Labeling an epic `triage:ready` didn't cascade to children. Daemon saw epic as "not spawnable" and children as "missing label".

**Fix:** Daemon infers children from epic's label (designed, pending implementation).

## The Principle

> **"If the system can't observe it, the system can't manage it."**

This applies at multiple levels:

| Level | What to Observe | Gap Example |
|-------|-----------------|-------------|
| **Agent lifecycle** | Spawn, progress, completion, death | Missing completion events |
| **Work state** | Phase, activity, staleness | No stalled indicator |
| **Metrics** | Unique completions, durations | Double-counting |
| **Relationships** | Epic → children, blockers | No label inheritance |

## Implementation Principles

### 1. Every State Transition Should Emit
If an agent can go from state A to state B, there should be an event. If there's a path that skips the event, that's a bug.

### 2. Dashboards Should Be Single Source of Truth
If you have to run CLI commands or grep logs to understand system state, the dashboard has failed. Add the visibility.

### 3. Metrics Should Be Deduplicated by Entity
Count unique entities (agents, issues), not events. Multiple events per entity should be handled.

### 4. Default to Visible, Not Hidden
When uncertain whether to surface something, surface it. False positives (showing something that's fine) are better than false negatives (hiding something broken).

### 5. Observation Gaps Are P1 Bugs
Treat missing visibility as a high-priority bug, not a nice-to-have. Invisible failures erode trust faster than visible ones.

## What This Changes

### Before
- "Stats show 72% completion rate, something is broken"
- Investigate → find metrics bug → fix → repeat
- Dashboard shows stale/wrong state → lose trust → stop using dashboard

### After
- Stats show accurate rate (deduplicated)
- All completion paths emit events
- Dead/stalled agents surface immediately
- Dashboard is trusted single source of truth

## Relationship to Other Decisions

- **Strategic Orchestrator Model** (2026-01-07): Orchestrator needs accurate observation to comprehend system state
- **Load-Bearing Guidance** (2026-01-08): Observation infrastructure is load-bearing (remove it and system breaks)
- **Pressure Over Compensation** (principles): Observation gaps create pressure to fix system, not work around it

## Outstanding Work

| Fix | Status | Issue |
|-----|--------|-------|
| Stats deduplication | Done | - |
| Dead agent detection | Done | - |
| bd close event emission | Done | - |
| Activity on initial load | Done | - |
| Stalled agent detection | Designed | Pending impl |
| Epic child inference | Designed | Pending impl |
| Spawn validation (invalid beads_id) | Done | - |

## Origin

Synthesis session with Dylan, 2026-01-08. After reviewing 11 investigations from the same day, the pattern became clear: most "failures" were observation failures, not system failures. The investigations were individually tactical, but together they revealed a strategic truth about observation infrastructure.
