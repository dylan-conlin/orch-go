# Design: Automated Synthesis Ranking for Overlapping Agent Work

**Date:** 2026-03-27
**Status:** Complete
**Model:** knowledge-accretion
**Issue:** orch-go-rhwly
**Skill:** architect

---

## Design Question

When multiple agents investigate the same or overlapping questions, how should orch-go automatically rank their syntheses to reduce Dylan's cognitive load while keeping human judgment for final selection?

---

## Problem Framing

**Success criteria:**
1. Reduces cognitive load when reviewing overlapping syntheses
2. Human remains decision-maker — system ranks/filters, doesn't auto-select
3. Works within existing infrastructure (completion pipeline, comprehension queue, briefs)
4. Fits Layer 2 "method-expressing ordering" from the ranking/attention probe

**Constraints:**
- "Synthesis is strategic orchestrator work" (decision 2026-01-07) — cannot auto-select or auto-synthesize across agents
- No local agent state (CLAUDE.md constraint) — cannot maintain ranking DB or similarity index
- HyperAgents finding: evolve coordination infrastructure, not selection logic
- Must reduce load, not add a new step

---

## Exploration: 5 Decision Forks

### Fork 1: Where in the pipeline does ranking happen?

| Option | Pros | Cons |
|--------|------|------|
| **(a) Post-completion, in brief metadata** | Extends existing pipeline; score is in artifact (no local state); stable since SYNTHESIS.md doesn't change | Scores can't be recalibrated without re-running |
| (b) Brief serving time (dynamic) | Flexible, recalibrable | Needs scoring cache (violates no-local-state) |
| (c) New `orch rank-syntheses` command | Explicit control | Adds a step for orchestrator (cognitive load) |
| (d) Comprehension queue ordering | Natural integration point | Too late — ordering is about briefs, not synthesis quality |

**→ Recommend (a).** Substrate: "Gate Over Remind" principle (signal-to-design-loop) — embed in existing flow. No-local-state constraint eliminates (b). "Reduce load, not add" eliminates (c).

### Fork 2: What signals indicate synthesis quality?

Six mechanically-detectable signals, grouped by reliability:

**Tier A — High signal, mechanically detectable:**

| Signal | Detection Method | Source Field |
|--------|-----------------|-------------|
| `structural_completeness` | Count populated D.E.K.N. sections (0-4 score) | All Synthesis fields |
| `evidence_specificity` | Regex for file paths, test output, command results | Evidence + Delta |
| `model_connection` | Regex for `.kb/models/`, "confirms/contradicts/extends" | Knowledge |
| `connective_reasoning` | Existing `HasConnectiveLanguage()` from quality.go | Knowledge + TLDR |

**Tier B — Detectable, noisier:**

| Signal | Detection Method | Source Field |
|--------|-----------------|-------------|
| `tension_quality` | UnexploredQuestions populated AND contains `?` | UnexploredQuestions |
| `insight_vs_report` | NOT `IsActionVerbSentence()` for Knowledge lines | Knowledge |

**Held-back (Layer 3, not designed here):**
- Semantic coherence (requires LLM)
- Decision contradiction detection (requires cross-reference)
- Novel insight detection (requires domain understanding)

**→ Recommend: 6 boolean/count signals.** Substrate: knowledge accretion model warns against "formula-shaped sentences" — no numeric score without calibrated weights. Signal list is the honest representation.

### Fork 3: How to detect overlapping work?

| Option | Pros | Cons |
|--------|------|------|
| **(a) Thread-based grouping** | Infrastructure exists (active_work, resolved_by) | Requires brief→thread reverse-lookup (not yet in API) |
| (b) Spawn context similarity | Would catch non-threaded overlap | Needs LLM or embedding index (no local state) |
| (c) Beads label matching | Simple | Low precision — same label ≠ same question |
| (d) Manual (current) | Orchestrator judgment | Doesn't scale (the problem we're solving) |

**→ Recommend (a) for future, (d) for now.** Thread-grouped display was already designed by the ranking/attention probe (Layer 2). This design adds quality signals that ENABLE that grouping to be useful. Overlap detection requires thread infrastructure that's a separate issue.

### Fork 4: Score representation

| Option | Pros | Cons |
|--------|------|------|
| (a) Numeric (0-100) | Single sortable value | False precision without calibration |
| (b) Categorical (high/medium/low) | Simple | Loses signal detail |
| **(c) Signal list with counts** | Honest, enables future learning | No single sort key |
| (d) Comparative rank | Natural for "which is best" | Requires grouped display |

