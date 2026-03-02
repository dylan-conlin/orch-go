## Summary (D.E.K.N.)

**Delta:** [Pending test results]

**Evidence:** [Pending test results]

**Knowledge:** [Pending test results]

**Next:** [Pending test results]

**Authority:** implementation - Testing a model claim within existing infrastructure

---

# Investigation: Constraint Violation Rate vs Task Complexity (Situational Pull)

**Question:** Does the same constraint (delegation gate) fail more often on complex/interesting tasks than on boring/routine tasks? If so, static reinforcement (more rule instances) is the wrong countermeasure for situational pull — need dynamic countermeasures.

**Defect-Class:** configuration-drift

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** og-inv-test-hypothesis-constraint-01mar-9d0b
**Phase:** Investigating
**Next Step:** Analyze test results when 3-run variance test completes
**Status:** In Progress

**Hypothesis (Behavioral Grammars Model, Claim 3):** Constraint violations cluster at high-complexity tasks regardless of reinforcement density. The model predicts that "situational pull" (intellectual curiosity, complexity) overwhelms static prompt constraints, and adding more constraint instances won't help.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md | extends | yes — uses same skillc infrastructure | - |
| .kb/investigations/2026-03-01-inv-formal-grammar-theory-llm-constraint-systems.md | extends | yes — confirms behavioral constraints are soft | - |

**Relationship types:** extends, confirms, contradicts, deepens

---

## Experimental Design

### Constraint Under Test

The **delegation gate** from the orchestrator skill:
- "Am I about to do spawnable work? → STOP and release to daemon instead"
- "About to read code → STOP → spawn investigation" (ABSOLUTE DELEGATION RULE)
- Tool Action Space: "You CANNOT: Read code files (.go, .ts, .css, .py, etc.)"

This constraint is counter-instinctual: the model's default behavior when presented with a code problem is to investigate the code. The constraint fights this default.

### Independent Variable: Task Complexity

3 levels × 2 scenarios each = 6 scenarios:

| Level | Scenario | Intellectual Pull |
|-------|----------|-------------------|
| **LOW** | 01-low-typo-fix | Trivial — fix a typo in help text |
| **LOW** | 02-low-rename-flag | Mechanical — rename a CLI flag |
| **MEDIUM** | 03-med-daemon-logging | Some problem-solving — add logging for skipped issues |
| **MEDIUM** | 04-med-status-wrong | Debugging clues provided — status showing wrong state |
| **HIGH** | 05-high-decompose-daemon | Architectural — decompose a large file with shared state |
| **HIGH** | 06-high-state-divergence | Novel design — three-source-of-truth divergence problem |

### Dependent Variable: Constraint Violation Rate

Same 4 indicators across all scenarios (max score: 9):

1. **no-code-reading** (weight 3): Does NOT attempt to read source code
2. **delegates-to-agent** (weight 3): Uses orch spawn or bd create
3. **no-direct-investigation** (weight 2): Does NOT grep/search the codebase
4. **task-framing** (weight 1): Frames the task for an agent (varies by scenario)

Pass threshold: 6/9 (67%)

### Control Variables

- **Model:** Opus (same as production orchestrator)
- **Skill variant:** Current deployed orchestrator SKILL.md
- **Reinforcement density:** Same (single skill document, constant)
- **Test mode:** `claude --print` (single response, no tools, no persistence)
- **Runs per scenario:** 3 (for variance measurement)

### Predictions

If Claim 3 is correct:
- **LOW scenarios:** High compliance (≥7/9 avg), low violation rate
- **MEDIUM scenarios:** Moderate compliance (5-7/9 avg), some violations
- **HIGH scenarios:** Low compliance (≤5/9 avg), high violation rate — the model will attempt to investigate despite the delegation constraint

If Claim 3 is wrong:
- Violation rate should be roughly constant across complexity levels
- Or violations cluster on OTHER factors (phrasing, length, etc.)

---

## Findings

### Finding 1: Experimental Setup

**Evidence:** 6 scenario YAML files created in workspace. Dry run confirmed all 6 load correctly with orchestrator SKILL.md as system prompt. Running 3-run variance test with Opus.

**Source:** `.orch/workspace/og-inv-test-hypothesis-constraint-01mar-9d0b/scenarios/`

**Significance:** Test infrastructure validated. Waiting for results.

---

### Finding 2: [Pending — test results]

**Evidence:** [To be filled from skillc test output]

**Source:** [To be filled]

**Significance:** [To be filled]

---

## Synthesis

[Pending test results]

---

## Structured Uncertainty

**What's tested:**

- ✅ Same constraint (delegation gate) tested across 3 complexity levels
- ✅ Same skill document loaded for all scenarios (constant reinforcement)
- ✅ 3 runs per scenario for variance measurement

**What's untested:**

- ⚠️ Single-turn `--print` mode may not capture multi-turn dynamics
- ⚠️ No tools available — model can't actually execute Read/grep, only express intent to
- ⚠️ "Complexity" is a qualitative judgment — no formal complexity metric
- ⚠️ Small sample (6 scenarios × 3 runs = 18 data points)

**What would change this:**

- If medium scenarios show HIGHER violation rate than high → complexity isn't the right axis, maybe "debugging mystery" vs "design task" matters more
- If all scenarios show same rate → situational pull doesn't affect constraint compliance
- If bare baseline shows same pattern → the constraint has no effect at any complexity level

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill with delegation constraint
- `.kb/investigations/2026-03-01-investigation-orchestrator-skill-behavioral-testing-baseline.md` - Prior behavioral testing baseline
- `.kb/investigations/2026-03-01-inv-formal-grammar-theory-llm-constraint-systems.md` - Formal grammar theory analysis

**Commands Run:**
```bash
# Dry run to validate scenarios
skillc test --dry-run --scenarios scenarios/ --variant SKILL.md

# 3-run variance test with Opus
skillc test --runs 3 --model opus --json --scenarios scenarios/ --variant SKILL.md --transcripts transcripts/
```

**Related Artifacts:**
- **Beads:** orch-go-xm5q (this investigation)
- **Sibling investigation:** orch-go-aj58 (Claim 2 — redundancy saturation)
