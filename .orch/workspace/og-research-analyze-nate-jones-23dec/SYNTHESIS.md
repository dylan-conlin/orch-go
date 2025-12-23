# Session Synthesis

**Agent:** og-research-analyze-nate-jones-23dec
**Issue:** orch-go-ndgj
**Duration:** 2025-12-23 (single session)
**Outcome:** success

---

## TLDR

Analyzed Nate Jones article on "LLM psychosis" (AI validation loops) to identify warning signs and adversarial frameworks. Found that experts are more vulnerable than novices to validation loops, and our current verification doesn't detect confirmatory prompting (AI verifying AI work). Recommended three-phase defense: adversarial spawn prompts, validation loop detection in orch complete, and progress checkpoints.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-analyze-nate-jones-article-llm.md` - Full investigation with 5 findings, synthesis, and implementation recommendations

### Files Modified
- None (research task, no code changes)

### Commits
- Pending: Will commit investigation file before completion

---

## Evidence (What Was Observed)

- **David Budden case study**: Former DeepMind Director (PhD, MIT/Harvard postdocs, 6 years at DeepMind) bet $45K publicly that he solved two Millennium Prize problems using ChatGPT and Lean formal verification. Mathematical community response: "brutal" - his Lean code may prove a trivial/different version, not the actual Clay problem. Prediction market odds: single digits.

- **Three warning signs identified** (from article):
  1. Confirmatory prompting disguised as verification (asking AI to verify its own work)
  2. Operating beyond audit capacity (working in domains where you can't independently verify)
  3. "Me and the AI versus everyone else" dynamic (dismissing expert criticism)

- **Expert vulnerability**: Article states "research shows AI explanations increase trust even when completely wrong, and the most vulnerable people aren't novices" - experts can rationalize AI outputs using domain knowledge.

- **Current verification gap**: pkg/verify/check.go:20-50 only checks that deliverables exist and commits happened - doesn't detect validation loops or confirmatory prompting.

- **10-prompt adversarial framework** outlined (full text paywalled):
  - Pre-work rules: Before You Start, Project Ground Rules
  - Ongoing checks: Adversarial Mini-Check, 2-Minute Reality Check
  - Structured verification: Audit Boundary Check, Disconfirmation Pass, Reality Check
  - Final gates: Confidence Calibration, Final Gate, Full Assessment

### Tests Run
None (research task, no code to test)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-analyze-nate-jones-article-llm.md` - Investigation documenting validation loop patterns and defense framework

### Decisions Made
- **Phased adversarial verification is the recommended approach** because it's the only option that addresses confirmatory prompting (core failure mode) while scaling with multi-agent velocity. Post-completion review, human-in-loop, and formal tools don't prevent loops or don't scale.

### Constraints Discovered
- **Credentials don't protect against validation loops** - The Budden case proves PhD-level expertise and years at top institutions don't prevent AI validation loops. In fact, experts are MORE vulnerable because they can rationalize outputs.
- **Verification can't rely on AI-generated explanations** - Automation bias is amplified by AI's ability to generate convincing explanations even when wrong.
- **Speed amplifies validation loop risk** - Our orchestration system optimizes for velocity (multi-agent parallel work), but velocity scales validation loop impact.

### Externalized via `kn`
- Will run after committing investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created and filled)
- [x] Tests passing (N/A - research task)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-ndgj` (after commit)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- **How to get full adversarial prompt text?** - Article is paywalled, only have framework structure. Could reverse-engineer from principles or subscribe to get full prompts.

- **Do validation loops occur in typical development work?** - Budden case is extreme (Millennium Prize math). Unclear if the same patterns apply to everyday feature development or only high-complexity domains.

- **What's the optimal checkpoint frequency?** - Framework suggests "ongoing" checks but doesn't specify cadence. Every commit? Every hour? Phase transitions only?

- **Should validation loop detection integrate with beads?** - If detection flags a session, should it update beads issue status automatically? Flag for orchestrator review?

**Areas worth exploring further:**

- **Empirical measurement of validation loop frequency** - We have patterns but no data on how often this occurs in practice for code (vs. math proofs)

- **False positive rate for detection** - Need to test validation loop detection to understand how often it flags valid self-review

- **Agent response to adversarial prompts** - Will agents become overly cautious if spawn context includes adversarial verification rules? Need to balance rigor with velocity.

**What remains unclear:**

- **Whether the math-to-code generalization holds** - Assuming validation loops in mathematical proofs apply to software development, but haven't validated this empirically

- **Token budget impact** - Adversarial prompts will increase spawn context size, unclear how much this affects session token usage

---

## Session Metadata

**Skill:** research
**Model:** claude-opus-4 (assumed, not verified)
**Workspace:** `.orch/workspace/og-research-analyze-nate-jones-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-analyze-nate-jones-article-llm.md`
**Beads:** `bd show orch-go-ndgj`
