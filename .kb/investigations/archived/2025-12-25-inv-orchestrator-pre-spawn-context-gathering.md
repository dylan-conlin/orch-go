<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator context-gathering is legitimately distinct from deep investigation - the skill conflates them, creating confusion about what's allowed.

**Evidence:** The skill simultaneously says "never investigate" (line 279) while requiring orchestrators to run `kb context`, read results, summarize findings, and provide "pointers to prior artifacts" before spawning (lines 905-944).

**Knowledge:** There are three categories of context work: (1) routing queries (always allowed), (2) context enrichment for spawn prompts (allowed), (3) deep investigation (always delegate). The skill needs explicit boundaries.

**Next:** Update orchestrator skill with "Pre-Spawn Context Gathering" section that explicitly defines allowed activities and time-boxing (< 5 min).

---

# Investigation: Orchestrator Pre-Spawn Context Gathering

**Question:** What context-gathering is appropriate for orchestrators before spawning vs what should be delegated as an investigation?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent (spawned for design-session)
**Phase:** Complete
**Next Step:** Propose skill update via this investigation's recommendations
**Status:** Complete
**Confidence:** High (85%) - Analysis based on skill text alone; would benefit from observing actual orchestrator behavior patterns

---

## Findings

### Finding 1: The Skill Contains Contradictory Guidance

**Evidence:** 

The ABSOLUTE DELEGATION RULE (line 279) states:
> "ANY investigation (even 'quick' ones)" [should be delegated]

But the Pre-Spawn Knowledge Check (lines 905-928) requires:
```
# Get all knowledge about task (decisions AND investigations)
kb context "<task keywords>"
```

And:
> "Include key findings (2-3 sentence summary + link)"

**Source:** ~/.claude/skills/meta/orchestrator/SKILL.md lines 261-368, 905-944

**Significance:** An orchestrator following the skill literally would be paralyzed: they're told never to investigate, but also told they must read and summarize prior investigations before spawning. This creates cognitive dissonance that likely leads to inconsistent behavior.

---

### Finding 2: Three Distinct Context-Gathering Activities Exist

**Evidence:** Analysis of the skill reveals three categories of context work:

1. **Routing Queries** (instant, mechanical)
   - `orch status` - check what's running
   - `bd ready` - see ready work
   - `bd list | grep "keyword"` - find relevant issues
   - Purpose: Answer "what should I spawn next?"
   - Time: < 30 seconds

2. **Context Enrichment** (brief, targeted)
   - `kb context "<topic>"` - find prior knowledge
   - Read SYNTHESIS.md from completed agents
   - Scan workspace status files
   - Purpose: Write effective spawn prompts
   - Time: 1-5 minutes

3. **Deep Investigation** (sustained, exploratory)
   - Reading code to understand behavior
   - Testing hypotheses about bugs
   - Exploring multiple files for patterns
   - Purpose: Answer "how does X work?"
   - Time: 15+ minutes

**Source:** Inferred from skill patterns at lines 293 ("Read completed artifacts to synthesize"), 510-514 (Quick commands), 905-928 (Pre-spawn check)

**Significance:** The skill only explicitly forbids category 3 but uses language that accidentally prohibits category 2. The "never investigate" rule needs scoping.

---

### Finding 3: Time is the Key Distinguishing Factor

**Evidence:** The skill's examples of allowed vs forbidden work cluster by time:

**Allowed (< 5 min):**
- "Read completed artifacts to synthesize" (line 293)
- "Review `triage:review` issues" (line 293)  
- Run `kb context` and include results in spawn (line 910)
- Check status, monitor agents (lines 402-412)

**Forbidden (15+ min):**
- "ANY investigation" (line 279)
- Reading code to understand something (line 299)
- Debugging (line 281)
- Implementation thinking (line 280)

**Source:** Time estimates inferred from skill descriptions of "quick" vs sustained work

**Significance:** The implicit boundary is time-based: orchestrators can do brief, targeted context gathering (minutes) but not sustained exploration (tens of minutes). Making this explicit would resolve the confusion.

---

### Finding 4: The "Test" on Line 299 is Too Broad

**Evidence:** The skill provides this test:
> "If you're about to read code to understand something → STOP. That's an investigation. Spawn it."

