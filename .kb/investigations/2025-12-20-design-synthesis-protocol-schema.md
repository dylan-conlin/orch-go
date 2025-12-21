**TLDR:** Question: How should workers externalize session state for 30-second orchestrator handoff? Answer: Create SYNTHESIS.md with structured sections: Delta (what changed), Evidence (what was observed), Knowledge (what was learned), Next (what to do). This solves session amnesia by capturing exactly what fresh Claude needs to understand agent work. High confidence (90%) - validated against existing patterns in beads, kb, and spawn context.

---

# Investigation: Synthesis Protocol Design for Agent Handoffs

**Question:** What standardized schema enables a "30-second handoff" from worker agents to orchestrator, solving session amnesia by externalizing Delta, Evidence, Knowledge, and Next Actions?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Alpha Opus architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)
**Commit:** b83f048

---

## Problem Framing

### The Session Amnesia Problem

When the orchestrator resumes after context exhaustion or switches focus, it faces cold start:
- No memory of what worker agents did
- Must reconstruct state from scattered artifacts
- Beads comments are progress breadcrumbs, not summary documents
- Investigation files capture question/answer, not session outcome
- SPAWN_CONTEXT.md is input, not output

### What "30-Second Handoff" Means

A fresh Claude (orchestrator) should be able to:
1. Read one file (SYNTHESIS.md)
2. Understand what changed during the session
3. Know what evidence supports conclusions
4. See what knowledge was created/updated
5. Understand what actions are needed next
6. All within 30 seconds of reading

### Success Criteria

- **Scannable**: Orchestrator can understand session outcome in 30 seconds
- **Complete**: Contains all information needed to continue/close the work
- **Structured**: Consistent format enables parsing/aggregation
- **Linked**: References artifacts (commits, files, issues) for drill-down
- **Actionable**: Clear next steps or completion confirmation

---

## Findings

### Finding 1: Beads provides permanent progress trail but not synthesis

**Evidence:** Beads comments capture phase transitions and milestones:
```
bd comment orch-go-vf2 "Phase: Planning - Analyzing codebase structure"
bd comment orch-go-vf2 "Phase: Implementing - Adding authentication middleware"
bd comment orch-go-vf2 "Phase: Complete - All tests passing, ready for review"
```

These are breadcrumbs during execution, not a post-session summary. The orchestrator must read all comments to reconstruct what happened, vs reading a single synthesis.

**Source:** `pkg/spawn/context.go:92-103` (beads progress tracking section)

**Significance:** Beads is the right place for real-time progress. SYNTHESIS.md fills the gap for post-session summary.

---

### Finding 2: Investigation files focus on question/answer, not session outcomes

**Evidence:** Investigation template (`kb create investigation`) has:
- TLDR (question + answer)
- Findings (evidence gathered)
- Synthesis (answer to question)
- Confidence assessment
- Implementation recommendations

This is excellent for knowledge capture but doesn't capture:
- What files were created/modified
- What commits were made
- What the session accomplished vs intended
- What the orchestrator should do next

**Source:** `.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md` template, `.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md` (completed example)

**Significance:** Investigations answer "what did we learn?" but not "what did the session produce?" or "what should happen next?". SYNTHESIS.md fills this gap.

---

### Finding 3: SPAWN_CONTEXT.md is input-only, no corresponding output artifact

**Evidence:** SPAWN_CONTEXT.md contains:
- Task description
- Authority boundaries
- Deliverables expectations
- Skill guidance
- Beads tracking info

After session completes, there's no structured output that mirrors this input. The orchestrator must:
1. Check beads for "Phase: Complete" comment
2. Find investigation file path from beads comment
3. Read investigation file
4. Check git for commits
5. Verify deliverables exist

**Source:** `pkg/spawn/context.go` (template), `pkg/spawn/config.go` (config struct)

**Significance:** There's a clear asymmetry: structured input, unstructured output. SYNTHESIS.md creates symmetry.

---

### Finding 4: Existing patterns to preserve and extend

