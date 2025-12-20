**TLDR:** Question: Does tmux spawn work correctly in fire-and-forget mode without trying to capture session ID? Answer: Yes - confirmed via code review and end-to-end test. Spawn returns immediately (<1 sec), creates workspace/window, and agents remain discoverable via `orch status`. High confidence (90%) - validated with real spawn and status check.

---

# Investigation: tmux spawn fire-and-forget behavior

**Question:** Does the tmux spawn implementation correctly work in fire-and-forget mode (spawning agent without capturing session ID)?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Claude (worker agent)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: runSpawnInTmux does not attempt to capture session ID

**Evidence:** 
- Function creates tmux window, sends opencode command, and returns immediately (lines 251-318)
- Uses `BuildSpawnCommand` which does NOT include `--format json` flag (line 275, see tmux.go:76-77)
- Logs spawn event with window info but no session ID field (lines 293-308)
- No call to `ProcessOutput` or any session ID extraction logic
- Returns after sending Enter key, does not wait for command completion

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:251-318`

**Significance:** This confirms the fire-and-forget design - tmux spawn does not block waiting for session ID, allowing orchestrator to remain available for other operations.

---

### Finding 2: runSpawnInline uses different approach with session ID capture

**Evidence:**
- Function uses `BuildSpawnCommand` with `--format json` (client.go)
- Calls `ProcessOutput(stdout)` to parse JSON and extract session ID (line 337)
- Waits for command completion with `cmd.Wait()` (line 342)
- Logs event WITH session ID (line 350)

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:321-350`

**Significance:** This shows two distinct spawn modes exist: inline (blocking, with session ID) vs tmux (fire-and-forget, without session ID). The modes are correctly separated.

---

### Finding 3: Design decision documented in kn

**Evidence:**
- kn-34d52f states: "orch-go tmux spawn is fire-and-forget - no session ID capture"
- Reason: "opencode run --attach is TUI-based; --format json gives session ID but loses TUI. Accept title-matching via orch status for monitoring."

**Source:** `kn context "fire-and-forget spawn"`

**Significance:** This is an intentional design choice, not an oversight. The tradeoff is: TUI visibility for agent vs session ID capture. System chooses TUI and relies on title-matching for status tracking.

---

## Test Performed

**Test:** Ran `./orch spawn investigation "test fire-and-forget spawn behavior"` and measured:
1. Command execution time (fire-and-forget = returns immediately)
2. Workspace/window creation (files and tmux state)
3. Session ID presence in spawn output
4. Agent discoverability via `orch status`

**Result:** 
- Spawn returned in <1 second (fire-and-forget confirmed ✅)
- Created workspace at `.orch/workspace/og-inv-test-fire-forget-20dec/`
- Created tmux window `workers-orch-go:8` with emoji+name: `🔬 og-inv-test-fire-forget-20dec`
- **No session ID in spawn output** (only workspace, window, beads ID printed)
- Agent **discoverable via orch status** with session ID `ses_4c36f510dffeHR7bMJqk9ai7LZ`
- Title-matching worked: workspace name matched between spawn and status

**Commands run:**
```bash
./orch spawn investigation "test fire-and-forget spawn behavior"
tmux list-windows -t workers-orch-go | grep "og-inv-test-fire-forget-20dec"
./orch status | grep "og-inv-test-fire-forget"
```

---

## Synthesis

**Key Insights:**

1. **Fire-and-forget implementation is correct** - Code review shows runSpawnInTmux does not attempt session ID capture, uses TUI mode (no --format json), and returns immediately after sending command to tmux.

2. **Session ID tracking happens later via status endpoint** - While spawn doesn't capture session ID, `orch status` can discover it by querying OpenCode's /session endpoint and matching by title/workspace name.

3. **Two spawn modes serve different purposes** - Inline mode (blocking, with session ID) for scripts/automation; tmux mode (fire-and-forget) for interactive orchestration where orchestrator availability matters more than immediate session ID.

**Answer to Investigation Question:**

