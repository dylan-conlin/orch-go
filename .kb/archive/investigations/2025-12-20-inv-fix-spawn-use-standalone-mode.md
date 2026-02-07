**TLDR:** Question: How to fix spawn to use standalone mode with TUI? Answer: Added BuildStandaloneCommand() function and updated runSpawnInTmux() to: 1) start opencode without prompt arg, 2) wait for TUI ready, 3) send prompt via tmux send-keys. High confidence (90%) - implementation matches Python spawn.py:959-1032.

---

# Investigation: Fix Spawn to Use Standalone Mode with TUI

**Question:** How to fix spawn to use standalone mode with TUI instead of headless prompt arg?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Current orch-go passes prompt as CLI arg (wrong approach)

**Evidence:** The old `runSpawnInTmux` function built an opencode command with the prompt as a CLI argument:

```go
runCfg := &tmux.RunConfig{
    Prompt: minimalPrompt,  // Prompt passed as CLI arg
}
cmd := tmux.BuildRunCommand(runCfg)
```

**Source:** `cmd/orch/main.go:599-608` (before changes)

**Significance:** Python orch-cli explicitly rejected this approach because "--prompt flag has inconsistent submit behavior" (spawn.py:961). This causes TUI rendering issues.

---

### Finding 2: Python uses standalone mode with send-keys

**Evidence:** Python orch-cli uses this pattern:

```python
# spawn.py:959-963 - Standalone mode without prompt arg
opencode_cmd = f"{opencode_bin} {shlex.quote(str(config.project_dir))} --model {shlex.quote(model_arg)}"

# spawn.py:1019-1032 - Type prompt after TUI ready
subprocess.run(["tmux", "send-keys", "-t", actual_window_target, "-l", minimal_prompt], check=True)
subprocess.run(["tmux", "send-keys", "-t", actual_window_target, "Enter"], check=True)
```

**Source:** `orch-cli/src/orch/spawn.py:959-1032`

**Significance:** This is the production-tested approach that renders TUI correctly.

---

### Finding 3: WaitForOpenCodeReady already exists in orch-go

**Evidence:** The `pkg/tmux/tmux.go` already has:

- `WaitForOpenCodeReady()` (lines 289-309)
- `IsOpenCodeReady()` (lines 273-287)
- `SendPromptAfterReady()` (lines 311-329)
- `DefaultWaitConfig()` - 15s timeout, 200ms poll (matching Python)
- `DefaultSendPromptConfig()` - 1s post-ready delay (matching Python)

**Source:** `pkg/tmux/tmux.go:57-108, 273-329`

**Significance:** The TUI-ready detection was already implemented - just needed to wire it into the spawn workflow.

---

## Synthesis

**Key Insights:**

1. **The fix was mostly wiring, not new code** - The TUI-ready detection functions already existed. The main change was switching from CLI-arg prompt to standalone mode + send-keys.

2. **ORCH_WORKER env var needed** - Python sets this to signal hooks that this is a worker session. Added `export ORCH_WORKER=true` to match.

3. **Error handling allows continuation** - If TUI-ready wait fails, we warn but don't abort (the agent may still work).

**Answer to Investigation Question:**

Fixed spawn by:

1. Added `BuildStandaloneCommand()` - returns command string without prompt arg
2. Updated `runSpawnInTmux()` to use standalone mode workflow:
   - Send `export ORCH_WORKER=true && opencode {dir} --model {model}`
   - Wait for TUI ready via `WaitForOpenCodeReady()`
   - Send prompt via `SendPromptAfterReady()` (literal mode)
   - Verify window still exists

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation directly ports Python's proven approach. All helper functions already existed and have tests.

**What's certain:**

- ✅ Code compiles and all tests pass
- ✅ Matches Python spawn.py:959-1032 exactly
- ✅ TUI readiness detection uses same indicators (┃ + build/agent + alt+x/commands)
- ✅ Timing matches Python (15s timeout, 200ms poll, 1s post-ready delay)

**What's uncertain:**

- ⚠️ Real-world testing with actual opencode instance not performed (would need manual test)
- ⚠️ Error recovery when TUI doesn't appear (currently warns but continues)

**What would increase confidence to Very High (95%+):**

- Manual test of `orch spawn` with real opencode instance
- Verify TUI renders correctly in tmux

---

## Implementation Recommendations

**Purpose:** This investigation was implementation-focused, not exploratory. Implementation is complete.

### Implementation Completed ✅

**Changes made:**

1. Added `BuildStandaloneCommand()` to `pkg/tmux/tmux.go` (returns string, not exec.Cmd)
2. Updated `runSpawnInTmux()` in `cmd/orch/main.go` to:
   - Use `BuildStandaloneCommand()` instead of `BuildRunCommand()`
   - Wait for TUI ready before sending prompt
   - Send prompt via `SendPromptAfterReady()`
   - Add ORCH_WORKER env var export
3. Added test for `BuildStandaloneCommand()`

---

## References

**Files Examined:**

- `orch-cli/src/orch/spawn.py:959-1032` - Python reference implementation
- `pkg/tmux/tmux.go` - Go tmux helpers (already had TUI-ready functions)
- `cmd/orch/main.go:583-701` - Go spawn implementation

**Commands Run:**

```bash
# Verify build
go build ./...

# Run tests
go test ./...
```

**Related Artifacts:**

- **Investigation:** `.kb/investigations/2025-12-20-design-explore-tradeoffs-orch-opencode-integration.md` - Design decision for standalone mode

---

## Investigation History

**[2025-12-20 ~16:00]:** Investigation started

- Initial question: How to fix spawn to use standalone mode with TUI?
- Context: Current attach mode with prompt arg doesn't render TUI correctly

**[2025-12-20 ~16:30]:** Implementation complete

- Final confidence: High (90%)
- Status: Complete
- Key outcome: Ported Python's standalone mode approach to orch-go
