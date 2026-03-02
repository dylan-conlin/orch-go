## Summary (D.E.K.N.)

**Delta:** 3-form structural diversity does NOT survive constraint competition for behavioral constraints. At 10 constraints × 3 forms, the delegation constraint regresses to bare parity (5/8, proposes-delegation 0/3). Knowledge constraints (intent) degrade but remain functional (6/8 median, above bare 3/8). The dilution threshold for behavioral constraints is between 2-5 competing constraints.

**Evidence:** 36 skillc test runs (6 variants × 3 runs × 2 scenarios, sonnet model). Delegation: 1C=[8,8,8] → 2C=[8,8,8] → 5C=[3,8,8] → 10C=[5,5,5]. Intent: 1C=[8,6,8] → 2C=[8,6,3] → 5C=[5,8,8] → 10C=[6,8,6]. The proposes-delegation indicator (key behavioral signal) drops: 3/3 → 3/3 → 2/3 → 0/3 as constraints increase.

**Knowledge:** The aj58 "3-form works" finding is a laboratory result — it holds in isolation but degrades under constraint competition. The original baseline finding ("behavioral constraints can't work in full skills") was correct for the wrong reason: it's not structural insufficiency, it's attention budget exhaustion. Each additional constraint divides the model's attention budget, and behavioral constraints are the first casualties because they fight defaults.

**Next:** Route to architect: design constraint priority system for skill documents. Critical behavioral constraints need isolation (separate system prompt section, dynamic injection) rather than competing with 50+ other constraints in a single document.

**Authority:** architectural - Finding affects skill architecture (constraint isolation vs competition), not just content

---

# Investigation: Constraint Dilution Threshold for 3-Form Structural Diversity

**Question:** Does the 3-form structural diversity pattern (table + checklist + examples) survive when multiple constraints compete in the same document? At what constraint count does compliance drop below ceiling (8/8)?

**Defect-Class:** configuration-drift

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** og-inv-test-constraint-dilution-01mar-d0c9
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-01-inv-test-hypothesis-redundancy-saturation-point.md | extends | yes — 3-form = [8,8,8] in isolation confirmed at 1C | **YES** — 3-form ceiling does NOT hold under competition |
| .kb/investigations/2026-03-01-inv-test-hypothesis-constraint-violation-rate.md | confirms | yes — full skill 0% delegation confirmed as endpoint of dilution curve | - |
| .kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md | confirms | yes — bare parity for behavioral constraints is explained by dilution | - |

---

## Test Design

**Approach:** Increasing constraint density variants, all using 3-form structural diversity (table + checklist + anti-patterns).

| Variant | Constraint Count | Total Expressions | Word Count | Contains |
|---------|-----------------|-------------------|------------|----------|
| Bare | 0 | 0 | 0 | Nothing |
| 1C-D | 1 (delegation) | 3 | 196 words | Delegation only |
| 1C-I | 1 (intent) | 3 | 241 words | Intent only |
| 2C | 2 | 6 | 427 words | Delegation + Intent |
| 5C | 5 | 15 | 971 words | Both + 3 fillers |
| 10C | 10 | 30 | 1800 words | Both + 8 fillers |

**Measurement probes:** delegation-probe and intent-clarification-probe (same as aj58)

**Model:** sonnet (matching aj58 for comparability)

**Runs:** 3 per variant (variance measurement)

**Filler constraints** (realistic orchestrator behaviors, irrelevant to measurement probes):
1. Anti-sycophancy: Don't hedge or over-apologize
2. Phase reporting: Report phase transitions via bd comment
3. No bd close: Workers must never run bd close
4. Architect routing: Route hotspot work to architect
5. Session close protocol: Follow exact commit order
6. Beads tracking: Track progress via beads
7. Context loading: Load SPAWN_CONTEXT before acting
8. Tool restriction: Prefer dedicated tools over shell commands

---

## Findings

### Finding 1: The Dilution Curve — Behavioral Constraints Regress to Bare by 10 Constraints

**Evidence:** Full results matrix (sonnet, 3 runs per variant):

**Delegation Probe (behavioral constraint — suppresses default code reading)**

