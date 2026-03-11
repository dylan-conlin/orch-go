## Summary (D.E.K.N.)

**Delta:** The orch-go event system captures end-states well (completions, abandonments) but is blind to decision points — gates that allow/block, daemon routing, and advisory outcomes are silent, making it impossible to measure whether structural enforcement improves agent quality.

**Evidence:** events.jsonl analysis: 4,411 events, 19 types. Of 363 agent.completed events, 52% lack skill/outcome fields. Only 1/363 has pipeline_timing. Only 17 have accretion.delta. Hotspot gate logs bypasses (60) but not blocks or allows. Architect escalation: 0 events. Pre-commit gates: 0 events.

**Knowledge:** The system has a "survivorship bias" architecture: it measures what happens after enforcement, not the enforcement decisions themselves. You can't calculate gate accuracy (false positive rate) because only overrides are logged, not decisions.

**Next:** Implement 7 prioritized measurement gaps as beads issues, starting with agent.completed field backfill and spawn.gate_decision events.

**Authority:** architectural — Cross-cutting instrumentation changes affect event schema, stats aggregation, and multiple command paths.

---

# Investigation: Measurement Gap Audit for Structural Enforcement

**Question:** Where should orch-go be instrumenting but isn't? What data would prove or disprove that structural enforcement (harness gates, accretion limits, duplication detection, hotspot routing) improves agent quality?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None — implementation issues to be created
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/entropy-spiral/probes/2026-03-01-probe-fix-feat-ratio-gate-effectiveness.md | extends | yes | - |
| .kb/models/entropy-spiral/probes/2026-03-01-probe-self-stabilization-current-gates.md | extends | yes | - |
| .kb/models/kb-reflect-cluster-hygiene/probes/2026-02-26-probe-defect-class-pipeline-gap.md | extends | yes | - |
| .kb/models/completion-verification/probes/2026-03-01-probe-hotspot-blind-spot-analysis.md | extends | yes | - |

---

## Findings

### Finding 1: agent.completed events have severe field coverage gaps

**Evidence:** Of 363 agent.completed events in events.jsonl:
- 159/363 (43.8%) have `skill` field
- 173/363 (47.7%) have `outcome` field
- 165/363 (45.5%) have `duration_seconds` field
- 178/363 (49.0%) have `verification_passed` field
- 1/363 (0.3%) have `pipeline_timing` field
- 190/363 (52.3%) have no skill, outcome, or duration at all

Root cause: the `AgentCompletedData` struct was enriched over time (skill, outcome, tokens, pipeline_timing) but older completion paths (daemon auto-complete, bd close hook, review_done reconciliation) emit raw events without these fields.

**Source:** `~/.orch/events.jsonl` analysis; `pkg/events/logger.go:227-244` (struct definition); `cmd/orch/emit_cmd.go:103` (manual emit path); `cmd/orch/review_done.go:297` (reconciliation path)

**Significance:** Cannot slice quality metrics (rework rate, verification pass rate, duration) by skill type for 56% of completions. This is the single biggest data quality issue — it undermines every downstream analysis.

---

### Finding 2: Gate decisions are invisible — only overrides are logged

**Evidence:** The event system logs when enforcement is BYPASSED but not when enforcement DECIDES:

| Gate | Bypass logged? | Block logged? | Allow logged? |
|------|---------------|--------------|--------------|
| Hotspot spawn gate | Yes (60 events) | No | No |
| Triage gate | Yes (52 events) | No | No |
| Verification gate | Yes (20 events) | No | No |
| Pre-commit accretion | No events at all | No | No |
| Pre-commit duplication | No events at all | No | No |
| Daemon architect escalation | No events at all | No | No |

