# Formal Grammar Theory and LLM Constraint Systems

**Status:** Complete
**Date:** 2026-03-01
**Beads:** orch-go-ajy5

## D.E.K.N. Summary

- **Delta:** Formal grammar enforcement and behavioral constraint documents operate at fundamentally different layers of the LLM stack. Token-level constrained decoding (Outlines, LMQL, Guidance) provides 100% compliance for Type 2-3 grammars by masking logits before sampling. Behavioral constraints (skill documents, system prompts, CLAUDE.md) provide 0% formal guarantee but can express arbitrarily complex semantic requirements. The optimal strategy is layered: hard enforcement for structure, soft constraints for behavior, post-hoc validation for everything else.
- **Evidence:** Strobl et al. (TACL 2024) proves fixed-precision transformers recognize only star-free languages (a subclass of regular). Merrill & Sabharwal (2023) show CoT extends this to context-sensitive. Prompt injection literature shows 80-90% bypass rates on behavioral constraints. Constrained decoding tools (XGrammar, llguidance) achieve 97%+ structural compliance.
- **Knowledge:** A skill document is not a grammar — it's a probability-shaping document. The Chomsky hierarchy classifies what can be generated/recognized; it does not classify what can be *influenced*. The two mechanisms are complementary, not competing.
- **Next:** This analysis suggests orch-go skill documents could benefit from a "constraint type" annotation distinguishing hard-enforceable structural requirements (JSON output format) from soft behavioral guidance (reasoning approach, escalation rules).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

What is the relationship between formal grammar theory (Chomsky hierarchy) and the constraint mechanisms used with LLMs? Specifically: how does token-level constrained decoding (Outlines, LMQL, Guidance) compare to behavioral-level constraints (skill documents, system prompts)?

---

## Finding 1: The Chomsky Hierarchy and Transformer Expressivity

### The Hierarchy

| Type | Name | Automaton | Example |
|------|------|-----------|---------|
| Type 3 | Regular | Finite automaton | `a*b+` (regex) |
| Type 2 | Context-Free | Pushdown automaton | Balanced parentheses, programming syntax |
| Type 1 | Context-Sensitive | Linear bounded automaton | `a^n b^n c^n` |
| Type 0 | Recursively Enumerable | Turing machine | Halting problem |

### What Transformers Can Actually Recognize

The landmark survey by Strobl et al. (TACL 2024, "What Formal Languages Can Transformers Express?") establishes:

- **Fixed-precision transformers** (real hardware) recognize only **star-free languages** — a strict *subclass* of regular languages (Type 3). Star-free languages use union, concatenation, and complement but not Kleene star. This means real transformers can't even handle all regular languages.
- **Hard-attention transformers** (0/1 attention weights) recognize only languages in **AC0** (constant-depth boolean circuits). Excludes parity checking — a regular language.
- **Log-precision transformers** (theoretical idealization) handle all regular languages and some context-free, but not all CFLs.

**Critical insight: The Chomsky hierarchy doesn't map neatly to transformer difficulty.** Transformers can learn some context-free languages (`a^n b^n`) yet fail on some regular languages (parity). Circuit complexity classes are a better fit than the Chomsky hierarchy for characterizing transformer capabilities.

### Chain of Thought Changes the Picture

Merrill & Sabharwal (2023, "The Expressive Power of Transformers with Chain of Thought"):

- **Linear CoT steps** → reaches context-sensitive languages (Type 1)
- **Polynomial CoT steps** → reaches all of P (polynomial-time decidable)

A transformer *with CoT* can solve problems across the Chomsky hierarchy up to Type 1 and beyond. Without CoT, a single forward pass is fundamentally limited to sub-regular computation.

**Sources:**
- Strobl et al., "What Formal Languages Can Transformers Express?" TACL 2024 — https://direct.mit.edu/tacl/article/doi/10.1162/tacl_a_00663/120983
- Merrill & Sabharwal, "The Expressive Power of Transformers with Chain of Thought" 2023 — https://arxiv.org/abs/2310.07923

---

## Finding 2: Constrained Decoding — Hard Grammar Enforcement at Token Level

Constrained decoding tools solve grammar compliance by operating at the **logit layer**, before sampling. The mechanism:

1. Grammar (JSON Schema, regex, CFG) compiled into FSM or PDA
2. At each token generation step, current automaton state determines valid continuations
3. Invalid tokens get logits set to -infinity (logit masking)
4. Model samples only from valid tokens
5. Automaton state updates; repeat