**Evidence:** Key patterns from current system:

1. **Phase tracking (beads)**: `Phase: Planning/Implementing/Testing/Complete`
2. **Investigation path reporting**: `investigation_path: /path/to/file.md`
3. **Scope documentation**: Agents report scope in beads comments
4. **Workspace isolation**: `.orch/workspace/{name}/` per session
5. **TLDR pattern**: 1-2 sentence summary at top of investigations

These patterns work. SYNTHESIS.md should extend, not replace.

**Source:** `.beads/issues.jsonl` (comment patterns), `.kb/investigations/` (TLDR pattern)

**Significance:** SYNTHESIS.md should reuse these proven patterns, adding structure for session-level synthesis.

---

## Synthesis

**Key Insights:**

1. **Session amnesia is a structured output problem** - The system has structured input (SPAWN_CONTEXT.md) but unstructured output. Orchestrator must reconstruct session state from scattered artifacts (beads comments, git history, investigation files). A single synthesis artifact solves this.

2. **D.E.K.N. captures what fresh Claude needs** - Delta (what changed), Evidence (what supports it), Knowledge (what was learned), Next (what to do). This is the minimum viable structure for 30-second understanding.

3. **Extend existing patterns, don't replace** - Beads for real-time progress, investigations for knowledge capture, SYNTHESIS.md for session summary. Each has a distinct role.

**Answer to Investigation Question:**

Create SYNTHESIS.md in the workspace directory (`.orch/workspace/{name}/SYNTHESIS.md`) with four mandatory sections:

1. **Delta**: What changed in this session (files created/modified, commits made)
2. **Evidence**: What was observed that supports conclusions
3. **Knowledge**: What was learned and externalized (links to investigations, decisions)
4. **Next**: What the orchestrator should do (close issue, spawn follow-up, escalate)

The schema should:
- Be written by workers at session end, before `Phase: Complete`
- Be read by orchestrator to understand session outcome in 30 seconds
- Link to artifacts (beads issue, investigation file, commits) for drill-down
- Include explicit completion status and recommendation

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Analyzed all existing patterns (beads, kb, spawn context) and identified clear gap. The D.E.K.N. structure directly addresses session amnesia. High confidence because:
- Problem is clearly articulated (Finding 1-3)
- Solution extends existing patterns (Finding 4)
- Schema is minimal and focused

**What's certain:**

- ✅ Session amnesia is a real problem - orchestrator must reconstruct state from scattered artifacts
- ✅ Existing patterns (beads, investigations) serve distinct purposes and should not be replaced
- ✅ A structured output artifact in the workspace solves the synthesis gap

**What's uncertain:**

- ⚠️ Optimal verbosity level - how much detail in each section?
- ⚠️ Whether to mandate SYNTHESIS.md for all skills or just complex ones
- ⚠️ Integration with verification - should `orch complete` check for SYNTHESIS.md?

**What would increase confidence to Very High (95%+):**

- Test with real worker sessions across different skills
- Validate 30-second readability with actual orchestrator resumes
- Iterate on section structure based on usage patterns

---

## Implementation Recommendations

### Recommended Approach ⭐: D.E.K.N. Schema

**Create SYNTHESIS.md template** with four structured sections for session handoff.

**Why this approach:**
- Directly solves session amnesia (Finding 1-3)
- Extends existing patterns without replacing them (Finding 4)
- Minimal viable structure - can expand later based on usage

