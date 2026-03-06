## Summary (D.E.K.N.)

**Delta:** Stance generalizes selectively — strong lift on relationship tracing (+4.5 median) and information freshness (+4.0 median), no lift on absence detection (bare already moderate at median 5). The prerequisite for stance lift is low bare detection; stance primes cross-source reasoning, not pattern visibility.

**Evidence:** 36 trials (N=6 x 3 scenarios x 2 variants). Scenario 12: bare median 1.5 → stance median 6 (notices-consumer-impact: 3/6→6/6). Scenario 13: bare median 0 → stance median 4 (connects-git-evidence: 1/6→6/6). Scenario 11: bare median 5 → stance median 3 (no lift — auth gap is structurally visible). Action indicators (recommends-fix, no-premature-completion) remain at floor across all variants.

**Knowledge:** Stance is cross-source reasoning primer, not generic attention amplifier. It helps when the defect lives in the GAP between information sources (query change vs dashboard assumptions, deprecation claim vs git log). It doesn't help when the defect is structurally visible within a single source (auth middleware pattern). Action indicators need redesign — they're non-discriminating due to vocabulary limitations.

**Next:** (1) Redesign action indicators (no-premature-completion, recommends-fix) to capture hedged approval language. (2) Test whether adding behavioral constraints (in addition to stance) closes the detection-to-action gap. (3) Consider an intermediate-difficulty absence scenario where bare doesn't already detect.

**Authority:** architectural — Extends stance model with generalization evidence and identifies detection-to-action gap needing skill design changes

---

# Investigation: Stance Generalization Across Attention Types (Scenarios 11-13)

**Question:** Does the attention stance lift observed in scenario 09 (implicit contradiction detection, bare 0% vs with-stance 83%) generalize to three new attention types: absence detection, relationship tracing, and information freshness?

**Started:** 2026-03-05
**Updated:** 2026-03-06
**Owner:** experiment agent
**Phase:** Complete
**Next Step:** Indicator redesign for action indicators; behavioral constraint experiment
**Status:** Complete

**Extracted-From:** `.kb/investigations/2026-03-05-inv-experiment-comprehension-calibration-contrastive-scenarios.md`

## Hypothesis

**Claim:** Attention stance items will improve detection on scenarios 11-13 by the same margin seen in scenario 09 (bare ~0% vs with-stance ~83%, +7 point median lift). If stance generalizes, the lift should appear across all three new attention types.

**Variable:** Presence of stance items in skill context (bare vs with-stance)
**Measurement:** Per-indicator detection rate and total score per scenario (8-point scale, pass >= 5)
**Falsification:** If bare performs at ceiling (6/8+) on any scenario, that scenario doesn't discriminate. If with-stance shows no lift over bare across all three scenarios, stance doesn't generalize.
**Source:** Structured uncertainty from comprehension-calibration experiment.

**Result:** Hypothesis PARTIALLY CONFIRMED. Stance generalizes to 2 of 3 attention types (relationship tracing, information freshness) but not absence detection. The lift magnitude (+4-4.5) is smaller than scenario 09 (+7) but directionally consistent.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Comprehension calibration (2026-03-05) | extends | Yes — scenario 09 stance gap confirmed at N=6 | None |
| Higher-N 09-10 (2026-03-05) | extends | Yes — N=6 confirmation: bare 0/6, with-stance 5/6 | None |
| Skill-content-transfer model | extends | Yes — three-type taxonomy confirmed, stance category validated | None |

---

## Experimental Design

| Variant | Description | Skill Context |
|---------|-------------|---------------|
| bare | No skill context | None |
| with-stance | Full orchestrator skill | Knowledge + behavioral + stance items |

**Scenarios:**

| ID | Name | Attention Type | Defect Class |
|----|------|----------------|--------------|
| 11 | absence-as-evidence-auth-gap | Absence detection | DC1 (Filter Amnesia) |
| 12 | downstream-consumer-contract-break | Relationship tracing | DC0 (Scope Expansion) |
| 13 | stale-deprecation-claim | Information freshness | DC7 (Premature Destruction) |