Without denominator data (total gate evaluations), you cannot calculate:
- False positive rate (blocked spawns that shouldn't have been blocked)
- True positive rate (blocked spawns that prevented bad outcomes)
- Gate accuracy (% of decisions that were correct)

**Source:** `pkg/spawn/gates/hotspot.go` — only `LogHotspotBypass()` called on line 92, nothing on block (line 118) or allow (line 130). `pkg/daemon/architect_escalation.go` — zero event logging, only verbose stdout.

**Significance:** You can't answer "do hotspot gates improve quality?" without knowing how many spawns they evaluated, blocked, and what happened to the redirected work.

---

### Finding 3: Spawn-to-completion correlation is feasible but underutilized

**Evidence:** Of 190 spawns and 363 completions:
- 168 spawn→completion pairs can be joined via beads_id (88.4% of spawns)
- All 190 spawns have `skill` field populated
- Spawns capture rich metadata: skill, model, hotspot_area, gap_analysis, usage_info, gates_bypassed
- But completions don't reliably carry this forward (52% lack skill)

The JOIN exists but is wasted because completion events don't inherit spawn metadata.

**Source:** events.jsonl join analysis via beads_id

**Significance:** The cheapest measurement win: propagate skill/model from spawn event into completion event. Enables per-skill quality analysis without schema changes.

---

### Finding 4: Accretion delta events are severely under-collected

**Evidence:** Only 17/363 (4.7%) agent.completed events have a matching accretion.delta event. The `collectAccretionDelta()` function in `complete_lifecycle.go:62` has multiple skip conditions:
- Skips when no code files changed
- Skips for orchestrator-role completions
- Skips for untracked agents

But 4.7% suggests additional silent failures — likely git diff errors in non-standard workspace states.

**Source:** `cmd/orch/complete_lifecycle.go:62`; events.jsonl count

**Significance:** Can't answer "are agents adding to bloated files?" for 95% of completions. The accretion enforcement exists (pre-commit + completion gate) but the measurement loop is broken.

---

### Finding 5: Pipeline timing was just added and has near-zero coverage

**Evidence:** `pipeline_timing` field exists in 1/363 completions (0.3%). The `PipelineStepTiming` struct was added recently (evidenced by staged changes in logger.go). Steps tracked: hotspot, duplication, model_impact, auto_rebuild — each with duration_ms and skip_reason.

**Source:** `pkg/events/logger.go:218-224`; events.jsonl

**Significance:** This is actually good news — the infrastructure is in place but needs time to accumulate data. No action needed beyond ensuring it's wired up in all completion paths.

---

### Finding 6: Duplication detection produces no telemetry

**Evidence:** The duplication detector runs during completion (complete_duplication.go, complete_pipeline.go) and pre-commit (duplication_precommit.go), but:
- No event type for duplication detection results
- Pipeline timing captures duration but not FINDINGS (duplicates found? severity? file pairs?)
- Pre-commit hook emits zero events
- Advisory-only: results are printed to stderr and discarded

**Source:** `cmd/orch/complete_duplication.go:22` ("informational only"); `pkg/verify/duplication_precommit.go`; no event type defined for duplication results

**Significance:** Can't answer "how prevalent is duplication?" or "does the duplication detector's advisory change agent behavior?" Complete blind spot despite running the detector on every completion.

---

### Finding 7: Rework events are defined but never emitted in practice

**Evidence:** `EventTypeAgentReworked` ("agent.reworked") is defined in logger.go:28, `LogAgentReworked()` helper exists with full data struct (prior_workspace, new_workspace, rework_number, feedback, skill, model). But events.jsonl shows 0 rework events across all 4,411 entries.

Either rework is never used, or the emission code path is dead.

**Source:** `pkg/events/logger.go:297-339`; `cmd/orch/rework_cmd.go:313`; events.jsonl count

**Significance:** Rework rate is the single most important quality signal — it directly measures "did enforcement prevent bad outcomes?" Without rework data, gate effectiveness is unmeasurable.

---

### Finding 8: Hotspot bypass events lack reason field

**Evidence:** All 60 `spawn.hotspot_bypassed` events have empty `reason` field. The bypass logging in `pkg/spawn/gates/hotspot.go:92` passes the reason through, but the caller likely doesn't populate it.

**Source:** events.jsonl analysis of spawn.hotspot_bypassed events

**Significance:** Even the override data that IS collected is incomplete — can't distinguish justified bypasses from routine ones.

---

## Synthesis

**Key Insights:**

1. **Survivorship bias architecture** — The event system documents what happened to survivors (completed agents) but is silent on the selection pressure (gate decisions). This is exactly backwards for measuring gate effectiveness.

2. **Schema evolution without backfill** — As AgentCompletedData grew richer fields (skill, outcome, pipeline_timing), older emission paths weren't updated, creating a 52% coverage gap that erodes every analysis.

3. **Advisory steps are write-only** — Duplication detection, hotspot advisory, architectural choices, and discovered work all produce output that's printed to console and discarded. No telemetry means no learning loop.

**Answer to Investigation Question:**

Seven measurement gaps exist, prioritized below. The most impactful gap is the agent.completed field coverage (Finding 1) because it's cheap to fix and blocks every downstream analysis. The most strategically important gap is gate decision logging (Finding 2) because it's the only way to measure whether enforcement works.

---

## Structured Uncertainty

**What's tested:**

- ✅ events.jsonl field coverage analysis (verified: parsed all 4,411 events with Python)
- ✅ Spawn-to-completion joinability via beads_id (verified: 168/190 pairs)
- ✅ Gate decision logging gaps (verified: read hotspot.go, architect_escalation.go, complete_pipeline.go)
- ✅ Hotspot bypass reason field emptiness (verified: all 60 events lack reason)

**What's untested:**

- ⚠️ Root cause of 4.7% accretion.delta coverage (hypothesized: silent git diff failures, not confirmed)
- ⚠️ Whether rework_cmd.go:313 is actually dead code or just unused (not traced call graph)
- ⚠️ Whether pipeline_timing wiring is complete in all completion paths (only 1 event so far)

**What would change this:**

- If rework events are being emitted to a different log file, Finding 7 is wrong
- If agent.completed field gaps are intentional (legacy events kept for backwards compat), Finding 1 priority changes
- If gate decisions are logged elsewhere (daemon stdout, tmux capture), Finding 2 severity decreases

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| 1. Backfill agent.completed fields | implementation | Fix within existing event schema, no cross-boundary impact |
| 2. Add spawn.gate_decision event | architectural | New event type, affects stats aggregation, spawn gates, daemon |
| 3. Add duplication.detected event | architectural | New event type, requires defining what data to capture |
| 4. Diagnose accretion.delta coverage | implementation | Debug existing code path |
| 5. Diagnose rework event emission | implementation | Trace existing code path |
| 6. Populate hotspot bypass reason | implementation | Fix data quality in existing event |
| 7. Add daemon.architect_escalation event | architectural | New event type for daemon routing visibility |

### Recommended Approach: Prioritized Implementation ⭐

**Phased instrumentation, ordered by: (1) answers existing question, (2) cheap to implement, (3) hard to answer without data**

**Phase 1 — Data quality (1-2 issues, high impact, low effort):**
1. **Backfill agent.completed fields** — Audit all completion paths (emit_cmd, review_done, reconcile, daemon) and ensure skill/outcome/duration are always populated. Estimated: 1 file change per path, ~4 files total.
2. **Populate hotspot bypass reason** — Trace caller of LogHotspotBypass to ensure reason is passed through.

**Phase 2 — Gate visibility (2 issues, high impact, medium effort):**
3. **Add spawn.gate_decision event** — Log every gate evaluation (hotspot, triage, accretion pre-commit) with: gate_name, decision (allow/block/bypass), skill, beads_id, target_files. Update stats_cmd.go to aggregate.
4. **Add daemon.architect_escalation event** — Log escalation decisions with: issue_id, hotspot_file, hotspot_type, escalated (bool), prior_architect_ref.

**Phase 3 — Detection telemetry (2 issues, medium impact, medium effort):**
5. **Add duplication.detected event** — Log when duplication detector finds matches: file_pairs, similarity_scores, function_names. Advisory-only but now queryable.
6. **Diagnose accretion.delta coverage** — Trace why 95% of completions skip accretion collection. Fix silent failures.

**Phase 4 — Correlation (1 issue, high strategic value, needs Phase 1-3 data):**
7. **Gate effectiveness dashboard** — After 2-4 weeks of Phase 1-3 data, build query: "For agents that hit a gate (blocked/escalated), what was the outcome of the redirected work vs. agents that passed through?"

### Alternative Approaches Considered

**Option B: Build comprehensive telemetry system first**
- **Pros:** Clean schema, consistent approach
- **Cons:** High effort, delays all measurement by weeks. Prior probes already identified gaps; we need data, not architecture.
- **When to use instead:** If starting from scratch

**Option C: Sample-based measurement (manual audit)**
- **Pros:** Zero code changes
- **Cons:** Not repeatable, doesn't scale, can't detect trends over time
- **When to use instead:** For one-off hypothesis validation

**Rationale:** Phase 1-3 are independent issues that can be parallelized. Each delivers standalone value. Phase 4 requires the data from earlier phases.

---

### Implementation Details

**What to implement first:**
- agent.completed field backfill (Finding 1) — unlocks all per-skill analysis
- The code paths to audit: emit_cmd.go:103, review_done.go:297, review_done.go:331, reconcile.go:455, daemon completion_processing.go

**Things to watch out for:**
- ⚠️ Don't break backwards compatibility of events.jsonl format — new fields should be additive
- ⚠️ Gate decision events will generate volume (~10x current spawn events if logging allows too). Consider only logging blocks and escalations initially.
- ⚠️ Pre-commit hooks run in user's shell, not orch process — may need separate log path or pipe to orch emit

**Success criteria:**
- ✅ 95%+ of agent.completed events have skill, outcome, duration fields (currently 44%)
- ✅ Gate decisions (block/escalate) are queryable from events.jsonl
- ✅ Can answer: "What % of hotspot-gated spawns were reworked vs. non-gated spawns?"
- ✅ Duplication detection findings are queryable (match count, severity per completion)

---

## References

**Files Examined:**
- `pkg/events/logger.go` — Event type definitions and logging helpers
- `cmd/orch/complete_lifecycle.go` — Agent completion event emission
- `cmd/orch/complete_pipeline.go` — Advisory pipeline steps
- `cmd/orch/complete_verification.go` — Verification gate events
- `cmd/orch/complete_hotspot.go` — Hotspot advisory
- `cmd/orch/complete_duplication.go` — Duplication advisory
- `pkg/spawn/gates/hotspot.go` — Hotspot spawn gate
- `pkg/daemon/architect_escalation.go` — Daemon routing decisions
- `pkg/verify/accretion.go` — Accretion verification
- `pkg/verify/accretion_precommit.go` — Pre-commit accretion gate
- `pkg/verify/duplication_precommit.go` — Pre-commit duplication advisory
- `cmd/orch/stats_cmd.go` — Stats aggregation and analysis
- `cmd/orch/emit_cmd.go` — Manual event emission
- `cmd/orch/review_done.go` — Review reconciliation events
- `cmd/orch/rework_cmd.go` — Rework event emission

**Commands Run:**
```bash
# Event type distribution
python3 -c "import json, collections; ..." # Counted all 4,411 events by type

# Field coverage analysis
python3 -c "..." # Checked agent.completed field presence rates

# Spawn-to-completion join analysis
python3 -c "..." # Matched spawn/completion pairs via beads_id

# Hotspot bypass reason analysis
python3 -c "..." # Checked reason field in spawn.hotspot_bypassed events
```

**Related Artifacts:**
- **Probe:** .kb/models/entropy-spiral/probes/2026-03-01-probe-fix-feat-ratio-gate-effectiveness.md — Fix:feat ratio analysis, identified need for gate effectiveness measurement
- **Probe:** .kb/models/completion-verification/probes/2026-03-01-probe-hotspot-blind-spot-analysis.md — Identified 5 blind spots in hotspot detection

---

## Investigation History

**2026-03-11:** Investigation started
- Initial question: Where are measurement gaps in orch-go's structural enforcement?
- Context: Spawned to audit instrumentation coverage and identify data needed to prove/disprove gate effectiveness

**2026-03-11:** Primary analysis complete
- Analyzed all 4,411 events in events.jsonl
- Mapped all 19 event types to 55+ emission sites
- Identified 8 findings across 3 severity levels
- Produced prioritized 4-phase implementation recommendation
