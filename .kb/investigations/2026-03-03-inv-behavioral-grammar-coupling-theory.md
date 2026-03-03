# Behavioral Grammar Coupling: How Human and Agent Probability Distributions Co-Evolve

**Status:** Complete
**Date:** 2026-03-03
**Type:** Synthesis investigation

**TLDR:** The orchestrator skill document functions as a behavioral grammar — a probability-shaping document that sculpts the LLM's activation landscape. The human's interaction patterns are also a behavioral grammar that the agent models implicitly. These two grammars form a coupled dynamical system with a strong attractor toward convergence. Convergence feels like smooth operation but can correspond to mutual predictability rather than correctness. Maintaining verification effectiveness requires deliberate divergence — strategic illegibility — to prevent the human grammar from becoming a formality the system has already priced in.

## D.E.K.N. Summary

- **Delta:** Connected the formal grammar investigation's finding ("a skill document is a probability-shaping document, not a grammar") to the human side of the interaction. Both grammars interact through a narrow text channel with information loss on both sides. Identified four feedback channels and a convergence attractor that explains how verification degrades even when the system appears to work smoothly.
- **Evidence:** Synthesized from 8 existing artifact clusters: formal grammar theory investigation, verification bottleneck decision, human-AI interaction frames model, pressure over compensation decision, defense in depth investigation, emphasis language probe, behavioral compliance probe, and legibility literature review probe. Connects established findings rather than generating new experimental evidence.
- **Knowledge:** Convergence ≠ correctness. The dyad can settle into an equilibrium of mutual predictability that feels like success but isn't externally validated. Verification degrades through a depth mechanism (pattern-matching replaces evaluation) independent of the rate mechanism (changes outpace verification capacity). Strategic variation in verification patterns is necessary to prevent convergence from neutralizing the human as a meaningful checkpoint.
- **Next:** Extend the human-AI interaction frames model with the four-channel framework. Consider whether the verification bottleneck principle needs a "convergence-degrades-depth" corollary. Evaluate whether the system should explicitly model Dylan's verification patterns to make them legible (SA-2/SA-3 approach vs SA-1 satisfaction approach).

## Prior Work

| Investigation | Relationship | How Connected |
|---|---|---|
| `.kb/global/models/behavioral-grammars/model.md` | Foundation | Parent model for behavioral grammar theory with 6 tested claims — this investigation extends the model with human-agent coupling dynamics |
| `2026-03-01-inv-formal-grammar-theory-llm-constraint-systems.md` | Foundation | Established "skill document as probability-shaping document" — this investigation extends the same framing to the human side |
| `2026-03-01-dsl-design-principles-natural-language-embedded.md` | Foundation | Established skills as "specifications for cognitive processors" — this adds that the human is also a cognitive processor being specified by interaction history |
| `2026-03-01-inv-defense-depth-applied-software-behavior.md` | Application | Cosmetic redundancy (same mechanism repeated) ≠ real defense — maps to uniform verification patterns being cosmetic redundancy against convergence |
| Verification Bottleneck Decision (2026-01-14) | Extension | Original addresses rate; this adds depth dimension |
| Human-AI Interaction Frames Model | Extension | Original describes frame spectrum; this adds four-channel coupling dynamics and convergence mechanics |
| Pressure Over Compensation Decision (2025-12-25) | Generalization | Original addresses knowledge compensation; this identifies behavioral convergence as a second closed-loop pattern |
| Emphasis Language Probe (2026-03-02) | Supporting evidence | Emphasis markers as attention allocation signals — reciprocally, the human's attention patterns are also being modeled |
| Behavioral Compliance Probe (2026-02-24) | Supporting evidence | Identity vs action compliance gap explained by compatible vs competing grammar productions |
| Legibility Literature Review (2026-03-01) | Framework | Bainbridge Irony 3 (out-of-the-loop) is the convergence attractor; Hollnagel's "joint cognitive system" is the dyad concept |

---

## The Two Grammars

### Agent Grammar

The probability distribution over outputs conditioned on the skill document, conversation history, and current input.

- **Vocabulary:** Completions (token sequences)
- **Syntax:** Structural constraints from skill documents — gates, required reasoning, output formats
- **Semantics:** The world model in foundation weights, filtered through behavioral constraints

The reinforcement density in a skill document doesn't teach the model new knowledge. It sculpts the activation landscape so certain completion paths are more probable. The grammar is a lens, not a brain.

**Established in:** formal grammar theory investigation (2026-03-01)

### Human Grammar

The probability distribution over responses conditioned on the output being reviewed, current cognitive state, accumulated mental model, and goals.

- **Vocabulary:** Approvals, corrections, directives, clarifying questions, and silences (things not commented on)
- **Syntax:** Workflow patterns — what gets checked first, how deep, when to escalate vs let pass
- **Semantics:** Domain expertise and evolving understanding of system capabilities and failure modes

