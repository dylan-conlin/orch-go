# Session Synthesis

**Agent:** og-inv-audit-orchestration-lifecycle-22dec
**Issue:** orch-go-sjo1
**Duration:** ~45 minutes
**Outcome:** success

---

## TLDR

Audited the orchestration lifecycle post-registry-removal and found 4 independent state sources (OpenCode sessions, tmux windows, beads issues, workspaces) with no single authoritative source. Recommended beads-centric reconciliation approach where beads issue status is authoritative and other sources provide liveness evidence.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` - Comprehensive audit findings with D.E.K.N. summary

### Files Modified
- None (investigation only)

### Commits
- (pending - will commit investigation file)

---

## Evidence (What Was Observed)

- `orch status --json` shows 26 agents combining tmux + OpenCode without perfect deduplication (`cmd/orch/main.go:1550-1683`)
- `orch clean --dry-run` identifies 102 cleanable workspaces but 90 have SYNTHESIS.md (some counted via closed beads) (`cmd/orch/main.go:2083-2153`)
- `orch complete` relies on beads Phase: Complete, not agent liveness (`pkg/verify/check.go:323-368`)
- Registry removal was documented in `pkg/opencode/service.go:100-105` with reference to investigation

### Tests Run
```bash
orch status --json | head -50  # Shows 26 agents, some may be phantom
orch clean --dry-run | head -30  # Shows 102 cleanable workspaces
ls .orch/workspace/*/SYNTHESIS.md | wc -l  # 90 completed
tmux list-sessions | grep worker  # 9 worker sessions
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` - Full lifecycle audit

### Decisions Made
- Decision: Recommend beads-centric reconciliation because beads is persistent, external, and already tracks phase progress

### Constraints Discovered
- Constraint: No single source of truth exists post-registry; each command queries different state subsets
- Constraint: SYNTHESIS.md is sufficient for completion but not necessary (Phase: Complete can work without it)

### Externalized via `kn`
- (will run `kn decide` after synthesis)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with D.E.K.N.)
- [x] Tests passing (investigation tested commands)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-sjo1`

### Follow-up Work (Optional)
If orchestrator agrees with recommendations:
1. Create beads issue for implementing `IsLive()` function
2. Create beads issue for updating `orch status` liveness checks
3. Create beads issue for adding warnings to `orch complete`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does `orch daemon` handle state conflicts when spawning multiple agents?
- What happens when an agent has Phase: Complete but tmux window is still running?
- Should `orch review` also show liveness indicators?

**Areas worth exploring further:**
- Headless spawns may have different lifecycle issues (no tmux window)
- OpenCode disk sessions vs in-memory sessions - when does each matter?

**What remains unclear:**
- Full edge case matrix (spawn crash, mid-work crash, phase-reported-but-crashed)
- Whether current phantom agent count (26) significantly impacts concurrency limits

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-inv-audit-orchestration-lifecycle-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md`
**Beads:** `bd show orch-go-sjo1`
