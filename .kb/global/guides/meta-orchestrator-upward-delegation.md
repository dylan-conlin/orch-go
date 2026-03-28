# Meta-Orchestrator Upward Delegation Patterns

**Purpose:** How meta-orchestrators (Dylan) delegate upward to executives (Jim) using capacity allocation instead of forced choices.

**Origin:** 2026-01-14, price-watch/toolshed/specs-platform prioritization conversation with Jim.

---

## The Pattern

**Don't ask:** "Which ONE should I do first?" (forced choice)

**Do ask:** "Here's my proposed allocation - does this match your priorities?" (propose-and-adjust)

---

## Why This Matters

Forced choices assume **sequential work** when the executive is thinking in **parallel work streams**.

When you ask "A or B?", the executive has to:
1. Invent the capacity allocation model
2. Fit their priorities into your framing
3. Add caveats to communicate parallel needs

When you propose an allocation, the executive can:
1. React to a concrete model
2. Adjust the balance
3. Confirm or redirect

---

## Example: What Happened

**What I sent Jim (forced choice):**

> Hey Jim - need priority guidance on three projects:
>
> 1. Specs-platform (Lea waiting) - 2-4 weeks
> 2. Toolshed (Phil + You waiting) - 1-2 weeks scaffolding
> 3. Price-watch (running fine) - ongoing
>
> **My question:** Lea or toolshed first?

**Jim's response (revealed parallel thinking):**

> Let's do the Lea thing next, but please leave room to polish Price Watch. Price Watch is amazing and it's probably the most valuable tool you've ever built for me. I love it. I want to make sure that you have time to address bugs and such long term

**What this revealed:**
- Jim wasn't thinking "one thing at a time"
- He wanted Lea as PRIMARY focus, but with BANDWIDTH for price-watch
- The forced choice made him have to add the caveat

---

## Better Framing: Capacity Allocation

**What I could have sent:**

> Hey Jim - need your input on how I'm allocating time across three projects:
>
> **Proposed allocation (40 hrs/week):**
> - **Specs-platform (Lea):** 30 hrs/week - Mobile + OAuth + admin CRUD (blocking main site launch)
> - **Price-watch (You):** 5 hrs/week - Bug fixes, polish, ongoing maintenance
> - **Toolshed (Phil + You):** 5 hrs/week - Incremental scaffolding (unlocks Phil's features + your second brain)
>
> **Rationale:**
> - Lea's been waiting longest and is blocking customer-facing launch
> - Price-watch is production-critical, needs steady maintenance bandwidth
> - Toolshed can progress incrementally while focused on Lea
>
> Does this allocation match your priorities, or would you adjust the balance?

**Why this is better:**
- Shows you've thought through the capacity constraints
- Makes parallel work explicit (not sequential)
- Gives Jim something concrete to react to
- Respects that he values price-watch highly (allocates ongoing bandwidth)
- Lets him adjust percentages rather than invent the model

---

## The Delegation Pattern Hierarchy

The same principles apply at every level:

### AI Orchestrator → Agents
- Propose approach, let agent execute
- Don't ask permission for obvious next steps
- Present options with recommendations

### Dylan (Meta-Orchestrator) → AI Orchestrator
- Propose approach, let AI orchestrator execute
- Don't ask permission for obvious allocations
- Present options with recommendations

### Dylan (Meta-Orchestrator) → Jim (Executive)
- Propose capacity allocation, let Jim adjust
- Don't force sequential choices when work is parallel
- Present allocation model with rationale

---

## When to Use This Pattern

**Use capacity allocation when:**
- Multiple projects compete for your time
- Executive cares about multiple things simultaneously
- Work can be parallelized (even with different time splits)
- You need executive input on balance, not just permission

**Use forced choice when:**
- Resources truly can't be split (e.g., single deployment slot)
- Decision is about direction, not capacity
- Options are mutually exclusive

---

## Applying Orchestrator Autonomy Principles

From the orchestrator skill's autonomy guidance:

**Propose-and-act:** "I'm allocating X/Y/Z hours - does this work?" (not "may I allocate?")

**Filter before presenting:** Only include allocations you'd actually recommend. Don't present "focus 100% on Lea" alongside "split evenly 3 ways" if you know split-evenly would leave all three projects half-done.

**Surface decision prerequisites:** Before asking Jim to allocate, show you understand:
- Who's waiting (Lea, Phil)
- What's production-critical (price-watch)
- What's dependent (toolshed unlocks second-brain)
- What the time estimates are

---

## Relationship to Principles

**Perspective is Structural:** You couldn't see "capacity allocation" from inside "prioritization" frame. Meta-level provides that perspective.

**Surfacing Over Browsing:** Present the allocation model, don't make Jim build it from project descriptions.

**Surface Decision Prerequisites:** Teach the context (who's waiting, why, what depends on what) before asking to decide on allocation.

---

## Common Failure Modes

**Failure Mode 1: Ask "which one?"**
- Forces sequential thinking when work is parallel
- Executive has to add caveats to communicate ongoing needs

**Failure Mode 2: Just pick one without asking**
- Misses executive's priority signals (Jim values price-watch highly)
- Can lead to "why are you ignoring X?" later

**Failure Mode 3: Present allocation but no rationale**
- "30/5/5 - does that work?"
- Executive doesn't know WHY you chose those numbers, can't evaluate if adjustment needed

**Correct Pattern: Allocation + Rationale + Adjustment Path**
- Here's the split
- Here's why I think this serves your priorities
- Does this match what you're thinking, or would you adjust?

---

## Follow-up After Initial Response

If Jim just says "do Lea next" without addressing the allocation:

**Option A: Assume maintenance bandwidth**
- Interpret "do Lea next" as "Lea is primary focus"
- Allocate small bandwidth to price-watch (5 hrs/week) proactively
- Update Jim if priorities shift

**Option B: Clarify the allocation**
- "Thanks for the direction. To make sure I'm balancing this right - I'm thinking:
  - 30 hrs/week on specs-platform (Lea's blocker)
  - 5 hrs/week on price-watch polish (bugs, enhancements)
  - 5 hrs/week on toolshed scaffolding (incremental progress)
- Does that allocation match what you're envisioning?"

Use Option B when:
- Executive's response doesn't address all competing priorities
- You suspect there's an implicit expectation you're missing
- The allocation has meaningful tradeoffs (e.g., 35/5/0 vs 25/10/5)

---

## Template: Capacity Allocation Request

```
Hey [Executive] - need your input on how I'm allocating time across [N] projects:

**Proposed allocation ([X] hrs/week):**
- **[Project 1]:** [Y] hrs/week - [What] ([Why it matters])
- **[Project 2]:** [Z] hrs/week - [What] ([Why it matters])
- **[Project 3]:** [W] hrs/week - [What] ([Why it matters])

**Rationale:**
- [Why this balance serves their priorities]
- [What tradeoffs this represents]
- [What's blocking/urgent]

Does this allocation match your priorities, or would you adjust the balance?
```

---

## Provenance

**Date:** 2026-01-14

**What happened:** Asked Jim "Lea or toolshed first?" Jim responded "Lea next, but leave room for price-watch." This revealed he was thinking in parallel capacity, not sequential priority.

**Insight:** Meta-orchestrators should propose capacity allocations (with rationale) rather than forcing executives to choose between sequential options. Applies same "propose-and-adjust" pattern that orchestrators use with agents.

**Application:** When multiple projects compete for Dylan's time and Jim cares about multiple things simultaneously, frame as capacity allocation with rationale, not forced choice.
