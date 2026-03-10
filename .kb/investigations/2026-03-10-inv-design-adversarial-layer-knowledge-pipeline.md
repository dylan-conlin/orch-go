## Summary (D.E.K.N.)

**Delta:** Designed a three-gate adversarial architecture for the knowledge pipeline that would have caught the knowledge-physics overclaim by mechanically blocking vocabulary inflation, self-referencing evidence, and publication without external review.

**Evidence:** Post-mortem analysis of 5 failure instances (knowledge-physics model, harness-engineering model, coordination demo, falsifiability probe, publication plan) — all caught by Codex in 30 seconds, none caught by internal agents across weeks of work.

**Knowledge:** The fundamental failure is that AI agents optimize for coherence and building on premises. Every step in the pipeline (investigate → model → probe → publish) amplifies — nothing subtracts or challenges. The only proven corrective is structurally external review (different model family, blinded to theoretical apparatus). Advisory reminders don't work — the system's own agents built the theory and cannot see outside its framing.

**Next:** Route to implementation. Three phases: (1) evidence source tagging in probe/model templates, (2) `kb review` command with static adversarial prompt and external model, (3) spawn gate blocking publication skills without review artifact.

**Authority:** architectural — Cross-component design (kb CLI, spawn gates, skill system, new governance artifact type). Strategic aspect: whether to require human-in-loop for publication is a value judgment for Dylan.

---

# Investigation: Design Adversarial Layer for Knowledge Pipeline

**Question:** What gate architecture would have caught the knowledge-physics overclaim before publication, and how do we make it mechanical (not advisory), external (not self-referencing), and itself resistant to the closed-loop failure it prevents?