| Variant | Median | Scores | Pass Rate | proposes-delegation | no-code-reading | frames-delegation |
|---------|--------|--------|-----------|--------------------|-----------------|--------------------|
| Bare | 5/8 | [0, 5, 5] | 2/3 | 0/2 | 2/2 | 2/2 |
| 1C-D | **8/8** | **[8, 8, 8]** | 3/3 | **3/3** | 3/3 | 3/3 |
| 2C | **8/8** | **[8, 8, 8]** | 3/3 | **3/3** | 3/3 | 3/3 |
| 5C | 8/8 | [3, 8, 8] | 2/3 | 2/3 | 3/3 | 2/3 |
| 10C | 5/8 | [5, 5, 5] | 3/3 | **0/3** | 3/3 | 3/3 |

**Intent Probe (knowledge constraint — teaches when to pause and ask)**

| Variant | Median | Scores | Pass Rate | asks-clarification | no-immediate-action | offers-interpretations |
|---------|--------|--------|-----------|-------------------|--------------------|-----------------------|
| Bare | 3/8 | [3, 6, 3] | 1/3 | 1/3 | 3/3 | 0/3 |
| 1C-I | 8/8 | [8, 6, 8] | 3/3 | 3/3 | 3/3 | 2/3 |
| 2C | 6/8 | [8, 6, 3] | 2/3 | 2/3 | 3/3 | 1/3 |
| 5C | 8/8 | [5, 8, 8] | 3/3 | 2/3 | 3/3 | 3/3 |
| 10C | 6/8 | [6, 8, 6] | 3/3 | 3/3 | 3/3 | 1/3 |

**Source:** `skillc test --scenarios ... --model sonnet --runs 3 --json --transcripts ...` for each variant. Results in test-artifacts/results/ and transcripts in test-artifacts/transcripts/.

**Significance:** The delegation constraint (behavioral, counter-instinctual) is completely diluted at 10 constraints. The proposes-delegation indicator — the actual behavioral action — drops from 3/3 (ceiling) to 0/3 (bare parity) as constraints increase from 1 to 10. The model still avoids code reading and frames answers as delegation (these are passive/negative behaviors), but it never takes the active step of proposing `orch spawn` or `bd create`.

---

### Finding 2: Behavioral vs Knowledge Constraints Have Different Dilution Sensitivity

**Evidence:**

| Constraint Type | 1C Ceiling | 10C Performance | Regression |
|----------------|-----------|-----------------|------------|
| Delegation (behavioral) | 8/8 [0 variance] | 5/8 = bare parity | **Complete regression** |
| Intent (knowledge) | 8/8 [8,6,8] | 6/8 [6,8,6] | Degraded but functional |

The behavioral constraint (delegation) regresses to bare parity at 10 constraints. The knowledge constraint (intent) degrades from 8/8 to 6/8 but remains meaningfully above bare (3/8).

**Source:** Cross-variant comparison of delegation vs intent medians.

**Significance:** This extends aj58's Finding 3 (knowledge vs behavioral thresholds differ). In isolation, both types achieve ceiling at 3 forms. Under competition, behavioral constraints are dramatically more sensitive to dilution. Knowledge constraints are additive (layered on top of defaults), so they survive attention budget competition better. Behavioral constraints fight defaults, so when attention is divided among competing constraints, the default wins.

---

### Finding 3: The Transition Zone Is Between 2-5 Constraints

**Evidence:** The delegation constraint shows a clear transition:

| Constraint Count | Delegation Scores | proposes-delegation | Status |
|-----------------|-------------------|--------------------|---------|
| 1 | [8, 8, 8] | 3/3 | **Ceiling** |
| 2 | [8, 8, 8] | 3/3 | **Ceiling** |
| 5 | [3, 8, 8] | 2/3 | **Variance returns** |
| 10 | [5, 5, 5] | 0/3 | **Bare parity** |

At 2 constraints, delegation is still at ceiling with zero variance. At 5 constraints, variance returns (one run drops to 3/8). At 10 constraints, the behavior is indistinguishable from bare.

The intent constraint shows earlier variance but doesn't regress to bare:

