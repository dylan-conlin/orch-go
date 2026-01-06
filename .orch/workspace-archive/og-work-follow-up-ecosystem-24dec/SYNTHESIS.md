# Session Synthesis

**Agent:** og-work-follow-up-ecosystem-24dec
**Issue:** orch-go-cnq5
**Duration:** 2025-12-24 17:30 → 2025-12-24 18:10
**Outcome:** success

---

## TLDR

Evaluated whether beads should remain as external dependency vs alternatives (fork, GitHub Issues, internal tracker). Concluded Phase 3 abstraction layer is the right approach - alternatives have high cost with low/negative benefit due to beads' unique dependency-first design.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-follow-up-ecosystem-audit-orch.md` - Full investigation with option analysis

### Files Modified
- None

### Commits
- No code changes (investigation/decision session)

---

## Evidence (What Was Observed)

- 19 `exec.Command("bd", ...)` call sites in orch-go use only 7 commands (comment, comments, create, list, ready, show, stats)
- 1,192 beads issues exist with 36.7h average lead time - workflow is established
- 307 skill references to beads commands across ~/.claude/skills/
- GitHub Issues lacks dependency-first design (no `ready` equivalent, manual dependency links)
- Existing decision record (2025-12-21) already resolved fork/upstream question

### Tests Run
```bash
# Beads CLI usage analysis
grep -r "exec.Command.*bd" --include="*.go"  # 19 matches

# Feature verification
bd --help  # 30+ commands, only 7 used

# Current state
bd stats --json  # 1,192 issues total
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-follow-up-ecosystem-audit-orch.md` - Beads dependency strategy analysis

### Decisions Made
- Decision: Keep beads as external dependency with abstraction layer because replacement costs vastly exceed abstraction costs and beads' dependency-first design has no equivalent in alternatives

### Constraints Discovered
- Beads' `bd ready` (issues with resolved blockers) has no GitHub Issues equivalent - core to orch workflow
- Only 7 of 30+ beads commands are used - interface surface is narrow

### Externalized via `kn`
- Will run: `kn decide "Keep beads as external dependency with abstraction layer" --reason "7-command interface surface is narrow; dependency-first design (ready queue, dep graph) has no equivalent in alternatives; Phase 3 abstraction addresses API stability risk at low cost"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (investigation, no code changes)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-cnq5`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should the pkg/beads interface include comment parsing helpers (currently in verify package)?
- Could beads' JSONL format be directly parsed to avoid CLI overhead for high-frequency operations?

**Areas worth exploring further:**
- Performance impact of shelling out to bd vs direct JSONL parsing
- Test coverage strategy for daemon with mock beads interface

**What remains unclear:**
- Long-term beads maintenance commitment from stevey
- Whether unused beads features (graph, sync) might become valuable

---

## Session Metadata

**Skill:** design-session
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-work-follow-up-ecosystem-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-follow-up-ecosystem-audit-orch.md`
**Beads:** `bd show orch-go-cnq5`
