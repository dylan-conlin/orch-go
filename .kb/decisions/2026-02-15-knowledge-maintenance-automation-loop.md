# Decision: Knowledge Maintenance Automation Loop

**Date:** 2026-02-15
**Status:** Proposed
**Context:** `kb reflect` runs manually today. Model staleness detection works at spawn-time (warnings in SPAWN_CONTEXT) but nothing triggers model updates. The remediation path from detection to update is missing. Prior work designed individual pieces (Jan 6 two-tier automation, Feb 14 detect-annotate-queue for models). This decision designs the complete closed loop.
**Extends:**
- `.kb/decisions/2026-02-14-model-staleness-detection.md` (adds remediation path)
- `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md` (throttles automation to verification bandwidth)
- `.kb/investigations/2026-01-06-inv-automated-reflection-daemon-kb-reflect.md` (implements two-tier automation recommendation)

## Decision

Implement a **three-layer automation loop** (Detection → Queueing → Remediation) for knowledge maintenance, throttled to human verification bandwidth.

### The Loop

```
┌──────────────────────────────────────────────────────────┐
│ LAYER 1: DETECTION (automated, always-on)                │
│                                                          │
│ A. Spawn-time model staleness (existing, working)        │
│    → Annotates SPAWN_CONTEXT.md with warnings            │
│    → Records staleness event to staleness-events.jsonl   │
│                                                          │
│ B. Daemon periodic reflection (hourly):                  │
│    → synthesis: issue creation ≥10 investigations        │
│    → open: issue creation >3 days without action         │
│    → model-drift: issue creation ≥3 stale spawns         │
│    → stale/drift/promote/refine: surface-only            │
│                                                          │
│ C. Completion-time reverse linkage (new):                │
│    → When agent modifies files referenced by models,     │
│      add note to completion output about affected models │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│ LAYER 2: QUEUEING (throttled to verification bandwidth)  │
│                                                          │
│ A. Deduplication: one issue per stale model              │
│    → Key: model file path                                │
│    → Check existing open issues before creating          │
│                                                          │
│ B. Backpressure: max 3 open model-update issues          │
│    → When 3+ unverified model-update issues exist,       │
│      stop creating new ones                              │
│    → Staleness still annotated at spawn (detection       │
│      continues, queueing pauses)                         │
│                                                          │
│ C. Batching: related models grouped                      │
│    → When 3+ models in same domain are stale,            │
│      create one "batch update" issue                     │
│    → Domain = model directory (e.g., completion-*)       │
│                                                          │
│ D. Priority mapping:                                     │
│    → Deleted files: P2 (model references nonexistent)    │
│    → Major changes (≥5 commits): P2                      │
│    → Minor changes (1-4 commits): P3                     │
│    → Tier 2 verification (explain-back only, no          │
│      behavioral gate for model updates)                  │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────┐
│ LAYER 3: REMEDIATION (human-in-loop, orchestrator-owned) │
│                                                          │
│ A. Primary path: Orchestrator updates during reflection  │
│    → Reflection session picks "model-drift" lane         │
│    → Orchestrator reads current code, updates model      │
│    → Closes issue via orch complete --explain "..."      │
│                                                          │
│ B. Delegation path: Architect worker for severe drift    │
│    → When >50% of code_refs are stale/deleted            │
│    → Orchestrator spawns architect to draft update        │
│    → Orchestrator reviews and verifies update             │
│                                                          │
│ C. Opportunistic path: Agent-initiated corrections       │
│    → Agent sees staleness warning in SPAWN_CONTEXT       │
│    → Agent verifies claims against current code           │
│    → Agent includes correction in SYNTHESIS.md            │
│    → Orchestrator applies correction during completion    │
└──────────────────────────────────────────────────────────┘
```

## Problem

Three gaps in the current system:

1. **Missing remediation path.** Spawn-time staleness detection annotates stale models but creates no mechanism to update them. Agents see warnings, nothing happens next.

2. **Incomplete daemon automation.** Daemon only runs `synthesis` type with issue creation. `open` and `model-drift` types should also auto-create issues per Jan 6 investigation recommendation.

3. **No throttling against verification bandwidth.** If automation creates issues for every stale model on every scan, the backlog grows faster than Dylan can verify. Verifiability-first decision (Feb 14) says: "system cannot change faster than human can verify behavior."

## Design Decisions (Forks Navigated)

### Fork 1: What triggers model update issues?

**Decision:** Spawn-count threshold (3 stale spawns for same model).

**Why:** Measures actual impact. A stale model that's never served doesn't need urgent update. Three spawns means three agents received outdated understanding.

**Alternatives rejected:**
- Time-based only (7 days stale): Doesn't measure impact. Some models are rarely served.
- Combined (time OR count): Over-creates issues for rarely-served models.

**Implementation:** Record staleness events to `~/.orch/model-staleness-events.jsonl` at spawn time. Daemon counts events per model when running model-drift reflection.

### Fork 2: Where does the model update work happen?

**Decision:** Primarily orchestrator direct update during reflection sessions. Architect worker delegation for severe drift only.

