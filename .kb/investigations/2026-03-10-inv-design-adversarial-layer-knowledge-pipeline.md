## Summary (D.E.K.N.)

  * [ ] **Delta:** The fix is a separate challenge lane with hard publication blockers for endogenous evidence, vocabulary inflation, and unsupported novelty language.

**Evidence:** The closed-loop risk thread shows probes and publications amplifying internal framing until an external reviewer punctured them immediately, while the current completion pipeline exposes no adversarial stage and existing KB tooling measures claims/orphans but not external validity.

**Knowledge:** Internal models remain useful, but public-theory publication needs asymmetric gates that validate evidence class and novelty claims rather than rewarding coherence.

**Next:** Implement the publication gate first, then push the same claim/canonicalization requirements upstream into models.

**Authority:** architectural - The design adds a new pipeline stage, new KB artifact types, and new publish-time blocking behavior across investigations, models, probes, and publications.

---

# Investigation: Design Adversarial Layer Knowledge Pipeline

**Question:** What gate architecture would mechanically prevent investigate -> model -> probe -> publish from escalating internally coherent observations into overclaimed public theory?

**Started:** 2026-03-10
**Updated:** 2026-03-10
**Owner:** Orchestrator System
**Phase:** Complete
**Next Step:** Implement publication gate from [docs/designs/2026-03-10-adversarial-gate-knowledge-pipeline.md](../../docs/designs/2026-03-10-adversarial-gate-knowledge-pipeline.md)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md` | deepens | yes | none |
| `.kb/plans/2026-03-10-knowledge-physics-publication.md` | confirms | yes | yes - plan still assumed theory publication readiness before the adversarial pattern was fully named |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: The current pipeline has verification and advisories, but no challenge stage

**Evidence:** The completion pipeline is explicitly `resolveCompletionTarget -> executeVerificationGates -> runCompletionAdvisories -> executeLifecycleTransition`, and the advisory stage is framed as side-effects rather than independent challenge. Existing `orch kb` commands expose ask/extract/claims/orphans workflows, but nothing that blocks publication on external review or lineage validity.

**Source:** `cmd/orch/complete_pipeline.go:1-5`, `cmd/orch/complete_pipeline.go:234-242`, `cmd/orch/kb.go:61-178`

**Significance:** The system can verify artifact presence and run review ceremony, but it cannot mechanically stop internally coherent theory from advancing to publication.

---

### Finding 2: The observed failure mode is vocabulary inflation plus endogenous evidence

**Evidence:** The closed-loop risk thread documents the exact escalation path: real observations become formula-shaped theory, probes run inside the same framing, and external review identifies the result as repackaged governance/software concepts with agent vocabulary. It also records the key self-reference problem: “Both models' evidence is endogenous: one system interpreting itself.”

**Source:** `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md:13-19`

**Significance:** Any gate that only asks for more probing, more rigor, or more internal review will fail, because the problem is not missing effort; it is missing independence and subtraction.

---

### Finding 3: The publication plan already acknowledges external validation need, but not a reusable blocker

**Evidence:** The publication plan now says external validation must happen before the essay, because internal consistency is not external validation, but that plan is a one-off sequencing correction rather than a system-level gate. The repo already has claim and orphan metrics, which means the infrastructure supports machine-readable KB checks; what is missing is a publication-specific validity gate.

**Source:** `.kb/plans/2026-03-10-knowledge-physics-publication.md:69-87`, `.kb/plans/2026-03-10-knowledge-physics-publication.md:121-136`, `cmd/orch/kb.go:133-177`

**Significance:** The project already accepts measurement-oriented KB tooling. The right fix is not “be more careful”; it is a structural gate added to the artifact lifecycle.

---

## Synthesis

**Key Insights:**

1. **Challenge must be a separate lane** - Probes inside a model lane cannot reliably invalidate the framing they inherit, so publication needs a distinct challenge artifact and gate.

2. **Novelty must survive translation** - The fastest way to catch vocabulary inflation is to force coined language back into plain terms, map it to prior art, and block novelty claims when nothing predictive remains.

3. **Evidence class matters more than prose quality** - A gate based on lineage structure and reviewer independence is less susceptible to rhetorical self-confirmation than a gate based on persuasive summaries.

**Answer to Investigation Question:**

The required architecture is an asymmetric `challenge` stage inserted between `probe` and `publish`, with hard blockers based on claim lineage, vocabulary canonicalization, external reviewer independence, and publication-language policy. Models and publications should carry a claim ledger; challenge artifacts should be generated from a fixed template that records blind canonicalization, prior-art mapping, endogenous-evidence findings, and structured severity codes. Publication should fail whenever generalized or novel claims are supported only by internal model/probe loops, whenever coined terms collapse to existing concepts without predictive residue, or whenever strong language such as “physics” or “new framework” is used without passing those checks. This directly addresses the documented failure pattern while avoiding the same trap, because the gate relies on structural invariants and negative-authority external review rather than internal persuasive coherence.

---

## Structured Uncertainty

**What's tested:**

- ✅ The failure mode is real and already documented in project artifacts, including external-model puncture and endogenous evidence diagnosis.
- ✅ The current orchestration pipeline has no explicit challenge stage or publish-time adversarial blocker.
- ✅ The project already supports machine-readable KB analysis patterns (`claims`, `orphans`), so a mechanical gate fits existing system shape.

**What's untested:**

- ⚠️ Whether one external-model challenge is sufficient independence, or whether public-theory claims should require human-external review.
- ⚠️ Whether canonicalization can be linted reliably enough from markdown alone without a small amount of reviewer judgment.
- ⚠️ The exact UX for challenge creation and publication failure messaging.

**What would change this:**

- If implementation shows that challenge artifacts cannot be made machine-readable enough to drive non-zero exit codes, the design would need a sidecar format or stricter markdown schema.
- If prior-art mapping causes excessive false positives on internal-only models, the gate should narrow to publications first and relax model-time enforcement.
- If independent external users consistently produce substantive novelty delta that survives canonicalization, the publication-language policy can be loosened for that claim class.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add a publication-blocking challenge lane with claim-ledger and vocabulary/lineage checks | architectural | It changes the artifact lifecycle and applies across KB tooling, review flow, and publish behavior |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside -> implementation
- Reaches to other components/agents -> architectural
- Reaches to values/direction/irreversibility -> strategic

### Recommended Approach ⭐

**Asymmetric Challenge Lane** - Insert a mechanical `challenge` stage that can block publication by validating evidence lineage, reviewer independence, and novelty-language eligibility.

**Why this approach:**
- It directly targets the two documented root causes: endogenous evidence and vocabulary inflation.
- It uses external perspective as a hard dependency, not a best practice.
- It resists the original failure mode because it validates structure and downgrade/block conditions, not rhetorical confidence.

**Trade-offs accepted:**
- Publication gets slower and more bureaucratic for theory-like claims.
- Some genuinely new concepts will be forced to start life as “working model” language until independent evidence accumulates.

**Implementation sequence:**
1. Add publish-time blockers first: challenge artifact template, publication contract, lineage gate, banned-language gate.
2. Add claim ledger and canonicalization requirements to models so failures appear before publication.
3. Distinguish confirmatory probes from adversarial probes so probes stop being mistaken for independent evidence.

### Alternative Approaches Considered

**Option B: Stronger internal probes only**
- **Pros:** Minimal workflow change; keeps everything inside current investigate/model/probe loop.
- **Cons:** Fails the core requirement because probes still inherit the framing and cannot guarantee external perspective.
- **When to use instead:** Never as the only gate for public-theory publication.

**Option C: Human editorial review only**
- **Pros:** Better taste and domain nuance than LLM-only review.
- **Cons:** Advisory unless encoded as a blocking contract; also unavailable for every artifact.
- **When to use instead:** As an additional requirement for especially high-stakes publications after the mechanical gate exists.

**Rationale for recommendation:** The asymmetric challenge lane is the only option that satisfies all constraints at once: mechanical, external, vocabulary-aware, lineage-aware, and structurally resistant to self-confirming theory inflation.

---

### Implementation Details

**What to implement first:**
- `orch kb gate publish` with non-zero exit on endogenous evidence, missing challenge, or banned novelty language
- `.kb/challenges/` artifact template with severity-code table
- Claim ledger schema for publications

**Things to watch out for:**
- ⚠️ Reviewer-independence metadata must be strict enough that the originating model family cannot satisfy the external gate.
- ⚠️ The gate should downgrade unsupported novelty claims instead of forcing all publications to be abandoned.
- ⚠️ Canonicalization tables need a precise schema or the lint will become fuzzy and gameable.

**Areas needing further investigation:**
- Whether to require a different provider versus merely a different model family for “external model” status
- Whether external human review should be mandatory for public essays that claim general theory
- How to store publication artifacts if the project does not yet have a formal `.kb/publications/` directory

**Success criteria:**
- ✅ A publication supported only by internal model/probe loops fails mechanically.
- ✅ A coined term with no prior-art mapping or novelty delta fails mechanically.
- ✅ A publication cannot use “physics”, “new framework”, or equivalent language unless the gate allows it.

---

## References

**Files Examined:**
- `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md` - Failure pattern and external puncture evidence
- `.kb/plans/2026-03-10-knowledge-physics-publication.md` - Existing plan and validate-then-publish constraint
- `cmd/orch/complete_pipeline.go` - Current pipeline shape and absence of challenge stage
- `cmd/orch/kb.go` - Existing KB metrics/commands that show how machine-readable gates fit the system
- `docs/designs/2026-03-10-adversarial-gate-knowledge-pipeline.md` - Resulting implementation-ready design

**Commands Run:**
```bash
# Identify relevant KB/design/CLI files
rg --files . | rg '(^|/)AGENTS\\.md$|(^|/)README|(^|/)\\.kb/|(^|/)kb|(^|/)orch|(^|/)docs/'

