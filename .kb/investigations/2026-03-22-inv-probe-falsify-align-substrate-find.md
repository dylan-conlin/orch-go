## Summary (D.E.K.N.)

**Delta:** "Substrate" overclaims — Align is a multiplier/validity condition on the other three primitives, not a substrate; Route/Sequence/Throttle mechanically operate without Align but produce no useful coordination value.

**Evidence:** 15 classified failure cases from 4 sources (MAST 14 modes, McEntire 4 architectures, 80-trial experiment, orch-go production). 5 cases show Align broken with Route/Sequence/Throttle mechanically holding. McEntire hierarchical achieved 64% success with only partial Align, proving proportional (not binary) dependency.

**Knowledge:** The asymmetry between Align and the other three is real (7/14 MAST failures, dominant in orch-go history), but "substrate" implies mechanical dependency that doesn't exist. "Validity condition" or "multiplier" is more precise.

**Next:** Update coordination model language — replace "meta-primitive"/"substrate" with "multiplier" or "validity condition" on the other three primitives' effectiveness.

**Authority:** implementation - Updates existing model within established patterns, no architectural impact

---

# Investigation: Probe Falsify Align-as-Substrate

**Question:** Can we find cases where Align is broken but Route/Sequence/Throttle still function correctly, falsifying the claim that Align is the substrate for the other three primitives?

