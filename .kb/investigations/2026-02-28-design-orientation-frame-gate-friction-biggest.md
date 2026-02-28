## Summary (D.E.K.N.)

**Delta:** The V0-V3 level system (designed Feb 20) correctly defines which gates fire per level, but 8 duplicate skill-classification lists and a residual "light/full" tier concept create massive bypass friction — 1943 auto-skips and 107 manual synthesis bypasses in 7 days.

**Evidence:** Code analysis of 17 gate implementations in pkg/verify/ shows ShouldRunGate() already prevents irrelevant gates from running, but each gate internally re-checks skill class and generates auto-skip events. Stats confirm: synthesis gate has 16.3% fail rate with bypass reasons overwhelmingly citing "light tier, no synthesis by design."

**Knowledge:** The gate friction is not a design problem — V0-V3 is the right architecture. It's a migration debt problem: the old tier-based and skill-based gating code was never cleaned up after V0-V3 was implemented. Six concrete cleanup tasks eliminate the friction without changing the gate architecture.

**Next:** Six implementation issues created below. Priority order: (1) map spawn tier→verify level, (2) remove per-gate skill lists, (3) remove internal auto-skip logic, (4) remove ad-hoc tier guards, (5) deprecate legacy path, (6) add missing skills to level defaults.

**Authority:** architectural - Cross-component changes across pkg/verify/ and pkg/spawn/ affecting all gate execution paths.

---

# Investigation: Gate Friction Landscape and Tier-Aware Redesign

**Question:** Why does the verification gate system generate so much bypass friction (107 manual bypasses, 1943 auto-skips, 16.3% synthesis fail rate), and what changes would eliminate unnecessary friction while preserving quality gates?

**Defect-Class:** configuration-drift

**Started:** 2026-02-28
**Updated:** 2026-02-28
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - implementation issues created
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-20-verification-levels-v0-v3.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-20 verification-levels-v0-v3 decision | extends | Yes - V0-V3 is implemented but migration incomplete | No conflict - this investigation addresses migration debt identified as risk |
| 2026-02-25 no-code-review-gate decision | confirms | Yes - three-type gate taxonomy (execution/evidence/judgment) validates level groupings | None |
| 2026-01-04 gate-refinement-passable-by-gated | confirms | Yes - gates that agents can't pass become bypassed (exactly what synthesis bypass data shows) | None |

---

## Findings

### Finding 1: V0-V3 Level System Is Correct But Underutilized

**Evidence:** The level system in `pkg/spawn/verify_level.go` correctly maps skills to levels and `pkg/verify/level.go` correctly defines which gates fire at each level. The pipeline in `VerifyCompletionFullWithComments` (check.go:285) already calls `ShouldRunGate(verifyLevel, gateName)` before each gate. This means the level system IS preventing irrelevant gates from running.

However, the stats show 1943 auto-skip events. These come from internal logic WITHIN gates that re-checks skill class after the level system already decided to run the gate. The auto-skips are defense-in-depth that was never removed after V0-V3 shipped.

**Source:**
- `pkg/verify/level.go:61-71` — `ShouldRunGate()` function
- `pkg/verify/check.go:328-475` — pipeline using `ShouldRunGate()` before each gate
- `orch stats` output — 1943 auto-skip events

**Significance:** The architecture is sound. The friction is migration debt, not design failure. This makes the fix tractable: cleanup, not redesign.

---

### Finding 2: Eight Duplicate Skill-Classification Systems

**Evidence:** Every gate maintains its own skill include/exclude lists:

| File | Lists | Purpose |
|------|-------|---------|
| `test_evidence.go:32-48` | `skillsRequiringTestEvidence`, `skillsExcludedFromTestEvidence` | Who needs test output |
| `visual.go:20-38` | `skillsRequiringVisualVerification`, `skillsExcludedFromVisualVerification` | Who needs screenshots |
| `build_verification.go:27-43` | `skillsRequiringBuildVerification`, `skillsExcludedFromBuildVerification` | Who needs go build |
| `architectural_choices.go:17-21` | `skillsRequiringArchitecturalChoices` | Who needs tradeoff declaration |
| `escalation.go:70-77` | `knowledgeProducingSkills` | Who gets synthesis auto-skipped |
| `check.go:559,661` | Inline `IsKnowledgeProducingSkill()` calls | Synthesis gate skip logic |

Adding a new skill (e.g., `capture-knowledge`, `probe`, `ux-audit`) requires checking and potentially updating **all 8 locations**. The level system already encodes this information in one place (`pkg/spawn/verify_level.go:22-37`).

