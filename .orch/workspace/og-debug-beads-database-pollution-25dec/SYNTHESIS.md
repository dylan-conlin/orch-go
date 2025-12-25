# Session Synthesis

**Agent:** og-debug-beads-database-pollution-25dec
**Issue:** orch-go-mazg (originally bd-37bw - polluted)
**Duration:** 2025-12-25 12:46 → 2025-12-25 13:05
**Outcome:** success

---

## TLDR

Beads database pollution in orch-go was caused by `config.yaml` having `additional: [beads]` which imported 787 bd-* and 18 kb-cli-* issues. Fixed by filtering issues.jsonl to only orch-go-* prefixed issues and reinitializing the database.

---

## Delta (What Changed)

### Files Created
- `.beads/issues.jsonl.backup` - Backup of original before fix
- `.beads/issues.jsonl.polluted` - Backup of polluted version

### Files Modified
- `.beads/issues.jsonl` - Cleaned from 1303 lines to 498 (orch-go-* only)
- `.beads/config.yaml` - Removed `additional: [beads]` reference
- `.beads/.gitignore` - Added nested `.beads/`, `*.jsonl.backup`, `*.jsonl.polluted`, `interactions.jsonl`

### Files Deleted
- `.beads/.beads/.gitignore` - Nested pollution artifact
- `.beads/.beads/issues.jsonl` - Nested pollution artifact

### Commits
- `5fdb0ca` - fix: clean beads database pollution from cross-repo config

---

## Evidence (What Was Observed)

- issues.jsonl had 1303 lines: 498 orch-go-*, 787 bd-*, 18 kb-cli-* (jq analysis)
- Git history shows `additional: ["/Users/.../beads"]` added in commit 38e79ef
- Nested `.beads/.beads/` directory was git-tracked (artifact of beads repo sync)
- Original spawn issue `bd-37bw` was itself polluted (wrong prefix)
- kb-cli has similar issue: 235 orphaned dependencies from cross-repo config

### Tests Run
```bash
# Verify clean database
$ bd stats
Total Issues: 498
Open: 55

# Verify only orch-go prefix
$ bd list --json | jq -r '.[].id' | cut -d'-' -f1-2 | sort -u
orch-go
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-beads-database-pollution-orch-go.md` - Root cause analysis and fix documentation

### Decisions Made
- Filter-and-reinitialize approach: Clean JSONL, remove nested dirs, reinit database (vs trying to delete individual issues)
- Update .gitignore preventatively: Block nested .beads/, backup files from future git tracking

### Constraints Discovered
- Beads multi-repo config (`additional:`) imports ALL issues from referenced repos - potentially dangerous without guardrails
- Issue prefix indicates ownership - bd-* = beads repo, orch-go-* = orch-go repo, kb-cli-* = kb-cli repo
- Spawning issues during cross-repo pollution inherits wrong prefix (bd-37bw was created with bd- instead of orch-go-)

### Externalized via `kn`
- N/A (existing constraint already captured: "Beads cross-repo contamination can create orphaned FK references")

---

## Next (What Should Happen)

**Recommendation:** close + spawn-follow-up

### If Close
- [x] All deliverables complete (issues.jsonl clean, db reinitialized, gitignore updated)
- [x] Tests passing (bd stats shows 498 orch-go-* issues)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-mazg`

### Spawn Follow-up
**Issue:** Clean kb-cli beads pollution
**Skill:** systematic-debugging
**Context:**
```
kb-cli/.beads/config.yaml has `additional: [orch-go]`. Database shows 235 orphaned dependencies. 
JSONL is clean (18 kb-cli issues) but database is corrupted. May need similar filter-and-reinitialize 
approach, or just `bd init --force --prefix kb-cli` since JSONL is already clean.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should beads CLI warn when `additional` config would import issues with different prefix?
- Is the beads multi-repo `additional` config feature documented? What's the intended use case?

**Areas worth exploring further:**
- Filing upstream beads issue about multi-repo config documentation/warnings
- Checking other repos for similar cross-repo config pollution

**What remains unclear:**
- Whether beads upstream considers importing cross-repo issues a feature or a bug

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-debug-beads-database-pollution-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-beads-database-pollution-orch-go.md`
**Beads:** `bd show orch-go-mazg`