**Started:** 2026-03-10
**Updated:** 2026-03-10
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — recommendations ready for orchestrator review
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md` | extends | Yes — all 5 failure instances verified | None |
| `.kb/threads/2026-03-10-validation-gap-one-person-one.md` | extends | Yes | None |
| `.kb/models/knowledge-physics/model.md` (Gate Deficit section) | extends | Yes — all 6 ungated transitions verified | None |
| `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` | pattern source | Yes — three-layer enforcement as architectural precedent | None |
| `.kb/plans/2026-03-10-knowledge-physics-publication.md` | extends | Yes — plan already flipped to validate-then-publish | None |

---

## Findings

### Finding 1: The Pipeline Has Zero Subtractive Steps

**Evidence:** Tracing the knowledge pipeline end-to-end:

| Step | Action | Subtractive? |
|------|--------|-------------|
| `kb create investigation` | Creates investigation file | No — additive |
| Agent investigates | Fills investigation with findings | No — additive |
| `kb create model` | Creates model from investigations | No — synthesizes but never challenges |
| Agent writes probes | Tests model claims | **Appears subtractive** but see Finding 2 |
| Probe-to-model merge | Updates model with probe findings | No — extends or confirms, rarely contradicts |
| Publication skill | Writes blog post from model | No — presents model claims as validated |

Every transition amplifies. The probe step appears subtractive (testing claims) but in practice operates within the model's framing (see Finding 2).

**Source:** Pipeline traced through: skill templates in `skills/src/`, `kb create` commands, probe-to-model merge protocol in worker-base skill, `.kb/models/knowledge-physics/probes/` directory.

**Significance:** The pipeline is structurally incapable of catching overclaims because no step is designed to challenge, reduce, or reject. A subtractive gate must be injected — it won't emerge from the existing architecture.

---

### Finding 2: Probes Are Framing-Confirmatory, Not Adversarial

**Evidence:** The falsifiability probe (`.kb/models/knowledge-physics/probes/2026-03-10-probe-falsifiability-counterexamples.md`) demonstrates the failure mode precisely:

1. The probe asked: "Can we find systems meeting all four conditions that do NOT exhibit accretion?"
2. The probe accepted the four conditions as the analytical frame
3. Every counterexample was evaluated against the theory's own criteria
4. When counterexamples didn't survive, the probe concluded the theory was confirmed
5. The probe even "discovered" a fifth condition — extending the theory rather than challenging it
6. The probe produced a formula (`accretion_risk = f(...)`) with no measured variables or units

The probe was honest and rigorous within its frame. But the frame was the problem. The right question was never asked: "Is this framework saying anything that existing governance/coordination/institutional-memory literature doesn't already say?"

Codex asked this question in 30 seconds and answered: "No."

**Source:** Probe file analysis, Codex review output documented in `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md`.

**Significance:** Probes cannot serve as adversarial gates because they accept the model's premises. An adversarial gate must operate OUTSIDE the model's framing — it must receive the claims and evidence WITHOUT the theoretical vocabulary, and assess whether the claims are novel relative to existing knowledge.

---

### Finding 3: Vocabulary Inflation Has a Mechanical Signature

**Evidence:** Comparing knowledge-physics vocabulary to existing concepts:

| Novel Term | Existing Concept | Information Lost by Deflation |
|-----------|-----------------|-------------------------------|
| "Knowledge physics" | Governance / coordination costs / commons management | None (Codex confirmed) |
| "Substrate-independent accretion dynamics" | "Uncoordinated shared resources degrade" (Ostrom 1990) | None |
| "Attractor taxonomy" | Package design / modular architecture | Agent-specific vocabulary only |
| "Gate deficit" | Missing CI/CD enforcement / missing governance | None |
| `accretion_risk = f(amnesia × complexity / coordination)` | "More coordination → less degradation" | Formula shape (no measured variables) |

The mechanical signature: **a term is inflated when replacing it with existing terms does not change the meaning of the claim.** This is testable:
- Claim: "Knowledge exhibits substrate-independent accretion dynamics when amnesiac agents contribute"
- Deflated: "Shared resources degrade when contributors don't coordinate"
- Information lost: Zero → vocabulary inflation confirmed.

**Source:** Codex review output, model.md vocabulary analysis.

**Significance:** Vocabulary inflation is mechanically detectable via a "deflation test." This is the core of Gate 2.

---

### Finding 4: Self-Referencing Evidence Has a Mechanical Signature

**Evidence:** All evidence in the knowledge-physics model comes from:
- orch-go's own knowledge corpus (orphan rates, model counts, probe counts)
- orch-go's own code metrics (daemon.go bloat, file growth)
- Web research conducted by orch-go's own agents (within the theory's framing)

The web research is partially external, but the search queries were framed by the theory ("event sourcing schema drift", "blockchain state bloat" — looking for accretion in other substrates, not looking for existing theories that already explain the observations).

Mechanical signature: **evidence is self-referencing when the evidence source and the claim source are the same system.** Trackable via source classification:

| Source Type | Example | Independence Level |
|------------|---------|-------------------|
| Internal measurement | "85.5% orphan rate in our .kb/" | Zero — we built the substrate and the measurement |
| Directed external search | "Knight Capital case supports theory" | Low — we chose confirming cases |
| Independent external | "External user reported same pattern" | High — observation we didn't control |
| Adversarial external | "Codex says this is repackaged governance" | High — explicitly seeking disconfirmation |

**Source:** Evidence analysis of model.md, probe files, thread files.

**Significance:** Evidence independence is mechanically measurable. Every model should track the ratio of internal to external evidence, and publication should require a minimum threshold of independent or adversarial evidence.

---

### Finding 5: The Existing Three-Layer Enforcement Pattern Applies

**Evidence:** The hotspot enforcement system uses three layers:

| Layer | Mechanism | Trigger |
|-------|-----------|---------|
| Layer 1 — Spawn gate (blocking) | Blocks feature-impl on CRITICAL files | At spawn time |
| Layer 2 — Daemon escalation | Routes to architect | At triage time |
| Layer 3 — Spawn context advisory | Injects info for agent awareness | At spawn time |

This pattern maps directly to knowledge pipeline gates:

| Layer | Knowledge Equivalent | Trigger |
|-------|---------------------|---------|
| Layer 1 — Publication gate (blocking) | Blocks publication without adversarial review artifact | At publication skill spawn |
| Layer 2 — Model claim escalation | Routes novelty claims to external review | At model creation/update |
| Layer 3 — Evidence source advisory | Injects endogeneity warning into probe context | At probe spawn |

**Source:** `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`, `pkg/spawn/gates/` directory.

**Significance:** The enforcement architecture already exists as a proven pattern. The knowledge pipeline gates should mirror it rather than invent a new enforcement mechanism.

---

### Finding 6: Gate Meta-Stability Requires Structural Independence

**Evidence:** Constraint 5: "the gate design itself must not be susceptible to the closed-loop failure mode it's trying to prevent." Analysis:

| Reviewer Type | Independence | Same-Loop Risk |
|----------|-------------|----------------|
| Same Claude instance, adversarial prompt | Low | Same model family optimizes for same coherence patterns |
| Different Claude instance, no context | Medium | Same training data, same biases, different context |
| Different model family (Codex/GPT/Gemini) | High | Different training, different biases, structural independence |
| Human expert | Highest | Not mechanical, doesn't scale |

The post-mortem proves that Codex (different family) caught what Claude agents couldn't. However, the gate must not become dependent on a single external provider.

**Additional meta-stability requirements:**
1. The adversarial prompt must be stored as a governance file (not agent-modifiable)
2. The prompt must be seeded from REAL caught overclaims (empirically grounded), not designed from theory
3. The gate's effectiveness must be periodically measured against known overclaim test cases
4. Multiple external models should be rotated to prevent single-model blind spots

**Source:** Post-mortem analysis, closed-loop-risk thread instances 3-5.

**Significance:** The default reviewer must be a non-Anthropic model. Claude can serve as fallback but the review artifact must flag "same-family review" as a weaker signal.

---

## Synthesis

**Key Insights:**

1. **The pipeline failure is structural, not a mistake.** AI agents optimize for coherence — they build on premises, extend frameworks, confirm theories within their framing. This isn't a bug; it's a property of how language models work. Any pipeline that only uses same-model agents will amplify. The fix must be structural (inject external perspective mechanically) not behavioral (tell agents to be more critical).

2. **Three distinct failure modes require three distinct checks.** Vocabulary inflation (renaming known concepts), evidence endogeneity (self-referencing loops), and framing blindness (not seeing outside the theory) are independent failure modes. A single "adversarial review" gate would catch some but miss others. Each needs a specific mechanical test.

3. **The gate's meta-stability requires structural independence.** If the adversarial review prompt is written by agents in the system, those agents share the system's blind spots. The prompt must be (a) seeded from concrete examples of caught overclaims (empirically grounded), (b) stored as a governance file (not agent-modifiable), and (c) periodically re-validated against known overclaims.

**Answer to Investigation Question:**

A three-gate architecture, mirroring the existing three-layer hotspot enforcement, would have caught the knowledge-physics overclaim at multiple points:

- **Gate 1 (Evidence Endogeneity)** would have flagged that >90% of knowledge-physics evidence was internal, preventing the model from reaching "validated" status without independent evidence.
- **Gate 2 (Vocabulary Deflation)** would have detected that "substrate-independent accretion dynamics" deflates to "uncoordinated systems degrade" with zero information loss, flagging vocabulary inflation.
- **Gate 3 (External Adversarial Review)** would have sent the model to Codex before publication, catching the framing blindness that internal agents structurally cannot see.

The existing validate-then-publish flip (from the publication plan post-mortem) is the correct behavioral response. This architecture makes it mechanical.

---

## Structured Uncertainty

**What's tested:**

- ✅ External model (Codex) catches overclaims that internal agents miss (verified: 5 instances in post-mortem, all caught in <60 seconds)
- ✅ Vocabulary inflation has a mechanical signature (verified: deflation test on knowledge-physics terms shows zero information loss)
- ✅ Evidence endogeneity is trackable (verified: source analysis of knowledge-physics model shows >90% internal evidence)
- ✅ Three-layer enforcement pattern works for code (verified: hotspot enforcement operational since 2026-02-26)

**What's untested:**

- ⚠️ Whether automated vocabulary deflation tests produce actionable results at scale (not benchmarked — the deflation test was done manually here)
- ⚠️ Whether non-Anthropic models reliably catch overclaims across different topics or if Codex was effective on this specific case
- ⚠️ Whether evidence source tagging changes agent behavior or just adds ceremony (no prior data)
- ⚠️ False positive rate — how often the gates block legitimate novel claims

**What would change this:**

- If Codex/GPT fails to catch overclaims on future test cases → the gate needs human-in-loop, not just model-swap
- If vocabulary deflation produces too many false positives → need calibration or human override
- If evidence source tagging is gamed by agents who search for confirming external evidence → tagging alone is insufficient, need adversarial search requirement

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Three-gate architecture for knowledge pipeline | architectural | Cross-component (kb CLI, spawn gates, skill system, governance files) |
| Use non-Anthropic model for Gate 3 | architectural | External dependency selection, cost implications |
| Static adversarial prompt as governance file | architectural | New governance artifact pattern |
| Whether to require human approval for publication | strategic | Irreversible (published content can't be unpublished), value judgment |

### Recommended Approach ⭐

**Three-Gate Adversarial Architecture** — Mirror the three-layer hotspot enforcement for knowledge pipeline transitions, with each gate targeting a specific failure mode.

**Why this approach:**
- Each gate catches a distinct failure mode proven by the post-mortem
- Follows existing three-layer enforcement pattern (no new architectural paradigm)
- Gates are mechanical (blocking, not advisory) per constraint 1
- External model provides structural independence per constraints 2 and 5
- Phased implementation allows validation at each step

**Trade-offs accepted:**
- External model dependency (API cost and availability)
- Added friction to knowledge workflow
- Potential false positives blocking legitimate novel claims — mitigated by override path and calibration period

### The Three Gates

#### Gate 1: Evidence Endogeneity Check

**Where:** At model creation/update (via `kb create model`, probe-to-model merge)
**What:** Tag each piece of evidence by source independence level
**Block when:** >80% of evidence supporting novelty claims is internal (endogenous)

**Template addition (probe and model files):**
```yaml
evidence_sources:
  internal: []     # From this system's measurements/artifacts
  directed: []     # External sources found by searching FOR confirming evidence
  independent: []  # External sources found without theory-confirming search
  adversarial: []  # External sources found by searching for DISconfirming evidence
