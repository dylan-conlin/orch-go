---
title: "Design: Claims-as-Atoms Infrastructure for Model-Driven Strategy"
status: Complete
created: 2026-03-19
skill: architect
beads_id: orch-go-uq3s6
---

# Design: Claims-as-Atoms Infrastructure for Model-Driven Strategy

**TLDR:** Claims.yaml per model directory makes model claims machine-readable atoms that three consumption points use: orient surfaces edges (tensions, staleness, unconfirmed claims in active areas), daemon generates probe demand driven by staleness + model activity, completion pipeline connects findings to the claim graph. Schema adapts the agent-trust-enforcement testable claims pattern (C1-C17) — already the closest existing implementation. Self-referential validation risk is mitigated by an independence requirement (probes cannot confirm using cited evidence), contradiction asymmetry (contradictions surface louder than confirmations), and staleness decay.

## Design Question

How should claims.yaml per model directory work as the atomic unit of model-driven strategy? Three consumption points: (1) orient surfaces edges, (2) daemon generates probes demand-driven, (3) completion pipeline connects findings to claim graph. Key constraint: models drive inquiry, not implementation — claims generate probes, not spawns.

## What I Tried

- Read thread context (.kb/threads/2026-03-19-model-driven-strategy-models-generate.md)
- Read prior decision: adopt uncontaminated Codex gate design (claim ledger)
- Read existing claims infrastructure: pkg/kbmetrics/claims.go (prose extraction), pkg/kbgate/claims.go (claim-upgrade scanning), .kb/publications/harness-engineering-claim-ledger.yaml (YAML format)
- Read all 4 seed models: architectural-enforcement, agent-trust-enforcement, measurement-honesty, skill-content-transfer
- Read orient_cmd.go and pkg/orient/model_freshness.go for current orient integration
- Read investigation audit of model/probe/investigation claims

## What I Observed

### Observation 1: Agent-trust-enforcement already implements the target pattern

The agent-trust-enforcement model has a "Testable Claims" table (C1-C17) with exactly the fields needed:
- ID, claim text, status (Confirmed/Open/Untested/Partially confirmed)
- Evidence (what supports it)
- Falsification condition (what would disprove it)

This is the closest existing implementation to what claims.yaml needs. The other 3 seed models use "Critical Invariants" (numbered lists in prose) which lack status tracking and falsification conditions.

### Observation 2: Three distinct existing claims systems serve different purposes

| System | Location | Purpose | Limitation for strategy |
|--------|----------|---------|----------------------|
| `pkg/kbmetrics/claims.go` | Extracts from prose via regex | Bloat detection (>30 claims = warning) | No status, no falsification, no machine-readable output |
| `pkg/kbgate/claims.go` | Scans for novelty/causal language | Publication quality gate | Detects overclaims, not claim lifecycle |
| Publication claim ledger | YAML in .kb/publications/ | External-facing claim quality | Per-publication, not per-model; no consumption pipeline |

None of these tracks claim confidence lifecycle or connects to orient/daemon/completion.

### Observation 3: Orient already has model freshness infrastructure

`pkg/orient/model_freshness.go` already:
- Scans all model directories
- Extracts last_updated dates
- Detects stale models (>14 days without probes)
- Surfaces top 3 relevant + up to 2 stale in orient output

Claims.yaml integration would extend this: instead of just "model is stale," surface "claim AE-08 in architectural-enforcement is stale + recent work touched the gates area."

### Observation 4: The self-referential validation risk is real and already documented

Measurement-honesty invariant #5: "Metrics that validate themselves are circular." The displacement thread warns: "wrong knowledge in skill docs WAS already driving agent strategy and producing phantom constraints."

The specific risk: if claims.yaml says "gates are signaling infrastructure" and the daemon generates a probe to test it, the probe agent reads the model (which states the claim), and "confirms" using the same evidence cited in the model. This is the self-validating probe pattern that `pkg/kbgate/claims.go` already detects.

---

## Phase 1: Problem Framing

**Design question:** How to make model claims machine-readable atoms that drive inquiry through three consumption points without creating a self-reinforcing echo chamber.

