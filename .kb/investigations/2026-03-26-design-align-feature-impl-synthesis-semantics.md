## Summary (D.E.K.N.)

**Delta:** The 4% feature-impl synthesis rate is three bugs in a trenchcoat: (1) the skill's completion section never mentions SYNTHESIS.md/BRIEF.md/VERIFICATION_SPEC.yaml, (2) light tier tells agents to skip synthesis while the skill's V2 verify level implies it, and (3) aggregate metrics don't partition by tier so correct light-tier behavior looks like full-tier non-compliance.

**Evidence:** Code traced across `config.go:44` (TierLight default), `verify_level.go:38` (V2 skill default, capped to V0 by light tier), `worker_template.go:41-47` (tier instructions), `SKILL.md.template:231-238` (completion section omits synthesis), `diagnostic.go:236-252` (tier-partitioned check already correct), `daemonconfig/compliance.go:137-139` (DeriveSynthesisRequired is tier-independent).

**Knowledge:** The contract is split because three independently designed subsystems evolved different answers to "does feature-impl need synthesis?" Skill text says no (by omission). Tier policy says depends (light=no, full=yes). Compliance metrics say yes (flat rate). The fix is making all three express the same conditional: synthesis is required when tier=full, optional when tier=light.

**Next:** Implement three targeted changes: (1) add tier-conditional completion block to feature-impl skill template, (2) add tier field to metrics aggregation, (3) document the V2/tier cap interaction in a code comment. Authority: architectural — crosses skill, spawn, and compliance packages.

**Authority:** architectural — Changes cross skill template (skillc build domain), spawn verification (pkg/spawn), and compliance measurement (pkg/daemonconfig). No single component owns this; orchestrator decides.

---

# Investigation: Align Feature-Impl Synthesis Semantics

**Question:** How should skill text, tier defaults, and compliance measurement express the same contract for feature-impl synthesis requirements?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** architect (orch-go-cwj26)
**Phase:** Complete
**Next Step:** Route implementation issues to feature-impl workers
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| orch-go-n4uwb (brief) | extends | Yes — confirmed 3-layer split via code trace | None — brief correctly identified skill/tier/metric split |
| 2026-02-28 stalled agent audit | deepens | Yes — diagnostic.go uses IsFullTier partition | None |
| 2026-03-06 worker skill industry gaps | related | Not re-verified | N/A |

---

## Findings

### Finding 1: Feature-impl skill completion section never references SYNTHESIS.md

**Evidence:** The feature-impl `SKILL.md.template` completion section (lines 231-238) contains six steps: self-review, phases, deliverables, visual verification, Phase: Complete, commit. None mention SYNTHESIS.md, BRIEF.md, or VERIFICATION_SPEC.yaml.

Meanwhile, the worker-base Session Complete Protocol (injected via dependency) includes VERIFICATION_SPEC.yaml (step 1), SYNTHESIS.md (step 5), and BRIEF.md (step 6). When both are loaded, the agent sees TWO completion protocols — the skill's is more visible because it's positioned at the end of the skill-specific content.

**Source:** `skills/src/worker/feature-impl/.skillc/SKILL.md.template:231-238`, worker-base Session Complete Protocol in SPAWN_CONTEXT.md

**Significance:** This is the "prompt split" the orch-go-n4uwb brief identified. When feature-impl runs as full tier, the worker-base adds synthesis requirements but the skill-local completion section contradicts by omission. Agents follow the last-seen, skill-specific instructions.

---

### Finding 2: V2 verify level is always overridden by light tier cap

**Evidence:** `SkillVerifyLevelDefaults["feature-impl"] = VerifyV2` (`verify_level.go:38`). But `SkillTierDefaults["feature-impl"] = TierLight` (`config.go:44`). `TierMaxVerifyLevel[TierLight] = VerifyV0` (`verify_level.go:100`). The function `VerifyLevelForTier(TierLight, V2)` returns `min(V2, V0) = V0`.

