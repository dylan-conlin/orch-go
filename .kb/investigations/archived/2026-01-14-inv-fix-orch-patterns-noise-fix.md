## Summary (D.E.K.N.)

**Delta:** `orch patterns` noise caused by three issues: incompatible schema in action log (98 "bash on empty"), benign commands flagged (grep/ls returning empty is normal), and miscalibrated severity (5+ occurrences = critical was too aggressive).

**Evidence:** Before: 78 patterns, 23 critical. After: 56 patterns, 3 critical. 87% reduction in critical noise.

**Knowledge:** Action log has mixed schemas from different logging sources - must filter entries without expected fields. Many bash commands returning empty is normal behavior, not a pattern worth surfacing.

**Next:** Commit and deploy the fix.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Fix Orch Patterns Noise

**Question:** Why does `orch patterns` show 78 patterns with 23 critical including "bash on (empty)" 98 times?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Agent og-feat-fix-orch-patterns-14jan-e4a7
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Action log contains incompatible schema entries

**Evidence:** The action-log.jsonl contains two different formats:
- Old format: `{timestamp, tool, target, outcome, session_id}`
- New format: `{timestamp, sessionID, tool, callID, title, outputLength, directory, isWorker}`

When the Go code unmarshals new format entries into the old struct, `target` and `outcome` become empty strings. This results in 98 "bash on (empty)" patterns from lowercase "bash" entries.

**Source:** `~/.orch/action-log.jsonl`, `pkg/action/action.go:273-315`

**Significance:** The pattern detection was counting entries from incompatible schemas as real patterns. Filtering these at load time eliminates 122 spurious entries.

---

### Finding 2: Benign commands were flagged as patterns

**Evidence:** Commands like `grep`, `ls`, `git add`, `go build` returning empty is normal behavior:
- `grep` → empty means no matches (normal)
- `ls` → empty means empty directory (normal)
- `git add` → empty means success (normal)
- `go build` → empty means compilation succeeded (normal)

These were being flagged as "empty result" patterns when they're expected outcomes.

**Source:** `orch patterns` output showing "Empty result: Bash on grep" 15 times, etc.

**Significance:** Filtering known-benign commands from pattern detection eliminates false positives and surfaces only actionable signals.

---

### Finding 3: Severity thresholds were miscalibrated

**Evidence:** Original thresholds:
- 5+ occurrences = critical
- 3+ occurrences = warning

This meant any bash command that failed 5 times became critical, even if failures were expected.

New thresholds:
- Errors: 10+ = critical, 5+ = warning
- Empty results: 15+ = warning (never critical - empty is often normal)
- Fallbacks: 8+ = warning

**Source:** `cmd/orch/patterns.go:468-498`

**Significance:** Recalibrating severity ensures critical patterns are actually critical issues requiring immediate attention.

---

## Synthesis

**Key Insights:**

1. **Schema hygiene matters** - Mixed logging formats in a single file create noise. Filter incompatible entries at load time.

2. **Context determines significance** - Empty output from `grep` vs empty output from `curl` have different meanings. Benign patterns need explicit filtering.

3. **Severity should reflect action needed** - Critical should mean "requires immediate attention", not just "happens often".

**Answer to Investigation Question:**

The 78 patterns with 23 critical was caused by:
1. 98 entries from incompatible schema (lowercase "bash" with missing fields) → Fixed by filtering entries where both target AND outcome are empty
2. Benign commands being counted (grep/ls/git empty returns) → Fixed by adding BenignEmptyCommands filter
3. Low severity thresholds (5+ = critical) → Fixed by raising thresholds and making severity outcome-dependent

After fixes: 56 patterns, 3 critical (87% reduction in critical noise).

---

## Structured Uncertainty

**What's tested:**

- ✅ Filtering removes 122 incompatible schema entries (verified: debug script counting skipped vs kept)
- ✅ Benign commands like grep/ls/git are no longer flagged (verified: patterns output)
- ✅ Noise reduced by 87% for critical patterns (verified: before/after comparison)
- ✅ All existing tests pass (verified: go test ./pkg/action/... and ./pkg/patterns/...)

**What's untested:**

- ⚠️ Long-term false positive rate (need to observe over multiple days)
- ⚠️ Edge cases where benign commands should be flagged (timeout scenarios)

**What would change this:**

- Finding would be wrong if legitimate patterns are now being hidden
- Should revisit if actionable patterns are missed after deployment

---

## Implementation Summary

**Changes made:**

1. `pkg/action/action.go:302-317`: Skip entries with incompatible schema (empty target AND outcome, or empty target for tools that require targets)

2. `pkg/action/action.go:324-407`: Added `BenignEmptyCommands` map and `isBenignEmptyPattern()` function to filter known-benign empty outcomes

3. `cmd/orch/patterns.go:468-498`: Recalibrated severity thresholds - errors get critical at 10+, empty results never critical, fallbacks at 8+

---

## References

**Files Examined:**
- `pkg/action/action.go` - Action event loading and pattern detection
- `cmd/orch/patterns.go` - Pattern collection and display
- `~/.orch/action-log.jsonl` - Action log data

**Commands Run:**
```bash
# Debug script to test filtering
go run /tmp/test_full_patterns.go

# Verify noise reduction
./build/orch patterns 2>&1 | head -60
```

---

## Investigation History

**2026-01-14 21:50:** Investigation started
- Initial question: Why is orch patterns showing 78 patterns with 23 critical including "bash on (empty)" 98 times?

**2026-01-14 22:00:** Found incompatible schema issue
- Discovered action log has mixed formats from different logging sources

**2026-01-14 22:10:** Implemented filtering
- Added schema filtering and benign command filtering

**2026-01-14 22:15:** Recalibrated severity
- Adjusted thresholds based on outcome type

**2026-01-14 22:20:** Investigation completed
- Status: Complete
- Key outcome: 87% reduction in critical pattern noise (23 → 3 critical)