**Success criteria:**
1. Orient surfaces 3-5 claim-level edges per session (not a full claim dump)
2. Daemon generates probe issues only for stale claims in actively-referenced models
3. Completion pipeline can match findings to relevant claims and update status
4. Self-referential validation is structurally detectable
5. Bootstrap works with 4 seed models without requiring changes to all 39

**Constraints:**
- Models drive inquiry, not implementation (claims → probes, not claims → spawns)
- Must integrate with existing orient/daemon/completion infrastructure
- Must not require prose restructuring of existing models (claims.yaml is an overlay)
- Governance-protected files cannot be modified

**Scope:**
- IN: Claim schema, orient integration design, daemon probe logic, completion claim updates, self-referential validation safeguards
- OUT: Implementation code, migration of all 39 models, UI changes

---

## Phase 2: Exploration (Fork Navigation)

### Fork 1: Claim Schema Design

**Options:**

**A) Adapt publication claim ledger format** (type/scope/evidence/strength from harness-engineering-claim-ledger.yaml)
- Pro: Already accepted via Codex gate decision
- Con: Optimized for external-facing quality gating, not lifecycle tracking. No status field, no falsification condition, no cross-model tension tracking.

**B) Adapt agent-trust-enforcement testable claims format** (C1-C17 with status/evidence/falsification)
- Pro: Already implements the target pattern in one model. Has status tracking, evidence, falsification conditions. Closest to what consumption points need.
- Con: Not all models have claims this structured. The table format doesn't capture cross-model tensions.

**C) New hybrid schema optimized for three consumption points**
- Pro: Purpose-built for orient/daemon/completion needs
- Con: Yet another claims format (third after kbmetrics extraction and publication ledger)

**Substrate consultation:**
- Principle: "Evolve by distinction — when problems recur, ask what are we conflating?" The three existing systems conflate claim quality gating (kbgate), claim count monitoring (kbmetrics), and claim lifecycle tracking (what we need). A separate system is warranted because the purpose is distinct.
- Decision: Adopt uncontaminated Codex gate design (claim ledger) — this accepted the YAML format and type/scope/evidence/strength taxonomy. Extend it, don't replace it.
- Defect class exposure: Class 5 (Contradictory Authority Signals) — if claims.yaml and prose claims in model.md diverge, which is authoritative? Must be clear.

**Recommendation: Option C (hybrid), inheriting from both A and B.**

The publication ledger's taxonomy (type, scope, evidence, strength) provides classification. The testable claims table's lifecycle fields (status, falsification_condition) provide tracking. The hybrid adds what neither has: domain_tags (for matching), last_validated (for staleness), tensions (for cross-model edges).

---

### Fork 2: claims.yaml as Overlay vs Source of Truth

**Options:**

**A) claims.yaml is the source of truth; model.md prose references it**
- Pro: Single authoritative location for claims. No divergence.
- Con: Requires restructuring how models are written. Breaking change for 39 models.

**B) claims.yaml is a machine-readable overlay; model.md prose remains authoritative**
- Pro: No changes to how models are written. claims.yaml extracts and tracks what's already in prose. Can bootstrap incrementally.
- Con: Defect Class 5 risk — claims.yaml and prose can diverge. Need a sync mechanism.

**Substrate consultation:**
- Principle: "Evidence hierarchy — Code is truth; artifacts are hypotheses." By analogy: model.md prose is truth; claims.yaml is structured metadata about that truth.
- Constraint: "Must not require prose restructuring of existing models"

**Recommendation: Option B (overlay).**

claims.yaml is a machine-readable index of claims that already exist in model.md. The prose remains authoritative. Sync is validated by the existing `pkg/kbmetrics/claims.go` — it already extracts claims from prose, so a discrepancy detector can compare kbmetrics extraction against claims.yaml entries.

**The authority rule:** If claims.yaml says "confirmed" but the prose contradicts it, the prose wins and claims.yaml needs updating. This is the same pattern as beads tracking — beads tracks issue status, but the actual state is in the work.

---

### Fork 3: Orient Integration — What to Surface

**Options:**

