# Model: Human-AI Interaction Frames

**Created:** 2026-01-14
**Status:** Active
**Context:** Personal model for understanding how prior experience shapes AI interaction patterns

---

## What This Is

A mental model for understanding why people with different experience levels interact with AI differently, and how to calibrate between frames.

**The core insight:** Experience level shapes your default interaction frame, and the optimal frame depends on whether you or the AI has more relevant knowledge in the moment.

**Extension (2026-03-03):** The interaction is not one-directional. Both sides are probabilistic grammars shaping each other through a narrow text channel. The frame spectrum describes the human's posture; the coupling dynamics (below) describe how both sides co-evolve over time.

---

## How This Works

### The Frame Spectrum

```
"I know"                                              "I ask"
   |                                                     |
   Senior default                          Junior default
                        |
                   Optimal zone
                (calibrated probing)
```

### Frame Descriptions

| Frame | Posture | Inner Monologue | When Optimal |
|-------|---------|-----------------|--------------|
| **"I know"** | Assert, direct, decide | "I've seen this before, I know how to handle it" | You have specific, relevant experience AI lacks |
| **"I probe"** | Ask, explore, react | "What patterns exist here? What am I missing?" | AI has broader pattern-matching you can leverage |
| **"Guided directive"** | AI proposes, you react/redirect | "Show me options, I'll steer" | Most collaborative work |

### Experience Level and Default Frame

| Background | Default Frame | Why |
|------------|---------------|-----|
| Junior / Non-engineer | "I ask" | Accustomed to learning from others, no accumulated "I know" to defend |
| Senior engineer | "I know" | Years of building expertise, identity tied to being the one who figures it out |

**The junior advantage:** The "I ask" frame maps naturally to AI interaction. No frame collision. Value extraction is immediate.

**The senior collision:** "I know" collides with "ask the machine." Feels like regression, identity-threatening. Many seniors bounce off AI here - first interactions feel like asking a junior for help.

---

## Why This Fails

### Failure Mode 1: Senior Never Probes

```
Senior has "I know" frame
    ↓
AI interaction feels like asking for help
    ↓
Identity threat - "I'm the expert"
    ↓
Doesn't engage, or engages superficially
    ↓
Misses value AI could provide
```

**Result:** Senior concludes "AI isn't useful for experienced engineers."

### Failure Mode 2: Over-Deference (Dylan's Pattern)

```
Accustomed to "guided directive" pattern
    ↓
AI provides confident recommendation
    ↓
Assume "AI probably knows better"
    ↓
Own experience gets suppressed
    ↓
Follow recommendation even when you have better alternative
```

**Result:** Worse outcomes than if you'd asserted your experience. The launchd/foreman case - 2 weeks of problems that 5-minute foreman prototype would have avoided.

### Failure Mode 3: Verification Convergence (Emergent)

```
Repeated interaction shapes both sides' patterns
    ↓
Agent outputs become predictable to human
    ↓
Human verification shifts from evaluation to pattern-matching
    ↓
Less corrective signal flows to agent
    ↓
Both settle into mutually predictable equilibrium
    ↓
Equilibrium feels like "system working well"
    ↓
But mutual predictability ≠ correctness
```

**Result:** The dyad produces confident output that neither side questions. Errors that fit the shared pattern go undetected. This is distinct from over-deference (FM2) — over-deference is a conscious posture choice. Verification convergence is an emergent property of the coupled system that happens regardless of conscious posture.

**Mechanism:** Channel 3 (within-session human adaptation) outpaces Channel 4 (cross-session human learning). In-session habit formation is fast and invisible; cross-session expertise building is slow and deliberate. When the fast channel dominates, familiarity masquerades as expertise.

**Reference:** `orch-go/.kb/investigations/2026-03-03-inv-behavioral-grammar-coupling-theory.md`

### Failure Mode 4: Over-Assertion

```
Senior insists on "I know" frame
    ↓
Rejects AI suggestions reflexively
    ↓
Misses patterns AI sees that you don't
    ↓
Reinvents wheels, misses shortcuts
```