### Major Tools and Their Grammar Classes

| Tool | Grammar Class | Mechanism | Notable |
|------|--------------|-----------|---------|
| **Outlines** (Willard & Louf, 2023) | Regular (FSM) | Precomputed vocabulary→FSM transition index | Pioneered approach; complex schemas can take minutes to compile |
| **LMQL** (ETH Zurich) | Regular + custom | Character-level constraints → token masks | Programming language for LLM interaction; eager evaluation |
| **Microsoft Guidance / llguidance** | Context-Free (CFG) | Near-zero overhead (~1.5ms for 128k tokenizer) | OpenAI credited llguidance for Structured Outputs foundation |
| **XGrammar** (Dong et al., MLSys 2025) | Context-Free (PDA) | 97.1% schema accuracy vs Outlines' 76.4% | Default in vLLM as of 2025 |

### Industry Adoption (2024-2025)

- OpenAI Structured Outputs (Aug 2024) — built on llguidance
- Google Gemini `response_schema` (May 2024)
- Anthropic constrained decoding for Claude (Nov 2025)

### Theoretical Scope

Most tools enforce **Type 2-3** (regular and context-free). No mainstream tool enforces context-sensitive (Type 1) grammars, though recent research on "logically constrained decoding" explores formal logical constraints (chess legality, propositional resolution).

**Sources:**
- Brenndoerfer, "Constrained Decoding: Grammar-Guided Generation" — https://mbrenndoerfer.com/writing/constrained-decoding-structured-llm-output
- LMQL docs — https://lmql.ai/docs/language/constraints.html
- llguidance — https://github.com/guidance-ai/llguidance
- LMSYS, "Compressed FSM for Fast JSON Decoding" — https://lmsys.org/blog/2024-02-05-compressed-fsm/

---

## Finding 3: Natural Language Documents as "Soft" Constraint Systems

### The Core Problem

LLMs are unconstrained generative models. They lack intrinsic mechanisms to guarantee constraint satisfaction (IJCAI 2025). When constraints are specified in natural language (system prompts, skill documents), they function as **soft biases on the probability distribution**, not as hard enforcement.

### Evidence for Weakness

Prompt injection and jailbreak literature quantifies the gap:

- **OWASP LLM01:2025** ranks prompt injection as #1 LLM vulnerability
- Roleplay-based attacks: **89.6% bypass rate** on system prompt constraints
- Logic trap attacks: **81.4% bypass rate**
- Study bypassing 12 recent defenses with >90% success — defenses had originally reported near-zero attack success

### What a Skill Document Actually Does

When a system prompt or skill document says "always do X" or "never do Y":

1. The text becomes part of the context that shapes attention patterns
2. The instruction influences (but does not determine) every subsequent token probability
3. There is no enforcement mechanism — the model may violate the constraint at any token
4. Compliance probability decreases with output length and constraint complexity
5. The model is performing "fuzzy pattern matching against training data that included similar instructions" — not executing the constraint as a rule

