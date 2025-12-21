<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session boundaries are defined by three distinct mechanisms: worker sessions (Phase: Complete → /exit), orchestrator sessions (context full → session-transition skill), and hybrid handoff sessions (SESSION_HANDOFF.md for cross-session continuity).

**Evidence:** Analyzed SPAWN_CONTEXT.md template (pkg/spawn/context.go:14-161), session-transition skill (~/.claude/skills/session-transition/SKILL.md), orchestrator skill, and SESSION_HANDOFF.md. Verified monitor.go:165-175 detects idle→complete transitions via SSE.

**Knowledge:** Worker session end is strictly protocol-driven (beads comment + exit). Orchestrator session end is state-detection driven (git status, workspace parsing). The gap is synthesis timing - currently triggered by orchestrator after worker reports Phase: Complete, not during worker session.

**Next:** Implement reflection checkpoint pattern (orch-go-4kwt.8) - add pause-before-complete for worker sessions to enable interactive follow-up before final synthesis.

**Confidence:** High (85%) - Codebase evidence strong; orchestrator session patterns less formalized than worker patterns.

---

# Investigation: Orchestrator Session Boundaries

**Question:** When should handoffs happen? What triggers synthesis? How do we detect session end?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Three Distinct Session Types with Different Boundary Patterns

**Evidence:** The orchestration system has three session types, each with different end detection:

| Session Type | Boundary Trigger | Handoff Mechanism | Synthesis Timing |
|--------------|------------------|-------------------|------------------|
| **Worker** | `bd comment "Phase: Complete"` + `/exit` | SPAWN_CONTEXT.md → SYNTHESIS.md | Before exit (required) |
| **Orchestrator** | Context full OR explicit end | session-transition skill | After agent completions |
| **Cross-session** | End of working day/session | SESSION_HANDOFF.md | End of major session |

**Source:** 
- Worker: `pkg/spawn/context.go:25-33` (SESSION COMPLETE PROTOCOL)
- Orchestrator: `~/.claude/skills/session-transition/SKILL.md:29-36` (Trigger section)
- Cross-session: `.orch/SESSION_HANDOFF.md` (manually created at session boundaries)

**Significance:** These three patterns are independent and not unified. Worker boundaries are strictly enforced via templates; orchestrator boundaries are state-detected; cross-session boundaries are manual.

---

### Finding 2: Worker Session End Detection is Multi-Layered

**Evidence:** Worker session completion detection uses four layers:

1. **Beads comments** - Agent reports `Phase: Complete` via `bd comment`
2. **OpenCode SSE** - Monitor.go:165-175 detects busy→idle transition
3. **SYNTHESIS.md existence** - `pkg/verify/check.go:267-280` verifies file exists
4. **Registry reconciliation** - `orch clean` checks all layers for consistency

**Source:** 
- `pkg/opencode/monitor.go:165-175` (completion detection logic)
- `pkg/verify/check.go:284-327` (VerifyCompletion function)
- `pkg/spawn/context.go:25-33` (agent instructions)

**Significance:** Worker session end is detected, not explicitly signaled to orchestrator. The orchestrator must poll/monitor for completions. This creates latency between agent finishing and orchestrator awareness.

---

### Finding 3: Synthesis Triggers are Orchestrator-Initiated, Not Agent-Initiated

**Evidence:** Two distinct synthesis patterns exist:

1. **Worker SYNTHESIS.md** - Created by agent before `/exit` (required by skill)
2. **Orchestrator synthesis** - Created after completing agents, combining results

Current flow:
```
Agent completes work → Agent creates SYNTHESIS.md → Agent calls /exit
↓
Orchestrator polls (orch wait / orch status) → Detects Phase: Complete
↓  
Orchestrator runs orch complete → Verifies SYNTHESIS.md exists
↓
Orchestrator synthesizes across agents (if multiple)
```

**Source:**
- `pkg/verify/check.go:315-324` (SYNTHESIS.md verification)
- Orchestrator skill lines 318-319 (Always Act: synthesize after completion)
- SESSION_HANDOFF.md (cross-agent synthesis after major work)

**Significance:** Synthesis is currently post-hoc. The reflection checkpoint pattern (orch-go-4kwt.8) proposes adding interaction before final synthesis: `Autonomous work → Human probe → Deeper synthesis`

---

### Finding 4: Session-Transition Skill Provides Orchestrator-Level State Detection

**Evidence:** The session-transition skill (534 lines) implements state assessment:

```
Detected state types:
- BLOCKED (workspace contains Blocking-Issue)
- COMPLETED (Phase: Complete + clean git)
- INVESTIGATION_IN_PROGRESS (uncommitted + investigation files)
- MID_IMPLEMENTATION (uncommitted + Phase: Implementing)
- CLEAN_UNKNOWN (clean git, no signals)
```

**Source:** `~/.claude/skills/session-transition/SKILL.md:98-122`

**Significance:** This provides a model for how orchestrator sessions detect their own boundaries. Could be adapted for worker sessions to self-detect when checkpointing is needed.

---

### Finding 5: Gap - No Unified Session Boundary Protocol

**Evidence:** Current gaps in session boundary handling:

1. **Worker→Orchestrator handoff timing** - Agent finishes, orchestrator discovers async
2. **Context exhaustion detection** - No automatic "context running low" trigger for workers
3. **Synthesis quality** - Agent creates SYNTHESIS.md under time pressure at end, not progressively
4. **Orchestrator session boundaries** - Manual via session-transition skill, not automatic

The reflection checkpoint pattern addresses #3 and partially #1 by adding interactive pause:
```
Agent autonomous work → Orchestrator probe ("what else?") → Agent deeper synthesis → Exit
```