**→ Recommend (c).** Signal list preserves information for both human interpretation and future calibration. The `signal_count` (how many of 6 signals fired) is a natural sort key for comprehension queue ordering — sufficient for Layer 2 without pretending calibrated weighting.

### Fork 5: New infrastructure or extend existing?

**Existing infrastructure that nearly covers this:**
- `verify.ParseSynthesis()` — extracts all needed fields ✓
- `debrief.CheckQuality()` — has connective language, action-verb detection ✓
- `complete_brief.go` — maps synthesis → brief, injection point for metadata ✓
- `comprehension_queue.go` — ordering point (currently mod-time only) ✓

**What's missing:**
1. `SynthesisQualitySignals()` function (new, in `pkg/verify/`)
2. Signal metadata in brief frontmatter (extend `buildBriefFromSynthesis`)
3. Signal-aware ordering in comprehension queue (extend queue sort)

**→ Recommend: Extend existing.** No new packages, no new commands, no new data stores. Three functions added to existing files.

---

## Synthesis: Recommended Design

### Component 1: `SynthesisQualitySignals` (pkg/verify/synthesis_quality.go)

New file in `pkg/verify/` (NOT a governance-protected path). Takes a parsed `*Synthesis` struct, returns signal results:

```go
type QualitySignal struct {
    Name     string // e.g., "structural_completeness"
    Detected bool   // Whether the signal was found
    Score    string // e.g., "4/4" or "3/5 lines"
    Evidence string // Excerpt that triggered detection
}

type SynthesisQuality struct {
    Signals     []QualitySignal
    SignalCount int // How many of 6 fired (natural sort key)
    Total       int // Always 6 (denominator)
}

func ComputeSynthesisQuality(s *Synthesis) SynthesisQuality
```

Detection logic per signal:
- `structural_completeness`: count of non-empty fields in {TLDR, Delta, Evidence, Knowledge}
- `evidence_specificity`: regex `(pkg/|cmd/|\.go|\.ts|\.md|PASS|FAIL|test|assert)` in Evidence+Delta
- `model_connection`: regex `(\.kb/models/|confirms?|contradicts?|extends?)` in Knowledge
- `connective_reasoning`: reuse `debrief.HasConnectiveLanguage()` on Knowledge+TLDR
- `tension_quality`: UnexploredQuestions non-empty AND contains `?`
- `insight_vs_report`: ratio of Knowledge lines that are NOT `IsActionVerbSentence()`

### Component 2: Brief Metadata Injection (cmd/orch/complete_brief.go)

Extend `buildBriefFromSynthesis()` to prepend YAML frontmatter with quality signals:

```yaml
---
beads_id: orch-go-xxxxx
quality_signals:
  structural_completeness: "4/4"
  evidence_specificity: true
  model_connection: true
  connective_reasoning: true
  tension_quality: true
  insight_vs_report: "3/5"
signal_count: 6
signal_total: 6
---
```

This metadata is:
- Machine-readable (parseable by brief API for ordering)
- Human-readable (orchestrator sees signal summary)
- In the artifact (no local state needed)
- Stable (SYNTHESIS.md doesn't change post-completion)

### Component 3: Comprehension Queue Ordering (pkg/daemon/comprehension_queue.go)

When listing briefs (or comprehension queue items), sort by:
1. Comprehension state: unread > processed > read (existing, unchanged)
2. **Signal count** within state: higher signal_count = read first
3. Recency within signal-count tier: newest first (existing fallback)

This is Layer 2 "method-expressing ordering" from the ranking probe. It says: "a synthesis with evidence, model connections, and open questions should be read before a synthesis that's just a task report."

### Component 4: Thread-Grouped Display (future, not in this design)

Quality signals ENABLE thread-grouped ranking but don't implement it. When the brief→thread reverse-lookup is built (per the ranking probe's recommendation), signal_count becomes the within-thread sort key.

---

## Architectural Choices

### Choice 1: Signals, Not Scores
- **Chose:** Boolean/count signal list with `signal_count` as sort key
- **Rejected:** Weighted numeric score (0-100)
- **Why:** Knowledge accretion model explicitly warns about "formula-shaped sentences" — numeric scores without calibrated weights pretend precision that doesn't exist. Signal count is honest: 6/6 signals detected is better than 2/6, but we don't know the relative weight of each signal.
- **Risk:** `signal_count` treats all signals as equal weight, which is probably wrong. Evidence specificity likely matters more than structural completeness. Accepted: once brief feedback data accumulates, weights can be calibrated. Until then, equal weight is the honest prior.