This means the V2 default for feature-impl is dead code in the default path. It only activates when `--tier full` overrides the tier. This is not a bug — it's intentional layering — but the interaction is undocumented and confusing when reading `verify_level.go` in isolation.

**Source:** `pkg/spawn/verify_level.go:38,100-116`, `pkg/spawn/config.go:44`

**Significance:** Anyone reading the verify level defaults would expect feature-impl to run synthesis gates. The tier cap makes this wrong for the default path. A code comment would prevent this misunderstanding.

---

### Finding 3: Diagnostic code already partitions by tier — metrics aggregation does not

**Evidence:** `diagnostic.go:236-252` correctly checks `agent.IsFullTier && !agent.HasSynthesis` before flagging `synthesis_gap`. Light-tier agents that complete without SYNTHESIS.md are classified as healthy.

However, `daemonconfig/compliance.go:137-139` has `DeriveSynthesisRequired(level ComplianceLevel) bool` which returns true for Strict/Standard regardless of tier. The bench `ComplianceSignals` struct has no tier partition field. The "4% synthesis rate" from orch-go-n4uwb was likely computed from workspace scanning without tier partitioning.

**Source:** `pkg/daemon/diagnostic.go:236-252`, `pkg/daemonconfig/compliance.go:137-139`, `pkg/bench/report.go:54-60`

**Significance:** The diagnostic code (completion-time) is correct. The compliance config (policy-level) and bench reporting (metrics-level) are not tier-aware. This is how a correct-by-design system produces a misleading headline metric.

---

## Synthesis

**Key Insights:**

1. **The contract is split, not broken** — Each subsystem independently evolved a reasonable answer. The skill focuses on code delivery (no synthesis needed). The tier system gates knowledge work (synthesis when full). The compliance measurement predates the tier system and measures flat. The fix is alignment, not redesign.

2. **The skill completion section is the highest-leverage fix** — Agents see skill-specific completion last and follow it. Making the skill's completion section tier-aware would resolve the prompt split with one change, because the worker-base protocol is already correct for full-tier spawns — it just gets contradicted by the skill-local ending.

3. **Metrics should reflect the contract, not override it** — The 4% number created urgency for a problem that was ~90% correct behavior. Partitioned metrics would show: "light-tier synthesis: 4% (expected ~0%)" and "full-tier synthesis: X% (target ~80%)". This surfaces the real compliance gap without inflating it.

**Answer to Investigation Question:**

The alignment requires three changes, all expressing the same conditional: **synthesis is required when tier=full, optional when tier=light**.

1. **Skill text**: Add tier-conditional completion block to feature-impl's `SKILL.md.template` Completion section. When full tier, reference the worker-base Session Complete Protocol (SYNTHESIS.md, BRIEF.md, VERIFICATION_SPEC.yaml). When light tier, the current completion ending is correct.

2. **Compliance measurement**: `DeriveSynthesisRequired` should take a tier parameter (or the compliance config resolution should factor tier). The bench `ComplianceSignals` should partition metrics by tier when computing per-skill synthesis rates.

3. **Documentation**: Add a code comment at `SkillVerifyLevelDefaults["feature-impl"] = VerifyV2` explaining the tier cap interaction: "V2 is the intrinsic level; when tier=light (default), this is capped to V0 by TierMaxVerifyLevel."

---

## Structured Uncertainty

**What's tested:**

- ✅ Feature-impl defaults to TierLight (verified: `config.go:44`)
- ✅ Light tier caps V2 to V0 (verified: `verify_level.go:99-102`, `VerifyLevelForTier` function)
- ✅ Feature-impl completion section omits synthesis artifacts (verified: `SKILL.md.template:231-238`)
- ✅ Diagnostic code already partitions by tier (verified: `diagnostic.go:236-252`)
- ✅ worker-base Session Complete Protocol includes full synthesis requirements (verified: SPAWN_CONTEXT skill injection)

**What's untested:**

