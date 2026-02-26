# Session Synthesis

**Agent:** og-debug-skillc-deploy-exits-25feb-0016
**Issue:** orch-go-1255
**Outcome:** success

---

## TLDR

`skillc deploy` always exited 0 even when individual skills failed to compile or deploy, making CI gating impossible. Fixed by tracking failure counts and calling `os.Exit(1)` when any failures occurred.

---

## Delta (What Changed)

### Files Modified
- `cmd/skillc/main.go` (in skillc repo) - Added `failCount` and `agentFailCount` tracking to `handleDeploy()`. Each error `continue` path now increments `failCount`. Agent deploy failures increment `agentFailCount`. Added `os.Exit(1)` at end of function when either counter is non-zero.

---

## Evidence (What Was Observed)

- `handleDeploy()` (main.go:1618-1835) loops over `.skillc` dirs with `continue` on errors but never tracks failure count
- `successCount` tracked but only used in summary output, never for exit code
- `agentCount` tracked but agent failures silently ignored
- After fix: broken skill.yaml causes exit code 1, all-success causes exit code 0

### Tests Run
```bash
# Partial failure test (1 bad yaml + 1 good skill)
./build/skillc deploy --target "$TMPDIR/target" "$TMPDIR/src"
# Result: Deployed 1/2, EXIT CODE: 1 ✓

# All-success test (2 good skills)
./build/skillc deploy --target "$TMPDIR/target" "$TMPDIR/src"
# Result: Deployed 2/2, EXIT CODE: 0 ✓

# Real source tree (11 skills)
./build/skillc deploy --target /tmp/skillc-test-deploy ~/orch-knowledge/skills/src
# Result: Deployed 11/11, EXIT CODE: 0 ✓

# Combined --target + --agent-dir with failure
./build/skillc deploy --target "$TMPDIR/target" --agent-dir "$TMPDIR/agents" "$TMPDIR/src"
# Result: Deployed 1/2 + 1/2 agents, EXIT CODE: 1 ✓
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `skillc` repo is at `~/Documents/personal/skillc/`, separate from orch-go
- Pre-existing test issue: `main_test.go:741` calls `runCheck` with wrong arity (unrelated to this fix)
- Dependencies listed in skill.yaml don't cause compile failures if missing — they're resolved at deploy time from the source tree

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Fix verified via smoke tests
- [x] Binary builds cleanly
- [x] Existing deploy (11/11 skills) unaffected

---

## Unexplored Questions

- The pre-existing `runCheck` test arity mismatch in `main_test.go:741` should be fixed separately
- Token budget warnings (3 skills exceed budget) are non-blocking — should they contribute to exit code with a `--strict` flag?

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-skillc-deploy-exits-25feb-0016/`
**Beads:** `bd show orch-go-1255`
