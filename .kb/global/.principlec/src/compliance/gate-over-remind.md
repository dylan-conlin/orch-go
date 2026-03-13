### Gate Over Remind

Enforce behavior through gates (cannot proceed without), not reminders (easily ignored). Measure whether gates improve outcomes.

**Why:** Reminders fail under cognitive load. When deep in a complex problem, "remember to update kn" gets crowded out. Gates make capture unavoidable. But gates without measurement are theology — you *believe* the gate works without evidence.

**The two-part test:**

1. "Is this a reminder that can be ignored, or a gate that blocks progress?"
2. "Can you measure whether this gate improves outcomes?"

**Part 1: Gate vs Reminder**

- Reminders: "Please update your investigation file" → ignored when busy
- Gates: Cannot `/exit` without kn check → capture happens

**Part 2: Measurement vs Faith**

A gate that blocks progress but has never been measured is an article of faith. Enforcement without measurement is theological — you believe the gate works because it feels rigorous. The harness work revealed this gap: gates existed for months before `orch harness audit` could answer "does this gate actually improve agent quality?"

| Gate State | What You Have |
|------------|---------------|
| Reminder only | Hope |
| Gate without measurement | Faith |
| Gate with measurement | Engineering |

**The measurement test:** "For agents that hit this gate (blocked/escalated), what was the outcome of the redirected work vs agents that passed through?" If you can't answer this, the gate is theological.

**Caveat: Gates must be passable by the gated party.**

A gate that the agent cannot satisfy by its own work is not a gate - it's a human checkpoint disguised as automation.

| Gate Type | Example | Outcome |
|-----------|---------|---------|
| **Valid gate** | Build must pass | Agent fixes build errors → proceeds |
| **Valid gate** | Test evidence required | Agent runs tests, reports output → proceeds |
| **Human checkpoint** | Repro verification | Requires orchestrator to manually verify → disabled |

**The refined test:**
1. Is this a reminder that can be ignored? → Make it a gate
2. Can the gated party pass it by their own work? → Valid gate
3. Does it require someone else to act? → Human checkpoint, not a gate
4. Can you measure whether it improves outcomes? → If not, instrument it

**Implementation patterns:**

- **Declarative gates:** Phase gates use HTML comment blocks in SPAWN_CONTEXT.md (`SKILL-PHASES`, `SKILL-CONSTRAINTS`) — declarative, backwards compatible, parseable.
- **Verification gates:** Constraint verification at completion time (`pkg/verify/constraint.go`) parses required/optional patterns from SPAWN_CONTEXT.md and blocks completion until satisfied.
- **Measurement infrastructure:** `orch harness audit` computes per-gate metrics (fire rate, pass rate, cost). Gate effectiveness query compares outcomes of gated vs ungated work.

**Evidence:** Mar 2026 harness engineering work. Gates existed since Jan 2026. Measurement capability (`orch harness audit`, gate event emission) added Mar 2026. For two months, gates were theological — enforced with conviction, measured never. The accretion gate checkpoint (Mar 24) is the first empirical test of whether structural enforcement improves agent quality.

**Provenance:** Harness engineering model synthesis (Mar 2026). Thread: "enforcement without measurement is theological" (Mar 12 2026).
