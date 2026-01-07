## Summary (D.E.K.N.)

**Delta:** Orchestrator skill's 16.7% completion rate is primarily BY DESIGN - orchestrators run until context exhaustion or session interruption, not until "Phase: Complete". Additionally, completion event correlation is broken due to missing session_id in tmux spawns and workspace-based matching gaps.

**Evidence:** 
1. Code at `stats_cmd.go:33-39` explicitly recognizes orchestrator/meta-orchestrator as `CoordinationSkill` that runs "until context exhaustion, not complete discrete tasks"
2. Orchestrator completion requires SESSION_HANDOFF.md, not Phase: Complete via beads (per `verify/check.go:240-243`)
3. Of 24 orchestrator spawns in 7 days, many are via tmux (no session_id) with "untracked" beads IDs - these cannot correlate to spawn events
4. `agent.completed` events for orchestrators lack skill field and use workspace-based matching, but workspace names differ between spawn and complete

**Knowledge:** 
1. Orchestrator sessions have DIFFERENT lifecycle than workers - they're coordination roles, not tasks
2. The stats system already knows this (`coordinationSkills` map) but still displays them in the same table as task skills
3. The 16.7% rate conflates two issues: (a) legitimate design difference and (b) event correlation bugs

**Next:** 
1. Don't treat this as a bug to fix - it's by design
2. Consider separating coordination skills from task skills in `orch stats` display
3. Optionally: fix event correlation for orchestrators (add skill to agent.completed events, improve workspace matching)

**Promote to Decision:** recommend-no (this is documentation of existing design, not a new architectural choice)

---

# Investigation: Diagnose Orchestrator Skill 18% Completion Rate

**Question:** Is the orchestrator skill's low completion rate (16.7%) a problem to fix, or expected behavior by design?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Worker agent (og-feat-diagnose-orchestrator-skill-06jan-79b6)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Orchestrators Are Classified as Coordination Skills by Design

**Evidence:** `stats_cmd.go:33-39` explicitly defines orchestrator and meta-orchestrator as coordination skills:

```go
// coordinationSkills lists skills that are coordination roles, not completable tasks.
// These are excluded from the completion rate warning because they're interactive sessions
// designed to run until context exhaustion, not complete discrete tasks.
var coordinationSkills = map[string]bool{
	"orchestrator":      true,
	"meta-orchestrator": true,
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/stats_cmd.go:33-39`

**Significance:** The system was deliberately designed to recognize that orchestrators don't complete like workers. They run until:
- Context exhaustion
- Session interruption by user
- Or replacement by a new orchestrator session

This is not a bug but an intentional lifecycle difference.

---

### Finding 2: Different Completion Criteria for Orchestrator Tier

**Evidence:** Workers complete via "Phase: Complete" beads comment. Orchestrators complete via SESSION_HANDOFF.md:

```go
// verifyOrchestratorCompletion checks if an orchestrator session is ready for completion.
// Orchestrators have different verification requirements than workers:
//   - No beads-dependent phase checks (orchestrators manage sessions, not issues)
//   - SESSION_HANDOFF.md instead of SYNTHESIS.md
//   - Session end verification instead of Phase: Complete
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:292-296`

**Significance:** Orchestrators fundamentally work differently:
- No beads issue tracking (they manage sessions, not issues)
- SESSION_HANDOFF.md replaces SYNTHESIS.md
- Sessions are closed by the level above (meta-orchestrator or Dylan), not self-closed

---

### Finding 3: Tmux Spawns Break Event Correlation

**Evidence:** Many orchestrator spawns via tmux have empty `session_id`:

```json
{"type":"session.spawned","timestamp":1767580570,"data":{"beads_id":"orch-go-vizg","session_id":"","session_name":"orchestrator","skill":"orchestrator","spawn_mode":"tmux",...}}
```

But `agent.completed` events don't include the skill field:

