# Session Synthesis

**Agent:** og-debug-bd-show-returns-25dec
**Issue:** orch-go-bgei
**Duration:** 2025-12-25 20:36 → 2025-12-25 20:50
**Outcome:** success

---

## TLDR

Fixed `orch complete` and `orch abandon` failing to parse epic child beads IDs (dot notation like `orch-go-ph1.9`). Root cause: `bd show --json` returns an array, not a single object, and the `FallbackShow` function expected a single object.

---

## Delta (What Changed)

### Files Modified
- `pkg/beads/client.go:422-440` - `FallbackShow` now unmarshals `bd show` output as array and returns first element
- `pkg/beads/types.go:109-122` - `Issue.Dependencies` changed from `[]string` to `json.RawMessage` to handle nested Issue objects
- `pkg/beads/client_test.go:394-445` - Added `TestBdShowArrayFormat` to verify array parsing

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- `bd show orch-go-bgei --json` returns `[{...}]` (array) not `{...}` (object) - verified via direct CLI execution
- Epic children like `orch-go-ph1.9` include full `dependencies` objects with `dependency_type` field, not just string IDs
- Error "json: cannot unmarshal array into Go value of type beads.Issue" occurred at `FallbackShow` (client.go:431)

### Tests Run
```bash
# All beads package tests pass
go test ./pkg/beads/... -v
# PASS: 15/15 tests including new TestBdShowArrayFormat

# Full test suite passes
go test ./...
# ok - all packages pass

# Smoke test with epic child ID
./orch complete orch-go-ph1.9 --force
# Issue orch-go-ph1.9 is already closed in beads (no JSON error!)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-25-inv-bd-show-returns-array-epic.md` - Full investigation with D.E.K.N. summary

### Decisions Made
- Use `json.RawMessage` for Dependencies instead of typed slice because:
  1. bd show returns full Issue objects with dependency_type
  2. No current code uses the Dependencies field
  3. Forward compatible with any format changes

### Constraints Discovered
- `bd show --json` always returns array format, even for single issue
- Epic children include parent Issue in dependencies with `"dependency_type": "parent-child"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-bgei`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- RPC daemon's show operation response format - might also return array (but wasn't blocking)

**What remains unclear:**
- Whether RPC client's `Show` method needs same fix (currently working, may get different format from daemon)

*(Minor uncertainty, not blocking completion)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Claude
**Workspace:** `.orch/workspace/og-debug-bd-show-returns-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-bd-show-returns-array-epic.md`
**Beads:** `bd show orch-go-bgei`
