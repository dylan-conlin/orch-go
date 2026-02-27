# Session Synthesis

**Agent:** og-debug-hotspot-exclude-opencode-26feb-3790
**Issue:** orch-go-1117
**Outcome:** success

---

## Plain-Language Summary

The accretion gate in `pkg/verify/accretion.go` was missing directory exclusions for `.opencode/`, `.orch/`, `.beads/`, and other non-source directories. While the hotspot bloat scanner (`cmd/orch/hotspot.go`) was already fixed (via orch-go-1229) to exclude these directories from `filepath.Walk` and git log analysis, the accretion verification — which checks if agents added too many lines to large files — still counted `.opencode/plugin/` files as source code. This meant modifying deployed plugin copies could trigger false accretion warnings/errors during `orch complete`. The fix extracts a shared `nonSourceDirs` list and uses it in the `isSourceFile()` function, aligning accretion verification with the hotspot scanner's exclusions.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for test commands and expected outcomes.

---

## Delta (What Changed)

### Files Modified
- `pkg/verify/accretion.go` - Added `nonSourceDirs` slice with 12 non-source directory patterns; refactored `isSourceFile()` to use it instead of inline vendor/node_modules/dist/build checks
- `pkg/verify/accretion_test.go` - Added 8 new test cases for `.opencode/plugin/`, `.orch/`, `.beads/`, `.svelte-kit/`, `__pycache__/`, `.next/`, `.nuxt/`, `.output/` paths

---

## Evidence (What Was Observed)

- `.opencode/plugin/coaching.ts` (1570 lines) and `.opencode/plugin/slow-find-warn.ts` are git-tracked in this project
- `orch hotspot --json` correctly excludes `.opencode/` files (fix from orch-go-1229 via `skipBloatDirs` and `buildOutputPrefixes`)
- But `pkg/verify/accretion.go` `isSourceFile()` only excluded `vendor/`, `node_modules/`, `dist/`, `build/` — missing 8 other non-source directories
- Git log shows `.opencode/plugin/coaching.ts` appears in recent commits (3 in last 28 days), meaning it would be caught by accretion verification if modified

### Tests Run
```bash
go test ./pkg/verify/ -run 'TestVerifyAccretion|TestIsSourceFile|TestGetFileLineCount' -v
# PASS: 15 tests passing (1.730s)
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Used a shared `nonSourceDirs` slice rather than duplicating the inline conditions — reduces drift between the exclusion list and `hotspot.go`'s `buildOutputPrefixes`
- Added "Keep in sync" comments referencing `cmd/orch/hotspot.go` for future maintainers

### Constraints Discovered
- Concurrent agent modifications in `pkg/verify/check.go`, `pkg/orch/extraction.go`, and untracked test files prevent full package compilation — had to temporarily isolate conflicting files to run tests

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (15/15 accretion + isSourceFile tests)
- [x] Ready for `orch complete orch-go-1117`

---

## Unexplored Questions

- `plugins/` (source directory at project root) appears as a legitimate bloat hotspot at 1743 lines — should this directory also be excluded from hotspot scanning, or is it correctly flagged as needing extraction?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-hotspot-exclude-opencode-26feb-3790/`
**Beads:** `bd show orch-go-1117`