# Verify issue context and claim work
bd ready
bd show orch-go-yvndh
bd update orch-go-yvndh --status in_progress

# Read evidence and code paths
sed -n '1,240p' .kb/investigations/2026-03-10-inv-design-adversarial-layer-knowledge-pipeline.md
sed -n '1,220p' .kb/plans/2026-03-10-knowledge-physics-publication.md
sed -n '1,220p' .kb/threads/2026-03-10-closed-loop-risk-ai-agents.md
sed -n '1,260p' cmd/orch/complete_pipeline.go
sed -n '1,260p' cmd/orch/kb.go
```

**External Documentation:**
- None - this investigation was grounded in project-local artifacts and code.

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md` - Captures the failure mode this design addresses
- **Plan:** `.kb/plans/2026-03-10-knowledge-physics-publication.md` - Shows why external validation became necessary
- **Design:** `docs/designs/2026-03-10-adversarial-gate-knowledge-pipeline.md` - Proposed gate architecture

---

## Investigation History

**2026-03-10 14:49:** Investigation started
- Initial question: Design an adversarial layer for the knowledge pipeline that prevents closed-loop theory escalation
- Context: External review punctured a published overclaim that internal probes and publication readiness failed to catch

**2026-03-10 15:10:** Investigation completed
- Conclusion: Add a separate challenge lane with lineage, vocabulary, external-review, and publication-language blockers
- Output: Design document written at `docs/designs/2026-03-10-adversarial-gate-knowledge-pipeline.md`
