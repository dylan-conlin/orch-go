# Session Synthesis

**Agent:** og-arch-design-orch-rework-17feb-e3ed
**Issue:** orch-go-1025
**Duration:** 2026-02-17T22:53 → 2026-02-17T23:15
**Outcome:** success

---

## Plain-Language Summary

Designed the `orch rework` command that closes the gap between completion review finding problems and getting those problems fixed. Currently when an orchestrator finds gaps in agent work (e.g., agent claimed "Phase: Complete" but didn't actually remove deprecated flags), the workaround is to manually spawn a fresh agent with verbose rework instructions — losing the connection to the original work. The design creates a single command (`orch rework <beads-id> "feedback"`) that reopens the beads issue, creates a new workspace with the prior SYNTHESIS.md embedded, and spawns a fresh agent with structured rework context. This makes rework traceable (event logging), measurable (rework rate metrics), and ergonomic (one command instead of manual context assembly).

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Design investigation produced at `.kb/investigations/2026-02-17-design-orch-rework-command.md`
- Probe produced at `.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md`
- 7 decision forks navigated with substrate-backed recommendations
- Implementation plan with file targets and acceptance criteria

---

## TLDR

Designed `orch rework <beads-id> "feedback"` — creates new workspace with prior SYNTHESIS context, reopens beads issue with rework comment, and logs `agent.reworked` event for metrics. Reuses 90% of spawn pipeline. Key design decisions: new workspace (not session resume), rework context embedded in SPAWN_CONTEXT.md (not separate file), reopen same beads issue (not new issue).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-17-design-orch-rework-command.md` — Full design investigation with 7 fork navigations, implementation plan, and acceptance criteria
- `.kb/models/completion-verification/probes/2026-02-17-rework-loop-design-for-verification-gaps.md` — Probe extending the completion verification model with rework path
- `.orch/workspace/og-arch-design-orch-rework-17feb-e3ed/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-orch-rework-17feb-e3ed/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (design-only task)

### Commits
- (to be committed)

---

## Evidence (What Was Observed)

- `complete_cmd.go` verification pipeline has no rework path — Block/Failed escalation is a dead-end
- `archiveWorkspace()` preserves all workspace files including .beads_id, SYNTHESIS.md
- `beads.FallbackUpdate(id, "open")` exists but is never called anywhere (reopening is unused)
- `resume.go` handles paused agents (same session) — fundamentally different from rework (wrong work)
- SPAWN_CONTEXT template in `pkg/spawn/context.go` has extensible `contextData` struct
- `AGENT_MANIFEST.json` preserves skill, model, and tier — enabling rework to inherit original config
- No `agent.reworked` event type exists in `pkg/events/logger.go`

---

## Knowledge (What Was Learned)

### Design Decisions Made
1. **New workspace** (not resume) — because Session Amnesia means fresh agent needs full context
2. **Rework context in SPAWN_CONTEXT.md** — because Surfacing Over Browsing says bring state to agent
3. **TLDR + Delta inline, full path for deep dive** — Progressive Disclosure
4. **Reopen same beads issue** — maintains full comment history and connection
5. **Both event + beads tracking** — Observation Infrastructure
6. **Positional feedback arg** — consistent with `orch spawn`, `orch send` patterns
7. **Inherit original skill/model from AGENT_MANIFEST.json** — with override flags

### Key Distinction: rework vs resume
- `orch resume`: agent paused mid-work → sends message to EXISTING session
- `orch rework`: agent completed but work wrong → creates NEW session with feedback

### Completion Verification Model Extension
- The escalation model needs a sixth outcome: EscalationRework
- Currently Block/Failed are dead-ends requiring manual intervention

---

## Next (What Should Happen)

**Recommendation:** close (design complete, ready for implementation)

### Implementation Task
**Issue:** Create `orch rework` command per this design
**Skill:** feature-impl
**Context:**
```
Implement orch rework per design at .kb/investigations/2026-02-17-design-orch-rework-command.md.
Create cmd/orch/rework_cmd.go, pkg/spawn/rework.go, extend pkg/spawn/context.go and pkg/events/logger.go.
```

### If Close
- [x] Design investigation produced with 7 navigated forks
- [x] Probe produced extending completion verification model
- [x] Implementation plan with file targets and acceptance criteria
- [x] SYNTHESIS.md completed
- [x] Ready for `orch complete orch-go-1025`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- **Auto-rework in daemon:** Should the daemon automatically trigger rework when specific gate failures occur? What's the max attempt limit? (Deferred to Phase 2 in design)
- **Rework chains:** If rework #2 also fails, should rework #3 get context from both prior attempts or just the most recent? (Recommend: most recent + rework count)
- **Rework for orchestrator sessions:** Design only covers worker agents. Orchestrator session rework is a different pattern (SESSION_HANDOFF vs SYNTHESIS).
- **orch send as lightweight rework:** For small fixes where the session is still alive, `orch send` could serve as an alternative. Should `orch rework` detect live sessions and offer `orch send` instead?

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-orch-rework-17feb-e3ed/`
**Investigation:** `.kb/investigations/2026-02-17-design-orch-rework-command.md`
**Beads:** `bd show orch-go-1025`