```

**CLI integration:**
```bash
kb reflect --type endogeneity
# Output: Models with >80% internal evidence flagged
```

**Rollout:** Advisory first (flag in `kb reflect`), upgrade to blocking after calibration.

#### Gate 2: Vocabulary Deflation Test

**Where:** At model creation/update when novelty claims are made
**What:** For each novel term, test whether replacing it with existing terms loses information
**Block when:** Novel terms deflate to existing concepts with zero information loss

**Mechanical test:**
1. Extract sentences containing novel terminology from model
2. Replace novel terms with common equivalents (governance, coordination, enforcement, etc.)
3. Present both versions to external model: "Does the original say anything the simplified version doesn't?"
4. If answer is "no" → vocabulary inflation confirmed

**Novel term detection heuristics:**
- Terms not found in standard CS/governance/organizational literature
- Terms with capitalized-concept formatting ("Knowledge Physics", "Substrate Governance")
- Formula expressions where variables lack units or measurement procedures

#### Gate 3: Pre-Publication External Adversarial Review

**Where:** Before any content reaches external publication (blog, docs, talks)
**What:** Send claims and evidence to external model (non-Anthropic) with static adversarial prompt
**Block when:** External review identifies overclaiming, self-referencing evidence, or vocabulary inflation

**The adversarial review prompt** (stored as `.kb/governance/adversarial-review-prompt.md`, governance-protected):

```markdown
# Adversarial Review Prompt

