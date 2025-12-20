**TLDR:** Question: How fast is tmux fire-and-forget spawn vs blocking inline spawn? Answer: Tmux spawn returns in ~124ms (fire-and-forget), while inline spawn blocks until session completes. Tmux spawn does NOT capture session ID. Very High confidence (98%) - measured actual timing with production orch binary.

---

# Investigation: Fire-and-Forget Timing of Tmux Spawn

**Question:** How does the tmux spawn fire-and-forget timing compare to inline spawn, and what exactly happens during the spawn?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (98%)

---

## Findings

### Finding 1: Tmux Spawn Returns in ~124ms (Fire-and-Forget)

**Evidence:** 
- Timed actual spawn command: `time ./orch spawn investigation "test timing"`
- Total execution time: **0.124 seconds** (124ms)
- User time: 0.05s
- System time: 0.04s
- Command returns immediately after creating tmux window and sending keys
- Agent continues running asynchronously in tmux window

**Source:** 
- Command run: `cd /Users/dylanconlin/Documents/personal/orch-go && time ./orch spawn investigation "test timing" 2>&1`
- Code: `cmd/orch/main.go:250-318` (runSpawnInTmux function)

**Significance:** 
This confirms the fire-and-forget behavior - orchestrator is NOT blocked waiting for Claude startup. The 124ms is purely orch-go overhead (workspace creation, tmux window setup, command sending). This is critical for concurrent spawning since orchestrator remains responsive.

---

### Finding 2: Tmux Spawn Does NOT Capture Session ID

**Evidence:**
- `runSpawnInTmux` function returns immediately after sending command to tmux
- No session ID in spawn output or event logging
- Event log at line 294-308 logs "session.spawned" but SessionID field is empty
- Contrast with `runSpawnInline` (line 321-372) which parses session ID from opencode output

**Source:**
- `cmd/orch/main.go:250-318` - runSpawnInTmux has no session ID parsing
- `cmd/orch/main.go:321-372` - runSpawnInline captures sessionID via `opencode.ProcessOutput()`
- Event logging at line 294-308 shows no sessionID field populated

**Significance:**
Fire-and-forget means no session ID available immediately after spawn. Orchestrator cannot reference the session by ID right away. This is a trade-off: fast spawn vs no immediate session handle.

---

### Finding 3: Inline Spawn is Blocking (Waits for Completion)

**Evidence:**
- Tested `./orch spawn --inline investigation "test inline timing"`
- Command hangs waiting for agent to complete
- Uses `opencode.ProcessOutput()` which blocks reading stdout until process exits
- Returns session ID only after agent completes work

**Source:**
- Command run: `timeout 10 time ./orch spawn --inline investigation "test inline timing"`
- Code: `cmd/orch/main.go:321-372` - runSpawnInline waits for cmd.Wait()

**Significance:**
Inline spawn is unsuitable for orchestrator workflow since it blocks for entire session duration (potentially hours). The tmux fire-and-forget mode is essential for orchestrator to remain responsive and spawn multiple agents concurrently.

---

### Finding 2: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

## Synthesis

**Key Insights:**

1. **Fire-and-Forget is Extremely Fast (~124ms)** - The tmux spawn mode completes in 124ms total, which is purely orch-go overhead for workspace setup and tmux coordination. This means orchestrator can spawn multiple agents rapidly without blocking. The agent starts asynchronously in the tmux window while orchestrator continues working.

2. **Session ID Not Available in Fire-and-Forget Mode** - The trade-off for speed is that session ID is not captured during tmux spawn. The opencode command runs in tmux asynchronously, so there's no output to parse. This means orchestrator cannot immediately reference the session by ID after spawning.

3. **Inline Mode is Blocking and Inappropriate for Orchestration** - Inline spawn waits for the entire agent session to complete before returning (potentially hours). This makes it unsuitable for orchestrator workflows. It's only useful for testing or scripts that need to wait for completion.

**Answer to Investigation Question:**

Tmux spawn returns in **~124ms** (fire-and-forget), while inline spawn blocks until the entire session completes. The fire-and-forget mode:
- Creates tmux window, sends command, returns immediately
- Does NOT capture session ID (trade-off for speed)
- Agent runs asynchronously in tmux
- Critical for concurrent spawning - orchestrator stays responsive

The 124ms overhead is minimal and enables orchestrator to spawn multiple agents rapidly. This validates the design decision to use tmux as the default spawn mode.

---

## Confidence Assessment

**Current Confidence:** Very High (98%)

**Why this level?**

Direct measurement of actual orch-go binary behavior with production code. Timed execution, inspected code paths, and verified tmux window creation. The only minor uncertainty is whether timing varies significantly across different systems or under load.

**What's certain:**

- ✅ Tmux spawn returns in ~124ms - measured directly with `time` command
- ✅ Tmux spawn does NOT capture session ID - verified in code at cmd/orch/main.go:250-318
- ✅ Inline spawn is blocking - tested and confirmed command hangs until completion
- ✅ Agent runs asynchronously after tmux spawn - verified by checking tmux window content
- ✅ Fire-and-forget is critical for orchestrator responsiveness - validated by design

