<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** When an epic is labeled `triage:ready`, the daemon should auto-include its unblocked children in the spawn queue, inferring the label inheritance from the parent.

**Evidence:** Tested with `bd list --parent orch-go-lv3yx` showing 4 children exist; `orch daemon preview` shows epic as "type 'epic' not spawnable" while children show "missing label 'triage:ready'".

**Knowledge:** Daemon already has parent-child awareness (blocking behavior in `GetBlockingDependencies`), and beads CLI supports `--parent` filter. Adding daemon-side inference avoids beads knowing about triage workflow (separation of concerns).

**Next:** Implement daemon-side epic child inference in `NextIssue()` to auto-include children when their epic parent has `triage:ready` label and the child is unblocked.

**Promote to Decision:** recommend-no (implementation detail, follows existing pattern of daemon inferring spawning strategy)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Triage Ready Epic Cascade Children

**Question:** When an epic is labeled `triage:ready`, should the label cascade to children (beads feature) or should the daemon infer children from the epic's label (daemon feature)?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** agent (feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Daemon already skips epics as non-spawnable types

**Evidence:** `IsSpawnableType()` in `pkg/daemon/skill_inference.go:10-17` only returns true for `bug`, `feature`, `task`, `investigation`. Epics are explicitly rejected. The `checkRejectionReason()` method returns `"type 'epic' not spawnable"` for epic issues.

**Source:** 
- `pkg/daemon/skill_inference.go:10-17` - IsSpawnableType function
- `pkg/daemon/daemon.go:478-479` - checkRejectionReason using IsSpawnableType
- `cmd/orch/spawn_cmd.go:235-236` - Similar logic in spawn command

**Significance:** The daemon's current behavior is correct - epics SHOULD NOT be spawned directly. The problem is that children of `triage:ready` epics are not being considered.

---

### Finding 2: Beads CLI supports `--parent` filter to list epic children

**Evidence:** `bd list --parent orch-go-lv3yx --json` returns 4 children of the epic. The CLI client in `pkg/beads/cli_client.go:217` already supports the `--parent` argument for `Create()`.

**Source:**
- `bd list --help` shows `--parent string` flag: "Filter by parent issue ID (shows children of specified issue)"
- `pkg/beads/cli_client.go:126-144` - List method (does not yet support --parent filter)
- Manual test: `bd list --parent orch-go-lv3yx --json` returns children

**Significance:** The infrastructure exists in beads to query children. We need to add support in the daemon to query children of `triage:ready` epics and include them in the spawn queue.

---

### Finding 3: User mental model expects "label epic = process entire epic"

**Evidence:** From spawn context: "User mental model: 'Label the epic ready for the daemon' means 'ensure the entire epic is processed now' - not 'label this one non-spawnable thing'."

**Source:** SPAWN_CONTEXT.md problem description

