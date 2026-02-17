# Probe: Extraction Gate Fail-Fast Fix

**Status:** Complete
**Date:** 2026-02-17
**Model:** daemon-autonomous-operation

## Question

Does the daemon extraction gate correctly fail-fast when extraction setup fails, or does it violate the spawn prerequisite fail-fast constraint by proceeding with normal spawn anyway?

## What I Tested

1. Read `pkg/daemon/daemon.go` lines 820-857 to examine extraction setup logic
2. Identified the warn-and-continue anti-pattern at lines 832-837
3. Will implement fix to skip issue and return error instead of falling through

## What I Observed

**Finding:** The fix was already implemented in commit bb055f49 (2026-02-17 09:35:28).

**Fixed behavior (lines 829-842 in pkg/daemon/daemon.go):**
```go
extractionID, err := createFunc(extraction.ExtractionTask, issue.ID)
if err != nil {
	// Extraction gate is non-negotiable: if setup fails, skip the issue
	// and return error (fail-fast). Do not proceed with normal spawn.
	if d.Config.Verbose {
		fmt.Printf("  Extraction setup failed for %s: %v (skipping issue)\n", issue.ID, err)
	}
	return &OnceResult{
		Processed: false,
		Message:   fmt.Sprintf("Extraction setup failed for %s: %v (issue skipped, will retry on next poll)", issue.ID, err),
	}, nil
}

// On success, spawn extraction work...
```

**The fix:**
- When `extraction.Needed == true` (hotspot detected, extraction required)
- And `createFunc` fails (can't create extraction issue)
- Code now returns immediately with Processed=false (fail-fast)
- Issue is NOT spawned - stays in `triage:ready` for next poll
- No fallback to normal spawn without extraction

**Test verification:**
Updated test `TestOnceExcluding_AutoExtraction_FailsFastOnExtractionFailure` to verify:
1. When extraction setup fails, Processed=false
2. spawnFunc is NOT called (issue skipped)
3. Result includes explanatory message

Test passes: ✅

## Reproduction Verification

**Original bug:** Code logged warning and proceeded with normal spawn when extraction setup failed.

**Reproduction steps:**
1. Create issue targeting file >1500 lines (hotspot detected)
2. Make extraction issue creation fail (e.g., `bd create` fails)
3. Observe daemon behavior

**Before fix:** Would spawn work on the hotspot file anyway (warn-and-continue).

**After fix:** Skips the issue, returns Processed=false, waits for next poll (fail-fast).

**Verified via test:** The test simulates extraction failure and confirms no spawn occurs.

## Model Impact

**Confirms:** The daemon warn-and-continue anti-pattern inventory (probe 2026-02-15) identified this exact pattern. The fix validates that spawn prerequisites must be hard gates.

**Confirms:** kb-035b64 constraint: "Spawn prerequisites are hard gates, not soft warnings. Pattern: if a spawn prerequisite fails, return error or skip the issue - never log warning and spawn anyway."

**Extends:** Demonstrates the extraction gate is properly enforced as a non-negotiable spawn prerequisite, alongside dependency checks, epic expansion, and beads status updates.
