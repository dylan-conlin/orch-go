# Probe: Accretion Gate Pre-Existing Bloat Skip

**Status:** Complete
**Date:** 2026-02-24
**Model:** Completion Verification Architecture

## Question

Does the accretion gate correctly distinguish between pre-existing file bloat (file was already >1500 lines before agent's changes) and agent-caused bloat (agent's additions pushed file over threshold)?

## What I Tested

### Test 1: Pre-existing bloat behavior (before fix)
- Created file at 1600 lines, committed, then agent adds 70 lines (removes 10)
- `VerifyAccretionForCompletion()` checked `currentLines > 1500` and blocked with ERROR
- File was already 1600 lines before agent touched it — agent didn't cause bloat

### Test 2: Agent-caused bloat (should still block)
- Created file at 1400 lines, agent adds 120 lines → pushes to ~1520
- `preChangeLines = 1400 <= 1500` but `currentLines = 1520 > 1500`
- Agent's work crossed the threshold → should block

### Test 3: Fix implementation
Modified `VerifyAccretionForCompletion()` to calculate `preChangeLines = currentLines - netDelta`:
- If `preChangeLines > CriticalThreshold` → downgrade from ERROR to WARNING (pre-existing bloat)
- If `preChangeLines <= CriticalThreshold` but `currentLines > CriticalThreshold` → keep ERROR (agent caused it)

### Test 4: Skip flag escape hatch
Added `--skip-accretion` flag to `SkipConfig`, consistent with all other gates (`--skip-test-evidence`, `--skip-build`, etc.).

## What I Observed

### Before fix:
```
CRITICAL accretion: large.go is 1600 lines (+61 added, 1661 total). Files >1500 lines require extraction...
```
Gate blocked completion even though agent didn't cause the file to be 1600 lines.

### After fix:
```
Accretion warning (pre-existing bloat): large.go was already 1599 lines before this change (+61 added, 1660 total). File needs extraction but agent didn't cause the bloat.
```
Gate passes with warning. Agent's work isn't blocked by pre-existing problems.

### Agent-caused bloat still blocks:
```
CRITICAL accretion: growing.go is 1400 lines (+120 added, 1520 total). Files >1500 lines require extraction...
```
When agent pushes a file over the threshold, the gate still blocks correctly.

### Test results:
```
go test ./pkg/verify/ -run TestVerifyAccretion -v
--- PASS: TestVerifyAccretionForCompletion (0.75s)
    --- PASS: .../pre-existing_bloated_file_>1500_lines_downgrades_to_warning
    --- PASS: .../multiple_files,_mixed_results_(pre-existing_bloat_is_warning)
--- PASS: TestVerifyAccretionForCompletion_AgentCausedBloat (0.11s)
--- PASS: TestVerifyAccretionForCompletion_PreExistingBloatDetailed (0.11s)
--- PASS: TestVerifyAccretionForCompletion_BoundaryValues (0.54s)
    --- PASS: .../current_~1550_lines_(1500+50)_=_critical_(above_1500)   # agent pushes over → blocks
    --- PASS: .../current_~1551_lines_(1501+50)_=_critical_(well_above_1500)  # agent pushes over → blocks
PASS (12/12 tests)
```

## Model Impact

**Extends** the Completion Verification Architecture model:

- **Gate 6 (Accretion)** now has two behaviors:
  - Pre-existing bloat: downgrades from ERROR to WARNING (non-blocking)
  - Agent-caused bloat: keeps ERROR (blocking)
- The model's claim that accretion is a "hard gate" is now nuanced — it's hard only when the agent caused the threshold crossing
- Added `--skip-accretion` flag as escape hatch, consistent with model's "targeted bypasses" design
- The 14 gates still exist but gate behavior is now context-aware (pre-change vs post-change state)

**Key invariant preserved:** Files cannot grow past 1500 lines due to agent work. The gate still blocks agent-caused accretion.

**New invariant:** Agents are not penalized for pre-existing code debt they didn't create.