**A) Surface all claims with status != confirmed**
- Con: Overwhelming. A model with 17 claims (like agent-trust-enforcement) would dump most of them.

**B) Surface only edges — tensions, stale-in-active-area, unconfirmed-in-core**
- Pro: Matches thread design ("not a claim dump — shows where understanding is weakest relative to where we're working")
- Con: Requires defining "active area" (how to match claims to recent work)

**C) Surface a model-level summary with drill-down**
- Pro: Compact default, detailed on request
- Con: Orient is already a one-shot briefing, not interactive

**Recommendation: Option B (edges only).**

Three edge types, each producing at most 2 items in orient output:

1. **Tensions** (cross-model conflicts): Claims in different models that make opposing predictions. Detected by `tensions` field in claims.yaml. Max 2 surfaced per orient.
2. **Stale-in-active-area**: Claim with `last_validated` > 30 days AND `domain_tags` matching keywords in recent spawns/completions (from events.jsonl). Max 2 surfaced.
3. **Unconfirmed core claims**: Claims with `priority: core` AND `confidence: unconfirmed` in models referenced by recent spawns. Max 1 surfaced.

Total orient claim output: 3-5 lines. Fits the existing orient structure as a new section between "Stale Models" and "Divergence Alerts."

---

### Fork 4: Daemon Probe Generation Logic

**Options:**

**A) Generate probes for all stale claims**
- Con: Creates probe spam for dormant models. Violates thread constraint: "Dormant model stale claims don't trigger."

**B) Generate probes demand-driven: stale claim + model activity signal**
- Pro: Matches thread design exactly. Only generates probes when the system is actively working in an area where understanding is outdated.
- Con: Needs "model activity" signal — how to determine if a model is active?

**Model activity signals (available in existing infrastructure):**
- Model name appears in recent `kb context` queries (from spawn context generation)
- Model referenced in SPAWN_CONTEXT.md of recent spawns (extractable from events)
- Model's domain_tags overlap with recent issue keywords (from beads)
- Model updated in last 14 days (from model_freshness.go)

**Recommendation: Option B with two activity signals.**

A claim becomes probe-eligible when ALL of:
1. `confidence` is `unconfirmed` OR `last_validated` > staleness_threshold (30 days)
2. Model has activity signal: updated in last 14 days OR referenced in spawn context in last 7 days
3. `priority` is `core` or `supporting` (not `peripheral`)
4. No existing open probe issue for this claim (dedup by claim ID)

The daemon generates a probe issue via `bd create` with the claim's `falsifies_if` as the investigation question. The probe is an investigation skill spawn, not implementation.

**Probe generation rate:** Max 1 probe per daemon cycle. Probes are low-priority (P3) — they queue behind ready implementation work. This prevents probe generation from consuming all agent capacity.

---

### Fork 5: Completion Pipeline Claim Updates

**Options:**

**A) Automatic claim matching via keyword overlap**
- Con: High false positive rate. An agent completing a feature in the gates area might "touch" architectural-enforcement claims without actually testing them.

**B) Explicit claim references in probe completions only**
- Pro: Only probe-skill agents are expected to confirm/contradict claims. Feature-impl agents don't touch claims.yaml.
- Con: Misses organically discovered confirmations/contradictions from non-probe work.

**C) Hybrid: explicit for probes, advisory for non-probes**
- Pro: Probes update claims authoritatively. Non-probe completions generate suggestions ("this completion's findings may relate to claim AE-08") for orchestrator review.
- Con: More complex

**Recommendation: Option B (explicit probe updates only) for bootstrap.**

Simplest approach that works. When a probe agent completes:
1. The probe's investigation file references the claim ID (e.g., `claim: AE-08`)
2. The probe's verdict (confirms/contradicts/extends) maps to a claim status update
3. `orch complete` reads the probe's verdict and updates claims.yaml

Non-probe completions don't touch claims.yaml. If a non-probe agent discovers something that contradicts a claim, it surfaces as discovered work (a beads issue), not a direct claim update. This avoids the false-matching problem entirely.

