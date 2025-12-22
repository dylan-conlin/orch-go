---
linked_issues:
  - orch-go-4kwt.8
---
## Summary (D.E.K.N.)

**Delta:** Artifact-level approach (adding "Unexplored Questions" section to SYNTHESIS.md) is the minimal change that captures reflection value without adding new tooling or process overhead.

**Evidence:** Examined existing SYNTHESIS.md template (already has spawn-follow-up, escalate recommendations), found real example of post-synthesis reflection creating new epic (orch-go-ws4z), and analyzed 4 options (skill/spawn/protocol/artifact) for implementation complexity.

**Knowledge:** The value comes from Dylan's interactive follow-up on completed work, not from agents asking questions during execution; the existing orch review → orch send workflow already supports this pattern.

**Next:** Add "Unexplored Questions" section to SYNTHESIS.md template; no skill/spawn changes needed.

**Confidence:** High (85%) - Based on concrete evidence from existing patterns, but would benefit from user validation of the proposed template change.

---

# Investigation: Reflection Checkpoint Pattern for Agent Sessions

**Question:** Should reflection checkpoints be implemented at skill-level (new phase), spawn-level (--interactive flag), protocol-level (template update), or artifact-level (SYNTHESIS.md section)? What's the minimal change?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: SYNTHESIS.md Already Has Reflection Scaffolding

**Evidence:** The SYNTHESIS.md template at `.orch/templates/SYNTHESIS.md` includes:
- "**Recommendation:** {close | spawn-follow-up | escalate | resume}" (line 64)
- "If Spawn Follow-up" section with Issue/Skill/Context fields (lines 72-79)
- "If Escalate" section with Question/Options/Recommendation (lines 81-86)

**Source:** `.orch/templates/SYNTHESIS.md:64-86`

**Significance:** The artifact structure already supports capturing follow-up work and escalation questions. Agents can already indicate when work needs continuation. The gap is capturing questions/insights that emerged but weren't acted on.

---

### Finding 2: Post-Synthesis Reflection Already Happens Organically

**Evidence:** In workspace `og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md`:
- "Post-synthesis reflection with Dylan led to new epic (orch-go-ws4z)"
- The reflection surfaced 3 deeper questions Dylan hadn't asked
- Led to a 6-child epic exploring system self-reflection

**Source:** `.orch/workspace/og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md` (full content examined)

**Significance:** The highest value comes from orchestrator reviewing completed work and having follow-up discussion. The value isn't in agents asking questions during execution—it's in preserving insights that emerged so the orchestrator can decide what to pursue.

---

### Finding 3: Existing Workflow Supports Interactive Follow-up

**Evidence:** The existing command set includes:
- `orch review <id>` - Review agent work before completing (cmd/orch/review.go)
- `orch send <session-id> "message"` - Send follow-up messages to existing sessions
- `orch resume <id>` - Continue paused agents with workspace context

**Source:** 
- `cmd/orch/review.go:264-331` (runReviewSingle function)
- `cmd/orch/main.go:236-252` (send command)
- `cmd/orch/resume.go:38-47` (GenerateResumePrompt)

**Significance:** The tooling for interactive follow-up already exists. The gap is: agents don't consistently surface what questions/insights emerged, so the orchestrator doesn't know what to ask about.

---

### Finding 4: Four Options Have Different Implementation Costs

**Evidence:** Analyzed each option:

| Option | Change Required | Benefit | Cost |
|--------|-----------------|---------|------|
| **Skill-level (reflection phase)** | New phase in all spawnable skills, phase detection | Structured reflection forced | High - touches many skills |
| **Spawn-level (--interactive flag)** | New flag, behavior change, registry tracking | Some spawns wait for interaction | Medium - new spawn mode |
| **Protocol-level (template update)** | Update SPAWN_CONTEXT.md template | Prompts agents to capture questions | Low - template only |
| **Artifact-level (SYNTHESIS.md section)** | Add section to SYNTHESIS.md template | Questions preserved in output | Lowest - ~10 lines |

