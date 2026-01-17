# Session Synthesis

**Agent:** og-arch-design-agent-self-17jan-daa6
**Issue:** orch-go-oawlf
**Duration:** 2026-01-17 10:00 → 2026-01-17 11:15
**Outcome:** success

---

## TLDR

Designed three-layer Agent Self-Health Context Injection system: (1) coaching.ts metrics detection with 4 new worker-specific signals, (2) Pain-as-Signal tool-layer injection mechanism using proven noReply pattern, (3) tiered recovery protocol for stuck agents (auto-resume → surface → human decision). Architecture leverages existing infrastructure for minimal risk.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Full design document for Agent Self-Health Context Injection system

### Files Modified
- None (design-only session)

### Commits
- Pending commit of investigation file

---

## Evidence (What Was Observed)

- `plugins/coaching.ts:70-76` - 8 metric types already implemented (action_ratio, analysis_paralysis, frame_collapse, behavioral_variation, circular_pattern, dylan_signal_prefix, compensation_pattern, priority_uncertainty)
- `plugins/coaching.ts:697-710` - noReply pattern for context injection already working
- `plugins/coaching.ts:988-1036` - Worker detection via `.orch/workspace/` path pattern
- `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Tiered recovery design already established with advisory-first principle
- `pkg/spawn/kbcontext.go:96-142` - KB context injection pattern at spawn time

### Tests Run
```bash
# Design session - no tests run
# Verified findings by reading source files
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Complete design for self-health system

### Decisions Made
- Decision 1: Use plugin-based approach (extend coaching.ts) rather than centralized health service - because infrastructure already exists and is proven
- Decision 2: Add 4 new worker-specific metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) - because workers need different signals than orchestrators
- Decision 3: Phased implementation (metrics → spawn context → runtime injection → daemon recovery) - because each phase delivers standalone value

### Constraints Discovered
- Token estimation must be approximate initially (no direct OpenCode API for actual token counts)
- Worker detection via path pattern has edge cases that need validation
- Resume can perpetuate bad state - requires strict rate limiting (1/hour/agent)

### Externalized via `kn`
- N/A (design session - no kn entries needed)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (N/A - design session)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-oawlf`

**Implementation follow-ups (for orchestrator to spawn):**
1. Phase 1: Extend coaching.ts with worker health metrics
2. Phase 2: Add health context to SPAWN_CONTEXT.md template
3. Phase 3: Add real-time health injection
4. Phase 4: Add daemon recovery loop

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How accurate can token estimation be without OpenCode API integration?
- Should "Needs Attention" be a dashboard section or inline status indicator?
- What's the actual stuck agent rate in production (would validate need for automation)?

**Areas worth exploring further:**
- OpenCode token counting API - could provide accurate context usage
- Phase change detection without beads comment parsing
- Git status monitoring for commit gap warnings

**What remains unclear:**
- Resume success rate for different failure modes (untested)
- Optimal thresholds (10min stuck, 80% context, 3 tool failures) - need production data

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-agent-self-17jan-daa6/`
**Investigation:** `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md`
**Beads:** `bd show orch-go-oawlf`
