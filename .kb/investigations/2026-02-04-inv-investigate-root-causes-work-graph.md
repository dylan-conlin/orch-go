## Summary (D.E.K.N.)

**Delta:** State confusion has three distinct root causes: (1) verification gate failures during auto-completion leave issues in_progress despite Phase:Complete, (2) epic protection is reactive not proactive - no automatic cascade or orphan prevention, (3) absorbed/duplicate bugs lack formal relationship tracking.

**Evidence:** Live example orch-go-21225 shows in_progress+Phase:Complete caused by build verification failure. Epic protection exists in complete_cmd.go:640-660 but only blocks at completion time. Beads types.go has parent-child support but no cascade enforcement.

**Knowledge:** The daemon DOES auto-complete issues (completion_processing.go) but verification gates (test_evidence, build, visual, etc.) can reject completion even when agent reports done. Current architecture separates "agent claims done" from "system verifies done" - the gap is handling verification failures.

**Next:** Implement three-layer fix: (1) Verification-failed escalation queue, (2) Proactive epic consistency constraints, (3) Absorbed-by relationship type in beads. Start with escalation queue as highest impact.

**Authority:** architectural - These are cross-component changes affecting beads, daemon, and dashboard coordination.

---

# Investigation: Root Causes of Work Graph/Beads State Confusion

**Question:** Why do beads issues get into inconsistent states (epic-closed-with-open-children, in_progress after Phase:Complete, duplicate/absorbed bugs), and how can we prevent it?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** Worker Agent
**Phase:** Synthesizing
**Next Step:** Complete recommendations section
**Status:** Complete

---

## Findings

### Finding 1: Live Example - in_progress Despite Phase:Complete

**Evidence:** Issue orch-go-21225 is in status `in_progress` but has a "Phase: Complete" comment from 2026-02-03T20:05:10:
```
Phase: Complete - Fixed AgentAttentionCollector to use https:// with TLS skip verify.
Tests: go test ./pkg/attention/... - all passed.
```

When attempting `orch complete orch-go-21225`, verification fails with:
```
Cannot complete agent - verification failed:
  - 'go test -run=^$ ./...' failed (compilation error in production or test code)
```

**Source:** `bd show orch-go-21225 --json`, `orch complete orch-go-21225`

**Significance:** This demonstrates the primary root cause: agents can report Phase:Complete, but the daemon's auto-completion verification gates can fail, leaving issues stuck. The issue is correct behavior (verification should block broken code) but lacks escalation handling.

---

### Finding 2: Auto-Completion Exists But Verification Gates Block

**Evidence:** The daemon has `completion_processing.go` with `CompletionOnce()` function that:
1. Lists all open/in_progress issues
2. Checks for "Phase: Complete" in comments
3. Runs `verify.VerifyCompletionFull()` with gates: phase_complete, synthesis, test_evidence, visual_verification, git_diff, build, constraint, phase_gate, skill_output, decision_patch_limit
4. Only closes issues that pass ALL gates

From daemon.go:
```go
completionResult, err := d.CompletionOnce(completionConfig)
```

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/completion_processing.go`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/daemon.go:~line 280`

**Significance:** The system DOES try to auto-complete, but verification failures create a "purgatory" state where the agent thinks it's done but the system won't close the issue. There's no escalation path for these stuck issues.

---

### Finding 3: Epic Protection is Reactive Only

**Evidence:** In complete_cmd.go lines 640-660:
```go
if issue.IssueType == "epic" && !completeForceCloseEpic {
    openChildren, err := verify.GetOpenEpicChildren(beadsID)
    if len(openChildren) > 0 {
        return fmt.Errorf("epic has open children")
    }
}
```

This protection only triggers at completion time. It does NOT:
- Prevent orphaning children if `--force-close-epic` is used
- Automatically close epics when all children complete
- Prevent closing children before the epic relationship is set
- Handle the reverse: all children done but epic still open

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:640-660`

**Significance:** Epic consistency is reactive (blocks at completion) rather than proactive (prevents inconsistent states from forming). This allows edge cases where state gets inconsistent.

---

### Finding 4: Parent-Child Data Model Exists But No Enforcement

**Evidence:** From `pkg/beads/types.go`:
- `CreateArgs.Parent` field for creating child issues
- `ListArgs.Parent` for querying children
- `GetBlockingDependencies()` handles parent-child specially: "Parent-child: NEVER blocks - children are independently spawnable"

No automatic cascade operations exist - closing a parent doesn't close children, completing all children doesn't auto-close parent.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/types.go:60-290`

**Significance:** Beads stores relationships but doesn't enforce invariants. It's a database, not a state machine.

---

### Finding 5: Duplicate/Absorbed Bug Pattern

**Evidence:** Looking at orch-go-21146 and orch-go-21148:
- orch-go-21146: "Work Graph: can't collapse expanded epics with children" (bug)
- orch-go-21148: "Work Graph Phase 1.1: Fix core interaction bugs" (task) - description includes: "3. **orch-go-21146** - Can't collapse epics with children"