**Source:** All files listed above, verified by grep for skill list patterns.

**Significance:** This is the root cause of inconsistent gate behavior. The level system says "this is V1 work, only run V1 gates" but individual V2 gates then re-check internally whether to actually run. The per-gate skill lists can drift from the level defaults, creating confusing behavior.

---

### Finding 3: Synthesis Gate Is The Biggest Friction Source — But Already Handled by Levels

**Evidence:** From stats:
- 56 failures, 107 bypasses, 1943 auto-skips
- Bypass reasons: "Light tier feature-impl, no synthesis by design" (50+ instances across variations)
- Auto-skips: knowledge-producing skills (investigation, architect, research, etc.)

The V0-V3 system handles this correctly:
- V0 skills don't run synthesis gate (`ShouldRunGate("V0", "synthesis")` returns false)
- V1 skills run it but knowledge-producing skills auto-skip (check.go:559)

The 1943 auto-skips happen because V1 skills (investigation, architect) DO trigger the synthesis gate via the level system, but then the gate internally skips for knowledge-producing skills. These should simply not have the gate run at all — either by making them V0 or by keeping the auto-skip as an intentional V1 behavior (knowledge skills need synthesis checked differently).

The 107 manual bypasses are the real pain: feature-impl spawned as "light tier" but getting V2 verification (because feature-impl defaults to V2). The orchestrator then has to manually explain why synthesis isn't needed.

**Source:** `orch stats` output, `check.go:549-576`, bypass reason patterns

**Significance:** The fix is to ensure "light tier" spawns map to a lower verify level at spawn time. A feature-impl with `--tier light` should get V0 or V1, not V2.

---

### Finding 4: Residual Tier Guards Create Ad-Hoc Gate Bypasses

**Evidence:** Even within the level-aware pipeline, there are ad-hoc tier checks:
- `check.go:388`: `tier != "light"` guard on architectural_choices gate — this means the level system says "run this gate" but the tier override says "no"
- `check.go:653`: `tier != "light"` to skip synthesis in the **legacy** path

These create an inconsistency: the level says V2 (run architectural_choices), but `tier == "light"` overrides it. The correct fix is to have light tier map to a lower level at spawn time, not to have tier guards inside the pipeline.

**Source:** `pkg/verify/check.go:388`, `pkg/verify/check.go:653`

**Significance:** These are the visible symptoms of incomplete migration. Removing them requires ensuring tier→level mapping happens at spawn time.

---

### Finding 5: Legacy Tier-Based Path Still Used

**Evidence:** `VerifyCompletionWithTierAndComments` (check.go:598) uses the old tier-based gating (no levels). It's called by `VerifyCompletionForReview` (check.go:254) which the `orch review` command uses. This means the review command sees different gate results than the complete command.

The legacy path has its own synthesis gate logic at check.go:650-678 that duplicates the level-aware path's logic at check.go:549-576. Both have the knowledge-producing skill auto-skip.

**Source:**
- `check.go:598-681` — legacy path
- `check.go:254-275` — review path calling legacy path

**Significance:** Two paths to maintain, with divergent behavior. Review shows one set of gate results, complete shows another. Unifying them on the level system eliminates this discrepancy.

---

### Finding 6: Missing Skills in Level Defaults

**Evidence:** Skills observed in stats that are NOT in `SkillVerifyLevelDefaults`:
- `capture-knowledge` (3 spawns) — should be V0 or V1
- `ux-audit` (3 spawns) — should be V1 or V2
- `debug-with-playwright` (2 spawns) — should be V2 or V3
- `probe` (1 spawn) — should be V1