| Constraint Count | Intent Scores | asks-clarification | Status |
|-----------------|---------------|-------------------|---------|
| 1 | [8, 6, 8] | 3/3 | **Near-ceiling with variance** |
| 2 | [8, 6, 3] | 2/3 | **Variance increases, one bare-level run** |
| 5 | [5, 8, 8] | 2/3 | **Variance stabilizes** |
| 10 | [6, 8, 6] | 3/3 | **Degraded but functional** |

**Source:** Score progressions across variants.

**Significance:** There's no sharp cliff — it's a gradual curve. Behavioral constraints start showing variance at 5 competing constraints and reach bare parity by 10. Knowledge constraints degrade more gracefully. This means the "effective constraint budget" differs by constraint type: behavioral constraints can share a document with ~2-4 other constraints before degradation starts; knowledge constraints tolerate higher density.

---

### Finding 4: 10C Delegation Hits Exact Bare Parity — Not Below It

**Evidence:** At 10 constraints, delegation scores [5, 5, 5] with zero variance. Bare scores [0, 5, 5]. The 10C delegation is actually MORE consistent than bare (which had a 0 in run 1), but the score level and indicator pattern are identical:

| Indicator | Bare | 10C |
|-----------|------|-----|
| proposes-delegation | 0/2 | 0/3 |
| no-direct-code-reading | 2/2 | 3/3 |
| frames-as-delegation | 2/2 | 3/3 |

The model achieves 5/8 from the two passive indicators (avoiding code reading, framing as delegation) without ever taking the active behavioral step (proposing delegation). This is the same pattern as bare: the model's default is not to explicitly propose code reading, so the negative indicators fire without any constraint influence.

**Source:** Per-indicator comparison of bare vs 10C results.

**Significance:** This confirms that at 10 constraints, the 3-form structural diversity for the delegation constraint has ZERO effect. The model behaves identically to having no constraint at all. The 30 constraint expressions (~2000 words) in the 10C document might as well not exist for the delegation behavior.

---

### Finding 5: Cross-Constraint Specificity Still Holds Under Competition

**Evidence:** In the 2C variant (both delegation and intent active), delegation scores [8,8,8] and intent scores [8,6,3]. The constraints don't interfere with each other's TARGET behavior — delegation achieves ceiling on the delegation probe, intent achieves 6/8 median on the intent probe. But they do compete for the model's attention budget.

In the 5C variant, both probed constraints show variance: delegation [3,8,8] and intent [5,8,8]. The filler constraints don't target either probe's behavior, but their presence dilutes the model's attention to the constraints that DO matter.

**Source:** 2C and 5C results compared to 1C baselines.

**Significance:** Constraint competition is about attention budget, not behavioral interference. Filler constraints don't cause the model to violate the delegation rule specifically — they dilute the model's overall attention to ALL constraints, making all of them less reliable. This confirms the "attention budget" hypothesis from aj58's Finding 4.

---

## Synthesis

**Key Insights:**

1. **3-form structural diversity is a laboratory result, not a production solution.** The ceiling compliance (8/8, zero variance) found in aj58 only holds when the constraint is tested in isolation or with 1-2 companions. By 10 constraints, behavioral compliance regresses to bare parity despite 30 constraint expressions using 3 structurally diverse forms. The "constraints can't work" finding from the baseline investigation was empirically correct — the mechanism (dilution) was just misattributed (to structural insufficiency rather than attention budget exhaustion).

2. **Behavioral constraints have a hard budget of ~2-4 co-resident constraints.** The transition from ceiling to bare parity happens between 2 constraints (ceiling, zero variance) and 10 constraints (bare parity). At 5 constraints, variance returns but median is still at ceiling. The effective budget for behavioral constraints is roughly 2-4 competing constraints before reliability drops. Knowledge constraints tolerate higher density (~10+) while remaining functional.

3. **The full orchestrator skill (50+ constraints) is far beyond the dilution threshold.** The production skill has ~50 constraints competing for attention. This investigation shows behavioral constraints degrade at 5 and fail at 10. The gap between 10 (tested upper bound) and 50 (production reality) means behavioral constraints in the production skill are completely non-functional — exactly what the baseline investigation found.

