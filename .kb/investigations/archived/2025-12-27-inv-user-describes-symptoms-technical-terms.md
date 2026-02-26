<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** User symptom descriptions contain two layers: observation ("it's slow") and hypothesis ("hydration is slow"). Orchestrators must treat these differently per Evidence Hierarchy.

**Evidence:** The task describes a user saying "hydration is slow" when the real issue was backend latency. Evidence Hierarchy principle already states "Code is truth. Artifacts are hypotheses." - user hypotheses are secondary evidence.

**Knowledge:** User observations are PRIMARY evidence (they saw slowness); user technical hypotheses are SECONDARY evidence (claims to verify). Existing Premise Before Solution principle applies - but was designed for strategic questions, not symptom interpretation.

**Next:** Add symptom parsing guidance to orchestrator skill; consider adding premise-validation trigger for technical framings; record as constraint or pattern.

---

# Investigation: User Describes Symptoms with Technical Terms

**Question:** How should orchestrators handle the gap between user symptom descriptions and their technical framing? When should we validate premises vs trust framing?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None - recommendations ready for orchestrator review
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: User Symptom Reports Contain Two Distinct Layers

**Evidence:** The task describes a scenario:
> User describes symptoms using technical terms they're not sure about (e.g., 'hydration is slow'). Orchestrator takes the technical framing literally and spawns work on wrong problem. Real issue was backend latency, not hydration.

This reveals two layers in user symptom reports:
1. **Observation (Primary):** "Something is slow" - what the user actually experienced
2. **Hypothesis (Secondary):** "Hydration is slow" - user's guess at the cause

The orchestrator treated the hypothesis as fact, spawning work on "hydration" when the observation was "slowness" - a category error.

**Source:** Task description in SPAWN_CONTEXT.md

**Significance:** User symptom reports are not pure observations - they often contain embedded hypotheses that sound like facts. Treating hypotheses as facts leads to wrong-direction work.

---

### Finding 2: Evidence Hierarchy Already Covers This Pattern

**Evidence:** From `~/.kb/principles.md:129-141`:

```markdown
### Evidence Hierarchy

Code is truth. Artifacts are hypotheses.

| Source Type   | Examples                                    | Treatment            |
| ------------- | ------------------------------------------- | -------------------- |
| **Primary**   | Actual code, test output, observed behavior | This IS the evidence |
| **Secondary** | Workspaces, investigations, decisions       | Claims to verify     |

**Why:** LLMs can hallucinate or trust stale artifacts. Always verify claims against primary sources.

**The test:** Did the agent grep/search before claiming something exists or doesn't exist?
```

This principle already distinguishes primary vs secondary evidence. The application to user symptom reports:

| User Statement Type | Evidence Category | Treatment |
|---------------------|-------------------|-----------|
| "I saw X" (observation) | Primary | Trust it - user experienced this |
| "X is caused by Y" (hypothesis) | Secondary | Verify before acting |
| "The hydration is slow" (embedded hypothesis) | Secondary | Extract observation, verify hypothesis |

**Source:** `~/.kb/principles.md:129-141`

**Significance:** No new principle needed - Evidence Hierarchy applies. What's missing is guidance on PARSING user statements to separate observations from hypotheses.

---

### Finding 3: Premise Before Solution Principle Exists But Is Strategic-Scoped

**Evidence:** From `~/.kb/principles.md:311-341`:

```markdown
### Premise Before Solution

"How do we X?" presupposes X is correct. For strategic questions, validate the premise first.

**The test:** "Am I assuming the direction, or have I tested it?"

**The sequence:** SHOULD → HOW → EXECUTE
1. **SHOULD** (Premise): Is this direction correct? Is the problem real?
2. **HOW** (Design): Given it's right, how do we get there?
3. **EXECUTE** (Implementation): Do the work.
```

This principle was designed for STRATEGIC questions ("How do we evolve to X?"). But it applies equally to SYMPTOM interpretation:

| Strategic Context | Symptom Context |
|-------------------|-----------------|
| "How do we evolve skills?" assumes skills should evolve | "Hydration is slow" assumes slowness is in hydration |
| Validate: Should skills evolve? | Validate: Is hydration actually the slow part? |
| Test: Ask "should we" before "how" | Test: Separate observation from hypothesis |

**Source:** 
- `~/.kb/principles.md:311-341` (Premise Before Solution)
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` (origin investigation)

**Significance:** Premise Before Solution principle extends naturally to symptom parsing. The same pattern applies: user says "fix hydration" (assumes hydration is the problem) → should first verify "is hydration the slow part?"

---

### Finding 4: Issue-Creation Skill Has Hypothesis Testing Phase

**Evidence:** From issue-creation skill (`~/.opencode/skills/issue-creation/SKILL.md:80-96`):

```markdown
### Phase 2: Investigate Root Cause (10-20 min)

**Goal:** Understand WHY, not just WHERE.

1. **Trace the code path**
   - Start from symptom manifestation
   - Follow the logic backward

