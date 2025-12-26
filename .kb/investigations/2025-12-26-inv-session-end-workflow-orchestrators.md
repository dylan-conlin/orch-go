<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator session-end workflow has three gaps vs worker patterns: no friction audit, no gap capture ritual, and no system reaction check.

**Evidence:** Worker skills have "Leave it Better" phase (mandatory kn command before completion); orchestrator skill has "session-transition" for git/cleanup but nothing for reflection.

**Knowledge:** Workers externalize session learnings via kn commands because it's gated; orchestrators don't because there's no trigger - same reflection prompts every session indicates Dylan is compensating for missing automation.

**Next:** Add "Session Reflection" section to orchestrator skill (before "Landing the Plane" / session-transition) with three checkpoints: friction audit, gap capture, system reaction check.

**Confidence:** High (85%) - clear gap identified but optimal trigger mechanism needs validation.

---

# Investigation: Session End Workflow Orchestrators

**Question:** What should orchestrator session-end pattern look like? Should it be a checklist in skill, separate section, and what triggers it?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Dylan Conlin
**Phase:** Complete
**Next Step:** Present findings to orchestrator for decision
**Status:** Complete
**Confidence:** High (85%)

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Workers have "Leave it Better" - Orchestrators don't

**Evidence:** 
- Worker skills (feature-impl, investigation, codebase-audit, design-session) all include mandatory "Leave it Better" phase
- Pattern: "Before marking complete, you MUST externalize at least one piece of knowledge"
- Commands used: `kn decide`, `kn tried`, `kn constrain`, `kn question`
- Gate: Completion criteria includes "Leave it Better completed"

**Source:** 
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/SKILL.md:321`
- `/Users/dylanconlin/.claude/skills/src/worker/codebase-audit/SKILL.md:1418-1495` (full phase template)

**Significance:** Workers have a forcing function for knowledge externalization. Orchestrators don't - they rely on manual discipline, which fails under cognitive load ("Anti-pattern: 'I'll document later' → later never comes").

---

### Finding 2: Session-transition skill focuses on git/cleanup, not reflection

**Evidence:**
The session-transition skill has 5 steps:
1. Assess state (git status, workspace parsing)
2. Capture context (write Session Transition section)
3. Offer cleanup (tmux sessions, session learnings)
4. Recommend resume strategy
5. Execute transition

Step 3 mentions "session learnings detection" but this:
- Looks for indicators IN workspace files (Decision:, Learning:, Pattern:)
- Offers to run `extract-session-learnings` skill
- Is reactive (looks for what was already written) not proactive (prompts reflection)

**Source:** `/Users/dylanconlin/.claude/skills/shared/session-transition/SKILL.md:275-349`

**Significance:** The existing skill assumes learnings are already captured somewhere. It doesn't prompt for *new* reflection - just offers to extract existing artifacts. Dylan's "same reflection prompt every session" suggests he's manually filling this gap.

---

### Finding 3: Orchestrator skill has extensive guidance but no session-end reflection section

**Evidence:**
Orchestrator skill covers:
- Orchestrator Core Responsibilities (L535-567)
- Post-Completion Verification (L1227-1243)
- Orchestrator Completion Lifecycle (L1245-1315)
- Follow-up Extraction (L1303-1312) 

But these are for *agent* completion, not *orchestrator session* end. No equivalent of "before ending YOUR session, reflect on..."

The closest is `~/.claude/CLAUDE.md:45-51`:
```
Reflection checkpoint: After completing significant work, ask:
- Did I learn something that will recur? → kn or skill
- Did I establish a pattern others should follow? → guide or CLAUDE.md
- Did I make a decision that should outlast this session? → decision record
- Did I keep forgetting something until gated? → hook
```

But this is:
- In a separate file (not embedded in orchestrator skill)
- A general "reflection checkpoint" not an end-of-session trigger
- Not gated (no completion criteria enforces it)

**Source:** 
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` (full skill)
- `/Users/dylanconlin/.claude/CLAUDE.md:45-51`

**Significance:** Orchestrator sessions can end without any reflection ritual. Dylan compensates by asking the same reflection prompts manually.

---

### Finding 4: orch learn infrastructure exists but isn't triggered at session-end

**Evidence:**
The orchestrator skill documents `orch learn` commands:
- `orch learn` - Show suggestions for recurring context gaps
- `orch learn patterns` - Analyze gap patterns by topic
- `orch learn effects` - Check if past improvements helped

