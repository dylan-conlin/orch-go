# Session Synthesis

**Agent:** og-arch-bd-cli-slow-07jan-196d
**Issue:** orch-go-loev7
**Duration:** 2026-01-07 14:30 → 2026-01-07 15:15
**Outcome:** success

---

## TLDR

Fixed bd CLI 5s slowdown in launchd/minimal environments by setting `BEADS_NO_DAEMON=1` in all Fallback* CLI subprocess calls, ensuring direct mode is used when falling back from RPC to CLI.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-design-bd-cli-slow-launchd-env.md` - Root cause analysis and recommendation

### Files Modified
- `pkg/beads/client.go` - Added `setupFallbackEnv()` helper and applied to all 12 Fallback* functions
- `pkg/verify/beads_api.go` - Added `os` import and BEADS_NO_DAEMON=1 to FallbackCommentsWithDir

### Commits
- (pending commit after this synthesis)

---

## Evidence (What Was Observed)

- Interactive shell has `BEADS_NO_DAEMON=1` set (verified: `env | grep BEADS`)
- Without env var, bd CLI takes 10s (5s daemon timeout warning + wait): `env -i HOME=$HOME PATH=$PATH bd ready --json` 
- With env var, bd CLI takes 86ms: `env -i HOME=$HOME PATH=$PATH BEADS_NO_DAEMON=1 bd ready --json`
- Beads daemon_autostart.go:47-53 explicitly checks for this env var as escape hatch
- orch-go Fallback* functions weren't setting cmd.Env, inheriting launchd's minimal env

### Tests Run
```bash
# Build verification
go build ./...
# PASS

# Package tests
go test ./pkg/beads/... -v -count=1
# PASS: 37 tests

go test ./pkg/verify/... -v -count=1
# PASS: all tests
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-design-bd-cli-slow-launchd-env.md` - Complete root cause analysis with D.E.K.N. summary

### Decisions Made
- Use `setupFallbackEnv()` helper to set BEADS_NO_DAEMON=1 because: When we're already in fallback mode (RPC failed), retrying daemon via CLI is semantically incorrect and adds 5s latency

### Constraints Discovered
- Already documented in kb: "bd CLI requires full environment; runs 50x slower in minimal env (launchd)"
- This investigation provided the fix to work around that constraint

### Externalized via `kn`
- Not applicable - constraint already documented, this implements the fix

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + code fix)
- [x] Tests passing (go test ./pkg/beads/... and ./pkg/verify/...)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-loev7`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should orch-go set BEADS_NO_DAEMON=1 at serve startup for the entire process? (Current fix is per-command, which is more surgical)

**What remains unclear:**
- Whether launchd environment affects RPC client path (seems fine based on current fallback behavior)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-bd-cli-slow-07jan-196d/`
**Investigation:** `.kb/investigations/2026-01-07-design-bd-cli-slow-launchd-env.md`
**Beads:** `bd show orch-go-loev7`