**Significance:** The fix should be in the daemon (not beads) because:
1. Beads is issue tracking - shouldn't know about triage workflow
2. Daemon already makes spawning decisions - this is where triage logic belongs
3. Cascading labels in beads would create data duplication (labels on children that aren't explicitly set)

---

## Synthesis

**Key Insights:**

1. **Daemon-side inference is the right design** - Beads is a general-purpose issue tracker and shouldn't know about daemon-specific concepts like `triage:ready`. The daemon already makes spawning decisions based on type and labels (Finding 1, 3).

2. **Children query mechanism exists** - Beads CLI already supports `bd list --parent <id>`, so the daemon can query children of epics with minimal new infrastructure (Finding 2).

3. **Label inheritance is implicit, not explicit** - Children don't need to have `triage:ready` label themselves. The daemon infers spawnability from their parent's label. This avoids label duplication and maintains single source of truth.

**Answer to Investigation Question:**

The daemon should infer children from the epic's label (daemon feature), NOT cascade labels in beads. This approach:
- Keeps beads generic (no triage workflow knowledge)
- Avoids label duplication on children
- Follows existing daemon pattern of making spawning decisions
- Allows future flexibility (e.g., different processing rules for epic children vs standalone issues)

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd list --parent orch-go-lv3yx --json` returns 4 children (verified: ran command)
- ✅ `IsSpawnableType("epic")` returns false (verified: read code at skill_inference.go:10-17)
- ✅ Daemon rejects epics in checkRejectionReason (verified: read code at daemon.go:478-479)

**What's untested:**

- ⚠️ Performance impact of querying children for each triage:ready epic (needs benchmarking)
- ⚠️ Behavior with deeply nested epics (epic > child-epic > grandchild) - may need recursive handling

**What would change this:**

- Finding would be wrong if beads had a strong reason to own triage workflow (e.g., UI that shows triage status)
- Approach might need revision if epics can have 100+ children (performance concern)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Daemon Epic Child Inference** - Modify daemon's issue queue logic to automatically include children of `triage:ready` epics in the spawn queue.

**Why this approach:**
- Keeps beads generic - no triage workflow knowledge needed
- Follows existing daemon pattern of making spawning decisions
- Avoids label duplication on children

**Trade-offs accepted:**
- Additional beads queries when epics have `triage:ready` label
- Children won't show `triage:ready` label themselves (implicit from parent)

**Implementation sequence:**
1. Add `ListChildren()` function to beads client (uses `bd list --parent <id>`)
2. Modify `listIssuesFunc` or add preprocessing in `NextIssue()` to expand `triage:ready` epics
3. Add tests for epic child inclusion behavior

### Alternative Approaches Considered

**Option B: Beads `--cascade` flag**
- **Pros:** Labels visibly on children, works for any label
- **Cons:** Beads learns about triage workflow, data duplication, need to clean up labels after processing
- **When to use instead:** If we need labels to persist on children for tracking/UI purposes

**Option C: Manual labeling of children**
- **Pros:** No code changes needed
- **Cons:** Poor UX, defeats purpose of epics, error-prone
- **When to use instead:** Never - this is the problem we're solving

**Rationale for recommendation:** Option A keeps separation of concerns clean. The daemon is already the decision-maker for spawning - adding epic child inference follows that pattern. Beads stays generic.

---

### Implementation Details

**What was implemented:**
- Added `Parent` field to `ListArgs` in beads types
- Added `FallbackListByParent()` function to beads client
- Enhanced CLI client's `List()` to support `--parent` flag
- Added `ListEpicChildren()` function to daemon
- Added `expandTriageReadyEpics()` method to daemon
- Updated `NextIssueExcluding()` and `Preview()` to expand epic children
- Updated `checkRejectionReason()` to show helpful message for triage:ready epics

**Edge cases handled:**
- Children already in queue are not duplicated but marked as epic children
- Epics without triage:ready label are not expanded
- No label filter means no expansion needed

**Success criteria (verified):**
- ✅ `orch daemon preview` shows helpful message for triage:ready epics
- ✅ Children of triage:ready epics considered for spawning (even without label)
- ✅ All existing tests continue to pass
- ✅ 7 new tests added and passing

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` - NextIssue, Preview, and checkRejectionReason logic
- `pkg/daemon/skill_inference.go` - IsSpawnableType function
- `pkg/daemon/issue_adapter.go` - ListReadyIssues implementation
- `pkg/beads/types.go` - ListArgs structure
- `pkg/beads/client.go` - Fallback functions
- `pkg/beads/cli_client.go` - CLI client implementation

**Commands Run:**
```bash
# Check epic children
bd list --parent orch-go-lv3yx --json

# Test daemon preview
orch daemon preview

# Run tests
go test ./pkg/daemon/... -count=1
```

**Related Artifacts:**
- **Issue:** orch-go-tuofe - Children shouldn't have blocking dep on parent (separate bug)
- **Workspace:** `.orch/workspace/og-feat-triage-ready-epic-08jan-c08d/`

---

## Investigation History

**2026-01-08 16:00:** Investigation started
- Initial question: How should triage:ready on epic cascade to children?
- Context: User mental model expects labeling epic = processing entire epic

**2026-01-08 16:15:** Design decision made
- Daemon-side inference chosen over beads cascade (separation of concerns)

**2026-01-08 16:30:** Implementation complete
- Status: Complete
- Key outcome: Children of triage:ready epics are auto-included in spawn queue
