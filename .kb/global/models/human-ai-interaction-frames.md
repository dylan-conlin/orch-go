# Model: Human-AI Interaction Frames

**Created:** 2026-01-14
**Status:** Active
**Context:** Personal model for understanding how prior experience shapes AI interaction patterns

---

## What This Is

A mental model for understanding why people with different experience levels interact with AI differently, and how to calibrate between frames.

**The core insight:** Experience level shapes your default interaction frame, and the optimal frame depends on whether you or the AI has more relevant knowledge in the moment.

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

### Failure Mode 3: Over-Assertion

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
