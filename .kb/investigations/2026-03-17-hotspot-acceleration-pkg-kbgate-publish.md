---
Status: Complete
Question: Is pkg/kbgate/publish_test.go a hotspot requiring extraction?
Date: 2026-03-17
---

**TLDR:** publish_test.go is 708 lines but NOT growing — all 710 lines were added in a single day (Mar 10) across 3 feature commits, with no growth since. Below 800-line warning threshold. No extraction needed now.

## D.E.K.N. Summary

- **Delta:** The "+710 lines/30d" metric is misleading. All growth occurred on March 10 in a single development session (3 feature commits: initial publish gate, claim-upgrade detector, ledger validation). The file has been stable for 7 days.
- **Evidence:** `git log --numstat --since="30 days ago"` shows 710 net additions, but 708 lines came from Mar 10 commits (428 + 77 + 203). Only 2 trivial changes since (rename "physics" → "accretion"). No active features in pipeline for this gate.
- **Knowledge:** Burst-created test files trigger hotspot metrics even when growth has stabilized. The 30-day window conflates "created recently" with "growing rapidly." A file that was created 7 days ago and hasn't changed since is not the same risk as a file growing 25 lines/day.
- **Next:** No action needed. Re-evaluate if file exceeds 800 lines (i.e., when new gates are added to CheckPublish).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| N/A — novel investigation | - | - | - |

## Findings

### Finding 1: Growth is burst, not sustained

Git history for `pkg/kbgate/publish_test.go`:

| Commit | Date | Lines Added | Description |
| --- | --- | --- | --- |
| 6f5497d | Mar 10 15:27 | +428 | Initial publish gate (4 checks) |
| 372a28b | Mar 10 16:39 | +77 | Claim-upgrade boundary detector |
| fc39ba1 | Mar 10 17:04 | +1/-1 | Scope fix |
| 9d5b642 | Mar 10 18:07 | +203 | Claim ledger validation |
| fa4584a | Mar 12 17:54 | +1/-1 | Rename only |

All substantive growth happened in a 3-hour window on March 10. Zero growth since.

### Finding 2: Test-to-code ratio is normal

- Production code (`publish.go`): 461 lines
- Test code (`publish_test.go`): 708 lines
- Ratio: 1.54:1

This is standard for filesystem-heavy gate tests that require temp directory setup, artifact creation, and multi-case validation.

### Finding 3: Boilerplate is significant but not actionable yet

The file contains:
- 13 `t.TempDir()` calls
- 22 `os.MkdirAll` calls
- 38 `os.WriteFile` calls

A `publishTestEnv` fixture builder could reduce each test by 5-15 lines (~100-150 lines total). However, at 708 lines this optimization is premature — the file is well below the 800-line warning threshold.

### Finding 4: Package is well-structured overall

The `kbgate` package totals 3,506 lines across 8 files. File sizes range from 314-708 lines. No single file exceeds the warning threshold. The package has clear separation: `challenge.go`, `claims.go`, `model.go`, `publish.go` each handle distinct gate types with matching `_test.go` files.

## Test performed

```bash
# Verified all tests pass
go test ./pkg/kbgate/... -count=1 -v
# Result: PASS (0.402s) — all 20+ test functions passing

# Verified git history shows burst creation, not sustained growth
git log --numstat --since="30 days ago" -- pkg/kbgate/publish_test.go
# Result: 710 additions / 2 deletions, all from Mar 10-12
```

## Conclusion

**No extraction needed.** The file was created in a burst on March 10 and has been stable since. The 30-day hotspot metric conflated "recently created" with "actively growing." At 708 lines, it's below the 800-line warning threshold and far below the 1500-line critical threshold.

**If future extraction is needed** (e.g., new gates push past 800 lines):
1. **Priority 1:** Extract `publishTestEnv` fixture builder to reduce dir-setup boilerplate (~100-150 line reduction)
2. **Priority 2:** Split `publish_ledger_test.go` (ledger tests, ~185 lines) if file exceeds 1000 lines
