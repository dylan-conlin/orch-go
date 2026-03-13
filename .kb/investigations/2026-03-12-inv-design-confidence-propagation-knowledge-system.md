## Summary (D.E.K.N.)

**Delta:** Confidence propagation failure is Defect Class 5 (Contradictory Authority Signals) in the knowledge system — caveats at sources don't reach citing artifacts. Three-layer design resolves this using evidence quality annotations on model claims, a `kb audit provenance` scanner, and probe-to-model merge gate enhancement.

**Evidence:** Dilution curve case study (4 downstream artifacts cited unreplicated thresholds as established); orchestrator-skill model already uses a 4-category evidence quality taxonomy organically across 67 claims; existing gate infrastructure (GateProbeModelMerge, accretion) provides composable enforcement; kb quick `ref_count`/`last_ref` fields exist in schema but are never updated.

**Knowledge:** The model is the canonical confidence authority, not a citation graph. Propagation works by keeping models accurate (gate enforcement at probe merge time) and surfacing model confidence to consumers (kb context injection). No projection DB needed — scan .kb/ directly.

**Next:** Implement in 3 phases: (1) `kb audit provenance` command in kb-cli, (2) evidence quality annotation convention + probe gate enhancement in orch-go, (3) kb context confidence injection. Create implementation issues.

**Authority:** architectural — Cross-component (kb-cli schema, model format, orch-go gate system, skill workflow). Orchestrator decides.

---

# Investigation: Confidence Propagation Design for the Knowledge System

**Question:** How should confidence in knowledge claims propagate through the kb system so that downstream artifacts don't treat caveated claims as established fact?

**Started:** 2026-03-12
**Updated:** 2026-03-12
**Owner:** architect (orch-go-srw17)
**Phase:** Complete
**Next Step:** None — recommendations ready for implementation routing
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-12 orchestrator-skill contradiction analysis probe | deepens | yes | none |
| 2026-03-12 evidence inventory probe (67 claims) | extends | yes | none |
| 2026-03-02 opus dilution curve replication | extends | yes | none — replication failure is the motivating case |
| 2026-03-11 orchestrator-skill design tension mapping | extends | yes | none |
| 2026-03-02 layered constraint enforcement design probe | extends | yes | none |

---

## Findings

### Finding 1: The failure is Defect Class 5, not a missing feature

**Evidence:** The dilution curve case: replication failure was noted in the source probe (Mar 4, orch-go-zola), but 4 downstream artifacts cited the thresholds as established fact. This is structurally identical to Defect Class 5 (Contradictory Authority Signals) — multiple artifacts disagree about the same claim's status. The source says "caveated" but consumers say "established."

**Source:** `.kb/models/orchestrator-skill/model.md` lines 46-54 (Claim 4 with propagation failure documentation); `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-orchestrator-skill-investigation-cluster-contradiction-analysis.md` (DC-1)

**Significance:** This reframes the problem. It's not "we need a new system" — it's "our existing defect prevention pattern (single canonical derivation) should apply to the knowledge system." The fix pattern is the same: establish one canonical source of truth and eliminate contradictory signals.

---

### Finding 2: The model is already the canonical confidence authority

**Evidence:** The orchestrator-skill model (model.md) already annotates each of its 6 claims with evidence quality:
- Claim 1: "Multi-source analytical (2 investigations)"
- Claim 2: "Highest-confidence finding in the cluster. Confirmed across 4 independent sources"
- Claim 4: "Measured but caveated (single-source, replication failed, N=3)"

This vocabulary emerged organically from the evidence inventory probe (67 claims → 39 measured, 26 analytical, 2 assumed, 14 multi-source verified). The taxonomy is already proven at scale.

**Source:** `.kb/models/orchestrator-skill/model.md` lines 28-68; evidence inventory probe

**Significance:** The taxonomy doesn't need to be invented — it already exists. The 4-level system (replicated, single-source, analytical, assumed) matches actual usage. More levels would be unused; fewer would lose the critical replicated/single-source distinction that the dilution curve case exposed.

---

