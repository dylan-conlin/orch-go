# Session Synthesis

**Agent:** og-inv-debug-investigation-tab-08jan-de93
**Issue:** orch-go-4gxrr
**Duration:** 2026-01-08
**Outcome:** success

---

## TLDR

Debugged and fixed the Investigation tab showing wrong files for cross-project agents. Root cause: auto-discovery used unreliable session directory (orchestrator's cwd) instead of target project directory when workspace cache lookup failed. Fix: guard auto-discovery with `hasReliableProjectDir` flag.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_agents.go` - Added `hasReliableProjectDir` guard to prevent investigation auto-discovery when workspace cache doesn't provide reliable project directory

### Files Created
- `.kb/investigations/2026-01-08-inv-debug-investigation-tab-shows-wrong.md` - Full investigation documenting root cause and fix

### Commits
- `d5763345` - investigation: debug-investigation-tab-shows-wrong - checkpoint
- (pending) - fix: guard investigation auto-discovery with hasReliableProjectDir

---

## Evidence (What Was Observed)

- `agents[i].ProjectDir` initialized from `s.Directory` (session directory) at line 466
- For cross-project agents with `--workdir`, session directory is orchestrator's cwd due to OpenCode `--attach` bug
- `beadsProjectDirs[beadsID]` lookup only overwrites ProjectDir when workspace cache has entry (line 780-782)
- `discoverInvestigationPath` uses ProjectDir to search `.kb/investigations/` (line 792)
- When workspace cache lookup fails, wrong project's investigations are searched

### Tests Run
```bash
# Build verification
go build ./cmd/orch/...
# SUCCESS

# Test verification  
go test ./cmd/orch/... -v -count=1
# PASS: 52 tests passed
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-debug-investigation-tab-shows-wrong.md` - Investigation documenting the cross-project artifact path confusion bug

### Decisions Made
- Guard with `hasReliableProjectDir` instead of skipping auto-discovery entirely: Minimal fix that preserves happy path for agents with workspace cache entries

### Constraints Discovered
- Session directory from OpenCode is unreliable for cross-project agents due to `--attach` bug
- Workspace cache is the source of truth for PROJECT_DIR

### Externalized via `kn`
- (none needed - tactical bug fix, not architectural pattern)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-4gxrr`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does OpenCode `--attach` use server cwd instead of the specified `--workdir`? (This is the root cause that makes session directory unreliable)

**Areas worth exploring further:**
- Could we fix the OpenCode `--attach` bug upstream to make session directory reliable?
- Are there other places in serve_agents.go that assume session directory is correct?

**What remains unclear:**
- Live dashboard behavior not tested (would require orchestrator restart and real cross-project agent)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-debug-investigation-tab-08jan-de93/`
**Investigation:** `.kb/investigations/2026-01-08-inv-debug-investigation-tab-shows-wrong.md`
**Beads:** `bd show orch-go-4gxrr`
