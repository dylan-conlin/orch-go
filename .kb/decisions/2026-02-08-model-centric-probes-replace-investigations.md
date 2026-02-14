# Decision: Model-Centric Probes Replace Investigations for Confirmatory Work

**Date:** 2026-02-08
**Status:** Accepted
**Context:** Models exist (29 total) but investigations remained the default artifact. 414 investigations vs 29 models — the energy was flowing to the wrong place.

## Decision

Shift the knowledge system's primary evolution pattern from **investigations** to **model-scoped probes**. Models become the living source of truth. Probes confirm, contradict, or extend model claims. Investigations remain only for novel explorations not yet attached to a model.

## Problem

The Jan 12 "Models as Understanding Artifacts" decision created `.kb/models/` but didn't change the default workflow. Agents continued producing full 300-line investigations even when a model already described the domain. This created:

1. **Redundant work** — agents re-exploring territory a model already covers
2. **Discovery failure** — 414 investigations are unsearchable noise; 29 models are the signal
3. **No feedback loop** — investigations don't update models, so models drift from reality
4. **Wasted agent time** — full investigation template for confirmatory work that needs 30-50 lines

## The Probe Pattern

**Probes** are lightweight (30-50 line) artifacts scoped against an existing model. They answer: "Does the model's claim still hold?"

### Probe Template (`.orch/templates/PROBE.md`)

Four mandatory sections:
- **Question** — What model claim are we testing?
- **What I Tested** — Command/code run (not code review — actual execution)
- **What I Observed** — Actual output
- **Model Impact** — Confirms/contradicts/extends which invariant

### Probe Directory Structure

Probes live under their model: `.kb/models/{model-name}/probes/`

This makes discovery structural, not search-dependent. Agent B browsing the SSE model's probes/ directory sees Agent A's recent work without keyword search.

### Decision Boundary: Probe vs Investigation

Made at spawn time based on whether `kb context` injects model content:

| Condition | Artifact | Why |
|-----------|----------|-----|
| Model exists for domain | **Probe** | Confirmatory work — test model claims |
| No model exists | **Investigation** | Novel exploration — build understanding |

### Merge Workflow

During `orch complete`, if probes exist:
1. Display probe summary with model impact
2. Prompt orchestrator: "Agent found X. Model says Y. Update model?"
3. Orchestrator (sole model writer) merges findings

This prevents concurrent merge conflicts and keeps model quality high.

## What Changed (Implementation)

Five commits landed 2026-02-08:

1. **Model content injection** — `kb context` now injects model Summary, Constraints, and Why This Fails sections into SPAWN_CONTEXT.md (not just title/path pointers)
2. **Probe template** — `.orch/templates/PROBE.md` (48 lines) with mandatory sections
3. **Probe directory structure** — `.kb/models/*/probes/` directories for all 25 models, with recent probes listed in spawn context
4. **Merge workflow** — `probe_merge.go` wired into `complete_gates.go` pipeline (advisory, non-blocking)
5. **Archive old investigations** — 353 completed investigations archived, reducing active from 764 to 411

## What This Answers (from Jan 12 Open Questions)

From the "Models as Understanding Artifacts" decision:

> 3. **How do we prevent model drift?** → Probes. Agents test model claims against reality. Contradictions surface during merge.

> 4. **When do investigations promote to models?** → When 3+ investigations cover the same domain. But now probes prevent the 3+ investigation accumulation — each probe feeds the model directly.

## Trade-offs Accepted

- **Agents need model context to produce probes** — requires model injection working correctly (implemented)
- **Orchestrator is sole model writer** — bottleneck by design, prevents quality degradation
- **Old investigations become read-only provenance** — not migrated or classified, just frozen in place
- **Probe template is intentionally rigid** — prevents agents from writing mini-investigations as probes

## Success Criteria

1. New model-scoped work produces probes (not investigations)
2. Investigation-to-model ratio shifts from 14:1 toward 5:1 or lower
3. Models stay current via merge workflow (no drift > 30 days)
4. Agents orient faster when model content is injected (qualitative)

## Origin

Design produced by a manual dialogue experiment: Dylan relayed messages between a Claude web session (fresh eyes, no tools) and the orchestrator (deep system knowledge). The web session identified the 414:29 ratio, proposed the 5-step plan, and produced implementation issues. All 5 steps were implemented within hours.

This decision also validates the "dialogue as design" pattern — a toolless critic produced a more coherent architecture than typical tool-equipped agents.

## Related Decisions

- `2026-01-12-models-as-understanding-artifacts.md` — Created models (this extends with evolution mechanism)
- `2026-01-07-strategic-orchestrator-model.md` — Orchestrators synthesize (probes feed synthesis)
- `2026-01-07-synthesis-is-strategic-orchestrator-work.md` — Sole model writer = orchestrator

## Remaining Gaps (Issues Needed)

1. **Orchestrator skill** — No mention of probes, model-scoped work, or probe template
2. **AGENTS.md** — No guidance for agents on when to produce probes vs investigations
3. **Investigation skill** — No mode flag or guidance for "model exists → use probe"
4. **CLAUDE.md knowledge placement** — Table doesn't include probes as artifact type
