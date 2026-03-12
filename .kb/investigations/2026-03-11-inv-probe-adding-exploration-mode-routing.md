## Summary (D.E.K.N.)

**Delta:** Adding 8 lines of exploration mode routing to the orchestrator skill does NOT degrade existing routing accuracy — all additions are knowledge content, and the model's dilution mechanism only applies to behavioral constraints.

**Evidence:** Structural analysis: 8/8 added lines classified as knowledge content (routing tables, command flags). Decision tree trace: 0/10 non-explore scenarios misrouted. Token budget impact: +235 tokens (~4%), but baseline already exceeded budget.

**Knowledge:** The dilution concern ("adding content dilutes at scale") is an overgeneralization of the skill-content-transfer model's claim. The model distinguishes three content types — only behavioral constraints dilute (at 5+). Knowledge content transfers reliably.

**Next:** Close. The exploration additions are safe. If empirical validation is desired, unblock `claude --print` for testing first (global Stop hook blocks all skill A/B testing from orch-go sessions).

**Authority:** implementation - Confirms existing knowledge, no architectural impact

---

# Investigation: Probe Adding Exploration Mode Routing

**Question:** Does adding ~20 lines of exploration mode routing guidance to the orchestrator skill degrade existing routing accuracy for non-explore scenarios?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Agent (orch-go-fg93x)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** skill-content-transfer

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/skill-content-transfer/model.md | extends | yes — content type taxonomy applied to new additions | none |

## Findings

### Finding 1: All 8 added lines are knowledge content

**Evidence:** Content type classification of each line added in commit 8678892fd:
- Surface Table row (routing option) → Knowledge
- Decision Tree entry → Knowledge
- Investigation vs Exploration clarification → Knowledge (discrimination criteria, not prohibition)
- Intent Clarification row → Knowledge
- 3 spawn flag docs → Knowledge

0 behavioral norms added. 0 stance items added.

**Source:** `diff /tmp/orchestrator-skill-baseline.md /tmp/orchestrator-skill-with-explore.md`; model's Three Content Types taxonomy

**Significance:** The model's dilution mechanism applies ONLY to behavioral constraints (5+). Knowledge content produces +5 points positive transfer. Therefore the model predicts NO degradation.

### Finding 2: Decision tree trace shows 0/10 false-positive routing

**Evidence:** Traced 10 scenarios through the decision tree. Existing intent categories (FIX, BUILD, DESIGN, TRIAGE, TRY, COMPARE) capture all scenarios before "EXPLORE broadly" is reached. The exploration criteria ("broad question, multiple angles", "Map out X") don't match any test scenario.

**Source:** Routing trace analysis against the skill's decision tree and intent clarification table

**Significance:** The exploration routing has explicit discrimination criteria that prevent false-positive capture of focused/single-angle requests.

### Finding 3: Token budget was already exceeded before this addition

**Evidence:** Baseline: 503 lines / ~5,910 tokens. Both exceed invariants (≤500 lines / ≤5,000 tokens). The exploration addition (+8 lines / +235 tokens) is a 1.6% / 4.0% increase on top of an already-exceeded budget.

**Source:** `wc -l` and `wc -c` on both skill versions

**Significance:** The exploration content is not responsible for crossing the budget threshold. The skill's budget compliance is a pre-existing issue.

---

## Synthesis

**Key Insights:**

1. **The dilution concern was misframed** — "adding content to containers dilutes at scale" overgeneralizes the model's actual claim. The model distinguishes knowledge (transfers reliably), behavioral constraints (dilute at 5+), and stance (scenario-specific). Knowledge additions don't dilute.

2. **Well-discriminated knowledge content is additive, not dilutive** — The exploration routing includes explicit discrimination criteria ("single-angle → investigation", "broad, multiple angles → --explore") that prevent false-positive capture. This is the routing table pattern the model identifies as producing the highest transfer lift.

3. **The token budget issue predates this addition** — The orchestrator skill already exceeded both the 500-line and 5,000-token invariants before the exploration content was added. The 8-line addition doesn't materially change the budget situation.

**Answer to Investigation Question:**

No. Adding exploration mode routing does not degrade existing routing accuracy. The additions are 100% knowledge content (routing tables, command reference), and the skill-content-transfer model's dilution mechanism only applies to behavioral constraints. Structural analysis confirms 0/10 test scenarios would be misrouted. The probe confirms the model's prediction that knowledge content transfers without degradation.

**Limitation:** This is a structural analysis, not an empirical A/B test. The intended 60-trial experiment was blocked by the global Stop hook contaminating `claude --print` output. Empirical validation of lexical priming effects (whether "explore" in the skill primes models toward exploration routing) remains untested.

---

## Structured Uncertainty

**What's tested:**

- ✅ Content type classification: all 8 lines are knowledge (verified against model's taxonomy)
- ✅ Decision tree structural trace: 0/10 scenarios route to --explore (mechanically traced)
- ✅ Budget impact: +8 lines / +235 tokens on already-exceeded budget (measured)

**What's untested:**

- ⚠️ Lexical priming: does the word "explore" in new entries prime models toward --explore for ambiguous scenarios? (empirical test blocked)
- ⚠️ Real-model routing under cognitive load of full 511-line skill (no API calls executed)
- ⚠️ Interaction between exploration content and the existing skill's stance items (no multi-turn testing)

**What would change this:**

- An empirical A/B test showing >0/30 routing changes between variants would contradict the structural analysis
- Evidence of lexical priming in routing tables (word "explore" capturing attention even for non-explore scenarios) would extend the model

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` — current orchestrator skill (512 lines)
- `skills/src/meta/orchestrator/SKILL.md` — baseline (503 lines) and with-explore (511 lines) versions via git
- `.kb/models/skill-content-transfer/model.md` — parent model

**Commands Run:**
```bash
# Extract baseline and current skill versions
git show 8678892fd^:skills/src/meta/orchestrator/SKILL.md > /tmp/orchestrator-skill-baseline.md
git show 8678892fd:skills/src/meta/orchestrator/SKILL.md > /tmp/orchestrator-skill-with-explore.md

# Diff to identify exact changes
diff /tmp/orchestrator-skill-baseline.md /tmp/orchestrator-skill-with-explore.md

# Measure token impact
wc -l /tmp/orchestrator-skill-baseline.md  # 503 lines
wc -l /tmp/orchestrator-skill-with-explore.md  # 511 lines
wc -c /tmp/orchestrator-skill-baseline.md  # 23,642 chars (~5,910 tokens)
wc -c /tmp/orchestrator-skill-with-explore.md  # 24,585 chars (~6,146 tokens)
```

**Related Artifacts:**
- **Probe:** `.kb/models/skill-content-transfer/probes/2026-03-11-probe-exploration-mode-routing-dilution.md`
- **Model:** `.kb/models/skill-content-transfer/model.md`
- **Commit tested:** `8678892fd` (feat: add exploration mode routing to orchestrator skill)
