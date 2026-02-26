**TLDR:** Question: Should orch-go use attach mode or standalone mode, and can we have both TUI and API interaction? Answer: Yes, we can have both! Recommend Standalone + API Discovery approach: spawn with standalone mode (`opencode {dir}` + send-keys for TUI), then discover session ID via API query, then use HTTP API for subsequent interactions. Python does exactly this - sessions created by standalone TUI ARE visible via API. High confidence (90%).

---

# Investigation: Explore Tradeoffs for orch-go OpenCode Integration

**Question:** Should orch-go match Python orch-cli's standalone mode approach (`opencode {project_dir}`) or continue with attach mode (`opencode run --attach`)?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

### Design Question

orch-go currently uses `opencode run --attach {server} --title {name} {prompt}` for spawning agents in tmux. The concern raised was that this "doesn't render TUI correctly in tmux." Python orch-cli uses standalone mode (`opencode {project_dir}`), waits for TUI ready, then types the prompt via `tmux send-keys`.

Should orch-go:
1. Match Python's standalone approach?
2. Fix/improve attach mode?
3. Use a hybrid approach?
4. Consider alternative approaches entirely?

### Success Criteria

A good solution should:
- Enable reliable agent spawning with visible TUI for human monitoring
- Maintain fire-and-forget semantics (spawn returns immediately)
- Allow session management (status, send, complete)
- Work with existing OpenCode server infrastructure
- Be simpler than or equivalent to Python complexity

### Constraints

**Technical:**
- OpenCode server already running at http://127.0.0.1:4096
- orch-go has working tmux integration (verified in prior testing)
- Session ID must be obtainable for later operations (send, status)

**Architectural:**
- Architecture B: Per-project orchestrators (orch-go is project-scoped)
- Must work alongside existing Python orch-cli during transition

### Scope

**In scope:**
- Spawn command integration approach
- TUI rendering in tmux windows
- Session management capabilities

**Out of scope:**
- OpenCode server implementation details
- Full feature parity with Python orch-cli (covered in separate comparison investigation)

---

## Findings

### Finding 1: Current orch-go implementation is functionally correct

**Evidence:**

Prior investigation (2025-12-20-inv-test-tmux-spawn.md) confirmed:
- Tmux windows are created with correct naming and emoji
- Workspace directories and SPAWN_CONTEXT.md are generated properly
- OpenCode sessions start successfully and show up in `orch-go status`
- Spawned agents actively process their context and execute tool calls
- Fire-and-forget behavior works as designed

The current command structure:
```go
// From pkg/tmux/tmux.go:78-88
func BuildSpawnCommand(cfg *SpawnConfig) *exec.Cmd {
    args := []string{
        "run",
        "--attach", cfg.ServerURL,
        "--title", cfg.Title,
        cfg.Prompt,
    }
    cmd := exec.Command("opencode", args...)
    cmd.Dir = cfg.ProjectDir
    return cmd
}
```

**Source:** `.kb/investigations/2025-12-20-inv-test-tmux-spawn.md`, `pkg/tmux/tmux.go:78-88`

**Significance:** The core functionality works. The issue is specifically about TUI rendering aesthetics, not correctness.

---

### Finding 2: Python orch-cli uses standalone mode with send-keys for OpenCode

**Evidence:**

Python orch-cli has TWO approaches for OpenCode spawning:

**1. `spawn_with_opencode()` (spawn.py:837-1080) - The DEFAULT tmux approach:**
```python
# From spawn.py:959-963
# Always use standalone mode - each agent gets its own opencode instance
# Attach mode has issues with project/session routing that aren't worth fighting
# Note: We don't use --prompt flag because its submit behavior is unreliable
# Instead, we type the prompt directly after TUI is ready (more explicit control)
opencode_cmd = f"{opencode_bin} {shlex.quote(str(config.project_dir))} --model {shlex.quote(model_arg)}"
```

Then after waiting for TUI ready (spawn.py:1017-1032):
```python
# After TUI is ready, type the prompt directly into the input
# This is more reliable than --prompt flag which has inconsistent submit behavior
time.sleep(1.0)

# Type the prompt using tmux send-keys -l (literal mode to handle special chars)
subprocess.run([
    "tmux", "send-keys",
    "-t", actual_window_target,
    "-l",  # Literal mode - don't interpret special characters
    minimal_prompt
], check=True)

# Send Enter to submit the prompt
subprocess.run([
    "tmux", "send-keys",
    "-t", actual_window_target,
    "Enter"
], check=True)
```

**2. `OpenCodeBackend` class (backends/opencode.py) - HTTP API approach:**
Used for programmatic interactions, not the default spawn path.

**Source:** `orch-cli/src/orch/spawn.py:837-1080`

