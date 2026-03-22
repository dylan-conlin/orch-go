---
stability: foundational
---
## Summary (D.E.K.N.)

**Delta:** System resource visibility in the orchestrator provides marginal value - the 125% CPU bug was already diagnosed via external monitoring (sketchybar), and high CPU/memory in the orchestration layer typically indicates bugs rather than normal operation.

**Evidence:** The `IsSessionProcessing` bug that caused 125% CPU was found via sketchybar, not orch tooling; the fix was applied (serve.go:315-319 comment documents this); system resources are a secondary concern compared to agent status, phase, and Claude usage limits.

**Knowledge:** Orchestration layer focuses on agent coordination (spawn/complete/status), not process management. Resource monitoring already exists externally and is better suited for system-level tools.

**Next:** No implementation needed. If future demand emerges, consider minimal "health indicator" approach (Option B). Document sketchybar integration as the resource monitoring solution.

---

# Decision: Orchestrator System Resource Visibility

**Date:** 2025-12-25
**Status:** Proposed
**Enforcement:** context-only

---

## Context

Dylan observed `orch serve` at 125% CPU via sketchybar plugin. This raised the question: should the orchestrator (`orch status`, dashboard) have visibility into system resources like CPU and memory usage?

**The trigger:** Dylan noticed the spike externally, then diagnosed it as a bug in `orch serve` (making HTTP calls per session when dashboard polled for `is_processing` state). The bug has been fixed (serve.go:315-319).

**The question:** Would built-in resource monitoring have helped? Should we add it?

---

## Options Considered

### Option A: No System Resource Visibility
- **Pros:** 
  - Keeps orchestration layer focused on agent coordination
  - External tools (sketchybar, Activity Monitor, htop) already provide this
  - Less code to maintain
  - CPU/memory in orchestration layer typically means bugs, not normal operation
- **Cons:** 
  - Orchestrator can't detect its own pathological states
  - Requires external monitoring setup

### Option B: Minimal Health Indicator
- **Pros:** 
  - Light-touch: "healthy" vs "degraded" state only
  - Self-diagnostic capability
  - Could alert when orch serve itself is consuming excessive resources
- **Cons:** 
  - Still adds complexity
  - Thresholds are arbitrary (what's "too much" CPU for orch serve?)
  - Doesn't add value if external monitoring is already in place

### Option C: Full Resource Dashboard
Add `/api/resources` endpoint and dashboard panel showing:
- orch serve CPU/memory
- opencode serve CPU/memory  
- Per-agent process stats

- **Pros:**
  - Self-contained monitoring
  - Could help debug runaway agents
- **Cons:**
  - Significant implementation effort
  - Duplicates OS-level tooling
  - Dashboard constraint: "must be usable at 666px width" - resource panel competes for space
  - Polling process stats is itself CPU-intensive (ironic)
  - Agent processes (bun/opencode) are LLM-driven - high CPU is expected

---

## Decision

**Chosen:** Option A - No System Resource Visibility

**Rationale:** The 125% CPU bug Dylan observed:
1. Was found via external monitoring (sketchybar) that already exists
2. Was caused by a bug (excessive HTTP polling), not normal operation
3. Was fixed without needing orchestrator-level resource visibility

The orchestrator's job is agent coordination (spawn, complete, status, phase tracking). System resources are:
- **For orchestration processes (orch serve, daemon):** Should be low; if high, it's a bug to fix, not a metric to display
- **For agent processes (opencode, bun):** High CPU is expected during LLM generation; not actionable

External monitoring (sketchybar, Activity Monitor, htop) is better suited for this because:
- Already exists and is familiar
- System-wide visibility, not just orchestration processes
- Doesn't add complexity to the orchestration layer

**Trade-offs accepted:**
- If external monitoring isn't set up, pathological states may go unnoticed longer
- Orchestrator can't self-diagnose (relies on Dylan or external tooling)

---

## Structured Uncertainty

**What's tested:**
- ✅ 125% CPU bug was diagnosed and fixed via external monitoring (serve.go:315-319 comment)
- ✅ Dashboard already shows actionable metrics (agents, phases, usage, beads, focus)
- ✅ Dylan has sketchybar plugin for resource monitoring

**What's untested:**
- ⚠️ Assumption: external monitoring will always be available
- ⚠️ Assumption: high resource usage in orchestration layer always indicates bugs

**What would change this:**
- Multiple instances of resource issues that external monitoring failed to catch
- Demand for "at a glance" health check in CI/CD or automated contexts
- Orchestration layer running on remote servers without easy external monitoring

---

## Consequences

**Positive:**
- Keeps orchestration layer focused on its core job (agent coordination)
- No additional code to maintain
- No additional dashboard complexity (preserves 666px width constraint)
- Dylan's existing sketchybar integration continues to work

**Risks:**
- If sketchybar breaks or Dylan changes monitoring setup, resource issues may go unnoticed
- Other users of orch-go won't have built-in resource visibility (mitigation: document sketchybar integration)
