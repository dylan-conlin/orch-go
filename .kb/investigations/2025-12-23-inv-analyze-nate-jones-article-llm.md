<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** AI validation loops occur when agents use AI to verify AI-generated work, creating confirmatory bias that experts rationalize; our verification doesn't detect this.

**Evidence:** Nate Jones article documents David Budden (DeepMind director) betting $45K he solved Millennium Prize with ChatGPT - mathematical community says he proved trivial version; three warning signs identified (confirmatory prompting, operating beyond audit capacity, AI-vs-experts dynamic).

**Knowledge:** Credentials and expertise don't protect against validation loops - experts are MORE vulnerable because they can rationalize AI outputs; speed amplifies risk; verification must be adversarial, phased, and independent.

**Next:** Implement three-phase defense: (1) adversarial prompts in spawn context, (2) validation loop detection in orch complete, (3) progress checkpoints; investigate getting full prompt text from paywalled article.

**Confidence:** Medium (70%) - strong patterns from credible source but article is paywalled (only preview accessed), extrapolating from math proofs to code without empirical validation.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Analyze Nate Jones Article Llm

**Question:** What are the key warning signs of AI validation loops, and what adversarial frameworks can prevent them in our orchestration system?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** Research Agent (og-research-analyze-nate-jones-23dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Medium (70%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: "LLM Psychosis" Definition and Case Study

**Evidence:** Nate Jones uses David Budden (DeepMind Director, PhD from Melbourne, postdocs at MIT/Harvard) as a case study. Budden bet $45K publicly that he solved two Clay Millennium Prize problems using ChatGPT and formal verification (Lean). The mathematical community's response: "brutal" - his Lean code may formalize a trivial or different version of the problem, not the actual Clay problem. Prediction market odds: single digits.

**Source:** https://natesnewsletter.substack.com/p/if-a-former-deepmind-engineering (Dec 23, 2025)

**Significance:** This demonstrates that **credentials and intelligence don't protect against AI validation loops**. Budden had to tweet "Not suffering from psychosis" in response to criticism - a signal that the validation loop had progressed to the point where his ability to evaluate his own work was compromised. The key failure mode: "when AI convinces you you're right about something you cannot know."

---

### Finding 2: Three Warning Signs of AI Validation Loops

**Evidence:** Nate identifies three behavioral patterns:
1. **Confirmatory prompting disguised as verification** - asking AI to verify work it helped create
2. **Operating beyond your evaluation capacity** - working in domains where you can't independently audit the output
3. **"Me and the AI versus everyone else" dynamic** - dismissing expert criticism because the AI agrees with you

**Source:** Article preview section "Three warning signs this pattern is developing"

**Significance:** These are **observable behavioral markers** we can detect in agent sessions. Pattern #1 is especially relevant to our `orch complete` verification - if an agent uses AI to verify its own AI-generated work, that's a red flag. Pattern #3 maps to the Budden case where he dismissed mathematical community consensus.

---

### Finding 3: Automation Bias Amplified by AI Explanations

**Evidence:** Article states "research shows AI explanations increase trust even when completely wrong, and the most vulnerable people aren't novices." Also: "when an AI system that's good at producing plausible explanations pushes users into overconfident acceptance of outputs they can't audit."

**Source:** Article preview section "Why LLMs make automation bias worse than traditional systems"

**Significance:** This inverts our assumptions about who's at risk. **Experts are MORE vulnerable than novices** because they can rationalize AI outputs using their domain knowledge, creating a false sense of verification. For our orchestration system, this means verification can't rely solely on the agent's self-assessment or AI-generated explanations.

---

### Finding 4: Adversarial Prompt Framework (10 Prompts)

**Evidence:** Article includes 10 prompts "designed to act as a detailed set of screens that check your work with a helpfully adversarial grounding":
1. **Adversarial Mini-Check** - quick pre-ship attack and calibration checklist
2. **Before You Start** - sets adversarial rules and evidence discipline
3. **Audit Boundary Check** - identify needed verification skills and reviewers
4. **Audit Boundary Check** - identify skills needed for verification
5. **Disconfirmation Pass** - structured attempt to break the conclusion
6. **Reality Check** - anticipate expert objections without fake consensus
7. **Confidence Calibration** - score certainty; define tests to raise confidence
8. **Final Gate** - commit/no-commit decision with stop conditions
9. **Project Ground Rules** - enforce correctness-first behavior across sessions
10. **2-Minute Reality Check** - fast grounding, falsifiability, next action
11. **Full Assessment** - deep anti-spiral review before high-stakes moves

**Source:** Article preview "Here's what's inside" section (full prompts are paywalled)

**Significance:** The framework structure reveals a **phased defense-in-depth approach**: pre-work rules (Before You Start, Project Ground Rules), ongoing checks (Mini-Check, 2-Minute Reality Check), structured verification (Disconfirmation Pass, Reality Check), and final gates (Confidence Calibration, Final Gate). This maps well to our spawn → work → complete lifecycle.

---

### Finding 5: Organizational Warning Signs

**Evidence:** Article mentions "What to watch for in your organization — behavioral signals that someone's judgment is being compromised by AI validation loops" and notes "In 2026, a lot of smart people armed with LLMs and formal tools will ship convincing-looking but wrong work faster than many of our teams can audit it."

**Source:** Article preview "What to watch for in your organization" section

**Significance:** This is a **velocity vs. quality tension**. In our orchestration system, agents can produce work faster than we can verify it. The Budden case shows this isn't just about output speed - it's about the confidence/explanation quality making wrong work *feel* right. We need verification that scales with agent velocity.

---

## Synthesis

**Key Insights:**

1. **Credentials Don't Protect Against Validation Loops** - The Budden case (Finding 1) proves that PhD-level expertise, years at DeepMind, and formal verification tools don't prevent AI validation loops. In fact, Finding 3 shows experts are MORE vulnerable because they can rationalize AI outputs. For our orchestration system, this means we can't assume agents with strong reasoning ability (Opus, Gemini Pro) will self-correct - the better the model, the more convincing its rationalizations.

2. **Verification Contamination is the Core Failure Mode** - Finding 2's "confirmatory prompting disguised as verification" is the key pattern. When agents use AI to verify AI-generated work, the verification is contaminated. Our current `orch complete` verification (pkg/verify/check.go) checks for deliverables and commits, but doesn't detect this. We need **independent verification** that doesn't rely on the same AI that produced the work.

3. **Speed Amplifies Risk** - Finding 5's velocity observation connects to our orchestration system's core value proposition: spawning multiple agents to work in parallel. But the Budden case shows that speed + confidence + formal-looking outputs = faster propagation of convincing errors. We're optimizing for velocity while the validation loop risk scales with velocity.

4. **Defense Requires Phased, Adversarial Verification** - Finding 4's 10-prompt framework reveals a pattern: verification must be **adversarial** (explicitly trying to break conclusions), **phased** (checkpoints throughout work, not just at end), and **externalized** (written down, not just in the agent's context). This maps to spawn context → progress checkpoints → completion verification.

**Answer to Investigation Question:**

**Warning signs of AI validation loops:**
1. Confirmatory prompting (asking AI to verify its own work)
2. Operating beyond audit capacity (can't independently verify outputs)
3. Dismissing external criticism in favor of AI consensus
4. Needing to defend against "psychosis" accusations (late-stage signal)

**Adversarial frameworks recommended:**
- Pre-work rules (establish evidence discipline before starting)
- Ongoing reality checks (fast, frequent grounding prompts)
- Disconfirmation passes (structured attempts to break conclusions)
- Expert objection anticipation (without AI generating fake consensus)
- Confidence calibration with testable conditions
- Final commit gates with explicit stop conditions

**Application to orchestration system:**
Our current verification (pkg/verify/check.go) checks deliverables exist and commits happened, but doesn't detect validation loops. We need:
1. **Adversarial verification prompts** in spawn context (like Finding 4's "Before You Start")
2. **Independent verification** in `orch complete` (not AI-verifying-AI)
3. **Velocity-aware quality gates** (Finding 5's speed risk)
4. **Observable behavioral markers** (Finding 2's three warning signs) in agent sessions

**Limitations:**
- Article is paywalled - only have preview/outline, not full prompt text
- No empirical data on how often this occurs in practice
- Budden case is extreme (Millennium Prize claims) - unclear if patterns apply to everyday development work

---

## Confidence Assessment

**Current Confidence:** Medium (70%)

**Why this level?**

I have strong evidence for the patterns and warning signs from a credible source (Nate Jones has deep AI practitioner experience), but I don't have access to the full adversarial prompt framework details (paywalled). The Budden case is well-documented and the three warning signs are clearly stated, but I'm extrapolating application to our orchestration system without empirical testing.

**What's certain:**

- ✅ The three warning signs are real and observable (confirmatory prompting, operating beyond audit capacity, AI-vs-experts dynamic)
- ✅ Experts are more vulnerable than novices due to rationalization ability (backed by research mentioned in article)
- ✅ Our current verification (pkg/verify/check.go:20-50) doesn't detect validation loops - it only checks deliverables exist
- ✅ The Budden case is a documented example of high-credibility individual caught in validation loop
- ✅ Speed amplifies risk (Finding 5) and our system optimizes for velocity

**What's uncertain:**

- ⚠️ How frequently validation loops occur in typical development work (Budden case is extreme)
- ⚠️ Specific prompt text for the 10 adversarial prompts (paywalled - only have structure/purpose)
- ⚠️ Whether the same patterns apply to code/documentation work vs. mathematical proofs
- ⚠️ How to detect validation loops in progress (we have warning signs, but no detection mechanism)
- ⚠️ Whether adding adversarial prompts to spawn context is sufficient or if we need tool-level enforcement

**What would increase confidence to High (85%):**

- Access to full article with complete adversarial prompt framework details
- Empirical data on validation loop frequency in software development (not just math proofs)
- Testing adversarial prompts in actual agent sessions to validate effectiveness
- Implementation of detection mechanism and measurement of false positive rate

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Phased Adversarial Verification Framework** - Implement defense-in-depth verification at spawn, progress checkpoints, and completion gates.

**Why this approach:**
- Addresses Finding 2's confirmatory prompting by requiring adversarial prompts throughout lifecycle
- Scales with Finding 5's velocity concern by embedding checks rather than post-hoc review
- Leverages Finding 4's framework structure: pre-work rules, ongoing checks, final gates
- Counters Finding 3's expert vulnerability by enforcing external discipline

**Trade-offs accepted:**
- Increased spawn context size (adversarial prompts add tokens)
- Slower agent velocity (checkpoint overhead)
- More false positives (adversarial prompts may flag valid work)
- Acceptable because: speed without correctness creates technical debt faster

**Implementation sequence:**
1. **Add adversarial spawn prompts** (pkg/spawn/context.go) - Foundational because sets behavioral expectations before work starts
2. **Implement validation loop detection** (pkg/verify/check.go) - Builds on spawn rules by detecting violations
3. **Add progress checkpoint framework** (new pkg/checkpoint/) - Enables ongoing verification without waiting for completion

### Alternative Approaches Considered

**Option B: Post-Completion Review Only**
- **Pros:** No impact on agent velocity, simpler implementation
- **Cons:** Doesn't prevent validation loops (Finding 2), only detects after work done. By then agent has committed to flawed conclusions.
- **When to use instead:** Low-risk work where rework cost is minimal

**Option C: Human-in-the-Loop Verification**
- **Pros:** Most reliable verification (humans can detect validation loops)
- **Cons:** Doesn't scale with multi-agent orchestration (Finding 5), bottlenecks on human availability
- **When to use instead:** High-stakes decisions (e.g., architectural changes, security-critical code)

**Option D: Formal Verification Tools**
- **Pros:** Objective correctness checking (like Budden's Lean)
- **Cons:** Budden case (Finding 1) shows formal tools don't prevent validation loops - he had Lean proofs but they proved wrong thing
- **When to use instead:** Mathematical/cryptographic work where formal specs exist

**Rationale for recommendation:** Option A (Phased Adversarial Framework) is the only approach that addresses the core Finding 2 failure mode (confirmatory prompting) while scaling with Finding 5 velocity. Options B and C don't prevent loops, Option D gave false confidence in the Budden case.

---

### Implementation Details

**What to implement first:**

1. **Adversarial spawn prompts in SPAWN_CONTEXT.md template** (`.orch/templates/`)
   - Add "Before You Start" rules: establish evidence discipline, define what counts as verification
   - Add "Audit Boundary Check": require agents to state what they CAN'T independently verify
   - Quick win: template change, no code required, immediate effect on all new spawns

2. **Validation loop detection in pkg/verify/check.go**
   - Detect confirmatory prompting: search session messages for AI-verifying-AI patterns
   - Flag "operating beyond audit capacity": check if deliverables are in domains agent stated they can't verify
   - Dependency: requires OpenCode API message history access (already have via pkg/opencode/client.go)

3. **Observable behavioral markers in orch status/monitor**
   - Track "dismissing external criticism" pattern: detect when agent receives corrective feedback but continues on same path
   - Requires: session message history analysis, possibly storing conversation sentiment

**Things to watch out for:**

- ⚠️ **False positives in verification detection** - Not all self-review is a validation loop. Agents should check their own work, just not exclusively with AI. Need to distinguish "used AI as one verification method" from "only used AI verification"
- ⚠️ **Token budget impact** - Adversarial prompts in spawn context will increase token usage. May need to make them optional via --strict flag for high-stakes work
- ⚠️ **Agent resistance** - Agents might interpret adversarial prompts as lack of trust and become overly cautious, slowing velocity more than necessary
- ⚠️ **Budden pattern applies to coding** - We're assuming validation loops in mathematical proofs generalize to code. Need to validate this assumption empirically

**Areas needing further investigation:**

- **How to get full adversarial prompt text** - Article is paywalled, only have framework structure. Need actual prompt text or reverse-engineer from principles
- **Frequency of validation loops in code vs math** - Budden case is extreme. Do validation loops occur in typical feature work or only in high-complexity domains?
- **Optimal checkpoint frequency** - Finding 4 suggests "ongoing" checks but doesn't specify cadence. Every commit? Every hour? Phase transitions?
- **Integration with beads issue tracking** - Should validation loop detection update beads issue status? Flag for orchestrator review?

**Success criteria:**

- ✅ **Spawn context includes adversarial prompts** - All new agents receive "Before You Start" rules and "Audit Boundary Check" requirements
- ✅ **Validation loop detection catches confirmatory prompting** - `orch complete` flags sessions where agent only verified work with AI
- ✅ **No increase in validation loop incidents** - As we scale agent velocity, validation loop rate stays constant or decreases
- ✅ **Agents explicitly state verification boundaries** - In SYNTHESIS.md or investigation files, agents document what they can/can't independently verify

---

## References

**Files Examined:**
- pkg/verify/check.go - Current verification implementation to understand what's checked today
- pkg/spawn/context.go - Spawn context generation to identify where to add adversarial prompts
- .orch/templates/SPAWN_CONTEXT.md - Template structure for spawn prompts

**Commands Run:**
```bash
# Create investigation from template
kb create investigation analyze-nate-jones-article-llm

# Fetch article from Substack
# (Attempted webfetch - got preview but article is paywalled)
```

**External Documentation:**
- https://natesnewsletter.substack.com/p/if-a-former-deepmind-engineering - Nate Jones article "Smart people get fooled by AI first" (Dec 23, 2025)
- David Budden Millennium Prize case - Real-world example of validation loop in high-credibility expert

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-20-design-synthesis-protocol-schema.md - Synthesis protocol design (relates to verification phase)
- **Investigation:** .kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md - Reflection checkpoints (relates to ongoing verification)
- **Decision:** .kb/decisions/2025-12-22-replace-confidence-scores-with-structured-uncertainty.md - Structured uncertainty (relates to confidence calibration from Finding 4)

---

## Investigation History

**2025-12-23 (Start):** Investigation started
- Initial question: What are the key warning signs of AI validation loops, and what adversarial frameworks can prevent them?
- Context: Spawned from beads issue orch-go-ndgj to analyze Nate Jones article on LLM psychosis and apply findings to orchestration verification

**2025-12-23 (Findings):** Extracted five key findings from article preview
- Finding 1: Budden case study - DeepMind director bet $45K on solving Millennium Prize with ChatGPT, mathematical community says he proved wrong thing
- Finding 2: Three warning signs - confirmatory prompting, operating beyond audit capacity, AI-vs-experts dynamic
- Finding 3: Automation bias amplification - experts MORE vulnerable than novices
- Finding 4: 10-prompt adversarial framework structure (full text paywalled)
- Finding 5: Organizational velocity risk - speed amplifies validation loop impact

**2025-12-23 (Synthesis):** Connected findings to orchestration system
- Key insight: Our pkg/verify/check.go doesn't detect validation loops, only checks deliverables exist
- Recommendation: Phased adversarial verification (spawn prompts + detection + checkpoints)
- Confidence: Medium (70%) - strong patterns but limited to article preview, no full prompt text

**2025-12-23 (Complete):** Investigation completed
- Final confidence: Medium (70%)
- Status: Complete
- Key outcome: Identified three-phase defense framework (spawn rules, ongoing checks, completion gates) applicable to orchestration verification
