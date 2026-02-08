<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The Question/Gate boundary is crisp but dynamic - Questions have unknown option spaces while Gates have known options requiring authority to choose; the same issue transitions from Question to Gate as understanding develops.

**Evidence:** Analyzed 3 examples against model criteria; found that apparent fuzziness comes from lifecycle transitions, not definitional overlap.

**Knowledge:** The key distinction is **option space known vs unknown**, not surface linguistic form. Questions explore; Gates commit. An issue can transition between types as resolution progresses.

**Next:** Update decidability-graph.md to make lifecycle transition explicit; add transition detection criteria.

**Promote to Decision:** Actioned - decision exists (questions-as-first-class-entities)

---

# Investigation: Analyze Question Vs Gate Distinction

**Question:** Is the Question vs Gate distinction in the decidability graph model crisp or fuzzy? If fuzzy, what criteria can sharpen it?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** og-arch-analyze-question-vs-19jan-9457
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/models/decidability-graph.md` - extends the model with lifecycle transition clarity

---

## Findings

### Finding 1: The Model Defines Question and Gate by Resolution Shape, Not Content

**Evidence:** From `decidability-graph.md` lines 35-51:

| Node Type | Characteristics | Daemon Behavior | Resolution Shape |
|-----------|-----------------|-----------------|------------------|
| **Question** | Resolution uncertain, might branch/dissolve/reframe | Surface, don't resolve | Open (might fracture, collapse, or converge) |
| **Gate** | Judgment required, tradeoffs, irreversibility | Stop, accumulate options | Binary (decision made) |

**Source:** `.kb/models/decidability-graph.md:35-51`

**Significance:** The distinction is about **resolution shape**, not about the surface form of the question. "Should we adopt X?" could be either type depending on whether options are known (Gate) or need exploration (Question).

---

### Finding 2: Question Subtypes Already Encode Authority Levels

**Evidence:** From `decidability-graph.md` lines 105-113:

| Question Subtype | Example | Who Resolves | How |
|------------------|---------|--------------|-----|
| **Factual** | "How does X work?" | Daemon (via investigation) | Spawn agent, answer surfaces |
| **Judgment** | "Should we use X or Y?" | Orchestrator | Synthesize tradeoffs, decide |
| **Framing** | "Is X even the right question?" | Dylan | Reframe the problem space |

Critically, line 113: "A question's subtype may not be known at creation."

**Source:** `.kb/models/decidability-graph.md:105-113`

**Significance:** The Question taxonomy already handles authority escalation within Questions. The fuzziness isn't between Question and Gate, but between Question subtypes which can escalate (factual→judgment→framing). Gates are distinct - they require **commitment**, not just understanding.

---

### Finding 3: The Three Examples Reveal Lifecycle Transitions, Not Category Fuzziness

**Evidence:** Analyzing the three concrete examples:

**Example 1: "Should we adopt event sourcing?"**
- Initial state: Question (exploring whether ES is viable)
- After investigation: Options become clear [Full ES, Hybrid, No ES]
- Transition point: When options are enumerable → becomes Gate
- Final state: Gate (requires Dylan's commitment to architecture)

**Example 2: "How should we encode subtypes?"**
- Initial state: Question (exploring encoding options)
- After investigation: Options become clear [Labels, Separate field, Inferred]
- Transition point: Depends on reversibility of encoding choice
- Final state: Question (if easily changed) OR Gate (if schema-baked and irreversible)

**Example 3: "Is our caching strategy correct?"**
- Initial state: Factual Question (investigating behavior)
- After investigation: Either "yes, it's fine" (Question resolved) OR "needs changing"
- Transition point: Only becomes Gate if architectural change needed
- Final state: May never become Gate if answer is affirmative

**Source:** Analysis of examples against model criteria

**Significance:** The perceived fuzziness comes from **lifecycle transitions**, not definitional overlap. An issue can transition from Question to Gate as understanding develops. This is not a bug in the model - it's how resolution actually works.

---

### Finding 4: The "Questions Can Reveal They Were Actually Gates" Clause

**Evidence:** From `decidability-graph.md` line 45:
> "Might reveal they were actually Gates"

And from `2026-01-07-strategic-orchestrator-model.md` refinement section lines 106-107:
> "The hierarchy isn't about reasoning capability - workers can do any kind of reasoning IF they have the right context loaded. The irreducible orchestrator function is **deciding what context to load**."

**Source:** `.kb/models/decidability-graph.md:45`, `.kb/decisions/2026-01-07-strategic-orchestrator-model.md:106-107`

**Significance:** The model already acknowledges that Questions can "reveal" themselves to be Gates. Combined with the context-scoping insight, this suggests: **Questions are what you start with when you don't yet know if a commitment is required. Gates are what you have once you know a commitment is required.**

---

## Synthesis

**Key Insights:**

1. **The boundary is crisp but the category is dynamic** - Question vs Gate is not fuzzy; the same issue can transition between them as understanding develops. The distinction is phase-based, not content-based.

2. **Option space knowability is the key differentiator** - Questions have unknown/unexplored option spaces. Gates have known, enumerable options. When you can list the options clearly, you've transitioned from Question to Gate.

3. **Authority maps to commitment, not complexity** - Questions require understanding (can be resolved by appropriate authority level). Gates require commitment (irreversible binding decision). The complexity of reasoning is orthogonal to the Question/Gate distinction.

**Answer to Investigation Question:**

The Question/Gate distinction is **crisp**, not fuzzy, but it operates on a **lifecycle** rather than static categorization.

**Sharpened Criteria:**

| Property | Question | Gate |
|----------|----------|------|
| **Option space** | Unknown or unclear | Known (options enumerable) |
| **Resolution shape** | Open (might fracture, collapse, reframe) | Binary (choose or defer) |
| **What's needed** | Understanding | Commitment |
| **Reversibility** | N/A (not a commitment yet) | Low (irreversible or costly to reverse) |
| **Authority** | Depends on subtype (factual/judgment/framing) | Always Dylan (for irreversible) or Orchestrator (for scoped) |

**The Lifecycle Test:**

```
Ask: Can you enumerate the concrete options?
     ├─ NO → It's a Question (keep exploring)
     └─ YES → Ask: Does choosing require irreversible commitment?
              ├─ NO → It's a Question (options known, but choice is reversible)
              └─ YES → It's a Gate (accumulate options, await authority)