2. **Form hypothesis**
   - What could cause this behavior?
   - What evidence would confirm/deny?

3. **Test hypothesis**
   - Run specific tests, check logs, reproduce with variations
```

The issue-creation skill DOES have hypothesis testing - it forms its OWN hypothesis and tests it. But it doesn't explicitly challenge the user's INCOMING hypothesis.

**Source:** `/Users/dylanconlin/.opencode/skills/issue-creation/SKILL.md:80-96`

**Significance:** Issue-creation has the right instinct (test hypotheses before creating issues) but applies it to agent-formed hypotheses, not user-provided framings. The gap is in the INTAKE step.

---

### Finding 5: The Gap Is in Symptom Parsing at Orchestrator Level

**Evidence:** The orchestrator skill has extensive guidance on:
- When to delegate (ABSOLUTE DELEGATION RULE)
- Which skill to use (Skill Selection Guide)
- How to spawn (Spawning Checklist)

But it lacks guidance on:
- How to parse user symptom reports
- When user technical framing is hypothesis vs observation
- How to rephrase user requests to separate observation from diagnosis

The current Bug Triage section (`orchestrator SKILL.md`) says:
```markdown
**First: Is the cause known?**

├── NO (symptom only) → issue-creation
└── YES (cause clear) → systematic-debugging
```

But this assumes the orchestrator can distinguish "symptom only" from "cause embedded in symptom phrasing." The hydration example shows this is exactly where the failure occurs.

**Source:** Analysis of orchestrator skill vs described failure mode

**Significance:** The fix belongs at the orchestrator level, not the worker skill level. Workers receive already-parsed requests; orchestrators interpret raw user statements.

---

## Synthesis

**Key Insights:**

1. **User statements have two layers** - Observations (what they saw) are primary evidence; hypotheses (what they think caused it) are secondary. Most symptom reports mix both: "hydration is slow" = "something is slow" (observation) + "it's the hydration" (hypothesis).

2. **Evidence Hierarchy already provides the framework** - We just need to apply it to user statements. User observations → primary → trust. User technical hypotheses → secondary → verify.

3. **Premise Before Solution extends to symptoms** - Just as "how do we X" assumes X is correct direction, "fix Y problem" assumes Y is the problem. Same validation discipline applies.

4. **The gap is at orchestrator intake, not worker execution** - Workers receive parsed tasks. The orchestrator interprets raw user statements. That's where the parsing guidance is needed.

5. **Issue-creation skill is downstream** - By the time issue-creation runs, the orchestrator has already decided the framing. The skill refines within that frame but doesn't challenge the frame itself.

**Answer to Investigation Question:**

**How should orchestrators handle the gap between user symptom descriptions and their technical framing?**

By applying Evidence Hierarchy to user statements:

1. **Parse the statement** - Identify observation component vs hypothesis component
   - "Hydration is slow" → Observation: "slow"; Hypothesis: "hydration"
   - "Login fails intermittently" → Observation: "intermittent failure"; Hypothesis: "login"

2. **Trust observations, verify hypotheses** - The user saw slowness (primary evidence). The attribution to hydration is a hypothesis (secondary evidence).

3. **Rephrase spawn prompts** - Instead of "fix hydration slowness", spawn "investigate slowness in [area] - user hypothesizes hydration"

4. **Let workers validate** - Issue-creation or systematic-debugging should confirm the hypothesis before acting on it

**When should we validate premises vs trust framing?**

| User Statement Type | Action |
|---------------------|--------|
| Pure observation ("login is slow") | Trust and investigate |
| Pure diagnosis ("the JWT parser is broken") | Verify before acting |
| Mixed ("hydration is slow") | Separate and verify hypothesis |
| Technical term user admits uncertainty about | Always verify |

**Red flag signals** that indicate hypothesis-in-symptom:
- User uses technical jargon for their level ("hydration", "race condition", "memory leak")
- User sounds uncertain ("I think it's...", "seems like...")
- Diagnosis is highly specific from vague symptom
- User is guessing at cause (not reporting observed error message)

---

## Structured Uncertainty

**What's tested:**

- ✅ Evidence Hierarchy principle exists and covers primary/secondary distinction (read principles.md)
- ✅ Premise Before Solution principle exists for strategic questions (read principles.md)
- ✅ Issue-creation skill has hypothesis testing phase (read skill)
- ✅ Orchestrator skill lacks symptom parsing guidance (searched orchestrator skill)

**What's untested:**

- ⚠️ Whether orchestrators will reliably apply this parsing (no real-world test)
- ⚠️ Whether the red flag signals reliably identify hypotheses (pattern-matched from single example)
- ⚠️ Whether this is common enough to warrant skill updates (single incident reported)

**What would change this:**

- More examples of user-hypothesis-as-symptom failures would strengthen the case
- Counter-examples where user technical framing was correct would refine the heuristics
- Pilot implementation showing reduced wrong-direction work would validate approach

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add Symptom Parsing Guidance to Orchestrator Skill** - Explicitly guide orchestrators to separate observations from hypotheses when interpreting user symptom reports.

**Why this approach:**
- Addresses the gap at the right level (orchestrator intake, not worker execution)
- Applies existing principles (Evidence Hierarchy, Premise Before Solution) - no new concepts
- Low implementation cost (documentation update to orchestrator skill)

**Trade-offs accepted:**
- Adds friction to bug triage (brief parsing step before spawning)
- May slow response to legitimate diagnoses (users who actually know the cause)

**Implementation sequence:**

1. **Update Bug Triage section** in orchestrator skill with symptom parsing step
2. **Add red flag signals** that trigger hypothesis verification
3. **Record as constraint** in kn for immediate availability

### Alternative Approaches Considered

**Option B: Update issue-creation skill to challenge framing**
- **Pros:** Workers would validate hypotheses during investigation
- **Cons:** By then the framing is set; orchestrator has already decided direction
- **When to use instead:** If orchestrator updates prove insufficient

**Option C: New "symptom-validation" phase for design-session**
- **Pros:** Would catch hypothesis-as-symptom in design discussions
- **Cons:** Design-session is for scoping features, not bug triage; wrong skill
- **When to use instead:** If pattern appears in feature requests, not just bugs

**Rationale for recommendation:** The failure mode occurs at orchestrator level when parsing user statements. Fix should be at that level.

---

### Implementation Details

**What to implement first:**

1. Add to orchestrator skill Bug Triage section:

```markdown
### Symptom Parsing (Before Skill Selection)

