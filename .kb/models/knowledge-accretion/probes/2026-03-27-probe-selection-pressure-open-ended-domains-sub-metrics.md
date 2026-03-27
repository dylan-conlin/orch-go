# Probe: Selection Pressure in Open-Ended Domains with Well-Defined Sub-Metrics

**Model:** knowledge-accretion
**Date:** 2026-03-27
**Status:** Complete
**claim:** KA-5 (substrate independence), effectiveness hierarchy, selection pressure as coordination
**verdict:** extends

---

## Question

HyperAgents manages accretion via evolutionary selection: let agents accrete freely, then apply selection pressure so bad variants die and good ones propagate. This works on closed benchmarks (coding, math grading, paper review) where evaluation is automated and scalar.

Could this approach work for **open-ended domains that have well-defined sub-metrics** — domains where the whole isn't reducible to a number, but measurable components exist?

Specifically:
1. What sub-metrics in orch-go's domain are already automatable?
2. Would selection pressure on those sub-metrics produce useful pruning, or would it optimize for the measurable at the expense of the unmeasurable?
3. What's the minimum evaluation infrastructure needed to get partial selection pressure working?

---

## What I Examined

Analyzed the HyperAgents approach against three categories of open-ended work that have measurable sub-dimensions, using orch-go's existing data.

---

## What I Observed

### 1. The Spectrum Between Closed and Open Evaluation

HyperAgents operates at one extreme: fully automated evaluation, scalar score, closed benchmark. orch-go operates at the other: human judgment required for compositional quality. But there's a spectrum:

| Domain Type | Example | Evaluation | Selection Viable? |
|-------------|---------|------------|-------------------|
| Closed benchmark | HyperAgents coding tasks | Automated, scalar | Yes — this is what HyperAgents does |
| Closed with style | Code review (correctness + readability) | Partially automated | Partially — correctness is automatable, style needs judgment |
| Open with sub-metrics | Knowledge base quality (orphan rate, staleness, coverage) | Sub-metrics automated, composition human | **This is the question** |
| Fully open | "Is this investigation insightful?" | Human judgment only | No — selection pressure requires evaluation |

The question targets row 3: domains that are open-ended in aggregate but have measurable facets.

### 2. Automatable Sub-Metrics Already Present in orch-go

orch-go already measures several things that could serve as selection signals:

| Sub-Metric | Current Measurement | Selection Pressure Analogue |
|------------|--------------------|-----------------------------|
| Orphan rate | 87.6% overall, 52% in model era | Investigations that connect to models score higher than orphans |
| Test passage | `go test ./...` | Contributions that break tests are discarded (already a gate) |
| File size / accretion | `orch hotspot` | Contributions that bloat files beyond threshold score lower |
| Knowledge staleness | `kb reflect` identifies stale models | Models that haven't been probed recently score lower for routing |
| Probe-to-model merge | Completion gate checks | Probes without model updates are flagged |
| Build success | `go build` | Already a hard gate |

**Key observation:** Most of these are already implemented as gates (binary pass/fail), not as selection pressure (continuous scoring). The difference matters.

### 3. Gates vs. Selection Pressure: The Crucial Distinction

Gates say: "Does this contribution meet minimum quality?" (binary)
Selection pressure says: "How good is this contribution relative to alternatives?" (ranking)

Gates prevent the worst outcomes. Selection pressure promotes the best outcomes. HyperAgents uses both — compilation is a gate, benchmark score is selection pressure. orch-go currently uses gates extensively but has no selection pressure mechanism.

**Why selection pressure requires alternatives:** Selection only works when you have multiple candidates for the same slot. HyperAgents generates a population of variants for each task. orch-go typically spawns one agent per task. Without alternatives, there's nothing to select between.

This is the structural reason HyperAgents' approach doesn't directly translate: orch-go's workflow is single-agent-per-task, not population-per-task.

### 4. Where Sub-Metric Selection Could Work

Despite the single-agent constraint, there are places where orch-go already produces alternatives that could be ranked:

**a) Investigation synthesis selection (multiple investigations on same topic)**
- orch-go already has cases where 3+ agents investigate overlapping questions
- Currently: each produces an orphaned investigation, human picks the useful one
- With selection: automated scoring on coverage (mentions relevant files), freshness (uses current code state), connection density (links to existing models) could surface the best synthesis
- Risk: Goodhart's Law — agents optimize for connection density by adding spurious links

**b) Skill variant selection (multiple approaches to same task)**
- HyperAgents' core mechanism: generate variants, evaluate, keep the best
- orch-go could: spawn N agents with different approaches (TDD vs direct, different decompositions), run automated sub-metrics (test coverage, code complexity, build time), surface the best for human selection
- Cost: N× token usage. HyperAgents spent 88.6M tokens for 100 iterations. Generating 3 variants of a feature triples spawn cost.
- Benefit: only valuable when the task is hard enough that approach matters

**c) Knowledge base quality maintenance (continuous selection)**
- Instead of gates that block bad contributions, use continuous scoring to surface which knowledge artifacts are valuable
- Sub-metrics: citation count (how often other investigations reference this), prediction accuracy (did the model's claims hold up in subsequent probes), freshness (when was this last validated)
- This is closer to PageRank than HyperAgents — ranking existing artifacts rather than generating variants
- Risk: recency bias (recent = higher score, but older stable knowledge is more valuable)

### 5. The Goodhart Problem: Sub-Metrics Are Not the Metric

The fundamental risk of applying selection pressure to sub-metrics in open-ended domains:

**Goodhart's Law:** When a measure becomes a target, it ceases to be a good measure.

HyperAgents avoids this because their benchmarks *are* the evaluation — passing the coding test means the code works. In open-ended domains, sub-metrics are proxies:

| Sub-Metric | What It Measures | What It Doesn't Measure |
|------------|-----------------|------------------------|
| Orphan rate | Connection to models | Whether the connection is meaningful |
| Test coverage | Code path execution | Whether tests verify the right things |
| Knowledge freshness | Recency of updates | Whether updates improved understanding |
| Citation count | How often referenced | Whether references are substantive |

Applying selection pressure to these sub-metrics would optimize agents for gaming proxies. An agent that adds a spurious "Related to model X" link reduces orphan rate without improving knowledge quality. An agent that writes trivial tests increases coverage without catching bugs.

**HyperAgents partially avoids Goodhart because:**
1. Their benchmarks have ground truth (the code either passes tests or doesn't)
2. Their evaluation is holistic within the benchmark (not a proxy for a larger goal)
3. Their meta agent can modify the evaluation strategy itself (co-evolution of metrics and agents)

Point 3 is the most interesting. HyperAgents' meta agent invented bias detection (`_analyze_evaluations()`) that catches when the task agent games the metric (e.g., 99% acceptance rate = classification collapse). The meta agent evolved to detect Goodhart failure.

### 6. The Viable Middle Ground: Composite Selection with Human Veto

The approach that could work for open-ended domains:

1. **Automate sub-metric evaluation** — orphan rate, test passage, build success, file size, staleness
2. **Composite scoring** — weight sub-metrics into a single quality signal (not a replacement for human judgment, but a prioritization signal)
3. **Human veto on composition** — the composite score surfaces candidates; human judges compositional quality
4. **Selection at triage, not at generation** — don't generate N variants (too expensive); instead, use sub-metric scores to prioritize which completed work gets human attention first

This is not HyperAgents' approach (generate variants, select winners). It's a hybrid: gate the worst (existing), score the middle (new), human-judge the composition (existing but more focused).

**The key shift:** Selection pressure doesn't replace human judgment on open-ended quality. It reduces the volume of work that needs human judgment by filtering and ranking.

### 7. Minimum Infrastructure for Partial Selection

To test this in orch-go:

| Component | Implementation | Effort |
|-----------|---------------|--------|
| Sub-metric scorer | Script that computes orphan status, test coverage, staleness per artifact | Small — mostly wiring existing tools |
| Composite score | Weighted sum with configurable weights | Small |
| Triage ranking | `orch triage --ranked` that orders work by composite score | Medium — extends existing daemon triage |
| Feedback loop | Track whether human accepts/rejects high-scored vs low-scored work | Medium — needs tracking infrastructure |
| Weight tuning | Adjust composite weights based on human accept/reject patterns | Large — this is the learning loop |

The first three are implementable. The last two make the system learn which sub-metrics actually predict quality — which is the HyperAgents insight applied to open-ended domains.

---

## Findings

### The Direct Answer

**Yes, with constraints.** HyperAgents' "let accretion happen, apply selection pressure" can work for open-ended domains with well-defined sub-metrics, but not as a direct transplant:

1. **Population-based selection doesn't translate** — orch-go can't afford N variants per task. Selection must happen at triage (ranking completed work for human review), not at generation (picking the best variant).

2. **Sub-metric selection is useful for filtering, dangerous for optimization** — using orphan rate, test coverage, and staleness to *prioritize* human attention is safe. Using them as *optimization targets* for agents invites Goodhart failure. The distinction: selection pressure for routing (which work to review first) vs. selection pressure for generation (which agents to keep).

3. **The meta-agent insight is the most transferable part** — HyperAgents' meta agent evolved to detect when its own metrics were being gamed (bias detection, classification collapse detection). If orch-go automates sub-metrics, it also needs automated detection of metric gaming. This is the self-disconfirming knowledge thread's core insight: the system must pressure its own measures.

4. **Minimum viable version: scored triage** — composite sub-metric scores on completed work, surfaced during daemon triage, with human accept/reject feedback that tunes weights over time. This is the smallest intervention that brings selection pressure into an open-ended system without generating population-level waste.

### Connection to Existing Threads

- **Self-disconfirming knowledge:** Selection pressure on sub-metrics IS partial self-disconfirmation — the system automatically identifies which contributions are weak on measurable dimensions, freeing human judgment for compositional quality.
- **Verification bottleneck:** Sub-metric scoring automates the easy part of verification (did it break? is it connected? is it fresh?) while preserving human judgment for the hard part (is it insightful? does it compose well?).
- **Epistemic status:** The Goodhart risk means sub-metric selection should be framed as "bounded rationality under measurement constraints" not "objective quality scoring." The metrics are heuristics, not ground truth.

---

## Model Impact

- [x] **Extends** model with:
  1. **Selection pressure taxonomy for open-ended domains:** Population-based selection (HyperAgents) requires automated holistic evaluation. Sub-metric selection (proposed) works for routing/prioritization but not optimization. The distinction maps to: selection-for-routing (safe in open-ended domains) vs. selection-for-generation (requires closed evaluation or risks Goodhart).
  2. **Goodhart boundary as a design constraint:** When applying coordination mechanisms from closed to open domains, the boundary where Goodhart's Law activates is a design constraint, not just a risk. Sub-metrics can coordinate routing (which work gets attention) but cannot coordinate generation (which work gets created) without ground-truth evaluation.
  3. **Meta-evaluation as a coordination mechanism:** HyperAgents' meta agent evolving bias detection is a new form of coordination: evaluating the evaluation. This maps to the self-disconfirming knowledge thread — the system pressuring its own metrics is a coordination mechanism at the meta-evaluation level.

---

## Notes

### Why Not Just Generate Variants?

The obvious objection: "If HyperAgents works by generating variants, why not just spawn 3 agents per task and pick the best?"

Cost. HyperAgents spent 88.6M tokens on 100 iterations across 4 domains. Tripling orch-go's spawn cost for marginal improvement on tasks where approach usually doesn't matter (most feature-impl tasks have one obvious approach) is wasteful. The exception: genuinely hard design problems where multiple approaches are viable. For those, the existing architect skill already explores alternatives — it just doesn't use automated sub-metrics to rank them.

### The PageRank Analogy

Sub-metric scoring of existing knowledge artifacts is closer to PageRank than to HyperAgents. PageRank doesn't generate new pages — it ranks existing ones by link structure. Sub-metric scoring doesn't generate new investigations — it ranks existing ones by connection density, freshness, and validation status. The selection pressure is retrospective (which existing work is most valuable) rather than generative (which new work to create).

This is a more natural fit for orch-go's single-agent-per-task architecture than HyperAgents' population-based approach.