**Source:** Analysis of:
- Skill system: `~/.claude/skills/` (87 files)
- Spawn command: `cmd/orch/main.go:169-222`
- SPAWN_CONTEXT template: `pkg/spawn/context.go`
- SYNTHESIS.md template: `.orch/templates/SYNTHESIS.md`

**Significance:** Artifact-level change has lowest implementation cost while capturing the core value: preserving emergent questions/insights for orchestrator review.

---

## Synthesis

**Key Insights:**

1. **Value is in the output, not the process** - The examples show value comes from Dylan reading what emerged and deciding to explore further. Adding process (new phases, interactive flags) adds friction without necessarily improving output quality.

2. **Existing workflow is sufficient** - `orch review` → read synthesis → `orch send "follow-up question"` is already the pattern. The gap is just ensuring synthesis captures what to follow up on.

3. **Minimal change principle** - Given that the scaffolding exists (SYNTHESIS template, review command, send command), adding a section to SYNTHESIS.md is the smallest change that captures the value.

**Answer to Investigation Question:**

The minimal change is **artifact-level**: Add an "Unexplored Questions" section to SYNTHESIS.md. This captures emergent questions/insights during agent work without adding new tooling, process steps, or behavior changes.

The other options are unnecessary because:
- **Skill-level phase:** Would require touching multiple skills, adds execution time, and agents already reflect naturally during synthesis
- **Spawn-level flag:** Adds complexity for orchestrator to decide upfront; review-time interaction is better (orchestrator sees what agent produced first)
- **Protocol-level template update:** Could reinforce reflection, but SYNTHESIS.md is where the output lives and is already reviewed

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence from:
- Real example of post-synthesis reflection creating valuable follow-up (orch-go-ws4z epic)
- Concrete analysis of existing templates and tooling
- Clear cost comparison across options

**What's certain:**

- ✅ SYNTHESIS.md template already has scaffolding for follow-up recommendations
- ✅ Post-synthesis reflection happens organically when orchestrator reviews work
- ✅ Existing tooling (review, send) supports the interaction pattern
- ✅ Artifact-level change has lowest implementation cost

**What's uncertain:**

- ⚠️ Whether agents will consistently populate an "Unexplored Questions" section without reinforcement
- ⚠️ Optimal phrasing/structure for the new section
- ⚠️ Whether skill-level prompts would improve quality of captured questions

**What would increase confidence to Very High:**

- User validation of proposed template change
- Testing with 3-5 agent sessions to see quality of captured questions
- Feedback on whether orchestrator finds the questions actionable

---

## Implementation Recommendations

### Recommended Approach ⭐

**Artifact-level: Add "Unexplored Questions" section to SYNTHESIS.md template**

**Why this approach:**
- Captures emergent questions at the point they're most clear (session end)
- Integrates with existing `orch review` workflow—orchestrator already reads SYNTHESIS.md
- Zero new commands, flags, or behavior changes
- Agents already produce SYNTHESIS.md; just adding structure

**Trade-offs accepted:**
- Relies on agent discipline to fill section (vs forcing via process)
- Questions may be lower quality without explicit reflection time
- Orchestrator must actively check this section

**Implementation sequence:**
1. Add "Unexplored Questions" section to `.orch/templates/SYNTHESIS.md` (before Session Metadata)
2. Update investigation skill's self-review to include "questions captured" check
3. Optionally: Update `orch review` display to highlight unexplored questions

### Alternative Approaches Considered

**Option B: Protocol-level (SPAWN_CONTEXT.md template update)**
- **Pros:** Reinforces reflection early, sets expectation at spawn
- **Cons:** Agents forget by session end; SYNTHESIS.md is where output lives anyway
- **When to use instead:** If artifact-level alone produces empty sections consistently

**Option C: Skill-level (new reflection phase)**
- **Pros:** Structured reflection time, quality control
- **Cons:** High cost (many skills to update), extends session time
- **When to use instead:** If investigation shows agents need forced reflection pause