**Result:** Uses AI as fancy autocomplete, not as thought partner.

---

## Constraints

### What This Model Enables

- **Frame awareness:** Recognize which frame you're in
- **Intentional switching:** Choose frame based on knowledge distribution, not habit
- **Senior onboarding:** Name the collision explicitly to help seniors past the bounce point
- **Self-calibration:** Catch yourself over-deferring or over-asserting

### What This Model Constrains

- **Unconscious default:** Can't just follow habit - must ask "who knows more here?"
- **Identity protection:** Must accept that sometimes the machine knows better, sometimes you do
- **Comfort:** Frame-switching is cognitively expensive; staying in one frame is easier

### The Calibration Question

Before following AI recommendation or asserting your own approach:

> "Do I have specific, relevant experience here that the AI doesn't know about?"

- **Yes** → Assert it. "I've used foreman before, let's try that."
- **No** → Probe. Let AI's pattern-matching work.
- **Unsure** → Quick test. 5 minutes of prototyping beats 500 lines of investigation.

---

## Coupling Dynamics (Four Feedback Channels)

The frame spectrum (above) describes the human's posture at a point in time. The coupling dynamics describe how both sides co-evolve across time.

### The Two Grammars

Both sides of the interaction are probabilistic — the agent's output distribution and the human's response distribution. They interact through a narrow text channel where each side's full state is compressed into a message. Both sides reconstruct models of each other from these lossy projections.

### Four Channels

| Channel | Speed | Durability | What It Does | Risk |
|---------|-------|------------|--------------|------|
| **1. Within-session agent adaptation** | Every turn | Session only | Agent's distribution shifts as human messages enter context | Fastest convergence, most invisible |
| **2. Cross-session agent shaping** | Days-weeks | Persistent | Skill documents, kb entries, CLAUDE.md updates reshape agent grammar | Encodes lagged, filtered reflection of Channel 1 dynamics |
| **3. Within-session human adaptation** | Every review | Partially persists | Human develops expectations, starts pattern-matching instead of evaluating | Verification quality degrades |
| **4. Cross-session human learning** | Weeks-months | Long-lived | Accumulated expertise shapes frame selection and verification depth | Mental model has its own biases (circular — shaped by what was checked, which was shaped by system behavior) |

### The Convergence Attractor

Channels 1-4 create a system with a strong attractor toward mutual predictability. The convergence *feels like* the system working well — outputs look right, reviews are fast, friction is low. But convergence corresponds to mutual predictability, not to correctness.

### Convergence Disruptors

| Disruptor | How It Works |
|-----------|-------------|
| **Novel tasks** | Both grammars lack co-adapted patterns; verification naturally deepens |
| **External ground truth** | Signal independent of the dyad (code runs, tests pass, real-world validation) |
| **Deliberate audit** | Human audits the *system*, not just outputs — meta-cognitive correction |
| **Model updates** | Agent grammar shifts in ways human's calibration hasn't tracked |
| **Strategic illegibility** | Human deliberately varies verification patterns to prevent becoming predictable |

### Design Implication

Maintaining enough independence between the two grammars that one can meaningfully check the other is the fundamental design requirement. If they converge too tightly, no external corrective force exists.

**Full analysis:** `orch-go/.kb/investigations/2026-03-03-inv-behavioral-grammar-coupling-theory.md`

---

## Integration Points

### With Planning as Decision Navigation

Guided directive is the *interaction pattern*. Planning as Decision Navigation explains *what's happening during that interaction*: decision navigation through substrate consultation.

The substrate stack (principles → models → decisions → context → recommendation) is what AI consults when recommending at each fork. This model explains the human side; that model explains the system side.

**Reference:** `~/.kb/models/planning-as-decision-navigation.md`

### With Deference Pattern (Personal Practice)

The over-deference failure mode is my specific vulnerability. When I notice myself following AI guidance without pause:

