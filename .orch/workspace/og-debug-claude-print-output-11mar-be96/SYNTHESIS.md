# Session Synthesis

**Agent:** og-debug-claude-print-output-11mar-be96
**Issue:** orch-go-ssbwv
**Duration:** 2026-03-11T19:26 → 2026-03-11T19:45
**Outcome:** partial (governance file blocks worker from applying hook patch)

---

## Plain-Language Summary

The `enforce-phase-complete.py` Stop hook fires on ALL Claude Code session exits, including nested `claude --print` calls made from within spawned workers. When a spawned worker runs `claude --print "test prompt"`, the child process inherits `ORCH_SPAWNED=1` and `CLAUDE_CONTEXT=worker` from the parent's environment. The hook thinks the child IS the spawned worker, blocks its exit (no Phase: Complete reported), and the block message creates a 2nd conversation turn that contaminates the `--print` output. The fix is a 3-line addition to the hook: check for `ORCH_PRINT_MODE=1` env var and skip enforcement. The orchestrator must apply the hook patch since it's governance-protected.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key test: `go test ./cmd/orch/ -run TestEnforcePhaseCompleteSkipsPrintMode`

---

## TLDR

Root cause: env var inheritance. `claude --print` inherits `ORCH_SPAWNED=1` from parent worker, Stop hook blocks it, output gets contaminated. Fix: 3-line patch to hook + `ORCH_PRINT_MODE=1` env var for callers. Tests written (1 failing = demonstrates bug). `run-trials.sh` already fixed.

---

## Delta (What Changed)

### Files Created
- `cmd/orch/enforce_phase_complete_test.go` - 5 tests for the Stop hook behavior, including bug reproduction test
- `.orch/workspace/og-debug-claude-print-output-11mar-be96/VERIFICATION_SPEC.yaml` - Exact patch and verification

### Files Modified
- `evidence/2026-03-06-human-calibration/run-trials.sh` - Added `export ORCH_PRINT_MODE=1` to prevent hook contamination

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **Root cause confirmed**: `enforce-phase-complete.py` line 45-50 checks `ORCH_SPAWNED == "1"` and `CLAUDE_CONTEXT == "worker"`. These are exported by `pkg/spawn/claude.go:142` (`export ORCH_SPAWNED=1; export CLAUDE_CONTEXT=worker`). Child processes inherit them.
- **Prior workarounds tried (from probe 2026-03-11)**: CLAUDECODE unset, CLAUDE_CONFIG_DIR override, stream-json output, JSON output — all failed because they didn't address the actual env vars (`ORCH_SPAWNED`, `CLAUDE_CONTEXT`)
- **Claude Code Stop hook input**: No `print_mode` or `session_type` field in the JSON input (confirmed via Claude Code docs research). Detection must use env vars.
- **Governance protection**: `~/.orch/hooks/*.py` matched by `gate-governance-file-protection.py` — workers cannot modify

### Tests Run
```bash
go test ./cmd/orch/ -run TestEnforcePhaseComplete -v
# PASS: TestEnforcePhaseCompleteBlocksSpawnedWorker (0.05s)
# PASS: TestEnforcePhaseCompleteAllowsNonWorker (0.03s)
# PASS: TestEnforcePhaseCompleteAllowsPhaseCompleteMessage (0.03s)
# FAIL: TestEnforcePhaseCompleteSkipsPrintMode (0.05s) ← BUG CONFIRMED
# PASS: TestEnforcePhaseCompleteStillBlocksWithoutPrintMode (0.05s)
```

---

## Architectural Choices

### ORCH_PRINT_MODE env var (not process tree inspection)
- **What I chose:** New `ORCH_PRINT_MODE=1` env var that callers set, hook checks
- **What I rejected:** (a) Process tree inspection (checking parent cmdline for `--print`), (b) Transcript length heuristic, (c) Clearing `ORCH_SPAWNED` in callers
- **Why:** Env var is simple, explicit, and works at the hook level (single point of control). Process inspection is OS-specific and fragile. Clearing `ORCH_SPAWNED` uses existing logic but requires ALL callers to know about it and doesn't make intent clear. The env var follows the established pattern (cf. `ORCH_COMPLETING=1` for `on_close` hook suppression).
- **Risk accepted:** Callers must remember to set `ORCH_PRINT_MODE=1`. Mitigated by documenting the pattern.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Claude Code Stop hook input JSON has no `print_mode`, `session_type`, or `non_interactive` field — no way to detect `--print` mode from hook input alone
- Governance file protection blocks workers from fixing hooks — this is working as designed but adds a round-trip for hook bugs

### Externalized via `kb quick`
- The constraint "claude --print is unusable for A/B skill testing from sessions with global Stop hook" (kb-c64baf) already exists

---

## Next (What Should Happen)

**Recommendation:** Orchestrator applies 3-line hook patch, then runs tests to verify

### Orchestrator Action Required

**Step 1: Apply patch to `~/.orch/hooks/enforce-phase-complete.py`**

After line 111 (`sys.exit(0)  # Can't parse input, allow exit`), before `# --- Skip conditions ---`, add:

```python
    # Skip for non-interactive/print-mode sessions (e.g. claude --print
    # called from within a spawned worker — inherits ORCH_SPAWNED=1 but
    # is not the real worker session)
    if os.environ.get("ORCH_PRINT_MODE") == "1":
        sys.exit(0)
```

Also update the docstring skip conditions (line 14) to include:
```
- Non-interactive mode (ORCH_PRINT_MODE=1) — nested claude --print calls
```

**Step 2: Run tests**
```bash
go test ./cmd/orch/ -run TestEnforcePhaseComplete -v
# All 5 tests should pass after patch
```

**Step 3: Cross-repo issue for skillc**
skillc's test runner (`pkg/scenario/runner.go`) should set `ORCH_PRINT_MODE=1` in the env when invoking `claude --print`. This is in a separate repo.

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/skillc
  title: "Set ORCH_PRINT_MODE=1 in claude --print test runner env"
  type: task
  priority: 2
  description: "The skillc test runner calls claude --print for A/B testing. When run from within a spawned orch-go worker session, the Stop hook (enforce-phase-complete.py) contaminates output. After the hook is patched to check ORCH_PRINT_MODE=1, skillc's runner.go should set this env var when building the claude command environment."

---

## Unexplored Questions

- **Would an `--allowlist-hooks` or `--skip-hooks` flag in Claude CLI be better?** A native Claude Code mechanism to disable hooks for `--print` mode would eliminate the need for custom env vars entirely. Might be worth a feature request.
- **Should `ORCH_SPAWNED` be set via process-level mechanisms instead of `export`?** Using `env ORCH_SPAWNED=1 claude ...` (without `export`) would prevent child processes from inheriting, but would require changing how tmux SendKeys works.

---

## Friction

- **governance**: Worker cannot modify governance files (`~/.orch/hooks/*.py`), requiring orchestrator round-trip to apply a 3-line patch. The friction is appropriate (governance protection is working correctly) but adds latency for simple hook fixes.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-claude-print-output-11mar-be96/`
**Beads:** `bd show orch-go-ssbwv`
