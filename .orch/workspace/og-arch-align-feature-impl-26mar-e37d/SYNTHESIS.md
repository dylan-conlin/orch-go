# Session Synthesis

**Agent:** og-arch-align-feature-impl-26mar-e37d
**Issue:** orch-go-cwj26
**Duration:** 2026-03-26T16:13 → 2026-03-26T17:00
**Outcome:** success

---

## Plain-Language Summary

The "4% feature-impl synthesis rate" alarm was mostly noise: feature-impl defaults to light tier, which explicitly tells agents not to create SYNTHESIS.md. Three parts of the system evolved different answers to "does feature-impl need synthesis?" — the skill text ignores it, the tier system makes it conditional, and the metrics count everything flat. The fix is making all three say the same thing: synthesis is required when tier=full, optional when tier=light. I designed three changes (skill template update, code comment, metric partitioning) and created four implementation issues.

## TLDR

Designed alignment for feature-impl synthesis semantics across three layers (skill text, tier policy, compliance metrics). The 4% synthesis rate is correct light-tier behavior being miscounted as non-compliance. Three targeted changes bring all layers into agreement.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-design-align-feature-impl-synthesis-semantics.md` — Full architect investigation with findings, synthesis, and recommendations

### Commits
- (pending — will commit with this synthesis)

---

## Evidence (What Was Observed)

- `config.go:44`: `feature-impl` defaults to `TierLight`
- `verify_level.go:38`: `feature-impl` skill verify level is `V2` (Evidence level)
- `verify_level.go:100`: `TierLight` caps verify level to `V0` — meaning V2 is never active for default feature-impl
- `SKILL.md.template:231-238`: Completion section lists 6 steps, none mention SYNTHESIS.md/BRIEF.md/VERIFICATION_SPEC.yaml
- `diagnostic.go:236-252`: `classifyCompleted()` already partitions by `IsFullTier` — this is correct
- `daemonconfig/compliance.go:137-139`: `DeriveSynthesisRequired()` is tier-independent — this is the gap
- Worker-base Session Complete Protocol includes full synthesis requirements, but is contradicted by the skill-local completion ending

---

## Architectural Choices

### Tier-conditional skill text over changing defaults
- **What I chose:** Add tier-conditional completion block to skill template (reference worker-base protocol when full tier)
- **What I rejected:** Changing feature-impl default tier to full
- **Why:** Light tier default was validated (Dec 20-23 changes, 77% overhead reduction). Most feature-impl work is code delivery, not knowledge production. Changing the default would regress agent efficiency.
- **Risk accepted:** Agents might still miss the conditional text if they skim. Mitigation: the worker-base protocol is already injected and handles full-tier correctly — the skill text fix just stops contradicting it.

### Code comment over refactoring verify level hierarchy
- **What I chose:** Add explanatory comment, leave V2 default in place
- **What I rejected:** Removing V2 from SkillVerifyLevelDefaults for feature-impl, or making tier-aware verify levels
- **Why:** V2 is the correct intrinsic level — it activates properly when full tier is used. The interaction is confusing but not wrong. A comment costs nothing and prevents the next reader from misunderstanding.

---

## Knowledge (What Was Learned)

### Decisions Made
- Decision: Feature-impl keeps light-tier default (validated, not changing)
- Decision: Skill text gets tier-conditional completion, not a second gate
- Decision: Metric partitioning is follow-up work (separate from contract alignment)

### Constraints Discovered
- skillc template doesn't have access to Go template variables — tier-conditional text must be prose ("if your SPAWN_CONTEXT says FULL TIER..."), not a Go template conditional
- The `@section` markers in the skill template control progressive disclosure but the Completion section is not inside one, so it's always visible

---

## Next (What Should Happen)

**Recommendation:** close (architect design complete, implementation routed to follow-up issues)

### If Close
- [x] All deliverables complete (investigation, SYNTHESIS.md, BRIEF.md, VERIFICATION_SPEC.yaml)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-cwj26`

### Implementation Issues Created
- `orch-go-36tvq` — Add tier-conditional completion section to feature-impl skill template
- `orch-go-cerm1` — Add code comment to verify_level.go explaining V2/tier-cap interaction
- `orch-go-41dm3` — Partition bench ComplianceSignals by spawn tier
- `orch-go-7xoc6` — Add tier parameter to DeriveSynthesisRequired

---

## Unexplored Questions

- How often is feature-impl actually spawned with `--tier full`? If very rare, the prompt split barely matters in practice.
- Is `DeriveSynthesisRequired` consumed by any code path that affects agent behavior at spawn time, or is it advisory-only?
- Would it be worth adding a SPAWN_CONTEXT variable that skills can reference (e.g., `{{.IsSynthesisRequired}}`) to avoid prose conditionals?

---

## Friction

Friction: none

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-align-feature-impl-26mar-e37d/`
**Investigation:** `.kb/investigations/2026-03-26-design-align-feature-impl-synthesis-semantics.md`
**Beads:** `bd show orch-go-cwj26`