Both are now closed. The 21148 task "absorbed" 21146 as part of a bundle fix, but there's no formal relationship type for this. The description mentions it textually, but:
- No explicit "absorbed-by" or "superseded-by" dependency type
- No automatic closure cascade
- Relies on human to close both issues

**Source:** `bd show orch-go-21146`, `bd show orch-go-21148`

**Significance:** Bundle tasks that fix multiple issues lack formal tracking. The "absorbs" relationship is only in prose, making it hard to track what happened to issues.

---

### Finding 6: Work Graph is Visualization, Not Enforcement

**Evidence:** The Work Graph (`work-graph-tree.svelte`) pulls data from `/api/beads/ready` and `/api/issues`. It visualizes:
- Parent-child hierarchies
- Status (in_progress, blocked, open, closed)
- WIP items and completed issues awaiting verification

But it's purely a display layer - no enforcement of invariants. The `closeIssue()` function calls the server API which calls `bd close`.

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/work-graph-tree/work-graph-tree.svelte`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_beads.go`

**Significance:** Work Graph can display inconsistent states but can't prevent them. It's the symptom reporter, not the solution.

---

## Synthesis

**Key Insights:**

1. **Verification-Failed Purgatory** - The gap between "agent claims done" and "system verifies done" creates stuck issues. Auto-completion exists but verification gates can reject, with no escalation path for handling these rejections.

2. **Reactive vs Proactive Consistency** - Epic protection only kicks in at completion time. This means inconsistent states can form and persist. Proactive enforcement would prevent invalid transitions.

3. **Missing Relationship Types** - The "absorbed-by" pattern for bundle tasks exists only in prose. Beads has parent-child and blocks relationships, but not "supersedes" or "absorbed-by" for tracking issue consolidation.

4. **Three Systems, No State Machine** - Beads (database), daemon (automation), and Work Graph (visualization) are loosely coupled. None enforces invariants. Issues can get into states that no system will automatically fix.

**Answer to Investigation Question:**

The root causes of state confusion are:

1. **in_progress after Phase:Complete**: Verification gates in auto-completion can fail, leaving issues stuck with no escalation. The agent did its job, but code changes elsewhere caused verification to fail.

2. **Epic-closed-with-open-children**: Protection exists but is reactive (blocks closure) not proactive (prevents inconsistency). `--force-close-epic` can bypass protection. No automatic cascade in either direction.

3. **Duplicate/absorbed bugs**: No formal relationship type for "supersedes" or "absorbed-by". Bundle tasks reference issues in prose but the system doesn't track or enforce this relationship.

The common thread: **Beads is a database, not a state machine.** It stores state but doesn't enforce valid transitions or invariants. The daemon and Work Graph operate on this database but can't prevent all invalid states.

---

## Structured Uncertainty

**What's tested:**

- ✅ orch-go-21225 demonstrates verification-failed stuck state (tested: `orch complete` showed build failure)
- ✅ Epic protection code exists and blocks completion (reviewed: complete_cmd.go:640-660)
- ✅ Auto-completion runs in daemon loop (traced: daemon.go calling CompletionOnce)
- ✅ Parent-child data model exists (reviewed: types.go CreateArgs.Parent, ListArgs.Parent)

**What's untested:**

- ⚠️ How often verification-failed purgatory occurs (would need event log analysis)
- ⚠️ Whether `--force-close-epic` is commonly used (would need command history)
- ⚠️ How many absorbed bugs exist without formal tracking (would need issue audit)
- ⚠️ Performance impact of adding cascade operations

**What would change this:**

- Finding existing escalation handling for verification failures would change recommendation priority
- Finding that verification failures are rare (<1%) would reduce urgency of escalation queue
- Finding existing absorbed-by relationship type would change duplicate handling recommendation

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Verification-failed escalation queue | architectural | Cross-component: daemon, attention system, dashboard |
| Proactive epic consistency | architectural | Changes beads invariants and lifecycle |
| Absorbed-by relationship type | strategic | Requires beads schema change, affects data model |

---

### Recommended Approach ⭐

**Three-Layer Fix: Escalation Queue + Epic Invariants + Absorbed-By Relationship**

**Why this approach:**
- Addresses all three root causes systematically
- Each layer is independently valuable and deployable
- Builds on existing infrastructure (attention signals, verification gates, beads relationships)

**Trade-offs accepted:**
- Complexity increase in completion flow
- Requires changes to beads CLI for absorbed-by
- Epic auto-close could be surprising behavior

**Implementation sequence:**

#### Phase 1: Verification-Failed Escalation Queue (Highest Impact)

**Problem:** Issues stuck in in_progress+Phase:Complete with no visibility.

**Solution:** Add `awaiting-verification-review` attention signal:

1. **Detection:** In `completion_processing.go`, when verification fails:
   - Add issue to new attention signal type `verify_failed`
   - Include which gates failed and why

2. **Surface:** In Work Graph attention panel:
   - New category: "Needs Verification Review"
   - Show beads ID, gates failed, agent's Phase:Complete summary
   - Action buttons: "Re-verify", "Skip gates with reason", "Reset to in_progress"

