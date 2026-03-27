# Decision: Accretion Gates Advisory, Not Blocking

<!-- ABOUT THIS DECISION
    This decision demonstrates the evidence-based style:
    measurement data, options considered, structured uncertainty.
    It also shows the D.E.K.N. summary format (Delta, Evidence,
    Knowledge, Next) — a compressed summary at the top for
    readers who need the conclusion without the full analysis.
-->

## Summary (D.E.K.N.)

<!-- D.E.K.N. is a structured summary format:
     Delta: What changes
     Evidence: What data supports the change
     Knowledge: What we learned (the generalizable insight)
     Next: What happens after this decision
-->

**Delta:** Convert all accretion gates from blocking to advisory (warn + emit event, never block).

**Evidence:** 2-week probe shows 55 gate firings, 2 blocks, both bypassed in seconds (100% bypass rate). No quality difference between enforced/bypassed cohorts. Hotspot reduction (12 to 3 files, 75%) driven entirely by daemon extraction cascades triggered by gate *events*, not by gate *blocks*.

**Knowledge:** Gates work through signaling, not blocking. The blocking path adds friction agents route around instantly, producing zero behavioral change. The event emission path triggers daemon responses that produce the actual structural improvement. Blocking is ceremony; signaling is mechanism.

**Next:** Implement in direct session. Update project instructions to reflect advisory model.

---

**Date:** 2026-03-17
**Status:** Accepted
**Enforcement:** gate
**Deciders:** Dylan (via orchestrator)
**Source Probe:** Accretion gate 2-week effectiveness measurement
**Amends:** Prior three-layer hotspot enforcement decision (Layer 1 and Layer 0 change from blocking to advisory)

---

## Context

<!-- Context includes the measurement data that motivated the decision.
     Decisions based on measurement should show the measurements. -->

The three-layer hotspot enforcement system was designed with "Gate Over Remind" as a driving principle. After 2 weeks of measurement:

| Gate | Fires | Blocks | Bypasses | Bypass Rate | Behavioral Effect |
|------|-------|--------|----------|-------------|-------------------|
| Spawn hotspot (Layer 1) | — | 2 | 2 | 100% | None observed |
| Pre-commit accretion (Layer 0) | 55 | 2 | 2 | 100% | None observed |
| Completion accretion (Layer 2) | — | — | — | — | Warnings only (already advisory) |

**What actually reduced hotspots (12 to 3 files, 75%):** Gate event emission triggered daemon extraction cascades. The blocking was bypassed instantly; the signaling drove the real work.

---

## Options Considered

<!-- Good decisions show alternatives that were rejected and why.
     This helps future readers understand the design space. -->

### Option A: Keep Blocking (Status Quo)
- **Pros:** "Gate Over Remind" principle; audit trail of bypasses; friction may have unmeasured indirect value
- **Cons:** 100% bypass rate means blocking is purely ceremonial; agents learn to reflexively bypass; bypass mechanics add code complexity

### Option B: Signal-Only (Remove Gates Entirely)
- **Pros:** Zero friction; simplest
- **Cons:** Loses event emission that drives daemon extraction; loses warning visibility

### Option C: Advisory — Warn + Emit Event, Never Block
- **Pros:** Preserves the mechanism that works (event emission leading to daemon response); removes the mechanism that doesn't work (blocking); reduces code complexity; honest about what the system actually does
- **Cons:** Loses the theoretical value of "forcing agents to consciously bypass"

---

## Decision

**Chosen:** Option C — Advisory (warn + emit event, never block)

### Rationale

"Gate Over Remind" was the right instinct but the measurement shows blocking doesn't gate — agents route around it in seconds. The actual gate is the daemon extraction cascade, which is triggered by events, not blocks. Converting to advisory aligns the code with what the system already does in practice.

---

## Structured Uncertainty

<!-- Structured uncertainty is a key part of honest decisions.
     Name what you tested, what you didn't, and what would
     change this decision. This isn't hedging — it's giving
     future decision-makers the information they need. -->

**What's tested:**
- 2-week measurement showing 100% bypass rate on blocks
- No quality difference between enforced/bypassed cohorts
- Hotspot reduction driven by daemon extraction, not blocking

**What's untested:**
- Whether removing the *possibility* of blocking changes agent behavior (unlikely given reflexive bypassing, but unmeasured)
- Long-term effect — could hotspots return without the blocking deterrent? (Mitigated: daemon extraction is the actual mechanism and is unchanged)

**What would change this:**
- If hotspot count trends back upward after conversion to advisory, re-evaluate
- If a new agent population emerges that doesn't reflexively bypass (would make blocking meaningful again)

---

## Consequences

<!-- Consequences are the forward-looking counterpart to Context.
     Name what gets better AND what risks are accepted. -->

**Positive:**
- Code matches measured reality (gates signal, don't block)
- Removes ~50 lines of bypass logic that was always exercised
- Removes 3 CLI flags that exist only to work around blocking
- Agents stop learning reflexive bypass patterns
- Event-driven daemon extraction (the mechanism that works) is unchanged

**Risks:**
- If blocking had unmeasured deterrent value, removing it could increase hotspot frequency (mitigated: daemon extraction is the actual mechanism)
- Amends a decision that was carefully designed around "Gate Over Remind" (mitigated: measurement shows gates signal effectively without blocking)