```json
{"type":"agent.completed","timestamp":1767642002,"data":{"forced":false,"orchestrator":true,"reason":"Orchestrator session completed","untracked":true,"workspace":"og-work-update-meta-orchestrator-05jan"}}
```

The stats aggregator at `stats_cmd.go:274-278` and `stats_cmd.go:289-306` tries to match:
1. For `session.completed`: By session_id (empty for tmux → no match)
2. For `agent.completed`: By beads_id (untracked → no match)

**Source:** 
- `~/.orch/events.jsonl` (orchestrator events)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/stats_cmd.go:274-310`

**Significance:** Even when orchestrators DO complete properly (with SESSION_HANDOFF.md), the completion events aren't attributed back to the skill because:
1. Tmux spawns don't preserve session_id
2. Untracked spawns (common for orchestrators) don't have correlatable beads_id
3. `agent.completed` events don't include the skill field

---

### Finding 4: Many Orchestrator Sessions Are Test/Experimental

**Evidence:** Examining orchestrator spawn tasks in events.jsonl:

```
"task":"Test the new spawnable orchestrator infrastructure"
"task":"Test spawnable orchestrator with tmux default"
"task":"Test tmux spawn into orchestrator session"
"task":"Test template wiring - verify SESSION_HANDOFF.template.md is copied"
"task":"test meta-orchestrator lifecycle observation"
"task":"Test orchestrator session for E2E verification"
```

**Source:** `~/.orch/events.jsonl` (grepped for orchestrator skill spawns)

**Significance:** A significant portion of "orchestrator" spawns are actually infrastructure tests during development, not production orchestration sessions. These are spawned, verified to work, then abandoned without formal completion.

---

### Finding 5: Workspaces Show Mixed Completion States

**Evidence:** Checking orchestrator workspaces:

```
meta-orch-continue-meta-orch-06jan-2c9a       - orchestrator tier, NO SESSION_HANDOFF.md
meta-orch-resume-last-meta-06jan-08ba         - orchestrator tier, HAS SESSION_HANDOFF.md (3311 bytes)
meta-orch-resume-meta-orch-06jan-9172         - orchestrator tier, HAS SESSION_HANDOFF.md (3010 bytes)
meta-orch-strategic-session-review-05jan-c3eb - orchestrator tier, HAS SESSION_HANDOFF.md (5955 bytes)
```

**Source:** `.orch/workspace/meta-orch-*` directories examined via bash

**Significance:** Some orchestrator sessions complete properly with SESSION_HANDOFF.md, but these completions aren't being counted because:
1. The completion events don't include skill field
2. No session_id correlation for tmux spawns
3. "untracked" flag prevents beads_id matching

---

## Synthesis

**Key Insights:**

1. **Not a Bug, a Design Difference** - Orchestrator and meta-orchestrator skills are explicitly designed as coordination roles that run until context exhaustion, not task completions. The code already recognizes this via `coordinationSkills` map.

2. **Completion Tracking IS Broken (But Intentionally Loose)** - Even when orchestrators complete properly (SESSION_HANDOFF.md exists), the events aren't correlating because:
   - Tmux spawns lack session_id
   - Untracked spawns (common for orchestrators) can't correlate by beads_id
   - `agent.completed` events don't include skill field

3. **Mixed Metric is Misleading** - Displaying orchestrators in the same skill breakdown table as task skills creates false concern. The 16.7% completion rate isn't alarming because:
   - It's by design (coordination ≠ task completion)
   - Many are test/experimental spawns
   - Some completions aren't counted due to correlation bugs

**Answer to Investigation Question:**

The orchestrator skill's 16.7% completion rate is **primarily by design**, not a bug to fix. Orchestrators are coordination sessions that:
- Run until context exhaustion or interruption
- Don't complete via "Phase: Complete"
- Are often replaced by new sessions rather than formally completed

However, there IS a secondary issue: when orchestrators DO complete properly (via SESSION_HANDOFF.md), the completions aren't being attributed because of event correlation gaps.

**Recommendation:** Don't try to "fix" the completion rate. Instead:
1. Consider separating coordination skills from task skills in stats display
2. Optionally improve event correlation (add skill to agent.completed, improve workspace matching)

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrators are classified as CoordinationSkill (verified: read stats_cmd.go:33-39)
- ✅ Orchestrator completion uses SESSION_HANDOFF.md (verified: read verify/check.go:240-243)
- ✅ Tmux spawns have empty session_id (verified: grep events.jsonl)
- ✅ agent.completed events lack skill field (verified: grep events.jsonl)
- ✅ Some orchestrator workspaces have SESSION_HANDOFF.md completed (verified: ls workspace)

**What's untested:**

- ⚠️ Whether improving event correlation would significantly change the rate (not benchmarked)
- ⚠️ What percentage of test spawns vs production spawns (not categorized)
- ⚠️ Whether users expect orchestrators to "complete" like workers (not surveyed)

**What would change this:**

- If Dylan says orchestrators SHOULD complete like workers, the design needs to change
- If event correlation is fixed and rate is still low, there might be a real completion problem

---

## Implementation Recommendations

### Recommended Approach: Display Separation

**Purpose:** Make it clear that coordination skills are different from task skills.

**Option A: Separate Section in Stats Output** - Add a new section for "Coordination Sessions" that shows spawn count and active sessions, without a misleading completion rate.

**Why this approach:**
- Matches the intent of `coordinationSkills` map already in code
- Doesn't require fixing event correlation (which may be intentionally loose for orchestrators)
- Makes the metric meaningful

**Trade-offs accepted:**
- Doesn't fix the event correlation issue
- Some orchestrator completions will still be uncounted

**Implementation sequence:**
1. Add conditional in `outputStatsText()` to separate coordination skills
2. Show different metrics: Sessions spawned, Active, Duration (no completion rate)
3. Keep task skills with completion rate

### Alternative Approaches Considered

**Option B: Fix Event Correlation**
- **Pros:** Accurate completion tracking for all skills
- **Cons:** Orchestrators still shouldn't have high completion rates by design
- **When to use instead:** If there's a need to audit orchestrator session lifecycles

**Option C: Remove Coordination Skills from Stats**
- **Pros:** Simple, removes misleading data
- **Cons:** Loses visibility into orchestrator activity
- **When to use instead:** If orchestrator metrics aren't useful

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/stats_cmd.go` - Stats aggregation and coordinationSkills classification
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go` - Orchestrator completion verification
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/orchestrator_context.go` - Orchestrator lifecycle design
- `~/.orch/events.jsonl` - Event log analysis
- `.orch/workspace/meta-orch-*` - Orchestrator workspace states

**Commands Run:**
```bash
# Check stats output
orch stats
orch stats --json

# Analyze orchestrator events
cat ~/.orch/events.jsonl | grep -E '"skill":"(orchestrator|meta-orchestrator)"'
cat ~/.orch/events.jsonl | grep 'agent.completed.*orchestrator'

# Check workspace states
for ws in .orch/workspace/meta-orch-*; do
  cat "$ws/.tier" 2>/dev/null
  ls -la "$ws/SESSION_HANDOFF.md" 2>/dev/null
done
```

---

## Investigation History

**2026-01-06 18:51:** Investigation started
- Initial question: Is 16.7% orchestrator completion rate a bug or by design?
- Context: orch stats shows orchestrator skill with low completion rate

**2026-01-06 18:55:** Discovered coordinationSkills classification
- Found stats_cmd.go already recognizes orchestrators as non-completing coordination roles

**2026-01-06 18:58:** Analyzed event correlation
- Found tmux spawns have empty session_id, breaking correlation
- Found agent.completed events lack skill field

**2026-01-06 19:05:** Investigation completed
- Status: Complete
- Key outcome: Low rate is by design, not a bug. Orchestrators are coordination sessions, not tasks.
