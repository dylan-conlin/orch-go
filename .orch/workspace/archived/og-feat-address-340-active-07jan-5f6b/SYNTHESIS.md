# Session Synthesis

**Agent:** og-feat-address-340-active-07jan-5f6b
**Issue:** orch-go-b47oe
**Duration:** 2026-01-07 16:45 → 2026-01-07 17:00
**Outcome:** success

---

## TLDR

Investigated why 340+ workspaces accumulated: not a completion gap but an archival gap. 132 stale workspaces (>7 days old) are ready to archive via `orch clean --stale`, but this command isn't being run regularly. Recommend adding auto-archive to `orch complete` workflow.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md` - Investigation documenting root cause analysis

### Files Modified
- None (investigation only, no code changes)

### Commits
- None yet (investigation file to be committed)

---

## Evidence (What Was Observed)

- Total non-archived workspaces: 409 (was 340 in original report)
- With SYNTHESIS.md (completed): 141
- Meta-orchestrator (no synthesis expected): 9
- Regular agents without synthesis: 259 (many are light-tier which don't require SYNTHESIS.md)
- Age distribution: 256 from last 7 days (normal), 132 from 8-14 days (stale)
- `orch clean --stale --dry-run` confirms 132 workspaces ready to archive

### Key Finding
The cleanup mechanism exists (`orch clean --stale`) but isn't automated. `orch complete` closes issues but intentionally preserves workspaces for investigation reference. Without periodic cleanup, workspaces accumulate.

### Tests Run
```bash
# Verify stale cleanup would work
orch clean --stale --dry-run
# Output: Would archive 132 stale workspaces.
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md` - Full analysis with recommendations

### Decisions Made
- Workspace accumulation is an archival gap, not a completion gap
- Recommended approach: Add auto-archive to `orch complete` with `--no-archive` opt-out

### Constraints Discovered
- Workspaces are intentionally preserved after completion for investigation reference
- Auto-archive must respect orchestrator workspaces (different lifecycle)
- Light-tier workspaces don't have SYNTHESIS.md by design (still archivable based on age + .beads_id)

### Externalized via `kn`
- No new kn entries needed - findings documented in investigation file

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, implementation is separate issue)

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (dry-run verified cleanup works)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-b47oe`

### Immediate Action for Orchestrator
Run `orch clean --stale` to immediately clear 132 stale workspaces:
```bash
orch clean --stale
# Expected: Archives 132 workspaces
```

### Follow-up Work (New Issue)
**Issue:** Add auto-archive to orch complete workflow
**Skill:** feature-impl
**Context:**
```
After successful completion, move workspace to archived/ unless --no-archive specified.
Should skip orchestrator workspaces (different lifecycle).
References: .kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be a "recently archived" view in the dashboard? (Currently archived workspaces are hidden)
- Should archived workspaces be automatically deleted after N days? (Currently preserved indefinitely)

**Areas worth exploring further:**
- Whether daemon-based cleanup (periodic) would be better than completion-triggered cleanup
- Whether 7-day default threshold is optimal (could be configurable)

**What remains unclear:**
- How often `orch clean --stale` has been run historically (no audit trail)

---

## Session Metadata

**Skill:** feature-impl (investigation phase)
**Model:** opus
**Workspace:** `.orch/workspace/og-feat-address-340-active-07jan-5f6b/`
**Investigation:** `.kb/investigations/2026-01-07-inv-address-340-active-workspaces-completion.md`
**Beads:** `bd show orch-go-b47oe`
