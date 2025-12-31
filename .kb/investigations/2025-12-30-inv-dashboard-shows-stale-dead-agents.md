<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard and CLI use different thresholds (3 min vs 30 min) for determining agent "dead" status, causing status discrepancy - not a bug, a design inconsistency.

**Evidence:** Dashboard marks agents "dead" after 3 min inactive (opencode.StaleSessionThreshold), CLI shows "running" up to 30 min inactive. Both are working as designed but using different definitions.

**Knowledge:** Dead agents are intentionally shown in "Active Agents" section (< 4h old) to get user attention. This is confusing UX, not a data bug.

**Next:** Align CLI and dashboard thresholds, or improve UX to clarify "dead" status distinction from "active".

---

# Investigation: Dashboard Shows Stale Dead Agents

**Question:** Why does dashboard show stale/dead agents as "Active" with "Dead - needs attention" while CLI shows correct data?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** og-debug-dashboard-shows-stale-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** .kb/investigations/2025-12-28-debug-dashboard-shows-stale-agent-data.md (same topic, deeper investigation)

---

## Findings

### Finding 1: Different Thresholds Between Dashboard and CLI

**Evidence:** 
- Dashboard uses `deadThreshold = opencode.StaleSessionThreshold` (3 minutes) - serve.go:784
- CLI uses `maxIdleTime = 30 * time.Minute` for "active" consideration - main.go:2661
- Result: Agents inactive 3-30 minutes show as "dead" on dashboard but "running" on CLI

**Source:** 
- `cmd/orch/serve.go:784`: `deadThreshold := opencode.StaleSessionThreshold`
- `cmd/orch/main.go:2661`: `const maxIdleTime = 30 * time.Minute`
- `pkg/opencode/client.go:386`: `const StaleSessionThreshold = 3 * time.Minute`

**Significance:** The task description said "CLI shows correct data" but this is a matter of perspective. CLI has a more lenient threshold (30 min) while dashboard catches dead agents faster (3 min). Neither is "wrong" - they're different views.

---

### Finding 2: Dead Agents Intentionally Shown in Active Section

**Evidence:**
- `activeAgents` store includes dead agents: `$agents.filter((a) => a.status === 'active' || a.status === 'idle' || a.status === 'dead' || a.status === 'stalled')`
- Comment explains: "dead and stalled agents are shown in the active section too (with visual distinction) so the user knows they need attention"
- UI shows "Dead - needs attention" with 💀 icon

**Source:**
- `web/src/lib/stores/agents.ts:227-228`
- `web/src/lib/components/agent-card/agent-card.svelte:424-443`

**Significance:** This is intentional UX design, not a bug. Dead agents need user attention (orch abandon or respawn), so they're prominently displayed.

---

### Finding 3: Dashboard Correctly Discovers Cross-Project Agents

**Evidence:**
- Dashboard uses `discoverAllProjectDirs()` to find sessions from ALL project directories
- CLI only queries current project directory + global sessions
- Result: Dashboard shows pw-* agents from price-watch project, CLI doesn't

**Source:**
- `cmd/orch/serve.go:721`: `allProjectDirs := discoverAllProjectDirs()`
- `cmd/orch/main.go:2630`: Only queries `projectDir` and global

**Significance:** Dashboard has broader visibility across projects, which is correct behavior. CLI's narrower scope is a separate issue.

---

### Finding 4: Dead Agent Filtering is Working Correctly

**Evidence:**
- Dead agents > 4 hours old are filtered out (deadDisplayThreshold)
- Current dead agents are all < 1 hour old:
  - orch-go-t8pl: 0.5 hours old
  - orch-go-gxwu: 0.5 hours old
  - pw-i1gh: 0.5 hours old
  - pw-k7sx: 0.7 hours old

**Source:**
- `cmd/orch/serve.go:786-788`: `deadDisplayThreshold := 4 * time.Hour`
- API response shows updated_at for dead agents

**Significance:** The "stale" agents shown are actually recent agents that became dead. No old/historical agents are incorrectly persisting.

---

## Synthesis

**Key Insights:**

1. **Threshold Discrepancy (Not a Bug)** - Dashboard and CLI have different definitions of "active" vs "dead". Dashboard is more aggressive (3 min) at detecting dead agents, CLI is more lenient (30 min). This causes confusion but isn't incorrect behavior.

2. **Intentional UX Pattern** - Dead agents appear in "Active Agents" section by design, with "Dead - needs attention" indicator. The goal is to ensure users notice and address stuck agents. This may be confusing for users expecting a clean separation.

3. **Cross-Project Visibility Difference** - Dashboard sees agents from all projects while CLI only sees current project. This is a feature gap in CLI, not a dashboard bug.

**Answer to Investigation Question:**

The dashboard is NOT showing "stale" agents incorrectly. It's working as designed:
1. Agents that haven't had activity in > 3 minutes are marked "dead" (correct)
2. Dead agents < 4 hours old are shown in Active section (intentional - for user attention)
3. Cross-project agents appear (correct - discovered from all project directories)

