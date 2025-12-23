# Session Synthesis

**Agent:** og-inv-beads-multi-repo-22dec
**Issue:** orch-go-djpb
**Duration:** 2025-12-22 18:10 → 18:50
**Outcome:** success

---

## TLDR

Investigated why `bd show` fails for cross-repo IDs after multi-repo sync. Found **two root causes**: (1) stale bd binary writing repos to database instead of YAML (causing `GetMultiRepoConfig()` to miss additional repos), and (2) database corruption in kb-cli blocking operations. After rebuilding bd from commit 634c0b93, multi-repo works correctly with 784 beads issues imported.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-beads-multi-repo-hydration-why.md` - Full investigation with findings and recommendations

### Files Modified
- None (investigation only)

### Commits
- `a75db54` - Investigation: beads multi-repo hydration works correctly

---

## Evidence (What Was Observed)

**Stale Binary Issue (Primary Finding):**
- Before rebuild: `cat .beads/config.yaml` showed only `repos.primary: "."`, no `additional` array
- `bd config list` showed malformed data: `repos.additional = {"/path":"/path"}` (map instead of array)
- Commit 634c0b93 (2025-12-21) fixed this: "bd repo add wrote repos config to the database config table, but GetMultiRepoConfig() reads from YAML only"
- After rebuild: Config correctly shows `additional: ["/path/to/beads"]` and 784 issues imported

**Architecture:**
- Multi-repo hydration imports issues into SQLite database via `HydrateFromMultiRepo()` in `/Users/dylanconlin/Documents/personal/beads/internal/storage/sqlite/multirepo.go:22`
- `bd show` queries database via `store.GetIssue()`, not JSONL files (show.go:254-261)
- YAML is sole source of truth for multi-repo config; database config table is legacy

**Additional Finding:**
- kb-cli database corrupted: "found 235 orphaned dependencies" (blocking all operations)

### Tests Run
```bash
# Rebuild bd binary with fix
cd ~/Documents/personal/beads && go build -o ~/go/bin/bd ./cmd/bd

# Add repo and sync (in orch-go)
cd ~/Documents/personal/orch-go
bd repo add /Users/dylanconlin/Documents/personal/beads
bd repo sync

# Verify cross-repo resolution works
bd show bd-8507
# SUCCESS: Shows "Publish bd-wasm to npm" from beads repo

# Verify database counts
sqlite3 .beads/beads.db "SELECT COUNT(*) FROM issues WHERE source_repo = '/Users/dylanconlin/Documents/personal/beads'"
# 784 issues imported

# Verify config written correctly
cat .beads/config.yaml
# Shows additional: ["/Users/dylanconlin/Documents/personal/beads"]
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-beads-multi-repo-hydration-why.md` - Investigation with D.E.K.N. summary, findings, confidence assessment, and recommendations

### Decisions Made
- Multi-repo feature is working as designed - no code changes needed
- Reported failure was environment-specific (kb-cli database corruption)
- Database health is a prerequisite for multi-repo sync

### Constraints Discovered
- **Binary must be current for multi-repo features** - Fix commit 634c0b93 changed YAML-writing behavior; stale binary = broken multi-repo
- **YAML is sole source of truth** - `GetMultiRepoConfig()` reads from viper (YAML), never from database config table
- Multi-repo sync requires healthy database (no orphaned dependencies)

### Externalized via `kn`
- `kn decide "Multi-repo hydration requires healthy database" --reason "Orphaned dependencies in kb-cli blocked all database operations including multi-repo sync. Fix with 'bd doctor --fix' before attempting multi-repo setup."` - kn-741ba1

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - Investigation file created with D.E.K.N. summary
- [x] Tests passing - Test repo successfully imported 323 issues and resolved cross-repo IDs
- [x] Investigation file has `**Phase:** Complete` - Status updated
- [x] Ready for `orch complete orch-go-djpb`

**Follow-up actions for orchestrator:**
1. Fix kb-cli database corruption: `cd ~/Documents/personal/kb-cli && bd doctor --fix`
2. Verify multi-repo works in kb-cli after fix
3. Consider documenting database health prerequisite in beads README or `bd repo sync --help`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does daemon mode fail with "unexpected end of JSON input" when parsing repos config? (--no-daemon workaround used)
- Does mtime caching work correctly across multiple syncs? (only tested single sync)
- What's the performance impact of syncing very large repos (1000+ issues)? (only tested with 323 issues)

**Areas worth exploring further:**
- Daemon mode behavior with multi-repo sync
- Edge cases with symlinks, relative paths, or non-standard repo structures
- Automated health check before `bd repo sync` to prevent corruption issues

**What remains unclear:**
- Root cause of kb-cli database corruption (how did 235 orphaned dependencies accumulate?)
- Whether there are other edge cases where multi-repo sync might fail

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-inv-beads-multi-repo-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-beads-multi-repo-hydration-why.md`
**Beads:** `bd show orch-go-djpb`