**Trade-offs accepted:**
- More work for agents at session end (acceptable for orchestrator benefit)
- Another file to maintain (acceptable - it's output, not config)

**Implementation sequence:**
1. Create SYNTHESIS.md template in `.orch/templates/`
2. Update SPAWN_CONTEXT.md to require SYNTHESIS.md before Phase: Complete
3. Optionally integrate with `orch complete` verification

### SYNTHESIS.md Schema ⭐

```markdown
# Session Synthesis

**Agent:** {workspace-name}
**Issue:** {beads-id}
**Duration:** {session-duration}
**Outcome:** {success | partial | blocked | failed}

---

## Delta (What Changed)

**Files Created:**
- `path/to/file.go` - Brief description

**Files Modified:**
- `path/to/existing.go` - What was changed

**Commits:**
- `abc1234` - Commit message summary

---

## Evidence (What Was Observed)

- Observation 1 with source reference
- Observation 2 with source reference
- Key finding that informed decisions

---

## Knowledge (What Was Learned)

**New Artifacts:**
- `.kb/investigations/YYYY-MM-DD-*.md` - Brief description

**Decisions Made:**
- Decision 1 with rationale

**Constraints Discovered:**
- Constraint 1 - Why it matters

---

## Next (What Should Happen)

**Recommendation:** {close | spawn-follow-up | escalate | resume}

**If close:**
- All deliverables complete
- Tests passing
- Ready for orchestrator verification

**If spawn-follow-up:**
- Issue: {new-issue-title}
- Skill: {recommended-skill}
- Context: {brief context for next agent}

**If escalate:**
- Question: {what needs decision}
- Options: {what was considered}
- Recommendation: {preferred option and why}

**If resume:**
- Next step: {what to do when resuming}
- Blocker: {what prevented completion}
```

### Alternative Approaches Considered

**Option B: Extend investigation file with synthesis section**
- **Pros:** Single artifact, reuses existing template
- **Cons:** Investigations are knowledge-focused, not session-focused. Mixing concerns.
- **When to use instead:** If every session produces an investigation

**Option C: Use beads closing comment for synthesis**
- **Pros:** Already integrated, no new files
- **Cons:** Comments are append-only, limited formatting. Can't update.
- **When to use instead:** For very simple sessions

**Rationale for recommendation:** SYNTHESIS.md is distinct from investigation (session vs knowledge) and beads (summary vs breadcrumbs). Separate file = separate concern.

---

### Implementation Details

**What to implement first:**
1. Create `.orch/templates/SYNTHESIS.md` with schema above
2. Update architect skill to require SYNTHESIS.md before Phase: Complete
3. Test with this session (meta-validation)

**Things to watch out for:**
- ⚠️ Keep sections concise - 30-second readability is the goal
- ⚠️ Delta section should be machine-parseable (for future automation)
- ⚠️ Next section is the most important - what should orchestrator DO?

**Areas needing further investigation:**
- Should simple skills (hello, quick tasks) require SYNTHESIS.md?
- Should `orch complete` parse SYNTHESIS.md to auto-extract next actions?
- How to handle multi-phase sessions (checkpoint syntheses)?

**Success criteria:**
- ✅ Orchestrator can understand session outcome in 30 seconds from SYNTHESIS.md
- ✅ All information needed to continue/close is in one file
- ✅ Schema is consistent across different skill types

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template structure
- `pkg/spawn/config.go` - Spawn configuration options
- `.beads/issues.jsonl` - Beads comment patterns and progress tracking
- `.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md` - Example completed investigation

**Commands Run:**
```bash
# Verify project location
pwd  # /Users/dylanconlin/Documents/personal/orch-go

# Create investigation file
kb create investigation design/synthesis-protocol-schema
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md` - Example of investigation pattern
- **Workspace:** `.orch/workspace/og-arch-alpha-opus-synthesis-20dec/` - This session's workspace

---

## Investigation History

**[2025-12-20 14:40]:** Investigation started
- Initial question: How to create standardized SYNTHESIS.md schema for 30-second orchestrator handoff?
- Context: Spawned by orchestrator to design synthesis protocol for solving session amnesia

**[2025-12-20 14:45]:** Problem framing complete
- Identified session amnesia as structured output problem
- Analyzed existing patterns (beads, investigations, spawn context)

**[2025-12-20 15:00]:** Schema design complete
- Created D.E.K.N. structure (Delta, Evidence, Knowledge, Next)
- Defined SYNTHESIS.md template with all required sections

**[2025-12-20 15:15]:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: SYNTHESIS.md schema designed with D.E.K.N. structure for 30-second orchestrator handoff
