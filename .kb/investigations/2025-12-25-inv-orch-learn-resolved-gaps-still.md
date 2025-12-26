## Summary (D.E.K.N.)

**Delta:** Resolved gaps still appeared in `orch learn` suggestions due to two bugs: `RecordResolution` only marked ONE event, and `FindRecurringGaps` counted ALL events including resolved ones.

**Evidence:** Code inspection showed `RecordResolution` breaks after first match (line 259), and `FindRecurringGaps` has no filter for Resolution field. Tests confirm fix works.

**Knowledge:** When resolving a gap pattern, ALL matching events must be marked resolved; otherwise subsequent FindRecurringGaps calls still count unresolved events toward recurrence threshold.

**Next:** Close - fix implemented and tested.

**Confidence:** Very High (98%) - tested behavior change with unit tests.

---

# Investigation: Orch Learn Resolved Gaps Still Appear

**Question:** Why do resolved gaps still appear in `orch learn` suggestions?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (98%)

---

## Findings

### Finding 1: RecordResolution only marks the most recent event

**Evidence:** In `pkg/spawn/learning.go:253-261`, the `RecordResolution` function uses `break` after finding the first matching unresolved event, leaving older events with the same query unmarked.

```go
for i := len(t.Events) - 1; i >= 0; i-- {
    if t.Events[i].Query == query && t.Events[i].Resolution == "" {
        t.Events[i].Resolution = resolution
        t.Events[i].ResolutionDetails = details
        break  // BUG: Only marks ONE event
    }
}
```

**Source:** `pkg/spawn/learning.go:253-261`

**Significance:** If a gap occurred 5 times and you resolve it, only 1 event gets marked. The other 4 remain unresolved.

---

### Finding 2: FindRecurringGaps counts ALL events regardless of resolution status

**Evidence:** In `pkg/spawn/learning.go:292-298`, the function groups events by query without filtering out resolved events:

```go
for _, e := range t.Events {
    normalized := normalizeQuery(e.Query)
    queryGroups[normalized] = append(queryGroups[normalized], e)  // No Resolution check
}
```

**Source:** `pkg/spawn/learning.go:292-298`

**Significance:** Even if some events are resolved, they still count toward the recurrence threshold (3+), causing resolved gaps to keep appearing in suggestions.

---

## Synthesis

**Key Insights:**

1. **Combined bug effect** - The two bugs compound: RecordResolution marks only 1 event, then FindRecurringGaps counts all 5 events. Result: resolved gap with 5 occurrences still shows as "4+ recurring" because 4 events remain unresolved.

2. **Fix requires both changes** - Must mark ALL matching events as resolved AND filter out resolved events from suggestions.

**Answer to Investigation Question:**

Resolved gaps still appeared because `RecordResolution` only marked one event (the most recent), and `FindRecurringGaps` counted all events without filtering resolved ones. Fixed by: (1) updating RecordResolution to mark ALL matching events, and (2) adding a Resolution filter in FindRecurringGaps.

---

## References

**Files Modified:**
- `pkg/spawn/learning.go:252-262` - Fixed RecordResolution to mark all matching events
- `pkg/spawn/learning.go:291-307` - Fixed FindRecurringGaps to exclude resolved events
- `pkg/spawn/learning_test.go` - Updated existing test, added 3 new tests

**Tests Added:**
- `TestGapTrackerFindRecurringGapsExcludesResolved` - Verifies resolved events are excluded
- `TestGapTrackerRecordResolutionRemovesFromSuggestions` - Verifies full workflow
