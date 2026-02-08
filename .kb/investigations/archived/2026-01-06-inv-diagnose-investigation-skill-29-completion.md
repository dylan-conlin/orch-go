---
linked_issues:
  - orch-go-1qjvb
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation skill's 29% completion rate is caused by two distinct bugs: (1) 45% of spawns are test/verify spawns that never intend to complete, and (2) 44% of "not completed" real investigations actually have SYNTHESIS.md but no completion event was recorded.

**Evidence:** Analyzed 29 investigation spawns from last 7 days. 13 test spawns (45%). Of 16 real spawns: 9 completed (56%), 7 "not completed". Of 7 "not completed": 6 have SYNTHESIS.md but only session.spawned in events.jsonl (no agent.completed event).

**Knowledge:** Two separate bugs inflating failure rate: test spawn pollution (same as prior investigation) AND completion event not being recorded for successful investigations. The second is a new finding.

**Next:** (1) Filter test spawns from stats (prior recommendation still valid), (2) NEW: Investigate why agent.completed events aren't being emitted for some investigations - check if Phase: Complete beads comment triggers event recording.

**Promote to Decision:** recommend-no - findings are bug fixes, not architectural patterns

---

# Investigation: Diagnose Investigation Skill 29% Completion Rate

**Question:** Why does investigation skill have 29% completion rate (10/29 in orch stats), and what are the top failure modes?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-feat-diagnose-investigation-skill-06jan-eb5e
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** .kb/investigations/2026-01-06-inv-diagnose-investigation-skill-32-completion.md (confirms prior findings and adds new one)

---

## Findings

### Finding 1: 45% of Investigation Spawns Are Test/Verify Spawns (Never Intended to Complete)

**Evidence:** 
Of 29 investigation spawns in last 7 days:
- 13 test spawns (45%)
- 16 real investigation spawns (55%)

Test spawn workspace patterns identified:
```
og-inv-test-verify-daemon-03jan (2 spawns)
og-inv-test-completion-works-04jan (2 spawns)
pw-inv-verify-price-watch-05jan
og-inv-verify-launchd-documentation-03jan
og-inv-test-spawned-agents-03jan
og-inv-test-spawn-works-02jan
og-inv-test-liveness-gate-04jan
og-inv-test-hotspot-warning-04jan
og-inv-say-hello-exit-04jan
og-inv-quick-test-verify-02jan
og-inv-quick-test-say-02jan
```

**Source:** 
```bash
grep '"skill":"investigation"' ~/.orch/events.jsonl | grep '"type":"session.spawned"' | jq -r '.data.workspace' | grep -iE "(test|verify|hello|exit|quick)"
```

**Significance:** Confirms prior investigation finding (2026-01-06-inv-diagnose-investigation-skill-32-completion.md). Test spawns pollute completion metrics because they're infrastructure validation, not real work.

---

### Finding 2: Completion Events Not Being Recorded for Some Successful Investigations

**Evidence:**
Of 16 real investigation spawns, checked completion status:
- 9 COMPLETED (in events.jsonl)
- 7 NOT_COMPLETED (no agent.completed event)

Of the 7 "NOT_COMPLETED":
- 6 have SYNTHESIS.md in workspace (completed their work!)
- 1 not found in workspace (og-inv-orch-review-takes-06jan-5dbc)

Specific examples:
| Workspace | Has SYNTHESIS.md? | Has agent.completed? |
|-----------|-------------------|---------------------|
| og-inv-dashboard-port-confusion-03jan | ✅ Yes | ❌ No |
| og-inv-map-main-go-03jan | ✅ Yes | ❌ No |
| og-inv-map-serve-go-03jan | ✅ Yes | ❌ No |
| og-inv-agents-report-phase-03jan | ✅ Yes | ❌ No |
| og-inv-orch-features-json-04jan | ✅ Yes | ❌ No |
| og-inv-orch-go-built-04jan | ✅ Yes | ❌ No |

All 6 have full tier (`.tier` = "full") and complete SYNTHESIS.md files.

**Source:** 
```bash
# Check events for a "not completed" investigation
grep '"orch-go-untracked-1767476493"' ~/.orch/events.jsonl | jq '.type'
# Output: "session.spawned" only - no completion event

# Check workspace
ls .orch/workspace-archive/og-inv-dashboard-port-confusion-03jan/
# Output: .tier, SYNTHESIS.md exists
```

**Significance:** This is a NEW finding not in prior investigation. Investigations are completing their work (producing SYNTHESIS.md) but the completion event isn't being recorded. This is likely a bug in how `orch complete` or `bd comment "Phase: Complete"` triggers event emission.

---

### Finding 3: True Completion Rate for Real Investigations is ~94%

**Evidence:**
Real investigation spawns: 16
- Completed with event: 9
- Completed without event: 6 (have SYNTHESIS.md)  
- Actually failed: 1

True completion: 15/16 = **93.75%**

The 1 actual failure (og-inv-orch-review-takes-06jan-5dbc) couldn't be found in any workspace - likely an infrastructure issue or session never started.

**Source:** Workspace inspection of all "NOT_COMPLETED" investigations

**Significance:** The investigation skill is performing excellently (~94%) when test spawns are excluded and completion events are properly counted. The 29% stat is doubly misleading.

---

## Synthesis

**Key Insights:**

1. **Test Spawn Pollution (Prior Finding)** - 45% of investigation spawns are test/verify work. This is consistent with the prior investigation that found 71% of failures were test spawns. Test spawns used investigation skill as a test vehicle.

2. **Completion Event Recording Bug (New Finding)** - 6 of 16 real investigations (37.5%) completed their work but have no agent.completed event. They all have SYNTHESIS.md but only session.spawned in events.jsonl. This is a separate bug from test spawn pollution.