**Option D: Spawn-level (--interactive flag)**
- **Pros:** Explicit orchestrator intent to follow up
- **Cons:** Requires upfront decision before seeing output; review-time interaction is better
- **When to use instead:** For specific high-value tasks where follow-up is known ahead

**Rationale for recommendation:** Artifact-level has highest value-to-cost ratio. The value comes from preserving questions for orchestrator review, not from forcing process changes. Start minimal, escalate if needed.

---

### Implementation Details

**What to implement first:**
- Add section to SYNTHESIS.md template (5-10 lines)
- Section should prompt for: questions that emerged, areas worth exploring, things that remain unclear

**Proposed template addition (before Session Metadata):**
```markdown
---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- [Question 1 - why it's interesting]
- [Question 2 - why it's interesting]

**Areas worth exploring further:**
- [Area 1]
- [Area 2]

**What remains unclear:**
- [Uncertainty 1]
- [Uncertainty 2]

*(If nothing emerged, note: "Straightforward session, no unexplored territory")*
```

**Things to watch out for:**
- ⚠️ Agents may skip section if not prompted clearly
- ⚠️ Questions may be too vague without context
- ⚠️ Could clutter synthesis if every session tries to find questions

**Areas needing further investigation:**
- Whether this should be mandatory vs optional
- If investigation skill's Leave it Better pattern should mention this

**Success criteria:**
- ✅ Orchestrator finds actionable questions in SYNTHESIS.md after 5+ sessions
- ✅ At least 1 follow-up spawned from captured questions (like orch-go-ws4z example)
- ✅ No complaints about section adding noise to synthesis

---

## Test Performed

**Test:** Analyzed real agent sessions to verify that post-synthesis reflection creates value.

**Method:**
1. Read SYNTHESIS.md from architect session `og-arch-synthesize-findings-investigations-21dec`
2. Identified post-synthesis reflection pattern
3. Verified new epic (orch-go-ws4z) was created from that reflection
4. Counted 6 child issues spawned from Dylan's follow-up questions

**Result:** Concrete evidence that orchestrator reviewing completed work and asking follow-up questions produces new work items. The value is in capturing what emerged, not in changing the execution process.

---

## Self-Review

- [x] Real test performed (analyzed actual agent sessions, not just code review)
- [x] Conclusion from evidence (based on concrete SYNTHESIS.md examples)
- [x] Question answered (minimal change is artifact-level)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Leave it Better

`kn decide "Reflection value comes from orchestrator review + follow-up, not execution-time process changes" --reason "Evidence: post-synthesis reflection with Dylan created orch-go-ws4z epic (6 children) from captured questions"`

---

## References

**Files Examined:**
- `.orch/templates/SYNTHESIS.md` - Current template structure
- `cmd/orch/review.go` - Review command implementation
- `cmd/orch/resume.go` - Resume workflow
- `.orch/workspace/og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md` - Real example with post-synthesis reflection
- `pkg/spawn/context.go` - SPAWN_CONTEXT template generation
- `pkg/verify/check.go` - Synthesis parsing logic

**Commands Run:**
```bash
# Find workspaces with follow-up patterns
rg -l "spawn-follow-up|unexplored|questions" .orch/workspace/

# List recent workspaces
ls -la .orch/workspace/ | head -30

# Examine synthesis examples
cat .orch/workspace/og-arch-synthesize-findings-investigations-21dec/SYNTHESIS.md
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Taxonomy that includes SYNTHESIS.md as essential artifact
- **Workspace:** `.orch/workspace/og-arch-synthesize-findings-investigations-21dec/` - Example of reflection value

---

## Investigation History

**2025-12-21 16:30:** Investigation started
- Initial question: Should reflection checkpoint be skill/spawn/protocol/artifact level?
- Context: Observation that agent's most valuable output comes from interactive follow-up

**2025-12-21 16:45:** Found key evidence
- Discovered SYNTHESIS.md template already has spawn-follow-up recommendation
- Found real example of post-synthesis reflection creating new epic

**2025-12-21 17:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Artifact-level change (SYNTHESIS.md section) is minimal change that captures value