**Started:** 2026-03-22
**Updated:** 2026-03-22
**Owner:** probe agent (orch-go-ekkut)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** coordination

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` | extends | Yes — MAST mapping and McEntire data verified | None |
| `.kb/models/coordination/model.md` | tests claim from | Yes — "meta-primitive" claim at line 75 | Extends: "substrate" overclaims |
| `.kb/threads/2026-03-22-coordination-protocol-primitives-route-sequence.md` | tests hypothesis from | Yes | None |

---

## Findings

### Finding 1: Five cases show Align broken with Route/Sequence/Throttle mechanically holding

**Evidence:** MAST FM-1.1 (agent disobeys task spec but is correctly routed/sequenced), McEntire hierarchical failures (36% of tasks — agents routed correctly but diverge from intent), launchd post-mortem (186 investigations correctly routed toward wrong solution), orch-go competing instructions (orchestrator correctly routed but system prompt overrides skill constraints), orch-go stale knowledge cascade (correctly routed to debugging with wrong model of the system).

**Source:** MAST taxonomy (14 modes), McEntire experiment (CIO article), `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md`, `.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md`, `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md`

**Significance:** Route/Sequence/Throttle CAN mechanically operate without Align. "Substrate" (binary dependency) is falsified. But in all 5 cases, the mechanical operation produced wrong outputs — the primitives ran but didn't produce useful coordination.

---

### Finding 2: McEntire hierarchical proves proportional (not binary) dependency

**Evidence:** McEntire's hierarchical architecture had partial Align (human orchestrator, but agents sometimes diverged) and achieved 64% success — not 0%. This means imperfect Align + working Route/Sequence/Throttle still produces value. Compare pipeline (all four broken, 0%) and single-agent (all four trivially satisfied, 100%).

**Source:** McEntire experiment: single 100%, hierarchical 64%, swarm 32%, pipeline 0%

**Significance:** If Align were a binary substrate, partial Align should produce 0% (like a broken foundation). Instead, success degrades proportionally. Align is a **multiplier** on the others' effectiveness: `Coordination_value ≈ Route × Sequence × Throttle × Align_quality`.

---

### Finding 3: The 80-trial messaging condition reveals Align's internal contradictions

**Evidence:** In 18/20 messaging trials, both agents wrote coordination plans acknowledging the other agent's work. Both agents AGREED on the correct insertion point ("after FormatDurationShort"). Both agents UNDERSTOOD the coordination goal ("don't conflict"). But task alignment ("this is the semantically correct location") defeated coordination alignment ("pick a different spot") in every trial. Agents had perfect state alignment and task alignment but zero coordination alignment.

**Source:** `experiments/coordination-demo/redesign/results/20260310-174045/analysis.md`, messaging prompt analysis showing conflicting placement instruction vs coordination instruction

**Significance:** Align itself contains sub-components (task, state, coordination) that can conflict. This supports the model's open question about decomposing Align into sub-primitives. It also shows that "Align holds" vs "Align is broken" is not binary — partial Align (state + task but not coordination) still produces individually correct but uncoordinated work.

---

## Synthesis

**Key Insights:**

1. **Mechanical vs functional distinction is the crux.** Route/Sequence/Throttle mechanically operate independently of Align (messages get delivered, steps get ordered, velocity gets limited). But their functional value — producing correct coordination outcomes — requires Align. "Substrate" conflates these two levels.

2. **Align is a multiplier, not a substrate.** Success degrades proportionally with Align quality (McEntire: 100% → 64% → 32% → 0%). A substrate would produce binary failure (works/doesn't work). A multiplier produces proportional degradation.

3. **The asymmetry is real but needs better language.** Align IS special — 7/14 MAST failure modes map to it, it's the most common orch-go failure type, and breaking it degrades all other primitives' value. But Route/Sequence/Throttle can also produce value without perfect Align (McEntire hierarchical 64%). The right framing: Align is the **validity condition** under which the other three produce correct outcomes.

**Answer to Investigation Question:**

Yes, 5 cases show Align broken with Route/Sequence/Throttle holding. But "holding" means "mechanically operating" not "producing coordination value." The substrate claim is partially falsified: Align is not a binary dependency. It's partially confirmed: breaking Align degrades all others' effectiveness. The more precise claim: **Align is the highest-leverage primitive and the validity condition for the other three, with proportional (not binary) impact on coordination effectiveness.** "Multiplier" or "validity condition" is more precise than "substrate" or "meta-primitive."

---

## Structured Uncertainty

**What's tested:**

- ✅ Case 1 exists — 5 independent cases from 3 data sources show Align broken with others holding mechanically
- ✅ McEntire hierarchical achieves 64% with partial Align — confirms proportional, not binary, relationship
- ✅ 80-trial messaging shows Align internal decomposition — task alignment defeats coordination alignment in 18/20 trials
- ✅ Case 2 exists — system spiral shows Align breaking first and cascading into other failures
- ✅ Case 3 exists — 80-trial no-coord shows Route broken with Align intact

**What's untested:**

- ⚠️ Whether there's a case where broken Align causes Route/Sequence/Throttle to mechanically fail (would support stronger substrate claim)
- ⚠️ Whether perfect Align + broken Route can still produce correct outcomes (would weaken multiplier model)
- ⚠️ Precise decomposition of Align sub-primitives (task, state, coordination) and their individual contributions

**What would change this:**

- Finding mechanical (not just functional) dependency of Route/Sequence/Throttle on Align would restore the substrate claim
- Finding a system where perfect Align compensates for missing Route would show Align is more than a multiplier
- Finding that Align decomposes into 3+ sub-primitives would suggest the framework should be 6 primitives, not 4

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Update "meta-primitive"/"substrate" language in model.md | implementation | Terminology refinement within existing model, no structural change |
| Add Align decomposition to open questions | implementation | Extends existing open question with new evidence |

### Recommended Approach

**Update model language from "substrate" to "multiplier/validity condition"**

**Why this approach:**
- 5 counterexamples show mechanical independence — "substrate" is empirically inaccurate
- "Multiplier" captures the proportional relationship McEntire demonstrates
- "Validity condition" captures why Align matters without overclaiming mechanical dependency

**Trade-offs accepted:**
- Less dramatic framing than "meta-primitive" — may reduce rhetorical impact
- More precise but harder to explain in one sentence

---

## References

**Files Examined:**
- `.kb/models/coordination/model.md` — substrate claim at line 75
- `.kb/investigations/2026-03-22-research-test-coordination-protocol-primitives-external-frameworks.md` — MAST mapping, McEntire data
- `.kb/threads/2026-03-22-coordination-protocol-primitives-route-sequence.md` — origin of substrate hypothesis
- `experiments/coordination-demo/redesign/results/20260310-174045/analysis.md` — 80-trial results
- `experiments/coordination-demo/redesign/results/20260310-174045/messaging/simple/trial-1/agent-a/prompt.md` — messaging condition prompt
- `experiments/coordination-demo/redesign/results/20260310-174045/context-share/simple/trial-1/agent-a/prompt.md` — context-share condition prompt
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` — system spiral failure data
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` — launchd failure data

**Related Artifacts:**
- **Probe:** `.kb/models/coordination/probes/2026-03-22-probe-falsify-align-as-substrate.md` — detailed case classification
- **Model:** `.kb/models/coordination/model.md` — parent model to be updated

---

## Investigation History

**2026-03-22:** Investigation started
- Initial question: Can Align be falsified as the substrate for Route/Sequence/Throttle?
- Context: Thread claimed Align is substrate, needed empirical falsification attempt

**2026-03-22:** Evidence collected and classified
- 15 cases from 4 data sources classified into 3 categories
- 5 Case 1 examples found (potential falsifiers)

**2026-03-22:** Investigation completed
- Status: Complete
- Key outcome: "Substrate" overclaims — Align is a multiplier/validity condition with proportional (not binary) impact on coordination effectiveness