The "bug" is really a UX/design issue:
- Threshold mismatch between CLI (30 min) and dashboard (3 min) causes confusion
- Dead agents in "Active" section might be unclear to users

---

## Structured Uncertainty

**What's tested:**

- ✅ Dashboard correctly marks agents "dead" after 3 min inactive (verified: API returns status=dead for 4 agents)
- ✅ CLI shows agents "running" up to 30 min inactive (verified: orch status showed t8pl/gxwu as running)
- ✅ Dead agents < 4h shown, > 4h filtered (verified: all dead agents are < 1 hour old)
- ✅ Cross-project sessions discovered (verified: pw-* from price-watch project visible)

**What's untested:**

- ⚠️ User comprehension of "Dead - needs attention" indicator
- ⚠️ Whether 4-hour threshold is appropriate for all use cases
- ⚠️ Impact of aligning thresholds on user experience

**What would change this:**

- Finding would be wrong if dead agents older than 4 hours are appearing (they're not)
- Finding would be wrong if agents with recent activity are marked dead (they're not)
- Finding would be wrong if there's an OpenCode restart persistence bug (sessions are persisted correctly)

---

## Implementation Recommendations

**Purpose:** Address the threshold/UX discrepancy between dashboard and CLI.

### Recommended Approach ⭐

**Align CLI threshold with dashboard** - Change CLI's `maxIdleTime` from 30 minutes to use `opencode.StaleSessionThreshold` (3 minutes)

**Why this approach:**
- Consistent definition of "active" vs "dead" across all tools
- Dashboard's 3-minute threshold is more practical (agents constantly do work)
- CLI will match dashboard view, reducing user confusion

**Trade-offs accepted:**
- CLI will show fewer "running" agents (only truly active ones)
- Users may see more "phantom" or dead agents in status output

**Implementation sequence:**
1. Update `cmd/orch/main.go` to use `opencode.StaleSessionThreshold` instead of 30 minutes
2. Consider adding dead agent count to CLI status output
3. Add visual distinction in CLI for "dead" agents (like dashboard does)

### Alternative Approaches Considered

**Option B: Reduce dead display threshold**
- **Pros:** Fewer dead agents cluttering dashboard
- **Cons:** Users may miss stuck agents that need attention
- **When to use instead:** If users complain about too many dead agents showing

**Option C: Separate "Needs Attention" section**
- **Pros:** Clearer UI - active agents vs dead agents
- **Cons:** More UI work, may reduce visibility of dead agents
- **When to use instead:** If users find current merged view confusing

**Rationale for recommendation:** Option A is minimal code change with high impact on consistency. The dashboard's 3-minute threshold is well-reasoned (agents are constantly active when working).

---

### Implementation Details

**What to implement first:**
- Align CLI threshold with dashboard (single constant change)
- Add dead agent indicator to CLI output (optional enhancement)

**Things to watch out for:**
- ⚠️ CLI may now show fewer agents as "running" - users may notice the change
- ⚠️ Need to update any documentation that references the 30-minute threshold

**Areas needing further investigation:**
- Whether cross-project visibility should be added to CLI status
- UX research on whether users understand the "dead" concept

**Success criteria:**
- ✅ CLI and dashboard show same agents as "running" vs "dead"
- ✅ No false positives (active agents marked dead)
- ✅ Users understand what "dead" means and what action to take

---

## References

**Files Examined:**
- `cmd/orch/serve.go:770-890` - Dashboard API handleAgents logic
- `cmd/orch/main.go:2611-2960` - CLI runStatus function
- `pkg/opencode/client.go:382-386` - StaleSessionThreshold constant
- `web/src/lib/stores/agents.ts:227-228` - Frontend activeAgents filter
- `web/src/lib/components/agent-card/agent-card.svelte:424-443` - Dead agent UI

**Commands Run:**
```bash
# Check OpenCode session states
curl -s "http://localhost:4096/session?directory=/path" | python3 -c "..."

# Check API response
curl -s http://localhost:3348/api/agents | python3 -c "..."

# Check CLI output
/Users/dylanconlin/go/bin/orch status
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-28-debug-dashboard-shows-stale-agent-data.md - Prior investigation on similar issue
- **Investigation:** .kb/investigations/2025-12-28-inv-dashboard-shows-stale-agent-data.md - Another prior investigation

---

## Investigation History

**2025-12-30 17:00:** Investigation started
- Initial question: Dashboard shows stale/dead agents as "Active" after OpenCode restart
- Context: Spawned from beads issue orch-go-3ukg

**2025-12-30 17:15:** Threshold discrepancy identified
- Found CLI uses 30 min, dashboard uses 3 min
- Both working as designed but with different definitions

**2025-12-30 17:25:** Cross-project visibility confirmed
- Dashboard discovers agents from all projects
- CLI only shows current project

**2025-12-30 17:30:** Investigation completed
- Status: Complete
- Key outcome: Not a bug - design/UX issue with different thresholds and intentional dead agent visibility