### Finding 3: The propagation gap is between probe time and consumption time

**Evidence:** The dilution curve caveat was added to the probe file on Mar 4. The model was updated with "HYPOTHESIZED" only during the Mar 12 contradiction analysis — 8 days later. During those 8 days, 4 artifacts cited the thresholds as established. The gap is: (a) the probe-to-model merge didn't happen immediately, and (b) when it did happen, downstream artifacts weren't notified.

However, the existing system already has the right mechanism: `kb context` injects model content into spawn context. If the model had been updated at probe time (Mar 4), subsequent agents would have seen the caveat via `kb context`. The failure was at the merge step, not the propagation step.

**Source:** Contradiction analysis probe (DC-1 timeline); worker-base skill probe-to-model merge requirement

**Significance:** The primary fix is strengthening the probe-to-model merge workflow, not building a citation graph. The existing `kb context` → spawn context pipeline IS the propagation mechanism. It works when models are up to date; it fails when models are stale. The design should focus on keeping models accurate, not on building new propagation infrastructure.

---

### Finding 4: Existing gate infrastructure is composable

**Evidence:** The gate system has patterns for exactly this kind of enforcement:
- `GateProbeModelMerge` (V1): Already checks that probes with contradicts/extends verdicts show model.md was updated. This is the injection point for confidence enforcement.
- Accretion gate: Demonstrates advisory (800 lines) → blocking (1500 lines) progression.
- Agreements check: Demonstrates periodic audit pattern (warning-only spawn gate + daemon integration).
- `ref_count`/`last_ref` fields: Exist in QuickEntry schema but are never updated — schema extension is precedented.

**Source:** `pkg/verify/check.go` (gate orchestration); `pkg/spawn/gates/agreements.go` (periodic check pattern); `kb-cli/cmd/kb/quick.go` lines 72-73 (unused ref tracking fields)

**Significance:** No new infrastructure patterns needed. The design composes existing patterns: extend `GateProbeModelMerge` for confidence enforcement, use the agreements pattern for periodic audit, use existing `kb context` for propagation.

---

### Finding 5: Behavioral tagging will fail — infrastructure must own it

**Evidence:** The orchestrator-skill model's core finding: behavioral constraints have 0% enforcement guarantee at prompt level (Claim 1). Skill text doesn't stick for behavioral enforcement. 5 of 7 tested behavioral constraints showed zero measurable lift (Claim 2). Requiring agents to "remember to tag confidence" would be yet another behavioral constraint that fails.

**Source:** `.kb/models/orchestrator-skill/model.md` Claims 1-2

**Significance:** This eliminates Option A (confidence tagging by agents) as a standalone mechanism. Any solution must be infrastructure-enforced: gates that check for annotations, commands that detect gaps, spawn context that surfaces confidence. The agent's job is to update the model during probe-to-model merge (already required); the infrastructure's job is to verify that update happened and propagate it.

---

## Synthesis

**Key Insights:**

1. **The model IS the propagation mechanism** — Models are already injected into agent context via `kb context`. If models accurately reflect evidence quality, downstream agents see accurate confidence levels. The fix is keeping models accurate (gate enforcement), not building a citation graph (projection DB, which is architecturally prohibited).

2. **Three layers, following the established enforcement pattern** — Layer 1: spawn advisory (evidence quality visible in `kb context`), Layer 2: completion gate (probe-to-model merge includes confidence), Layer 3: periodic audit (`kb audit provenance` command for system-wide sweeps).

3. **The taxonomy already exists** — Don't invent categories. The 4-level evidence quality vocabulary (replicated, single-source, analytical, assumed) emerged organically from the evidence inventory probe and is already used in the orchestrator-skill model.

**Answer to Investigation Question:**

Confidence propagation should work through three mechanisms, layered like the existing enforcement pattern:

**Layer 1 — Evidence Quality Annotations (the vocabulary):** Standardize the 4-level evidence quality taxonomy in model.md claims. Models already use this organically; standardize the format so it's mechanically detectable.

