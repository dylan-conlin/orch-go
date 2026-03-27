---
title: "Whether the training confound can ever be resolved — LLM applying training knowledge vs discovering substrate-general laws"
status: forming
created: 2026-03-27
updated: 2026-03-27
resolved_to: ""
parent_threads:
  - 2026-03-26-epistemic-status-ai-behavioral-experiments.md
  - 2026-03-26-self-disconfirming-knowledge-system-that.md
---

# Whether the training confound can ever be resolved

## The confound

When HyperAgents' meta-agent "discovers" PerformanceTracker, MemoryTool, and evaluation analysis, is it:

**(A)** GPT-4 applying known software engineering patterns from its training data — "I've read millions of codebases that have logging and caching, so I'll add logging and caching"

**(B)** An agent under coordination pressure converging on the only viable solutions to bounded-rationality problems — the same way ant colonies converge on pheromone trails without having read optimization textbooks

The HyperAgents cross-domain transfer result (meta-level patterns transfer, task-level don't) is *consistent with* (B) but *doesn't exclude* (A). An LLM's general-purpose "how to improve programs" knowledge would also transfer across domains, because it's domain-general knowledge from training, not domain-general knowledge from the coordination problem.

## Three strategies for partial resolution

### 1. Non-LLM agents

Run the same experiment with agents that have no training on human coordination: pure RL agents, evolutionary algorithms with no language model, cellular automata. If they converge on the same patterns (persistent state, performance tracking, template reuse), training is eliminated.

**Problem:** These agents may be too simple to face real coordination problems. The confound exists precisely because LLMs are sophisticated enough to face coordination challenges AND sophisticated enough to have absorbed human solutions to those challenges. Simpler agents might not face the same problems at all.

**Partial evidence already exists:** Ant colonies DO converge on persistent state (pheromone trails), performance tracking (trail reinforcement proportional to food quality), and template reuse (stereotyped behavioral sequences). Immune systems converge on memory (B cells), performance tracking (affinity maturation), and template reuse (recombination of known gene segments). These are substrate-general coordination patterns in non-trained systems. But the mapping is loose enough to be pattern-matching rather than evidence.

### 2. Genuinely novel coordination problems

Design problems where training data contains no solution. If LLMs still converge on attractor-like patterns in domains their training data genuinely doesn't cover, that's evidence for (B).

**Problem:** You can never verify that a domain is "novel" to the training data. LLMs are trained on everything public. Even if you design a custom coordination game, the LLM may have seen structurally isomorphic problems. The training corpus is a black box.

### 3. Predictive power

If the attractor-gate model makes *non-obvious predictions* that are subsequently confirmed, that's evidence the model captures real structure — regardless of whether the LLM "discovered" or "remembered" it.

**This is the most tractable strategy.** The constraint scaling experiment (2026-03-26) is an example: the model predicted "constraints dilute at 10+" but the experiment found the actual variable is tension, not count. The model was *wrong in a specific way*, and the correction (tension-based degradation) was non-obvious. A pure training echo wouldn't generate falsifiable predictions that fail in informative ways.

**The standard:** A model that is "just training echo" would reproduce training-data patterns but fail to predict novel findings. A model that captures real structure would sometimes be wrong in ways that point toward the right answer. Track the ratio.

## Why full resolution is probably impossible

The confound may be structural, not empirical. Here's why:

**Observational equivalence.** If coordination laws ARE substrate-general, then humans discovered them, wrote about them, and that writing is in training data. An LLM reproducing those laws could be "applying training" or "independently deriving them" — and these are the same output. You cannot distinguish "correct retrieval of a true fact" from "independent discovery of a true fact" by looking at the fact alone.

**This is the Chinese Room in a new costume.** Searle asks whether symbol manipulation constitutes understanding. The training confound asks whether pattern reproduction constitutes discovery. Both questions resist empirical resolution because the proposed distinction (understanding vs. simulation, discovery vs. retrieval) isn't observable from outside the system.

**Even human scientists face this.** A physicist who learned F=ma from a textbook is "applying training knowledge." But F=ma is also a real law. Their knowledge of it is simultaneously a training echo AND correct understanding of reality. We don't usually call this a "confound" for humans because we grant them the benefit of the doubt on understanding. The confound only feels like a confound because we're withholding that benefit from LLMs.

## Why it might not matter (the pragmatic frame)

The confound matters for epistemics — for knowing whether you've discovered something real. But it doesn't necessarily matter for engineering.

**What matters for orch-go is predictive power, not origin story.** If attractor+gate pairing reliably produces better outcomes than gates alone (empirically confirmed across 3 domains), the engineering recommendation is the same regardless of whether that pattern was "discovered" or "retrieved." The model is useful if it predicts; it doesn't need to be novel.

**The risk of caring too much about the confound:** It becomes a reason to discount observations that are actually useful. "These patterns might just be training echoes" is true but unhelpful if the patterns work. It's the same trap as "correlation isn't causation" — technically correct, but paralyzing if you refuse to act on strong correlations while waiting for causal proof.

**The risk of caring too little:** You overclaim. You say "we've discovered substrate-general coordination laws" when you've actually documented "LLMs reproduce known coordination patterns when given coordination problems." The knowledge-accretion model already corrected for this (Codex review, Mar 10), but the temptation to overclaim persists because "new discovery" is more exciting than "confirmed known patterns in a new context."

## Where this leaves orch-go

**The honest position:** The attractor-gate observations are real and useful for engineering orch-go. Whether they constitute a novel theoretical contribution is unresolved and may be unresolvable. The model's value is practical (predicts what works), not theoretical (explains why).

**What would change the assessment:**
- Non-obvious predictions that hold → strengthens "real structure" interpretation
- Non-obvious predictions that fail systematically → weakens it, suggests overfitting to training patterns
- Non-LLM systems showing same convergence → strongest possible evidence for substrate-generality (but hardest to obtain)

**The bounded-rationality framing from the epistemic status thread remains the most parsimonious explanation:** Agents (human or AI) facing coordination under bounded rationality converge on similar solutions because the solution space is small, not because there are "laws" governing it. The training confound dissolves under this frame — it doesn't matter whether the LLM "remembers" or "discovers" solutions if the reason they work is that the problem only admits a few viable solutions.

This is the difference between "there's a law that produces these patterns" and "these patterns are the only ones that survive contact with the problem." The second framing doesn't need the confound resolved. The problem itself constrains the solution space. How you arrive at the solution (training, evolution, first-principles derivation) is irrelevant to why it works.
