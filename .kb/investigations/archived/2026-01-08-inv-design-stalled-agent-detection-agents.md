---
linked_issues:
  - orch-go-y4kbm
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed minimal stalled agent detection using ONE signal (phase unchanged for 15+ minutes) with advisory-only surfacing in Needs Attention section.

**Evidence:** Post-mortem shows prior attempt (Dec 27-Jan 2) failed due to complexity (multiple thresholds, multiple states); 25-28% investigation found ~21 abandonments were "stuck at Planning for 19+ minutes" - a clear signal that phase stagnation indicates stalled agents.

**Knowledge:** Stalled ≠ Dead. Dead = no heartbeat (3 min silence, already implemented). Stalled = has heartbeat but not progressing. Phase change is the simplest progress signal. Keep it advisory (surface, don't auto-abandon) to avoid complexity trap.

**Next:** Implement - add `isStalled` field to AgentAPIResponse, surface in Needs Attention component with orange indicator, single 15-minute threshold.

**Promote to Decision:** recommend-yes - Establishes the principle "one threshold, one signal, advisory only" for agent health monitoring.

---

# Investigation: Design Stalled Agent Detection

**Question:** How should we detect agents that are active (have heartbeat) but not making progress, given that the prior attempt (Dec 27-Jan 2) failed due to complexity?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-arch-design-stalled-agent-08jan-9b08
**Phase:** Complete
**Next Step:** None - design ready for implementation
**Status:** Complete

---

## Problem Framing

### Design Question

How do we detect and surface "stalled" agents (active but not progressing) without repeating the Dec 27-Jan 2 complexity spiral?

### Success Criteria

1. **Simple** - ONE threshold, ONE signal (not multiple thresholds like 1min/3min/1hr)
2. **Advisory only** - Surface in Needs Attention, don't auto-abandon
3. **Clear definition** - Unambiguous criteria for what "progress" means
4. **Minimal code** - <100 lines of new code
5. **No new states** - Use existing status field with flag, not new `stalled` status

### Constraints

- **Learned from failure**: Dec 27-Jan 2 spiral was caused by adding complexity (multiple time thresholds, `dead` and `stalled` states) - lesson: keep it simple
- **Dead detection exists**: 3-minute heartbeat threshold already implemented and working
- **Beads integration exists**: Phase parsing from comments already in `pkg/verify/beads_api.go`
- **No auto-abandon**: Advisory only - human decides what to do with stalled agents

### Scope

**In scope:**
- Detect when active agent hasn't changed phase for N minutes
- Surface in dashboard Needs Attention section
- Single configurable threshold

**Out of scope:**
- Multiple thresholds for different staleness levels
- Auto-abandonment or auto-restart
- Token usage patterns (too complex)
- File edit tracking (not available in current architecture)

---

## Findings

### Finding 1: Prior Attempt Failed Due to Complexity

**Evidence:** Post-mortem at `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` documents:
- Agent states grew from 5 to 7 (added `dead`, `stalled`)
- 3 time-based thresholds added (1min, 3min, 1hr)
- 347 commits in 6 days, 40 "fix:" commits
- Result: complete loss of trust in the system

**Source:** `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md:11-13`

**Significance:** The failure wasn't the concept of stalled detection - it was the complexity. Multiple thresholds, multiple states, and auto-actions created an unmanageable system.

---

### Finding 2: Phase Change is the Clearest Progress Signal

**Evidence:** Analysis of abandonments shows clear patterns:
- "Stuck at Planning for 19+ minutes" - ~21 abandonments
- "Stalled - no phase comments after 4+ minutes" - common pattern
- "Session stuck in Planning for 3 attempts" - another pattern

All these patterns key on **phase not changing**.

**Source:** `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md:77`, events.jsonl analysis

**Significance:** Phase change is already tracked via beads comments. An agent that reports "Phase: Planning" but never progresses to "Phase: Implementing" is clearly stalled. This is simpler than tracking file edits or token usage.

---

### Finding 3: Dead vs Stalled are Distinct Failure Modes

**Evidence:** Current implementation has:
- **Dead**: `timeSinceUpdate > 3 minutes` in `serve_agents.go:437-438`
- **Stalled**: Not implemented

These are orthogonal:
- Dead = no heartbeat (agent crashed/killed/disconnected)
- Stalled = has heartbeat but same phase for extended time

**Source:** `cmd/orch/serve_agents.go:406-441`

**Significance:** We need stalled detection because dead detection won't catch agents that are "alive" but stuck in an infinite loop, waiting for external input, or hitting repeated errors without crashing.

---

### Finding 4: NeedsAttention Component Already Exists

**Evidence:** Dashboard has `web/src/lib/components/needs-attention/needs-attention.svelte` that displays:
- Errors
- Blocked issues  
- Pending reviews
- Dead agents

**Source:** `web/src/lib/components/needs-attention/needs-attention.svelte`

**Significance:** Stalled agents fit naturally into this component. No new UI sections needed.

---

## Exploration (3 Approaches)

### Option A: Phase-Based Stalled Detection (⭐ Recommended)

**Mechanism:**
- Track `phase_reported_at` timestamp (when last phase comment was posted)
- If `now - phase_reported_at > 15 minutes` AND agent is active → `isStalled = true`
- Surface in Needs Attention as "Stalled at [Phase] for [duration]"

**Implementation:**
1. Add `PhaseReportedAt` field to beads phase status response
2. Add `isStalled` boolean to AgentAPIResponse
3. In serve_agents.go, calculate stalled status after fetching beads comments
4. Frontend: add stalled agents to Needs Attention component with orange indicator

**Pros:**
- Simple (ONE threshold, ONE signal)
- Uses existing infrastructure (beads comments already parsed)
- Advisory only (just surfaces the info)
- Clear actionable guidance ("Agent stuck at Planning for 25 min")

**Cons:**
- Requires beads comments (won't work for untracked spawns)
- Can't detect stalled agents that never reported any phase

**Complexity:** ~50-75 lines of new code

---

### Option B: Activity-Based Stalled Detection

**Mechanism:**
- Track file modification time in workspace
- If workspace files unchanged for N minutes AND agent is active → stalled

**Pros:**
- Doesn't require beads
- Works for untracked spawns

**Cons:**
- More complex to implement (file watching)
- Less precise (some agents read more than write)
- Workspace may not exist for all agents
- Can't distinguish "thinking" from "stuck"

**Complexity:** ~150-200 lines of new code

---

### Option C: Token Usage Pattern Detection

**Mechanism:**
- Monitor token consumption rate via OpenCode API
- If tokens consumed but no output for N minutes → stalled

**Pros:**
- Precise (directly measures "work")
- Model-agnostic

**Cons:**
- Requires OpenCode API changes to expose token history
- Complex to tune thresholds
- Rate limiting can cause false positives
- Significantly more implementation complexity

**Complexity:** ~300+ lines of new code, API changes

---

## Synthesis

### Recommendation ⭐

**Option A: Phase-Based Stalled Detection** with 15-minute threshold.

### Why This Approach

1. **Simplicity first** - One threshold (15 min), one signal (phase unchanged)
2. **Uses existing infrastructure** - Phase parsing already exists in `pkg/verify/beads_api.go`
3. **Advisory only** - Surfaces in Needs Attention, no auto-actions
4. **Addresses real problem** - "Stuck at Planning for 19+ minutes" was the top abandonment pattern
5. **Respects post-mortem lessons** - Avoids multiple thresholds, multiple states, auto-abandon

### Trade-offs Accepted

- **Won't detect untracked spawns** - Acceptable because untracked spawns are rare and have separate visibility mechanisms
- **Won't detect pre-phase-report stalls** - Acceptable because agents that never report ANY phase are already flagged by `orch doctor` as "stalled sessions"
- **15 minutes may be too long for some cases** - Acceptable because advisory-only means no harm from false negatives; human can intervene earlier if watching dashboard

### When This Would Change

- If token usage API becomes available easily → Option C might provide more precision
- If untracked spawns become common → Option B might be needed as supplement
- If 15 minutes proves too short (false positives) → increase to 20-30 minutes

---

## Structured Uncertainty

**What's tested:**

- ✅ Phase parsing from beads comments works (verified: existing implementation in `pkg/verify/beads_api.go`)
- ✅ Dead detection with 3-minute threshold works (verified: restored in `serve_agents.go:406-409`)
- ✅ NeedsAttention component exists and handles dead agents (verified: read component source)

**What's untested:**

- ⚠️ 15-minute threshold is the right duration (educated guess, may need tuning)
- ⚠️ Stalled agents surface correctly in dashboard (needs implementation)
- ⚠️ No false positives from agents legitimately in long phases

**What would change this:**

- If 15 minutes causes too many false positives (agents legitimately thinking)
- If phase comments aren't reliable enough for timing
- If user feedback indicates different threshold needed

---

## Implementation Recommendations

### Recommended Approach ⭐

**Phase-Based Stalled Detection** - Add `isStalled` flag when active agent has same phase for 15+ minutes.

### Implementation Sequence

1. **Server-side stalled detection** (serve_agents.go)
   - Add `PhaseReportedAt *time.Time` to PhaseStatus struct in beads_api.go
   - Parse timestamp from beads comment metadata
   - In serve_agents.go, after fetching beads comments:
     - If agent is active AND phase_reported_at > 15 minutes ago → `isStalled = true`
   - Add `IsStalled bool` field to AgentAPIResponse

2. **Frontend surfacing** (needs-attention.svelte)
   - Add `stalledAgents` derived store filtering `isStalled: true`
   - Add section to Needs Attention: "⚠️ Stalled Agents" with orange indicator
   - Show: agent name, phase, time stalled, action hint

3. **Stats bar indicator** (stats-bar.svelte)
   - Include stalled count in "+N need attention" indicator

### File Targets

- `pkg/verify/beads_api.go` - Add PhaseReportedAt to PhaseStatus
- `cmd/orch/serve_agents.go` - Calculate isStalled, add to response
- `web/src/lib/stores/agents.ts` - Add stalledAgents derived store  
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Add stalled section

### Acceptance Criteria

- [ ] Agents with same phase for 15+ minutes show `isStalled: true` in API
- [ ] Dashboard Needs Attention shows stalled agents with orange indicator
- [ ] Stats bar shows stalled count in "+N need attention"
- [ ] No new agent states added (status remains active/idle/completed/dead)
- [ ] Threshold is configurable (environment variable or constant)

### Out of Scope

- Auto-abandonment of stalled agents
- Multiple stalled levels (warning/critical)
- Token usage tracking
- File modification tracking

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go:406-441` - Dead detection implementation
- `pkg/verify/beads_api.go:114-133` - Phase parsing from comments
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Existing attention component
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Lessons from failed attempt

**Commands Run:**
```bash
# Find stalled patterns in events
grep -i "stuck at Planning\|stalled" ~/.orch/events.jsonl | head -20

# Analyze abandonment reasons
grep '"type":"agent.abandoned"' ~/.orch/events.jsonl | grep -o '"reason":"[^"]*"' | sort | uniq -c
```

**Related Artifacts:**
- **Post-mortem:** `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` - Why Dec 27-Jan 2 spiral happened
- **Investigation:** `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md` - Shows ~21 abandonments were "stuck at Planning"
- **Investigation:** `.kb/investigations/2026-01-08-inv-restore-dead-agent-detection-surfacing.md` - Dead detection restoration (precursor to this)

---

## Investigation History

**2026-01-08 09:00:** Investigation started
- Initial question: Design stalled agent detection without repeating Dec 27-Jan 2 complexity spiral
- Context: Dead detection restored, stalled is the next phase

**2026-01-08 09:30:** Problem framing complete
- Established success criteria: ONE threshold, ONE signal, advisory only
- Identified constraints from post-mortem lessons

**2026-01-08 09:45:** Exploration complete
- Evaluated 3 approaches: phase-based, activity-based, token-based
- Recommended phase-based for simplicity

**2026-01-08 10:00:** Investigation completed
- Status: Complete
- Key outcome: Design for minimal stalled detection using phase-unchanged-for-15-min signal with advisory-only surfacing