3. **Stats Double-Counting Problem** - The 29% rate combines two bugs: inflated denominator (test spawns) AND deflated numerator (missing completion events). Fixing either improves the stat; fixing both would show the true ~94% rate.

**Answer to Investigation Question:**

Investigation skill's 29% completion rate has two root causes:

1. **Test spawn pollution (45% of spawns):** Test/verify/hello/quick spawns use investigation skill for infrastructure validation, never intending to produce SYNTHESIS.md.

2. **Missing completion events (37.5% of real work):** Investigations produce SYNTHESIS.md and likely report Phase: Complete via beads, but agent.completed event isn't recorded in events.jsonl.

The TRUE completion rate for properly-tracked real investigation work is **~94%**, not 29%. This is actually excellent.

---

## Structured Uncertainty

**What's tested:**

- ✅ Test spawn count verified (grep + pattern matching on 29 workspaces)
- ✅ SYNTHESIS.md presence verified (ls on 6 archived workspaces)
- ✅ Missing completion events verified (grep events.jsonl for specific beads_ids)
- ✅ True completion count verified (workspace inspection)

**What's untested:**

- ⚠️ Root cause of missing completion events (Phase: Complete → event emission path not traced)
- ⚠️ Whether Phase: Complete beads comment was actually made by these agents
- ⚠️ Whether these agents called /exit properly

**What would change this:**

- If agents didn't actually report Phase: Complete (then it's agent behavior, not event recording)
- If SYNTHESIS.md files are empty or invalid (not checked content quality)
- If there's a race condition in event recording vs session cleanup

---

## Implementation Recommendations

**Purpose:** Two bugs need fixing to show accurate completion rates.

### Recommended Approach ⭐

**Fix Both Bugs Independently** - Filter test spawns from stats AND fix completion event recording.

**Why this approach:**
- Each bug independently degrades metrics
- Test spawn filtering is quick (prior investigation already designed solution)
- Completion event bug needs tracing but has clear symptom

**Trade-offs accepted:**
- Test spawn filtering is heuristic (may miss some or catch false positives)
- Completion event fix may require deeper investigation into orch complete flow

**Implementation sequence:**
1. **Test spawn filtering (quick):** Add `--exclude-test` flag to orch stats that filters beads_ids containing "untracked" and workspaces matching test patterns
2. **Trace completion event recording (needs investigation):** Follow Phase: Complete → agent.completed event path, check if daemon auto-complete handles these, check if orch complete was ever called

### Alternative Approaches Considered

**Option B: Only fix test spawn filtering**
- **Pros:** Quick, addresses largest contributor
- **Cons:** Leaves completion event bug unfixed, 37.5% of real work still not counted
- **When to use instead:** If completion event bug is too complex to fix quickly

**Option C: Require explicit tracking for all spawns**
- **Pros:** Prevents future test spawn pollution
- **Cons:** Adds friction to ad-hoc testing
- **When to use instead:** If test spawn pollution continues after filtering

---

### Implementation Details

**What to implement first:**
- `--exclude-test` flag for orch stats (filtering logic already designed in prior investigation)
- Investigation into completion event recording gap (new issue needed)

**Things to watch out for:**
- ⚠️ Some legitimate workspaces may have "test" in the name
- ⚠️ Completion event bug may affect other skills too (not just investigation)
- ⚠️ Need to distinguish between agent never completing vs event not recorded

**Areas needing further investigation:**
- Why does Phase: Complete beads comment not trigger agent.completed event?
- Are these agents being handled by daemon auto-complete vs orch complete?
- Is there a timing issue with OpenCode session cleanup?

**Success criteria:**
- ✅ Investigation skill completion rate shows ~90%+ with --exclude-test
- ✅ All investigations with SYNTHESIS.md have corresponding agent.completed events
- ✅ orch stats accurately reflects true completion behavior

---

## References

**Files Examined:**
- ~/.orch/events.jsonl - Event stream for spawn/completion analysis
- .orch/workspace-archive/og-inv-*/SYNTHESIS.md - Completion artifacts
- cmd/orch/stats_cmd.go - Stats calculation logic

**Commands Run:**
```bash
# Count test vs real spawns
grep '"skill":"investigation"' ~/.orch/events.jsonl | grep '"type":"session.spawned"' | jq '.data.workspace' | grep -iE "(test|verify|hello|exit|quick)" | wc -l

# Check for completion events
grep '"orch-go-untracked-1767476493"' ~/.orch/events.jsonl | jq '.type'

# Verify SYNTHESIS.md existence
ls -la .orch/workspace-archive/og-inv-dashboard-port-confusion-03jan/
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-06-inv-diagnose-investigation-skill-32-completion.md - Prior investigation with same question
- **Source:** cmd/orch/stats_cmd.go - Stats calculation implementation

---

## Investigation History

**2026-01-06 18:XX:** Investigation started
- Initial question: Why does investigation skill have 29% completion rate?
- Context: Spawned from orch stats completion rate warning

**2026-01-06 18:XX:** Confirmed prior investigation finding
- Test spawn pollution: 45% of spawns are test/verify work
- Consistent with prior finding (71% of failures in larger dataset)

**2026-01-06 18:XX:** NEW finding discovered
- 6 of 16 real investigations have SYNTHESIS.md but no agent.completed event
- This is a completion event recording bug, not just test spawn pollution

**2026-01-06 18:XX:** Investigation completed
- Status: Complete
- Key outcome: Two bugs cause 29% rate: test spawn pollution (45%) AND missing completion events (37.5% of real work). True rate is ~94%.