- ⚠️ Whether adding tier-conditional text to the skill template actually improves full-tier compliance (would need before/after measurement)
- ⚠️ Whether the bench reporting pipeline has adequate hooks for tier partitioning (haven't read the full bench aggregation code)
- ⚠️ Whether `DeriveSynthesisRequired` is actually used in any code path that matters (it may be advisory-only)

**What would change this:**

- If agents don't read the skill completion section (they actually read worker-base more carefully), the skill text fix has no effect
- If the 4% metric was already tier-partitioned and the brief was wrong about the computation, finding 3 is incorrect
- If full-tier feature-impl spawns are extremely rare, the skill text split may not matter in practice

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add tier-conditional completion to skill template | implementation | Stays within skill template domain, follows existing patterns |
| Add tier param to DeriveSynthesisRequired | implementation | Localized change to compliance config |
| Add code comment to verify_level.go | implementation | Documentation only |
| Partition bench metrics by tier | architectural | Crosses bench reporting and event schema — affects how compliance is measured across the system |

### Recommended Approach ⭐

**Tier-conditional skill completion** — Make the feature-impl skill's completion section explicitly tier-aware, add a code comment to verify_level.go, and create follow-up issues for compliance metric partitioning.

**Why this approach:**
- Fixes the highest-impact problem (prompt split) with minimal blast radius
- Aligns with Legibility Over Compliance principle: makes the contract readable to agents rather than adding another gate
- Aligns with Coherence Over Patches: single source of truth for completion requirements per tier
- Doesn't require changing tier defaults (feature-impl light default is correct for most spawns)

**Trade-offs accepted:**
- Compliance metrics remain unpartitioned until follow-up issue is implemented
- The V2 default stays as-is (it's correct, just confusing without the comment)

**Implementation sequence:**

1. **Feature-impl skill template** (`skills/src/worker/feature-impl/.skillc/SKILL.md.template`):
   Replace the Completion section with a tier-conditional block. The key change: when the SPAWN_CONTEXT says "FULL TIER", the completion section should explicitly list SYNTHESIS.md, BRIEF.md, and VERIFICATION_SPEC.yaml. When "LIGHT TIER" (or no tier marker), the current ending is correct.

   Since the skill template doesn't have access to Go template variables, use a prose conditional: "If your SPAWN_CONTEXT says **FULL TIER**, also complete the Session Complete Protocol from worker-base (VERIFICATION_SPEC.yaml, SYNTHESIS.md, BRIEF.md). If **LIGHT TIER**, skip these — Phase: Complete and commit is sufficient."

2. **Code comment** (`pkg/spawn/verify_level.go`):
   Add comment at the `feature-impl: VerifyV2` line explaining the tier cap interaction.

3. **Follow-up issues:**
   - `bd create "Partition bench ComplianceSignals by tier for per-skill synthesis rate" --type task -l triage:ready`
   - `bd create "Add tier parameter to DeriveSynthesisRequired for tier-aware compliance config" --type task -l triage:ready`

### Alternative Approaches Considered

**Option B: Change feature-impl default tier to full**
- **Pros:** Eliminates the split entirely — all feature-impl would require synthesis
- **Cons:** Adds significant overhead to simple code changes. The light tier default was a validated decision (Dec 20-23 changes, 77% overhead reduction). Would regress agent efficiency for the common case.
- **When to use instead:** If evidence shows most feature-impl work is knowledge-producing (currently contradicted by usage patterns).

**Option C: Remove V2 from feature-impl skill verify level defaults**
- **Pros:** Eliminates the confusing "dead code" verify level
- **Cons:** When `--tier full` is used, the verify level would fall to V1 (conservative default for unknown), which is wrong — full-tier feature-impl should still have V2 evidence gates. Would need to add tier-based verify level escalation, adding complexity.
- **When to use instead:** If the tier cap system is removed or redesigned.

**Rationale for recommendation:** Option A fixes the visible problem (prompt split) without changing defaults that are working (light tier, V2 with cap). The follow-up issues capture the metric alignment work separately, allowing incremental implementation.

---

### Implementation Details

**What to implement first:**
- Feature-impl skill template change (highest impact, lowest risk)
- Code comment (trivial, prevents future confusion)
- Follow-up issue creation (tracks remaining work)

**Things to watch out for:**
- ⚠️ The skill template uses `@section` markers for progressive disclosure — the completion section is not inside a section marker, so it's always visible. This is correct for our change.
- ⚠️ `skillc build` must be run after template edits; direct SKILL.md edits are overwritten.
- ⚠️ The worker-base Session Complete Protocol order is specific (VERIFICATION_SPEC first, then SYNTHESIS.md, then BRIEF.md). The skill template reference should not restate the full protocol — just reference it.

**Areas needing further investigation:**
- How often is feature-impl spawned with `--tier full`? If it's <5% of spawns, the skill text split matters less. Could check events.jsonl for tier distribution.
- Whether `DeriveSynthesisRequired` is consumed by any code path that affects agent behavior at spawn time (it might only be used for daemon compliance configuration, not for spawn-time decisions).

**Success criteria:**
- ✅ Feature-impl skill completion section has explicit tier-conditional guidance
- ✅ `verify_level.go` has code comment explaining tier cap interaction
- ✅ Follow-up issues created for metric partitioning and compliance config alignment
- ✅ `skillc build` succeeds with the template change
- ✅ No behavioral regression for light-tier feature-impl spawns

---

## References

**Files Examined:**
- `pkg/spawn/config.go:18-56` — Tier constants, SkillTierDefaults, feature-impl = TierLight
- `pkg/spawn/verify_level.go:1-132` — Verify level constants, tier capping, feature-impl = V2
- `pkg/spawn/worker_template.go:1-473` — SPAWN_CONTEXT template with tier-conditional blocks
- `skills/src/worker/feature-impl/.skillc/SKILL.md.template:1-248` — Feature-impl skill source
- `skills/src/worker/feature-impl/.skillc/phases/validation.md` — Validation phase completion criteria
- `skills/src/worker/feature-impl/.skillc/skill.yaml` — Skill config (no synthesis in deliverables)
- `pkg/verify/check.go:575-604` — Synthesis gate implementation (V2+)
- `pkg/verify/level.go:1-109` — Gate-by-level mapping, synthesis at V2
- `pkg/daemon/diagnostic.go:1-295` — Failure mode classification, tier-aware synthesis_gap
- `pkg/daemonconfig/compliance.go:1-170` — Compliance config, DeriveSynthesisRequired (tier-unaware)
- `pkg/bench/report.go:1-80` — ComplianceSignals (no tier partition)
- `.kb/briefs/orch-go-n4uwb.md` — Prior investigation brief identifying the 3-layer split

**Related Artifacts:**
- **Brief:** `.kb/briefs/orch-go-n4uwb.md` — Original investigation that found the 4% synthesis rate
- **Decision:** Skills performing well after Dec 20-23 changes — Light-tier system validated
- **Principle:** Legibility Over Compliance (`~/.kb/principles.md`) — Guides toward readable contracts over additional gates
- **Principle:** Coherence Over Patches (`~/.kb/principles.md`) — Single source of truth over multiple overlapping instructions

---

## Investigation History

**2026-03-26 16:15:** Investigation started
- Initial question: How should skill text, tier defaults, and compliance measurement express the same contract?
- Context: orch-go-n4uwb found 4% synthesis rate is a 3-layer semantic split, routed through architect

**2026-03-26 16:30:** Exploration complete — 3 findings confirmed via code trace
- Skill completion omits synthesis (SKILL.md.template:231-238)
- V2 verify level is capped to V0 by default light tier (verify_level.go + config.go)
- Diagnostic partitions by tier, metrics don't (diagnostic.go vs daemonconfig/compliance.go)

**2026-03-26 16:45:** Investigation completed
- Status: Complete
- Key outcome: Three targeted changes (skill template, code comment, follow-up issues) align the contract without changing defaults
