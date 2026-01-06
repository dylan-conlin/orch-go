<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The JSON field is `issue_type`, not `type`. Querying for `.type` in jq returns `null` because the field doesn't exist.

**Evidence:** `bd list --json | jq '.[0].type'` returns `null`, while `bd list --json | jq '.[0].issue_type'` returns `"task"` correctly.

**Knowledge:** This is not a serialization bug - the JSON output is correct. The confusion arose from using the wrong field name in jq queries.

**Next:** Close as "could-not-reproduce" - the reported bug is user error (wrong field name), not a code defect.

---

# Investigation: Fix Beads Type Field Showing

**Question:** Why does `bd show` display 'Type: task' but JSON serialization shows `"type": null`?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** AI Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: JSON field name is `issue_type`, not `type`

**Evidence:** 
```bash
$ bd list --json | jq '.[0]'
{
  "id": "orch-go-4v2il",
  "title": "test-type-field",
  "status": "open",
  "priority": 2,
  "issue_type": "task",  # <-- Field name is issue_type
  ...
}
```

**Source:** 
- Ran `bd list --json` and examined output
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/types.go:144` shows `IssueType string json:"issue_type"`
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go:33` shows `IssueType IssueType json:"issue_type,omitempty"`

**Significance:** The JSON serialization uses `issue_type` as the field name, not `type`. Querying for `.type` in jq returns `null` because no such field exists.

---

### Finding 2: The issue description mentions incorrect flag usage

**Evidence:** The issue reproduction says "bd list --format json | jq shows type: null" but:
1. The correct flag for JSON output is `--json`, not `--format json`
2. `--format` is for Go templates or graph formats (digraph, dot)
3. `bd list --format json` produces no output at all

**Source:** `bd list --help` output shows:
```
--format string    Output format: 'digraph' (for golang.org/x/tools/cmd/digraph), 'dot' (Graphviz), or Go template
--json             Output in JSON format
```

**Significance:** Two possible sources of confusion: wrong field name (`.type` vs `.issue_type`) and possibly wrong flag (`--format json` vs `--json`).

---

### Finding 3: bd show correctly displays type, JSON serialization is also correct

**Evidence:**
```bash
$ bd show orch-go-4v2il
...
Type: task
...

$ bd show orch-go-4v2il --json | jq '.[0].issue_type'
"task"
```

Both display and JSON output are consistent - they both show the type correctly.

**Source:** Direct command execution

**Significance:** There is no inconsistency between display and JSON output. Both work correctly when using the right field name.

---

## Synthesis

**Key Insights:**

1. **Field naming convention** - Beads uses `issue_type` (snake_case) in JSON to match Go's json tag convention, while the display shows it as "Type" for readability.

2. **User query error** - When querying JSON for a field named `.type`, jq returns `null` because the field doesn't exist. This is expected jq behavior for missing fields.

3. **No serialization bug exists** - The original report interpreted `null` as a serialization error, but it's actually the expected result of querying a non-existent field.

**Answer to Investigation Question:**

The JSON serialization does NOT show `"type": null`. The JSON field is named `issue_type`, not `type`. When you query `.type` with jq, it returns `null` because that field doesn't exist in the JSON output. This is standard jq behavior, not a serialization bug.

The daemon correctly rejects issues with empty `IssueType` field (the case where `issue_type: ""` in JSON), showing "missing type (required for skill inference)" as the rejection reason. This is working as designed.

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd list --json` outputs correct `issue_type` field (verified: ran command, saw `"issue_type": "task"`)
- ✅ `bd show --json` outputs correct `issue_type` field (verified: ran command, saw `"issue_type": "bug"`)
- ✅ `.type` query returns null as expected (verified: `jq '.[0].type'` returns `null`)
- ✅ `.issue_type` query returns correct value (verified: `jq '.[0].issue_type'` returns `"task"`)

**What's untested:**

- ⚠️ Original session ses_474f behavior (historical, cannot reproduce exact conditions)
- ⚠️ Whether there was ever a race condition in the daemon (no evidence found)

**What would change this:**

- Finding would be wrong if `bd list --json` ever outputs a field literally named `type`
- Finding would be wrong if there's a beads version that serializes differently

---

## Implementation Recommendations

**Purpose:** This investigation found no bug to fix.

### Recommended Approach: Close as Could Not Reproduce

**Why this approach:**
- JSON serialization works correctly
- The reported symptom ("type: null") was due to querying wrong field name
- No code changes needed

**Trade-offs accepted:**
- If there was a real historical bug, it may have been fixed in a prior beads version
- We can't verify the exact conditions from session ses_474f

**Implementation sequence:**
1. Close this issue as "could-not-reproduce"
2. Document the correct field name for future reference

### Alternative Approaches Considered

**Option B: Add a `type` alias field**
- **Pros:** Would make both `.type` and `.issue_type` work
- **Cons:** Introduces redundancy, could confuse tooling, not needed since docs/examples should use correct field
- **When to use instead:** If many users make this mistake

**Rationale for recommendation:** The JSON output is correct. The issue was user error, not a code defect.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/beads/types.go` - Issue struct with `issue_type` json tag
- `/Users/dylanconlin/Documents/personal/beads/internal/types/types.go` - Canonical beads Issue type
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/skill_inference.go` - IsSpawnableType function
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/daemon.go` - Daemon issue filtering

**Commands Run:**
```bash
# Create test issue
bd create 'test-type-field' --type task

# Check JSON output with wrong field name
bd list --json | jq '.[0].type'
# Result: null

# Check JSON output with correct field name
bd list --json | jq '.[0].issue_type'
# Result: "task"

# Check bd show display vs JSON
bd show orch-go-4v2il
bd show orch-go-4v2il --json
```

**Related Artifacts:**
- **Beads Issue:** orch-go-llbd - The original issue this investigation addresses

---

## Investigation History

**2026-01-05 20:54:** Investigation started
- Initial question: Why does bd show display 'Type: task' but JSON shows "type": null?
- Context: Spawned from beads issue orch-go-llbd

**2026-01-05 20:58:** Found root cause
- JSON field is `issue_type`, not `type`
- Querying `.type` with jq returns null for missing field

**2026-01-05 21:00:** Investigation completed
- Status: Complete
- Key outcome: Not a bug - user error (wrong field name in jq query)
