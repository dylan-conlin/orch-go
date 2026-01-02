# Session Synthesis

**Agent:** og-debug-headless-spawns-not-22dec
**Issue:** orch-go-ttbc
**Duration:** 2025-12-22 12:15 → 2025-12-22 12:45
**Outcome:** success

---

## TLDR

Fixed a bug where headless-spawned agents couldn't be found by their beads ID because `runTail()` and `runQuestion()` used a naive directory name lookup instead of the correct `findWorkspaceByBeadsID()` function that scans SPAWN_CONTEXT.md.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/main.go` - Fixed `runTail()` and `runQuestion()` to use `findWorkspaceByBeadsID()` instead of naive `strings.Contains(entry.Name(), beadsID)` lookup
- `cmd/orch/main_test.go` - Added documentation comment explaining headless spawn discoverability scenario

### Files Created
- `.kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md` - Investigation documenting root cause and fix

### Commits
- (pending) `fix: use findWorkspaceByBeadsID for headless spawn discoverability`

---

## Evidence (What Was Observed)

- Workspace names are generated as `og-{prefix}-{task-slug}-{date}` without beads ID (`pkg/spawn/config.go:55-81`)
- `runTail()` at lines 415-428 used `strings.Contains(entry.Name(), beadsID)` which never matches
- `findWorkspaceByBeadsID()` at lines 2015-2063 correctly scans SPAWN_CONTEXT.md for beads ID
- `resolveSessionID()` correctly uses `findWorkspaceByBeadsID()`, but `runTail()`/`runQuestion()` duplicated broken logic

### Tests Run
```bash
# All tests pass
go test ./...
# ok  	github.com/dylan-conlin/orch-go/cmd/orch	0.154s

# Smoke test works
go build -o /tmp/orch-test ./cmd/orch
/tmp/orch-test tail orch-go-ttbc
# === Output from og-debug-headless-spawns-not-22dec (via API, last 50 lines) ===
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md` - Root cause analysis and fix documentation

### Decisions Made
- Use `findWorkspaceByBeadsID()` everywhere we need to look up workspace by beads ID, instead of duplicating lookup logic

### Constraints Discovered
- Workspace names are task-based, not beads-based - the beads ID only lives in SPAWN_CONTEXT.md
- Tmux spawns accidentally worked because window names contain `[beads-id]`, masking the bug

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file complete
- [x] Ready for `orch complete orch-go-ttbc`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-headless-spawns-not-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md`
**Beads:** `bd show orch-go-ttbc`
