<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux agents (Claude CLI escape hatch) need enhanced dashboard visibility through better status determination using beads comments and pane activity detection.

**Evidence:** Current code (serve_agents.go:546-591) adds tmux agents with status="active" always; beads Phase comments already work for status if beads ID is extracted correctly from window names.

**Knowledge:** The Four-Layer State Model establishes beads as source of truth for completion. Tmux is a UI layer with limited data. Token usage isn't available for Claude CLI agents - this is an architectural constraint, not a gap to fill.

**Next:** Implement the recommended enhancements: beads Phase lookup for tmux agents, pane activity detection for dead vs active, spawn time capture for runtime calculation.

**Promote to Decision:** recommend-no (implementation details, not architectural pattern)

---

# Investigation: Dashboard Add Tmux Session Visibility

**Question:** How should the dashboard integrate tmux data source alongside OpenCode sessions to provide visibility for Claude CLI agents (escape hatch spawns)?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Problem Statement

Claude CLI agents (spawned with `--backend claude --tmux`) exist in tmux windows but lack visibility parity with OpenCode agents in the dashboard. Currently:

| Data Point | OpenCode Agents | Claude CLI Agents |
|------------|-----------------|-------------------|
| Status | active/idle/dead/completed | Always "active" |
| Phase | From beads comments | Not fetched |
| Tokens | From OpenCode API | Not available |
| Session ID | Yes | N/A |
| Runtime | From session created_at | Not captured |
| Window target | If --tmux used | Yes |

**Why this matters:** The escape hatch exists for critical infrastructure work where the primary path might fail. These agents need the SAME visibility as OpenCode agents - they're often working on the most important tasks.

---

## Findings

### Finding 1: Tmux agents already get beads data if properly identified

**Evidence:** In serve_agents.go:578-588, tmux-only agents collect beads IDs for batch fetch:

```go
// Collect beads ID for batch fetch
if beadsID != "" && !seenBeadsIDs[beadsID] {
    beadsIDsToFetch = append(beadsIDsToFetch, beadsID)
    seenBeadsIDs[beadsID] = true
    // For cross-project agent visibility: use cached PROJECT_DIR
    if agentProjectDir := wsCache.lookupProjectDir(beadsID); agentProjectDir != "" {
        beadsProjectDirs[beadsID] = agentProjectDir
    }
}
```

**Source:** cmd/orch/serve_agents.go:578-588

**Significance:** The infrastructure exists to fetch beads data for tmux agents. The gap is that status remains hardcoded to "active" instead of using Phase from beads comments.

---

### Finding 2: Window name format enables beads ID extraction

**Evidence:** Window names follow pattern: `{emoji} {workspace-name} [{beads-id}]`
Example: `🔬 og-inv-test-18jan-ab12 [orch-go-xyz]`

The tmux package has `extractBeadsIDFromWindowName()` which parses this format.

**Source:** cmd/orch/serve_agents.go:555, pkg/tmux/tmux.go:399-414

**Significance:** Beads ID extraction already works. Tmux agents with proper window names get beads data fetched.

---

### Finding 3: Dead detection requires pane activity analysis

**Evidence:** OpenCode agents use `timeSinceUpdate` from session metadata:
```go
// serve_agents.go:454-458
if timeSinceUpdate > deadThreshold {
    status = "dead" // No activity for 3+ minutes = dead
}
```

Tmux agents don't have equivalent metadata - the window existing says nothing about whether the agent is actively processing.

**Source:** cmd/orch/serve_agents.go:454-458

**Significance:** Dead detection for tmux agents requires new mechanism: either pane content analysis or external activity tracking (like file modification times in workspace).

---

### Finding 4: Token usage is architecturally unavailable for Claude CLI

**Evidence:** Claude CLI (Max subscription) doesn't expose token usage via API. The usage dashboard at https://console.anthropic.com tracks aggregate usage, not per-session.

**Source:** Model access knowledge from `.kb/models/model-access-spawn-paths.md`

**Significance:** This is a constraint, not a gap. Tmux agents won't have token visibility. This is acceptable because:
1. Max subscription is flat-rate ($200/mo)
2. Token budgeting isn't necessary for unlimited usage
3. Progress tracking via beads Phase comments is more valuable anyway

---

## Decision Forks

### Fork 1: Data Source Architecture

**Question:** Should tmux be a first-class data source or remain a fallback?

**Options:**
- A: Keep tmux as fallback (current) - only add agents not in OpenCode
- B: Elevate tmux to first-class - query tmux first, enrich with OpenCode
- C: Unify through registry - central state tracks all spawn types

**Substrate says:**
- Principle (Graceful Degradation): "Core functionality works without optional layers" - tmux should add value, not be required
- Model (Four-Layer State): "Agent state exists across four independent layers" - each layer has different authority
- Constraint: "Post-registry lifecycle uses 4 state sources: OpenCode sessions, tmux windows, beads issues, workspaces" - registry was removed due to false positive detection