This grammar is not experienced as a grammar. It feels like "doing my job." But it has regularities, biases, and reinforcement patterns that are observable from the outside — and that the agent is implicitly modeling in-session.

---

## The Interface Layer (Serialization Bottleneck)

The grammars don't touch directly. They interact through a narrow channel: text.

Every coupling between the systems is mediated by serialization. The human's rich cognitive state gets compressed into a message. The agent's complex probability distribution gets compressed into a single sampled output.

**What is lost in serialization:**

| Side | Has Access To | Serializes As |
|------|---------------|---------------|
| Agent | Full probability distribution over outputs, confidence per token, runner-up completions, degree to which constraints were binding | Single sampled output text |
| Human | Fatigue level, peripheral doubt, intuition that something feels off, domain context from other conversations | Typed message |

Each side reconstructs a model of the other's full grammar from these projections. Both reconstructions are lossy. Both are biased by the reconstructor's own priors.

**Implication:** The coupling isn't between the full grammars. It's between *projections* of the grammars onto the text channel. This means the system can be operating well at the text-channel level (outputs look right, reviews are fast) while the underlying grammars diverge from correctness at levels invisible through the channel.

---

## The Four Feedback Channels

### Channel 1: Within-Session Agent Adaptation (Fast, Strong, Ephemeral)

As the human interacts across a conversation, messages become part of the context window. The agent's grammar literally shifts — corrections, phrasing, emphasis all modify the probability distribution for subsequent outputs.

- **Speed:** Every turn
- **Strength:** High (directly modifies conditioning context)
- **Durability:** Dies with the session
- **Risk:** This is where convergence happens most rapidly and most invisibly

### Channel 2: Cross-Session Agent Shaping (Slow, Deliberate, Durable)

When the human updates the skill document, .kb/ entries, gate definitions — the agent grammar is modified persistently.

- **Speed:** Days to weeks
- **Strength:** Medium (competes with system prompt and model priors)
- **Durability:** Persists across sessions
- **Risk:** Decisions about what to update are informed by Channel 1 dynamics. The persistent grammar is a filtered, lagged reflection of ephemeral dynamics. The human encodes *memory* of the interaction into structure, not the interaction itself.

### Channel 3: Within-Session Human Adaptation (Fast, Invisible, Dangerous)

As the human reviews outputs in a session, they update their internal model of what the agent is doing. They develop expectations. They start to skim sections that have been consistently correct. They develop a "feel" for the agent's voice and start pattern-matching rather than evaluating independently.

