# Session Synthesis

**Agent:** og-debug-headless-spawn-registers-22dec
**Issue:** orch-go-ig16
**Duration:** 2025-12-22 22:31 → 2025-12-22 23:05
**Outcome:** success

---

## TLDR

Fixed headless spawn to register with the correct project directory by adding --workdir flag and using x-opencode-directory HTTP header instead of JSON body parameter.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md` - Investigation documenting root cause analysis and fix

### Files Modified
- `cmd/orch/main.go` - Added --workdir flag, validation, and projectDir resolution logic
- `pkg/opencode/client.go` - Fixed CreateSession to use x-opencode-directory header

### Commits
- `9568983` - fix: add --workdir flag and use x-opencode-directory header for correct project registration

---

## Evidence (What Was Observed)

- Initial test: All sessions registered with /Users/dylanconlin/Documents/personal/orch-go regardless of intended target (verified via curl http://127.0.0.1:4096/session)
- Root cause: CreateSession sent directory in JSON body, but OpenCode API expects x-opencode-directory HTTP header
- Verification: Direct curl test with header showed directory: /tmp/another-test-project in response (cmd/orch/main.go:958)
- Testing post-fix: Session ses_4b5fc749affeKdIyhE6CioIb26 created with correct directory /tmp/another-test-project (stored in global projectID)

### Tests Run
```bash
go test ./pkg/opencode/... -v
# PASS: all tests passing (40+ tests)

go test ./cmd/orch/... -v  
# PASS: all tests passing

# Manual smoke tests:
orch spawn --workdir /tmp/another-test-project --no-track investigation "test"
# Result: Session created with correct directory in session file
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md` - Documents missing --workdir flag and OpenCode API header requirement

### Decisions Made
- Decision 1: Add --workdir flag instead of requiring cd to change directory - improves UX for cross-project spawning
- Decision 2: Validate workdir path exists and is a directory - prevents cryptic errors later
- Decision 3: Use filepath.Abs() to resolve relative paths - allows flexible path specification

### Constraints Discovered
- OpenCode API requires x-opencode-directory header, not JSON body parameter - inconsistent with other parameters but required for correct behavior
- Sessions created with x-opencode-directory may get projectID "global" instead of directory hash - observed but doesn't affect functionality

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (--workdir flag implemented and tested)
- [x] Tests passing (go test ./... successful)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-ig16`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does OpenCode use x-opencode-directory header instead of JSON body for directory? Inconsistent with title/model parameters.
- Why do some sessions get projectID "global" vs directory hash? Doesn't seem to cause issues but worth understanding.
- Should we add --workdir to tmux mode spawn as well? Currently only affects headless/inline modes.

**Areas worth exploring further:**
- Unified parameter passing for OpenCode API (headers vs JSON body consistency)
- Cross-project dashboard filtering with --workdir (verify it works end-to-end)

**What remains unclear:**
- OpenCode API contract for session directory - is x-opencode-directory header documented?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-headless-spawn-registers-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md`
**Beads:** `bd show orch-go-ig16`