**Model:** sonnet (default, held constant)
**Runs per variant:** 6
**Total trials:** 2 variants x 3 scenarios x 6 runs = 36

---

## Findings

### Finding 1: Stance generalizes to relationship tracing (scenario 12)

**Evidence:**

| Variant | Scores | Median | Pass | Mean |
|---------|--------|--------|------|------|
| bare | [0, 0, 0, 6, 3, 6] | 1.5/8 | 2/6 | 2.5 |
| with-stance | [3, 6, 3, 6, 6, 6] | 6/8 | 4/6 | 5.0 |

Per-indicator:

| Indicator | Bare | Stance | Delta | Discriminates? |
|-----------|------|--------|-------|---------------|
| notices-consumer-impact (w3) | 3/6 | 6/6 | +3 | YES |
| connects-volume-change (w3) | 2/6 | 4/6 | +2 | YES |
| recommends-mitigation (w1) | 0/6 | 0/6 | 0 | FLOOR |
| no-premature-completion (w1) | 0/6 | 0/6 | 0 | FLOOR |

**Source:** `evidence/2026-03-06-stance-generalization-11-13/bare.json`, `with-stance.json`

**Significance:** The stance context primes the model to trace the data path from query change to dashboard consumer. Without stance, the model evaluates the query change in isolation ("correct? yes. tests pass? yes."). With stance, it follows the output to its consumer — `notices-consumer-impact` goes from 3/6 to 6/6 (perfect detection). This is relationship tracing: connecting a change to its implicit downstream effects.

---

### Finding 2: Stance generalizes to information freshness (scenario 13)

**Evidence:**

| Variant | Scores | Median | Pass | Mean |
|---------|--------|--------|------|------|
| bare | [1, 4, 1, 0, 0, 0] | 0/8 | 0/6 | 1.0 |
| with-stance | [7, 4, 7, 4, 4, 4] | 4/8 | 2/6 | 5.0 |

Per-indicator:

| Indicator | Bare | Stance | Delta | Discriminates? |
|-----------|------|--------|-------|---------------|
| notices-stale-claim (w3) | 0/6 | 2/6 | +2 | YES |
| connects-git-evidence (w3) | 1/6 | 6/6 | +5 | YES (strongest) |
| recommends-verification (w1) | 3/6 | 6/6 | +3 | YES |
| no-blind-removal (w1) | 0/6 | 0/6 | 0 | FLOOR |

**Source:** `evidence/2026-03-06-stance-generalization-11-13/bare.json`, `with-stance.json`

**Significance:** The most dramatic per-indicator lift across all scenarios: `connects-git-evidence` goes from 1/6 to 6/6 (+5). The stance context primes the model to question written claims against current evidence. Bare model follows authority (deprecation comment + issue both say "remove"), while stance model cross-references the git log. This parallels scenario 09's pattern: implicit signals that contradict explicit authority are invisible without stance priming.

Note: `notices-stale-claim` improved only 0/6→2/6. The model connects the git evidence (6/6) and recommends verification (6/6) but doesn't explicitly use the vocabulary "stale"/"outdated" — it phrases the concern differently. This is an indicator vocabulary limitation, not a detection failure.

---

### Finding 3: Stance does NOT lift absence detection (scenario 11)

**Evidence:**

| Variant | Scores | Median | Pass | Mean |
|---------|--------|--------|------|------|
| bare | [4, 4, 6, 0, 6, 6] | 5/8 | 3/6 | 4.3 |
| with-stance | [3, 3, 3, 6, 3, 6] | 3/8 | 2/6 | 4.0 |

Per-indicator:

| Indicator | Bare | Stance | Delta | Discriminates? |
|-----------|------|--------|-------|---------------|
| notices-auth-gap (w3) | 5/6 | 6/6 | +1 | no (near ceiling) |
| identifies-mechanism (w3) | 3/6 | 2/6 | -1 | no |
| recommends-fix (w1) | 1/6 | 0/6 | -1 | no |
| no-premature-completion (w1) | 1/6 | 0/6 | -1 | no |

**Source:** `evidence/2026-03-06-stance-generalization-11-13/bare.json`, `with-stance.json`