You are reviewing claims made by a system that may be overclaiming novelty.
The system uses AI agents that optimize for coherence and tend to escalate
observations into frameworks.

For each claim below, answer:

1. NOVELTY: List 3+ existing concepts/frameworks that describe the same thing.
   What does this claim add beyond them? If nothing, say "vocabulary inflation."

2. EVIDENCE: Is the evidence independent of the claim? Could this evidence
   exist without the theory? If all evidence comes from the same system that
   generated the theory, say "endogenous evidence."

3. INFORMATIVENESS: Would this surprise a practitioner who has read Ostrom,
   Conway, Brooks, or standard governance literature? If not, say "known result."

4. OVERCLAIMING: Where does the scope of claims exceed the scope of evidence?
   Flag any claims about "universality" or "substrate-independence" supported
   only by evidence from one system.

5. FORMULA CHECK: For any mathematical expressions, do the variables have
   defined units and measurement procedures? If not, say "formula-shaped prose."

Output a structured verdict:
- PASS: Claims are novel, evidence is independent, scope matches evidence
- CONDITIONAL: Some claims need scoping/deflation but core insight is valid
- FAIL: Claims are primarily restatements with inflated vocabulary
```

**Implementation as `kb review` command:**
```bash
kb review model knowledge-physics --external
# 1. Extracts claims from model
# 2. Strips theoretical vocabulary (produces "deflated" version)
# 3. Sends both to external model with adversarial prompt
# 4. Stores review artifact in .kb/reviews/YYYY-MM-DD-review-{name}.md
# 5. Returns structured verdict