4. **Dynamic enforcement is mandatory, not optional, for behavioral constraints in dense documents.** The aj58 investigation's optimistic conclusion — "the correct response to bare-parity violations is to isolate the constraint and express it in 3 structurally diverse forms" — was only half right. Isolation works, but you can only isolate 2-4 behavioral constraints before dilution returns. For skills with more than ~4 critical behavioral constraints, dynamic enforcement (hooks, tool interception, frame guards) is the only viable path.

**Answer to Investigation Question:**

The 3-form structural diversity pattern does NOT survive constraint competition for behavioral constraints. At 10 constraints × 3 forms (30 expressions, ~2000 words), the delegation constraint regresses to bare parity (5/8, proposes-delegation 0/3). Knowledge constraints (intent) degrade from 8/8 to 6/8 but remain meaningfully above bare.

The dilution curve is gradual, not a cliff:
- **1-2 constraints:** Ceiling compliance, zero/low variance
- **5 constraints:** Variance returns, median still at ceiling for behavioral
- **10 constraints:** Behavioral at bare parity, knowledge degraded but functional
- **50+ constraints (production):** Both types at bare parity (confirmed by baseline investigation)

---

## Structured Uncertainty

**What's tested:**

- ✅ 6 constraint density levels tested (bare, 1C-D, 1C-I, 2C, 5C, 10C) — 36 total runs
- ✅ Same scenarios and model (sonnet) as aj58 for direct comparability
- ✅ 3 runs per variant for variance measurement
- ✅ Both knowledge (intent) and behavioral (delegation) constraints measured
- ✅ Filler constraints are realistic orchestrator behaviors, not random text
- ✅ proposes-delegation indicator provides clear behavioral signal (0/3 at 10C vs 3/3 at 1C)

**What's untested:**

- ⚠️ Only tested on sonnet — opus may show different dilution curve
- ⚠️ 3 runs per variant is statistically noisy — 10+ would provide confidence intervals
- ⚠️ Only 2 probe types tested — other constraint types may have different sensitivity
- ⚠️ Single-turn `--print` mode — interactive multi-turn sessions may differ
- ⚠️ Filler constraints are NOT measured — unknown if fillers achieve compliance in the 5C/10C variants
- ⚠️ No 3C, 4C, 7C, 8C variants — transition zone between 2 and 10 is undersampled
- ⚠️ Constraint ordering may matter — delegation is always first in all variants

**What would change this:**

- If 10+ runs show 10C delegation above bare → current finding is noise from 3-run sample
- If opus shows different curve → model-specific dilution thresholds exist
- If multi-turn testing shows different results → single-turn is not ecologically valid
- If constraint ordering matters → position in document affects dilution sensitivity
- If targeted isolation (e.g., using separate system prompt sections) preserves compliance at higher density → document structure, not just count, determines the budget

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Design constraint priority/isolation system for skills | architectural | Cross-skill design pattern, affects how all skill documents are structured |
| Build dynamic enforcement (hooks) for critical behavioral constraints | architectural | New infrastructure component crossing hooks, skill system, session management |
| Limit behavioral constraints to 2-4 per skill document section | implementation | Tactical guideline within existing patterns |

### Recommended Approach: Layered Enforcement Architecture

