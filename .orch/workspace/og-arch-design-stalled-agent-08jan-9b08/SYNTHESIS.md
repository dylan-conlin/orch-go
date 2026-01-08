# Session Synthesis

**Agent:** og-arch-design-stalled-agent-08jan-9b08
**Issue:** orch-go-y4kbm
**Duration:** 2026-01-08 09:00 → 10:15
**Outcome:** success

---

## TLDR

Designed minimal stalled agent detection: ONE signal (phase unchanged for 15+ minutes), ONE threshold, advisory-only surfacing in Needs Attention. Avoids complexity trap that caused Dec 27-Jan 2 spiral.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` - Full architect design with problem framing, 3 approaches explored, synthesis, and implementation recommendations

### Files Modified
- `.orch/features.json` - Added feat-044 for stalled detection implementation

### Commits
- (pending) `architect: design stalled agent detection - phase-based with 15-min threshold, advisory only`

---

## Evidence (What Was Observed)

- Post-mortem `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` documents failure from complexity (multiple thresholds, multiple states, auto-abandon)
- Investigation `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md:77` shows ~21 abandonments were "stuck at Planning for 19+ minutes" - phase stagnation is a clear signal
- events.jsonl shows abandonment patterns: "Stuck at Planning for 19+ minutes", "Session stuck in Planning for 3 attempts"
- Dead detection (3-min heartbeat) already implemented at `cmd/orch/serve_agents.go:406-441` - stalled is orthogonal
- Phase parsing infrastructure exists at `pkg/verify/beads_api.go:114-133`
- NeedsAttention component exists at `web/src/lib/components/needs-attention/needs-attention.svelte`

### Design Validation
- Reviewed 3 approaches: phase-based (recommended), activity-based, token-based
- Phase-based wins on simplicity (50-75 LOC) vs activity-based (150-200 LOC) vs token-based (300+ LOC)
- 15-minute threshold based on "19+ minutes stuck" being a common abandonment reason

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` - Full design document with D.E.K.N. summary

### Decisions Made
- **ONE threshold, ONE signal**: 15 minutes of phase unchanged = stalled (avoids complexity)
- **Advisory only**: Surface in Needs Attention, don't auto-abandon (human decides)
- **Phase-based signal**: Use phase change as progress indicator (simpler than file edits or token usage)
- **Reuse existing components**: Add to NeedsAttention, don't create new UI

### Constraints Discovered
- Dead ≠ Stalled: Dead = no heartbeat (3 min silence), Stalled = has heartbeat but not progressing
- Untracked spawns won't benefit (no beads comments to track phase)
- Pre-phase-report stalls already handled by `orch doctor` "stalled sessions" check

### Externalized via `kn`
- N/A - Design decision captured in investigation file with "Promote to Decision: recommend-yes"

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-y4kbm`

### Follow-up Work
**Issue:** feat-044 in features.json
**Skill:** feature-impl
**Context:**
```
Implement stalled agent detection per design in .kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md.
Key: Add PhaseReportedAt to beads status, calculate isStalled flag when phase unchanged for 15+ minutes,
surface in Needs Attention component. ~50-75 lines of new code.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should 15-minute threshold be configurable via environment variable or constant? (Design says constant is fine)
- How to handle agents that never report any phase? (Already handled by orch doctor)

**What remains unclear:**
- Whether 15 minutes is the optimal threshold (may need tuning after production use)
- Edge cases for agents legitimately in long phases (e.g., large file analysis)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-stalled-agent-08jan-9b08/`
**Investigation:** `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md`
**Beads:** `bd show orch-go-y4kbm`