These fall through to the conservative default of V1. For `capture-knowledge` this is too high (it's purely knowledge-producing). For `debug-with-playwright` it may be too low (it likely needs visual verification).

**Source:** `orch stats` skill breakdown, `pkg/spawn/verify_level.go:22-37`

**Significance:** As new skills are added, the level defaults must be updated. Currently this is easy to miss because gates have their own fallback logic.

---

## Synthesis

**Key Insights:**

1. **The architecture is right; the migration is incomplete.** V0-V3 was designed to replace three implicit systems (spawn tier, checkpoint tier, skill-based auto-skips). The design shipped but the old code was never cleaned up. The friction comes from two systems fighting each other, not from the wrong system.

2. **The 107 synthesis bypasses trace to one root cause:** "light tier" spawns don't map to lower verify levels at spawn time. Feature-impl defaults to V2, so even "light" feature-impl gets V2 gates. The orchestrator manually bypasses because the level doesn't reflect the intent.

3. **The 1943 auto-skips are pure waste:** Gates run, execute logic (including git commands in some cases), then decide to skip based on skill class. The level system should prevent them from running at all. Removing internal auto-skip logic and trusting `ShouldRunGate()` eliminates this waste.

4. **Per-gate skill lists are the maintainability hazard.** Eight separate lists encoding the same concept (which skills need which verification) creates drift risk and makes adding new skills error-prone.

**Answer to Investigation Question:**

Gate friction is NOT caused by too many gates or wrong gate logic. The 14 gates are real and well-designed (confirmed by prior audit orch-go-1153). The friction comes from incomplete migration to the V0-V3 level system: (1) "light tier" doesn't map to a lower verify level, causing 107 synthesis bypasses; (2) per-gate skill lists duplicate the level system, causing 1943 auto-skips; (3) ad-hoc tier guards within the level-aware pipeline create inconsistencies. Six concrete changes eliminate the friction without touching gate logic.

---

## Structured Uncertainty

**What's tested:**

- ✅ `ShouldRunGate()` correctly filters gates by level (verified: read `level.go:44-71`, confirmed V0 excludes synthesis)
- ✅ Per-gate skill lists exist in 8 locations (verified: grep across all verify/ files)
- ✅ Synthesis bypass reasons cite "light tier" as primary cause (verified: `orch stats --verbose` output)
- ✅ Pipeline already calls `ShouldRunGate()` before each gate (verified: read `check.go:328-475`)
- ✅ Legacy tier-based path still active for review command (verified: read `check.go:254`)

**What's untested:**

- ⚠️ Removing per-gate skill lists won't create edge cases where a V2 gate should skip for a specific V2 skill (need to audit each gate's internal logic)
- ⚠️ Mapping "light tier" to V0 at spawn time won't break downstream consumers that check `.tier` file (need to verify all `.tier` readers)
- ⚠️ Auto-skip event removal won't break orch stats reporting (stats may depend on auto-skip events for metrics)

**What would change this:**

- If any per-gate skill list encodes a distinction that the level system CANNOT express (e.g., "V2 skill X needs test evidence but V2 skill Y doesn't"), the per-gate lists serve a purpose
- If "light tier" is used for purposes beyond verification (e.g., spawn resource allocation), removing it would have side effects

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Map spawn tier to verify level | architectural | Affects spawn and verify subsystems, changes how tier concept works |
| Remove per-gate skill lists | architectural | Affects all gate implementations, changes verification contract |
| Remove internal auto-skip logic | implementation | Cleanup within existing patterns, no behavior change |
| Remove ad-hoc tier guards | implementation | Cleanup, behavior already handled by level system |
| Deprecate legacy path | architectural | Changes review command behavior, unifies verification paths |
| Add missing skills to level defaults | implementation | Adding data to existing mapping |

### Recommended Approach ⭐

**Complete V0-V3 Migration** — Remove all residual tier-based and skill-list-based gating code, trusting the level system as single source of truth.

**Why this approach:**
- V0-V3 was designed specifically to solve this problem (decision 2026-02-20)
- The architecture is already implemented and working (`ShouldRunGate()` is called correctly)
- Each change is independently safe — can be done incrementally
- Eliminates 107 manual bypasses and 1943 auto-skip events