**Substrate:**
- Decision: "Synthesis is strategic orchestrator work" → models are orchestrator understanding
- Principle: Verification Bottleneck → human must verify model accuracy
- Investigation (Feb 14): "annotate + queue, not auto-fix"

**Why not auto-spawn update agents for every stale model:** Creates review obligations faster than can be verified. Each model update needs orchestrator review (explain-back gate). Spawning 12 model-update workers simultaneously violates verifiability-first.

**Severity threshold for delegation:** >50% of a model's code_refs are stale or deleted. This indicates the model's core mechanism understanding may be wrong, not just file paths outdated.

### Fork 3: How to throttle issue creation?

**Decision:** Backpressure + batching.

**Backpressure rule:** When 3+ open model-update issues exist (label: `model-maintenance`), stop creating new ones. Detection continues (staleness still annotated at spawn time). Issue creation resumes when queue drains below threshold.

**Batching rule:** When 3+ models in the same directory/domain are stale, create one grouped issue instead of three individual ones. Example: "Update completion-verification model family (3 stale models)" instead of 3 separate issues.

**Why 3:** Matches verification cadence. Dylan can realistically review 1-3 model updates per reflection session. Creating more queues work that won't be processed.

### Fork 4: Cadence for each reflection type

| Type | Cadence | Auto-Issue | Threshold | Rationale |
|------|---------|------------|-----------|-----------|
| **synthesis** | Hourly | Yes | ≥10 investigations | Proven valuable (existing) |
| **open** | Hourly | Yes | Any item >3 days | Self-declared actionable (Jan 6 rec) |
| **model-drift** | Every 4 hours | Yes (throttled) | ≥3 stale spawns | New: closes remediation gap |
| **stale** (decisions) | Hourly | Surface only | N/A | Medium signal, needs judgment |
| **drift** (constraints) | Hourly | Surface only | N/A | ~30-50% false positive rate |
| **promote** | Hourly | Surface only | N/A | Requires human judgment |
| **refine** | Daily | Surface only | N/A | Low urgency |
| **skill-candidate** | Daily | Surface only | N/A | Noisy clustering |

**Why model-drift at 4 hours instead of hourly:** Prevents thrashing. If spawns happen in bursts (3 spawns in 10 minutes hit same stale model), hourly check would immediately create issue. 4-hour cadence gives time for the count to accumulate meaningfully.

### Fork 5: Staleness event recording mechanism

**Decision:** Dedicated `~/.orch/model-staleness-events.jsonl` file.

**Format:**
```jsonl
{"timestamp":"2026-02-15T10:30:00Z","model":".kb/models/completion-verification.md","changed_files":["cmd/orch/complete_cmd.go"],"deleted_files":["pkg/verify/phase.go"],"spawn_id":"og-feat-xyz","agent_skill":"feature-impl"}
```

**Why not extend gap-tracker:** Semantically different. Context gaps = missing knowledge. Model staleness = outdated knowledge. Different detection mechanisms, different remediation paths.

**Retention:** 30-day window, same as gap-tracker. Events marked "resolved" when model is updated (Last Updated date refreshed).

## Reflection Session Integration

The reflection sessions guide (`.kb/guides/reflection-sessions.md`) already defines lanes. This decision adds:

### New Lane: model-drift

**When to pick this lane:**
- `kb reflect --type model-drift` reports models with stale code references
- ATS score: typically Recurrence 2-3, Delay 2, Radius 1-2, Preventability 2, Noise 0 = **ATS 7-9** (high priority)
- Models referenced by recently-active spawns get higher priority

**Session workflow:**
1. Run `kb reflect --type model-drift` (or read daemon output)
2. For each stale model, check: is the model's core mechanism still accurate?
3. If yes (just file renames/moves): update code_refs and Last Updated → quick fix
4. If no (mechanism changed): spawn architect worker to draft new model → review
5. Close model-update beads issues via `orch complete --explain "..."`

**Integration with other lanes:**
- model-drift can be the maintenance lane (20-30%) alongside a primary synthesis lane
- If model-drift ATS > 7, it becomes the primary lane

## Completion-Time Reverse Linkage (New Mechanism)

When `orch complete` runs, it already checks git diff. Add:

1. Read list of files modified in agent's commits
2. Cross-reference against all model code_refs
3. If modified files appear in model code_refs, output:
   ```
   NOTE: Modified files referenced by models:
   - cmd/orch/complete_cmd.go → .kb/models/completion-verification.md
   Consider updating affected models.
   ```

This is **informational only** (not a gate). It surfaces the linkage at the natural moment (completion review) without blocking.

**Why not a gate:** Model updates require orchestrator synthesis, not worker action. Making it a gate would block workers who modified code legitimately.

## Verifiability-First Compliance

| Mechanism | Verifiability Compliance |
|-----------|------------------------|
| Spawn-time annotation | Informational, doesn't create work |
| Issue creation (synthesis, open) | Creates work, but high-signal + proven |
| Issue creation (model-drift) | Throttled: backpressure (max 3), batching, 4h cadence |
| Remediation (orchestrator) | Human-in-loop, explain-back gate |
| Remediation (architect worker) | Only for severe drift, orchestrator reviews |
| Completion reverse linkage | Informational, doesn't create work |

