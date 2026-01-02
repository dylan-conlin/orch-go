# Session Synthesis

**Agent:** og-inv-cross-project-epic-21dec
**Issue:** orch-go-pp7l
**Duration:** 2025-12-21 ~18:30 → 2025-12-21 ~19:30
**Outcome:** success

---

## TLDR

Investigated how to handle epics spanning multiple repositories (orch-go + kb-cli). Recommend Option A (ad-hoc spawns + manual coordination) as the working pattern today. Beads multi-repo features exist but have bugs.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` - Full investigation with findings, options analysis, and recommendations

### Files Modified
- None

### Commits
- Will commit investigation file with this synthesis

---

## Evidence (What Was Observed)

- `bd show orch-go-ivtg` works in orch-go, fails in kb-cli ("no issue found") - confirms per-repo isolation
- `bd repo add` fails with JSON parsing error even after setting valid repos config (beads bug)
- `beads/docs/MULTI_REPO_HYDRATION.md` documents that hydration is read-only aggregation, not cross-repo relationships
- Existing orchestrator guidance covers cross-repo spawning but not cross-repo epics

### Tests Run
```bash
# Test cross-repo visibility
cd ~/Documents/personal/kb-cli && bd show orch-go-ivtg
# Error: no issue found matching "orch-go-ivtg"

# Test multi-repo config
bd config set repos '{"primary": ".", "additional": []}'
bd repo add ~/Documents/personal/kb-cli "kb-cli"
# Error: failed to parse repos config: unexpected end of JSON input
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` - Complete analysis of cross-project epic patterns

### Decisions Made
- Use Option A for cross-project epics: epic in primary repo, ad-hoc spawns with `--no-track` in secondary repos, manual `bd close` with commit references

### Constraints Discovered
- Beads issues are strictly per-repository
- Multi-repo hydration is read-only aggregation, not cross-repo relationships
- `bd repo` commands have JSON parsing bugs in v0.29.0

### Externalized via `kn`
- `kn decide "Cross-project epics use Option A..."` - kn-43aa5e
- `kn tried "Beads multi-repo config via bd repo add" --failed "JSON parsing error..."` - kn-399392

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Tests performed (bash commands documented)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-pp7l`

### Follow-up Actions (for orchestrator)

1. **Add cross-project epic pattern to orchestrator skill** - Document the concrete workflow discovered here
2. **File beads bug** - `bd repo` commands fail with JSON parsing error
3. **File beads feature request** - Cross-repo issue references (Option D)
4. **Test pattern on orch-go-ivtg** - Use Option A for phases 2-4 implementation

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to surface cross-project epic progress in a unified view?
- Could the daemon be enhanced to poll multiple repos for a meta-epic?

**Areas worth exploring further:**
- Whether beads `bd repo sync` would enable better workflows after bugs are fixed
- How other multi-repo orchestration systems handle this (Linear, Jira cross-project)

**What remains unclear:**
- Exact effort required for Option D (beads enhancement)
- Long-term scalability of Option A with many cross-project epics

---

## Session Metadata

**Skill:** investigation
**Model:** opus (Claude)
**Workspace:** `.orch/workspace/og-inv-cross-project-epic-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`
**Beads:** `bd show orch-go-pp7l`