**Trade-offs accepted:**
- Removing auto-skip logging reduces observability into gate behavior (mitigate: `ShouldRunGate()` could log which gates are skipped by level)
- Removing per-gate skill lists means all V2 skills get ALL V2 gates, no exceptions (this is by design — if a V2 skill shouldn't get a V2 gate, it should be V1)

**Implementation sequence:**
1. Map spawn tier to verify level (foundational — enables all other changes)
2. Add missing skills to level defaults (data fix, immediate value)
3. Remove internal auto-skip logic in synthesis gate (biggest auto-skip source)
4. Remove per-gate skill lists (consolidation, requires V2 skill behavior audit)
5. Remove ad-hoc tier guards (cleanup, depends on #1)
6. Deprecate legacy tier-based path (final migration step)

### Alternative Approaches Considered

**Option B: Add "light" as a V0 alias in the level system**
- **Pros:** Minimal code change, backward compatible
- **Cons:** Perpetuates two naming schemes for the same concept, doesn't clean up per-gate lists
- **When to use instead:** If migration risk is too high for a single effort

**Option C: Keep per-gate skill lists as guard rails alongside levels**
- **Pros:** Defense in depth, catches level misconfiguration
- **Cons:** Perpetuates maintenance burden, generates confusing auto-skip stats
- **When to use instead:** If level system proves unreliable in practice

**Rationale for recommendation:** Option A addresses the root cause (incomplete migration) rather than adding workarounds. The V0-V3 decision explicitly identified this migration as a consequence. The friction data proves the need is real.

---

### Implementation Details

**What to implement first:**
1. `pkg/spawn/`: At spawn time, if `tier == "light"`, set `VerifyLevel = V0` in AGENT_MANIFEST.json (unless overridden by `--verify-level`)
2. `pkg/spawn/verify_level.go`: Add missing skills: `capture-knowledge→V0`, `probe→V1`, `ux-audit→V1`, `debug-with-playwright→V3`

**Things to watch out for:**
- ⚠️ The `knowledgeProducingSkills` list in escalation.go serves a DIFFERENT purpose (escalation routing, not gate selection). Do NOT remove this list — it's not a gate list, it's an escalation classifier.
- ⚠️ Some gates (e.g., test_evidence) check for code changes WITHIN the gate and return early. This is valid change-detection, not skill-based gating. Keep the change-detection logic; remove the skill-based skip logic.
- ⚠️ `orch stats` parses auto-skip events. After removing auto-skip logging, update stats to report based on level-based gate selection instead.

**Areas needing further investigation:**
- The `IsKnowledgeProducingSkill()` auto-skip in the synthesis gate (check.go:559) may be intentionally kept for V1+ knowledge skills that DO require a deliverable but not SYNTHESIS.md specifically. This needs to be validated: do V1 knowledge skills actually need a different kind of synthesis check?

**Success criteria:**
- ✅ `orch stats` shows <10 manual bypasses per week (down from 107/week)
- ✅ Auto-skip count drops to near-zero (from 1943)
- ✅ Adding a new skill requires updating exactly ONE location (verify_level.go)
- ✅ `orch complete` and `orch review` produce consistent gate results
- ✅ All existing tests in pkg/verify/ continue to pass

---

## References

**Files Examined:**
- `pkg/verify/check.go` — Main verification pipeline, gate constants, both legacy and level-aware paths
- `pkg/verify/level.go` — Gate-to-level mapping, `ShouldRunGate()`, `GatesForLevel()`
- `pkg/verify/test_evidence.go` — Test evidence gate with per-gate skill lists
- `pkg/verify/visual.go` — Visual verification gate with per-gate skill lists
- `pkg/verify/build_verification.go` — Build/vet gate with per-gate skill lists
- `pkg/verify/architectural_choices.go` — Architectural choices gate with per-gate skill list
- `pkg/verify/escalation.go` — Escalation logic with `knowledgeProducingSkills`
- `pkg/verify/constraint.go` — Constraint gate (no skill list — uses SPAWN_CONTEXT)
- `pkg/verify/phase_gates.go` — Phase gate (no skill list — uses SPAWN_CONTEXT)
- `pkg/verify/decision_patches.go` — Decision patch limit gate (no skill list)
- `pkg/verify/accretion.go` — Accretion gate (no skill list)
- `pkg/verify/git_diff.go` — Git diff gate (no skill list)
- `pkg/verify/skill_outputs.go` — Skill output gate (uses skill.yaml)
- `pkg/verify/unverified.go` — Unverified work tracking
- `pkg/spawn/verify_level.go` — Verification level constants and skill defaults

**Commands Run:**
```bash
# Get gate friction stats
orch stats
orch stats --verbose
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` — V0-V3 level system design
- **Decision:** `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md` — Gate taxonomy
- **Decision:** `~/.kb/decisions/2026-01-04-gate-refinement-passable-by-gated.md` — Gates must be passable by gated party

---

## Investigation History

**2026-02-28:** Investigation started
- Initial question: Why does the gate system generate 107 bypasses and 1943 auto-skips per week?
- Context: Orientation frame identified gate friction as biggest systemic tax on orchestration throughput

**2026-02-28:** Core finding — migration debt, not design failure
- V0-V3 level system is correctly implemented but old code wasn't cleaned up
- 8 duplicate skill-classification systems identified across verify/ package

**2026-02-28:** Investigation completed
- Status: Complete
- Key outcome: Six concrete implementation tasks identified, all traceable to incomplete V0-V3 migration
