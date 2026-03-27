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

**Operationalizing the test:** Maintain a ledger of non-obvious predictions and their outcomes. Each entry: (prediction, whether it was confirmed/refuted, whether the correction was itself non-obvious). The signal isn't the confirmation rate — it's whether *refutations point somewhere useful*. A training echo fails randomly; a model with real structure fails in ways that expose the next question. The constraint-tension finding (predicted count-based dilution, found tension-based degradation) is the first entry. At N=10+ entries, the pattern of failures tells you more about the model's epistemic status than any single confirmation could.

## Why full resolution is probably impossible

The confound may be structural, not empirical. Here's why:

**Observational equivalence.** If coordination laws ARE substrate-general, then humans discovered them, wrote about them, and that writing is in training data. An LLM reproducing those laws could be "applying training" or "independently deriving them" — and these are the same output. You cannot distinguish "correct retrieval of a true fact" from "independent discovery of a true fact" by looking at the fact alone.

**This is NOT the Chinese Room, despite the surface resemblance.** Searle asks whether symbol manipulation constitutes *understanding* — a conceptual question about the nature of comprehension. The training confound asks about *origin* — an empirical question about how a pattern was produced. Origin questions are resolvable in principle (run the experiment with a system that provably lacks the training data), they're just intractable in practice because you can't verify what's absent from an LLM's training corpus. The Chinese Room says the distinction is *conceptually* impossible to observe; the training confound says it's *empirically* impossible given current tools. That gap matters: empirical obstacles can dissolve with better instruments, conceptual obstacles can't.

**The better analogy is convergent evolution.** Eyes evolved independently 40+ times across unrelated lineages. We don't say "maybe they all copied from a common ancestor" — independent derivation is established through phylogenetic evidence. The training confound is what convergent evolution would look like if you couldn't examine the phylogenetic tree: you see the same structure everywhere but can't tell if it was inherited or independently derived. The non-LLM agent strategy (Strategy 1) is attempting to build the phylogenetic evidence — showing convergence in lineages that provably don't share training ancestry. The reason it's hard is that simple non-LLM agents may not face the same coordination problems, just as single-celled organisms don't independently evolve eyes.

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

## The real question behind the confound

The confound is unresolvable *as a general question* ("is this model discovering or remembering?") but *doesn't need to be resolved for any specific prediction*. Each prediction the model makes can be tested independently. "Is the attractor-gate model a real theory?" is unanswerable in one shot. "Does the attractor-gate model predict what happens when we add quality signals to the comprehension queue?" is testable right now.

This is how all empirical knowledge works. You never prove a theory is "real" — you accumulate predictive hits and informative misses until the theory either earns trust or gets replaced. The training confound is a distraction disguised as a deep question. It asks "what is the nature of this knowledge?" when the actionable question is "does this knowledge make the next experiment more informative than chance?"

The distinction between "law" and "small solution space" is the genuinely interesting question underneath the confound. Laws explain *why* the solution space is small. If you can articulate why only a few coordination patterns survive (bounded rationality + compositional requirements + amnesiac agents → attractors; locally-correct contributions + non-trivial composition → gates), you have an explanation that generates predictions beyond the observed cases. That's the threshold: not "did the LLM discover this," but "does the explanation generate predictions that hold outside the contexts where it was articulated?"

The constraint-tension finding is the first test of this. The model predicted count-based dilution (extrapolating from observations). Reality showed tension-based degradation (a deeper mechanism). The correction itself — that constraint interference depends on semantic tension, not cardinality — is a genuinely non-obvious structural claim. If it holds in future experiments, the model is earning trust the way theories do: by being wrong in ways that teach you something.