**User symptom reports contain two layers:**
- **Observation (Primary):** What they actually saw ("it's slow", "it fails")
- **Hypothesis (Secondary):** Their guess at cause ("hydration", "JWT parser")

**Parsing rule:** Trust observations, verify hypotheses.

| User Says | Observation | Hypothesis | Action |
|-----------|-------------|------------|--------|
| "Hydration is slow" | Slow | Hydration | Verify hydration is actually slow part |
| "Login fails intermittently" | Intermittent failure | Login | Verify it's login, not downstream |
| "I think the cache is stale" | [Need to ask] | Cache | Ask what they observed, then verify cache |

**Red flags** (hypothesis embedded in symptom):
- Technical jargon user seems uncertain about
- Highly specific diagnosis from vague symptom
- User phrases with "I think", "seems like", "might be"
- Mismatch between user's expertise and term specificity

**Reframe spawn prompts:**
- BAD: "Fix hydration slowness"
- GOOD: "Investigate slowness - user hypothesizes hydration, verify before acting"
```

2. Record constraint:

```bash
kn constrain "Separate user observations from hypotheses before spawning" --reason "Evidence Hierarchy: observations are primary evidence, hypotheses are secondary claims to verify"
```

**Things to watch out for:**

- ⚠️ Don't over-apply: Users who actually know the cause shouldn't have their expertise questioned
- ⚠️ Don't slow down: Parsing should be quick mental check, not full investigation
- ⚠️ Don't ignore user context: A developer saying "memory leak" is different from a user guessing

**Areas needing further investigation:**

- How often does this pattern actually occur? (need more examples)
- What's the cost of verifying correct user diagnoses? (false positive rate)
- Should workers explicitly report "user hypothesis validated/invalidated"?

**Success criteria:**

- ✅ Next user symptom with embedded hypothesis triggers parsing
- ✅ Spawn prompts separate observation from hypothesis
- ✅ Reduced wrong-direction work from accepted-hypothesis spawns

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Evidence Hierarchy, Premise Before Solution principles
- `~/.opencode/skills/issue-creation/SKILL.md` - Current hypothesis testing phase
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Bug triage section (lacks parsing)
- `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` - Related principle origin

**Commands Run:**
```bash
# Get context on related topics
kb context "premise before solution"
kb context "evidence hierarchy"
kb context "issue-creation"

# Check for related issues
bd list | grep -i "premise\|symptom\|hypothesis\|framing"
```

**Related Artifacts:**
- **Principle:** Evidence Hierarchy (`~/.kb/principles.md:129-141`)
- **Principle:** Premise Before Solution (`~/.kb/principles.md:311-341`)
- **Investigation:** Strategic Question-Asking Sequence (`.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md`)
- **Issue:** orch-go-khf5 (related premise validation investigation)

---

## Investigation History

**2025-12-27 ~08:00:** Investigation started
- Initial question: How to handle gap between user symptoms and technical framing?
- Context: Task describes "hydration is slow" when real issue was backend latency

**2025-12-27 ~08:30:** Context gathering complete
- Found Evidence Hierarchy already covers primary/secondary distinction
- Found Premise Before Solution exists but scoped to strategic questions
- Identified gap is at orchestrator intake level

**2025-12-27 ~09:00:** Investigation completed
- Status: Complete
- Key outcome: Apply Evidence Hierarchy to user statements; parse observations from hypotheses; update orchestrator skill guidance
