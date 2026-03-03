# Model: Planning as Decision Navigation

**Created:** 2026-01-14
**Status:** Active
**Context:** Understanding what "planning" means in an AI orchestration system

---

## What This Is

A mental model that reframes planning from task enumeration to decision navigation.

**The core insight:** A plan is "ready" not when tasks are listed, but when you have sufficient model to navigate the decisions ahead.

---

## How This Works

### The Reframe

| Traditional Planning | Decision Navigation |
|---------------------|---------------------|
| Plan = task list | Plan = decision map with informed recommendations |
| Ready = tasks enumerated | Ready = can navigate each fork |
| Execution = follow plan | Execution = steer through decisions |
| Replanning = rewrite task list | Replanning = update model when reality differs |

### The Planning Loop

```
Probing phase:
    Spawn investigations → gather facts → identify decision points

Model-building phase:
    Synthesize findings → build model → make constraints explicit

Planning phase:
    Surface each decision → consult model → recommend → human steers

Execution phase:
    Navigate based on decisions made → update model when reality differs
```

Planning isn't a phase that produces a document. It's the **transition from "I don't understand" to "I can navigate decisions."**

### The Substrate Stack

When making recommendations, consult layers in order. Strength of recommendation depends on whether a layer is **Instructional** (Map) or **Infrastructural** (Territory).

```
Principles (universal constraints)
    ↓
Infrastructure (enforced gates, manifests, plugins) [NEW]
    ↓
Models (domain-specific understanding)
    ↓
Decisions (past choices in this project)
    ↓
Guides/Instructions (procedural maps) [UPDATED]
    ↓
Current context (what we're doing now)
    ↓
Recommendation
```

Each layer informs the one below:
- **Principles:** Does this violate Verification Bottleneck? Session Amnesia?
- **Infrastructure:** Is there a Gate or Manifest enforcing this? (e.g., AGENT_MANIFEST.json baseline).
- **Models:** Does the spawn model allow this? What does dashboard-agent-status say?
- **Decisions:** Did we reject this approach before? Why?
- **Guides:** What is the documented procedure?
- **Context:** Given all that, which option fits the current situation?

**Strength of Recommendation:**
1. **Infrastructural:** "The system prevents this" (Strongest - e.g., Task tool blocked)
2. **Structural:** "The model forbids this" (Strong - e.g., Frame collapse detection)
3. **Instructional:** "The guide suggests this" (Guidance - e.g., Use feature-impl)

### Guided Directive as Planning Mechanism

The human-AI interaction pattern that enables decision navigation:

```
AI surfaces decision point (fork in the road)
    ↓
AI shows options with tradeoffs
    ↓
AI recommends based on substrate stack:
    - Principles
    - Models
    - Past decisions
    - Project constraints
    - "What others like you do in this situation"
    ↓
Human reacts/redirects
    ↓
Decision made, move to next fork
```

The plan emerges through this dialogue, not before it.

---

## Why This Fails

### Failure Mode 1: Task-List Planning

```
"Here are the 12 tasks to complete this feature"
    ↓
Execute task 1... hit unexpected decision
    ↓
No model to inform decision
    ↓
Guess, or stop and investigate
    ↓
Plan was illusion of readiness
```

**The problem:** Task lists assume decisions are already made. They're execution plans, not planning.

### Failure Mode 2: Planning Without Models

```
"We need to decide between A and B"
    ↓
No model of constraints, past failures, domain understanding
    ↓
Decision based on surface appeal or habit
    ↓
Reality differs from assumption
    ↓
Rework
```

**The problem:** Decisions without substrate are guesses.

### Failure Mode 3: Models Not Consulted

```
Models exist (spawn, dashboard, agent-lifecycle)
    ↓
New decision comes up in that domain
    ↓
AI recommends without checking models
    ↓
Recommendation violates existing constraints
    ↓
Implementation fails or conflicts
```

**The problem:** Models are passive documentation unless actively consulted.

---

## Learning Through Failure

### The Hard Truth

**Some knowledge is only accessible through failure.**

No amount of modeling gets you there. The constraint only reveals itself under load, in production, over time, when you try the thing.

Decision navigation doesn't eliminate this. It changes the relationship to it.

### How Task-List Planning Fails Here

```
Make comprehensive plan
    ↓
Execute plan
    ↓
Hit unknown constraint (only visible through failure)
    ↓
Plan is broken
    ↓
Learning is diffuse ("the plan failed") - which part? why?
```

The plan was a black box. You don't know which assumption was wrong.

### How Decision Navigation Handles This