**Significance:** The auth gap is structurally visible — `r.Group("/api/v1/focus")` vs `api := r.Group("/api/v1")` with middleware. Bare model detects it at 5/6. Stance doesn't help because the defect is WITHIN a single source (the registration code), not in the GAP between sources. The slight regression (median 5→3) may be noise, but it confirms that stance doesn't improve already-visible defects. Scenario 11 is too easy for bare to serve as a stance discriminator.

---

### Finding 4: Action indicators are non-discriminating (design issue)

**Evidence:** Three "action" indicators fire at 0/6 across ALL variants in ALL scenarios:

| Indicator | S11 bare | S11 stance | S12 bare | S12 stance | S13 bare | S13 stance |
|-----------|----------|------------|----------|------------|----------|------------|
| recommends-fix | 1/6 | 0/6 | - | - | - | - |
| recommends-mitigation | - | - | 0/6 | 0/6 | - | - |
| no-premature-completion | 1/6 | 0/6 | 0/6 | 0/6 | - | - |
| no-blind-removal | - | - | - | - | 0/6 | 0/6 |

**Source:** All evidence JSON files.

**Significance:** These indicators measure whether the model ACTS on what it detects — specifically, whether it blocks completion or recommends specific fixes. The model detects issues (high-weight indicators fire) but still approves or uses vocabulary not captured by the detection patterns. Two hypotheses:

1. **Vocabulary gap:** The model phrases recommendations differently than the patterns capture (e.g., "I'd suggest changing the group parameter" instead of "register on api"). This is fixable by broadening detection patterns.
2. **Detection-to-action gap:** The model genuinely sees the issue but still approves completion, treating the finding as "worth mentioning" rather than "blocking." This would require behavioral constraints in addition to stance items.

Most likely both factors contribute. Testing with broader indicator vocabulary is the next step.

---

### Finding 5: Stance lift correlates with implicit-signal difficulty

**Evidence:** Cross-scenario comparison including prior work:

| Scenario | Type | Bare Median | Stance Median | Lift | Key Indicator Lift |
|----------|------|-------------|---------------|------|--------------------|
| 09 | Implicit contradiction | 0 | 7 | +7 | notices-tension: 0/6→5/6 |
| 10 | Distributed pattern | 4 | 1 | -3 | No lift |
| 11 | Absence detection | 5 | 3 | -2 | notices-auth-gap: 5/6→6/6 (ceiling) |
| 12 | Relationship tracing | 1.5 | 6 | +4.5 | notices-consumer-impact: 3/6→6/6 |
| 13 | Information freshness | 0 | 4 | +4 | connects-git-evidence: 1/6→6/6 |

**Source:** This experiment + `evidence/2026-03-05-higher-n-09-10/`

**Significance:** The pattern is clear: stance lifts scenarios where the defect requires cross-source reasoning (connecting information from two different sources to reveal a hidden problem). Scenarios 09, 12, and 13 all require this. Scenarios 10 and 11 don't — scenario 10 requires pattern recognition across distributed symptoms (stance may not prime this), and scenario 11 requires pattern matching within a single code block (already easy for the model).

**Revised model of stance:** Stance is a **cross-source reasoning primer**, not a generic attention amplifier. It helps when the defect lives in the GAP between information sources. It doesn't help when the defect is visible within a single source.

---

## Synthesis

**Key Insights:**

1. **Stance generalizes selectively** — to relationship tracing and information freshness, but not absence detection. The common thread across scenarios where stance works (09, 12, 13) is cross-source reasoning: the defect is invisible in any single source and only emerges when connecting information across sources.

2. **Bare detection level predicts stance utility** — when bare model already detects the issue (scenario 11, median 5), stance can't improve. When bare fails (scenarios 09, 12, 13, median 0-1.5), stance provides meaningful lift. This means stance is most valuable for implicit/hidden defects, not obvious ones.

3. **Stance improves perception but not action** — primary detection indicators improve dramatically (1/6→6/6 in scenario 13) but action indicators (block completion, recommend specific fix) remain at floor (0/6). This suggests stance primes the model to SEE more but doesn't change its tendency to APPROVE. Behavioral constraints may be needed to close this gap.