**Source:**
- SESSION_HANDOFF.md pattern (manual reflection capture)
- orch-go-4kwt.8 (reflection checkpoint beads issue)
- session-synthesis-21dec.md line 67-73 (insight about interactive follow-up)

**Significance:** The system lacks a unified "session boundary approaching" signal. Workers complete abruptly; orchestrators handle boundaries manually.

---

## Synthesis

**Key Insights:**

1. **Worker boundaries are protocol-driven** - SPAWN_CONTEXT.md dictates exact sequence (commit → SYNTHESIS.md → bd comment → /exit). Detection is via beads comments + SSE monitoring.

2. **Orchestrator boundaries are state-driven** - session-transition skill detects state via git/workspace. Triggering is manual ("context is full"). No automatic detection.

3. **Synthesis is currently post-hoc** - Both worker (create at end) and orchestrator (combine after completions) create synthesis after work is done, not progressively.

**Answer to Investigation Question:**

**When should handoffs happen?**
- Workers: When work is complete (all deliverables ready, tests passing). Currently self-determined by agent.
- Orchestrator: When context is exhausted OR major phase completes. Currently manual trigger.
- Cross-session: End of working session or before context loss risk.

**What triggers synthesis?**
- Worker SYNTHESIS.md: Agent completion protocol (before /exit)
- Orchestrator synthesis: After `orch complete` verifies agent work
- Cross-session: Manual reflection at session end

**How do we detect session end?**
- Workers: `Phase: Complete` in beads comments + OpenCode idle status via SSE
- Orchestrator: session-transition skill state detection OR manual invocation
- OpenCode level: `monitor.go` tracks busy→idle transitions

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong codebase evidence for worker patterns (SPAWN_CONTEXT.md template is authoritative). Orchestrator patterns less formalized - relies on skill guidance which is more advisory than enforced.

**What's certain:**

- ✅ Worker session end protocol is defined in SPAWN_CONTEXT.md template
- ✅ Phase: Complete detection via beads comments is implemented in pkg/verify/
- ✅ SSE monitoring for idle detection exists in pkg/opencode/monitor.go
- ✅ SYNTHESIS.md verification is part of orch complete flow

**What's uncertain:**

- ⚠️ Orchestrator session boundaries are skill-guided, not enforced
- ⚠️ Context exhaustion is not automatically detected for workers
- ⚠️ Progressive synthesis (vs end-of-session) not currently implemented

**What would increase confidence to Very High (95%+):**

- Implement reflection checkpoint pattern and observe behavior
- Add automatic context-low detection for worker sessions
- Formalize orchestrator session boundaries in tooling (not just skill)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Phased Enhancement

**Why this approach:**
- Addresses highest-value gap first (reflection checkpoint)
- Progressive rollout minimizes risk
- Each phase delivers independent value

**Implementation sequence:**

1. **Reflection checkpoint pattern (orch-go-4kwt.8)** - Add `--interactive` flag to spawning that pauses before complete for orchestrator review
2. **Progressive synthesis** - Update SYNTHESIS.md template to encourage filling during work, not just at end
3. **Auto-detection** - Add context exhaustion warning to agents (message count threshold)

### Alternative Approaches Considered

**Option B: Unified session boundary protocol**
- **Pros:** Single pattern for all session types
- **Cons:** May over-engineer worker sessions which currently work well
- **When to use instead:** If worker completion rates drop significantly

**Option C: Remove SYNTHESIS.md requirement**
- **Pros:** Simpler agent protocol
- **Cons:** Loses structured handoff context for orchestrator
- **When to use instead:** Never - SYNTHESIS.md is critical for orchestrator review

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT template generation
- `pkg/verify/check.go` - Completion verification logic
- `pkg/opencode/monitor.go` - SSE session monitoring
- `~/.claude/skills/session-transition/SKILL.md` - Orchestrator transition handling
- `~/.claude/skills/policy/orchestrator/SKILL.md` - Orchestrator responsibilities
- `.orch/SESSION_HANDOFF.md` - Cross-session handoff artifact
- `.orch/workspace/session-synthesis-21dec.md` - Session reflection

**Commands Run:**
```bash
# Verified SPAWN_CONTEXT template content
rg "SESSION COMPLETE|Phase.*Complete|/exit" pkg/spawn/context.go

# Counted session references in opencode package
rg "session" --type go pkg/opencode/ -c

# Tested orch status command output
orch status

# Found SYNTHESIS.md files in workspaces
find .orch/workspace -name "SYNTHESIS.md" -type f
```

**Related Artifacts:**
- **Decision:** Pending - may promote reflection checkpoint to decision
- **Investigation:** This file
- **Beads Issue:** orch-go-4kwt.3 (parent epic: orch-go-4kwt)

---

## Self-Review

- [x] Real test performed (ran actual commands, examined actual files)
- [x] Conclusion from evidence (based on codebase analysis)
- [x] Question answered (all three sub-questions addressed)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-21 14:01:** Investigation started
- Initial question: When should handoffs happen? What triggers synthesis? How do we detect session end?
- Context: Part of Amnesia-Resilient Artifact Architecture epic (orch-go-4kwt)

**2025-12-21 14:20:** Found three distinct session types
- Worker, Orchestrator, Cross-session patterns identified
- SPAWN_CONTEXT.md template analyzed

**2025-12-21 14:35:** Completed codebase analysis
- Verified monitor.go completion detection
- Analyzed session-transition skill
- Identified reflection checkpoint pattern connection

**2025-12-21 14:45:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Session boundaries are defined by three mechanisms; gap is synthesis timing (post-hoc vs progressive)
