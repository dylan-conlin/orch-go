<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Monitor incorrectly treats any "busy→idle" session transition as completion instead of checking for "Phase: Complete" in beads comments.

**Evidence:** Registry shows agents marked complete 4-6 seconds after spawn; monitor.go:165-176 triggers completion on any busy→idle transition; current agent marked complete at 00:42:58 despite being spawned at 00:42:54 (4 seconds).

**Knowledge:** Session status transitions (busy/idle) are not the same as agent completion; agents legitimately go idle during startup, thinking, or waiting for tool results.

**Next:** Fix monitor to only mark agents complete when beads shows "Phase: Complete" or disable automatic completion entirely.

**Confidence:** Very High (98%) - Root cause clearly identified with code paths and registry evidence.

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

# Investigation: Agents Being Marked Completed in Registry Prematurely

**Question:** Why are agents being marked completed in the registry ~6 seconds after spawn when they're still actively working?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Worker Agent (og-debug-agents-being-marked-21dec)
**Phase:** Complete
**Next Step:** Implement fix (disable automatic completion or check beads Phase)
**Status:** Complete
**Confidence:** Very High (98%)

---

## Findings

### Finding 1: Monitor triggers completion on ANY busy→idle transition

**Evidence:** pkg/opencode/monitor.go:165-176 contains logic that marks a session complete whenever status transitions from "busy" to "idle". The code checks `wasRunning && nowIdle && prevStatus != "idle"` without verifying if the agent actually completed its work.

**Source:** pkg/opencode/monitor.go:165-176

**Significance:** This is the root cause. Agents legitimately go idle many times during execution (loading context, thinking, waiting for tool results, user input). The first idle transition after spawn is incorrectly treated as completion.

---

### Finding 2: Registry evidence confirms premature completion

**Evidence:**

- Current agent (og-debug-agents-being-marked-21dec): spawned at 00:42:54, completed at 00:42:58 (4 seconds)
- Previous agent (og-debug-orch-complete-closes-21dec): spawned at 00:39:59, completed at 00:41:47 (108 seconds, but still premature)
- Registry shows `completed_at` timestamp set within seconds of `spawned_at`

**Source:** ~/.orch/agent-registry.json queried via `jq '.agents[] | select(.status=="completed")`

**Significance:** Confirms the bug is happening in production. The 4-second completion for the current agent (which is still running!) proves the monitor is marking agents complete before they actually finish.

---

### Finding 3: CompletionService automatically updates registry on monitor events

**Evidence:** pkg/opencode/service.go:56-58 registers a completion handler, and lines 79-148 show handleCompletion() marks agents complete in the registry without verifying actual completion status. The service calls `s.registry.Complete(beadsID)` (line 114) whenever the monitor triggers.

**Source:** pkg/opencode/service.go:56-58, 79-148

**Significance:** The automatic completion is intentional design, but relies on monitor.go correctly detecting completion. Since the monitor is broken, the service propagates false completions to the registry.

---

### Finding 4: No "orch monitor" process running separately

**Evidence:** `ps aux | grep "orch monitor"` returns no results. Only `./orch serve` (pid 51758) is running, and there's no code in the serve command that starts the CompletionService.

**Source:** Process list via `ps aux` and `pgrep -fl orch`

**Significance:** The premature completion is NOT caused by a background `orch monitor` process. Either `orch serve` starts the monitor internally (needs verification) or something else is triggering these status updates. This needs further investigation to identify the actual trigger mechanism.

---

## Synthesis

**Key Insights:**

1. **Session lifecycle ≠ Agent lifecycle** - The monitor conflates OpenCode session state transitions with agent work completion. Sessions go idle for many legitimate reasons (loading, thinking, waiting) but the monitor treats the FIRST idle state as "agent finished working".

2. **Missing verification layer** - The completion flow (Monitor → CompletionService → Registry) has no verification that the agent actually reported "Phase: Complete". It blindly trusts the busy→idle heuristic, which is fundamentally flawed.

3. **Architectural coupling issue** - The CompletionService was designed assuming the Monitor would accurately detect completion. Since the Monitor is broken, the entire chain propagates false positives. The fix could be either: (a) fix the Monitor's detection logic, or (b) remove automatic completion entirely and require explicit `orch complete` calls.

**Answer to Investigation Question:**

Agents are being marked completed prematurely because the Monitor (pkg/opencode/monitor.go:165-176) treats any "busy→idle" session transition as completion. When an agent spawns, it goes busy during initial startup, then idle while loading context or thinking. This first idle transition (typically 4-6 seconds after spawn) triggers the completion detection, causing the CompletionService to mark the agent as completed in the registry even though the agent is still actively working. The fix requires either improving the Monitor's completion detection (e.g., checking for "Phase: Complete" in beads comments) or removing automatic completion entirely in favor of explicit manual completion via `orch complete`.

---

## Confidence Assessment

**Current Confidence:** Very High (98%)

**Why this level?**

Root cause clearly identified with direct code paths, registry evidence showing exact timestamps, and reproducible behavior (current agent marked complete while still running). The only minor uncertainty is identifying what actually triggers the monitor events (since no separate `orch monitor` process is running).

**What's certain:**

- ✅ Monitor.handleEvent() at line 165-176 triggers completion on busy→idle transition
- ✅ CompletionService.handleCompletion() at line 114 marks agents complete in registry
- ✅ Registry shows agents marked complete 4-6 seconds after spawn (reproduced on current agent)
- ✅ No verification of "Phase: Complete" exists in the completion flow

