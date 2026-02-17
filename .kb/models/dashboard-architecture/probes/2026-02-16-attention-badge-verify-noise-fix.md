# Probe: Attention Badge Verify Noise Fix

**Model:** dashboard-architecture
**Date:** 2026-02-16
**Status:** Active

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

### Current Behavior

- `mapSignalToBadge()` has explicit cases for 6 signal types:
  - `recently-closed` (with verification status handling)
  - `likely-done`
  - `stuck`
  - `unblocked`
  - `verify`
  - `verify-failed`
- Default case at line 138: `return 'verify'`
- This causes all unmapped signals (`issue-ready`, `stale`, `duplicate-candidate`, `competing`, `epic-orphaned`) to show as 'verify' badges

### The Cascade

```
BeadsCollector → signal: "issue-ready"
  → mapSignalToBadge() → default → 'verify'
    → attention store → signals.set(issueId, { badge: 'verify', ... })
      → tree-helpers:317 → if (node.attentionBadge === 'verify')
        → text: 'Awaiting review (Phase: Complete)'
```

### Implementation Plan

**File 1: attention.ts**
- Line 136-139: Change default case to return `null`
- Line ~80-90: Update the store's `set()` method to filter out null badges

**File 2: work-graph-tree-helpers.ts**
- Line 317: Add null guard before checking badge type

---

## Model Impact

- [ ] **Testing in progress** - Implementation not yet verified

---

## Notes

- The probe from 2026-02-16-attention-badge-verify-noise.md identified this issue
- This probe documents the fix implementation
- Will update Status to Complete after verification