### Choice 2: Completion-Time Scoring
- **Chose:** Compute signals at `orch complete`, store in brief frontmatter
- **Rejected:** Compute at brief-serve time (dynamic scoring)
- **Why:** SYNTHESIS.md is immutable after completion. Scoring at read time adds latency and requires parsing archived workspaces. No-local-state constraint means we can't cache scores in a DB.
- **Risk:** If signal definitions improve, old briefs have stale scores. Mitigation: a batch `orch rescore-briefs` command could re-parse and update frontmatter.

### Choice 3: Extend Existing, Don't Create New
- **Chose:** Three functions across existing packages (verify, complete_brief, comprehension_queue)
- **Rejected:** New `pkg/ranking/` package
- **Why:** The design adds ~150 lines of code. A new package for 150 lines creates overhead without benefit. If ranking grows, extraction is straightforward.
- **Risk:** Functions scattered across 3 files may be hard to find. Mitigation: `synthesis_quality.go` name is self-documenting.

---

## Defect Class Exposure

| Class | Risk | Mitigation |
|---|---|---|
| 1: Filter Amnesia | Quality signals computed in headless path but missing from manual brief creation | Single `ComputeSynthesisQuality()` function called from both paths |
| 3: Stale Artifact Accumulation | Old briefs with outdated signal format if taxonomy changes | Signals are simple YAML; backward-compatible additions don't break old format |
| 5: Contradictory Authority Signals | Quality signals could disagree with human feedback (shallow/good) | Signals are advisory. Human feedback (brief feedback mechanism) is authoritative. If they consistently disagree, that's calibration data. |

---

## Recommendations

### Implementation Plan (single-component, no decomposition needed)

One feature-impl issue covering all three components:

**Title:** "Implement synthesis quality signals with brief metadata injection"
**Scope:**
1. `pkg/verify/synthesis_quality.go` — signal computation
2. `cmd/orch/complete_brief.go` — metadata injection into brief frontmatter
3. `pkg/daemon/comprehension_queue.go` — signal-aware ordering (read briefs' frontmatter for sort)
4. Tests for signal detection functions

**Acceptance criteria:**
- `ComputeSynthesisQuality()` correctly detects all 6 signals from SYNTHESIS.md
- `generateHeadlessBrief()` produces brief with YAML frontmatter containing quality signals
- Comprehension queue lists briefs ordered by signal_count within each state tier
- Existing completion pipeline behavior unchanged (signals are advisory, not gates)

### Blocking Questions (1)

**Q1 (architectural, judgment):** Should the signal-aware ordering apply ONLY to the comprehension queue, or also to `orch review` batch listing? Currently `orch review --needs` filters failures only. Adding signal-based ordering to review would help the orchestrator triage completions.

Recommendation: Start with comprehension queue only. Extend to `orch review` if orchestrator finds it useful. Incremental.

### Migration Status

```
MIGRATION_STATUS:
  designed: synthesis quality signal taxonomy (6 signals), brief metadata injection, signal-aware comprehension ordering
  implemented: none
  deployed: none
  remaining: feature-impl issue for all 3 components
```

---

## Connection to Models

**Knowledge accretion:** This design adds a coordination mechanism (quality signals as structural metadata) that makes synthesis quality visible without LLM runtime judgment. It sits in the effectiveness hierarchy at "structural attractor" level — signals embedded in artifacts route attention at read time, not requiring correct decisions at write time.

**Signal-to-design-loop:** Quality signals are a clustering mechanism. Individual syntheses are signals; quality metadata makes them groupable by "how much should this be read?" This moves from Stage 1 (capture) — which the completion pipeline already does — to Stage 2 (clustering), which was previously manual.

**HyperAgents:** Following the key finding exactly: evolve coordination infrastructure (signal taxonomy, metadata injection, ordering), not selection logic (which synthesis is "best"). The handcrafted 6-signal taxonomy is deliberately not auto-tuned, consistent with HyperAgents' finding that self-modified selection mechanisms don't outperform handcrafted ones.

---

## Next

**Recommendation:** close

Follow-up: Create feature-impl issue for implementation.