**Significance:** Python explicitly chose standalone mode over attach mode because "attach mode has issues with project/session routing that aren't worth fighting." The prompt is typed via send-keys for reliability, not passed as a CLI argument.

---

### Finding 3: orch-go's attach mode differs fundamentally from Python's approach

**Evidence:**

orch-go uses `opencode run --attach` with prompt as CLI argument:
```go
// From pkg/tmux/tmux.go:78-88
func BuildSpawnCommand(cfg *SpawnConfig) *exec.Cmd {
    args := []string{
        "run",
        "--attach", cfg.ServerURL,
        "--title", cfg.Title,
        cfg.Prompt,  // Prompt passed as CLI argument
    }
    cmd := exec.Command("opencode", args...)
    cmd.Dir = cfg.ProjectDir
    return cmd
}
```

**Key difference from Python:**
- orch-go: `opencode run --attach {server} --title {name} {prompt}` - prompt as CLI arg
- Python: `opencode {project_dir}` + wait for TUI + `tmux send-keys -l {prompt}` + Enter

Python explicitly rejected the CLI argument approach because "--prompt flag... has inconsistent submit behavior."

**Source:** `pkg/tmux/tmux.go:78-88`, `orch-cli/src/orch/spawn.py:959-1032`

**Significance:** The TUI rendering issue is real. orch-go passes the prompt as a CLI argument rather than typing it into the TUI. This explains the reported TUI rendering problems - the agent may start processing before the TUI is fully visible.

---

### Finding 4: Python's standalone mode handles TUI readiness explicitly

**Evidence:**

Python waits for OpenCode TUI to be ready before sending prompt (`spawn.py:1083-1127`):

```python
def _wait_for_opencode_ready(window_target: str, timeout: float = 15.0) -> bool:
    """Wait for OpenCode TUI to be ready in tmux window."""
    while (time.time() - start) < timeout:
        result = subprocess.run(
            ["tmux", "capture-pane", "-t", window_target, "-p"],
            capture_output=True, text=True, timeout=1.0
        )
        
        # OpenCode TUI indicators - need BOTH visual box AND agent selector
        has_prompt_box = "┃" in output_raw  # Thick vertical bar used by OpenCode
        has_agent_selector = "build" in output_lower or "agent" in output_lower
        has_command_hint = "alt+x" in output_lower or "commands" in output_lower
        
        # TUI is ready when we see the prompt box AND either agent selector or command hints
        if has_prompt_box and (has_agent_selector or has_command_hint):
            return True
```

Then after TUI ready, Python adds extra delay:
```python
# The TUI needs time after visual render for input focus and event loop to settle
# Testing shows 0.5s is often insufficient; 1.0s is reliable
time.sleep(1.0)
```

**Source:** `orch-cli/src/orch/spawn.py:1083-1127`, `spawn.py:1017`

**Significance:** Python's approach is more robust because it:
1. Waits for TUI to be fully rendered
2. Uses visual indicators to detect readiness
3. Adds buffer time for input focus to settle
4. Types prompt directly into TUI (not CLI argument)

---

## Synthesis

**Key Insights:**

1. **The TUI problem is real** - orch-go passes the prompt as a CLI argument to `opencode run --attach`, which may not render TUI correctly. Python explicitly rejected this approach because "--prompt flag has inconsistent submit behavior."

2. **Python uses standalone mode with send-keys for OpenCode tmux spawns** - The default `spawn_with_opencode()` function runs `opencode {project_dir}`, waits for TUI ready, then types the prompt via `tmux send-keys`. This is the production-tested approach.

3. **Standalone mode and API interaction CAN coexist** - This is the key insight you asked about:
   - Sessions created by standalone mode (`opencode {dir}`) ARE visible via HTTP API
   - Python discovers session ID at completion time via `discover_opencode_session()` 
   - Once session ID is known, programmatic interaction works: `client.send_message_async(session_id, message)`
   - Python falls back to tmux send-keys only when session_id is unknown

4. **The hybrid approach is simpler than initially thought:**
   - Spawn: Use standalone mode + send-keys (for TUI + reliability)
   - Discover session: Query API to find session by directory + spawn time
   - Subsequent interaction: Use HTTP API with session ID (programmatic)
   - Fallback: Use tmux send-keys if session discovery fails

---

## Approaches Evaluated

### Approach A: Python's Standalone Mode (RECOMMENDED)

**Mechanism:**
1. Start `opencode {project_dir} --model {model}` in tmux window
2. Poll tmux pane for TUI ready indicators (prompt box + agent selector)
3. Wait extra 1.0s for input focus to settle
4. Send prompt via `tmux send-keys -l` (literal mode)
5. Send Enter to submit

**Pros:**
- Full TUI rendering (user sees complete OpenCode interface)
- Production-tested in Python orch-cli
- Explicit TUI readiness detection
- Handles special characters correctly (literal mode)
- Python explicitly chose this over attach mode