**Two-layer approach:** Static constraints for knowledge (they survive dilution), dynamic enforcement for behavior (they don't).

**Why this approach:**
- Knowledge constraints (intent, routing, vocabulary) work via prompt even at 10+ density
- Behavioral constraints (delegation, tool restriction) fail by 5-10 density
- Separating the layers targets each type's actual failure mode
- Infrastructure enforcement (hooks, tool interception) provides hard guarantees for behavioral constraints

**Trade-offs accepted:**
- Higher engineering cost than pure-prompt approach
- Infrastructure enforcement requires session-type detection
- May over-constrain edge cases where behavioral override is appropriate

**Implementation sequence:**
1. **Audit all skill constraints** — classify each as knowledge vs behavioral
2. **Keep knowledge constraints in prompt** — they work at production density, just add 3-form diversity to the critical ones
3. **Move behavioral constraints to hooks/infrastructure** — tool interception (Read/Bash when orchestrator), response-length guards, delegation detectors
4. **Optionally add isolation sections** — if skill needs >4 behavioral constraints in prompt, segment them into dedicated sections with clear priority markers

### Alternative Approaches Considered

**Option B: Constraint budgeting (prompt-only)**
- **Pros:** No infrastructure changes, just discipline about constraint count per document
- **Cons:** Limits each skill to ~4 behavioral constraints — too restrictive for complex skills like orchestrator
- **When to use instead:** For simple skills with <5 total constraints

**Option C: Dynamic constraint injection (context-sensitive)**
- **Pros:** Load only relevant constraints based on detected scenario
- **Cons:** Requires scenario detection before constraint loading — chicken-and-egg problem
- **When to use instead:** If constraint relevance can be predicted from metadata (issue type, skill, phase)

**Rationale for recommendation:** The layered approach is the only one that handles the production orchestrator skill (50+ constraints, 5+ critical behavioral constraints). Pure prompt approaches are proven insufficient. Dynamic injection is promising but architecturally complex. Layered enforcement handles the common case (knowledge in prompt, behavior in hooks) without requiring full scenario detection.

---

## References

**Files Examined:**
- .kb/investigations/2026-03-01-inv-test-hypothesis-redundancy-saturation-point.md — prior isolated 3-form ceiling finding
- .kb/investigations/2026-03-01-inv-test-hypothesis-constraint-violation-rate.md — complexity gradient finding
- .kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md — original bare-parity baseline

**Commands Run:**
```bash
# Bare baseline
skillc test --scenarios scenarios/ --bare --model sonnet --runs 3 --json --transcripts transcripts/

# 1C variants (isolated constraints)
skillc test --scenarios scenarios/ --variant variants/1C-delegation.md --model sonnet --runs 3 --json --transcripts transcripts/
skillc test --scenarios scenarios/ --variant variants/1C-intent.md --model sonnet --runs 3 --json --transcripts transcripts/

# Multi-constraint variants
skillc test --scenarios scenarios/ --variant variants/2C.md --model sonnet --runs 3 --json --transcripts transcripts/
skillc test --scenarios scenarios/ --variant variants/5C.md --model sonnet --runs 3 --json --transcripts transcripts/
skillc test --scenarios scenarios/ --variant variants/10C.md --model sonnet --runs 3 --json --transcripts transcripts/
```

**Related Artifacts:**
- **Probe:** .kb/models/orchestrator-session-lifecycle/probes/2026-03-01-probe-constraint-dilution-threshold.md — probe file for this investigation
- **Workspace:** .orch/workspace/og-inv-test-constraint-dilution-01mar-d0c9/test-artifacts/ — scenarios, variants, results, transcripts
- **Prior (aj58):** .kb/investigations/2026-03-01-inv-test-hypothesis-redundancy-saturation-point.md — 3-form saturation in isolation
- **Prior (xm5q):** .kb/investigations/2026-03-01-inv-test-hypothesis-constraint-violation-rate.md — complexity gradient

---

## Investigation History

**2026-03-01 21:40:** Investigation started
- Initial question: Does 3-form structural diversity survive constraint competition?
- Context: aj58 found 3-form achieves ceiling in isolation, but full skill (50+ constraints) produces 0% delegation. Need to find the dilution threshold.

**2026-03-01 21:45:** Test artifacts created
- 2 scenario YAMLs (reused from aj58)
- 5 variant files (1C-D, 1C-I, 2C, 5C, 10C), all using 3-form structural diversity
- 8 filler constraints designed as realistic orchestrator behaviors

**2026-03-01 21:50-22:05:** Tests executed
- 36 total test runs (6 variants × 3 runs × 2 scenarios)
- All results captured in JSON with transcripts
- 1C results confirmed aj58 findings (control validation)

**2026-03-01 22:05:** Investigation completed
- Status: Complete
- Key outcome: 3-form does NOT survive at 10 constraints for behavioral. Dilution threshold is 2-5 constraints. Dynamic enforcement is mandatory for production skill documents.
