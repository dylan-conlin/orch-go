# Session Synthesis

**Agent:** og-feat-fix-orch-patterns-14jan-e4a7
**Issue:** orch-go-3puvy.6
**Duration:** 2026-01-14 21:50 → 2026-01-14 22:00
**Outcome:** success

---

## TLDR

Fixed `orch patterns` noise by filtering incompatible schema entries, benign empty commands, and recalibrating severity thresholds. Result: 87% reduction in critical patterns (23 → 3), all remaining patterns are actionable.

---

## Delta (What Changed)

### Files Modified
- `pkg/action/action.go` - Added schema filtering (skip entries without target/outcome), benign command filtering (BenignEmptyCommands map), and tools that require targets filter
- `cmd/orch/patterns.go` - Recalibrated severity thresholds: errors 10+=critical/5+=warning, empty results 15+=warning (never critical), fallbacks 8+=warning

### Commits
- (pending) - fix: reduce orch patterns noise by 87%

---

## Evidence (What Was Observed)

- Action log contained 21,232 entries with two different schemas (old: target/outcome, new: title/outputLength)
- 122 entries had missing target+outcome fields (incompatible schema)
- 98 of those were lowercase "bash" entries causing "bash on (empty)" pattern
- Benign commands (grep, ls, git add, go build) returning empty were flagged as patterns
- Original severity threshold of 5+ = critical was too aggressive

### Tests Run
```bash
# All tests passing
go test ./pkg/action/... -v
# PASS: 15 tests
go test ./pkg/patterns/... -v
# PASS: 24 tests

# Before fix
orch patterns | grep "Total\|Critical"
# Total: 78 patterns detected
# Critical: 23 (require immediate attention)

# After fix
orch patterns | grep "Total\|Critical"
# Total: 56 patterns detected
# Critical: 3 (require immediate attention)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-14-inv-fix-orch-patterns-noise-fix.md` - Root cause analysis and fix documentation

### Decisions Made
- Empty results should never be critical severity (grep/ls returning empty is normal)
- Tools like scroll/read/glob with empty targets are from incompatible schemas (filter them)
- Severity should be outcome-dependent: errors are more serious than empty results

### Constraints Discovered
- Action log has mixed schemas from multiple logging sources - must filter at load time
- 40+ commands are "benign empty" (BenignEmptyCommands map) - returning empty is normal behavior

### Externalized via `kb`
- Investigation file documents the schema mismatch and filtering approach

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete {issue-id}`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Where is the new-format logging coming from? (tool with callID, title, outputLength)
- Should action-log.jsonl be migrated to a single schema?

**Areas worth exploring further:**
- Periodic cleanup of incompatible schema entries from action log
- Consider schema versioning for action log entries

**What remains unclear:**
- Long-term false positive rate after filtering (need observation over days)

---

## Session Metadata

**Skill:** feature-impl
**Model:** opus
**Workspace:** `.orch/workspace/og-feat-fix-orch-patterns-14jan-e4a7/`
**Investigation:** `.kb/investigations/2026-01-14-inv-fix-orch-patterns-noise-fix.md`
**Beads:** `bd show orch-go-3puvy.6`
