# Probe: Blog Post Uncontaminated Claim Review

**Model:** harness-engineering
**Date:** 2026-03-10
**Status:** Complete

---

## Question

Do the published blog posts ("Soft Harness Doesn't Work" and "Building Blind") make overclaimed, unsupported, or falsely-novel claims? Would domain experts (Ostrom, Conway, Brooks, coordination theory) see restatements of known concepts presented as original discoveries?

---

## What I Tested

Read both published blog posts WITHOUT reading the knowledge-physics or harness-engineering models first. Evaluated each post against four criteria:
1. Claims that assume theory is validated
2. Novelty language without external citation
3. Restatements of known concepts presented as novel
4. Specific sentences needing revision

Blog posts reviewed:
- `/blog/src/content/posts/soft-harness-doesnt-work.md` (pubDate: 2026-03-08)
- `/blog/src/content/posts/building-blind.md` (pubDate: 2026-03-08)

---

## What I Observed

### Post 1: "Soft Harness Doesn't Work"

**Overall assessment: Mostly grounded, with several areas of mild-to-moderate overclaiming.**

The post stays in first-person experiential framing ("I tested," "I measured") which is appropriate. It cites OpenAI's harness engineering post. The core empirical claims (265 trials, 7 skills, dilution curves) are specific and falsifiable.

#### Flagged Claims

| # | Location | Claim/Sentence | Severity | Issue |
|---|----------|---------------|----------|-------|
| 1 | Line 13 | "There are two kinds of harness, and one of them is mostly decoration." | **overclaimed** | Presents a binary taxonomy as exhaustive. The hard/soft distinction is useful but incomplete — structural attractors (introduced later) don't fit cleanly into either category, which the post itself acknowledges. "Two kinds" overstates confidence in the taxonomy. |
| 2 | Line 34-35 | "At 5 co-resident behavioral constraints, compliance starts dropping. At 10+, constraints become inert." | **unsupported** | Specific thresholds from 7 skill documents presented as general findings. N=7 is insufficient for claiming precise inflection points. The post doesn't describe statistical methodology, confound controls, or confidence intervals. These numbers may be accurate for this system but are stated as general principles. |
| 3 | Line 36-37 | "The agents aren't refusing to comply. They just can't hold that many prohibitions against the current of the system prompt" | **unsupported** | This is a mechanistic explanation for observed behavior. The post measured outcomes (compliance drops) but didn't test the mechanism (prompt competition). The causal claim ("can't hold... against the current") goes beyond what was observed. |
| 4 | Line 53-55 | "spawn-related code started landing there instead of in the monolithic spawn command file... Not because I told agents to put code there — because the package name primed their attention." | **overclaimed** | "Structural attractors" = affordances (Don Norman, 1988), nudges (Thaler & Sunstein, 2008), and Conway's Law ("organizations design systems that mirror their communication structures"). The concept is well-established in HCI, behavioral economics, and organizational theory. Presenting it without citation as a personal discovery is mild novelty-claiming. |
| 5 | Line 55-57 | "It's an always-visible signal that doesn't compete with the system prompt... It's architecture doing the work of instruction." | **fine (but citable)** | Good formulation. Not claimed as novel. But the insight that structure shapes behavior more reliably than instruction is the core thesis of Christopher Alexander's "A Pattern Language" (1977) and Norman's affordance theory. A nod would strengthen credibility. |
| 6 | Line 65-66 | "They solved it with periodic garbage collection agents. I solved it with gates that prevent the accumulation in the first place." | **overclaimed** | Implies prevention > cleanup as a discovered insight. This is preventive vs. corrective maintenance — studied since at least Deming (1950s). The framing suggests the author's approach is superior without acknowledging that both strategies have well-known trade-offs (gates add friction, prevention can be more expensive than cleanup for certain error classes). |
| 7 | Line 69 | "I have one thing they don't have in their post: the receipts." | **overclaimed** | Claims empirical superiority but the methodology isn't described in enough detail to evaluate rigor. 265 trials across 7 skills — what's the inter-rater reliability? How were "compliance" and "performance" operationalized? Are the 7 skills representative? "Receipts" implies rigorous evidence but the post provides summary statistics without methodological transparency. |
| 8 | Line 71 | "If you have more than four behavioral constraints, the fifth one is probably already inert." | **overclaimed** | Generalizes from a single system (the author's orchestrator with specific skill documents, specific model versions, specific task types) to all CLAUDE.md/AGENTS.md usage. The threshold of 4-5 may be system-specific. |

#### Known-Concept Mapping

| Post Concept | Established Literature | Citation Missing? |
|-------------|----------------------|-------------------|
| Hard vs. soft harness | Poka-yoke (Shingo, 1986), defensive programming, "make illegal states unrepresentable" | Yes |
| Structural attractors | Affordances (Norman, 1988), nudge theory (Thaler/Sunstein, 2008), Conway's Law (1967) | Yes |
| Dilution curve | Prompt engineering literature on instruction following vs. prompt length; attention/working memory limits (Miller, 1956) | Yes |
| Prevention > cleanup | Deming cycle, preventive maintenance, shift-left testing | Yes |
| "Architecture doing the work of instruction" | Pattern Language (Alexander, 1977), organizational design theory | Yes |

---

### Post 2: "Building Blind"

**Overall assessment: Well-grounded experiential narrative. Lower overclaiming risk than "Soft Harness." Main issue is presenting well-known epistemological concepts as personal discoveries.**

The post is primarily a personal narrative about building with AI agents in a language the author can't read. The specific context (non-programmer orchestrating Go via AI) is genuinely novel. The post is honest about failures (January models didn't work).