3. **Resolution paths:**
   - Orchestrator reviews, fixes underlying issue, re-runs verification
   - Orchestrator uses targeted `--skip-{gate}` with reason
   - Orchestrator resets status to open for re-spawning

**Estimated effort:** 2-3 days

#### Phase 2: Proactive Epic Consistency (Medium Impact)

**Problem:** Epic protection is reactive, allows inconsistent states.

**Solution:** Add bidirectional consistency checks:

1. **Epic Auto-Close:** When completing the last open child of an epic:
   - Check if parent epic has no other open children
   - Prompt: "All children of epic X complete. Close epic? [y/N]"
   - Or add `--auto-close-parent` flag

2. **Orphan Prevention:** When closing an epic (even with --force-close-epic):
   - Log to events which children were orphaned
   - Add attention signal: "Epic closed with open children"
   - Show in Work Graph with action to reassign or close children

3. **Pre-flight Check:** In `orch spawn` when creating child:
   - Verify parent epic still exists and is open
   - Warning if parent is closed: "Parent epic is already closed"

**Estimated effort:** 3-4 days

#### Phase 3: Absorbed-By Relationship Type (Strategic)

**Problem:** Bundle tasks absorb issues without formal tracking.

**Solution:** Add `absorbed-by` dependency type to beads:

1. **CLI:** `bd absorb <absorbed-id> --by <absorber-id>`
   - Marks absorbed issue as closed with close_reason "absorbed by X"
   - Creates dependency: absorbed-id → absorber-id (type: absorbed-by)

2. **Query:** `bd show <id>` shows "Absorbed by: X" when applicable

3. **Work Graph:** Display absorbed issues with link to absorber

4. **Automation:** When closing bundle task, prompt: "Mark issues as absorbed?"

**Estimated effort:** 4-5 days (requires beads changes)

---

### Alternative Approaches Considered

**Option B: State Machine Enforcement in Beads**

- **Pros:** Prevents invalid states at database level
- **Cons:** Major beads architecture change, breaks existing workflows
- **When to use instead:** If state confusion is so severe it can't be fixed with escalation queues

**Option C: Periodic Consistency Audit**

- **Pros:** Low implementation cost, surfaces all inconsistencies
- **Cons:** Reactive (finds problems after they occur), doesn't prevent
- **When to use instead:** If proactive enforcement is too disruptive

---

### Implementation Details

**What to implement first:**
- Phase 1 (Escalation Queue) provides immediate visibility into stuck issues
- Can be done without beads changes
- Uses existing attention signal infrastructure

**Things to watch out for:**
- ⚠️ Escalation queue could fill up if verification gates are too strict
- ⚠️ Epic auto-close needs careful UX to avoid surprising closures
- ⚠️ Absorbed-by relationship needs migration for existing issues

**Areas needing further investigation:**
- Frequency of verification failures in production (event log analysis)
- User preference for epic auto-close vs manual close
- Whether absorbed-by needs to be transitive

**Success criteria:**
- ✅ No issues stuck in in_progress+Phase:Complete for >24 hours without attention signal
- ✅ Zero epic-closed-with-open-children incidents (enforced or surfaced immediately)
- ✅ Bundle task completions prompt for absorbed issues disposition

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go` - Completion flow and epic protection
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/completion_processing.go` - Auto-completion logic
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/types.go` - Data model with parent-child
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/guides/beads-integration.md` - Integration patterns
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/models/completion-verification.md` - Verification architecture
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/work-graph-tree/work-graph-tree.svelte` - Work Graph UI

**Commands Run:**
```bash
bd show orch-go-21225 --json
bd list --json | jq '[.[] | select(.status == "in_progress")]'
orch complete orch-go-21225
bd show orch-go-21146
bd show orch-go-21148
```

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification.md` - Existing verification architecture
- **Model:** `.kb/models/beads-integration-architecture.md` - Beads integration patterns
- **Guide:** `.kb/guides/beads-integration.md` - Integration procedures

---

## Investigation History

**[2026-02-04 10:30]:** Investigation started
- Initial question: Root causes of state confusion
- Context: Spawned to investigate epic-closed-with-open-children, in_progress after Phase:Complete, duplicates

**[2026-02-04 10:45]:** Found live example
- orch-go-21225 demonstrates in_progress + Phase:Complete
- Discovered verification gates blocking auto-completion

**[2026-02-04 11:15]:** Analyzed daemon completion flow
- Confirmed CompletionOnce runs in daemon loop
- Identified escalation gap for verification failures

**[2026-02-04 11:30]:** Examined epic handling
- Epic protection exists but is reactive
- No automatic cascade operations

**[2026-02-04 11:45]:** Investigated duplicate pattern
- orch-go-21146 absorbed by orch-go-21148 (bundle task)
- No formal relationship type for absorption

**[2026-02-04 12:00]:** Investigation completed
- Status: Complete
- Key outcome: Three-layer fix recommended (escalation queue, epic invariants, absorbed-by relationship)