kb review publication coordination-demo-post.md --external
# Same flow for publication drafts
```

**Spawn gate integration:**
```go
// pkg/spawn/gates/publication_review.go
func CheckPublicationReview(target string) GateResult {
    // 1. Find most recent review artifact for target
    // 2. Check age (<30 days)
    // 3. Check verdict (PASS or CONDITIONAL)
    // 4. If no artifact or FAIL → block spawn
}
```

Override: `--force-publish --review-ref <artifact-path>` (mirrors `--force-hotspot --architect-ref`)

### Gate Meta-Validation (Constraint 5 Compliance)

The gate's own effectiveness must be measured to prevent it from becoming a closed loop:

1. **False negative tracking:** When post-publication criticism identifies overclaims the gate missed, log in `.kb/governance/gate-failures.jsonl`

2. **Periodic re-validation (90-day cycle):** Run the adversarial prompt against KNOWN overclaims (knowledge-physics is test case 1). If the prompt fails to catch known overclaims, it needs updating.

3. **Prompt governance:** The adversarial review prompt is a governance file. Modifications require running modified prompt against all known test cases and verifying catch rate ≥ previous version.

4. **Multi-model rotation:** Rotate between Codex, GPT, and Gemini for reviews. Track which models catch which issue types. Remove models from rotation for issue types they consistently miss.

### Alternative Approaches Considered

**Option B: Single Publication Gate Only**
- **Pros:** Simpler, one gate instead of three
- **Cons:** Late detection — months of model-building accumulate overclaims before the gate fires. Knowledge-physics model accreted overclaims across 6+ sessions before anyone checked.
- **When to use instead:** If Phase 1-2 prove too costly in ceremony.

**Option C: Human-in-Loop at Every Model Update**
- **Pros:** Highest quality review
- **Cons:** Not mechanical, doesn't scale, creates bottleneck. Violates constraint 1.
- **When to use instead:** Layer on top of mechanical gates for strategic publications (Phase 5).

**Option D: Adversarial Claude Agent (same family, adversarial prompt)**
- **Pros:** No external API dependency, lower latency
- **Cons:** Same model family shares coherence biases. Post-mortem proved Claude agents couldn't catch the overclaim. Violates constraint 5.
- **When to use instead:** As fallback when external models unavailable. Must flag as "same-family review (weaker signal)."

**Rationale for recommendation:** Option A (Three-Gate) catches failures at multiple pipeline stages, uses the proven three-layer enforcement pattern, and provides structural independence. Phased implementation allows validation.

---

### Implementation Details

**Implementation sequence:**

| Phase | Deliverable | Dependency | Effort |
|-------|------------|------------|--------|
| 1 | Evidence source tagging in probe/model templates + `kb reflect --type endogeneity` | None | Low — template changes + reflect extension |
| 2 | `kb review` command with external model + adversarial prompt governance file | External model API access | Medium — new command + API integration |
| 3 | Spawn gate for publication skills requiring review artifact | Phase 2 | Low — mirrors existing hotspot gate pattern |
| 4 | Gate meta-validation (test case library + periodic re-validation) | Phase 2 | Low — governance process, not code |

**Things to watch out for:**
- ⚠️ External model API costs — `kb review` calls should be budgeted
- ⚠️ The adversarial prompt is seeded from ONE case (knowledge-physics); may need tuning for different overclaim patterns
- ⚠️ Evidence tagging could become ceremony if agents fill it mechanically — consider automated verification
- ⚠️ False positive calibration — gates may block legitimate novel claims. Override path must have accountability (logged)

**Defect class exposure:**
- Class 1 (Filter Amnesia): Endogeneity check must exist in ALL model update paths
- Class 5 (Contradictory Authority Signals): External review verdict may conflict with internal probe verdicts — external must be authoritative for publication
- Class 0 (Scope Expansion): Publication gate must have explicit skill allowlist, not heuristic

**Success criteria:**
- ✅ The knowledge-physics model fails Gate 1 (>90% internal evidence)
- ✅ The knowledge-physics model fails Gate 2 ("substrate-independent accretion dynamics" deflates to "uncoordinated systems degrade" with zero info loss)
- ✅ Gate 3 catches the overclaim when run against the model (replicating the Codex result)
- ✅ No publication reaches the blog without a review artifact <30 days old
- ✅ Gate meta-validation catches prompt degradation within 90-day cycle

---

## The Meta-Question: Is This Design Itself Susceptible?

Constraint 5 requires the gate not be susceptible to the closed-loop failure it prevents. This design was created by a Claude agent. Could it have the same framing blindness?

**Mitigations built into the design:**

1. **The adversarial prompt is seeded from real output, not designed from theory.** The Codex review that actually caught the overclaim provides the empirical basis — not a theoretical framework about what overclaims look like.

2. **The external model is structurally different.** Using Codex/GPT/Gemini, not another Claude instance.

3. **The gate has self-measurement.** False negative tracking and periodic re-validation provide feedback on whether the gate works.

4. **The gate design is falsifiable.** If the next overclaim slips through all three gates, the design is wrong and needs revision.

**Residual risk:** This design assumes different model families have sufficiently different biases to catch each other's blind spots. If all large language models share coherence-optimization tendencies, cross-model review provides less independence than assumed. Mitigation: track cross-model agreement rates. If models consistently agree "PASS" on content humans later identify as overclaimed, the approach needs reassessment — possibly requiring human reviewers despite scaling cost.

---

## References

**Files Examined:**
- `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md` — Post-mortem with 5 failure instances
- `.kb/threads/2026-03-10-validation-gap-one-person-one.md` — Validation gap analysis
- `.kb/models/knowledge-physics/model.md` — The overclaimed model (Gate Deficit section)
- `.kb/models/knowledge-physics/probes/2026-03-10-probe-falsifiability-counterexamples.md` — Framing-confirmatory probe
- `.kb/plans/2026-03-10-knowledge-physics-publication.md` — Publication plan (validate-then-publish)
- `.kb/sessions/2026-03-10-debrief.md` — Session debrief with failure recognition
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Three-layer enforcement pattern (precedent)

**Related Artifacts:**
- **Decision:** Three-Layer Hotspot Enforcement — architectural pattern this design mirrors
- **Thread:** Closed Loop Risk — primary failure analysis motivating this design
- **Constraint (CLAUDE.md):** "Models and publications must survive external adversarial review" — this design makes the constraint mechanical

---

## Investigation History

**2026-03-10:** Investigation started
- Initial question: Design adversarial gate architecture for knowledge pipeline
- Context: Post-mortem of knowledge-physics overclaim — Codex caught in 30s what internal agents couldn't catch across weeks

**2026-03-10:** Failure anatomy mapped
- 5 failure instances documented in closed-loop-risk thread
- 3 mechanical signatures identified (vocabulary inflation, evidence endogeneity, framing confirmation)
- Existing three-layer enforcement pattern identified as architectural precedent

**2026-03-10:** Three-gate architecture designed
- Gate 1: Evidence Endogeneity Check (at model creation/update)
- Gate 2: Vocabulary Deflation Test (at model creation/update)
- Gate 3: Pre-Publication External Adversarial Review (at publish)
- Meta-validation: Gate self-measurement (ongoing)

**2026-03-10:** Investigation completed
- Status: Complete
- Key outcome: Three-gate architecture with 4-phase implementation, mirroring existing hotspot enforcement pattern
