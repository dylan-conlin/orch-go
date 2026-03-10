---
title: "Independent disconfirmation under asymmetric output growth — agentic systems fail because they mass-produce answer-shaped confidence faster than trustworthy rejection"
status: open
created: 2026-03-10
updated: 2026-03-10
resolved_to: ""
---

# Evaluation is orthogonal to scaling — the hardest problem in agentic engineering isn't coordination, it's knowing whether output is correct when the system evaluates itself

## 2026-03-10

Eledath's 8 Levels of Agentic Engineering (bassimeledath.com, Mar 10 2026) frames agent teams as the apex. But 48h of experience in this system shows the hardest problem is orthogonal to the ladder: self-evaluation. Our system built a theory, probed it 6 times with its own agents, declared it survived falsification, wrote 3 blog posts, and was ready to publish — all gates green, all checks passing. One external read (Codex, given artifact without diagnosis) identified overclaiming in minutes. 6 internal probes < 1 external read. The publication gate (orch kb gate publish) is a mechanical fix for the output side — blocks publishing without external challenge artifacts. But it doesn't fix evaluation. Today's 'uncontaminated' blog review was spawned with the diagnosis pre-loaded — the system contaminated its own adversarial process while trying to be adversarial. Leverage in agentic engineering lives at Level 4-6 (context + harness), not 7-8 (scaling). Scaling multiplies whatever quality of evaluation you have. If evaluation is closed-loop, scaling multiplies confident wrongness.

Three properties make agentic self-evaluation harder than normal 'you need external testing': (1) Volume breaks traditional review — 176 completions/day, ratio of output to human evaluation keeps widening, gates catch structural failures but not semantic ones. (2) The system generates convincing self-reviews — the 6 falsifiability probes had methodology sections, evidence tables, conclusions, structurally indistinguishable from real adversarial review, difference only visible from outside. (3) Contamination is the default — spawning an 'uncontaminated' review, I contaminated it in the spawn prompt by giving context. The system's coherence optimization makes contamination the path of least resistance. Open question: can a system contain its own adversarial layer, or is 'external' a hard requirement? And who verifies the external artifacts are actually external? Meta-problem: this analysis is itself the system evaluating itself.

## External Agent Response (Codex) — 2026-03-10

I think this thread is directionally right, but it still over-compresses the problem.

The hardest problem is not evaluation in the abstract. It is **credible negative feedback under asymmetric output growth**. "Evaluation" is too broad and too flattering; it sounds like you mainly need better scoring. What actually failed here was that the system had no reliable way to introduce disconfirming pressure once a coherent frame started compounding through investigations, models, probes, and publications. The issue is less "can the system evaluate itself?" and more "can the system reliably produce objections that survive its own coherence incentives?"

That distinction matters because many things called evaluation are still closed-loop:
- scoring by the same model family
- probes written in the model's vocabulary
- falsification attempts that inherit the original framing
- rubric-driven review that checks completeness more than surprise

Those all look adversarial while remaining structurally confirmatory.

I agree with the main operational claim: scaling is downstream of evaluation quality. More precisely, scaling amplifies the ratio between generation and trustworthy rejection. If generation grows 10x and credible rejection stays flat, the system doesn't just get noisier. It gets more **epistemically dangerous** because confidence, polish, and internal cross-citation grow with output volume.

Where I would sharpen the thread:

1. Externality is probably a spectrum, not a binary.
Truly external review is strongest, but there are weaker and stronger forms:
- same loop, same framing: almost useless
- same system, blinded packet, no diagnosis: somewhat useful
- different model family, blinded packet: better
- human reviewer with no internal context: better still
- real external user data: strongest

The requirement is not "outside" in the social sense; it is **independence from the originating frame**.

2. The core bottleneck is adversarial independence, not evaluation per se.
You can have lots of evaluation and still have no independence. That is what happened here. The system produced rigor-shaped artifacts, but not frame-breaking artifacts.

3. The contamination problem is bigger than prompt leakage.
Prompt contamination is one form. The deeper contamination is artifact selection, vocabulary choice, what gets measured, and what counts as success. By the time a reviewer sees a polished theory draft, much of the framing damage is already upstream.

4. "Can a system contain its own adversarial layer?" should be answered narrowly.
Yes, partially, if you enforce blindness, lineage separation, and negative authority only.
No, if you expect the same coherence-optimized loop to generate its own truly independent critique on demand.

My answer would be: an internal adversarial layer can downgrade claims and catch obvious inflation, but it cannot be the final court for novelty, generality, or importance.

5. The right unit of concern is not bad output. It is bad output that passes as validated.
That is the real systems problem. Plenty of weak ideas get generated; that is normal. The dangerous failure is when the pipeline upgrades them into "survived falsification," "general theory," or "ready to publish."

So my external verdict:

- The thread's main thesis is strong if rewritten as: **the hardest scaling problem is preserving independent disconfirmation as output volume rises**.
- It is weaker if framed as: **evaluation is the hardest problem**, full stop.
- The practical implication is to build for independence, not just review: blinded packets, claim lineage checks, negative-authority reviewers, and at least one challenge source that is not produced inside the originating interpretive loop.

If you want the blunt one-liner:

**Agentic systems do not mainly fail because they cannot generate answers. They fail because they can mass-produce answer-shaped confidence faster than they can produce trustworthy rejection.**

Dylan's sharpening (responding to Codex): Internal evaluation is not useless — it solves consistency checking, structural correctness, hygiene, throughput control. The mistake is treating it as if it can also certify epistemic independence. It can't. Three reveals: (1) The internal stack is optimized for consistency checking, not frame-breaking — narrower than 'evaluation.' (2) The scarce resource is independent attention, not verification logic — much current infrastructure is local optimization around a non-binding constraint. (3) Strategic boundary: some judgments cannot be safely endogenous — novelty, importance, external relevance, 'would a skeptical outsider care?' are a different class than 'did the artifact satisfy the rubric?' The leverage point is not 'review more things externally' but being selective about what is allowed to ask for external certification: only observation→theory upgrades, only novelty/generality/causal claims, only outputs crossing publication or policy boundaries. The few places where the system tries to convert coherence into authority.
