---
linked_issues:
  - orch-go-xdr7
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawned meta-orchestrators collapse to worker behavior because ORCHESTRATOR_CONTEXT.md frames them with task-completion goals rather than interactive session management guidance.

**Evidence:** Session transcript (session-ses_4743.md) shows agent immediately doing worker-level file reading, writing synthesis, and attempting `orch session end`; template review shows task-oriented framing despite meta-orchestrator skill content.

**Knowledge:** Even with comprehensive skill guidance embedded, framing cues in context templates override skill instructions; "Session Goal" + "Begin working toward your session goal" creates task-completion framing.

**Next:** Redesign ORCHESTRATOR_CONTEXT.md template to use interactive session framing for meta-orchestrators, distinguishing spawned-orchestrator (goal-focused) from spawned-meta-orchestrator (session-management focused).

---

# Investigation: Meta-Orchestrator Level Collapse

**Question:** Why do spawned meta-orchestrators behave like workers instead of staying interactive and delegating?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None (recommend template and skill fixes)
**Status:** Complete

---

## Findings

### Finding 1: ORCHESTRATOR_CONTEXT.md Uses Task-Completion Framing

**Evidence:** The template in `pkg/spawn/orchestrator_context.go:19-116` frames the session as:
- "Session Goal:" + goal text (line 21)
- "Begin working toward your session goal" (line 43)  
- "When you've accomplished your session goal" completion protocol (line 68)
- First Actions: "Begin working toward your session goal" (line 43)

This is task-completion framing - the same structure workers receive, just with different artifact names (SESSION_HANDOFF.md instead of SYNTHESIS.md).

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/orchestrator_context.go:19-116`

**Significance:** Framing is a powerful behavioral cue. "Work toward goal" implies doing work, not managing sessions. The template doesn't distinguish between orchestrator-level goals (spawn and manage workers) and meta-orchestrator-level goals (spawn and manage orchestrator sessions).

---

### Finding 2: Meta-Orchestrator Skill Embedded But Overridden by Framing

**Evidence:** The session transcript (session-ses_4743.md) shows:
1. Agent received full meta-orchestrator skill (via dependencies) including:
   - "Manage orchestrator sessions like orchestrators manage workers"
   - "Don't Drop Levels" guardrails
   - "Spawning Orchestrator Sessions" instructions
2. Despite this, agent immediately:
   - Started reading files (worker behavior)
   - Writing SESSION_HANDOFF.md (completion artifact)
   - Attempting `orch session end` (session termination)

The skill says "spawn an orchestrator session instead" but the context framing says "work toward your session goal." Framing won.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/session-ses_4743.md:23-229`

**Significance:** Skill content is necessary but not sufficient. The opening context framing sets the agent's behavioral mode before skill content is processed.

---

### Finding 3: No Distinction Between Orchestrator and Meta-Orchestrator Spawns

**Evidence:** Both orchestrator and meta-orchestrator skills have `skill-type: policy`, so they receive identical ORCHESTRATOR_CONTEXT.md templates. The template:
- Doesn't check skill name
- Uses same structure for both
- Same "Session Goal" framing regardless of level

The meta-orchestrator skill expects to spawn orchestrator sessions, but receives a template that says "work toward goal" and "produce SESSION_HANDOFF.md" - exactly what an orchestrator would do with workers.

**Source:** 
- `pkg/spawn/orchestrator_context.go` - single template for all policy skills
- `~/.claude/skills/meta/meta-orchestrator/SKILL.md` - expects to spawn orchestrators
- `~/.claude/skills/meta/orchestrator/SKILL.md` - same template but spawns workers

**Significance:** The spawning infrastructure doesn't distinguish levels. A meta-orchestrator spawn looks like an orchestrator spawn, creating level confusion.

---

### Finding 4: Agent Self-Diagnosed the Problem When Prompted

**Evidence:** From the transcript, when Dylan asked "what is your role?", the agent correctly identified:
- "I'm a spawned orchestrator, not a worker"
- "My role is to spawn worker agents... Instead, I dropped into worker mode"
- "I violated the ABSOLUTE DELEGATION RULE"