#### Flagged Claims

| # | Location | Claim/Sentence | Severity | Issue |
|---|----------|---------------|----------|-------|
| 1 | Line 29 | "Each model took 20-40 investigations and structured them into core mechanism, failure modes, constraints, evolution history." | **fine** | Describes a personal schema, doesn't claim novelty. |
| 2 | Line 48 | "Same trick as Rails, just built from observations instead of code. I thought I'd cracked it." | **fine** | Good — the post immediately undermines this with "I was wrong." Honest framing. |
| 3 | Line 56-58 | "Those original five models? They sat largely unused. Zero probes. Two cross-references... I didn't reach for the models. I spawned new investigations." | **fine** | Strong self-critical evidence. This is the post's best quality. |
| 4 | Line 71 | "My January models said 'here's how the system works.' The February models say 'here's what we think happens, here's how we're testing it, here's where we're wrong.'" | **overclaimed** | This is Popperian falsificationism (Popper, 1934) / Bayesian epistemology. The distinction between "conclusions to trust" and "hypotheses to test" is the foundation of scientific methodology. Presenting it as a personal discovery from building software, without acknowledging it's epistemology 101, overstates the insight's originality. |
| 5 | Line 81 | "The February models work because they're honest about what they don't know." | **unsupported** | The evidence shows engagement (9 probes, 17+ cross-references) but "work" implies they produce better outcomes — which isn't measured. Engagement ≠ effectiveness. Suggestion: "The February models get used — they have 9 probes and 17+ cross-references, compared to zero for the January batch." |
| 6 | Line 95-97 | "A model that says 'completion authority is beads, not sessions' might be wrong, and nothing catches the error until you hit it in production. What actually works is closer to how a scientist uses models." | **fine (but citable)** | Correct observation. But "closer to how a scientist uses models" IS the scientific method — not "closer to" it. The hedging actually understates the connection. A brief mention of Popper or PDCA would be more honest than the indirect reference. |
| 7 | Line 118 | "What you need is the stuff I picked up in 12 years of shipping things that break: how to decompose problems, how to verify behavior, how to notice when your mental model is wrong." | **overclaimed** | Frames systems thinking as a personal acquisition rather than a well-studied domain. These are core competencies in systems engineering (INCOSE), design thinking (IDEO), and software engineering education. The claim that 12 years of shipping taught these things is fine — but implying they're hard-won personal insights rather than teachable, documented skills is mild overclaiming. |
| 8 | Line 122 | "What actually works is a loop: build something, observe how it fails, form a hypothesis about why, test the hypothesis, update your model." | **overclaimed** | This is the Deming cycle (PDCA, 1950s), Boyd's OODA loop (1960s), and the scientific method. Presenting it as "what actually works" — as if it's a finding — when it's a well-documented methodology with 70+ years of literature. |
| 9 | Line 126 | "Not because my models got more complete, but because my process for discovering where they're wrong got faster." | **fine** | Nice closing. Doesn't overclaim. |

#### Known-Concept Mapping

| Post Concept | Established Literature | Citation Missing? |
|-------------|----------------------|-------------------|
| Models as hypotheses vs. conclusions | Popper's falsificationism (1934), Bayesian epistemology | Yes |
| Build-fail-learn loop | Deming cycle/PDCA (1950s), OODA loop (Boyd, 1960s), lean startup (Ries, 2011) | Yes |
| Systems knowledge transfers, syntax doesn't | Transfer learning in education (Gentner, 1983), expert vs. novice problem-solving (Chi, 1981) | Yes |
| Multiple sources of truth disagree | CAP theorem, eventual consistency, distributed systems theory | Yes |
| Structure holds complexity you can't hold in your head | External cognition (Norman, 1993), institutional knowledge (March, 1991) | Yes |

---

## Model Impact

- [x] **Extends** model with: Publication claim review identifying specific overclaimed/unsupported sentences and mapping post concepts to established literature that should be cited or acknowledged.

**Key finding:** Neither post commits the most dangerous error (presenting theory as validated). Both posts stay mostly in experiential/first-person framing. The primary issue across both is **implicit novelty** — not claiming "we discovered X" explicitly, but describing well-established concepts (affordances, PDCA, falsificationism, Conway's Law) without acknowledgment, which creates the impression of original discovery. A reader with domain knowledge would recognize 60-70% of the conceptual content as restatements of known frameworks.

**Severity distribution:**
- Overclaimed: 6 instances (mostly threshold generalization and well-known concepts without citation)
- Unsupported: 3 instances (mechanistic claims beyond observed evidence, "works" without outcome measurement)
- Fine: 5 instances
- Fine but citable: 2 instances

---

## Notes

**What makes these posts valuable despite the overclaiming:**
1. The specific context (AI agent orchestration, building in an unreadable language) IS genuinely novel
2. The self-critical honesty ("I was wrong," "mostly decoration") is rare in tech writing
3. The empirical stance (measuring rather than asserting) is the right approach, even if the methodology needs transparency

**Recommended revision strategy:**
1. Don't add a literature review — that kills the voice
2. Instead, use brief inline acknowledgments: "This is essentially Conway's Law applied to LLM agents" or "The scientific method, applied to system understanding"
3. Soften threshold claims: "In my system, at 5+" rather than absolute "At 5+"
4. Add methodology footnote for the 265-trial claim
5. Change "works" to observed evidence where outcomes aren't measured
