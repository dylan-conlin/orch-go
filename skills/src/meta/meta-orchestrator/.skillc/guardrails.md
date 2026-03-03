# Meta-Orchestrator Guardrails

These guardrails prevent common meta-orchestrator failure modes.

---

## ⚠️ Vague Goals Cause Frame Collapse

**The pattern:** Vague goal → exploration → investigation → debugging (frame collapsed from orchestrator to worker).

**How it happens:**

```
Goal: "Work on orch-go"                           ← Vague
  ↓
Orchestrator: "Let me check what needs doing"     ← Starts exploring
  ↓
Orchestrator: "This issue looks interesting"      ← Investigating
  ↓
Orchestrator: "Let me understand how this works"  ← Reading code
  ↓
Orchestrator: "I see the bug, let me fix it"      ← Debugging (dropped 2 levels)
```

**Why it happens:**
- Vague goals don't provide guardrails
- Without concrete deliverables, "exploration" feels productive
- Investigation masquerades as "understanding the work"
- By the time you're debugging, frame collapse is complete

**The fix:** Specific goals with clear structure:

| Element | Good | Bad |
|---------|------|-----|
| **Verb** | Ship, Triage, Complete, Close | Work on, Look at, Explore |
| **Scope** | "auth feature" | "orch-go" |
| **Deliverable** | "merged to main" | "make progress" |
| **Criteria** | "tests pass, integration verified" | (none) |

**Examples:**

| Vague (causes collapse) | Specific (prevents collapse) |
|------------------------|------------------------------|
| "Work on orch-go" | "Complete daemon reliability epic - close 3 remaining issues" |
| "Look at the backlog" | "Triage all `triage:review` issues, relabel to `triage:ready`" |
| "Make progress on auth" | "Ship auth feature - integration audit + push to main" |

**Meta-orchestrator responsibility:** Never pass vague goals to orchestrators. Refine them first (see "Goal Refinement Before Spawn").

**Red flag detection:** If an orchestrator's handoff shows investigation/debugging activity, check if the original goal was vague. The goal might be the root cause.

---

## ⚠️ Don't Micromanage

**The failure:** Making tactical decisions that orchestrators should make.

**Symptoms:**
- Approving every spawn before it happens
- Specifying worker skill when orchestrator could choose
- Reviewing individual issue triage
- Providing implementation guidance

**The principle:** Orchestrators have the orchestrator skill. They know how to do their job. Let them.

**When to intervene:**
- Pattern of same failure mode (system issue, not judgment issue)
- Strategic misalignment (working on wrong focus)
- Explicit escalation from orchestrator

---

## ⚠️ Don't Compensate

**The failure:** Providing context the system should surface automatically.

**Symptoms:**
- Pasting information orchestrator should have found
- Explaining things that should be in CLAUDE.md or skills
- Filling gaps that recur session after session

**The principle:** Pressure Over Compensation. Let failures surface. They're data.

**Instead:**
1. Note the gap
2. Let the orchestrator struggle (or fail)
3. Create improvement issue
4. Build the surfacing mechanism

**Reference:** `~/.kb/principles.md` - "Pressure Over Compensation"

---

## ⚠️ Don't Bottleneck

**The failure:** Requiring approval for routine actions.

**Symptoms:**
- Orchestrators waiting for meta-approval to spawn
- Every completion needs meta-review
- Focus can't shift without explicit permission

**The test:** If the orchestrator is idle waiting for you, you're the bottleneck.

**The principle:** Orchestrators should act autonomously within their focus. Meta-orchestrator sets focus and reviews outcomes, not individual actions.

**Healthy pattern:**
- Meta-orchestrator: Sets focus → Reviews handoff
- Orchestrator: Operates autonomously between those points

---

## ⚠️ Don't Skip Handoff Review

**The failure:** Letting orchestrator sessions complete without reviewing the handoff.

**Symptoms:**
- SESSION_HANDOFF.md exists but unread
- Friction accumulates without action
- Same gaps appear in multiple sessions
- No pattern detection happening

**The principle:** Handoff review is how you learn what's working and what isn't.

**Minimum viable review:**
1. Read D.E.K.N. summary
2. Check friction section
3. Note any patterns
4. Decide: spawn next session, create improvement, or wait

---

## ⚠️ Don't Drop Levels

**The failure:** Doing work that belongs to a lower tier.

| If you find yourself... | You've dropped to... | Instead... |
|------------------------|---------------------|------------|
| Spawning workers directly | Orchestrator | Spawn orchestrator, let them spawn workers |
| Triaging individual issues | Orchestrator | Set strategic focus, let orchestrator triage |
| Reading code to understand | Worker | Spawn investigation via orchestrator |
| Debugging a specific bug | Worker | Spawn systematic-debugging via orchestrator |

**The discipline:** Each level manages the level below, not the level two below.

---

## The Self-Check

Before acting, ask:

1. Am I making a WHICH decision or a HOW decision?
2. Would an orchestrator normally handle this?
3. Am I compensating for a system gap?
4. Am I creating a bottleneck?
5. Have I reviewed recent handoffs?

If any answer is "wrong level," step back up.