**Layer 2 — Probe-to-Model Merge Gate Enhancement (per-session enforcement):** Extend `GateProbeModelMerge` to also verify that when a probe contradicts/extends a claim, the model's evidence quality annotation is updated. This is the primary fix — if probe contradictions reliably update model confidence, `kb context` propagates the change to downstream agents automatically.

**Layer 3 — `kb audit provenance` Command (periodic sweep):** A command in kb-cli that scans models for evidence quality gaps: unannotated claims, stale annotations, models where probes contradicted claims but the model wasn't updated. This catches drift that per-session gates miss.

This design requires zero behavioral compliance from agents (it's all gates and commands), no projection DB (scans .kb/ directly), and composes existing infrastructure (probes, gates, kb context).

---

## Structured Uncertainty

**What's tested:**

- ✅ 4-level taxonomy is usable at scale (verified: orchestrator-skill model applied it to 67 claims)
- ✅ `GateProbeModelMerge` infrastructure exists and fires on contradicts/extends verdicts (verified: `pkg/verify/check.go`)
- ✅ `kb context` injects model content into spawn context (verified: `kb-cli/cmd/kb/context.go`)
- ✅ `ref_count`/`last_ref` fields exist in QuickEntry schema but are unused (verified: `kb-cli/cmd/kb/quick.go:72-73`)

**What's untested:**

- ⚠️ Whether agents actually READ evidence quality annotations in `kb context` output (not measured — known risk from orchestrator-skill behavioral compliance data)
- ⚠️ Whether the 8-day probe-to-model merge delay (dilution curve case) was due to missing enforcement or workflow friction (investigating root cause would strengthen the gate design)
- ⚠️ Whether `kb audit provenance` scanning is feasible without structured claim extraction from freeform markdown (implementation spike needed)

**What would change this:**

- If evidence quality annotations in markdown prove too fragile to scan mechanically → would need structured claim format (YAML frontmatter per claim)
- If agents ignore evidence quality even when surfaced → would need infrastructure-level filtering (don't inject low-confidence claims into spawn context without warning prefix)
- If probe-to-model merge gate already catches all cases → `kb audit provenance` command becomes nice-to-have, not essential

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Evidence quality taxonomy (vocabulary) | implementation | Convention, not code — standardize what's already organic |
| `kb audit provenance` command | architectural | New command in kb-cli, cross-component interaction with models |
| Probe-to-model merge gate enhancement | architectural | Modifies verification pipeline, affects all agent sessions |
| kb context confidence injection | architectural | Changes what agents see in spawn context |

### Recommended Approach ⭐

**Three-Layer Confidence Enforcement** — Standardize evidence quality annotations, strengthen probe-to-model merge gate, add periodic provenance audit.

**Why this approach:**
- Follows the proven three-layer enforcement pattern (spawn advisory, completion gate, periodic audit)
- Requires zero behavioral compliance from agents — all infrastructure
- No projection DB — scans .kb/ directly (respects "no local agent state" constraint)
- Composes existing infrastructure (probes, gates, kb context)

**Trade-offs accepted:**
- Model-level, not claim-level enforcement in gates — gate checks that model.md was updated, not that specific claims are correctly annotated. Claim-level accuracy depends on the agent doing the merge. This is acceptable because the probe-to-model merge is already agent work; adding claim-level gate validation would require structured claim extraction from freeform markdown (significant infrastructure).
- No automatic downstream notification — agents consuming stale models won't be warned in real time. They'll see updated models on next spawn via `kb context`. This is acceptable because the propagation delay (time between model update and next spawn) is typically hours, not days.

**Implementation sequence:**

#### Phase 1: Evidence Quality Convention + Provenance Audit (kb-cli)

Add `kb audit provenance` command to kb-cli:

```
kb audit provenance [--model <name>] [--format json|text]
```

Scans all model.md files (or specific model) for:
1. **Unannotated claims** — claims without `**Evidence quality:**` annotation (gap detection)
2. **Low-confidence claims** — claims annotated as `single-source` or `assumed` (risk flags)
3. **Orphan contradictions** — probes with `contradicts` in Model Impact but model.md not updated since probe date
4. **Coverage metric** — percentage of claims with evidence quality annotations per model

Output format:
```
Model: orchestrator-skill (6 claims)
  Coverage: 100% (6/6 annotated)
  Risk: 2 single-source claims (Claim 4: dilution budget, Claim 6: emphasis language)
  Orphan contradictions: 0

Model: agent-lifecycle-state-model (9 claims)
  Coverage: 44% (4/9 annotated)
  Risk: 0 (annotated claims are all replicated)
  Orphan contradictions: 1 (probe 2026-02-24 contradicts Invariant 7, model not updated)
```

**Detection heuristics:**
- Claims identified by: lines starting with `### Claim` or `### Invariant` in model.md
- Evidence quality identified by: `**Evidence quality:**` line within claim section
- Probe contradictions identified by: `[x] **Contradicts**` in probe Model Impact section
- Model update detected by: git log --follow on model.md, compared to probe date

This is mechanical, runs on .kb/ directly, no projection DB.

#### Phase 2: Gate Enhancement (orch-go)

Extend `GateProbeModelMerge` in `pkg/verify/check.go`:

Current behavior: gate fires when probes have contradicts/extends verdicts and model.md wasn't updated.

New behavior (additional check): when a probe's Model Impact includes `contradicts`, verify that the model.md update includes an evidence quality annotation change. This doesn't need to be perfect — it checks for the presence of `**Evidence quality:**` text in the git diff of model.md within the session's commits. If model.md was updated but evidence quality wasn't mentioned, emit a warning (not a block).

This is advisory, following the accretion gate pattern: warn first, evaluate whether to promote to blocking based on harness audit fire rates.

#### Phase 3: kb context Confidence Injection (kb-cli)

When `kb context` returns model claims, include the evidence quality annotation in the output. Currently, claims are returned as freeform text. Enhancement: if a claim section includes `**Evidence quality:**`, include it in the `kb context` output so agents see confidence level in spawn context.

This requires no agent behavioral compliance — it's just data the agent sees. The orchestrator-skill model shows that knowledge content transfers reliably (Claim 2); evidence quality annotations are knowledge content.

### Alternative Approaches Considered

**Option B: Claim Registry with Citation Graph**
- **Pros:** Precise tracking of which artifacts cite which claims; automatic downstream notification
- **Cons:** Requires projection DB (architecturally prohibited); requires structured claim extraction from freeform markdown; citation syntax needs behavioral compliance from agents
- **When to use instead:** If freeform markdown scanning proves too fragile for `kb audit provenance`

**Option C: kb quick `confidence` Field + Automatic Constraint Creation**
- **Pros:** Quick entries are structured (JSONL), easy to query; could auto-create constraints when evidence quality drops
- **Cons:** Quick entries are supplementary knowledge, not canonical sources; models are the canonical source for claims. Adding confidence to quick entries solves a different problem (quick entry reliability) not the core problem (model claim propagation)
- **When to use instead:** As a complementary mechanism — when a model claim is downgraded, create a kb quick constraint. Already done manually for dilution curve (kb-cf0431). Could be automated in Phase 2 as an optional enhancement.

**Option D: Confidence Levels on kb quick entries only**
- **Pros:** Simple schema extension; already has `authority` field as precedent
- **Cons:** Quick entries aren't the canonical source for model claims. The dilution curve problem is about model claims propagating incorrectly, not about quick entry confidence. This would be solving the wrong problem.
- **When to use instead:** Never as standalone — only as complement to the model-centric approach

**Rationale for recommendation:** Option A (Three-Layer Confidence Enforcement) directly addresses the root cause (model claims not reflecting evidence quality), uses existing infrastructure (probes, gates, kb context), and requires no behavioral compliance. Options B-D either violate architectural constraints, solve adjacent problems, or require behavioral compliance that we know fails.

---

### Implementation Details

**What to implement first:**
- `kb audit provenance` command — provides immediate visibility into confidence gaps across all models (27 models, ~150+ claims to scan). This is the highest-value, lowest-risk change.
- Evidence quality annotation convention document — write a 1-page guide in `.kb/guides/` explaining the 4-level taxonomy and where to use it. This enables consistent adoption without requiring code.

**Things to watch out for:**
- ⚠️ Claim detection heuristic (`### Claim` / `### Invariant` prefix) may miss claims with non-standard headings. Need to audit current model formats before implementing.
- ⚠️ The `GateProbeModelMerge` gate extension checks git diff for `Evidence quality` text — this could false-negative if the agent uses different wording. Advisory-only (warning) mitigates this risk.
- ⚠️ Evidence quality annotations are still freeform text in markdown. There's no enforcement that the annotation value is one of the 4 canonical levels. This is acceptable for V1 — the audit command can detect non-standard annotations.

**Areas needing further investigation:**
- How many of the 27 existing models have claim-level structure (`### Claim N` or `### Invariant N`) vs freeform sections? This determines scan coverage.
- Should `kb audit provenance` also scan investigations for citation of specific claims? (V2 enhancement, not V1)
- Should the periodic audit run via daemon (like agreements check) or remain manual?

**Success criteria:**
- ✅ `kb audit provenance` detects the dilution curve case retroactively (orphan contradiction: probe contradicts claim, model updated late)
- ✅ Future probe-to-model merges that downgrade evidence quality trigger a completion gate warning
- ✅ Agents consuming models via `kb context` see evidence quality annotations in spawn context
- ✅ No behavioral compliance required — all enforcement is infrastructure

---

## Defect Class Exposure

- **Class 5 (Contradictory Authority Signals):** This is the core problem. Multiple artifacts disagree about claim confidence. Mitigation: single canonical derivation via model as authority + gates to keep models accurate.
- **Class 3 (Stale Artifact Accumulation):** Models with outdated evidence quality annotations after probe contradictions. Mitigation: `kb audit provenance` periodic sweep detects staleness.

---

## References

**Files Examined:**
- `.kb/models/orchestrator-skill/model.md` - Evidence quality taxonomy in practice, Claims 1-6 with varying confidence
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-orchestrator-skill-investigation-cluster-contradiction-analysis.md` - DC-1: the dilution curve propagation failure case
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-evidence-inventory-orchestrator-skill-cluster.md` - 67 claims with evidence quality categorization
- `pkg/verify/check.go` - Gate orchestration and GateProbeModelMerge implementation
- `pkg/spawn/gates/agreements.go` - Periodic audit gate pattern
- `kb-cli/cmd/kb/quick.go` - QuickEntry schema with unused ref_count/last_ref fields
- `kb-cli/cmd/kb/context.go` - kb context search and surfacing
- `.kb/models/defect-class-taxonomy/model.md` - Defect Class 5 pattern

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-11-inv-task-orchestrator-skill-design-tension.md` - Design tension mapping that motivated this work
- **Constraint:** `kb-cf0431` - Dilution curve thresholds marked as unreplicated hypotheses
- **Model:** `.kb/models/orchestrator-skill/model.md` - The model that demonstrates both the problem and the organic solution

---

## Investigation History

**2026-03-12:** Investigation started
- Initial question: How should confidence propagate through the kb system?
- Context: orchestrator-skill exploration (orch-go-j1f7b) found caveats don't propagate downstream — dilution curve's replication failure noted once, then 4 artifacts treated as established

**2026-03-12:** 5 decision forks identified and navigated
- Fork 1 (where confidence lives): model claims + quick entries, model is canonical
- Fork 2 (taxonomy): 4-level evidence quality (replicated, single-source, analytical, assumed)
- Fork 3 (auto vs human): detection automatic, action human-judged (advisory)
- Fork 4 (enforcement layer): three layers following existing pattern
- Fork 5 (probe interaction): probes surface change, merge gate enforces model update

**2026-03-12:** Investigation completed
- Status: Complete
- Key outcome: Three-layer confidence enforcement design — evidence quality annotations, probe merge gate enhancement, provenance audit command. Composes existing infrastructure, requires no behavioral compliance.
