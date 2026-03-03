# Principles as Trust Infrastructure

**Context:** Emerged from strategic session (Feb 28, 2026) reflecting on what Price Watch's development methodology means beyond the orchestration system.

---

## The Reframe

The principles in `~/.kb/principles.md` were written as operational scaffolding for AI agent orchestration — how to make human-AI collaboration produce reliable output. But they answer a bigger question:

**"How do you trust what AI builds?"**

Anyone building production systems with AI faces this question. The principles are the answer — not as theory, but as a catalog of specific failure modes and the mechanisms that prevent them.

## The Trust Problem

When one person builds a system by orchestrating AI agents — without reading or writing every line of code — trust becomes the central engineering challenge. The client/stakeholder question is always: "How do I know this works?"

Traditional answers fail:
- "I reviewed the code" — you didn't write it and may not have read every line
- "The tests pass" — tests were also written by AI
- "The agent says it works" — AI explaining AI is a closed loop

The principles provide structural answers:

| Trust Question | Principle | Mechanism |
|---------------|-----------|-----------|
| "How do I know the system works?" | Verification Bottleneck | A human observed every behavioral change — not "the agent said it works" |
| "How do I know claims are real?" | Provenance | Every conclusion traces to something verifiable outside the conversation |
| "Can someone else maintain this?" | Session Amnesia + Self-Describing Artifacts | The system carries its own operating instructions; the next person can resume without the builder |
| "What if AI made a mistake?" | AI Validation Loops | Methodology explicitly rejects AI explanations of AI work as verification |
| "What about edge cases?" | Gate Over Remind | Critical checks are enforced structurally, not suggested |
| "Won't patches accumulate?" | Coherence Over Patches | 3+ fixes to the same area triggers architectural review, not more patches |
| "How do you catch silent failures?" | Pain as Signal + Observation Infrastructure | Friction is surfaced to the agent in real-time; invisible states are treated as P1 bugs |

## Why This Matters

Most people building with AI in 2026 operate without any of this. They prompt, get output, ship it, hope. They haven't experienced the specific failures that generated these principles:

- **347 rolled-back commits** because the system self-reported healthy while deteriorating (→ Verification Bottleneck)
- **7,344 silently corrupted records** from an API lockdown the system didn't classify (→ Observation Infrastructure, Pain as Signal)
- **Correlation escalated to causation** without evidence of mechanism (→ Provenance)
- **Locally correct patches producing globally incoherent code** (→ Coherence Over Patches)

The principles are battle-tested against real production failures, not theoretical concerns.

## The Analogy

A structural engineer doesn't sell buildings. They sell the guarantee that the building won't fall down. Their methodology — load calculations, material testing, safety factors, code compliance — is what makes that guarantee credible.

These principles serve the same function. You don't sell the methodology. You sell the reliable system. The methodology is why the system deserves trust.

## Positioning

When explaining this work to others:

- **Not:** "I built this with AI" (invites skepticism)
- **Not:** "AI is amazing, it wrote everything" (invites dismissal)
- **Instead:** "I have a methodology for building with AI that has specific mechanisms to catch the 18 ways it goes wrong. Here's my failure catalog and what I do about each one."

That's what an experienced practitioner sounds like — not enthusiasm about the tool, but fluency with its failure modes.