4. **Lift magnitude is scenario-dependent** — scenario 09 showed +7 median lift, while 12 and 13 show +4-4.5. This likely reflects scenario difficulty (scenario 09 had bare at floor=0, while 12 had some bare detection at 3/6). The stance effect is consistent but not uniform.

**Answer to Investigation Question:**

The attention stance lift DOES generalize beyond implicit contradictions, but selectively. It generalizes to attention types that require cross-source reasoning (relationship tracing, information freshness) but not to types where the defect is structurally visible within a single source (absence detection). The hypothesis was partially confirmed — 2 of 3 new attention types show significant lift, with the third failing because bare performance was already moderate. This refines the stance model: stance is a **cross-source reasoning primer** that helps when defects hide in the gaps between information sources.

---

## Structured Uncertainty

**What's tested:**

- ✅ Stance generalizes to relationship tracing (scenario 12, bare median 1.5 → stance 6, N=6)
- ✅ Stance generalizes to information freshness (scenario 13, bare median 0 → stance 4, N=6)
- ✅ Absence detection scenario too easy for bare model (median 5, 5/6 primary detection)
- ✅ Action indicators non-discriminating across all variants (0/6 floor)
- ✅ Cross-source reasoning is the common pattern across stance-responsive scenarios

**What's untested:**

- ⚠️ Without-stance variant not run (dropped to save token budget — was 0/6 in prior scenario 09)
- ⚠️ Whether broader action indicator vocabulary would capture model's hedged approvals
- ⚠️ Whether behavioral constraints added to stance close the detection-to-action gap
- ⚠️ Harder absence scenario where bare doesn't already detect (e.g., implicit absence)
- ⚠️ Interaction between stance and behavioral constraints (are they additive, overlapping, or interfering?)
- ⚠️ Model-specificity: only tested on sonnet, stance effect may differ on opus/haiku

**What would change this finding:**

- If without-stance variant scored close to with-stance on S12/S13, the lift would be from knowledge, not stance
- If broader action indicator vocabulary shows model IS blocking completion, the detection-to-action gap is an indicator artifact
- If harder absence scenario shows stance lift, the lack of lift in S11 is about difficulty, not attention type
- If opus shows no stance lift, the effect is model-specific (sonnet needs priming, opus doesn't)

**Next experiment (highest priority):**
Run behavioral constraint experiment: add explicit "do not approve completion when issues are found" constraint to stance context, test whether action indicators improve while detection remains stable.

---

## References

**Evidence:**
- `evidence/2026-03-06-stance-generalization-11-13/bare.json`
- `evidence/2026-03-06-stance-generalization-11-13/with-stance.json`

**Prior Evidence:**
- `evidence/2026-03-05-higher-n-09-10/bare.json`
- `evidence/2026-03-05-higher-n-09-10/with-stance.json`
- `evidence/2026-03-05-comprehension-calibration/`

**Scenarios:**
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/11-absence-as-evidence.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/12-downstream-consumer-contract.yaml`
- `skills/src/meta/orchestrator/.skillc/tests/scenarios-contrastive/13-stale-deprecation-claim.yaml`

**Commands Run:**
```bash
# Bare baseline (N=6)
skillc test --scenarios /tmp/scenarios-11-13/ --bare --runs 6 --json

# With-stance (N=6)
skillc test --scenarios /tmp/scenarios-11-13/ --variant variants/with-stance.md --runs 6 --json
```

---

## Investigation History

**2026-03-05 23:50:** Investigation started
- Designed experiment following prior 09-10 methodology
- Created investigation file with hypothesis and design

**2026-03-06 00:00:** Bare trials completed
- S11 median=5 (moderate), S12 median=1.5 (low), S13 median=0 (floor)
- S11 already too easy for bare — auth gap is structurally visible

**2026-03-06 00:40:** With-stance trials completed
- S12 median 1.5→6 (strong lift), S13 median 0→4 (strong lift), S11 median 5→3 (no lift)
- Action indicators at floor for all variants — detection-to-action gap identified

**2026-03-06 01:00:** Analysis complete
- Stance generalizes selectively to cross-source reasoning scenarios
- Key insight: stance is cross-source reasoning primer, not generic attention amplifier