And the "Pressure Over Compensation" principle (L570-616) explicitly says:
- "When the system fails to surface knowledge, don't compensate by providing it manually"
- "Let the failure surface → That failure is data"
- "Use `orch learn` to track gaps"

But there's no trigger to run `orch learn` at session end.

**Source:** `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md:1504-1510`

**Significance:** The tooling exists for gap analysis, but there's no session-end forcing function to use it.

---

### Finding 5: Dylan's "same reflection prompt every session" indicates a repeatable need

**Evidence:**
From task description: "User says same reflection prompt every session"

This signals:
1. Dylan has a mental checklist he runs at session end
2. He's having to prompt it manually (not automated)
3. The same prompts recurring = a pattern that should be externalized

This maps to the Knowledge Placement table trigger: "I keep explaining how to do X" → Skill.

**Source:** Task description / SPAWN_CONTEXT.md

**Significance:** Dylan is doing manually what should be automated. This is exactly the pattern the knowledge system is designed to detect and fix.

---

## Synthesis

**Key Insights:**

1. **Asymmetry between worker and orchestrator patterns** - Workers have "Leave it Better" as a gated phase; orchestrators have nothing equivalent. Workers can't complete without knowledge externalization; orchestrators can end sessions without any reflection.

2. **Session-transition is about context preservation, not reflection** - The existing skill captures "what state is the work in" for resumption. It doesn't prompt "what did we learn" - a different concern. These are complementary but distinct phases.

3. **The "same reflection prompt every session" is a design signal** - When Dylan does the same manual work repeatedly, that's exactly the trigger for skill creation. The system is telling us what to automate.

**Answer to Investigation Question:**

The orchestrator session-end pattern should have **two distinct phases**:

1. **Session Reflection (NEW)** - BEFORE git cleanup
   - Friction audit: "What was harder than it should have been?"
   - Gap capture: "What knowledge should have been surfaced but wasn't?"
   - System reaction check: "Does this need a hook/skill/CLAUDE.md update?"
   - Gate: Must run `orch learn` or `kn` command (or explicitly skip)

2. **Landing the Plane (EXISTING)** - Session-transition skill
   - Git status/cleanup
   - Context capture for resume
   - Tmux cleanup offer

**What triggers it?**
- User says "end session" / "let's wrap up" / "that's it for today"
- Context limit warning
- Orchestrator announces intention to end
- Could be enforced by a SessionEnd hook