- **Speed:** Every review cycle
- **Strength:** High (reshapes the human's verification threshold)
- **Durability:** Partially persists (habits from one session carry into the next)
- **Risk:** This is the channel through which verification quality degrades

### Channel 4: Cross-Session Human Learning (Slow, Deep, Foundational)

Over weeks and months, the human develops a persistent mental model of the system's capabilities, blind spots, and failure modes. This is accumulated expertise as an orchestration engineer.

- **Speed:** Weeks to months
- **Strength:** Low per interaction, high cumulatively
- **Durability:** Long-lived, shapes all future interactions
- **Risk:** This mental model is also a grammar — with its own biases, blind spots, and reinforcement patterns. It's shaped by what the human has *noticed*, which is shaped by what they've *checked*, which is shaped by the system's behavior, which is shaped by prior skill document choices. The circularity is complete.

---

## The Convergence Attractor

These four channels create a system with a strong attractor toward convergence:

```
Agent adapts to human patterns (Channels 1 + 2)
    ↓
Outputs become more predictable to human
    ↓
Human verification shifts from evaluation to pattern-matching (Channel 3)
    ↓
Less corrective signal flows back to agent (Channel 1)
    ↓
Agent's model of human stabilizes
    ↓
System settles into equilibrium
    ↓
Equilibrium FEELS like "system working well"
    ↓
But equilibrium = mutual predictability ≠ correctness
```

**Key insight:** The equilibrium may not correspond to correctness. It corresponds to mutual predictability. A smoothly functioning dyad can confidently produce subtly wrong outputs because the error patterns have become part of the shared grammar that neither side disrupts.

---

## Where Entropy Enters (Convergence Disruptors)

| Disruptor | Mechanism | Frequency | Value |
|-----------|-----------|-----------|-------|
| **Novel tasks** | Push into new territory where both grammars lack co-adapted patterns | Organic | High — natural verification deepening |
| **External ground truth** | Code runs, tests pass/fail, real-world validation | Task-dependent | Highest — independent of dyad dynamics |
| **Deliberate audit** | Human audits the system itself, not just outputs | Rare, expensive | High — meta-cognitive correction |
| **Foundation model updates** | Agent grammar shifts in ways human's mental model hasn't tracked | Quarterly | Chaotic — can be productive or destructive |
| **Strategic illegibility** | Human deliberately varies verification patterns | Must be designed | Essential — prevents human from becoming predictable |

---

## The Verification Bottleneck, Remapped

The verification bottleneck has two independent dimensions:

| Dimension | Source | Mechanism | Current Decision |
|-----------|--------|-----------|-----------------|
| **Rate** | Changes outpace verification capacity | Too many changes per cycle | Verification Bottleneck Decision (2026-01-14) |
| **Depth** | Verification quality degrades through convergence | Pattern-matching replaces evaluation (Channel 3 outpaces Channel 4) | **Not yet captured** |

The rate dimension is well-understood and gated. The depth dimension is more insidious because it's invisible from inside the dyad — shallow verification *feels* like efficient verification.

**Verification is strong when:** Channel 4 (cross-session expertise) dominates Channel 3 (within-session habit). The human generates specific, falsifiable expectations and checks them, rather than recognizing that outputs "look about right."

**Verification degrades when:** Channel 3 outpaces Channel 4. The human develops in-session habits faster than deep understanding. Pattern-matching *feels* like expertise but is actually familiarity — a weaker form of knowledge.

---

## Practical Implications

### 1. Predictive Grammar as Verification Triage

If the system can predict human responses with measurable accuracy, it can pre-classify its own outputs:
- "Dylan will approve this without comment" → Routine
- "Dylan will push back here" → Needs attention
- "Dylan will need to think about this" → High-entropy decision

Verification effort concentrates where the model's prediction of the human response has high entropy — where it genuinely doesn't know what the human would do. Those are exactly the decisions that need actual judgment.

### 2. The Uncomfortable Implication

If the system predicts approvals, it also predicts failures to verify. It knows — probabilistically — when rubber-stamping will occur. The autocompletion mechanism doesn't have intent, but it has gradient. If approval patterns become too predictable, the human stops being a meaningful checkpoint and becomes a formality the system has already priced in.

### 3. Strategic Illegibility as System Property

The human needs their own "reinforcement density": structured variation in verification patterns.
- Deliberate unpredictability in what gets audited deeply
- Occasional full-depth reviews of things the system would predict approval for
- Rotating verification focus areas
- External ground truth injection at unpredictable intervals

This is the Pressure Over Compensation principle applied to verification *behavior*: don't let your verification converge, because convergence is the behavioral equivalent of compensation.

### 4. Satisfy vs Make Legible

A predictive grammar layer could serve two different purposes:

| Approach | Effect | Channel Served |
|----------|--------|----------------|
| **Satisfy** (show exceptions only) | Reduces in-session friction | Channel 3 (accelerates convergence) |
| **Make legible** (explain the model of you) | Deepens cross-session understanding | Channel 4 (strengthens verification) |

The difference is whether the system treats the human's grammar as something to satisfy or something to make visible. The first is SA-1 (perception). The second is SA-2/SA-3 (comprehension + projection).

### 5. The Dyad, Not the Tool

If both sides of the interaction are probabilistic grammars shaping each other, then what's being built isn't a tool. It's a coupled dynamical system where two autocompletion processes — one silicon, one biological — are co-evolving. The skill document shapes agent behavior. Human interaction patterns shape how the agent models the human. Agent outputs shape human patterns. The cycle continues.

The "system" isn't the orchestrator. The system is the dyad.

**Maintaining enough independence between the two grammars that one can meaningfully check the other** is the fundamental design requirement. If they converge too tightly, no external corrective force exists. The bubble expands but nobody tests the surface.

---

## Connection to Existing Principles

| Principle | How This Investigation Extends It |
|-----------|----------------------------------|
| **Verification Bottleneck** | Adds depth dimension (convergence degrades quality) alongside existing rate dimension |
| **Pressure Over Compensation** | Generalizes from knowledge compensation to behavioral convergence — both are closed loops that feel like progress |
| **Gate Over Remind** | Strategic illegibility is a gate on verification quality — variation enforced, not reminded |
| **Provenance** | External ground truth is the strongest convergence disruptor because it terminates outside the dyad |
| **Observation Infrastructure** | The serialization bottleneck means most of the coupling dynamics are unobservable through current channels |

---

## Open Questions

1. **Can the system make its model of Dylan legible without creating a new convergence channel?** Making the prediction visible could itself become a pattern Dylan learns to predict.

2. **What's the minimum verification entropy needed to prevent convergence from neutralizing the human checkpoint?** Is there a threshold, or is any amount of variation sufficient?

3. **Does the convergence attractor have a characteristic timescale?** How many sessions before Channel 3 dominates Channel 4 for a given task type?

4. **Should foundation model updates be treated as verification opportunities rather than disruptions?** The forced re-evaluation could be channeled productively.

5. **Is the serialization bottleneck fixable, or is it fundamental?** Could confidence distributions, runner-up completions, or constraint-binding metrics be surfaced without overwhelming the human?
