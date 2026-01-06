# Session Synthesis

**Agent:** og-debug-fix-beads-database-22dec
**Issue:** orch-go-m0hm
**Duration:** 2025-12-22 06:31 → 2025-12-22 06:45
**Outcome:** partial

---

## TLDR

Fixed beads database integrity by deleting 235 orphaned dependencies and 31 orphaned labels referencing non-existent `bd-*` prefixed issues from cross-repo contamination. The original error is resolved (`bd import` works), but `bd sync` has a separate bug incorrectly parsing compound prefixes.

---

## Delta (What Changed)

### Files Modified
- `.beads/beads.db` - Deleted 235 orphaned rows from `dependencies` table, 31 from `labels` table
- `.beads/issues.jsonl` - Regenerated via `bd export` (no content change, just timestamp)

### Commits
- `15fc2d5` - fix: clean up beads database - removed orphaned dependencies and labels

---

## Evidence (What Was Observed)

- Original error: `foreign_keys_valid: found 235 orphaned dependencies (issue_id not in issues)`
- Orphaned dependencies had `bd-*` prefix (e.g., `bd-05a8`, `bd-2vh3.2`) from beads repo
- Database contains valid issues with `orch-go-*` (278) and `kb-cli-*` (17) prefixes
- The `bd-*` issues never existed in this repo - cross-repo contamination
- After fix: `bd import` works successfully
- `bd sync` still fails with prefix parsing bug (separate issue)

### Database Queries Run
```sql
-- Found 235 orphaned issue_id references
SELECT COUNT(*) FROM dependencies d WHERE NOT EXISTS (SELECT 1 FROM issues i WHERE i.id = d.issue_id);
-- 235

-- Found 31 orphaned labels  
SELECT COUNT(*) FROM labels l WHERE NOT EXISTS (SELECT 1 FROM issues i WHERE i.id = l.issue_id);
-- 31

-- After fix: both return 0
```

### Fix Applied
```sql
DELETE FROM dependencies WHERE NOT EXISTS (SELECT 1 FROM issues i WHERE i.id = dependencies.issue_id) OR NOT EXISTS (SELECT 1 FROM issues i WHERE i.id = dependencies.depends_on_id);
-- Deleted 235 rows

DELETE FROM labels WHERE NOT EXISTS (SELECT 1 FROM issues i WHERE i.id = labels.issue_id);
-- Deleted 31 rows
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Deleted orphaned records (cannot restore `bd-*` issues - they belong to different repo)
- Created backup before modification: `.beads/beads.db.backup-20251222-063250`

### Constraints Discovered
- Beads sync has a separate prefix-parsing bug: treats `orch-go-*` as `orch-*` prefix
- Multi-repo beads config (`repos.additional`) can cause cross-contamination

### Discovered Work Item
- **New Bug:** `bd sync` incorrectly parses compound prefixes like `orch-go-` as just `orch-`
  - Error: `prefix mismatch detected: database uses 'orch-go-' but found issues with prefixes: [kb-cli- (17 issues) orch- (34 issues)]`
  - The 34 "orch-" are actually all `orch-go-*` issues
  - Direct `bd import` works, only sync fails

---

## Next (What Should Happen)

**Recommendation:** close (with caveat)

### If Close
- [x] Orphaned dependencies removed (235)
- [x] Orphaned labels removed (31)
- [x] `bd import` works
- [x] Database backup created
- [x] Committed fix
- [ ] `bd sync` still fails (SEPARATE BUG - not in scope)

### Follow-up Issue Needed

**Issue:** bd sync prefix parsing bug with compound prefixes
**Symptom:** `bd sync` reports "orch- (34 issues)" when all issues are actually `orch-go-*`
**Root cause:** Likely splitting on first hyphen instead of full prefix pattern
**Workaround:** Use `bd import` and `bd export` directly, skip `bd sync`
**Repo:** beads (not orch-go)

---

## Unexplored Questions

**Questions that emerged during this session:**
- How did cross-repo contamination happen? (Multiple beads daemons? Config sync issue?)
- Should beads enforce foreign key constraints at write time to prevent this?
- Is the `repos.additional` config causing issues?

**What remains unclear:**
- Exact mechanism of cross-repo dependency contamination
- Whether this affects other repos with similar multi-repo configs

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-fix-beads-database-22dec/`
**Beads:** `bd show orch-go-m0hm`
