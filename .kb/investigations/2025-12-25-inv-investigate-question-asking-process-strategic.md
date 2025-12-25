<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Strategic questions require a SHOULD-HOW-EXECUTE sequence; "how do we" assumes premise validity that often hasn't been tested.

**Evidence:** Epic orch-go-erdw was created from "how do we" question without first validating premise; architect review found the premise (skills should absorb orchestration) was wrong.

**Knowledge:** "How do we X?" presupposes X is correct direction; Reflection Before Action principle applies at question formation, not just pattern surfacing.

**Next:** Add premise validation phase to design-session skill; add question sequencing guidance to orchestrator skill; consider principle promotion.

**Confidence:** High (85%) - Based on single instance; would need more examples to confirm universal applicability.

---

# Investigation: Strategic Question-Asking Sequence

**Question:** What's the right sequence for strategic questions? Should premise validation come before solution design?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None - recommendations ready for implementation
**Status:** Complete
**Confidence:** High (85%)

**Extracted-From:** Observed pattern from orch-go-erdw epic creation/pause
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: "How" Questions Presuppose Valid Premise

**Evidence:** The original question asked:
> "How do we evolve skills to be where true value resides?"

This question assumes:
1. Skills are NOT currently where value resides
2. Skills SHOULD be where value resides
3. Evolution toward this is the correct direction

None of these were tested before spawning the design-session that created epic orch-go-erdw.

The architect review later found all three assumptions were wrong:
- Skills ALREADY contain their domain value (procedures, workflows, constraints)
- The "leaked value" is actually orchestration infrastructure that correctly belongs in spawn
- Current separation follows Compose Over Monolith and Evolve by Distinction principles

**Source:** 
- `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md` (original "how" question)
- `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md` (architect review finding premise wrong)

**Significance:** The work product (epic with 5 children) was created from an unvalidated premise. The epic had to be paused and marked blocked because the foundational assumption was incorrect.

---

### Finding 2: Reflection Before Action Applies at Question Formation

**Evidence:** The existing Reflection Before Action principle states:

> "When patterns surface, pause before acting. Build the process that surfaces patterns, not the solution to this instance."

This principle operates at the ACTION level - when you see 46 unacted recommendations, don't manually act on them, build the process (`kb reflect`).

But the same logic applies at the QUESTION level:
- "How do we fix X?" → presupposes X is broken → should first ask "Is X actually broken?"
- "How do we evolve to Y?" → presupposes Y is better → should first ask "Is Y actually better?"
- "How do we implement Z?" → presupposes Z is the right solution → should first ask "Is Z the right solution?"

The constraint kn-c12998 captured this:
> "Ask 'should we' before 'how do we' for strategic direction changes"

But this is actually a specific application of the deeper principle: test premises before executing on them.

**Source:**
- `~/.kb/decisions/2025-12-21-reflection-before-action.md`
- kn-c12998 (constraint created from this incident)

