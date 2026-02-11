# Session Synthesis

**Agent:** og-arch-simplification-architecture-review-10feb-8d87
**Issue:** orch-go-qv699
**Duration:** 2026-02-10
**Outcome:** success

---

## TLDR

Comprehensive architecture review of orch-go complexity. Identified that current 14 gates, 5-layer zombie defense, and model-specific bypasses are accidental complexity from patching failures at system boundaries. Recommends 4-phase simplification: (1) remove dead code, (2) fix critical boundaries (commit gate, process lifecycle), (3) reduce to core 5 gates, (4) shift to supervised-first with daemon as batch mode.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-10-design-simplification-architecture-review.md` - Full architecture review with decision forks and recommendations

### Files Modified
- None (investigation-only session)

### Commits
- Pending (to be committed with this synthesis)

---

## Evidence (What Was Observed)

### From Ghost Completions Investigation (2026-02-09)
- No "commit exists" gate in 14 verification gates
- GPT model bypass at `check.go:809-825` auto-passes Phase:Complete
- `git_diff` gate conflates working tree with committed history
- Result: 22 issues closed with zero committed code

### From Zombie Process Investigation (2026-02-10)
- Process ledger (`~/.orch/process-ledger.jsonl`) is 0 bytes
- Orphan detection pattern `"run --attach"` doesn't match current `opencode attach`
- OpenCode `Session.remove()` doesn't kill attached bun processes
- All 5 layers of zombie defense are non-functional

### From DYLANS_THOUGHTS.org
- "this system is just too fragile as is"
- Need for "diagnostic/firefighting mode"
- "Setting a 30 minute reap timer that destroys my primary UI is such a careless and thoughtless thing"

### From Open Issues Backlog
- 30+ open issues spanning spawn, daemon, dashboard, cli areas
- Multiple P1/P2 bugs related to system reliability
- Pattern: complexity created to patch failures, not address root causes

---

## Verification Contract

- **Spec:** Not applicable (design investigation, no code changes)
- **Key outcomes:**
  - Investigation file created with complete analysis
  - 4 decision forks identified with substrate-backed recommendations
  - 4-phase implementation plan with concrete actions

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-10-design-simplification-architecture-review.md` - Architecture simplification analysis

### Decisions Made (Recommendations)
1. **Operating mode:** Supervised-first with daemon as batch mode (reduces blast radius)
2. **Verification gates:** Reduce to core 5 (phase_complete, commit_evidence, synthesis, test_evidence, git_diff)
3. **Process lifecycle:** Fix at both boundaries (orch + OpenCode)
4. **CLAUDE.md:** Extract to guides, reduce to ~100 line orientation

### Key Insights
- **Essential vs Accidental:** Dual-mode (tmux+HTTP), worktrees, beads integration are essential. Model-specific bypasses, empty process ledger, coaching plugins are accidental.
- **Wrong boundaries:** orch owns session, OpenCode owns process — neither terminates on delete. This is root cause of zombie accumulation.
- **Pattern:** Complexity created to compensate for silent failures. Fix should make failures visible, not add more layers.

### Constraints Discovered
- Cannot remove dual-mode architecture — confirmed correct by prior decision
- Cannot eliminate verification entirely — need some gates for quality
- Supervised-first shift requires stability validation first

---

## Issues Created

**Discovered work tracked during this session:**

No new issues created — existing backlog already covers the identified work:
- `orch-go-w4pj9` — OpenCode boundary fix (process termination)
- `orch-go-cmdfh` — Audit ghost completion work loss
- `orch-go-6v2ta` — OpenCode crashes

The investigation provides the strategic framing for this existing work.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-qv699`

### Recommended Follow-up (for orchestrator)

1. **Immediate (Phase 0):** Remove dead code
   - Remove coaching plugin code (already disabled)
   - Remove model-specific bypass profiles
   - Update orphan detection pattern

2. **Priority (Phase 1):** Fix critical boundaries
   - Add GateCommitEvidence — prevents ghost completions
   - Fix orch complete tmux window kill ordering
   - Add startup sweep

3. **After Phase 1 proven (Phase 2):** Simplify verification
   - Reduce to core 5 gates
   - Remove model-specific bypasses

4. **After stability (Phase 3-4):** Documentation and operational shift
   - Simplify CLAUDE.md
   - Shift to supervised-first

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

1. **OAuth token detection** — Tokens disappeared without detection, caused silent fallback to pay-per-token. No gate exists for this.

2. **Scope limits for automation** — 30-minute reaper killed Dylan's UI. What scope limits should exist for automated cleanup?

3. **Model behavioral profiles** — Should we invest in per-model behavior profiles, or just require all models to meet baseline behavior (commit, report Phase:Complete)?

**What remains unclear:**

- Whether all 22 ghost completion issues had work that can be recovered
- Whether OpenCode upstream would accept the Session.remove() process termination change
- Optimal number of verification gates (recommended 5, but could be 3)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/worktrees/og-arch-simplification-architecture-review-10feb-8d87/`
**Investigation:** `.kb/investigations/2026-02-10-design-simplification-architecture-review.md`
**Beads:** `bd show orch-go-qv699`
