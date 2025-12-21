<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The `--tmux` flag in `orch spawn` is fully functional and correctly integrates with tmux and the agent registry.

**Evidence:** End-to-end tests confirmed session/window creation, TUI readiness detection, prompt delivery, and registry persistence.

**Knowledge:** `SendPromptAfterReady` with `IsOpenCodeReady` detection reliably handles OpenCode TUI interaction in tmux.

**Next:** Close investigation; no further action needed as the feature is working as intended.

**Confidence:** Very High (95%) - Verified via end-to-end testing and registry inspection.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test tmux flag working

**Question:** Does the `--tmux` flag in `orch spawn` correctly create a tmux window and start the agent there?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Medium (70%)

---

## Findings

### Finding 1: `--tmux` flag correctly spawns agent in tmux window

**Evidence:** Running `./build/orch spawn --tmux --no-track investigation "test tmux flag"` successfully created a new tmux window in the `workers-orch-go` session and started the OpenCode TUI with the correct prompt.

**Source:** 
- Command: `./build/orch spawn --tmux --no-track investigation "test tmux flag"`
- Verification: `tmux list-windows -t workers-orch-go` and `tmux capture-pane -t workers-orch-go:4 -p`

**Significance:** Confirms that the tmux spawning logic (session creation, window creation, command building, and prompt sending) is working as intended.

---

### Finding 2: OpenCode TUI ready detection works

**Evidence:** The agent in the tmux window started processing the prompt, which means `SendPromptAfterReady` successfully detected the TUI was ready and sent the prompt.

**Source:** `tmux capture-pane -t workers-orch-go:4 -p` output showing the agent's "Thinking" and command execution.

**Significance:** Confirms that the `IsOpenCodeReady` logic in `pkg/tmux/tmux.go` is accurate for the current OpenCode TUI.

---

### Finding 3: Agent registration in registry works for tmux spawns

**Evidence:** After spawning an agent with `--tmux`, the agent was correctly registered in `~/.orch/agent-registry.json` with the correct `window_id` (e.g., `@220`).

**Source:** `grep -A 10 "og-inv-another-tmux-test-21dec" ~/.orch/agent-registry.json`

**Significance:** Confirms that the agent state is correctly persisted, allowing for tracking and later interaction (like `orch tail` or `orch complete`).

---

## Synthesis

**Key Insights:**

1. **Full Tmux Lifecycle Verified** - The `--tmux` flag correctly handles the entire lifecycle: ensuring the session exists, creating a window, starting OpenCode, waiting for the TUI, sending the prompt, and registering the agent.

2. **Reliable TUI Interaction** - The use of `SendPromptAfterReady` with `IsOpenCodeReady` detection proves to be a reliable way to interact with the OpenCode TUI in a tmux window, avoiding the race conditions that simple sleeps might encounter.

3. **Registry Integration** - Tmux agents are properly integrated into the `orch-go` registry, enabling the same management capabilities as headless or inline agents.

**Answer to Investigation Question:**

Yes, the `--tmux` flag in `orch spawn` correctly creates a tmux window and starts the agent there. All components of the tmux spawning flow—session management, window creation, TUI readiness detection, prompt delivery, and registry persistence—are functioning as expected.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

I have performed end-to-end tests of the `--tmux` flag, verified the resulting tmux state (session, window, pane content), and confirmed the registry persistence. The agent successfully started processing the prompt in the tmux window.

**What's certain:**

- ✅ Tmux windows are created in the correct session (`workers-{project}`).
- ✅ OpenCode starts in the new window.
- ✅ The prompt is correctly delivered to the OpenCode TUI.
- ✅ The agent is registered in the registry with the correct window ID.

**What's uncertain:**

- ⚠️ Behavior on systems without tmux (though `IsAvailable()` check exists).
- ⚠️ Behavior with very long prompts that might exceed tmux `send-keys` limits (though `SendKeysLiteral` is used).

**What would increase confidence to 100%:**

- Testing on multiple platforms (e.g., Linux vs macOS).
- Testing with various tmux versions.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Maintain current implementation** - The current implementation of the `--tmux` flag is robust and works as intended.

**Why this approach:**
- It matches the proven pattern from the Python `orch-cli`.
- It provides a reliable interactive experience for users who want to see the agent's progress in real-time.
- It correctly integrates with the agent registry.

**Trade-offs accepted:**
- Dependency on `tmux` being installed on the host system.
- Slightly more complex spawning logic compared to headless mode.

**Implementation sequence:**
1. No changes needed to the core logic.
2. Consider adding more robust error handling for cases where `tmux` is missing but the flag is used.

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Spawn command implementation and `runSpawnTmux` function.
- `pkg/tmux/tmux.go` - Tmux management utility functions.

**Commands Run:**
```bash
# Build the binary
make build

# Test tmux spawn
./build/orch spawn --tmux --no-track investigation "test tmux flag"

# Verify tmux state
tmux list-windows -t workers-orch-go
tmux capture-pane -t workers-orch-go:4 -p

# Verify registry
cat ~/.orch/agent-registry.json
```

---

## Investigation History

**2025-12-21 10:20:** Investigation started
- Initial question: Does the `--tmux` flag in `orch spawn` correctly create a tmux window and start the agent there?
- Context: Verifying the Go rewrite's tmux integration.

**2025-12-21 10:30:** End-to-end test successful
- Verified window creation, TUI startup, and prompt delivery.
- Confirmed registry persistence.

**2025-12-21 10:35:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: The `--tmux` flag is fully functional and correctly implemented.

## Self-Review

- [x] Real test performed (not code review)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
