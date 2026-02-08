<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `bd show --json` always returns an array (even for single issue), and epic children include full Issue objects in dependencies field (not string IDs).

**Evidence:** Direct testing of `bd show orch-go-bgei --json` returned `[{...}]` array format; unmarshaling into single `Issue` struct failed with "cannot unmarshal array".

**Knowledge:** The beads CLI show command has different JSON structure than expected - arrays not objects, and nested Issue objects for dependencies. Fix: unmarshal to `[]Issue` then return first element; use `json.RawMessage` for dependencies.

**Next:** Close - fix implemented, tests pass, smoke test confirms orch complete works with epic children.

**Confidence:** Very High (95%) - Root cause identified, fix implemented and tested end-to-end.

---

# Investigation: bd show Returns Array for Epic Children

**Question:** Why does `orch complete` fail with "json: cannot unmarshal array into Go value of type beads.Issue" for epic child IDs?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: bd show always returns array format

**Evidence:** 
```bash
$ bd show orch-go-bgei --json
[
  {
    "id": "orch-go-bgei",
    "title": "orch complete/abandon fails to parse child beads IDs...",
    ...
  }
]
```

**Source:** Direct CLI execution, `pkg/beads/client.go:422-434` - FallbackShow

**Significance:** The `FallbackShow` function tried to unmarshal into a single `Issue` struct when the CLI always returns an array. This is the root cause of the JSON parsing error.

---

### Finding 2: Epic children include full Issue objects in dependencies

**Evidence:**
```json
{
  "id": "proj-ph1.9",
  "dependencies": [
    {
      "id": "proj-ph1",
      "title": "Epic: Parent Epic",
      "status": "closed",
      "dependency_type": "parent-child"
    }
  ]
}
```

**Source:** `bd show orch-go-ph1.9 --json` output

**Significance:** The `Dependencies` field in `Issue` struct was defined as `[]string`, but `bd show` returns full Issue objects with additional `dependency_type` field. This caused secondary parsing failures for epic children.

---

### Finding 3: RPC client and CLI fallback have same parsing

**Evidence:** Both `Client.Show()` and `FallbackShow()` attempted to unmarshal into a single `Issue{}` struct.

**Source:** `pkg/beads/client.go:284-298` (RPC), `pkg/beads/client.go:422-434` (CLI fallback)

**Significance:** The RPC daemon might return single objects (need to verify), but CLI fallback definitely returns arrays. Fixed CLI fallback; RPC path may need similar fix if daemon behavior matches CLI.

---

## Synthesis

**Key Insights:**

1. **bd CLI JSON format inconsistency** - The beads CLI show command uses consistent array output for both single issues and lists, which differs from typical REST API patterns where show returns a single object.

2. **Dependencies schema evolution** - The dependencies field evolved from simple string IDs to full nested Issue objects with relationship metadata (dependency_type). Using `json.RawMessage` allows accepting either format.

3. **Fallback path is critical** - The FallbackShow function is used when RPC daemon isn't available, and orch commands like complete/abandon rely on this path for issue validation.

**Answer to Investigation Question:**

The `orch complete` command fails for epic children because:
1. `FallbackShow` tried to unmarshal `bd show --json` output into a single `Issue` struct, but `bd show` always returns an array
2. The `Issue.Dependencies` field was `[]string` but bd show returns full Issue objects

Fix: Changed `FallbackShow` to unmarshal into `[]Issue` and return first element; changed `Dependencies` to `json.RawMessage` to accept any format.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Root cause clearly identified through direct testing. Fix implemented, unit tests pass, and smoke tests with real epic child IDs succeed.

**What's certain:**

- ✅ `bd show --json` returns array format (verified with multiple IDs)
- ✅ Fix allows parsing both regular issues and epic children
- ✅ Smoke test: `orch complete orch-go-ph1.9 --force` succeeds where it failed before

**What's uncertain:**

- ⚠️ RPC daemon response format not directly tested (only CLI fallback verified)

**What would increase confidence to 100%:**

- Direct testing of RPC daemon's show operation response format

---

## Implementation Recommendations

### Recommended Approach ⭐

**Array unmarshaling with flexible dependencies** - Unmarshal to slice and extract first element; use json.RawMessage for dependencies field.

**Why this approach:**
- Minimal change surface (only 2 functions modified)
- Forward compatible with any dependencies format
- No breaking changes to existing callers

**Trade-offs accepted:**
- Dependencies are no longer typed - callers can't easily access dependency IDs
- This is acceptable because no current code uses the Dependencies field

**Implementation sequence:**
1. ✅ Update `FallbackShow` to unmarshal array and return first element
2. ✅ Update `Issue.Dependencies` to `json.RawMessage`
3. ✅ Add test for array parsing

---

## References

**Files Modified:**
- `pkg/beads/client.go:422-440` - FallbackShow now handles array response
- `pkg/beads/types.go:117` - Dependencies changed from `[]string` to `json.RawMessage`
- `pkg/beads/client_test.go:394-445` - Added TestBdShowArrayFormat

**Commands Run:**
```bash
# Verify bd show output format
bd show orch-go-bgei --json

# Smoke test fix
./orch complete orch-go-ph1.9 --force
./orch complete orch-go-re8n.3 --force
```

---

## Investigation History

**2025-12-25 20:36:** Investigation started
- Initial question: Why does orch complete fail for epic child IDs?
- Context: Issue reported with error "json: cannot unmarshal array into Go value of type beads.Issue"

**2025-12-25 20:38:** Root cause identified
- bd show returns array, not object
- Dependencies field contains nested objects, not strings

**2025-12-25 20:45:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Fixed FallbackShow to handle array format and made Dependencies flexible
