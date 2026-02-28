# Probe: Flow-Integrated Knowledge Surfacing — Implementation Architecture

**Model:** Orchestrator Session Lifecycle
**Date:** 2026-02-27
**Status:** Complete

---

## Question

The orchestrator session lifecycle model claims session boundaries (start, completion) are the natural points for state transfer and comprehension. Does the current implementation provide sufficient integration points for knowledge surfacing at these moments, or are there structural gaps that require new commands/infrastructure?

Specifically testing:
1. Does the session start protocol have a programmatic hook for model surfacing, or is it purely skill-level guidance?
2. Does the completion review pipeline already surface model-level context, or only artifact-level context (quick entries, probe verdicts)?
3. Can `kb context` serve human-facing surfacing needs, or does it need adaptation?

---

## What I Tested

### Test 1: Session start integration points

Examined orchestrator skill session start protocol (lines 329-335):
```
1. Surface fires
2. Surface nagging
3. Backlog age check (HARD GATE)
4. Propose threads from bd ready + orch status
5. Confirm focus, run Proactive Hygiene Checkpoint
```

Checked for any programmatic command at session start:
```bash
grep -n "orient\|session.baseline\|session.start" cmd/orch/*.go
```

### Test 2: Completion review knowledge touchpoints

Read complete_cmd.go lines 1031-1067, knowledge_maintenance.go lines 199-313.

Current touchpoints in completion pipeline:
1. Probe verdicts surfacing (lines 1031-1039) — model-level
2. Architectural choices surfacing (lines 1041-1048) — work-level
3. Knowledge maintenance (lines 1050-1067) — quick entry promotion
4. Hotspot advisory (lines 1127-1133) — codebase risk

### Test 3: kb context output format

```bash
kb context --help
```

Current output: structured markdown with sections (Constraints, Decisions, Models, Guides, Investigations). Currently consumed by spawn context injection (pkg/spawn/kbcontext.go) with 80k char budget. Output is agent-facing (injected into SPAWN_CONTEXT.md), not human-facing.

### Test 4: Model freshness data availability

```bash
ls .kb/models/
```

22 models exist. Each has `Last Updated` date in model.md. Probes directory contains timestamped evidence. No programmatic freshness query exists — staleness detection is in `kb context --stale` flag.

### Test 5: Events.jsonl throughput data

```bash
orch stats --days 1
```

Stats command already parses events.jsonl: spawns (129), completions (98/76%), abandonments (34/26.4%), avg duration (44 min). Data exists but is formatted for CLI output, not session orientation.

---

## What I Observed

### Session Start: No programmatic hook exists

The session start protocol is **entirely skill-level guidance** — the orchestrator skill tells the orchestrator what to do (surface fires, check backlog), but there's no `orch orient` or `orch session-start` command. Steps 1-2 are conversational. Step 3 uses `orch backlog cull --dry-run`. Step 4 uses `bd ready` + `orch status`. No model surfacing happens.

**Gap:** No command exists that combines: relevant model summaries + throughput baseline + model freshness warnings. The orchestrator would need to manually run `kb context`, `orch stats`, and inspect model files — which violates the flow-integration principle (knowledge should surface through the flow, not through separate study).

### Completion Review: Artifact-level context exists, model-level impact is partial

The probe verdicts touchpoint (lines 1031-1039) does surface model-level impact — but only for probes created during that specific agent session. There's no cross-reference between SYNTHESIS.md topics and the broader model corpus.

The knowledge maintenance touchpoint (lines 1050-1067) surfaces quick entries by keyword relevance — good for promotion/obsoletion but doesn't answer "how does this completion change your model of X?"

**Gap:** No completion-time mechanism says "this agent's work confirms/contradicts/extends your model of [domain]." The probe verdicts are close but narrowly scoped to within-session probes only.

### kb context: Needs adaptation for human-facing use

Current kb context is designed for agent context injection:
- 80k char budget (way too large for human session start)
- Full model summaries included (need 2-3 sentence versions)
- No freshness scoring or drift warnings
- No throughput/baseline data

A human-facing variant would need: tighter budget (1-2k chars), summary-only model references, freshness annotations, and integration with throughput baseline.

### Trust calibration data: Partially available

Signals for trust tiers already exist in the system but aren't aggregated:
- Model freshness: `Last Updated` in model.md files
- Agent scope: Verification level (V0-V3) set at spawn
- Pattern history: `orch stats` has per-skill completion rates
- Model consistency: Probe verdicts have confirms/contradicts/extends

No mechanism combines these signals into a trust score.

---

## Model Impact

- [x] **Confirms** invariant: Session boundaries (start, completion) are the natural integration points for knowledge surfacing. The orchestrator skill's session protocol and the completion pipeline both have clear, extensible touchpoints.

- [x] **Extends** model with: The model correctly describes what happens at session boundaries but has a **surfacing gap**: knowledge exists in the system (models, probes, events, quick entries) but isn't composed into human-facing summaries at session start. Completion review has 4 touchpoints but none answer "how does this change your model of X?" across the full model corpus (only within-session probes).

- [x] **Extends** model with: The current architecture has a clear **layering opportunity**: Session start → `orch orient` command (new), Completion → model-impact enrichment (enhance existing touchpoint). Both are additive to the existing pipeline, not restructuring.

---

## Notes

This probe validates the design investigation's premise: the knowledge system serves agents (via `kb context` at spawn) but not the human directly. The infrastructure exists to close this gap without architectural change — the integration points are already there, they just need enrichment.

The key architectural recommendation: **Thread 1 (Model Surfacing at Engagement Moments) first**, split into two parallel tracks:
1. `orch orient` command for session start
2. Model-impact enrichment in completion review

Thread 3 (Calibrated Trust) is a natural follow-on because trust signals already exist — they just need aggregation and presentation at completion time.

Thread 2 (System Self-Awareness / Throughput) is partially solved by `orch stats` — needs format adaptation, not new data collection.

Thread 4 (Immersion Without Study) is the emergent property of Threads 1-3 working together.
