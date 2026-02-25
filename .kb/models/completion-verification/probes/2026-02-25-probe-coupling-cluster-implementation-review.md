# Probe: Coupling-Cluster Implementation Does Not Impact CRITICAL Files

**Date:** 2026-02-25
**Status:** Complete
**Model:** completion-verification
**Issue:** orch-go-1110

## Question

Does the coupling-cluster implementation (issue 1110) stay within safe accretion boundaries? Specifically: does it create a new file (`hotspot_coupling.go`) and only make minor integration changes to `hotspot.go` (under 1500 lines), without requiring extraction of any CRITICAL files?

## What I Tested

1. Checked line counts of all hotspot-related files
2. Verified `hotspot_coupling.go` exists as a standalone new file
3. Checked the integration surface in `hotspot.go` (grep for coupling references)
4. Ran `orch hotspot` to confirm coupling clusters appear in output
5. Checked all CRITICAL (>1500 line) files in the project against the implementation's touch surface

## What I Observed

### Finding 1: Implementation is contained in a new file (390 lines)

`cmd/orch/hotspot_coupling.go` is 389 lines. It contains all coupling-specific logic: `analyzeCouplingClusters()`, `parseGitLogCommits()`, `classifyLayer()`, `extractConcept()`, `buildCouplingClusters()`, `scoreCouplingCluster()`, etc. This matches the design spec's estimate of ~200 lines — actual is larger due to thorough healthy-coupling filtering, but well within bounds.

Test file: `hotspot_coupling_test.go` at 315 lines.

### Finding 2: hotspot.go integration is minimal (858 lines total, well under 1500)

`hotspot.go` is 858 lines. The coupling integration touches ~20 lines across 5 locations:
- Line 98: `TotalCouplingClusters` field in `HotspotReport` struct
- Lines 144-150: Call to `analyzeCouplingClusters()` in `runHotspot()`
- Line 517: Display coupling cluster count in summary
- Lines 541, 704-705, 749-750, 795: Icon/matching logic in output formatters
- Lines 838-840: Coupling call in `RunHotspotCheckForSpawn()`

This is textbook integration — new behavior in new file, minimal wiring in existing file.

### Finding 3: No CRITICAL files are touched

CRITICAL files (>1500 lines) in the project:
- `cmd/orch/complete_cmd.go` (2154) — NOT touched
- `plugins/coaching.ts` (1743) — NOT touched
- `cmd/orch/doctor.go` (1736) — NOT touched
- `.opencode/plugin/coaching.ts` (1570) — NOT touched
- `pkg/spawn/context.go` (1504) — NOT touched
- `web/.svelte-kit/` build artifacts — NOT touched

The coupling-cluster implementation touches zero CRITICAL files.

### Finding 4: Output is working correctly

`orch hotspot` output includes `Coupling Clusters: 15`, confirming the analysis runs and integrates with the existing hotspot report. Coupling clusters appear as a 4th hotspot type alongside fix-density, investigation-cluster, and bloat-size.

## Model Impact

**Confirms** the completion-verification model's accretion enforcement architecture:

1. **Accretion boundary respected:** The implementation followed the design's explicit constraint ("existing hotspot.go is 806 lines — new analysis belongs in separate file"). The new file pattern is exactly how accretion enforcement should work — new behavior in new files, minimal integration in existing files.

2. **No extraction needed:** `hotspot.go` at 858 lines is well under the 1500-line CRITICAL threshold. No files in the implementation's touch surface require extraction.

3. **Spawn gate integration is free:** Because coupling clusters use the existing `Hotspot` type, `RunHotspotCheckForSpawn()` picks them up with just 3 lines of integration code (lines 838-840). This validates the design's claim that "spawn gates get coupling awareness for free."
