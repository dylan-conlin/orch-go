# Session Synthesis

**Agent:** og-inv-knowledge-promotion-paths-21dec
**Issue:** orch-go-4kwt.2
**Duration:** 14:01 → 14:30
**Outcome:** success

---

## TLDR

Investigated how knowledge flows from project to global level. Found 4 documented promotion paths with CLI support for 2 (`kb promote`, `kb publish`). Current usage shows very low promotion rate (39 kn entries → 1 kb decision), which appears intentional for curation over accumulation.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Full investigation of knowledge promotion mechanisms

### Files Modified
- None

### Commits
- (pending) - Add investigation: knowledge promotion paths

---

## Evidence (What Was Observed)

- `kb promote kn-1d93ad --dry-run` successfully generates decision from kn entry
- Global `~/.kb/` has guides and principles but NO decisions directory
- orch-go has 39 kn entries but only 1 kb decision
- Investigation → Decision has no CLI support (fully manual workflow)
- Promotion triggers are documented in ~/.claude/CLAUDE.md but not enforced

### Tests Run
```bash
# Test kb promote mechanism
kb promote kn-1d93ad --dry-run --kn-dir /Users/dylanconlin/Documents/personal/orch-go/.kn
# SUCCESS: Generates decision template with kn content

# Test kb publish help
kb publish --help
# SUCCESS: Shows publish to ~/.kb/ mechanism

# Count kn entries
kn decisions  # 22 decisions
kn constraints  # 9 constraints
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Complete analysis of promotion paths and mechanisms

### Decisions Made
- No change recommended - observe first to determine if low promotion rate is intentional

### Constraints Discovered
- Investigation → Decision requires manual orchestrator effort (no `kb promote-investigation`)
- Global ~/.kb/decisions/ doesn't exist - decisions stay at project level

### Externalized via `kn`
- N/A - Straightforward investigation, captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - Investigation file written
- [x] Tests passing - CLI mechanisms tested
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-4kwt.2`

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-knowledge-promotion-paths-21dec/`
**Investigation:** `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md`
**Beads:** `bd show orch-go-4kwt.2`
