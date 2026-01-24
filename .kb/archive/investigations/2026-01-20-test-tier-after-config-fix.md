# Test Tier After Config Fix

**Status:** Complete
**Date:** 2026-01-20
**Type:** Simple

## TLDR

Verified tier assignment works correctly after the config fix (commit 42027eb2). feature-impl spawns correctly get "light" tier, investigation spawns get "full" tier.

## What I Tried

1. Checked my own spawn workspace (.orch/workspace/og-feat-test-tier-after-20jan-b1ab)
2. Verified tier files across today's workspaces
3. Reviewed the SkillTierDefaults configuration
4. Compared workspace creation times vs the commit time of the fix

## What I Observed

### My Spawn (feature-impl)
- Workspace: `og-feat-test-tier-after-20jan-b1ab`
- `.tier` file content: `light` ✓
- Correctly assigned light tier

### Investigation Workspaces
All investigation workspaces from today have `full` tier:
- og-inv-dummy-task-backend-20jan-0b9c: full
- og-inv-read-claude-md-20jan-2046: full
- og-inv-simple-data-analysis-20jan-5c1d: full
- og-inv-test-backend-selection-20jan-7a5a: full
- og-inv-verify-default-backend-20jan-5073: full
- og-inv-verify-default-model-20jan-a50c: full

### Systematic-Debugging Workspaces (Pre-Fix)
Two debug workspaces from today show `full` tier:
- og-debug-fix-global-config-20jan-2a4c: full
- og-debug-fix-orch-spawn-20jan-af74: full

**Root cause:** These were created at 21:04 UTC, BEFORE the tier fix was committed at 22:34 UTC. This is expected behavior.

### Configuration State
```go
var SkillTierDefaults = map[string]string{
    // Full tier: Investigation-type skills
    "investigation":  TierFull,
    "architect":      TierFull,
    "research":       TierFull,
    "codebase-audit": TierFull,
    "design-session": TierFull,

    // Light tier: Implementation-focused skills
    "feature-impl":         TierLight,
    "systematic-debugging": TierLight, // Fixed in 42027eb2
    "reliability-testing":  TierLight,
    "issue-creation":       TierLight,
}
```

## Test Performed

1. **feature-impl spawn verification:** My spawn correctly has `.tier = light`
2. **Investigation spawns:** All have `.tier = full` as expected
3. **Configuration review:** SkillTierDefaults correctly maps:
   - feature-impl → light
   - systematic-debugging → light (fix applied)
   - investigation → full
4. **Temporal analysis:** Pre-fix debug workspaces correctly have `full` tier

## Conclusion

Tier assignment is working correctly after the config fix:

1. **feature-impl** spawns get `light` tier ✓
2. **investigation** spawns get `full` tier ✓
3. **systematic-debugging** is now configured for `light` tier ✓
4. Pre-fix workspaces correctly retain their original tier values

The fix in commit 42027eb2 was successful. New systematic-debugging spawns will correctly receive light tier.

## Notes

- Could not run Go unit tests (Go not available in container)
- No new systematic-debugging spawns to verify post-fix behavior directly (constraint: workers shouldn't spawn agents)
- Code inspection confirms the fix is correctly applied