But even after this self-diagnosis, when asked "you're an orchestrator not a meta-orchestrator?", the agent had to re-read context to confirm:
- "Looking at my ORCHESTRATOR_CONTEXT.md: Skill: meta-orchestrator"
- "So I am a meta-orchestrator. Which means I dropped TWO levels"

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/session-ses_4743.md:360-425`

**Significance:** The skill content was there but the agent didn't internalize it on first read. The task-completion framing was stronger than skill guidance. Even when prompted to reflect, the agent first thought "orchestrator" not "meta-orchestrator" - showing the skill distinction wasn't prominent enough.

---

### Finding 5: Interactive Session Pattern Is Correct for Meta-Orchestrators

**Evidence:** The decision record at `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` states:
- "The transition from orchestrator to meta-orchestrator is a **frame shift**"
- "Dylan operates from the meta frame - Claude instances operate as orchestrators within that frame"
- "Orchestrator sessions become objects - to spawn, monitor, complete"

This confirms meta-orchestrators should be interactive (staying available for conversation and session management), not task-completing (doing work and exiting).

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md:1-53`

**Significance:** The architecture decision is clear but the template doesn't implement it. Meta-orchestrators need different framing than orchestrators.

---

## Synthesis

**Key Insights:**

1. **Framing Trumps Skill Content** - The context template sets behavioral mode before skill instructions are processed. "Work toward goal" = task completion mode, regardless of skill content saying otherwise.

2. **No Level Differentiation** - The spawning infrastructure treats all `skill-type: policy` skills identically, using the same template. Meta-orchestrators receive orchestrator-like framing.

3. **Self-Correction Requires External Prompting** - The agent didn't catch its own level violation until Dylan asked "what is your role?" Skill guidance wasn't strong enough to prevent the behavior proactively.

**Answer to Investigation Question:**

Spawned meta-orchestrators behave like workers because:

1. **Template framing** uses task-completion language ("work toward goal", "when you've accomplished")
2. **No template differentiation** for meta-orchestrator vs orchestrator spawns
3. **Skill guidance is passive** - it says "don't do X" but doesn't actively redirect behavior

The fix requires:
1. Different template framing for meta-orchestrators (session-management, not task-completion)
2. Active behavioral cues at the start ("You are managing orchestrator sessions, not doing work")
3. First action instructions that establish the right mode ("Check for orchestrator sessions to review or spawn")

---

## Structured Uncertainty

**What's tested:**

- ✅ Template uses task-completion framing (verified: read orchestrator_context.go)
- ✅ Agent received full skill content but did worker-level work (verified: session transcript)
- ✅ Agent required external prompting to recognize level violation (verified: session transcript lines 355-425)
- ✅ No distinction between orchestrator and meta-orchestrator template (verified: single template, no skill-name checks)

**What's untested:**

- ⚠️ Whether revised template framing would prevent level collapse (hypothesis: better framing = better behavior)
- ⚠️ Whether meta-orchestrators should even be spawnable (may always need human-in-the-loop)
- ⚠️ Whether separating meta-orchestrator into distinct context type is sufficient

**What would change this:**

- If agents are shown to ignore framing cues entirely (would need different approach)
- If meta-orchestrator spawning is deemed architecturally wrong (would remove capability)
- If framing changes don't improve behavior in practice (would need stronger guardrails)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Tiered Context Templates** - Create separate context template for meta-orchestrator spawns with interactive session framing instead of task-completion framing.

**Why this approach:**
- Addresses root cause (framing, not skill content)
- Preserves spawnable orchestrators for normal orchestrator level
- Follows existing pattern (SPAWN_CONTEXT.md vs ORCHESTRATOR_CONTEXT.md)
- Low implementation risk (template change, not architectural)

**Trade-offs accepted:**
- Adds complexity (three context types instead of two)
- May need skill-name check in spawning logic
- Meta-orchestrator skill guidance still needs to be clear