**Later evolution:** Once the system proves its value with explicit probe updates, consider adding advisory matching for non-probe completions. But start simple.

---

## Phase 3: Question Generation

### Q1: Who writes the initial claims.yaml for the 4 seed models?

**Authority:** implementation
**Subtype:** factual

The 4 seed models have claims embedded in prose. Someone needs to extract them into claims.yaml. This is a one-time task per model.

**Recommendation:** Spawn 4 investigation agents, one per seed model. Each reads the model.md, extracts claims into claims.yaml, and assigns initial status based on the prose (invariants with evidence → confirmed, assertions without evidence → unconfirmed). This is mechanical work, not judgment — appropriate for investigation skill.

### Q2: How should claims.yaml coexist with the existing kbmetrics claims extractor?

**Authority:** architectural
**Subtype:** judgment

`pkg/kbmetrics/claims.go` extracts claims from prose for bloat detection. claims.yaml is a separate, curated list. They serve different purposes but overlap in surface area.

**Recommendation:** They coexist. kbmetrics continues to count prose claims for bloat detection. claims.yaml is the curated subset that drives strategy. A future enhancement could use kbmetrics extraction as a sync check ("model.md has 45 extractable claims but claims.yaml only tracks 12 — are 33 uncaptured?"). But this is not needed for bootstrap.

### Q3: What happens when a claim is contradicted?

**Authority:** strategic
**Subtype:** judgment

A contradicted claim changes the model. But the model drives agent behavior (via spawn context). If the model is wrong and driving agents, updating it is urgent.

**What changes based on the answer:** If contradictions are treated as P1 (urgent), the daemon escalates immediately. If P2 (normal), they queue with other work. If they require human review, they block until the orchestrator decides.

**Recommendation:** Contradictions become P2 issues labeled `triage:review`, not `triage:ready`. The orchestrator must review a contradicted claim before it's resolved — this prevents autonomous agents from rewriting model understanding without human oversight. This aligns with the thread's key constraint: "models drive inquiry, not implementation."

---

## Phase 4: Synthesis

### The Claim Schema

```yaml
# .kb/models/{name}/claims.yaml
model: architectural-enforcement
version: 1
last_audit: 2026-03-19

claims:
  - id: AE-01
    text: "Gates are signaling infrastructure, not quality gates"
    type: mechanism        # observation | mechanism | generalization | invariant
    scope: local           # local | bounded | universal
    confidence: confirmed  # unconfirmed | confirmed | contested | stale
    priority: core         # core | supporting | peripheral
    evidence:
      - source: "Mar 17 gate effectiveness cohort (529 spawns, 258 with gates)"
        date: 2026-03-17
        verdict: confirms
    last_validated: 2026-03-17
    domain_tags: ["gates", "enforcement", "accretion", "signaling"]
    falsifies_if: "Gate enforcement produces measurably higher individual agent quality than bypass cohort (>10pp difference, N>100)"
    tensions: []
    model_md_ref: "## Summary, ## Critical Invariants #8"

  - id: AE-02
    text: "Enforcement without guidance creates displacement"
    type: mechanism
    scope: local
    confidence: confirmed
    priority: core
    evidence:
      - source: "Exploration orch-go-javy8 (3 investigations, Mar 19)"
        date: 2026-03-19
        verdict: confirms
      - source: "Agent scs-sp-8dm: concurrency gate displaced to spawn_preflight.go"
        date: 2026-03-19
        verdict: confirms
    last_validated: 2026-03-19
    domain_tags: ["hooks", "displacement", "governance", "deny"]
    falsifies_if: "Deny hooks without redirect hints produce architecturally correct placement in >80% of cases"
    tensions:
      - claim: MH-05
        model: measurement-honesty
        type: extends
        note: "Two-gap independence: deny count without displacement tracking is false confidence"
    model_md_ref: "## Why This Fails #4, ## Critical Invariants #9"

  - id: AE-03
    text: "Behavioral constraints dilute at 5+ co-resident items, inert at 10+"
    type: mechanism
    scope: bounded
    confidence: confirmed
    priority: core
    evidence:
      - source: "Orchestrator skill behavioral testing (N=7, Mar 1)"
        date: 2026-03-01
        verdict: confirms
      - source: "Constraint dilution threshold test (N=8, Mar 1)"
        date: 2026-03-01
        verdict: confirms
    last_validated: 2026-03-01
    domain_tags: ["skills", "constraints", "dilution", "behavioral"]
    falsifies_if: "10+ behavioral constraints in a skill document produce compliance rates significantly above bare Claude baseline (>20pp, N>20)"
    tensions:
      - claim: SCT-05
        model: skill-content-transfer
        type: confirms
        note: "Invariant 5 distinguishes attention primers (transfer) from action directives (don't transfer)"
    model_md_ref: "## Critical Invariants #6"
```

