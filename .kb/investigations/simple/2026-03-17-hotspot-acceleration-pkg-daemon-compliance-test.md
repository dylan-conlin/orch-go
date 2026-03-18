---
title: "Hotspot Acceleration: pkg/daemon/compliance_test.go"
status: Complete
created: 2026-03-17
beads_id: orch-go-jv0zl
---

## TLDR

`compliance_test.go` (+226 lines/30d) is a **false positive** — the file was born via extraction refactor on 2026-03-13, not via organic growth. At 226 lines with no further growth since creation, it is well below the 1,500-line accretion boundary and requires no action.

## D.E.K.N.

- **Delta:** Hotspot acceleration alert for compliance_test.go is a false positive caused by file birth via extraction
- **Evidence:** Git history shows 2 commits on 2026-03-13 — extraction (c9ea4a3c4) + feature wiring (aebe1d80e); no commits since
- **Knowledge:** Hotspot acceleration detector cannot distinguish "file born via extraction" from "file growing via accretion" — this is a known class of false positive
- **Next:** No extraction needed. Consider adding extraction-birth detection to hotspot acceleration to suppress this class of false positive.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

Is `pkg/daemon/compliance_test.go` (+226 lines/30d) a genuine hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: File was born via extraction, not organic growth

Git history shows exactly 2 commits:

1. `c9ea4a3c4` (2026-03-13) — "refactor: extract compliance/coordination from entangled daemon methods" — created the file as part of breaking `daemon_test.go` (1308 lines) into focused test files
2. `aebe1d80e` (2026-03-13) — "feat: wire ComplianceConfig into daemon compliance gates" — added 71 lines of compliance-level-aware tests

No further commits since creation. The +226 lines/30d metric reflects file birth, not growth.

### Finding 2: File structure is well-organized at current size

Three clear test groups:
- `TestCheckPreSpawnGates_*` (lines 9-68): 5 tests for pre-spawn gate signals
- `TestCheckIssueCompliance_*` (lines 70-157): 9 tests for per-issue compliance filters
- `TestNewWithConfig_Compliance*` (lines 159-226): 4 tests for compliance-level-aware configuration

### Finding 3: Companion files are also well-sized

- `compliance.go`: 250 lines
- `coordination.go`: 298 lines

The compliance/coordination extraction was a successful decomposition from the 1,308-line `daemon_test.go`.

## Test Performed

```bash
git log --format="%h %ad %s" --date=short --follow pkg/daemon/compliance_test.go
# Result: 2 commits, both 2026-03-13, no growth since
```

## Conclusion

**False positive.** The hotspot acceleration detector flagged compliance_test.go because it grew by 226 lines in 30 days, but this growth is entirely from file creation via a deliberate extraction refactor — the exact opposite of accretion. The file is well-organized at 226 lines, well below the 1,500-line threshold, and has seen no growth since birth.

**Root cause of false positive:** The hotspot acceleration detector treats "file born via extraction" the same as "file growing organically." Files born via extraction inherently show high delta (0 → N lines) in their first 30-day window.
