---
stability: foundational
---
# Decision: Synthesis is Strategic Orchestrator Work, Not Spawnable Work

**Date:** 2026-01-07
**Status:** Accepted
**Context:** Recurring duplicate synthesis issues causing backlog noise

## Decision

Synthesis (combining findings from multiple investigations into coherent understanding) is **strategic orchestrator work**, not work to be delegated to spawned agents.

Auto-creation of synthesis issues is disabled. Reflection should surface opportunities, not create work.

## Context

The daemon was running `kb reflect --type synthesis --create-issue` hourly, auto-creating beads issues like "Synthesize dashboard investigations (50)". This led to:

- 95+ duplicate synthesis issues created
- 44 of 50 open issues were synthesis issues at one point
- The mechanism for consolidation was creating more fragmentation
- Deduplication bugs (fail-open error handling) caused hourly duplicates

The irony: **the goal was consolidation, but the mechanism created the opposite**.

## The Deeper Problem

Even without the deduplication bug, auto-creating synthesis issues was wrong because:

1. **Synthesis requires strategic judgment** - Deciding what to synthesize, when, and how is orchestrator work
2. **Issues without capacity create debt** - Creating work faster than it can be processed adds noise, not value
3. **Synthesis is about understanding, not tasks** - You can't spawn "understand this topic" - understanding happens through engagement

## The Model

| Activity | Who Does It | Why |
|----------|-------------|-----|
| Investigation | Worker agent | Discovers facts, explores code |
| Synthesis | Strategic orchestrator | Combines findings into coherent model |
| Implementation | Worker agent | Writes code, fixes bugs |

Workers produce knowledge atoms. Strategic orchestrator composes knowledge into models.

## What Changed

1. **Daemon config:** Added `--reflect-issues=false` to launchd plist
2. **Backlog:** Closed all synthesis issues with reason explaining the decision
3. **Role clarity:** Synthesis is now explicitly orchestrator work

## What Reflection Should Do Instead

Reflection should **surface** opportunities, not **create work**:

- Show synthesis candidates in `orch status` or dashboard
- Alert orchestrator to investigation clusters
- Let orchestrator decide when to engage

The orchestrator sees "50 dashboard investigations" and decides: "I need to understand the dashboard before more work" - then engages directly, not by spawning.

## Relationship to Strategic Orchestrator Model

This decision is part of a broader shift:

- Orchestrator's job is **comprehension**, not just coordination
- Coordination is automated (daemon handles spawning)
- Understanding happens through engagement, not delegation

See: `2026-01-07-strategic-orchestrator-model.md`

## Origin

Session with Dylan, 2026-01-07. Started with "duplicate synthesis issues" as symptom, traced to design question about what synthesis actually is and who should do it.
