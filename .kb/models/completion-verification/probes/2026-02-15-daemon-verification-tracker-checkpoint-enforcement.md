# Probe: Daemon Verification Tracker Reads Labels Instead of Checkpoint File

**Date:** 2026-02-15
**Model:** Completion Verification Architecture
**Status:** Active
**Issue:** orch-go-8h9l

## Question

Does the daemon's verification tracker correctly read the checkpoint file to count unverified completions, or does it rely on labels that disappear when issues are closed?

**Model claim being tested:**

The Completion Verification Architecture model describes verification gates and the VerificationTracker pause mechanism. The prior probe (2026-02-15-verification-tracker-wiring.md) confirmed the VerificationTracker is wired into the daemon. This probe tests whether the SeedFromBacklog() method correctly reads from the checkpoint file (source of truth) or incorrectly reads from labels (view layer).

## What I Tested

### Bug Reproduction

1. **Verified checkpoint file contains unverified completions:**
   ```bash
   grep -c '"gate2_complete":false' ~/.orch/verification-checkpoints.jsonl
   # Result: 13 entries with gate2_complete:false
   ```

2. **Verified daemon reports zero unverified completions:**
   ```bash
   orch daemon run --dry-run
   # Result: "Verification check: 0/3 unverified completions"
   ```

3. **Confirmed the composition bug:**
   - Checkpoint file correctly persists 13 unverified completions
   - Daemon reports 0/3 unverified completions
   - Verification pause gate does not fire (should refuse to spawn at 3+)

## What I Observed

**Bug confirmed:** The daemon's verification tracker is not reading the checkpoint file for enforcement. 

**Root cause analysis:**


Looking at `pkg/daemon/issue_adapter.go:CountUnverifiedCompletions()`:
- Line 168: Starts with `ListIssuesWithLabel("daemon:ready-review")`
- Line 184-189: Builds set of checkpoint IDs where `gate1_complete == true`
- Line 192-198: Counts issues without checkpoints

**The composition bug:**
1. When `orch complete` runs, it closes the issue AND removes the `daemon:ready-review` label
2. So closed issues disappear from `ListIssuesWithLabel()` results
3. Even though the checkpoint file still has entries with `gate2_complete:false`
4. The verification tracker never sees these unverified completions

**Evidence:**
- `bd show orch-go-05k7 --json` shows: `"status": "closed"`, `"issue_type": "feature"` (Tier 1)
- Checkpoint file has: `"beads_id":"orch-go-05k7"`, `"gate2_complete":false`
- This is closed Tier 1 work with incomplete verification, but daemon reports 0 unverified

## Model Impact

**Contradicts** current implementation assumption:

The current implementation assumes labels are the source of truth for unverified completions. This is wrong.

**Correct invariant:**
- **Checkpoint file is the source of truth for verification state**
- **Labels are the view layer** (active review queue)
- **Verification enforcement must read checkpoint file directly**

## Fix Design

### Approach: Checkpoint-First Counting

**Replace label-based counting with checkpoint-based counting:**

```go
func CountUnverifiedCompletions() (int, error) {
    // Read all checkpoints
    checkpoints, err := checkpoint.ReadCheckpoints()
    if err != nil {
        return 0, fmt.Errorf("failed to read checkpoints: %w", err)
    }
    
    // For each checkpoint, determine if unverified based on tier
    unverified := 0
    for _, cp := range checkpoints {
        // Need to look up issue type to determine tier
        issue, err := GetIssue(cp.BeadsID) // Need this function
        if err != nil {
            // Issue may be deleted - skip
            continue
        }
        
        tier := checkpoint.TierForIssueType(issue.Type)
        
        // Check if unverified based on tier
        if tier == 1 && !cp.Gate2Complete {
            unverified++ // Tier 1: needs both gates
        } else if tier == 2 && !cp.Gate1Complete {
            unverified++ // Tier 2: needs gate1
        }
    }
    
    return unverified, nil
}
```

**Status:** Active - implementing fix now

## Implementation Verification

### Fix Applied

**Location:** `pkg/daemon/issue_adapter.go:CountUnverifiedCompletions()`

**Changed from:**
- Read issues with `daemon:ready-review` label
- Filter out those with checkpoints
- Problem: closed issues lose label, disappear from count

**Changed to:**
- Read all checkpoints from file directly
- For each checkpoint, look up issue type from beads
- Determine tier based on issue type:
  - Tier 1 (feature/bug/decision): count if gate2_complete == false
  - Tier 2 (investigation/probe): count if gate1_complete == false
  - Tier 3 (task/question/other): skip (no verification required)
- Skip if issue lookup fails (may be deleted)

**Also added:** `showIssueCLI()` helper for CLI fallback when RPC unavailable

### Test Results

**Before fix:**
```bash
orch daemon run --dry-run
# Output: "Verification check: 0/3 unverified completions"
```

**After fix:**
```bash
orch daemon run --dry-run --verbose
# Output:
#   Verification backlog: 7 unverified completions from previous sessions
#   Warning: Verification pause: backlog exceeds threshold (7/3)
#   Run 'orch daemon resume' after reviewing completed work
# [DRY-RUN] Verification pause: 7 unverified completions, threshold is 3
```

**Verification:**
- Checkpoint file has 15 total entries
- 7 of those are unverified (based on tier requirements)
- Daemon correctly reports 7/3 and refuses to spawn
- Verification pause gate is now firing as designed

### Root Cause Confirmed

The composition bug was exactly as described in the issue:

1. **Checkpoint writing works** - `gate1_complete` and `gate2_complete` recorded correctly
2. **Label-based counting works** - counts labels accurately
3. **But they measure different things:**
   - Labels track active review queue (disappear when closed)
   - Checkpoints track verification state (persist after closure)
4. **The daemon used the wrong one** - read labels instead of checkpoints

The fix makes the checkpoint file the source of truth, as intended by the verifiability-first decision.

## Model Impact

**Confirms** the model's claim about checkpoint-based verification, and **extends** it with the implementation detail:

**New invariant:**
- **Checkpoint file is source of truth for enforcement** - `CountUnverifiedCompletions()` MUST read from checkpoint file, not from labels
- **Labels are view layer only** - `daemon:ready-review` shows active review queue, not verification state
- **Tier-aware counting** - Different issue types have different verification requirements (gate1 vs gate2)

**Status:** Complete - Fix verified and working
