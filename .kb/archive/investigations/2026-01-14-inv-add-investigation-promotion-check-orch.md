<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added investigation-promotion gate to `orch session end` that warns when >5 candidates need triage.

**Evidence:** Tested with 37 candidates - gate fires, shows warning, prompts user, aborts on "n", proceeds on "y".

**Knowledge:** Gate pattern follows existing handoff section validation - check early, warn, prompt, abort or proceed.

**Next:** close - Feature complete and tested.

**Promote to Decision:** recommend-no - Tactical feature, not architectural decision.

---

# Investigation: Add Investigation Promotion Check to Orch Session End

**Question:** How to add investigation-promotion check that gates session end when backlog accumulates?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb reflect command output format

**Evidence:** `kb reflect --type investigation-promotion --format json` returns:
```json
{
  "investigation_promotion": [
    {"file": "...", "title": "...", "age_days": 7, "suggestion": "..."},
    ...
  ]
}
```

**Source:** Ran `kb reflect --type investigation-promotion --format json | head -100`

**Significance:** Provides structured data for counting candidates and parsing in Go.

---

### Finding 2: Existing gate pattern in session.go

**Evidence:** `completeAndArchiveHandoff()` validates handoff sections and prompts for unfilled ones before proceeding. Similar pattern: check → warn → prompt → abort or continue.

**Source:** `cmd/orch/session.go:522-566`

**Significance:** Established pattern to follow for investigation promotion gate.

---

### Finding 3: Threshold of 5 is reasonable

**Evidence:** Current count is 37 candidates. A threshold of 5 is low enough to catch accumulation early but high enough to not trigger on every session end.

**Source:** Task specification

**Significance:** Prevents backlog accumulation while not being overly restrictive.

---

## Synthesis

**Key Insights:**

1. **Gate pattern is reusable** - The check → warn → prompt → abort/proceed pattern from handoff validation works well for other session-end gates.

2. **JSON output enables clean parsing** - kb reflect's JSON format makes it easy to count candidates without complex text parsing.

3. **Early gate placement matters** - Placing the gate before handoff work means users see the warning immediately, not after spending time on handoff prompts.

**Answer to Investigation Question:**

Added `gateInvestigationPromotions()` function that:
1. Runs `kb reflect --type investigation-promotion --format json`
2. Parses the JSON to count candidates
3. If count > 5, shows warning with count and threshold
4. Prompts user to continue (y) or abort (N)
5. Returns error if aborted, nil if proceeding

Integrated into `runSessionEnd()` right after the active session check.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gate triggers when count > threshold (verified: 37 > 5 triggers warning)
- ✅ User can abort with "n" (verified: returns error, session not ended)
- ✅ User can proceed with "y" (verified: continues to handoff validation)
- ✅ Build compiles without errors (verified: `go build ./cmd/orch/...`)
- ✅ All tests pass (verified: `go test ./cmd/orch/...`)

**What's untested:**

- ⚠️ Behavior when kb command is not available (returns 0, proceeds silently)
- ⚠️ Behavior with exactly 5 candidates (should not trigger - uses `<=` comparison)

**What would change this:**

- Finding would be wrong if kb reflect output format changes
- Implementation would need adjustment if threshold should be configurable

---

## Implementation Recommendations

**Purpose:** Document what was implemented.

### Implemented Approach ⭐

**Add gate function with user prompt** - Follows existing handoff validation pattern.

**Why this approach:**
- Consistent with existing session.go patterns
- Non-blocking for users who explicitly acknowledge
- Clear messaging about what to do (run kb reflect)

**Trade-offs accepted:**
- Always prompts when above threshold (no auto-dismiss)
- Shells out to kb command (external dependency)

**Implementation sequence:**
1. Added constant `InvestigationPromotionThreshold = 5`
2. Added structs for JSON parsing
3. Added `checkInvestigationPromotions()` to run command and parse
4. Added `gateInvestigationPromotions()` to warn and prompt
5. Integrated gate into `runSessionEnd()` after active session check

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Main implementation file

**Commands Run:**
```bash
# Check kb reflect output format
kb reflect --type investigation-promotion --format json | head -100

# Count candidates
kb reflect --type investigation-promotion --format json | jq '.investigation_promotion | length'
# Result: 37

# Build and test
go build ./cmd/orch/...
go test ./cmd/orch/...
make install

# Test gate behavior
echo "n" | orch session end  # Aborts
echo "y" | orch session end  # Proceeds
```

---

## Investigation History

**2026-01-14 15:00:** Investigation started
- Initial question: How to add investigation-promotion check to session end
- Context: Task to prevent accumulation of promotion candidates

**2026-01-14 15:08:** Implementation complete
- Added gate function, integrated into runSessionEnd()
- Verified with 37 candidates - gate fires and works correctly

**2026-01-14 15:10:** Investigation completed
- Status: Complete
- Key outcome: Gate implemented, tested, and working