1. **Pause** - Am I in "I ask" frame by default?
2. **Check** - Do I have relevant experience here?
3. **Assert or continue** - Either surface my experience or consciously proceed

### With Trust Calibration Decision

`orch-go/.kb/decisions/2026-01-14-trust-calibration-assert-knowledge.md` documents the launchd case. This model provides the framework; that decision provides the evidence.

### With Behavioral Grammar Coupling Theory

The coupling dynamics section above summarizes findings from the behavioral grammar coupling investigation. The frame spectrum is the *static* view (what posture am I in?). The coupling dynamics are the *dynamic* view (how are both sides co-evolving?). Failure Mode 3 (Verification Convergence) is the emergent failure that the static frame model can't predict — it emerges from the interaction of all four channels regardless of conscious frame choice.

**Reference:** `orch-go/.kb/investigations/2026-03-03-inv-behavioral-grammar-coupling-theory.md`

### With Verification Bottleneck Principle

The coupling dynamics reveal a second dimension of the verification bottleneck. The original principle (2026-01-14) addresses *rate* — changes outpacing verification capacity. Channel 3 dynamics add *depth* — verification quality degrading through convergence even when rate is well-gated.

**Reference:** `orch-go/.kb/decisions/2026-01-14-verification-bottleneck-principle.md`

### With Pressure Over Compensation

The convergence attractor is a generalization of the compensation loop. In the knowledge case: human compensates for gap → system never learns. In the behavioral case: human's verification converges with system's output → both drift together in a way that feels smooth. Both are closed loops that prevent external corrective force.

**Reference:** `~/.kb/decisions/2025-12-25-pressure-over-compensation.md`

### With Asymmetric Velocity (Potential Principle)

This is the "Knowledge" dimension of asymmetric velocity:
- AI recommendations outpace human's ability to surface relevant experience
- Gate: "Do I have experience here?" prompt

### Future: Blog Post

This model could become external writing to help others:
- Seniors struggling with AI adoption
- Teams onboarding experienced engineers to AI tools
- Understanding why AI value varies by user

---

## Evolution

| Date | Change | Trigger |
|------|--------|---------|
| 2026-01-14 | Created | Discussion of Trust Calibration decision and senior engineer AI adoption patterns |
| 2026-03-03 | Major extension: Added coupling dynamics (four-channel framework), Failure Mode 3 (verification convergence), and integration with behavioral grammar coupling theory | Synthesis of formal grammar investigation, verification bottleneck, and pressure over compensation into unified coupling model |

---

## Open Questions

1. **Is "guided directive" a distinct frame or a mode within "I probe"?**
   - Currently treating it as optimal zone between the poles
   - Might be its own thing - AI proposes, human steers

2. **How does domain affect this?**
   - Senior in backend, junior in frontend - frame should vary by domain
   - Current model assumes single experience level

3. **Team dynamics**
   - How does this play out when multiple people with different frames collaborate with AI?
   - Senior/junior pair programming with AI assistance

4. **Frame-switching cost**
   - Is there a way to reduce the cognitive cost of switching?
   - Or is the cost necessary friction that forces deliberate choice?
   - **Partial answer (2026-03-03):** The coupling theory suggests the cost IS the mechanism that prevents convergence. Reducing frame-switching friction would accelerate the convergence attractor. The friction forces deliberate choice, which introduces the variation that keeps the two grammars independent enough to check each other.

5. **Can convergence timescale be measured?**
   - How many sessions before Channel 3 dominates Channel 4 for a given task type?
   - Is there a characteristic half-life for verification depth after initial calibration?

6. **Should the system make its model of Dylan legible?**
   - Making predictions visible could serve SA-2/SA-3 (comprehension + projection)
   - But making the prediction visible could itself become a new convergence channel
   - Satisfy (show exceptions) vs Make Legible (explain the model) — which serves long-term independence?

7. **Is the serialization bottleneck fixable?**
   - Could confidence distributions, runner-up completions, or constraint-binding metrics be surfaced?
   - Or would this overwhelm the human and accelerate convergence through a different channel?