```

**Transition Pattern:**

```
Question: "Should we adopt event sourcing?"
         ↓ investigation surfaces options
Question: "Options are [A: Full ES, B: Hybrid, C: No ES]"
         ↓ options are now enumerable; commitment would be irreversible
Gate:     "Which option do we commit to?" (Dylan decides)
```

**When Apparent Fuzziness Occurs:**

The examples felt fuzzy because:
1. The same linguistic form ("Should we X?") can be either type
2. We encounter issues at different points in their lifecycle
3. We conflate "complex to answer" with "Gate" (they're orthogonal)

The model is not fuzzy - our detection of which phase an issue is in can be unclear.

---

## Structured Uncertainty

**What's tested:**

- ✅ Three concrete examples analyzed against model criteria - all fit the lifecycle model
- ✅ Model documentation reviewed - explicitly states Questions can "reveal" as Gates
- ✅ Authority mapping checked - Questions have subtype authority, Gates require commitment authority

**What's untested:**

- ⚠️ Whether beads can track Question→Gate transitions (would need implementation)
- ⚠️ Whether orchestrators/daemon can reliably detect transition points
- ⚠️ Whether the "enumerable options" heuristic holds across all domain types

**What would change this:**

- Finding a Question/Gate that doesn't fit the lifecycle model (truly both at once)
- Evidence that "option space knowability" doesn't predict category
- Cases where irreversibility doesn't map to Dylan authority

---

## Implementation Recommendations

**Purpose:** Operationalize the lifecycle model for Question/Gate classification.

### Recommended Approach ⭐

**Lifecycle-Aware Classification** - Treat Question/Gate as lifecycle phases rather than static types; add transition detection.

**Why this approach:**
- Matches how resolution actually works (understanding → commitment)
- Preserves crisp boundary while allowing dynamic reclassification
- Explains observed "fuzziness" as lifecycle confusion, not model weakness

**Trade-offs accepted:**
- Adds complexity to issue tracking (issues can change type)
- Requires transition detection logic
- May confuse users expecting static categories

**Implementation sequence:**
1. Update decidability-graph.md with explicit lifecycle transition model
2. Add transition criteria to beads question documentation
3. Consider `bd convert` command for Question→Gate transitions

### Alternative Approaches Considered

**Option B: Spectrum Model (Question ← → Gate)**
- **Pros:** Acknowledges gradient nature; simpler mental model
- **Cons:** Loses crisp authority mapping; makes automation harder
- **When to use instead:** If transition detection proves impractical

**Option C: Question-With-Gate-Potential Flag**
- **Pros:** Static categorization preserved; flag indicates possible escalation
- **Cons:** Still doesn't capture actual transitions; flag maintenance burden
- **When to use instead:** If lifecycle tracking is too complex for beads

**Rationale for recommendation:** The lifecycle model matches the empirical observation (issues transitioning) while preserving the crisp boundary needed for authority routing.

---

### Implementation Details

**What to implement first:**
- Add "Lifecycle Model" section to decidability-graph.md
- Document transition criteria in beads question guide
- Add example of Question→Gate transition to model

**Things to watch out for:**
- ⚠️ Don't conflate Question subtypes (factual/judgment/framing) with Question→Gate transition
- ⚠️ "Reversibility" is context-dependent - what's reversible early becomes irreversible late
- ⚠️ Some Questions never become Gates (resolved without needing commitment)

**Areas needing further investigation:**
- Can beads track type transitions? (bd update --type?)
- Should dashboard show "potential Gate" status for Questions with enumerable options?
- How should daemon handle Questions approaching Gate transition?

**Success criteria:**
- ✅ Model documentation clearly explains lifecycle transition
- ✅ Users can apply enumerable-options test to classify
- ✅ Authority routing works consistently based on classification

---

## References

**Files Examined:**
- `.kb/models/decidability-graph.md` - Core model with Question/Gate definitions
- `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Authority levels and context-scoping refinement
- `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question entity implementation

**Commands Run:**
```bash
# Create investigation
kb create investigation analyze-question-vs-gate-distinction

# Report to orchestrator
bd comment orch-go-2yzjl "Phase: Planning - Analyzing Question vs Gate distinction"
bd comment orch-go-2yzjl "investigation_path: ..."
```

**Related Artifacts:**
- **Model:** `.kb/models/decidability-graph.md` - This investigation patches/extends this model
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question implementation context
- **Investigation:** `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md` - Parallel investigation on subtypes

---

## Investigation History

**2026-01-19 10:40:** Investigation started
- Initial question: Is the Question/Gate boundary crisp or fuzzy?
- Context: Spawned from decidability graph model dogfooding

**2026-01-19 10:55:** Key finding identified
- The distinction is crisp but lifecycle-based
- Apparent fuzziness is phase confusion, not category overlap

**2026-01-19 11:05:** Investigation completed
- Status: Complete
- Key outcome: Question/Gate is crisp lifecycle transition based on option space knowability