### Schema Field Rationale

| Field | Why | Consumed By |
|-------|-----|-------------|
| `id` | Stable reference for probes, tensions, completion pipeline | All three |
| `text` | Human-readable claim statement | Orient (display), daemon (probe question) |
| `type` | Inherited from publication ledger taxonomy | Orient (filtering) |
| `scope` | Inherited from publication ledger taxonomy | Daemon (priority weighting) |
| `confidence` | Lifecycle tracking: unconfirmed → confirmed/contested/stale | Orient (edge detection), daemon (probe eligibility) |
| `priority` | Prevents daemon from probing peripheral claims | Daemon (filtering) |
| `evidence` | Array of evidence entries with dates and verdicts | Completion pipeline (update), self-referential validation check |
| `last_validated` | Staleness detection | Orient (stale-in-active-area), daemon (probe eligibility) |
| `domain_tags` | Keyword matching for activity detection | Orient (match to recent work), daemon (activity signal) |
| `falsifies_if` | Probe question generation | Daemon (creates investigation question from this) |
| `tensions` | Cross-model conflict tracking | Orient (tension edges) |
| `model_md_ref` | Links claim to prose location in model.md | Sync validation, human navigation |

### Confidence Lifecycle

```
unconfirmed ──[probe confirms]──> confirmed ──[time passes]──> stale
     │                               │                           │
     │                               │                           │
     └──[probe contradicts]──> contested    [probe confirms]─────┘
                                     │           (resets to confirmed)
                                     │
                              [orchestrator reviews]
                                     │
                            ┌────────┴────────┐
                            ▼                  ▼
                      updated claim      claim removed
                     (new text/evidence)  (model restructured)
```

Key rules:
- **confirmed → stale**: Automatic. 30 days without new evidence AND model has activity signal.
- **stale → confirmed**: Only via new probe with independent evidence.
- **any → contested**: When a probe contradicts. Creates P2 issue for orchestrator review.
- **contested → updated/removed**: Only via orchestrator decision (not autonomous).

### Orient Integration

New section in orient output, between existing "Stale Models" and "Divergence Alerts":

```
## Knowledge Edges

Tensions:
  - AE-02 vs MH-05: enforcement-without-guidance creates displacement
    BUT deny count alone is false confidence (measurement-honesty)

Stale in active area:
  - AE-03 (skill constraint dilution): last validated 18d ago,
    3 skill-related spawns this week

Unconfirmed core:
  - MH-06 (delete-before-fix as universal principle):
    untested outside orch-go
```

**Implementation integration point:** `cmd/orch/orient_cmd.go` step 6 (model freshness). Add a `collectClaimEdges()` function that:
1. Reads claims.yaml from each model directory
2. Collects tensions across all models
3. Cross-references domain_tags with recent spawn keywords (from events.jsonl)
4. Returns top 5 edges by relevance

### Daemon Probe Generation

**Integration point:** `cmd/orch/daemon_loop.go` Decide phase.

New periodic task: `claimProbeGeneration` (runs once per daemon cycle, after existing periodic tasks).

```
FOR each model WITH claims.yaml:
  IF model has activity signal (updated <14d OR referenced in spawn <7d):
    FOR each claim WHERE:
      - confidence = unconfirmed OR (confidence = confirmed AND last_validated > 30d ago)
      - priority = core OR supporting
      - no open probe issue exists for this claim ID
    GENERATE probe issue:
      bd create "Probe: {claim.text} — {claim.falsifies_if}" \
        --type task -l triage:ready -l skill:investigation \
        -l claim:{claim.id} -l model:{model.name}
    LIMIT 1 probe per cycle
```

