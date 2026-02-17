# Probe: Daemon Treats relates_to Links as Blocking Dependencies

**Date:** 2026-02-16  
**Status:** Complete  
**Model:** Daemon Autonomous Operation

## Question

Does the daemon's dependency evaluation logic incorrectly treat `relates_to` edges as blocking dependencies when determining if an issue is spawnable?

**Model Claim Being Tested:**

From the daemon autonomous operation model, the daemon "polls beads for `triage:ready` issues" and performs skill inference and spawn decisions. The model describes capacity management and duplicate spawns but doesn't explicitly document how different dependency edge types (`blocks` vs `relates_to`) are handled during spawn evaluation.

**Expected Behavior:**
- `blocks` edges should prevent spawning (blocking dependency)
- `relates_to` edges should NOT prevent spawning (informational only)

## What I Tested

**Code Analysis:**

1. Traced dependency checking flow:
   - Daemon (`pkg/daemon/daemon.go:378`) calls `beads.CheckBlockingDependencies(issue.ID)`
   - This function (`pkg/beads/client.go:1075`) retrieves issue and calls `issue.GetBlockingDependencies()`
   - `GetBlockingDependencies()` (`pkg/beads/types.go:195-224`) determines which dependencies are blocking

2. Examined `GetBlockingDependencies()` switch logic:
   ```go
   switch dep.DependencyType {
   case "parent-child":
       isBlocking = false
   default:
       // "blocks" and other types: blocks unless closed or answered
       isBlocking = dep.Status != "closed" && dep.Status != "answered"
   }
   ```

3. Checked valid dependency types in codebase:
   - `cmd/orch/serve_beads.go` documents: `blocks, parent-child, relates_to`

## What I Observed

**Bug Confirmed:**

The `default` case in `pkg/beads/types.go:210-212` treats ALL non-"parent-child" dependency types as blocking, including `relates_to`. This is the root cause.

**Evidence:**
- Line 211 comment says: `"blocks" and other types: blocks unless closed or answered`
- This means `relates_to` dependencies are incorrectly treated as blocking
- The logic should ONLY treat `dependency_type="blocks"` as blocking, not use a catch-all default

**Impact Scope:**
- Every issue with a `relates_to` link to an open/in_progress issue is falsely marked as blocked
- Daemon silently skips these issues without user visibility
- Previous daemon cycles may have skipped work without anyone noticing

## Model Impact

**Extends the model with new invariant:**

The model documents "Skill Inference Mismatch" as a failure mode but doesn't document dependency type handling. This probe reveals:

**New Invariant: Dependency Type Semantics**
- `blocks`: Blocks spawning until closed/answered (intentional gate)
- `parent-child`: Never blocks (children are independently spawnable)
- `relates_to`: Should NEVER block (informational link only)

**Bug Type:** Logic error in `GetBlockingDependencies()` using catch-all default instead of explicit type checking.

**Recommended Model Update:** Add "Dependency Type Handling" section documenting the three types and their spawn semantics, plus this bug as a resolved failure mode.

## Verification

**Fix Applied:**
- Modified `pkg/beads/types.go:205-217` to explicitly check `dependency_type="blocks"` instead of using catch-all default
- Added explicit case for `dependency_type="relates_to"` → never blocking
- Preserved `dependency_type="parent-child"` → never blocking behavior

**Tests Added:**
Added 5 new test cases in `pkg/beads/client_test.go` for `relates_to` dependencies:
1. `relates_to: open does NOT block`
2. `relates_to: in_progress does NOT block`
3. `relates_to: closed does NOT block`
4. `mixed: blocks open + relates_to open` (only blocks dependency should block)
5. `mixed: blocks closed + relates_to open` (neither should block)

**Test Results:**
```bash
$ go test ./pkg/beads -run TestGetBlockingDependencies -v
=== RUN   TestGetBlockingDependencies
=== RUN   TestGetBlockingDependencies/relates_to:_open_does_NOT_block
=== RUN   TestGetBlockingDependencies/relates_to:_in_progress_does_NOT_block
=== RUN   TestGetBlockingDependencies/relates_to:_closed_does_NOT_block
=== RUN   TestGetBlockingDependencies/mixed:_blocks_open_+_relates_to_open
=== RUN   TestGetBlockingDependencies/mixed:_blocks_closed_+_relates_to_open
--- PASS: TestGetBlockingDependencies (0.00s)
PASS
ok  	github.com/dylan-conlin/orch-go/pkg/beads	0.005s
```

All 16 test cases pass (11 existing + 5 new).

**Behavior After Fix:**
- Issues with `relates_to` links to open/in_progress issues can now be spawned by daemon
- Only `blocks` dependency type prevents spawning
- `parent-child` and `relates_to` are correctly treated as non-blocking
