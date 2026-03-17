---
Status: Complete
Question: Is pkg/daemon/trigger.go a hotspot requiring extraction?
Date: 2026-03-17
---

**TLDR:** trigger.go is 204 lines, created 2 days ago (Mar 14-15) in 2 commits. The "+204 lines/30d" metric is a false positive — the file was created from scratch, not growing. Well below 800-line warning threshold. Already well-decomposed across 10 files in the trigger subsystem. No extraction needed.

## D.E.K.N. Summary

- **Delta:** The "+204 lines/30d" hotspot metric is misleading. The entire file is new — 189 lines on Mar 14 (Phase 1 implementation) + 15 lines on Mar 15 (wiring fix). Zero growth since creation. The 30-day window conflates "recently created" with "actively growing."
- **Evidence:** `git log --numstat --since="30 days ago" -- pkg/daemon/trigger.go` shows exactly 2 commits: f4e57b5 (+189) and 914c06a (+15). Total: 204 lines added, 0 modified. File created from scratch on Mar 14, unchanged since Mar 15. The trigger subsystem is already decomposed across 10 files (trigger.go, trigger_service.go, trigger_test.go, trigger_detectors.go, trigger_detectors_test.go, trigger_detectors_phase2.go, trigger_detectors_phase2_test.go, trigger_expiry.go, trigger_expiry_service.go, trigger_expiry_test.go).
- **Knowledge:** Burst-created files trigger hotspot metrics even when no growth trajectory exists. A 204-line file created 2 days ago with no subsequent changes has zero extraction risk. The trigger subsystem was created with good decomposition from the start — types/orchestrator in trigger.go, service implementation in trigger_service.go, detectors split across phase files.
- **Next:** No action needed. This file is healthy at 204 lines with clean separation of concerns. Re-evaluate only if it exceeds 800 lines.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| 2026-03-17-hotspot-acceleration-pkg-kbgate-publish.md | Same pattern (burst creation false positive) | yes | - |

## Findings

### Finding 1: Growth is burst creation, not sustained

Git history for `pkg/daemon/trigger.go`:

| Commit | Date | Lines Added | Description |
| --- | --- | --- | --- |
| f4e57b5 | Mar 14 | +189 | Phase 1 implementation (types, interfaces, orchestrator) |
| 914c06a | Mar 15 | +15 | TriggerSnapshot + Snapshot() method |
| **Total** | **2 days** | **+204** | **File creation, no subsequent growth** |

### Finding 2: Trigger subsystem already well-decomposed

The trigger layer spans 10 files with clear separation:

| File | Lines | Role |
| --- | --- | --- |
| trigger.go | 204 | Types, interfaces, orchestrator |
| trigger_service.go | ~impl | Default service implementation |
| trigger_test.go | ~tests | Orchestrator tests |
| trigger_detectors.go | ~impl | Phase 1 detectors |
| trigger_detectors_test.go | ~tests | Phase 1 detector tests |
| trigger_detectors_phase2.go | ~impl | Phase 2 detectors |
| trigger_detectors_phase2_test.go | ~tests | Phase 2 detector tests |
| trigger_expiry.go | ~impl | Expiry logic |
| trigger_expiry_service.go | ~impl | Expiry service |
| trigger_expiry_test.go | ~tests | Expiry tests |

### Finding 3: Code structure is clean

trigger.go contains:
- Types: `TriggerSuggestion`, `TriggerBudget`, `TriggerSnapshot`, `TriggerScanResult` (pure data)
- Interfaces: `PatternDetector`, `TriggerScanService` (clean contracts)
- One method: `RunPeriodicTriggerScan` (orchestrator logic)

No coupling issues, no code smells, no extraction candidates.

## Test Performed

Verified via `git log --numstat --since="30 days ago"` — confirmed all 204 lines were added in 2 commits within a 2-day window, with zero subsequent changes. Cross-referenced with prior investigation (pkg-kbgate-publish) which found the same false positive pattern.

## Conclusion

**False positive.** The hotspot acceleration metric flags any file with high absolute line additions in 30 days. For newly created files, this is noise — the file went from 0→204, not from 500→704. The trigger subsystem was designed with proper decomposition from the start (10 files). No risk of accretion.