**Significance:** The principle needs to be understood as operating at multiple levels: pattern level (don't act on instances), question level (don't design solutions without validating problem), and execution level (don't code without validating design).

---

### Finding 3: Design-Session Skill Lacks Premise Validation Phase

**Evidence:** Examined the design-session skill workflow:

```
Phase 1: Context Gathering (Autonomous)
    ↓
Phase 2: Design Synthesis (Semi-Autonomous)
    ↓
Phase 3: Output Creation (Autonomous)
    ↓
One of: Epic | Investigation | Decision
```

Phase 1 gathers CONTEXT (existing knowledge, related issues, codebase state).
Phase 2 SYNTHESIZES scope (boundaries, priorities, constraints).
Phase 3 CREATES artifacts (epic, investigation, decision).

Missing: **Premise Validation** - explicitly testing whether the direction implied by the question is correct.

The decision tree in Phase 2 asks:
- "Can we list the specific tasks needed?"
- "Do we understand all the tasks well enough to implement?"
- "Is this blocked by a strategic choice?"

It does NOT ask:
- "Is the direction implied by this request correct?"
- "Have we validated the assumption that X should happen?"
- "Should we do this at all?"

**Source:** `/Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/SKILL.md:38-178`

**Significance:** The skill proceeds from "I want to add X" directly to "how do we scope X" without asking "should we add X at all?" For vague feature requests, this may be fine. For strategic direction changes, it's a gap.

---

### Finding 4: Question Type Determines Required Sequence

**Evidence:** Not all questions require premise validation. Analyzing different question types:

| Question Type | Example | Premise Validation Needed? |
|--------------|---------|---------------------------|
| **Tactical how** | "How do I run the tests?" | No - premise is implicit/trivial |
| **Feature scoping** | "How should we implement auth?" | Maybe - depends on whether auth is questioned |
| **Strategic direction** | "How do we evolve skills to X?" | YES - assumes X is correct direction |
| **Architectural choice** | "Should we use Redis or Postgres?" | Already in "should" form |

The distinguishing factor is **whether the question assumes a direction change or strategic choice**.

Red flag words that indicate premise-testing is needed:
- "evolve to", "migrate to", "transition to" (assumes destination is correct)
- "fix the", "solve the" (assumes problem diagnosis is correct)
- "implement the" (assumes solution is correct)
- "because X" (assumes X is true)

**Source:** Analysis of question patterns from this investigation

**Significance:** We can identify questions that need premise validation by their structure. This makes the guidance actionable.

---

### Finding 5: The Correct Sequence Is SHOULD → HOW → EXECUTE

**Evidence:** Reconstructing what should have happened:

**Actual sequence:**
1. Question: "How do we evolve skills to be where true value resides?"
2. Design-session spawned → Created epic orch-go-erdw with 5 children
3. Architect spawned (later) → Found premise was wrong
4. Epic paused/blocked

**Correct sequence:**
1. Question: "How do we evolve skills to be where true value resides?"
2. **Recognize strategic direction in question** → Trigger premise validation
3. Reframe to: "SHOULD we evolve skills to be where true value resides? Is the current architecture wrong?"
4. Architect investigates (or investigation) → Finds current architecture is correct
5. No epic needed → Question answered, work avoided

The SHOULD question would have prevented the wasted epic creation.

**Source:** Timeline reconstruction from beads issue orch-go-erdw and related investigations

**Significance:** This is the generalizable process. When strategic questions arise, convert "how do we X" to "should we X" first.

---

## Synthesis

**Key Insights:**

1. **"How" questions carry hidden assumptions** - When someone asks "how do we do X", they've implicitly decided X is the right thing to do. For tactical questions this is fine. For strategic direction changes, it's a trap.

2. **Reflection Before Action extends to question formation** - The principle already says "pause before acting on patterns." The same discipline applies to questions: pause before designing solutions. Test the premise first.

3. **Skill gaps propagate to work products** - Design-session produced an epic because its workflow doesn't include premise testing. The skill did what it was designed to do; it just wasn't designed for strategic direction questions.

4. **Signal words can trigger premise testing** - Words like "evolve to", "migrate to", "fix the" signal questions that assume direction. When detected, the question should be reframed to "should" form before proceeding.

**Answer to Investigation Question:**

**The right sequence for strategic questions is: SHOULD → HOW → EXECUTE**

1. **SHOULD** (Premise Validation): Is this direction correct? Is the problem real? Is the solution appropriate?
2. **HOW** (Design/Scoping): Given it's the right direction, how do we get there? What's the scope? What are the tasks?
3. **EXECUTE** (Implementation): Do the work.

The incident with orch-go-erdw happened because step 1 was skipped. "How do we evolve skills" went directly to step 2 (design-session created epic).

**Relationship to existing principles:**

| Principle | How It Relates |
|-----------|---------------|
| **Reflection Before Action** | SHOULD-HOW-EXECUTE is the question-level application of this principle |
| **Evolve by Distinction** | The distinction here is "premise" vs "solution" - conflating them causes wrong work |
| **Evidence Hierarchy** | Premise validation is testing claims against primary evidence before acting |

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

This investigation is based on a single clear example where the pattern failure was caught. The reconstruction of what-should-have-happened is logical and consistent with existing principles. However, one example isn't definitive proof of a universal pattern.

**What's certain:**

- ✅ Epic orch-go-erdw was created from unvalidated premise (documented in bd show)
- ✅ Architect review found premise was wrong (documented in investigation)
- ✅ The constraint kn-c12998 was created as a result (kn get output shows this)
- ✅ Design-session skill lacks explicit premise validation phase (skill source confirms)
- ✅ Reflection Before Action principle exists but applies at pattern level (decision document)

**What's uncertain:**

- ⚠️ Whether this pattern recurs frequently or was a one-off (need more examples)
- ⚠️ Whether premise validation should be a separate phase or integrated into context gathering
- ⚠️ What the right trigger is for premise validation (all strategic questions? only "direction change" questions?)
- ⚠️ Whether this is worth promoting to principles.md or is sufficient as a constraint

**What would increase confidence to Very High (95%):**

- Additional examples of premise failure from historical investigations
- Pilot implementation of premise validation phase in design-session
- User validation that SHOULD-HOW-EXECUTE matches mental model

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Multi-Level Implementation** - Add guidance at both skill and orchestrator levels, plus consider principle promotion.

**Why this approach:**
- Design-session skill gaps need fixing (catches strategic questions at design time)
- Orchestrator skill needs question-sequencing guidance (catches strategic questions at delegation time)
- Pattern may be universal enough for principles.md (if it has teeth across projects)

**Trade-offs accepted:**
- Adds friction to design-session workflow (but prevents wrong-direction work)
- More questions asked before epic creation (but higher confidence in direction)

**Implementation sequence:**
1. Add premise validation phase to design-session skill (catches at design time)
2. Add question sequencing guidance to orchestrator skill (catches at delegation time)
3. Evaluate for principle promotion after more examples confirm pattern

### Alternative Approaches Considered

**Option B: Constraint only (no skill changes)**
- **Pros:** Minimal change; kn-c12998 already exists
- **Cons:** Constraints are easily forgotten under cognitive load; Gate Over Remind says build gates
- **When to use instead:** If skill changes prove too complex or the pattern doesn't recur

**Option C: Separate "premise-validation" skill**
- **Pros:** Clear separation of concerns; explicit phase
- **Cons:** Adds skill proliferation; may be overkill for what's a phase, not a full skill
- **When to use instead:** If premise validation is complex enough to warrant full skill treatment

**Rationale for recommendation:** Design-session already has phases; adding one more is lower friction than new skill. Orchestrator guidance catches questions before they reach design-session.

---

### Implementation Details

**What to implement first:**
- Add to design-session Phase 1 (Context Gathering) a decision point:
  ```
  Does this question assume a strategic direction change?
  ├── YES → Add premise validation before Phase 2
  │   └── Reframe "how do we X" to "should we X"
  │   └── If answer is "no", complete with recommendation, no epic
  └── NO → Proceed to Phase 2 (Design Synthesis)
  ```

**Things to watch out for:**
- ⚠️ Not all "how" questions need premise validation - only strategic direction changes
- ⚠️ Premise validation should be QUICK - not a full investigation, just a sanity check
- ⚠️ False positives (requiring validation when not needed) waste time but aren't dangerous
- ⚠️ False negatives (skipping validation when needed) risk wrong-direction work

**Areas needing further investigation:**
- What heuristics reliably identify "strategic direction" questions?
- Should premise validation be synchronous (same session) or spawn an architect?
- How long should premise validation take before escalating?

**Success criteria:**
- ✅ Next "how do we evolve to X" question triggers premise check
- ✅ Epic creation only happens after SHOULD is answered
- ✅ Time saved from avoided wrong-direction work exceeds time spent on validation

---

## Test Performed

**Test:** Examined the timeline of orch-go-erdw creation and pause to validate the premise-skip hypothesis.

**Method:**
1. Read the original "how do we" investigation (2025-12-25-inv-epic-question-how-do-we.md)
2. Read the architect "should we" investigation (2025-12-25-design-should-we-evolve-skills-where.md)
3. Checked bd show orch-go-erdw for epic status and reason for pause
4. Verified constraint kn-c12998 was created as a result

**Result:** 
- Original investigation concluded with "Epic with children" (design-session Phase 2 → Phase 3)
- Architect investigation found "Pause epic. Current architecture is sound."
- Epic is blocked with description explaining premise was wrong
- Constraint explicitly states "Ask 'should we' before 'how do we'"

**Conclusion:** The hypothesis is confirmed for this instance. The work would have been avoided if "should we" was asked first.

---

## Self-Review

- [x] Real test performed (timeline reconstruction against artifacts)
- [x] Conclusion from evidence (based on documented artifacts, not speculation)
- [x] Question answered (SHOULD-HOW-EXECUTE sequence defined)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## Discovered Work

| Type | Description | Action |
|------|-------------|--------|
| Enhancement | Add premise validation phase to design-session skill | Recommend to orchestrator |
| Enhancement | Add question sequencing guidance to orchestrator skill | Recommend to orchestrator |
| Decision | Evaluate SHOULD-HOW-EXECUTE for principle promotion | After more examples confirm pattern |

**Discovered issues tracked:** Will report to orchestrator for prioritization.

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-25-inv-epic-question-how-do-we.md` - Original "how" question investigation
- `.kb/investigations/2025-12-25-design-should-we-evolve-skills-where.md` - Architect "should" review
- `~/.kb/principles.md` - Existing principles including Reflection Before Action
- `~/.kb/decisions/2025-12-21-reflection-before-action.md` - Decision record for principle
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/design-session/SKILL.md` - Design-session skill source

**Commands Run:**
```bash
# Get constraint details
kn get kn-c12998

# Show epic status
bd show orch-go-erdw

# Search for related patterns
kb context "investigate"
```

**Related Artifacts:**
- **Epic:** orch-go-erdw - The epic created from unvalidated premise (now blocked)
- **Constraint:** kn-c12998 - "Ask 'should we' before 'how do we' for strategic direction changes"
- **Principle:** Reflection Before Action - The higher-level principle this pattern relates to

---

## Investigation History

**2025-12-25 ~12:00:** Investigation started
- Initial question: What's the right sequence for strategic questions?
- Context: Observed pattern where epic was created before validating premise

**2025-12-25 ~12:30:** Evidence gathering complete
- Read all relevant artifacts (investigations, decisions, principles)
- Identified that design-session skill lacks premise validation phase
- Recognized Reflection Before Action applies at multiple levels

**2025-12-25 ~13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: SHOULD-HOW-EXECUTE sequence; recommend skill and guidance updates