**RECOMMENDATION:** Option A (keep as fallback) because:
- OpenCode agents already have rich data from API
- Tmux fallback catches escape-hatch agents that aren't in OpenCode
- Registry was explicitly removed as a central state source due to drift
- Fallback approach matches Graceful Degradation principle

**Trade-off accepted:** Slightly different data flows for different spawn types
**When this would change:** If OpenCode backend is deprecated entirely

---

### Fork 2: Status Determination for Tmux Agents

**Question:** How should we determine status for tmux-only agents?

**Options:**
- A: Always show "active" (current)
- B: Parse tmux pane content for activity signals
- C: Use beads comments (Phase: X) as primary, pane activity as secondary
- D: Use workspace file modification times

**Substrate says:**
- Principle (Observation Infrastructure): "If the system can't observe it, the system can't manage it"
- Model (Dashboard Agent Status): "Priority Cascade - Phase check highest priority"
- Constraint: "Phase: Complete is the only reliable signal"
- Constraint: "Session idle ≠ agent complete"

**RECOMMENDATION:** Option C (beads Phase primary, pane activity secondary) because:
- Beads Phase is already the canonical source for all agents
- Pane activity can distinguish "dead" (no output for 3+ min) from "active"
- Matches the Priority Cascade model already used for OpenCode agents
- Avoids false positives from session idle detection

**Trade-off accepted:** Pane capture adds latency (~50ms per agent)
**When this would change:** If Claude CLI exposed session state via API

---

### Fork 3: Runtime Calculation for Tmux Agents

**Question:** How do we calculate runtime without session creation timestamp?

**Options:**
- A: No runtime shown (accept the gap)
- B: Parse workspace `.spawn_time` file
- C: Use window creation time from tmux
- D: Use beads issue created_at

**Substrate says:**
- Self-Describing Artifacts: Spawn creates workspace with metadata files
- Existing implementation: spawn writes `.spawn_time` to workspace directory

**RECOMMENDATION:** Option B (use .spawn_time file) because:
- Already exists in spawn workflow
- Accurate to actual spawn time
- Doesn't require tmux API changes
- Consistent with workspace as source of spawn metadata

**Trade-off accepted:** Requires workspace lookup (already done for beads ID)
**When this would change:** If spawn workflow changes to not write .spawn_time

---

### Fork 4: Pane Activity Detection Implementation

**Question:** How should we detect if a tmux agent is actively processing?

**Options:**
- A: Capture pane content, look for TUI activity indicators
- B: Track last pane content hash, detect change
- C: Check if Claude TUI shows "processing" state
- D: Use workspace file modification times as proxy

**Substrate says:**
- Existing function: `tmux.IsOpenCodeReady()` already parses pane content for TUI indicators
- Constraint: "Observation gaps are P1 bugs"

**RECOMMENDATION:** Option A (pane content analysis) with Option D as fallback because:
- IsOpenCodeReady() proves pane content parsing works
- Claude TUI shows different states (prompt box, processing indicator)
- Workspace file mtime is a good fallback for non-TUI agents
- Matches existing pattern from tmux package

**Trade-off accepted:** Pane parsing is fragile to TUI format changes
**When this would change:** If Claude CLI provides state API

---

## Synthesis

**Key Insights:**

1. **Beads Phase already works for tmux agents** - The infrastructure exists, but status isn't derived from Phase comments. Adding `Phase` to the response and updating status based on Phase: Complete would provide completion detection.

2. **"Dead" detection is the real gap** - Tmux agents stuck in a crashed state show as "active". Pane activity detection (no output for 3+ minutes) would catch this, matching the deadThreshold used for OpenCode agents.

3. **Token usage is an architectural constraint, not a fixable gap** - Claude CLI doesn't expose tokens. Accept this. Progress is better tracked through Phase comments anyway.

**Answer to Investigation Question:**

The dashboard should integrate tmux data by:

1. **Leveraging existing beads Phase lookup** (already partially implemented) - tmux agents with beads IDs already get Phase from beads batch fetch. Status should derive from this.

2. **Adding pane activity detection** for dead vs active status - A new function `tmux.GetPaneActivityAge()` that captures pane content and compares to previous capture (or checks for Claude TUI "waiting" state).

3. **Reading .spawn_time for runtime** - Workspace already contains spawn metadata.

4. **Accepting no token visibility** for Claude CLI agents - This is a constraint from the architecture, documented in dashboard as intentional.

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads ID extraction from window names works (verified: extractBeadsIDFromWindowName exists and is called)
- ✅ Beads batch fetch includes tmux agents (verified: code at lines 578-588)
- ✅ Pane content capture works (verified: GetPaneContent, IsOpenCodeReady exist)
- ✅ .spawn_time file is written at spawn (verified: spawn workflow writes metadata files)

**What's untested:**