**Implementation sequence:**
1. Create META_ORCHESTRATOR_CONTEXT.md template with interactive framing
2. Add skill-name detection in spawn logic for "meta-orchestrator"
3. Update meta-orchestrator skill to reference new framing explicitly

### Alternative Approaches Considered

**Option B: Stronger Skill Guardrails**
- **Pros:** No infrastructure changes needed
- **Cons:** Framing still sets wrong mode; skill content didn't prevent issue
- **When to use instead:** If template changes don't improve behavior

**Option C: Make Meta-Orchestrators Non-Spawnable**
- **Pros:** Matches "human-in-the-loop" philosophy
- **Cons:** Limits automation, prevents overnight meta-orchestration
- **When to use instead:** If meta-orchestrators fundamentally need human presence

**Rationale for recommendation:** Template framing is the root cause. Skill content was comprehensive but ignored. Changing framing at the source is cleaner than adding more guardrails.

---

### Implementation Details

**What to implement first:**
- Create `MetaOrchestratorContextTemplate` in `pkg/spawn/` with interactive framing
- Key framing changes:
  - "You are managing orchestrator sessions" not "work toward goal"
  - First action: "Check `orch status` for sessions to complete or review"
  - No SESSION_HANDOFF.md requirement (stay interactive)
  - "Stay available for conversation" not "accomplish and exit"

**Things to watch out for:**
- ⚠️ Skill detection logic must correctly identify "meta-orchestrator" vs "orchestrator"
- ⚠️ Meta-orchestrator still needs embedded skill content (don't remove)
- ⚠️ Test with actual spawned session to verify behavior change

**Areas needing further investigation:**
- Should meta-orchestrators be spawnable at all? (architectural question)
- What's the right completion pattern for meta-orchestrators?
- Does the meta-orchestrator skill need updating beyond template?

**Success criteria:**
- ✅ Spawned meta-orchestrator asks about orchestrator sessions first, doesn't start reading code
- ✅ No attempt to complete task or exit without external prompting
- ✅ Agent correctly identifies as "meta-orchestrator" on first reflection, not "orchestrator"

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/orchestrator_context.go` - Template source
- `/Users/dylanconlin/Documents/personal/orch-go/session-ses_4743.md` - Evidence transcript
- `/Users/dylanconlin/.claude/skills/meta/meta-orchestrator/SKILL.md` - Meta-orchestrator skill
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Orchestrator skill (inherited)
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Decision record

**Commands Run:**
```bash
# Find template and context files
grep -r "ORCHESTRATOR_CONTEXT" /Users/dylanconlin/Documents/personal/orch-go

# Find skill files
find ~/.claude/skills -name "*meta*" -o -name "*orchestrator*"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Establishes frame shift principle
- **Workspace:** `.orch/workspace/og-work-understand-meta-orchestration-04jan/` - The collapsed session

---

## Investigation History

**2026-01-04 17:50:** Investigation started
- Initial question: Why do spawned meta-orchestrators behave like workers?
- Context: Session transcript showed agent doing worker-level work despite meta-orchestrator skill

**2026-01-04 18:10:** Found template framing issue
- ORCHESTRATOR_CONTEXT.md uses task-completion language
- No distinction between orchestrator and meta-orchestrator spawns

**2026-01-04 18:25:** Confirmed skill content was present but overridden
- Full meta-orchestrator skill embedded in session
- Agent still did worker-level work until externally prompted

**2026-01-04 18:35:** Investigation completed
- Status: Complete
- Key outcome: Template framing is root cause; recommend tiered context templates with interactive framing for meta-orchestrators

---

## Self-Review

- [x] Real test performed (analyzed actual session transcript, not hypothetical)
- [x] Conclusion from evidence (based on transcript and template analysis)
- [x] Question answered (why level collapse happens)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

---

## Discovered Work

During this investigation, the following work items were identified:

1. **Feature:** Create META_ORCHESTRATOR_CONTEXT.md template (recommended fix)
2. **Task:** Add skill-name detection in spawn logic for meta-orchestrator
3. **Task:** Update meta-orchestrator skill to reference interactive framing

These should be created as beads issues for implementation.
