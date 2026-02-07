# Session Synthesis

**Agent:** og-arch-recurring-problem-duplicate-07jan-be76
**Issue:** orch-go-xqude
**Duration:** 2026-01-07 07:20 → 2026-01-07 08:15
**Outcome:** success

---

## TLDR

Investigated why duplicate synthesis issues are created hourly by kb reflect. Root cause: `synthesisIssueExists()` returns `false` (allow creation) when JSON parsing fails, enabling duplicates even when open issues exist. Recommended fix: fail-closed (assume duplicate on any error).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Full investigation with D.E.K.N. summary

### Files Modified
- `.orch/features.json` - Added feat-039 for the kb-cli fix

### Commits
- (uncommitted) - Investigation and feature list update

---

## Evidence (What Was Observed)

- 14 duplicate issues for "model" topic, created hourly matching daemon reflection interval
- `synthesisIssueExists()` at kb-cli/cmd/kb/reflect.go:498-508 returns `false, nil` on any error
- When JSON parse fails, dedup check silently allows creation
- Open issues DID exist when duplicates were created (traced orch-go-yyw7d timeline)
- bd list command works correctly and returns valid JSON when piped directly
- Shell variable assignment occasionally causes JSON parse issues (intermittent)

### Tests Run
```bash
# Verified bd list returns matching issues
bd list --all --title-contains "Synthesize model investigations" --json | jq 'length'
# Result: 14 (duplicates confirmed)

# Verified direct pipe works
bd list --all --title-contains "Synthesize dashboard investigations" --json | jq 'length'
# Result: 15 (works correctly)

# Traced issue timeline
bd list --all --title-contains "Synthesize dashboard investigations" --json | jq -r '.[] | "\(.id) \(.created_at) \(.status)"'
# Showed orch-go-yyw7d was open when orch-go-8qg67 was created
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Full root cause analysis

### Decisions Made
- **Fail-closed for idempotency:** For dedup checks, "assume duplicate exists on error" is safer than "assume no duplicate". False positive (skip creation when no issue exists) has low cost; false negative (create duplicate) has high cost.

### Constraints Discovered
- Error handling that silently allows creation is wrong for idempotency checks
- Daemon runs hourly, amplifying any dedup failure by ~1 duplicate/hour
- JSON parsing can fail intermittently (reason not fully diagnosed but fix handles it)

### Externalized via `kn`
- (None yet - pattern to externalize: "fail-closed for idempotency checks")

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, fix requires kb-cli changes)

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Feature list updated (feat-039)
- [ ] Ready for `orch complete orch-go-xqude`

### Follow-up Work

**Issue:** feat-039 in features.json
**Skill:** feature-impl
**Repo:** kb-cli
**Context:**
```
Change synthesisIssueExists() and openIssueExists() in reflect.go to return true on any error 
instead of false. Add warning logging. Lines 498-508 and 1273-1283.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does JSON parsing fail intermittently when stored in shell variable? (Suspected buffering, but not confirmed)
- Should daemon restart when kb binary is rebuilt? (Currently doesn't, relies on launchd restart)

**Areas worth exploring further:**
- Adding integration test for dedup logic
- Whether bd output can contain invalid JSON in edge cases

**What remains unclear:**
- Exact trigger for JSON parse failure (works most times, fails sometimes)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-recurring-problem-duplicate-07jan-be76/`
**Investigation:** `.kb/investigations/2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md`
**Beads:** `bd show orch-go-xqude`
