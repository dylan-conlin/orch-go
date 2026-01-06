# Session Synthesis

**Agent:** og-inv-audit-all-registry-21dec
**Issue:** orch-go-untracked-1766356990
**Duration:** 2025-12-21
**Outcome:** success

---

## TLDR

Audited all 15 registry callsites in orch-go. Found that the registry is a caching layer, not a source of truth - all data can be derived from OpenCode API, tmux, and beads. Session_id capture during spawn is the critical dependency; a phased migration approach is recommended.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md` - Comprehensive investigation documenting all registry usage, alternative sources, and migration checklist

### Files Modified
- None (investigation-only task)

### Commits
- (pending commit with investigation file)

---

## Evidence (What Was Observed)

- 15 distinct registry callsites across cmd/orch/{main.go, review.go, resume.go} and pkg/daemon/daemon.go
- Registry operations fall into 4 categories: write (6), read by ID (5), list (5), reconcile (2)
- All registry data has alternative sources: OpenCode API provides session listing, tmux provides window discovery, beads provides issue lifecycle
- Existing fallback code in status/tail commands already works without registry (proves feasibility)
- Session_id capture during tmux spawn uses retry logic (FindRecentSessionWithRetry) - this is the hardest to replace

### Tests Run
```bash
# Analyzed existing code for fallback patterns
# main.go:1325-1389 - status command already enriches from tmux
# main.go:418-433 - tail command has tmux fallback
# All callsites mapped and categorized
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md` - Complete audit with migration checklist

### Decisions Made
- Phased migration recommended over big-bang removal because existing fallback code proves derived lookups work

### Constraints Discovered
- Session_id capture timing is critical - spawn creates OpenCode session but it's not immediately discoverable (500ms-2s window)
- Concurrency limiting (daemon/spawn) depends on fast active count lookup - derived approach adds latency

### Externalized via `kn`
- `kn constrain "Registry is caching layer, not source of truth - all data exists in OpenCode/tmux/beads" --reason "Investigation found all registry data can be derived from primary sources"` - Created kn-d8d6ec

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with migration checklist)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has D.E.K.N. summary filled
- [x] Ready for `orch complete orch-go-untracked-1766356990`

### Follow-up Work (Optional)
If migration is pursued, create beads issues for:
1. **Phase 1:** Migrate status/tail/question to use OpenCode+tmux directly (low risk)
2. **Phase 2:** Migrate complete/abandon/clean to use beads+OpenCode for state (medium risk)
3. **Phase 3:** Evaluate spawn command - keep session_id capture or use derived lookup (high risk)

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-audit-all-registry-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md`
**Beads:** `bd show orch-go-untracked-1766356990`