But this conflicts with legitimate orchestrator needs:
- Reading a SYNTHESIS.md requires reading to understand
- Reading `kb context` output requires understanding prior findings
- Reviewing an agent's commits requires reading code

**Source:** SKILL.md line 299 vs lines 293, 910-923

**Significance:** The test needs refinement. "Reading to understand the codebase's behavior" is investigation. "Reading to know what to tell an agent" is context enrichment. These are different activities with the same surface action (reading).

---

## Synthesis

**Key Insights:**

1. **The prohibition is against exploration, not reading** - Orchestrators are forbidden from entering exploratory mode where they follow curiosity across files. They ARE allowed targeted reads with a specific spawn-prompt purpose.

2. **Purpose distinguishes allowed vs forbidden** - The question isn't "am I reading files?" but "am I reading to write a spawn prompt or reading to answer a codebase question?" The first is allowed, the second should be spawned.

3. **Time-boxing is the practical boundary** - If context-gathering takes more than ~5 minutes, it has likely crossed into investigation territory. This gives orchestrators a concrete signal.

**Answer to Investigation Question:**

Orchestrators should do **spawn context gathering** (1-5 minutes of targeted reading to write effective spawn prompts) but delegate **deep investigation** (sustained exploration to answer codebase questions). The distinguishing factors are:

| Dimension | Spawn Context (Allowed) | Deep Investigation (Delegate) |
|-----------|------------------------|-------------------------------|
| **Time** | < 5 minutes | > 15 minutes |
| **Purpose** | Write a better spawn prompt | Answer a codebase question |
| **Reading Pattern** | Targeted (specific files) | Exploratory (following links) |
| **Output** | Spawn command executed | Investigation artifact |
| **Question** | "What should I tell the agent?" | "How does X work?" |

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The analysis is based purely on the skill text, which provides clear evidence of contradictory guidance. The proposed distinction (purpose-based + time-boxed) is internally consistent and resolves the contradiction logically.

**What's certain:**

- ✅ The skill currently contains contradictory guidance about reading/investigation
- ✅ Three distinct activities exist (routing, enrichment, exploration)
- ✅ Time is a useful practical boundary between allowed and forbidden work

**What's uncertain:**

- ⚠️ Whether orchestrators in practice struggle with this distinction
- ⚠️ Whether 5 minutes is the right boundary (could be 3 or 10)
- ⚠️ Whether "purpose" is enforceable or just a rationalization

**What would increase confidence to Very High (95%):**

- Observation of actual orchestrator sessions hitting this boundary
- Dylan's confirmation that the proposed distinction matches his mental model
- Testing whether the updated skill produces better orchestrator behavior

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add "Pre-Spawn Context Gathering" section to orchestrator skill** - Explicitly define what context work is allowed, with time-boxing and purpose as distinguishing factors.

**Why this approach:**
- Resolves the contradiction in current skill guidance
- Gives orchestrators concrete rules instead of ambiguous prohibitions
- Preserves the "never do deep investigation" principle while clarifying allowed activities

**Trade-offs accepted:**
- Adds ~30-50 lines to an already long skill document
- Time-boxing (5 min) is somewhat arbitrary

**Implementation sequence:**
1. Add new section after "Pre-Spawn Knowledge Check" called "Context Gathering vs Investigation"
2. Provide explicit table distinguishing allowed from forbidden activities
3. Update the "test" on line 299 to be more nuanced
4. Add time-boxing guidance with 5-minute threshold

### Alternative Approaches Considered

**Option B: Keep skill as-is, rely on orchestrator judgment**
- **Pros:** No skill changes needed
- **Cons:** Leaves ambiguity that causes inconsistent behavior
- **When to use instead:** If the confusion is theoretical, not practical

**Option C: Stricter prohibition - orchestrators ONLY run commands, never read**
- **Pros:** Simpler, clearer boundary
- **Cons:** Makes orchestrators ineffective at spawn prompt creation
- **When to use instead:** If context enrichment proves to be a slippery slope

**Rationale for recommendation:** Option A resolves the actual confusion while preserving the useful constraint. Option B ignores a real problem. Option C is too restrictive.

---

### Implementation Details

**What to implement first:**
- Add the distinction table from the Synthesis section to the skill
- Update line 299 ("If you're about to read code...") with nuance

