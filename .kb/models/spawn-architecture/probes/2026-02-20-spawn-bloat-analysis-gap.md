# Probe: Spawn Bloat Analysis Gap in RunHotspotCheckForSpawn

**Model:** spawn-architecture
**Date:** 2026-02-20
**Status:** Complete

---

## Question

Does `RunHotspotCheckForSpawn` include bloat-size analysis when checking if a task targets a CRITICAL file (>1500 lines)? The spawn architecture claims hotspot detection blocks feature-impl/systematic-debugging skills on CRITICAL files, but is the bloat analysis actually performed at spawn time?

---

## What I Tested

1. Created failing test that validates `RunHotspotCheckForSpawn` returns bloat-size hotspots:

```bash
go test -v -run TestRunHotspotCheckForSpawn_IncludesBloatAnalysis ./cmd/orch/
```

2. Examined `RunHotspotCheckForSpawn` function in `cmd/orch/hotspot.go`:

```go
// Before fix - function called these:
analyzeFixCommits(projectDir, 28, 5)
analyzeInvestigationClusters(projectDir, 3)
analyzeCouplingClusters(projectDir, 28)
// BUT NOT:
// analyzeBloatFiles(projectDir, 800)  // <-- MISSING!
```

3. Compared with `runHotspot()` (the CLI command) which DOES call `analyzeBloatFiles`:

```go
// In runHotspot() at line 136:
bloatHotspots, totalBloat, err := analyzeBloatFiles(projectDir, hotspotBloatThreshold)
```

---

## What I Observed

**Before fix:**
- Test failed: `Expected non-nil result - bloat-size hotspot should be detected`
- `RunHotspotCheckForSpawn` did not call `analyzeBloatFiles`
- Bloat-size hotspots (files >800 lines, CRITICAL >1500 lines) were never included in spawn-time checks
- Unit tests for `checkSpawnHotspots` passed because they injected mock bloat hotspots directly

**After fix:**
- Added `analyzeBloatFiles(projectDir, 800)` call to `RunHotspotCheckForSpawn`
- Test passes: bloat-size hotspots now detected when task references large files
- All 32 hotspot-related tests pass

---

## Model Impact

- [x] **Extends** model with: Missing implementation detail. The spawn architecture model documents that hotspot checks run at spawn time, but doesn't specify which hotspot types are included. The `RunHotspotCheckForSpawn` function was incomplete — it included fix-density, investigation-cluster, and coupling-cluster analyses but omitted bloat-size analysis. This meant the "CRITICAL file blocking" feature (added in orch-go-1128) couldn't actually detect files by name in task descriptions.

**Recommended model update:**
Add to Critical Invariants section:
> 7. **RunHotspotCheckForSpawn must include all hotspot types**
>    - fix-density (git history)
>    - investigation-cluster (kb reflect)
>    - coupling-cluster (commit co-occurrence)
>    - bloat-size (file line counts)
>    - Violation: Spawn gates fail to detect CRITICAL files by name

---

## Notes

The bug was subtle because:
1. Unit tests tested `checkSpawnHotspots` with mock hotspot data (which included bloat types)
2. Integration tests tested blocking logic directly (which worked)
3. No test verified that `RunHotspotCheckForSpawn` actually populated bloat-size hotspots

This is a test coverage gap pattern: function composition wasn't tested, only component behavior.

Fix: Added `analyzeBloatFiles(projectDir, 800)` call to `RunHotspotCheckForSpawn` in `cmd/orch/hotspot.go:831`.
