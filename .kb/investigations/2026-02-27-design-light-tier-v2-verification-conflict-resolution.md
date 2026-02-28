## Summary (D.E.K.N.)

**Delta:** LIGHT tier and V2 verification level make contradictory promises about SYNTHESIS.md — agents are told to skip it, but verification requires it. The fix is to make feature-impl default to V1 (not V2), letting issue type minimums escalate to V2 only when backed by a feature/bug issue.

**Evidence:** Code analysis of `pkg/spawn/verify_level.go`, `pkg/verify/check.go`, `pkg/spawn/config.go`, and `pkg/spawn/context.go` confirms the conflict manifests in every LIGHT feature-impl completion.

**Knowledge:** The V0-V3 system was designed to replace the tier system but the migration was incomplete. The correct fix completes the migration by making V0-V3 the authoritative axis, with tier becoming a cosmetic label derived from the level.

**Next:** Implement Option A — change feature-impl skill default from V2 to V1. Remove tier from `GatesForLevel` entirely.

**Authority:** architectural — Cross-component change affecting spawn defaults, verification gates, and agent instructions.

---

# Investigation: LIGHT Tier / V2 Verification Level Conflict Resolution

**Question:** Should LIGHT spawn tier map to V1 verification level, should GatesForLevel check spawn tier, or should the LIGHT tier be removed entirely?

