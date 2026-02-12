## Summary (D.E.K.N.)

**Delta:** The core gate tier includes test_evidence and git_diff which are inapplicable to non-code skills, causing 123+ unnecessary bypasses; skill detection is broken (87% "unknown"), defeating the existing skill-based exclusion logic.

**Evidence:** Events log analysis: 789 total bypasses, 338 failures. test_evidence caught 5 real problems (all feature-impl) but was bypassed 106 times. git_diff caught 4 real problems but was bypassed 17 times. 104 bypasses had reason "docs-only change, no tests needed." Skill is "unknown" for 87 of 101 completions.

**Knowledge:** The two-tier (core/quality) split conflates "always enforce" with "code-specific." A three-tier or skill-aware tier system would eliminate most false-positive blocking. The orchestrator-override single-gate limit causes cascading friction (3 separate overrides needed for today's design session).

**Next:** Implement skill-aware gate evaluation: introduce `SkillClass` (code-producing vs knowledge-producing) and make test_evidence, git_diff, and build conditional on skill class. Fix skill name extraction. Allow multi-gate orchestrator-override.

**Authority:** architectural - Changes gate classification system which affects all completion flows across all skills

---

# Investigation: Completion Gate Architecture Review

**Question:** Is the current gate tier classification correct, and what's the minimal gate set that provides value without blocking legitimate work?

**Started:** 2026-02-11
**Updated:** 2026-02-11
**Owner:** Architect agent (orch-go-49023)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-02-06-completion-pipeline-parallel-redesign.md

---

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-06 completion-pipeline-parallel-redesign (decision) | extends | Yes - checked CoreGates/QualityGates in check.go | Current decision doesn't account for skill-class sensitivity |

---

## Findings

### Finding 1: Core gates block legitimate work at a 25:1 bypass-to-catch ratio

**Evidence:**

| Gate | Bypassed | Failed (caught real problem) | Bypass:Catch Ratio |
|------|----------|------------------------------|---------------------|
| test_evidence | 106 | 5 (all feature-impl) | 21:1 |
| git_diff | 17 | 4 (all feature-impl) | 4:1 |
| verification_spec | 198 | 165 (148 feature-impl, 9 debugging) | 1.2:1 |
| build | 64 | 143 (143 systematic-debugging) | 0.45:1 |
| synthesis | 104 | 8 | 13:1 |
| phase_complete | 64 | 7 | 9:1 |
| commit_evidence | 32 | 4 | 8:1 |

**Source:** `~/.orch/events.jsonl` - verification.bypassed (789 events) and verification.failed (338 events)

**Significance:** test_evidence and git_diff have high bypass-to-catch ratios because they catch problems ONLY in code-producing skills (feature-impl, systematic-debugging) but are enforced universally. In contrast, build has a low ratio (0.45:1) — it catches more real problems than it's bypassed, meaning it provides high value. The problem isn't the gates themselves; it's that they're classified as universal ("core") when they're actually code-specific.

---

### Finding 2: Skill detection is broken — 87% of completions have unknown skill

**Evidence:**
- 87 of 101 agent.completed events have skill = "unknown"
- All 789 verification.bypassed events have skill = "unknown"
- Only 14 completions have proper skill attribution (10 feature-impl, 3 systematic-debugging, 1 architect)

**Source:** `~/.orch/events.jsonl` - agent.completed and verification.bypassed events, cross-referenced with skill field

**Significance:** The test_evidence gate already has correct skill-based exclusion logic (`skillsExcludedFromTestEvidence` in `pkg/verify/test_evidence.go:43-51`). The map correctly excludes investigation, architect, design-session, research, codebase-audit, issue-creation, and writing-skills. But this logic is silently defeated when skill extraction returns empty string, because `IsSkillRequiringTestEvidence("")` returns `false` (permissive default). The result: test_evidence PASSES for unknown skills (gate not added to results), but git_diff has NO skill exclusion at all and runs unconditionally for all worker agents.

Wait — this means test_evidence isn't actually blocking investigation/design skills (it returns nil for unknown skills). The real blocker for today's design session must be git_diff and verification_spec, NOT test_evidence. The gate-skips.json confirms: all 3 gates (test_evidence, git_diff, verification_spec) were proactively overridden. The orchestrator may have been skipping test_evidence proactively even though it wouldn't have failed.

---

### Finding 3: The "docs-only" bypass pattern accounts for 270 bypasses (34% of all)

**Evidence:**
- 104 test_evidence bypasses with reason "docs-only change, no tests needed"
- 104 synthesis bypasses with same reason
- 31 build bypasses with same reason
- 31 model_connection bypasses with same reason

These all appear in batched bypass sets — the orchestrator is using `--force` or `--skip-*` clusters to push through agents that produced markdown-only output.

**Source:** Bypass reason analysis from events.jsonl

**Significance:** A third of all bypasses come from a single pattern: knowledge-producing agents (investigation, architect, design-session) that produce only markdown files. These agents correctly produce `.kb/investigations/`, `.kb/decisions/`, or SYNTHESIS.md — their work IS the markdown. The code-oriented gates (test, build, git_diff) are categorically inapplicable. The orchestrator has learned to bulk-skip these but the manual friction remains.

---

### Finding 4: Orchestrator-override only supports one gate, requiring serial overrides

**Evidence:** Today's design session (orch-go-49022) required 3 separate gate skips:
1. test_evidence at 20:29:44
2. git_diff at 20:29:53
3. verification_spec at 20:29:53

Each required a separate `--orchestrator-override` invocation because the flag only accepts a single gate name.

**Source:** `.orch/gate-skips.json` - current contents show 3 separate entries for orch-go-49022

**Significance:** The single-gate override limit means completing an investigation/design agent requires 3+ separate `orch complete` invocations with the overhead of re-running all passing gates each time. This is the direct cause of the "5+ minutes fighting the tooling" friction.

---

### Finding 5: Batch mode enforces core gates including code-specific ones

**Evidence:** `batch_complete_cmd.go:23-30` states:
> Core gates (always run, cannot be skipped):
>   - test_evidence: Test execution evidence
>   - git_diff: Diff matches SYNTHESIS claims

`buildBatchSkipConfig()` in `complete_verify.go:183-188` sets `BatchMode: true` with reason "batch mode - core gates only". The `shouldSkipGate` method at line 116 only skips quality gates in batch mode: `if c.BatchMode && verify.IsQualityGate(gate)`.

**Source:** `cmd/orch/batch_complete_cmd.go`, `cmd/orch/complete_verify.go:107-155`

**Significance:** Batch mode was designed for rapid iteration on trusted agents, but it can't batch-complete investigation/design agents because test_evidence and git_diff (core gates) will block them. The orchestrator must complete these one-by-one with manual overrides.

---

### Finding 6: verification_spec is the highest-friction gate overall

**Evidence:**
- 198 bypasses + 165 failures = 363 total friction events (highest of any gate)
- 139 bypasses have reason "Skip for rollout" — the gate was bypassed during its own rollout
- 148 of 165 failures are feature-impl — the gate catches real problems for code skills
- But it blocks ALL skills (no skill exclusion logic)

**Source:** Events analysis; `cmd/orch/complete_gates.go:164-178` (proofSpecEvaluator runs for all regular agents)

**Significance:** verification_spec is a quality gate (Tier 2) so it IS skippable via `--skip-*` and in batch mode. But because it runs unconditionally regardless of skill class, it creates noise for knowledge-producing skills. This is a lower-priority issue than Finding 1 since it CAN be skipped.

---

## Synthesis

**Key Insights:**

1. **The tier system conflates "universally important" with "code-specific"** - The Core 5 gates are defined by what failure mode they prevent (ghost completions, broken handoffs), but two of them (test_evidence, git_diff) only prevent failures in code-producing skills. Knowledge-producing skills can't ghost-complete with untested code because they don't produce code.

2. **Skill detection failure masks a working exclusion system** - test_evidence already has correct skill exclusion (7 skills excluded). But skill detection fails for 87% of completions, making the exclusion logic unreliable. Fixing skill detection would immediately fix test_evidence. git_diff has no skill exclusion at all and needs one.

3. **The pain is concentrated in investigation/design completion** - The bulk bypass pattern (270 events, 34% of all bypasses) comes from one workflow: completing agents that produce markdown artifacts. This is a design-level mismatch, not a one-off edge case.

**Answer to Investigation Question:**

The core/quality tier split is conceptually correct but under-specified. It needs a third dimension: **skill class**. Gates should be classified not just by tier (core vs quality) but by which skill classes they apply to. The minimal gate set that provides value WITHOUT blocking flow is:

**Universal core (all skills):**
- phase_complete
- commit_evidence
- synthesis

**Code-specific core (code-producing skills only):**
- test_evidence
- git_diff
- build (should be promoted from quality → code-specific core)

**Quality (skippable, all skills):**
- handoff_content, constraint, phase_gate, skill_output, decision_patch_limit, dashboard_health, model_connection

**Code-specific quality (skippable, code-producing only):**
- verification_spec, visual_verification

---

## Structured Uncertainty

**What's tested:**

- ✅ Gate bypass/failure ratios from events.jsonl (concrete counts from 789 bypass + 338 failure events)
- ✅ test_evidence skill exclusion logic exists and is correctly coded (verified: read `pkg/verify/test_evidence.go:43-78`)
- ✅ git_diff has no skill exclusion (verified: read `pkg/verify/git_diff.go` - no skill-based logic)
- ✅ Batch mode enforces core gates including test_evidence/git_diff (verified: read `batch_complete_cmd.go` and `complete_verify.go:107-116`)
- ✅ orchestrator-override accepts single gate only (verified: `SkipConfig.OrchestratorOverride string` is a single string, not a slice)

**What's untested:**

- ⚠️ Root cause of skill detection failure (87% unknown) — need to trace `ExtractSkillNameFromSpawnContext` to find where it breaks
- ⚠️ Whether fixing skill detection alone would eliminate the problem (the proactive bypass pattern may have obscured how many gates would actually self-skip)
- ⚠️ Impact of promoting build from quality to code-specific core (might block legitimate systematic-debugging completions where build failure is the expected state)

**What would change this:**

- If skill detection were 100% reliable, the existing test_evidence exclusion would work and test_evidence bypasses would drop to near-zero
- If investigation/design agents started producing code changes, the code-specific gates would become necessary for them
- If ghost completions from knowledge-producing skills became a problem, the universal core set would need expansion

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Introduce SkillClass and make gates skill-aware | architectural | Changes the gate evaluation system used by all completion flows |
| Fix skill name extraction | implementation | Bug fix within existing patterns |
| Multi-gate orchestrator-override | implementation | Extends existing flag without changing gate semantics |
| Reclassify build as code-specific core | architectural | Changes gate tier classification |

### Recommended Approach ⭐

**Skill-Aware Gate Evaluation** - Introduce a `SkillClass` concept that gates use to determine applicability, making code-oriented gates self-skip for knowledge-producing skills.

**Why this approach:**
- Addresses root cause: gates run when inapplicable, not just gate bypasses being inconvenient
- Preserves existing value: test_evidence/git_diff/build still catch real problems for feature-impl (5+4+143 catches)
- Eliminates 270+ bypasses (34% of all) from the "docs-only" pattern
- Makes batch mode work for investigation/design agents without manual overrides

**Trade-offs accepted:**
- Adds a classification dimension that must be maintained when new skills are added
- Knowledge-producing skills lose the protection against accidentally shipping untested code (acceptable: they don't produce code)

**Implementation sequence:**

1. **Fix skill name extraction** (implementation, quick win)
   - Trace `ExtractSkillNameFromSpawnContext` to find why 87% of completions have unknown skill
   - Likely the workspace path resolution fails to find SPAWN_CONTEXT.md in worktree-based spawns
   - This alone would fix test_evidence for knowledge skills since the exclusion logic already exists

2. **Add SkillClass to verify package** (architectural)
   - Define two classes: `SkillClassCode` and `SkillClassKnowledge`
   - Add `SkillClassForName(skillName string) SkillClass` function
   - Code-producing: feature-impl, systematic-debugging, reliability-testing
   - Knowledge-producing: investigation, architect, design-session, research, codebase-audit, issue-creation, writing-skills
   - Unknown skills: default to code-producing (conservative)

3. **Make git_diff skill-class-aware** (implementation)
   - In `checkGitDiff`, skip when skill class is knowledge-producing
   - git_diff currently has zero skill awareness, unlike test_evidence

4. **Reclassify gate tiers** (architectural)
   - Split CoreGates into `UniversalCoreGates` (phase_complete, commit_evidence, synthesis) and `CodeCoreGates` (test_evidence, git_diff)
   - Update `IsCoreGate` to consider skill class
   - Update batch mode to only enforce universal core + applicable code core

5. **Multi-gate orchestrator-override** (implementation)
   - Change `OrchestratorOverride string` to `OrchestratorOverrides []string`
   - Accept comma-separated values: `--orchestrator-override test_evidence,git_diff,verification_spec`
   - Update `shouldSkipGate` to check the slice

### Alternative Approaches Considered

**Option B: Expand the skill exclusion lists in each gate**
- **Pros:** No structural changes to gate system; tactical, fast
- **Cons:** Duplicates exclusion logic across test_evidence, git_diff, build, verification_spec; doesn't fix batch mode; must maintain N separate lists
- **When to use instead:** If the SkillClass approach is deferred, individual gate exclusion is a valid interim fix

**Option C: Remove test_evidence and git_diff from core entirely**
- **Pros:** Simplest change; immediately fixes batch mode and skip friction
- **Cons:** Loses the protection these gates provide for feature-impl (5+4 real catches); code skills would need manual `--skip-*` reason for what are currently automatic checks
- **When to use instead:** If the data showed these gates never catch real problems (it doesn't — they do catch real problems for code skills)

**Option D: Replace core/quality with per-skill gate profiles**
- **Pros:** Most precise — each skill defines exactly which gates apply
- **Cons:** Over-engineered for current needs; requires maintaining N*M matrix of skill-gate applicability; current issue is binary (code vs knowledge)
- **When to use instead:** If more than two skill classes emerge or gate applicability becomes more nuanced

**Rationale for recommendation:** Option A (SkillClass) gives the right abstraction level — binary classification that eliminates the bulk of friction while preserving all real value. The data clearly shows code vs knowledge as the critical dimension.

---

### Implementation Details

**What to implement first:**
- Fix skill name extraction (step 1) — this alone may eliminate most test_evidence friction since the exclusion logic already exists
- Multi-gate orchestrator-override (step 5) — quick ergonomic win, reduces cascading override friction from 3 invocations to 1

**Things to watch out for:**
- ⚠️ Build gate for systematic-debugging: 143 real failures caught. If build is promoted to code-specific core, ensure debugging agents that intentionally break builds can still complete
- ⚠️ Default for unknown skills: must be conservative (code-producing) to avoid losing protection for new/unclassified skills
- ⚠️ SPAWN_CONTEXT.md path resolution: worktree-based spawns may store SPAWN_CONTEXT.md in workspace/ while agents work in worktrees/; the skill extraction needs to check both

**Areas needing further investigation:**
- Root cause of skill name extraction failure (87% unknown)
- Whether verification_spec should also be skill-class-aware (currently quality tier, can be batch-skipped, lower priority)
- Whether the "docs-only" bypass reason pattern is driven by proactive orchestrator override or actual gate failures

**Success criteria:**
- ✅ Completing investigation/architect/design-session agents requires zero gate overrides
- ✅ Batch mode successfully completes knowledge-producing agents
- ✅ test_evidence and git_diff still block feature-impl agents without evidence
- ✅ Gate bypass rate drops by 30%+ (from current 789 baseline)

---

## References

**Files Examined:**
- `pkg/verify/check.go:20-119` - Gate constants, CoreGates, QualityGates definitions
- `pkg/verify/test_evidence.go:35-78` - Skill exclusion/inclusion maps, IsSkillRequiringTestEvidence
- `pkg/verify/git_diff.go` - Full file, confirmed no skill-based exclusion logic
- `cmd/orch/complete_verify.go:17-37` - SkipConfig struct, OrchestratorOverride (single string)
- `cmd/orch/complete_gates.go:41-235` - verifyCompletion, verifyRegularAgent flow
- `cmd/orch/batch_complete_cmd.go` - Batch mode flow, core gate enforcement
- `.orch/gate-skips.json` - Current skip state showing 3 overrides for orch-go-49022

**Commands Run:**
```bash
# Count total bypass events
cat ~/.orch/events.jsonl | grep -c '"verification.bypassed"'
# Result: 789

# Analyze bypasses by gate and skill
cat ~/.orch/events.jsonl | grep '"verification.bypassed"' | python3 -c "..."
# Result: Top gates - verification_spec: 198, agent_running: 157, test_evidence: 106

# Analyze failures by gate and skill
cat ~/.orch/events.jsonl | grep '"verification.failed"' | python3 -c "..."
# Result: Top combos - verification_spec+feature-impl: 148, build+systematic-debugging: 143

# Analyze completions by skill
cat ~/.orch/events.jsonl | grep '"agent.completed"' | python3 -c "..."
# Result: 87/101 completions have skill=unknown

# Analyze bypass reasons
cat ~/.orch/events.jsonl | grep '"verification.bypassed"' | python3 -c "..."
# Result: 104 bypasses with "docs-only change, no tests needed"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-06-completion-pipeline-parallel-redesign.md` - Established the two-tier gate system
- **Investigation:** Today's triggering example — orch-go-49022 design session blocked by 3 inapplicable gates

---

## Investigation History

**2026-02-11 12:50:** Investigation started
- Initial question: Is the gate tier classification causing unnecessary friction for non-code skills?
- Context: Design session (orch-go-49022) blocked by test_evidence, git_diff, verification_spec — all inapplicable to markdown-only output

**2026-02-11 13:15:** Data collection complete
- Analyzed 789 bypass events, 338 failure events, 101 completion events from events.jsonl
- Key finding: test_evidence bypasses 106 times but only catches 5 real problems (all feature-impl)
- Key finding: skill detection broken — 87% completions have unknown skill

**2026-02-11 13:30:** Investigation completed
- Status: Complete
- Key outcome: Core gates include code-specific gates (test_evidence, git_diff) that should be conditional on skill class. Fix requires skill-class-aware gate evaluation + skill detection fix.