**Where should it live?**
- Add "Session Reflection" section to orchestrator skill (`.claude/skills/meta/orchestrator/SKILL.md`)
- Before "session-transition" in the Coordination Skills section
- Include the three checkpoints as a checklist
- Gate completion on at least one of: `orch learn` / `kn` command / explicit skip with reason

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence that the gap exists (worker patterns vs orchestrator patterns, Dylan's manual compensation). Less certainty about the optimal trigger mechanism and whether gating is too heavy for orchestrator flow.

**What's certain:**

- ✅ Workers have "Leave it Better" as a gated phase; orchestrators don't
- ✅ Session-transition focuses on git/cleanup, not reflection
- ✅ Dylan is manually prompting the same reflection questions every session
- ✅ The three concerns (friction, gaps, system reaction) map to existing tools (orch learn, kn)

**What's uncertain:**

- ⚠️ Whether hard gating (must run command) is appropriate for orchestrator flow
- ⚠️ Whether a SessionEnd hook is the right enforcement mechanism vs skill section
- ⚠️ What Dylan's actual manual reflection prompt contains

**What would increase confidence to Very High (95%+):**

- Ask Dylan what his actual manual reflection prompt is
- Validate that the three checkpoints (friction/gap/reaction) capture what he's asking
- Test whether orch learn is the right tool or if something simpler is needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add "Session Reflection" section to orchestrator skill** - A new section in the orchestrator skill that provides a structured checklist for session-end reflection, triggered by session-end signals.

**Why this approach:**
- Aligns with existing skill structure (orchestrators already load the skill)
- Parallels the "Leave it Better" phase that works for workers
- Doesn't require new tooling (uses existing orch learn / kn commands)
- Progressive: can start as guidance, add hook enforcement later if needed

**Trade-offs accepted:**
- Not automatically enforced (relies on orchestrator following skill)
- Hook-based enforcement could be added later but may be too heavy
- May need iteration based on Dylan's actual needs

**Implementation sequence:**
1. Add "Session Reflection" section to orchestrator skill - establishes the pattern
2. Include the three checkpoints as a checklist - makes it concrete
3. Reference existing tools (orch learn, kn) - uses existing infrastructure
4. Optionally add SessionEnd hook later - if enforcement is needed

### Alternative Approaches Considered

**Option B: Create a new "session-reflection" skill**
- **Pros:** Cleaner separation, can be invoked explicitly
- **Cons:** Adds another skill to maintain; orchestrator skill already exists
- **When to use instead:** If session-end reflection becomes complex enough to warrant dedicated skill

**Option C: Add SessionEnd hook to enforce reflection**
- **Pros:** Automatic enforcement, can't be skipped
- **Cons:** May be too heavy for orchestrator flow; hooks run unconditionally
- **When to use instead:** If orchestrators consistently skip the reflection section

**Option D: Expand session-transition skill to include reflection**
- **Pros:** Keeps all session-end logic together
- **Cons:** session-transition is shared (worker/orchestrator); reflection is orchestrator-specific
- **When to use instead:** If reflection applies to all session types

**Rationale for recommendation:** Option A integrates with existing orchestrator patterns without new tooling. The orchestrator skill already has extensive guidance; adding a section is minimal friction. Hook enforcement (Option C) can be added later if needed.

---

### Implementation Details

**What to implement first:**
- Add "Session Reflection" section to orchestrator skill between "Orchestrator Completion Lifecycle" and existing session-transition reference
- Include three checkpoints as markdown checklist
- Reference the existing tools

**Proposed section content:**

```markdown
## Session Reflection (Before Ending)

**When to use:** Before ending an orchestrator session (user says "wrap up", context limit, natural stopping point).

**The Three Checkpoints:**

1. **Friction Audit:** What was harder than it should have been?
   - Did I have to explain something that should have been in context?
   - Did I hit a wall that another agent had already solved?
   - `orch learn` to see recurring gaps

2. **Gap Capture:** What knowledge should have been surfaced but wasn't?
   - Constraints discovered during spawns
   - Decisions made that should outlast this session
   - `kn decide/tried/constrain/question` to externalize

3. **System Reaction Check:** Does this session suggest system improvements?
   - New skill needed? (explained same procedure 3+ times)
   - New hook needed? (kept forgetting something)
   - CLAUDE.md update? (new constraint applies to all projects)

**Gate:** Run at least one of:
- `orch learn` (even if no action taken)
- Any `kn` command
- Explicit skip: "Session Reflection: No friction detected, no gaps to capture"

Then proceed to session-transition for git/cleanup.
```

**Things to watch out for:**
- ⚠️ Don't make this so heavy that orchestrators skip it
- ⚠️ The three checkpoints should map to actual tools (orch learn, kn)
- ⚠️ Need to validate with Dylan that these are the right questions

**Areas needing further investigation:**
- What is Dylan's actual manual reflection prompt?
- Is there a SessionEnd hook that could enforce this later?
- Should orch learn be modified to prompt for session-end context?

**Success criteria:**
- ✅ Dylan no longer needs to manually prompt the same reflection questions
- ✅ Session-end captures at least one of: orch learn run, kn entry, explicit skip
- ✅ Friction/gap patterns are captured for system improvement

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/shared/session-transition/SKILL.md` - Existing session-end workflow (git/cleanup focused)
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Full orchestrator guidance (no session-end reflection)
- `/Users/dylanconlin/.claude/skills/src/worker/codebase-audit/SKILL.md:1418-1495` - "Leave it Better" phase template
- `/Users/dylanconlin/.claude/CLAUDE.md:45-51` - Reflection checkpoint guidance

**Commands Run:**
```bash
# Find session-end patterns in skills
grep -r "session.*end|end.*session" ~/.claude/skills --include="*.md"

# Find "Leave it Better" patterns
grep -r "Leave it Better" ~/.claude/skills

# Search kb for session-related knowledge
kb context "session-end landing plane reflection"
```

**Related Artifacts:**
- **Skill:** `/Users/dylanconlin/.claude/skills/shared/session-transition/SKILL.md` - Related (session-end cleanup)
- **Principle:** Pressure Over Compensation in orchestrator skill - Related (let gaps surface)

---

## Investigation History

**2025-12-26 [start]:** Investigation started
- Initial question: What should orchestrator session-end pattern look like?
- Context: Dylan reports same reflection prompt every session; current skill focuses on git/cleanup

**2025-12-26:** Context gathering complete
- Found: Workers have "Leave it Better" (gated), orchestrators don't
- Found: Session-transition skill is cleanup-focused, not reflection-focused
- Found: orch learn infrastructure exists but not triggered at session-end

**2025-12-26:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend adding "Session Reflection" section to orchestrator skill with three checkpoints (friction audit, gap capture, system reaction check)