**Things to watch out for:**
- ⚠️ Don't add so much nuance that the simple "never investigate" message is lost
- ⚠️ The 5-minute threshold needs to be a guideline, not a hard rule
- ⚠️ Purpose-based distinctions can be rationalized - pair with time-boxing

**Areas needing further investigation:**
- How do other orchestration systems distinguish coordinator vs worker responsibilities?
- Is there a way to automatically detect when orchestrators cross into investigation mode?

**Success criteria:**
- ✅ Orchestrators can confidently do pre-spawn context work without guilt
- ✅ The "never investigate deeply" principle remains clear
- ✅ Spawn prompts become richer because orchestrators aren't afraid to read kb context results

---

## Proposed Skill Update

Add this section after "Pre-Spawn Knowledge Check":

```markdown
---

## Context Gathering vs Investigation

**Orchestrators do context gathering. Workers do investigation.**

The Pre-Spawn Knowledge Check requires reading - that's allowed. The ABSOLUTE DELEGATION RULE prohibits exploration - that's still forbidden. Here's the distinction:

| Activity | Allowed? | Purpose | Time Limit |
|----------|----------|---------|------------|
| Run `kb context`, read output | ✅ Yes | Know what to tell agent | < 2 min |
| Read completed SYNTHESIS.md | ✅ Yes | Know what agent found | < 2 min |
| Scan workspace for status | ✅ Yes | Know if agent finished | < 1 min |
| Read beads issue for context | ✅ Yes | Write effective spawn prompt | < 1 min |
| Summarize prior findings in spawn prompt | ✅ Yes | Prevent duplicate work | < 3 min |
| Read code to understand behavior | ❌ No | Understanding codebase | N/A - delegate |
| Follow links across multiple files | ❌ No | Exploring codebase | N/A - delegate |
| Test hypotheses about bugs | ❌ No | Debugging | N/A - delegate |
| Any reading > 5 minutes total | ❌ No | Likely investigation | N/A - delegate |

**The 5-minute rule:** If context-gathering for a spawn is taking more than 5 minutes, you've likely crossed into investigation territory. Stop and spawn instead:
- Can't figure out what prior investigation found? → Spawn to synthesize
- Need to understand how code works to write prompt? → Spawn investigation, then spawn implementation

**The purpose test:** Ask yourself: "Am I reading to write a spawn prompt, or reading to answer a question?" 
- Writing spawn prompt → Allowed (brief, targeted)
- Answering question → Delegate (that's what agents are for)

**Examples:**

✅ **Allowed:** "kb context shows a prior investigation on auth. Let me read the TLDR to include in the spawn prompt." (2 min)

❌ **Forbidden:** "kb context shows a prior investigation on auth. Let me read the full investigation and understand the auth system." (15+ min)

✅ **Allowed:** "Agent completed. Let me read SYNTHESIS.md to know what they found." (1 min)

❌ **Forbidden:** "Agent completed. Let me read their code changes to understand what they implemented." (10+ min)
```

Also update line 299 from:
```markdown
**The test:** If you're about to read code to understand something → STOP. That's an investigation. Spawn it.
```

To:
```markdown
**The test:** If you're about to read code to understand *how it works* → STOP. That's an investigation. Spawn it. (Reading kb context results and SYNTHESIS.md to write spawn prompts is allowed - see "Context Gathering vs Investigation" section.)
```

---

## References

**Files Examined:**
- ~/.claude/skills/meta/orchestrator/SKILL.md - Full orchestrator skill (1320 lines)

**Commands Run:**
```bash
# Get existing knowledge
kb context "orchestrator delegation spawn context"

# Get investigation knowledge context
kb context "investigation"
```

**Related Artifacts:**
- **Decision:** .kb/decisions/2025-12-04-orchestrator-delegates-all-investigations.md - Original delegation rule
- **Skill source:** /Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc

---

## Investigation History

**2025-12-25 ~10:00:** Investigation started
- Initial question: What's the right boundary between spawn context gathering and deep investigation?
- Context: Spawned as design-session to clarify orchestrator skill ambiguity

**2025-12-25 ~10:30:** Key findings complete
- Identified three categories of context work
- Found time and purpose as distinguishing factors
- Proposed explicit skill update

**2025-12-25 ~10:45:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Proposed 30-50 line skill update with explicit table and 5-minute rule
