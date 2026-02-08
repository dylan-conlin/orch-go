**TLDR:** Question: Can we port Python's standalone + API discovery approach to orch-go? Answer: Yes, implemented StandaloneConfig, BuildStandaloneCommand, WaitForOpenCodeReady, and SendPromptAfterReady in pkg/tmux. High confidence (90%) - all tests passing, follows Python patterns exactly.

---

# Investigation: POC Port Python Standalone + API Discovery to orch-go

**Question:** Can we port Python's standalone mode spawning and TUI ready detection to orch-go?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Python uses standalone mode, not attach mode

**Evidence:** Python spawn.py:959-1032 uses `opencode {dir} --model {model}` command format instead of `opencode run --attach {server}`. The key difference:
- Attach mode: `opencode run --attach http://127.0.0.1:4096 --title "name" "prompt"` - connects to existing server
- Standalone mode: `opencode {dir} --model {model}` - launches independent instance

**Source:** `orch-cli/src/orch/spawn.py:959-963`
```python
opencode_cmd = f"{opencode_bin} {shlex.quote(str(config.project_dir))} --model {shlex.quote(model_arg)}"
```

**Significance:** Standalone mode gives each agent its own OpenCode instance, avoiding session routing issues with attach mode.

---

### Finding 2: TUI readiness detection uses specific visual indicators

**Evidence:** Python spawn.py:1083-1127 defines `_wait_for_opencode_ready()` that polls tmux pane for:
1. `┃` - Thick vertical bar (prompt box indicator)
2. "build" or "agent" text (agent selector)
3. "alt+x" or "commands" text (command hints)

TUI is ready when prompt box AND (agent selector OR command hints) are present.

**Source:** `orch-cli/src/orch/spawn.py:1114-1119`
```python
has_prompt_box = "┃" in output_raw
has_agent_selector = "build" in output_lower or "agent" in output_lower
has_command_hint = "alt+x" in output_lower or "commands" in output_lower
if has_prompt_box and (has_agent_selector or has_command_hint):
    return True
```

**Significance:** Direct port to Go with same logic ensures consistent behavior.

---

### Finding 3: Post-ready delay is critical for input focus

**Evidence:** Python uses 1.0s delay after TUI ready before typing prompt. Comment explains: "TUI needs time after visual render for input focus and event loop to settle. Testing shows 0.5s is often insufficient; 1.0s is reliable."

**Source:** `orch-cli/src/orch/spawn.py:1014-1017`

**Significance:** Must preserve this delay in Go implementation.

---

## Synthesis

**Key Insights:**

1. **Standalone mode is simpler** - No session routing complexity, each agent is independent
2. **TUI detection is reliable** - Same visual indicators work across versions
3. **Timing matters** - Post-ready delay prevents race conditions with input focus

**Answer to Investigation Question:**

Yes, the Python standalone + API discovery approach ports cleanly to Go. Implemented:
- `StandaloneConfig` - config struct for standalone mode
- `BuildStandaloneCommand` - builds `opencode {dir} --model {model}` command
- `WaitForOpenCodeReady` - polls pane content for TUI indicators
- `SendPromptAfterReady` - orchestrates wait + delay + send

All functions have unit tests and integration tests passing.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from Python reference implementation and all tests passing.

**What's certain:**

- ✅ Command format matches Python exactly
- ✅ TUI detection logic is identical
- ✅ Timing values preserved (15s timeout, 200ms poll, 1s post-ready delay)

**What's uncertain:**

- ⚠️ Not yet integrated into runSpawnInTmux (existing code still uses attach mode)
- ⚠️ Model resolution not implemented (need to pass model from spawn config)

**What would increase confidence to Very High (95%+):**

- Integration into main spawn flow
- End-to-end test with real OpenCode

---

## Implementation Recommendations

**Purpose:** Bridge from investigation to actionable implementation.

### Recommended Approach ⭐

**Integrate standalone mode into spawn flow**

**Why this approach:**
- Functions are implemented and tested
- Just need to wire into cmd/orch/main.go runSpawnInTmux
- Minimal risk, incremental change

**Implementation sequence:**
1. Add Model field to spawn.Config
2. Update runSpawnInTmux to use BuildStandaloneCommand + SendPromptAfterReady
3. Remove old attach mode code

### Alternative Approaches Considered

**Option B: Keep both modes (attach + standalone)**
- **Pros:** Backwards compatible
- **Cons:** More code to maintain, attach mode has known issues
- **When to use instead:** If attach mode is needed for specific use cases

---

### Implementation Details

**What to implement first:**
- Model field in spawn.Config
- Wire SendPromptAfterReady into runSpawnInTmux

**Things to watch out for:**
- ⚠️ OPENCODE_BIN env var for dev builds
- ⚠️ ORCH_WORKER env var must be set for worker sessions

**Success criteria:**
- ✅ `orch-go spawn investigation "test"` opens TUI in tmux
- ✅ Prompt is typed after TUI renders
- ✅ Agent starts processing

---

## References

**Files Examined:**
- `orch-cli/src/orch/spawn.py:959-1127` - Python reference implementation
- `pkg/tmux/tmux.go` - Existing Go tmux package

**Commands Run:**
```bash
# Run tests
go test ./pkg/tmux/... -v

# Run all tests
go test ./...
```

---

## Investigation History

**2025-12-20 XX:XX:** Investigation started
- Initial question: Can we port Python's standalone mode to orch-go?
- Context: Spawned as POC implementation task

**2025-12-20 XX:XX:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Implemented all functions with tests passing
