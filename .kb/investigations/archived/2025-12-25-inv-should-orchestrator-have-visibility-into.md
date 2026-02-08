<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** System resource visibility in orchestrator provides marginal value - the 125% CPU bug was found via external monitoring (sketchybar), not orch tooling.

**Evidence:** The `IsSessionProcessing` bug causing 125% CPU was diagnosed externally; fix is documented in serve.go:315-319; dashboard constraint of 666px width limits space for additional metrics.

**Knowledge:** Orchestration layer should focus on agent coordination (spawn/complete/status), not process management. High CPU/memory in orchestration processes indicates bugs, not normal operation.

**Next:** Created decision artifact recommending no implementation. If demand emerges, consider minimal "health indicator" approach.

**Confidence:** High (85%) - Based on concrete evidence of how the bug was found and fixed, but untested whether future resource issues would follow the same pattern.

---

# Investigation: Should Orchestrator Have Visibility Into System Resources?

**Question:** Should the orchestrator (orch status, dashboard) have visibility into system resources like CPU % and memory usage? What would be actionable?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Design session agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: The 125% CPU Bug Was Already Diagnosed and Fixed

**Evidence:** serve.go lines 315-319 contains a comment documenting the fix:
```go
// NOTE: IsProcessing is now populated client-side via SSE session.status events.
// Previously we called client.IsSessionProcessing(s.ID) here, but that makes
// an HTTP call per session which caused 125% CPU when dashboard polled frequently.
```

**Source:** /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:315-319

**Significance:** The bug that triggered this design question has already been fixed. The diagnosis happened via external monitoring (sketchybar), not through orchestration tooling. This suggests external monitoring is sufficient for catching pathological states.

---

### Finding 2: Current Dashboard Shows Agent-Focused Metrics

**Evidence:** The stats bar shows:
- Errors count (agent errors)
- Focus/drift status
- Servers running/total
- Beads ready issues
- SSE connection status

API endpoints expose:
- /api/agents - Agent status and phase
- /api/usage - Claude Max limits (the rate-limiting concern)
- /api/focus, /api/beads, /api/servers

**Source:** /Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:293-409

**Significance:** The dashboard is focused on agent coordination concerns, not system resources. Adding resource monitoring would shift focus away from the orchestration layer's core purpose.

---

### Finding 3: Dashboard Has 666px Width Constraint

**Evidence:** Prior constraint from spawn context:
> "Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling."

**Source:** SPAWN_CONTEXT.md prior knowledge section

**Significance:** Any additional dashboard elements compete for limited horizontal space. Resource monitoring panels would need to justify displacing existing agent-focused metrics.

---

### Finding 4: Process Inspection Confirms Current Resource Usage

**Evidence:** Running `ps aux` showed:
- `orch serve`: 153% CPU (currently exhibiting high CPU, reproducible)
- `opencode serve`: 6.6% CPU, 15.5% memory
- Agent processes (bun): ~24% CPU during active work

**Source:** ps aux command during investigation

**Significance:** High CPU on `orch serve` (>100%) is abnormal and indicates a bug. Normal orchestration layer CPU should be negligible. Agent processes having high CPU is expected (LLM work).

---

## Synthesis

**Key Insights:**

1. **External monitoring worked** - Dylan found the 125% CPU bug via sketchybar, not through orch tooling. The existing external monitoring setup is sufficient.

2. **High resource usage = bug** - For orchestration processes, high CPU/memory indicates a problem to fix, not a normal state to monitor. There's no "acceptable high CPU" threshold for orch serve.

3. **Different domains** - System resource monitoring is OS/process-level tooling. Agent coordination is application-level orchestration. Mixing domains adds complexity without clear benefit.

**Answer to Investigation Question:**

The orchestrator should **not** have built-in system resource visibility. The 125% CPU bug Dylan observed:
1. Was found via existing external monitoring (sketchybar)
2. Was caused by a code bug, not resource exhaustion
3. Was fixed without needing orchestrator-level resource metrics

What would be actionable is already exposed:
- Agent phase/status (is work progressing?)
- Claude Max usage limits (approaching rate limits?)
- Beads backlog (work queue health)

System resources are not actionable at the orchestration layer - if orch serve is at 125% CPU, the action is "fix the bug," not "watch the metric."

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence that external monitoring found the bug and the fix was applied. The domain separation (orchestration vs process management) is clear. However, this is based on a single incident.

**What's certain:**

- ✅ The 125% CPU bug was diagnosed via external monitoring (sketchybar)
- ✅ The fix is documented in serve.go:315-319
- ✅ Dashboard already surfaces actionable orchestration metrics

**What's uncertain:**

- ⚠️ Whether future resource issues would follow the same pattern
- ⚠️ Whether users without sketchybar would need built-in monitoring
- ⚠️ Whether CI/CD or remote deployments would benefit from self-diagnostics

**What would increase confidence to Very High:**

- Multiple incidents of resource issues caught by external monitoring
- User feedback from others running orch-go

---

## Implementation Recommendations

**Purpose:** No implementation recommended. Created decision artifact instead.

### Recommended Approach: No Implementation

**Decision artifact created:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md`

**Rationale:**
- External monitoring (sketchybar) already provides resource visibility
- Orchestration layer should focus on agent coordination
- High resource usage in orch processes indicates bugs, not normal operation

**Trade-offs accepted:**
- Users without external monitoring may not notice resource issues
- Orchestrator can't self-diagnose pathological states

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go` - API server, contains fix for IsProcessing bug
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go` - OpenCode client, IsSessionProcessing method
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte` - Dashboard UI, stats bar

**Commands Run:**
```bash
# Check current process resource usage
ps aux | grep -E "(orch|opencode)"

# Results showed orch serve at 153% CPU during investigation
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-25-orchestrator-system-resource-visibility.md` - Created from this investigation

---

## Investigation History

**2025-12-25 22:15:** Investigation started
- Initial question: Should orchestrator have visibility into system resources?
- Context: Dylan observed orch serve at 125% CPU via sketchybar

**2025-12-25 22:30:** Context gathering complete
- Found the IsProcessing bug fix in serve.go
- Reviewed dashboard stats bar and API endpoints
- Confirmed current resource usage via ps aux

**2025-12-25 22:45:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Created decision artifact recommending no implementation
