# Session Synthesis

**Agent:** og-arch-design-writing-skill-20mar-875c
**Issue:** orch-go-npm1s
**Duration:** 2026-03-20 → 2026-03-20
**Outcome:** success

---

## Plain-Language Summary

Designed a `technical-writer` skill that treats blog post writing as a compositional correctness problem — the same failure pattern where individually valid components compose into non-functional wholes. The harness engineering blog post had well-written sections but wrong composition (framework before story, turn buried deep, no emotional voice). The skill fixes this with 4 phases: Story Discovery (map the story before writing), Draft (follow the map), Composition Review (self-audit with quote-based evidence — "quote the turn sentence," not "is there a turn?"), and Revision. The composition self-audit is the key innovation — it's enforceable without new infrastructure, analogous to how D.E.K.N. enforces investigation quality. Testing splits into proxy detection patterns (can detect turn language and emotional markers but not ordering) and structural artifact validation (can verify the audit exists with real quotes). LLM-as-judge deferred until self-audit proves insufficient.

## TLDR

Designed a standalone 4-phase `technical-writer` skill that operationalizes the 4 writing primers through Story Discovery (primers as attention context) and Composition Review (quote-based self-audit as composition-level gate). The key insight: composition quality can't be tested by regex but CAN be enforced by requiring the writer to produce structural evidence of composition quality.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md` - Full design investigation with 5 findings, synthesis, recommendations
- `.orch/workspace/og-arch-design-writing-skill-20mar-875c/ARCHITECT_OUTPUT.yaml` - Structured skill design: phases, testing scenarios, implementation issues
- `.orch/workspace/og-arch-design-writing-skill-20mar-875c/VERIFICATION_SPEC.yaml` - Verification specification

### Files Modified
- `.kb/models/writing-style/model.md` - Added probe reference for this design investigation

### Commits
- (pending — will commit all artifacts together)

---

## Evidence (What Was Observed)

- 5 existing worker skills examined: all produce internal artifacts (probes, investigations) for agent/orchestrator consumption — none write external publications
- skillc detection patterns support regex/contains/negation/OR only — no AND, no position awareness, no semantic evaluation
- Writing-style model is INERT: diagnostic confirmed but primers never applied to produce a piece
- Compositional correctness gap confirmed across 3 scales (DFM, LED, agent coordination) — writing is a 4th instance
- D.E.K.N. / structured uncertainty / VERIFICATION_SPEC patterns successfully enforce self-review in existing skills — composition self-audit follows the same pattern

---

## Architectural Choices

### Standalone skill vs shared dependency
- **What I chose:** Standalone `technical-writer` skill
- **What I rejected:** Shared dependency (`writer-base`) inherited by other skills
- **Why:** Existing skills write internal artifacts for AI consumers; writing primers target human readers. Mixing audiences degrades both.
- **Risk accepted:** If other skills later need publication quality, they can't inherit it automatically.

### Self-audit vs LLM-as-judge
- **What I chose:** Self-audit with quote-based evidence (Phase 3)
- **What I rejected:** LLM-as-judge for semantic composition evaluation
- **Why:** Self-audit requires zero new infrastructure and follows the established D.E.K.N. pattern. LLM-as-judge needs new skillc capabilities.
- **Risk accepted:** Self-audit could become checkbox theater if questions are too vague. Quote-based evidence mitigates but doesn't eliminate.

### Primers as Phase 1 context vs phase-spanning rules
- **What I chose:** Primers active only in Story Discovery phase
- **What I rejected:** Primers injected throughout all phases
- **Why:** Behavioral grammars model: constraints dilute at 5+. Phase-specific instructions compete with primers for attention. Primers work as attention lens before drafting; structural audit enforces outcomes after.
- **Risk accepted:** Writer may "forget" primers during drafting. Story map from Phase 1 compensates — it encodes primer outcomes structurally.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — all 5 checks verified:
- Investigation complete with D.E.K.N., 5 findings, structured uncertainty
- ARCHITECT_OUTPUT.yaml with skill structure, 3 test scenarios, 4 implementation issues
- Writing-style model updated with probe reference
- All 5 design questions from spawn context answered
- Implementation issues defined with dependencies

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md` - Design investigation answering how to structure, test, and connect a composition-level writing skill

### Decisions Made
- Standalone skill (not modifier) because existing skills serve different audiences
- One skill (not two) because self-review needs draft context
- Self-audit with quotes (not LLM-judge) because zero new infrastructure needed
- Two-tier testing (proxy patterns + artifact validation) because regex can't test composition

### Constraints Discovered
- skillc detection patterns cannot test ordering/position (fundamental limitation)
- Token budget concern: 4 primers + 4 phases + audit template may strain 5000-token default

---

## Next (What Should Happen)

**Recommendation:** close — then spawn follow-ups from ARCHITECT_OUTPUT.yaml issues

### If Close
- [x] All deliverables complete
- [x] Investigation file has Status: Complete
- [x] ARCHITECT_OUTPUT.yaml with structured recommendations
- [x] Writing-style model updated with probe reference
- [x] Ready for `orch complete orch-go-npm1s`

### Implementation Follow-ups (from ARCHITECT_OUTPUT.yaml)
1. Create `technical-writer` skill skeleton (feature-impl, P2)
2. Write 3 contrastive test scenarios (feature-impl, P2, depends on #1)
3. Apply skill to harness engineering rewrite (technical-writer, P3, depends on #1)
4. Update writing-style model status to TESTABLE (investigation, P3, depends on #1)

---

## Unexplored Questions

- Can the completion pipeline (`orch complete`) validate composition audit artifacts, or does it need awareness of the new artifact type?
- Should the skill embed source material (investigation outputs) or just reference it?
- Does the story-map-before-drafting pattern generalize to other creative skills (presentation design, documentation)?
- Could the composition self-audit pattern be extracted as a shared phase for any skill producing human-facing artifacts?

---

## Friction

Friction: none

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-writing-skill-20mar-875c/`
**Investigation:** `.kb/investigations/2026-03-20-inv-design-writing-skill-technical-blog.md`
**Beads:** `bd show orch-go-npm1s`