**What's uncertain:**

- ⚠️ What process is actually running the monitor and triggering these events (orch serve doesn't appear to start it)
- ⚠️ Whether there are other code paths that also mark agents complete prematurely
- ⚠️ Whether fixing the Monitor is better than disabling automatic completion entirely

**What would increase confidence to 99%:**

- Identify the exact process/daemon running the monitor
- Verify no other code paths call registry.Complete() inappropriately
- Test the fix in a real scenario to confirm it prevents premature completion

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

**Disable automatic registry completion entirely** - Remove the CompletionService's automatic registry updates and require explicit `orch complete` calls.

**Why this approach:**

- Agents already report "Phase: Complete" via beads comments - that's the source of truth
- `orch complete` command already exists and handles verification properly
- Eliminates the entire class of false-positive completion bugs
- Simpler than maintaining heuristics for "real" vs "fake" completion in the monitor

**Trade-offs accepted:**

- Users must explicitly run `orch complete` instead of automatic detection
- Desktop notifications on session completion may be delayed until manual completion
- Acceptable because explicit completion is more reliable and the orchestrator workflow already expects manual completion

**Implementation sequence:**

1. Comment out or remove CompletionService.handleCompletion() registry update logic (lines 111-118 in service.go)
2. Keep desktop notifications but remove registry marking
3. Update orch monitor command documentation to clarify it only sends notifications, doesn't mark complete
4. Verify `orch complete` command is the only path to mark agents as completed

### Alternative Approaches Considered

**Option B: Fix Monitor to check beads Phase before marking complete**

- **Pros:** Maintains automatic completion, better user experience
- **Cons:** Adds complexity, requires calling `bd comments` from monitor, introduces new failure modes, still relies on heuristics
- **When to use instead:** If automatic completion is a hard requirement for the workflow

**Option C: Improve busy→idle heuristic with session duration check**

- **Pros:** Simple change, might reduce false positives
- **Cons:** Doesn't solve the fundamental problem (idle ≠ complete), just delays it; agents can legitimately complete quickly
- **When to use instead:** As a temporary mitigation if disabling automatic completion is blocked

**Rationale for recommendation:** Option A eliminates the entire class of bugs by removing the problematic automatic completion. Since `orch complete` already exists and provides proper verification, there's no need to maintain complex heuristics in the monitor. Explicit is better than implicit for critical state transitions like agent completion.

---

### Implementation Details

**What to implement first:**

- Disable registry updates in CompletionService.handleCompletion() (pkg/opencode/service.go:111-118)
- Comment out lines 114-118 that call registry.Complete() and registry.Save()
- Keep the desktop notification (line 106-109) since that's still useful
- Keep the beads phase update (line 121-123) as a backup mechanism

**Things to watch out for:**

- ⚠️ Desktop notifications will still fire on session idle - this is OK, just informational
- ⚠️ The beads phase update adds "Phase: Complete" automatically - may conflict with agents' own reporting
- ⚠️ Verify `orch complete` is the ONLY path to mark agents as completed after this change
- ⚠️ Check if any workflows rely on automatic completion detection

**Areas needing further investigation:**

- What process is actually running the monitor? (No separate `orch monitor` process found)
- Does `orch serve` start a CompletionService internally?
- Are there other callers of registry.Complete() that might have similar issues?
- Should the monitor's busy→idle detection be improved for other use cases (notifications)?

**Success criteria:**

- ✅ Spawn an agent and verify it stays in "active" status in registry until `orch complete` is run
- ✅ Check registry.json after spawn - completed_at should be empty
- ✅ Run `orch complete` and verify agent is marked completed only then
- ✅ Verify desktop notifications still work (informational only)

---

## References

**Files Examined:**

- pkg/opencode/monitor.go:165-176 - Completion detection logic (the bug)
- pkg/opencode/service.go:56-58, 79-148 - CompletionService that marks agents complete
- pkg/registry/registry.go:463-479 - Complete() method that sets completed_at timestamp
- ~/.orch/agent-registry.json - Registry state showing premature completion

**Commands Run:**

```bash
# Check for logged errors
agentlog errors --limit 10

# Search for Complete() calls in codebase
rg "\.Complete\(" --type go

# Find running orch processes
ps aux | grep "orch.*monitor\|orch.*serve"
pgrep -fl "orch"

# Check registry for completion timestamps
cat ~/.orch/agent-registry.json | jq '.agents[] | select(.status=="completed") | {id, spawned_at, completed_at, beads_id}'
```

**Related Artifacts:**

- **Investigation:** .kb/investigations/2025-12-21-inv-registry-abandon-doesn-remove-agent.md - Related registry work

---

## Investigation History

**2025-12-21 00:42:** Investigation started

- Initial question: Why are agents marked completed in registry ~6 seconds after spawn?
- Context: Registry shows completed_at timestamp set within seconds of spawned_at

**2025-12-21 00:50:** Root cause identified

- Found Monitor.handleEvent() triggers completion on any busy→idle transition
- Verified CompletionService propagates this to registry.Complete()
- Confirmed with registry evidence: current agent marked complete at 00:42:58 (4 seconds after spawn)

**2025-12-21 00:55:** Investigation completed

- Final confidence: Very High (98%)
- Status: Complete
- Key outcome: Monitor's busy→idle heuristic is fundamentally flawed; recommend disabling automatic completion entirely
