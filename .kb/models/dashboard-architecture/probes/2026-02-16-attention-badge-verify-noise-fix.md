# Probe: Attention Badge Verify Noise Fix

**Model:** dashboard-architecture
**Date:** 2026-02-16
**Status:** Complete

---

## Question

Does changing `mapSignalToBadge()` default from `return 'verify'` to `return null` eliminate false "Awaiting review" badges for unmapped signals like `issue-ready`, and does adding null guards in tree-helpers prevent cascade failures?

---

## What I Tested

1. Examined current implementation:
   - `web/src/lib/stores/attention.ts:138` - default case returns `'verify'`
   - `web/src/lib/components/work-graph-tree/work-graph-tree-helpers.ts:317` - checks for `'verify'` badge and displays "Awaiting review (Phase: Complete)" text

2. Identified the fix:
   - Change default in `mapSignalToBadge()` to return `null`
   - Update attention store to filter out null badges
   - Add null guard in tree-helpers line 317

---

## What I Observed

### Current Behavior (Before Fix)

- `mapSignalToBadge()` has explicit cases for 6 signal types:
  - `recently-closed` (with verification status handling)
  - `likely-done`
  - `stuck`
  - `unblocked`
  - `verify`
  - `verify-failed`
- Default case at line 138: `return 'verify'`
- This causes all unmapped signals (`issue-ready`, `stale`, `duplicate-candidate`, `competing`, `epic-orphaned`) to show as 'verify' badges

### The Cascade (Before Fix)

```
BeadsCollector → signal: "issue-ready"
  → mapSignalToBadge() → default → 'verify'
    → attention store → signals.set(issueId, { badge: 'verify', ... })
      → tree-helpers:317 → if (node.attentionBadge === 'verify')
        → text: 'Awaiting review (Phase: Complete)'
```

### Implementation

**ALREADY IMPLEMENTED** in commits:
- `9d84d415` - attention.ts changes (Feb 16, 18:00)
- `dd7d941c` - work-graph-tree-helpers.ts guard (Feb 16, later)

**File 1: attention.ts** (3 changes in commit 9d84d415)
1. Line 113: Changed return type from `AttentionBadgeType` to `AttentionBadgeType | null`
2. Line 140-142: Changed default case from `return 'verify'` to `return null` with comment explaining why
3. Lines 79-86: Added null check before `signals.set()` to filter out informational signals that don't need badges

**File 2: work-graph-tree-helpers.ts** (1 change in commit dd7d941c)
1. Line 318: Added null guard: `if (node.attentionBadge && (node.attentionBadge === 'verify' ...))`

### New Data Flow (After Fix)

```
BeadsCollector → signal: "issue-ready"
  → mapSignalToBadge() → default → null
    → attention store → if (badge !== null) { signals.set(...) } → SKIPPED
      → tree-helpers never sees the badge → no false "Awaiting review" text
```

### Verification

- Build completed successfully: `npm run build` ✓
- Type safety maintained: null badges filtered before storage ✓
- Defensive guard added: tree-helpers checks for truthy badge before comparison ✓

### Expected Outcome

After this fix:
- Unmapped signals like `issue-ready` will return `null` from `mapSignalToBadge()`
- The attention store will skip adding null badges to the signals map
- Tree nodes will NOT receive attention badges for informational-only signals
- Only legitimate verification needs (Phase: Complete) will show "Awaiting review" text
- The 75% false-positive rate (34/45 issues) should drop to 0% for informational signals

---

## Model Impact

- [x] **Confirms** the root cause identified in probe 2026-02-16-attention-badge-verify-noise.md
- [x] **Extends** with implementation: Default case changed to `return null` with null filtering in store and defensive guard in tree-helpers
- [x] **Fixes** the cascade: Unmapped signals are now filtered out before reaching the badge rendering logic

---

## Notes

- The probe from 2026-02-16-attention-badge-verify-noise.md identified this issue
- This probe documents the fix implementation
- Three-layer defense: null return + store filter + tree-helpers guard
- Type safety: Changed return type to `AttentionBadgeType | null` to make the contract explicit