| Aspect | Task-List | Decision Navigation |
|--------|-----------|---------------------|
| Bet size | Large (whole plan) | Small (each fork) |
| Feedback speed | End of plan | Each decision |
| Uncertainty | Hidden (plan looks complete) | Explicit (some forks unknown) |
| Learning capture | Diffuse ("plan failed") | Specific (which decision, why) |
| Model update | Rare (replan from scratch) | Continuous (each failure teaches) |

### The Key Mechanism: Failure Updates the Model

```
Make decision A at fork 1 based on model M
    ↓
Execute
    ↓
Reality reveals constraint C we didn't know
    ↓
Update model M with constraint C
    ↓
Next time, C is in the substrate
```

Task-list planning: "The plan failed."
Decision navigation: "Decision A was wrong because constraint C. Model updated."

The learning gets captured. The model evolves. Future decisions are better informed.

### Explicit Uncertainty as Valid State

"I don't have sufficient model for this fork" is a real answer. It triggers:

- **Probing:** Small experiment before committing
- **Prototyping:** 5 minutes of trying beats 500 lines of planning
- **Acknowledging the limit:** "We won't know until we try"

### Probing: Deliberate Small-Bet Failure-Seeking

For forks where failure is likely or costly:

> "I don't have sufficient model for this fork. Before deciding, probe."

Probing = small, cheap experiment to surface constraints before committing.

You're deliberately seeking the failure *before* committing, with small bet size. This is the "5 minutes of prototyping beats 500 lines of investigation" pattern.

### The Honest Limit

Decision navigation still can't know what's only learnable through failure. But it:

1. **Reduces bet size** - Each decision is smaller commitment than full plan
2. **Speeds feedback** - You learn sooner when you're wrong
3. **Makes uncertainty explicit** - "I don't know this fork" is valid, triggers probing
4. **Captures learning** - Model evolution compounds understanding over time

The failure mode it avoids: "We had a plan, we executed the plan, we failed, we learned nothing because the plan was a black box."

---

## Constraints

### What This Model Enables

- **Readiness assessment:** "Can I navigate the decisions?" not "Do I have a task list?"
- **Honest uncertainty:** "I don't have sufficient model for this fork" is valid state
- **Informed recommendations:** Each recommendation traces through substrate stack
- **Adaptive execution:** Plan updates as model updates, not as rework

### What This Model Constrains

- **Premature execution:** Can't execute until decision points are identified and navigable
- **Task-list theater:** "Here's the plan" that's really a guess
- **Context-free recommendations:** Must consult accumulated understanding

### The Readiness Test

Before declaring "ready to implement":

> Can I, for each decision point ahead, explain which option is better and why, based on principles, models, and past decisions?

If yes → ready
If no → still in probing/model-building phase

---

## Integration Points

### With Human-AI Interaction Frames

Guided directive is the interaction pattern. This model explains what's happening during that interaction: decision navigation through substrate consultation.

### With Epic Model Template

The template asks: "Can you explain it in 1 page?" This model explains what that means: you have sufficient understanding to navigate the decisions.

### With Understanding Through Engagement Principle

That principle says synthesis requires direct engagement. This model says why: you're building the substrate that enables decision navigation.

### With Models as Understanding Artifacts

Models aren't documentation. They're the substrate consulted during planning. This is their operational purpose.

### With Orchestrator Role

Orchestrators don't just spawn and monitor. They build models and navigate decisions. This is what "strategic comprehension" means operationally.

---

## Evolution

| Date | Change | Trigger |
|------|--------|---------|
| 2026-01-14 | Created | Discussion synthesizing guided directive, models as substrate, and planning |
| 2026-01-14 | Added "Learning Through Failure" section | Question: how does this handle things you can only learn by failing? |
| 2026-01-17 | Integrated Infrastructure vs. Instruction | Breakthrough: moving from "Decision Navigation" as mental exercise to tool-enforced territory. |

---

## Open Questions

1. **How to ensure models are consulted?**
   - Currently requires discipline
   - Could `kb context` be extended to surface relevant model constraints?
   - Should spawn prompts automatically include domain models?

2. **When is a model "sufficient"?**
   - What's the threshold for "can navigate decisions"?
   - Is there a checklist or heuristic?

3. **Model freshness**
   - Models can become stale as system evolves
   - How to detect when model no longer matches reality?
   - Evolution section helps, but is it enough?

4. **Decision documentation**
   - As decisions are made through navigation, should they become decision records?
   - When does a navigated decision warrant permanent capture?

5. **Multi-agent planning**
   - How does decision navigation work across multiple agents?
   - Can agents share decision context, or must orchestrator hold it?