**Activity signal detection:** Parse `~/.orch/events.jsonl` for `session.spawned` events in the last 7 days. Extract model references from spawn context metadata. Cross-reference with model names.

### Completion Pipeline Claim Updates

**Integration point:** `cmd/orch/complete_*.go` — after verification, before issue close.

For probe-skill completions only:
1. Read the probe's investigation file
2. Extract claim reference (look for `claim:` field in frontmatter or `Claim:` in body)
3. Extract verdict (confirms/contradicts/extends)
4. Update claims.yaml:
   - `confirms` → update `last_validated`, add evidence entry
   - `contradicts` → set `confidence: contested`, create P2 review issue
   - `extends` → update claim text if minor, or create new claim if substantial
5. Commit claims.yaml changes

**Non-probe completions:** No automatic claim updates. If an agent's SYNTHESIS.md mentions a model claim, the orchestrator can manually update during completion review.

### Self-Referential Validation Safeguards

The measurement-honesty model identifies the core risk: metrics that validate themselves are circular (invariant #5). Three structural safeguards:

**Safeguard 1: Evidence Independence Requirement**

When a probe updates a claim's evidence, the completion pipeline checks: does the new evidence source overlap with existing evidence sources? If the probe's primary evidence is the same data cited in the claim's existing evidence entries, flag as `SELF_VALIDATING_PROBE` (reusing the existing code in `pkg/kbgate/claims.go`).

Concretely: if claim AE-01 cites "Mar 17 gate effectiveness cohort" and a probe "confirms" using the same cohort data, that's not new validation — it's circular reading.

**Safeguard 2: Contradiction Asymmetry**

Orient surfaces contradictions MORE prominently than confirmations. The orient output shows:
- Tensions and contradictions: always shown (up to 2)
- Confirmations: never shown (confirmations are silent — they update last_validated but don't appear in orient)
- Stale claims: shown when in active area (up to 2)

This structural asymmetry prevents the system from creating a confirmation-bias feedback loop where the orchestrator sees "claim confirmed" repeatedly and stops questioning the model.

**Safeguard 3: Staleness Decay**

Claims auto-decay to `stale` after 30 days without new evidence, even if they were previously `confirmed`. This forces re-examination. The decay threshold is shorter than the model staleness threshold (14 days) because claim-level staleness should trigger probes, while model-level staleness triggers broader orient warnings.

**Safeguard 4: External Validation Markers**

Each evidence entry can optionally include `external: true` indicating the evidence comes from outside the orch-go system (human observation, external tool output, production data). Claims with only internal evidence (probes, investigations) are labeled differently in orient output than claims with external validation.

---

## Phase 5: Externalization

### Recommendations

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Claim schema (YAML format per model dir) | implementation | Extends existing accepted YAML pattern (Codex gate decision) |
| Orient integration (claim edges section) | implementation | Extends existing model_freshness.go infrastructure |
| Daemon probe generation (periodic task) | implementation | Extends existing daemon periodic task pattern |
| Completion pipeline claim updates (probe-only) | implementation | Extends existing completion verification pipeline |
| Self-referential validation safeguards | implementation | Applies existing measurement-honesty invariants |
| Bootstrap with 4 seed models | implementation | Mechanical extraction, no judgment needed |
| claims.yaml as overlay (not source of truth) | architectural | Models remain authoritative; claims.yaml is index |
| Contradictions route to orchestrator review | strategic | Prevents autonomous model rewriting |

### Implementation Sequence

**Phase 1: Schema + Bootstrap (1-2 sessions)**
1. Define claims.yaml schema as Go types in new `pkg/claims/` package
2. Write YAML parser/serializer
3. Bootstrap claims.yaml for 4 seed models (can be 4 parallel investigation spawns)

**Phase 2: Orient Integration (1 session)**
4. Add `collectClaimEdges()` to orient_cmd.go
5. Parse claims.yaml files, compute tension/stale/unconfirmed edges
6. Add "Knowledge Edges" section to orient output

**Phase 3: Daemon Probe Generation (1 session)**
7. Add `claimProbeGeneration` periodic task to daemon
8. Implement activity signal detection (events.jsonl parsing)
9. Implement dedup check (no open probe for claim ID)

**Phase 4: Completion Pipeline (1 session)**
10. Add claim update logic to probe completion path
11. Implement evidence independence check
12. Wire contradiction → P2 issue creation

### Defect Class Exposure

| Defect Class | Exposure | Mitigation |
|-------------|----------|------------|
| Class 0 (Scope Expansion) | claims.yaml parsing could widen to non-model dirs | Allowlist scanner: only `.kb/models/*/claims.yaml` |
| Class 5 (Contradictory Authority) | claims.yaml vs model.md prose divergence | Overlay pattern: prose is authoritative, claims.yaml is index. Sync check via kbmetrics |
| Class 6 (Duplicate Action) | Daemon generates duplicate probe for same claim | Dedup by claim ID in open issues |
| Class 3 (Stale Artifact) | Stale claims.yaml after model rewrite | model_md_ref field enables sync validation |

---

## Structured Uncertainty

**What's tested:**
- The agent-trust-enforcement model already implements the target pattern (C1-C17 testable claims with status/evidence/falsification) — verified by reading the model
- Orient already has model freshness infrastructure that can be extended — verified by reading model_freshness.go
- The publication claim ledger YAML format is accepted (decision 2026-03-10) — verified by reading the decision
- Self-referential validation detection already exists in pkg/kbgate/claims.go — verified by reading the code

**What's untested:**
- Whether domain_tags can reliably match claims to recent spawn activity (depends on tag quality)
- Whether 30-day staleness threshold is correct (might be too aggressive or too lenient)
- Whether probe-only claim updates are sufficient or whether non-probe completions will create a blind spot
- Whether the independence check (safeguard 1) can detect circular evidence in practice

**What would change this:**
- If domain_tags prove too coarse for matching, would need semantic matching (more complex)
- If probe generation overwhelms agent capacity, would need tighter filtering or lower priority
- If self-referential validation proves undetectable, would need human-in-the-loop for all claim updates

---

## References

**Files Examined:**
- `.kb/threads/2026-03-19-model-driven-strategy-models-generate.md` — Thread with design context
- `.kb/decisions/2026-03-10-adopt-uncontaminated-codex-gate-design-claim-ledg.md` — Prior YAML format decision
- `.kb/publications/harness-engineering-claim-ledger.yaml` — Publication claim ledger (32 claims)
- `pkg/kbmetrics/claims.go` — Prose claim extraction (bloat detection)
- `pkg/kbgate/claims.go` — Claim-upgrade signal scanning (novelty, self-validation, causal)
- `pkg/orient/model_freshness.go` — Model freshness scanning for orient
- `cmd/orch/orient_cmd.go` — Orient command (825 lines, 13 data collection steps)
- `.kb/models/architectural-enforcement/model.md` — Seed model: 9 invariants, 5 failure modes
- `.kb/models/agent-trust-enforcement/model.md` — Seed model: 17 testable claims (C1-C17)
- `.kb/models/measurement-honesty/model.md` — Seed model: 6 invariants, 6 failure modes
- `.kb/models/skill-content-transfer/model.md` — Seed model: 7 invariants, 4 failure modes
- `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md` — Prior claims audit

**Related Models:**
- `.kb/models/measurement-honesty/model.md` — Invariant #5 (self-validating metrics) directly informs safeguard design
- `.kb/models/architectural-enforcement/model.md` — Invariant #8 (gates as signaling) is a test case for the schema
- `.kb/models/defect-class-taxonomy/model.md` — Defect classes 0, 3, 5, 6 apply to this design

**Decisions Informed:**
- Thread: `.kb/threads/2026-03-19-model-driven-strategy-models-generate.md`
- Prior: `.kb/decisions/2026-03-10-adopt-uncontaminated-codex-gate-design-claim-ledg.md`