**Defect-Class:** configuration-drift

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** orch-go-i9qi agent
**Phase:** Complete
**Next Step:** None — accept recommendation, spawn feature-impl to implement
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-20-verification-levels-v0-v3.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` | extends | Yes — decision says "three implicit systems → one explicit system" | Tier system persists alongside V0-V3 |
| `.kb/investigations/2026-02-20-inv-architect-verification-levels.md` | extends | Yes — designed V0-V3 to replace tier | LIGHT tier not accounted for in migration |
| `.kb/models/completion-verification/probes/2026-02-20-probe-verification-levels-design.md` | confirms | Yes — noted auto-skip for knowledge skills | Didn't address LIGHT non-knowledge skills |

---

## Findings

### Finding 1: The conflict is structural, not incidental

**Evidence:** Two independent mapping tables assign contradictory requirements to the same work type:

| System | Source | feature-impl → | SYNTHESIS.md |
|--------|--------|----------------|--------------|
| Spawn tier | `SkillTierDefaults` in `config.go:43` | TierLight | "NOT required" |
| Verify level | `SkillVerifyLevelDefaults` in `verify_level.go:34` | V2 | Required (GateSynthesis fires at V1+) |

The SPAWN_CONTEXT template (`context.go:116`) tells the agent: "⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required." Then `verifyCompletionWithLevelAndComments` (`check.go:551`) runs GateSynthesis because V2 ⊃ V1.

**Source:** `pkg/spawn/config.go:43`, `pkg/spawn/verify_level.go:34`, `pkg/spawn/context.go:116`, `pkg/verify/check.go:551-576`

**Significance:** This isn't an edge case — feature-impl is the most common spawn type. Every single LIGHT feature-impl completion hits this conflict, requiring `--skip-synthesis`.

---

### Finding 2: The legacy path got it right, the modern path lost it

**Evidence:** Two code paths exist:

- **Legacy** (`VerifyCompletionWithTierAndComments`, `check.go:653`): `if tier != "light"` → correctly skips synthesis for LIGHT
- **Modern** (`verifyCompletionWithLevelAndComments`, `check.go:551`): `ShouldRunGate(verifyLevel, GateSynthesis)` → no tier check, LIGHT agents fail

The modern level-aware path was designed to replace tier-based gating. But it replaced the tier check without encoding the same semantic: "implementation-focused skills don't produce synthesis."

**Source:** `pkg/verify/check.go:551-576` (modern), `pkg/verify/check.go:650-678` (legacy)

**Significance:** The V0-V3 decision doc explicitly says levels should replace tiers. But when the level system was implemented, the tier's role in suppressing synthesis was lost. This is a migration gap, not a design gap.

---

### Finding 3: There's already a partial workaround showing the pattern

**Evidence:** Line 388 in `check.go`:
```go
if !isOrch && ShouldRunGate(verifyLevel, GateArchitecturalChoices) && tier != "light" {
```

The `GateArchitecturalChoices` gate already has `tier != "light"` as an escape hatch. This was an ad-hoc fix for the same class of problem — someone hit the LIGHT/V2 conflict for architectural choices and added a tier check.

**Source:** `pkg/verify/check.go:388`

**Significance:** This proves the problem is known and has been worked around in at least one gate. But adding `tier != "light"` to individual gates is the wrong fix (Option B) — it makes tier a shadow authority alongside levels, contradicting the decision to unify on V0-V3.

---

### Finding 4: The intent of LIGHT and V2 both map to "implementation-focused work"

**Evidence:** Both systems agree on the semantic:

- `TierLight` comment: "Lightweight spawn - skips SYNTHESIS.md requirement" — for code-producing skills
- `VerifyV2` comment: "Evidence: V1 + test evidence, build, git diff" — for implementation skills

The V2 gates (test evidence, build, git diff, accretion) are all about *code quality*. The V1 gates (synthesis, handoff, skill output, constraints) are about *knowledge artifacts*. A feature-impl agent produces code, not knowledge artifacts. V2 accidentally inherits V1's knowledge-artifact gates because levels are strict supersets.

**Source:** `pkg/spawn/config.go:19-20`, `pkg/spawn/verify_level.go:7-10`, `pkg/verify/level.go:7-31`

**Significance:** The "strict superset" invariant (V0⊂V1⊂V2⊂V3) creates a semantic mismatch: V2 inherits V1 gates that don't apply to V2 work types. The solution must either break the superset property or change default level assignments.

---

### Finding 5: Issue type minimums already provide the V2 escalation path

**Evidence:** From `pkg/spawn/verify_level.go:41-48`:
```go
var IssueTypeMinVerifyLevel = map[string]string{
    "feature":       VerifyV2,
    "bug":           VerifyV2,
    "decision":      VerifyV2,
    "investigation": VerifyV1,
    "probe":         VerifyV1,
}
```

`DefaultVerifyLevel(skill, issueType)` returns `max(skill_level, issue_type_minimum)`. If feature-impl defaulted to V1 instead of V2, then:
- `DefaultVerifyLevel("feature-impl", "feature")` → `max(V1, V2)` → V2 ✓
- `DefaultVerifyLevel("feature-impl", "bug")` → `max(V1, V2)` → V2 ✓
- `DefaultVerifyLevel("feature-impl", "task")` → `max(V1, none)` → V1 ✓
- `DefaultVerifyLevel("feature-impl", "")` → V1 (no issue → lighter verification)

With issue backing, feature-impl still gets V2. Without issue backing (rare, ad-hoc spawns), it gets V1. This is the correct semantic: tracked implementation work gets evidence gates; untracked implementation gets artifact gates only.

**Source:** `pkg/spawn/verify_level.go:41-48`, `pkg/spawn/verify_level.go:50-65`

**Significance:** The issue type minimum system was designed exactly for this case. It allows the default skill level to be lower while ensuring tracked work gets appropriate scrutiny.

---

## Synthesis

**Key Insights:**

1. **The V0-V3 migration is incomplete** — Tier still controls agent instructions (SPAWN_CONTEXT), partially controls gates (architectural choices), and coexists with levels rather than being replaced by them. The fix should complete the migration, not add more tier checks.

2. **V2 level conflates "code evidence" with "knowledge artifacts"** — Because V2 is a strict superset of V1, implementation skills inherit knowledge-artifact gates (synthesis, handoff, skill output) that don't match their output type. The right fix is to not assign V2 to implementation skills by default — let issue type minimums provide the V2 escalation.

3. **Issue type minimums are the correct escalation mechanism** — `DefaultVerifyLevel("feature-impl", "feature")` → V2 already works. Lowering the skill default from V2 to V1 preserves V2 for tracked work while eliminating the synthesis conflict for LIGHT spawns.

**Answer to Investigation Question:**

**Option A (LIGHT → V1) is the correct approach**, but framed more precisely: change `SkillVerifyLevelDefaults["feature-impl"]` from V2 to V1. This makes the skill default match what the agent is actually told (no synthesis needed), while issue type minimums ensure feature/bug-backed work still gets V2 evidence gates. This completes the V0-V3 migration for this case and moves toward eliminating the tier concept entirely.

---

## Structured Uncertainty

**What's tested:**

- ✅ feature-impl defaults to TierLight AND V2 simultaneously (verified: read both config maps)
- ✅ V2 includes GateSynthesis via strict superset (verified: read gatesByLevel in level.go)
- ✅ Modern verification path doesn't check tier for synthesis gate (verified: read check.go:551-576)
- ✅ Issue type minimums would escalate feature-impl from V1 to V2 when backed by feature/bug issue (verified: traced DefaultVerifyLevel logic)
- ✅ SPAWN_CONTEXT tells LIGHT agents to skip synthesis (verified: read context.go:116)

**What's untested:**

- ⚠️ How many existing feature-impl agents are spawned without issue backing (affects whether V1 default would reduce gate coverage)
- ⚠️ Whether daemon auto-complete also hits this conflict (daemon likely uses VerifyCompletionFull → same code path)
- ⚠️ Whether `systematic-debugging` (also V2/FULL) has the same conflict (it's FULL tier, so no conflict there)

**What would change this:**

- If feature-impl is regularly spawned without issue backing AND evidence gates are important for those spawns → V1 default would be too lenient
- If breaking the strict superset property (V0⊂V1⊂V2⊂V3) is acceptable → could move GateSynthesis out of V2's inherited set instead

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Change feature-impl default from V2 to V1 | architectural | Cross-component: affects spawn config, verification pipeline, agent instructions |
| Also change reliability-testing from V2 to V1 | architectural | Same pattern — implementation skill, same conflict if ever made LIGHT |
| Remove ad-hoc `tier != "light"` from architectural choices gate | implementation | Cleanup — removing workaround once root cause fixed |
| Long-term: derive tier from verification level | strategic | Eliminates dual-axis system entirely — irreversible migration |

### Recommended Approach ⭐

**Option A: Change feature-impl (and reliability-testing) default from V2 to V1** — Make the skill default match its typical spawn tier by lowering the default verification level. Issue type minimums provide V2 when backed by feature/bug issues.

**Why this approach:**
- Eliminates the contradiction at its source — no more conflicting instructions
- Completes the V0-V3 migration intent (levels replace tiers as the control axis)
- Zero-skip-flag goal achieved: tracked feature work still gets V2 via issue type; untracked gets V1 (appropriate)
- Preserves strict superset property (V0⊂V1⊂V2⊂V3) — no structural change to the level system
- `DefaultVerifyLevel("feature-impl", "feature")` still returns V2

**Trade-offs accepted:**
- Untracked feature-impl spawns (no issue) get V1 instead of V2 — loses test evidence, build, and git diff gates
- Acceptable because: untracked spawns are ad-hoc; if they matter, they should have issues

**Implementation sequence:**
1. Change `SkillVerifyLevelDefaults["feature-impl"]` from V2 to V1 in `verify_level.go`
2. Change `SkillVerifyLevelDefaults["reliability-testing"]` from V2 to V1 (same pattern)
3. Remove `tier != "light"` workaround from architectural choices gate in `check.go:388`
4. Update the V0-V3 decision doc to note this refinement
5. Update tests in `verify_level.go` and `check.go` to reflect new defaults

### Alternative Approaches Considered

**Option B: GatesForLevel checks tier to suppress synthesis for LIGHT**
- **Pros:** Minimal code change (add tier param to GatesForLevel)
- **Cons:** Enshrines tier as a shadow authority alongside levels — directly contradicts the V0-V3 decision ("one concept — verification level — declared at spawn time, determines everything at completion time"). Makes the dual-axis problem permanent.
- **When to use instead:** Never — this is the approach that led to the ad-hoc `tier != "light"` workaround already.

**Option C: Remove LIGHT tier entirely, let V0-V3 be the only axis**
- **Pros:** Cleanest long-term solution — eliminates the dual-axis problem entirely
- **Cons:** Large migration. Tier controls SPAWN_CONTEXT instructions, agent behavior, context generation (synthesis template), and completion display. Many touch points.
- **When to use instead:** As a follow-up after Option A stabilizes. Option A is a stepping stone toward C — once levels correctly control everything, tier becomes a cosmetic label that can be removed at leisure.

**Rationale for recommendation:** Option A fixes the immediate bug (LIGHT/V2 conflict) while moving toward the long-term goal (Option C). Option B makes the problem permanent. Option C is correct but too large for this issue.

---

### Implementation Details

**What to implement first:**
- Change the two `SkillVerifyLevelDefaults` entries (feature-impl, reliability-testing) from V2 to V1
- This is the single change that fixes the bug

**Things to watch out for:**
- ⚠️ Existing tests may assert feature-impl → V2 default — update them
- ⚠️ The `complete_cmd.go:2037` display logic already checks `tier == "light"` for synthesis display — this will still work correctly
- ⚠️ Daemon auto-complete uses the same verification pipeline — confirm it benefits from this fix

**Areas needing further investigation:**
- What percentage of feature-impl spawns are untracked (no issue)? If high, V1 default may be too lenient
- Should `systematic-debugging` also move to V1? It's currently TierFull so no conflict, but the semantic question is the same

**Success criteria:**
- ✅ `orch spawn feature-impl "task" --issue <feature-issue>` → V2 (via issue type minimum)
- ✅ `orch spawn feature-impl "task"` → V1 (no issue → lighter verification)
- ✅ `orch complete <agent>` requires zero skip flags for standard LIGHT feature-impl
- ✅ No `--skip-synthesis` needed for any LIGHT spawn

---

### Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the LIGHT/V2 conflict that required --skip-synthesis for every feature-impl completion
- Future spawns might change verification level defaults

**Suggested blocks keywords:**
- verification level default
- feature-impl verification
- synthesis gate skip
- light tier verification

---

## References

**Files Examined:**
- `pkg/spawn/config.go:17-55` — Tier constants and skill tier defaults
- `pkg/spawn/verify_level.go:1-103` — Verification level constants, defaults, and DefaultVerifyLevel function
- `pkg/verify/level.go:1-89` — Gate-to-level mapping and GatesForLevel function
- `pkg/verify/check.go:280-680` — Both modern and legacy verification paths
- `pkg/spawn/context.go:113-122` — SPAWN_CONTEXT tier instructions to agents
- `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` — V0-V3 decision document

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` — The V0-V3 system this investigation patches
- **Model:** `.kb/models/completion-verification/model.md` — Completion verification architecture
- **Probe:** `.kb/models/completion-verification/probes/2026-02-27-probe-light-tier-v2-verification-conflict.md` — Confirmatory probe for this investigation

---

## Investigation History

**2026-02-27:** Investigation started
- Initial question: Should LIGHT tier map to V1, should GatesForLevel check tier, or should LIGHT be removed?
- Context: Every LIGHT feature-impl completion requires --skip-synthesis due to V2/LIGHT contradiction

**2026-02-27:** Conflict confirmed via code analysis
- Both mapping tables (tier defaults, level defaults) assign contradictory requirements to feature-impl
- Modern verification path lost the legacy path's tier check during V0-V3 migration
- Issue type minimums already provide the V2 escalation mechanism

**2026-02-27:** Investigation completed
- Status: Complete
- Key outcome: Option A (change feature-impl default from V2 to V1) fixes the bug while completing the V0-V3 migration intent