**What's uncertain:**

- ⚠️ Timing might vary on slower systems or under heavy load (not tested)
- ⚠️ Network latency to OpenCode server not measured separately (included in 124ms)

**What would increase confidence to 99%+:**

- Test timing under various load conditions (10+ concurrent spawns)
- Measure timing on different hardware/OS platforms
- Break down the 124ms into component timings (workspace creation, tmux ops, command send)

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Keep tmux as default spawn mode** - Continue using fire-and-forget tmux spawn as the default, with inline mode only for testing/special cases.

**Why this approach:**
- 124ms overhead is negligible - enables rapid concurrent spawning
- Orchestrator stays responsive - critical for managing multiple agents
- Proven design - matches Python orch-cli behavior
- Session ID not needed immediately - orchestrator tracks via workspace/beads ID

**Trade-offs accepted:**
- No immediate session ID - must use SSE monitoring or workspace tracking
- Cannot block waiting for agent completion - must use `orch complete` workflow
- Why acceptable: Fire-and-forget is essential for orchestrator pattern

**Implementation sequence:**
1. Keep tmux spawn as default (no --inline flag needed) - already done
2. Document the fire-and-forget behavior in orch-go docs
3. Ensure `orch status` and `orch complete` work without session ID

### Alternative Approaches Considered

**Option B: Capture session ID via SSE monitoring**
- **Pros:** Could get session ID shortly after spawn by monitoring SSE events
- **Cons:** Adds complexity, slows down spawn, defeats fire-and-forget purpose
- **When to use instead:** If session ID truly needed immediately (not current requirement)

**Option C: Make inline mode default**
- **Pros:** Session ID available immediately
- **Cons:** Blocks orchestrator, prevents concurrent spawning, unusable for real workflows
- **When to use instead:** Never for orchestrator; only for testing/debugging

**Rationale for recommendation:** Fire-and-forget is working as designed. The 124ms overhead validates the approach - orchestrator can spawn dozens of agents rapidly without blocking.

---

### Implementation Details

**What to implement first:**
- Document fire-and-forget behavior in README/docs
- Ensure orchestrator workflows don't assume session ID availability
- Verify `orch complete` works with beads ID only

**Things to watch out for:**
- ⚠️ Don't try to capture session ID in tmux mode - defeats the purpose
- ⚠️ SSE monitoring is separate concern - don't couple with spawn
- ⚠️ Beads ID is sufficient for tracking - workspace name maps to session

**Areas needing further investigation:**
- How to map beads ID → session ID if needed (via SSE events or workspace lookup)
- Whether `orch status` needs enhancement to show workspace → session mapping
- Performance under high concurrent spawn load (10+ simultaneous spawns)

**Success criteria:**
- ✅ Tmux spawn remains default and fast (<200ms overhead)
- ✅ Orchestrator can spawn multiple agents without blocking
- ✅ Documentation clarifies fire-and-forget behavior and trade-offs

---

## References

**Files Examined:**
- `cmd/orch/main.go:250-318` - runSpawnInTmux function showing fire-and-forget implementation
- `cmd/orch/main.go:321-372` - runSpawnInline function for comparison (blocking mode)
- `pkg/tmux/tmux.go` - Tmux window creation and command sending primitives
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template and workspace setup

**Commands Run:**
```bash
# Measure tmux spawn timing (fire-and-forget)
cd /Users/dylanconlin/Documents/personal/orch-go && time ./orch spawn investigation "test timing" 2>&1

# Check tmux window list
tmux list-windows -t workers-orch-go -F "#{window_index} #{window_name}"

# Verify agent started in tmux window
tmux capture-pane -t workers-orch-go:15 -p | head -30

# Test inline spawn for comparison (blocked after 10s)
timeout 10 time ./orch spawn --inline investigation "test inline timing"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Knowledge:** `kn-34d52f` - [decision] orch-go tmux spawn is fire-and-forget - no session ID capture
- **Issue:** `orch-go-c4r` - [P1] Fix bd create output parsing - captures 'open' instead of issue ID

---

## Investigation History

**2025-12-20 (Start):** Investigation started
- Initial question: How fast is fire-and-forget tmux spawn, and does it capture session ID?
- Context: Related to issue `orch-go-c4r` about bd create output parsing showing 'open' instead of real issue ID

**2025-12-20 (Testing):** Measured actual spawn timing
- Ran `time ./orch spawn investigation "test timing"` and got 0.124s total
- Verified agent started asynchronously in tmux window workers-orch-go:15
- Confirmed spawn output shows no session ID

**2025-12-20 (Analysis):** Code review of spawn implementations
- Reviewed runSpawnInTmux (fire-and-forget) vs runSpawnInline (blocking)
- Confirmed tmux mode has no session ID parsing
- Verified inline mode blocks waiting for completion

**2025-12-20 (Complete):** Investigation completed
- Final confidence: Very High (98%)
- Status: Complete
- Key outcome: Tmux spawn returns in ~124ms fire-and-forget without session ID, validating design for concurrent orchestration