**Analogy:** Token-level enforcement is physical guardrails (you *cannot* drive off the road). Behavioral constraints are speed limit signs (they influence behavior but don't physically prevent speeding).

### The CHI 2024 Framework

"We Need Structured Output: Towards User-centered Constraints on Large Language Model Output" (CHI 2024) took the first systematic look at this gap between what users *want* to constrain and what mechanisms can guarantee. No formal framework for "natural language as constraint system" exists yet.

**Sources:**
- OWASP LLM01:2025 — https://genai.owasp.org/llmrisk/llm01-prompt-injection/
- "We Need Structured Output" CHI 2024 — https://lxieyang.github.io/assets/files/pubs/llm-constraints-2024/llm-constraints-2024.pdf
- Willison, "The Attacker Moves Second" — https://simonwillison.net/2025/Nov/2/new-prompt-injection-papers/

---

## Finding 4: The Comparison — Token-Level vs. Behavioral-Level

### Guarantee Matrix

| Property | Token-Level (Outlines/Guidance) | Behavioral (Skill Docs/CLAUDE.md) |
|----------|-------------------------------|----------------------------------|
| Compliance guarantee | 100% for Type 2-3 grammars | 0% formal guarantee |
| Failure mode | Mathematically impossible to emit invalid token | Can fail at any token; probability increases with length |
| Adversarial robustness | Immune to prompt injection for structure | 80-90% bypass rates demonstrated |
| Constraint scope | Syntactic structure (format, schema, type) | Semantic behavior (tone, reasoning, decisions) |
| Expressivity | Regular or context-free languages | Arbitrary natural language; unbounded expressivity |

### What Each Can and Cannot Do

**Token-level can:** Guarantee JSON/XML/SQL syntax, enforce regex, ensure schema conformance, force enumerated selections, guarantee stop conditions.

**Token-level cannot:** Ensure factual accuracy, enforce reasoning quality, maintain persona, control semantic content, handle context-sensitive constraints.

**Behavioral can:** Influence reasoning approach, shape tone/persona, provide domain knowledge, set priorities, express arbitrarily complex requirements.

**Behavioral cannot:** Guarantee any structural property, prevent all violations, resist adversarial inputs, provide formal verification.

### The Quality Tradeoff

Constrained decoding can **degrade output quality** when constraints are tight. When the model's top-probability tokens are all invalid under the grammar, it samples from lower-probability alternatives — syntactically valid but potentially semantically awkward. Behavioral constraints don't have this problem because they allow full expressivity.

### Optimal Strategy: Layered Constraints

The literature converges on a three-layer approach:

1. **Hard token-level enforcement** for structural requirements (output format, schema)
2. **Soft behavioral constraints** for semantic requirements (reasoning quality, domain knowledge, escalation rules)
3. **Post-hoc validation** for properties neither layer guarantees (factual accuracy, logical consistency)

---

## Finding 5: Implications for Skill Documents and CLAUDE.md

### A Skill Document Is Not a Grammar

From a formal language theory perspective, a skill document (like `SKILL.md` or `CLAUDE.md`) cannot be classified within the Chomsky hierarchy because it doesn't define a formal language. It defines a **desired behavioral distribution** — a fuzzy target that the model's output should approximate.

The Chomsky hierarchy classifies what can be **generated or recognized**. It doesn't classify what can be **influenced**. Skill documents operate in the influence domain.

### What Skill Documents Actually Enforce

In orch-go's architecture, skill documents contain two types of constraints:

1. **Structurally enforceable** (could be hard-constrained):
   - "Report via `bd comment`" — could be enforced by post-hoc validation
   - "Create file at `.kb/investigations/{date}-inv-{slug}.md`" — could be validated structurally
   - Phase reporting format — could be regex-validated

2. **Behaviorally influenceable only** (cannot be hard-constrained):
   - "Test before concluding" — requires semantic judgment
   - "Escalate architectural decisions" — requires understanding context
   - "Evidence hierarchy: artifacts are claims, not evidence" — requires reasoning about epistemology
   - Constitutional hard limits — require ethical reasoning

### The Drift Connection

This maps directly to the Drift Taxonomy model's insight: **drift is duplicated state that silently diverges.** A skill document is a declaration of desired behavior, and the agent's actual behavior is the "duplicated state" that can silently diverge. The model's insight — "detection at consumption beats detection at production" — applies here. You can't prevent behavioral drift at the instruction layer (production); you detect it at the output layer (consumption) via verification gates.

This is exactly what `orch complete` does: post-hoc validation of behavioral compliance. The skill document is the "source of truth" for desired behavior, and verification checks whether the agent's actual output diverged.

---

## Conclusion

The relationship between formal grammar theory and LLM constraint mechanisms reveals a fundamental stratification:

1. **Transformers are weaker than commonly assumed.** Fixed-precision transformers can't even recognize all regular languages. The Chomsky hierarchy is the wrong framework for characterizing transformer capabilities — circuit complexity classes are more accurate.

2. **Constrained decoding solves a well-defined problem.** By restricting the action space at the token level, tools like Outlines/Guidance/XGrammar provide 100% compliance for Type 2-3 grammars. This is a solved problem for structural output.

3. **Behavioral constraints are fundamentally different.** Skill documents, system prompts, and CLAUDE.md files operate as probability-shaping inputs, not grammar enforcers. They can express arbitrarily complex requirements but guarantee nothing. The 80-90% prompt injection bypass rates quantify this gap.

4. **The two mechanisms are complementary.** Token-level enforcement handles "what shape should the output be?" Behavioral constraints handle "what should the output mean?" Neither alone is sufficient; the optimal architecture layers both with post-hoc validation.

5. **For orch-go specifically:** The skill system's constraints are behavioral (soft), but verification gates (`orch complete`, phase checking, deliverable validation) add a post-hoc hard layer. This three-layer pattern (behavioral instruction → agent execution → structural verification) mirrors the optimal strategy identified in the literature.