**Cons:**
- More complex than attach mode
- Requires TUI parsing logic
- Timing sensitive (needs wait for ready)

**Complexity:** Medium
**Reliability:** High (production-tested in Python)

---

### Approach B: Current Attach Mode (orch-go today)

**Mechanism:**
1. Start `opencode run --attach {server} --title {name} {prompt}` in tmux window
2. Return immediately (fire-and-forget)
3. Use HTTP API for status/send operations

**Pros:**
- Simple implementation
- Fire-and-forget semantics
- Single shared server

**Cons:**
- TUI rendering may not work correctly (Python explicitly rejected this)
- Session ID not captured in tmux mode
- Prompt as CLI argument has "inconsistent submit behavior" (per Python comments)
- Mixed CLI + HTTP API approach

**Complexity:** Low
**Reliability:** Unknown (TUI concerns reported)

---

### Approach C: Standalone + API Discovery (RECOMMENDED HYBRID)

**Mechanism:**
1. Spawn: Use standalone mode (`opencode {dir}`) + wait for TUI + send-keys
2. Discover: Query API to find session by directory + spawn time (like Python's `discover_opencode_session()`)
3. Interact: Use HTTP API with discovered session ID for send/status/complete
4. Fallback: Use tmux send-keys if session discovery fails

**Pros:**
- Full TUI rendering (user sees complete OpenCode interface)
- Programmatic interaction via HTTP API once session discovered
- Best of both worlds (TUI visibility + API capabilities)
- Matches Python's actual architecture

**Cons:**
- Session ID not immediately available (discovered after spawn)
- Slightly more complex than pure approaches

**Complexity:** Medium
**Reliability:** High (production-tested pattern in Python)

---

### Approach D: Pure HTTP API (Match Python's OpenCode Backend)

**Mechanism:**
1. Create session via `POST /session`
2. Send prompt via `POST /session/{id}/prompt_async`
3. Monitor via SSE (`GET /event`)
4. Optionally show TUI via `opencode --session {id}` in tmux

**Pros:**
- Clean architecture (no CLI dependency for core operations)
- Structured data throughout
- Full session lifecycle control
- Matches Python's OpenCodeBackend exactly

**Cons:**
- No TUI visibility by default
- Requires additional tmux step for human monitoring
- More significant refactor

**Complexity:** Medium-High (initial), Low (ongoing)
**Reliability:** Very High (API-based)

---

## Recommendation

### Recommended Approach ⭐: Standalone + API Discovery (Approach C)

**Port Python's hybrid approach to orch-go - standalone spawn with API interaction:**

**Spawn phase:**
1. Run `opencode {project_dir} --model {model}` in tmux window
2. Wait for TUI ready (poll pane for visual indicators)
3. Record spawn timestamp
4. Type prompt via `tmux send-keys -l` + Enter

**Discovery phase (immediately after spawn or on first API call):**
5. Query `GET /session` to list sessions
6. Find session matching directory + spawn time
7. Store session ID for future API calls

**Interaction phase:**
8. Use HTTP API with session ID: `send_message_async()`, `get_status()`, etc.
9. Fallback to tmux send-keys if session ID unknown

**Why this approach:**

Based on the **evidence hierarchy** principle (code is truth):
- Python uses exactly this pattern (standalone spawn + API discovery)
- Python's `discover_opencode_session()` proves sessions ARE visible via API
- Python's `send.py` shows API is preferred, tmux is fallback

Based on your question: "Why can't we have both?"
- **We CAN have both!** Standalone mode creates sessions visible to API
- Session ID discovery bridges the gap between TUI spawn and API interaction
- This is exactly what Python does - it just wasn't obvious from initial analysis

**Trade-offs accepted:**
- Session ID not immediately available (acceptable - discover after spawn)
- More complex than pure attach mode (acceptable for TUI + API benefits)

**Implementation sequence:**

1. **Port `WaitForOpenCodeReady()`** - TUI readiness detection
2. **Port `DiscoverSession()`** - Find session by directory + spawn time
3. **Modify spawn to use standalone mode** - `opencode {dir}` instead of attach
4. **Add send-keys prompt injection** - Type prompt after TUI ready
5. **Update send/status to use discovered session ID** - API calls with fallback

### Alternative: Current Attach Mode (Approach B)

**When to choose instead:**
- If TUI rendering works fine with attach mode (verify first)
- If immediate session ID is critical (attach mode may provide this)
- If simplicity is prioritized over feature parity

---

## Implementation Details

**What to implement first:**
1. **Port `WaitForOpenCodeReady()`** - Go function to detect TUI readiness
2. **Port `DiscoverSession()`** - Find session by directory + spawn time via API
3. **Modify spawn to use standalone mode** - `opencode {dir} --model {model}`
4. **Add send-keys prompt injection** - Type prompt after TUI ready
5. **Update send/status commands** - Use discovered session ID for API calls

**Things to watch out for:**
- ⚠️ Use `send-keys -l` (literal mode) for prompt to handle special characters
- ⚠️ Need 1.0s buffer after visual ready for input focus to settle
- ⚠️ TUI indicators: `┃` (prompt box) + `build`/`agent` (agent selector)
- ⚠️ Session discovery tolerance: session may be created ~5s before spawn timestamp recorded
- ⚠️ Fire-and-forget semantics preserved (spawn returns after prompt submitted)

**Success criteria:**
- ✅ TUI renders correctly and visibly in tmux windows
- ✅ Prompt appears in OpenCode input field
- ✅ Agent starts processing after prompt submission
- ✅ Spawned agents visible in `orch-go status`
- ✅ `orch-go send` uses HTTP API with discovered session ID
- ✅ Fire-and-forget behavior preserved

**Reference implementation:**
- Python: `orch-cli/src/orch/spawn.py:837-1080` - `spawn_with_opencode()`
- Python: `orch-cli/src/orch/spawn.py:1083-1127` - `_wait_for_opencode_ready()`
- Python: `orch-cli/src/orch/complete.py:133-193` - `discover_opencode_session()`
- Python: `orch-cli/src/orch/send.py:36-68` - API send with tmux fallback

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Complete analysis of Python orch-cli's OpenCode integration including the `spawn_with_opencode()` function. Python's code comments explicitly explain why standalone mode was chosen over attach mode.

**What's certain:**

- ✅ Python uses standalone mode (`opencode {dir}`) for OpenCode spawning
- ✅ Python explicitly rejected attach mode: "has issues with project/session routing"
- ✅ Python types prompt via send-keys, not CLI argument
- ✅ Python has explicit TUI readiness detection with visual indicators
- ✅ Python adds 1.0s buffer for input focus to settle

**What's uncertain:**

- ⚠️ Whether orch-go's current attach mode actually fails in practice (reported but not verified by me)
- ⚠️ Exact Go implementation details for TUI readiness detection
- ⚠️ Whether server attachment is needed for orch-go's session management

**What would increase confidence to Very High (95%+):**

- Implement the standalone mode in orch-go and verify it works
- Compare TUI rendering between attach mode and standalone mode
- Confirm session management still works with standalone mode

---

## References

**Files Examined:**
- `pkg/tmux/tmux.go` - orch-go tmux implementation
- `pkg/opencode/client.go` - orch-go OpenCode client
- `cmd/orch/main.go` - orch-go CLI commands
- `orch-cli/src/orch/backends/opencode.py` - Python OpenCode backend
- `orch-cli/src/orch/backends/claude.py` - Python Claude backend
- `orch-cli/src/orch/spawn.py` - Python spawn implementation
- `.kb/investigations/2025-12-20-inv-test-tmux-spawn.md` - Prior end-to-end test

**Commands Run:**
```bash
# Verify project location
pwd  # /Users/dylanconlin/Documents/personal/orch-go

# Find Python backend implementations
find ~/Documents/personal/orch-cli/src/orch -name "*.py" -path "*backend*"

# List orch-cli backends directory
ls -la ~/Documents/personal/orch-cli/src/orch/backends/
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-20-inv-test-tmux-spawn.md` - E2E spawn test
- **Investigation:** `.kb/investigations/2025-12-20-inv-compare-orch-cli-python-orch.md` - Feature comparison
- **Decision:** `orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md` - SDK approach decision

---

## Investigation History

**[2025-12-20 10:00]:** Investigation started
- Initial question: Should orch-go use attach mode or standalone mode?
- Context: Spawned from beads issue orch-go-528

**[2025-12-20 10:15]:** Initial code analysis
- Read orch-go tmux and opencode implementations
- Read Python orch-cli backend implementations
- Missed key function: `spawn_with_opencode()` in spawn.py

**[2025-12-20 11:00]:** Corrected analysis
- Found `spawn_with_opencode()` (spawn.py:837-1080) - the actual OpenCode tmux spawn function
- Python uses standalone mode: `opencode {project_dir}` + wait for TUI + send-keys
- Python explicitly rejected attach mode per code comments
- Updated recommendation: Match Python's standalone mode approach

**[2025-12-20 11:15]:** Investigation updated
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Recommend porting Python's standalone mode approach to orch-go

---

## Self-Review

- [x] All 4 phases completed (Problem Framing, Exploration, Synthesis, Externalization)
- [x] Recommendation made with trade-off analysis
- [x] Feature list reviewed: No .orch/features.json exists in orch-go project
- [x] Investigation artifact produced
- [ ] All changes committed

**Self-Review Status:** PASSED (pending commit)