**The critical constraint:** `C × V ≤ 1` where C = rate of issue creation, V = time to verify. With max 3 open issues and 4-hour model-drift cadence, worst case: 3 new model issues per 4 hours. Dylan's verification session can handle 1-3 per session. If sessions happen daily, queue stays bounded.

**Circuit breaker:** If model-update issues accumulate >5 open, daemon logs warning and halts model-drift issue creation until queue drains. This prevents the runaway automation that caused the entropy spirals.

## Implementation Plan

### Phase 1: Staleness Event Recording (orch-go)

**Scope:** Record spawn-time staleness events for daemon consumption.

1. Create `pkg/spawn/staleness_events.go` with event recording
2. Wire into `kbcontext.go` staleness detection path
3. Write to `~/.orch/model-staleness-events.jsonl`
4. Include: model path, changed files, deleted files, spawn ID, timestamp

**Acceptance:** Staleness events recorded when stale models served. `cat ~/.orch/model-staleness-events.jsonl` shows events after spawn.

### Phase 2: Daemon Model-Drift Reflection (orch-go)

**Scope:** Daemon reads staleness events and creates issues.

1. Add `ReflectModelDriftEnabled` and `ReflectModelDriftInterval` to daemon Config
2. Add `RunModelDriftReflection()` that reads staleness events, counts per model
3. Create beads issues when spawn count ≥ 3 (with deduplication, backpressure)
4. Label: `model-maintenance`, priority P2/P3 based on severity

**Acceptance:** Daemon creates model-update issues after 3+ stale spawns for same model.

### Phase 3: Daemon Open Reflection (orch-go + kb-cli)

**Scope:** Add `open` type auto-issue creation to daemon.

1. If not already in kb-cli: add `kb reflect --type open --create-issue` support
2. Add `ReflectOpenEnabled` config to daemon
3. Wire into daemon reflection cycle alongside synthesis

**Acceptance:** Daemon creates issues for investigations >3 days without action.

### Phase 4: Completion Reverse Linkage (orch-go)

**Scope:** Inform orchestrator about model impact at completion time.

1. In `orch complete`, after git diff analysis, cross-reference modified files against model code_refs
2. Output informational note about affected models
3. No gate, no blocking

**Acceptance:** `orch complete` shows which models reference modified files.

### Phase 5: Reflection Session Guide Update

**Scope:** Update `.kb/guides/reflection-sessions.md` with model-drift lane.

1. Add model-drift as available primary lane
2. Document ATS scoring for model-drift candidates
3. Add session workflow for model updates

**Acceptance:** Guide includes model-drift lane with clear workflow.

## Principles Applied

| Principle | How Applied |
|-----------|------------|
| **Verification Bottleneck** | Backpressure (max 3 issues), 4h cadence, batching |
| **Infrastructure Over Instruction** | Automated detection + issue creation, not "remember to check" |
| **Surfacing Over Browsing** | Staleness surfaced at spawn time (existing) + completion time (new) |
| **Gate Over Remind** | Staleness annotation = soft gate (agent sees warning in context) |
| **Capture at Context** | Detection happens when model consumed (spawn) and when code changes (completion) |
| **Session Amnesia** | Issues persist across sessions; staleness events survive session boundaries |

## Success Criteria

**Short-term (2 weeks):**
- [ ] Staleness events recorded at spawn time
- [ ] Daemon creates model-update issues after threshold
- [ ] Backpressure prevents issue flood

**Medium-term (1 month):**
- [ ] Model staleness rate decreasing (currently ~50% of models)
- [ ] Reflection sessions include model-drift lane
- [ ] Completion reverse linkage operational

**Long-term (3 months):**
- [ ] Model staleness rate < 20%
- [ ] Remediation loop self-sustaining (detection → issue → update → close)
- [ ] No model-update issue backlog > 5

## Open Questions

1. **Should model-drift issues block daemon spawning?** Currently proposed as non-blocking (model updates are P2/P3). If stale models cause agent failures, consider promoting to P1 with spawn blocking.

2. **How to handle models that are perpetually stale?** Some models may reference rapidly-changing files. Consider: if same model triggers >5 issues in 30 days, the model's code_refs are too granular. Surface to orchestrator for code_ref refinement.

3. **Cross-repo staleness events?** Models may reference files in other repos. Deferred — current models are primarily intra-repo. Flag for Phase 6 if needed.

## Related Decisions

- `.kb/decisions/2026-02-14-model-staleness-detection.md` — Detect-Annotate-Queue design (this extends with queueing + remediation)
- `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md` — Hard operating constraint this respects
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` — Why models are orchestrator work
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Models as first-class artifacts

## Investigation

- `.kb/investigations/2026-01-06-inv-automated-reflection-daemon-kb-reflect.md` — Two-tier automation design (this implements)
- `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md` — Staleness detection design
- `.kb/guides/reflection-sessions.md` — Reflection session protocol (this extends)