Yes, tmux spawn correctly works in fire-and-forget mode. The implementation:
- Returns immediately without blocking (Finding 1, Test result)
- Does not attempt session ID capture during spawn (Finding 1)
- Successfully creates workspace and tmux window (Test result)
- Agents remain discoverable via `orch status` for monitoring (Test result)

This confirms the design decision (kn-34d52f) is correctly implemented. The tradeoff (TUI visibility vs immediate session ID) is intentional and working as designed.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Strong evidence from both code review and end-to-end test. The implementation matches the design intent documented in kn, and behavioral test confirms it works as expected. Only minor uncertainty about edge cases.

**What's certain:**

- ✅ runSpawnInTmux does not attempt session ID capture (code review, lines 251-318)
- ✅ Spawn returns immediately in fire-and-forget mode (measured <1 second return time)
- ✅ Workspace and tmux window creation works correctly (verified via ls and tmux list-windows)
- ✅ Agents are discoverable via `orch status` after spawn (tested with real session)
- ✅ Design decision is intentional and documented (kn-34d52f)

**What's uncertain:**

- ⚠️ Edge cases not tested: network failures during spawn, tmux server unavailable, concurrent spawns
- ⚠️ Long-term reliability: haven't tested multiple spawns over time or cleanup behavior
- ⚠️ Error handling paths not exercised (what happens if window creation fails after spawn starts?)

**What would increase confidence to Very High (95%+):**

- Test concurrent spawns (multiple agents spawning simultaneously)
- Test error scenarios (tmux server down, network issues, workspace conflicts)
- Run integration tests over longer period (spawn 10+ agents, verify all tracked correctly)
- Test cleanup: verify abandoned agents are detectable and cleanable

---

## Implementation Recommendations

**Purpose:** Validate that fire-and-forget spawn is working correctly. No changes needed to current implementation.

### Recommended Approach ⭐

**Keep current implementation as-is** - Fire-and-forget spawn is correctly implemented and working as designed.

**Why this approach:**
- Code correctly separates inline (blocking) vs tmux (fire-and-forget) spawn modes (Finding 2)
- Design decision is intentional and documented (Finding 3)
- End-to-end test confirms expected behavior (Test result)
- Orchestrator availability is more valuable than immediate session ID capture

**Trade-offs accepted:**
- Session ID not available immediately after spawn (must query via `orch status`)
- Monitoring requires title-matching rather than direct session ID tracking
- This is acceptable because: TUI visibility for agent is more valuable, and status endpoint provides discovery mechanism

**Implementation sequence:**
1. ✅ Already complete - fire-and-forget spawn working
2. (Optional) Add integration tests for edge cases (concurrent spawns, error scenarios)
3. (Optional) Document spawn mode differences in user-facing docs

### Alternative Approaches Considered

**Option B: Try to capture session ID during tmux spawn**
- **Pros:** Would have session ID immediately available
- **Cons:** Requires using --format json which loses TUI (Finding 3), blocks orchestrator during spawn
- **When to use instead:** Never for tmux mode - defeats the purpose. Use inline mode if session ID needed immediately.

**Option C: Dual-mode capture (background job parses JSON while showing TUI)**
- **Pros:** Could theoretically get both TUI and session ID
- **Cons:** Complex, fragile, race conditions, unclear benefit over status endpoint discovery
- **When to use instead:** Not recommended - over-engineering for minimal gain

**Rationale for recommendation:** Current implementation correctly balances orchestrator availability (fire-and-forget) with agent visibility (TUI) and monitoring (status endpoint). No changes needed.

---

### Implementation Details

**What to implement first:**
- Nothing - current implementation is correct
- (Optional enhancement) Add integration test for tmux spawn to prevent regression

**Things to watch out for:**
- ⚠️ Beads ID parsing bug (orch-go-c4r) - shows "open" instead of actual issue ID, affects tracking but not spawn functionality
- ⚠️ Don't add --format json to tmux spawn - would break TUI (see Finding 1)
- ⚠️ Title-matching for status assumes unique workspace names - collision could cause mismatched tracking

**Areas needing further investigation:**
- None critical - fire-and-forget spawn is working as designed
- (Optional) Performance testing: how many concurrent spawns can system handle?
- (Optional) Error recovery: what happens if OpenCode server is down during spawn?

