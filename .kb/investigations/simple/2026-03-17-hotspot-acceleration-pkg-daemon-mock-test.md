---
title: "Hotspot acceleration: pkg/daemon/mock_test.go"
status: Complete
date: 2026-03-17
---

## TLDR

mock_test.go hotspot is a false positive — the file was born 2026-02-28 (208 lines) from a healthy interface refactoring, and its "+288 lines/30d" metric is just its birth size. At 282 lines serving 27 test files, it's well-structured shared test infrastructure with no extraction needed.

## D.E.K.N. Summary

- **Delta:** mock_test.go growth is entirely from file creation, not accretion. No action needed.
- **Evidence:** git log shows 208-line birth on Feb 28, then 4 small additions (+1, +28, +11, +37, +3) as new interfaces were added. Used by 27 test files (315 references).
- **Knowledge:** Hotspot detector flags new files whose total size equals their 30-day growth. Birth-growth should be filtered from hotspot reports.
- **Next:** No extraction needed. Consider adding birth-detection to hotspot tooling to suppress false positives.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A — novel investigation | - | - | - |

## Question

Is pkg/daemon/mock_test.go a genuine accretion hotspot requiring extraction, or a false positive from recent file creation?

## Findings

### Finding 1: File was born from interface refactoring

Commit `1f035752` (2026-02-28) created mock_test.go with 208 lines as part of "refactor: convert Daemon function fields to Go interfaces". This was a **healthy architectural change** — replacing function fields with testable interfaces.

### Finding 2: Growth is proportional to new interfaces

Post-birth commits:
- `212cbdb5e` (Feb 28): +1/-1 — rename
- `f430e182a` (Feb 28): +28 — mockKnowledgeHealthService + mockAgreementCheckService
- `e8ef8d20f` (Mar 3): +11 — HasExistingSessionOrError method
- `541e4e8d1` (Mar 11): +37 — mockArtifactSyncService
- `a55970e28` (Mar 12): +3/-3 — field rename

Each addition corresponds to a new daemon interface — this is organic, proportional growth.

### Finding 3: File serves as shared test infrastructure

11 mock structs used across **27 test files** with **315 total references**. Centralizing mocks in one file is standard Go practice and prevents duplication across the 70+ test files in pkg/daemon (25,000 total test lines).

### Finding 4: Size is far below any threshold

- Current: 282 lines
- Advisory threshold: 800 lines
- Critical threshold: 1,500 lines
- At ~6 lines per new interface, reaching 800 lines would require ~86 more interfaces

## Test Performed

```bash
# Verified file history
git log --format="%h %ad" --date=short --numstat -- pkg/daemon/mock_test.go

# Verified usage breadth
grep -rl 'mock(IssueQuerier|Spawner|...)' pkg/daemon/*_test.go | wc -l
# Result: 27 files, 315 references

# Verified current size
wc -l pkg/daemon/mock_test.go
# Result: 282 lines
```

## Conclusion

**False positive.** The "+288 lines/30d" metric equals the file's birth size — it was created 17 days ago as part of a healthy interface refactoring. At 282 lines serving 27 test files, this is well-structured shared test infrastructure. No extraction needed.
