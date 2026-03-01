# Probe: Session Debrief Artifact Design

**Date:** 2026-02-28
**Status:** Complete
**Model:** orchestrator-session-lifecycle
**Trigger:** Design task for durable session debrief artifacts that orch orient reads for cross-session comprehension continuity

## Question

The orchestrator session lifecycle model defines session boundaries with debrief protocols (SESSION_HANDOFF.md, reflection sessions), but comprehension generated during debriefs evaporates between sessions. Can a durable debrief artifact at `.kb/sessions/YYYY-MM-DD-debrief.md` integrate with `orch orient` to provide comprehension continuity without duplicating existing handoff/synthesis mechanisms?

**Model claims being tested:**
1. Session boundaries are well-defined with three patterns (worker, orchestrator, cross-session)
2. SESSION_HANDOFF.md is the primary cross-session continuity mechanism
3. Orchestrator session end debrief is "the RECONNECT phase applied to the whole session"
4. `orch orient` surfaces facts (throughput, ready work, models) but not comprehension

## What I Tested

### 1. Current orient output structure
Read `cmd/orch/orient_cmd.go` (288 lines) and `pkg/orient/orient.go` (190 lines).

**Current data sources:**
- Throughput metrics from `~/.orch/events.jsonl`
- Ready issues from `bd ready` (top 3)
- KB context enrichment per issue via `kb context`
- Model freshness from `.kb/models/` (top 3 relevant, top 2 stale)
- Focus goal from `~/.orch/focus.json`
- In-progress count from `bd list`

**OrientationData struct fields:** Throughput, ReadyIssues, RelevantModels, StaleModels, FocusGoal

**Observation:** Orient is purely factual — it answers "what is the state?" but not "why are we here?" or "what changed yesterday?" There's no comprehension layer.

### 2. Current session end debrief protocol (orchestrator skill)
Read orchestrator skill lines 365-383.

**Debrief sequence:**
1. What happened (threads worked — completions, spawns, decisions)
2. What changed (durable changes — constraints, decisions, model updates)
3. What's in flight (active agents, pending review, open questions)
4. What's next (1-3 proposed threads for next session)
5. Hygiene (checkpoint, commit, sync)

**Observation:** This is conversational — produced in chat, evaporates when session closes. The orchestrator skill defines WHAT the debrief contains but not WHERE it persists. SESSION_HANDOFF.md is the orchestrator-tier persistence mechanism, but interactive orchestrator sessions (which are most sessions) produce debriefs conversationally with no durable artifact.

### 3. SESSION_HANDOFF.md vs proposed debrief artifact
SESSION_HANDOFF.md is written by spawned orchestrator agents at `.orch/workspace/{name}/`. Interactive sessions write to `.orch/session/{window-name}/{timestamp}/`.

**Gap:** Interactive sessions (Dylan + orchestrator in Claude Code) don't consistently produce SESSION_HANDOFF.md. The debrief happens conversationally and dies with the session.

### 4. MEMORY.md role
Auto-memory at `~/.claude/projects/.../memory/MEMORY.md` captures tactical session knowledge with a 200-line cap. Currently 30 lines: build commands, parallel agent gotchas, refactoring patterns.

**Observation:** MEMORY.md is agent-scoped (helps the next Claude session), not human-facing. It captures "how to work in this codebase" not "what happened today."

## What I Observed

**Key finding:** There's a clear gap between three things:
1. **Facts** (what `orch orient` provides — throughput, ready work, model state)
2. **Tactical memory** (what MEMORY.md provides — how to build, known gotchas)
3. **Comprehension** (what the debrief produces but doesn't persist — why work matters, how threads connect, what changed about constraints)

The debrief artifact fills gap #3: durable comprehension that survives across sessions.

**Confirms model claim:** Session boundaries are well-defined, but the debrief output at those boundaries is conversational-only for interactive sessions.

**Extends model:** SESSION_HANDOFF.md works for spawned orchestrators but interactive sessions (the majority pattern) have no debrief persistence. `.kb/sessions/` fills this gap specifically for interactive orchestrator sessions.

## Model Impact

- **Confirms** invariant: Session boundaries are well-defined with distinct patterns
- **Extends** model: Adds a fourth artifact type (debrief) alongside SYNTHESIS.md, SESSION_HANDOFF.md, and MEMORY.md
- **Clarifies** gap: Interactive orchestrator sessions have no durable comprehension artifact — debrief evaporates
- **Extends** `orch orient`: Natural extension point for comprehension layer alongside existing fact layer