**Success criteria:**
- ✅ Spawn returns immediately (<2 seconds) - CONFIRMED
- ✅ Agent window created and visible in tmux - CONFIRMED
- ✅ Agent discoverable via `orch status` - CONFIRMED
- ✅ Orchestrator remains available during spawn - CONFIRMED

---

## Self-Review

### Scope Verification

**Did you scope the problem with rg before concluding?**

- ✅ **Problem scoped** - Searched for spawn-related code, session ID handling, fire-and-forget patterns
- ✅ **Scope documented** - Examined 2 spawn modes (inline vs tmux), documented differences
- ✅ **Broader patterns checked** - Reviewed related code (BuildSpawnCommand, session tracking, status endpoint)

### Investigation-Specific Checks

- ✅ **Real test performed** - Ran actual spawn command, measured timing, verified window/workspace creation, checked status discovery
- ✅ **Conclusion from evidence** - Based on code review (Finding 1, 2) + end-to-end test (Test section), not speculation
- ✅ **Question answered** - Investigation clearly answers: "Does tmux spawn work in fire-and-forget mode?" → Yes
- ✅ **Reproducible** - Commands documented, steps can be followed to verify same results

### Checklist

- ✅ **Test is real** - Ran `./orch spawn investigation ...` and measured behavior
- ✅ **Evidence concrete** - Specific timings (<1 sec), file paths, window IDs, session IDs
- ✅ **Conclusion factual** - Based on observed return time, code paths, status discovery
- ✅ **No speculation** - All claims backed by code review or test results
- ✅ **Question answered** - Original question fully addressed
- ✅ **File complete** - All sections filled with relevant information
- ✅ **TLDR filled** - Summarizes question, answer, confidence
- ✅ **NOT DONE claims verified** - No claims of missing features (only verified existing implementation)

### Discovered Work Check

**During this investigation, did you discover any issues?**

- ⚠️ **Related bug noted**: orch-go-c4r - "Fix bd create output parsing - captures 'open' instead of issue ID"
  - This bug affects beads tracking but not spawn functionality
  - Already tracked in beads, no new issue needed

**Other discoveries:**
- None - investigation focused on confirming existing behavior works correctly

**Self-Review Status:** PASSED ✅

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:251-318` - runSpawnInTmux implementation (fire-and-forget mode)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:321-350` - runSpawnInline implementation (blocking mode with session ID)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go:76-88` - BuildSpawnCommand (no --format json for TUI)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux_test.go:96-99` - Test asserting --format json NOT included

**Commands Run:**
```bash
# Test fire-and-forget spawn
./orch spawn investigation "test fire-and-forget spawn behavior"

# Verify window creation
tmux list-windows -t workers-orch-go | grep "og-inv-test-fire-forget-20dec"

# Verify workspace creation
ls -la .orch/workspace/og-inv-test-fire-forget-20dec/

# Verify agent discovery
./orch status | grep "og-inv-test-fire-forget"

# Check for context about fire-and-forget design
kn context "fire-and-forget spawn"
```

**Related Artifacts:**
- **Decision:** kn-34d52f - "orch-go tmux spawn is fire-and-forget - no session ID capture"
- **Beads Issue:** orch-go-7ch - "[orch-go] investigation: test tmux spawn - confirm fire-and-forget works"
- **Beads Issue:** orch-go-c4r - "Fix bd create output parsing - captures 'open' instead of issue ID" (related bug)

---

## Investigation History

**2025-12-20 08:19:** Investigation started
- Initial question: Does tmux spawn work correctly in fire-and-forget mode?
- Context: Verify design decision kn-34d52f is correctly implemented

**2025-12-20 08:20:** Code review completed
- Found runSpawnInTmux does not capture session ID (fire-and-forget confirmed in code)
- Identified two spawn modes: inline (blocking) vs tmux (fire-and-forget)

**2025-12-20 08:21:** End-to-end test performed
- Spawned test agent, measured <1 second return time
- Verified window/workspace creation and status discovery

**2025-12-20 08:25:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Fire-and-forget spawn is correctly implemented and working as designed