- ⚠️ Status derivation from Phase for tmux agents (not implemented)
- ⚠️ Pane activity age detection (function doesn't exist yet)
- ⚠️ Runtime display for tmux agents from .spawn_time (not connected)
- ⚠️ Dashboard UI correctly renders partial data (needs frontend verification)

**What would change this:**

- Claude CLI exposing session state API would enable richer data
- Beads Phase protocol changing would affect status derivation
- Tmux output format changing would break pane parsing

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Incremental Enhancement** - Add beads Phase to tmux agents, then pane activity detection, then runtime.

**Why this approach:**
- Beads Phase integration is low-risk (data already fetched)
- Pane activity detection can be added independently
- Runtime is nice-to-have, can be done last
- Each step provides value independently

**Trade-offs accepted:**
- Multiple PRs vs single large change
- Tmux agents will have partial data until fully implemented

**Implementation sequence:**

1. **Phase 1: Beads Phase for Status** - In serve_agents.go, for tmux-only agents, look up Phase from batch-fetched beads data and set status based on Phase: Complete or Phase: X
   - Files: cmd/orch/serve_agents.go
   - Estimated: 50 lines changed

2. **Phase 2: Pane Activity Detection** - Add GetPaneActivityAge() to tmux package, use 3-minute threshold for dead detection
   - Files: pkg/tmux/tmux.go, cmd/orch/serve_agents.go
   - Estimated: 100 lines new, 30 lines changed

3. **Phase 3: Runtime from .spawn_time** - Read spawn_time from workspace, calculate runtime for tmux agents
   - Files: cmd/orch/serve_agents.go
   - Estimated: 30 lines new

4. **Phase 4: Dashboard UI** - Ensure frontend gracefully handles missing fields (tokens, session_id) for tmux agents
   - Files: web/src/lib/components/AgentCard.svelte (or similar)
   - Estimated: 20 lines changed

### Alternative Approaches Considered

**Option B: Registry-based tracking**
- **Pros:** Central state for all spawn types
- **Cons:** Registry was explicitly removed due to state drift issues; would reintroduce the same problems
- **When to use instead:** Never - this approach failed before

**Option C: Claude CLI API wrapper**
- **Pros:** Would provide native session state, token usage
- **Cons:** Claude CLI has no API; would require forking Claude Code and adding endpoints
- **When to use instead:** If Anthropic adds API to Claude CLI

**Rationale for recommendation:** Incremental enhancement using existing data sources (beads, pane content, workspace files) is low-risk and aligns with established patterns. Registry was tried and removed. Claude CLI API doesn't exist.

---

### Implementation Details

**What to implement first:**
- Beads Phase integration for tmux agents is highest priority - provides completion detection
- This is low-risk as the data is already being fetched

**Things to watch out for:**
- ⚠️ Window name format variations - ensure beads ID extraction handles edge cases
- ⚠️ Pane content parsing is fragile - Claude TUI format may change
- ⚠️ Performance of pane capture at scale - consider caching or sampling for many tmux agents

**Areas needing further investigation:**
- How does dashboard frontend handle null/missing fields? (tokens, session_id)
- What's the performance impact of pane capture for 10+ tmux windows?

**Success criteria:**
- ✅ Tmux agents show Phase in dashboard (from beads comments)
- ✅ Tmux agents show completed status when Phase: Complete
- ✅ Tmux agents show dead status when no pane activity for 3+ minutes
- ✅ Tmux agents show runtime (from .spawn_time)
- ✅ Dashboard doesn't break when tmux agent data is partial

---

## References

**Files Examined:**
- cmd/orch/serve_agents.go:1-600 - Agent API response and tmux fallback logic
- pkg/tmux/tmux.go - Tmux window management and pane content capture
- .kb/models/dashboard-architecture.md - Dashboard data flow
- .kb/models/agent-lifecycle-state-model.md - Four-layer state model
- .kb/models/dashboard-agent-status.md - Priority Cascade for status
- .kb/models/escape-hatch-visibility-architecture.md - Dual-window visibility
- .kb/guides/status-dashboard.md - Status and dashboard guide

**Commands Run:**
```bash
# Search for tmux integration
grep -rn "tmux" cmd/orch/*.go | head -30

# Check spawn backend patterns
grep -rn "backend.*claude" cmd/orch/spawn_cmd.go
```

**Related Artifacts:**
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Establishes four-layer architecture
- **Model:** `.kb/models/dashboard-agent-status.md` - Priority Cascade approach
- **Guide:** `.kb/guides/status-dashboard.md` - Current status logic
- **Decision:** Beads as source of truth for completion (embedded in models)

---

## Investigation History

**2026-01-18 [start]:** Investigation started
- Initial question: How to integrate tmux data source for Claude CLI agents
- Context: Escape hatch spawns need dashboard visibility

**2026-01-18 [mid]:** Explored codebase
- Found existing tmux fallback at serve_agents.go:546-591
- Identified beads Phase lookup is already implemented but status not derived
- Confirmed token usage is architecturally unavailable

**2026-01-18 [end]:** Investigation completed
- Status: Complete
- Key outcome: Recommend incremental enhancement using existing data sources (beads Phase, pane activity, .spawn_time)
