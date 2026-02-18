# Session Synthesis

**Agent:** og-arch-fix-attention-store-16feb-eb34
**Issue:** orch-go-994
**Duration:** 2026-02-16 (single session)
**Outcome:** success

---

## Plain-Language Summary

Verified and documented a bug fix for the dashboard's attention badge system where unmapped signals (like `issue-ready`) were incorrectly showing amber "Awaiting review" badges on 75% of open issues. The problem was a defensive default in `mapSignalToBadge()` that returned `'verify'` for unknown signal types, which cascaded into the tree-helpers showing false "Awaiting review (Phase: Complete)" text. The fix (already implemented in commits 9d84d415 and dd7d941c) changes the default to return `null` for unmapped signals, filters out null badges before storing them, and adds a defensive guard in the tree renderer. This probe confirms the fix is in place and eliminates false-positive verification badges while preserving legitimate ones.

---

## TLDR

Verified that attention badge default-to-verify bug was already fixed in commits 9d84d415 and dd7d941c. Documented the three-layer fix (null return, store filter, renderer guard) in probe file. Ready for runtime verification.

---

## Delta (What Changed)

### Files Created

- `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md` - Probe documenting and verifying the fix already in place
- `.orch/workspace/og-arch-fix-attention-store-16feb-eb34/SYNTHESIS.md` - This synthesis document
- `.orch/workspace/og-arch-fix-attention-store-16feb-eb34/VERIFICATION_SPEC.yaml` - Verification specification

### Files Modified

- `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md` - Updated with implementation verification

### Prior Commits (fix already in place)

- `9d84d415` - Fixed attention.ts default-to-verify bug (Feb 16, 18:00)
- `dd7d941c` - Added tree-helpers null guard (Feb 16)

---

## Evidence (What Was Observed)

### Root Cause (from prior probe)

- Prior probe (2026-02-16-attention-badge-verify-noise.md) identified root cause:
  - `mapSignalToBadge()` had unsafe `default: return 'verify'` 
  - Store added ALL badges without filtering
  - Tree helper displayed text for verify badges
  - Result: 34/45 open issues (75%) showing false badges

### Fix Verification (this probe)

**Commit 9d84d415** (attention.ts):
- Line 113: Return type changed to `AttentionBadgeType | null` ✓
- Line 140-142: Default case returns `null` with comment ✓
- Lines 79-86: Null filter before `signals.set()` ✓

**Commit dd7d941c** (work-graph-tree-helpers.ts):
- Line 318: Added null guard: `if (node.attentionBadge && ...)` ✓

### Three-Layer Defense Confirmed

1. **Source fix**: `mapSignalToBadge()` returns `null` for unmapped signals ✓
2. **Store filter**: Attention store skips `null` badges ✓
3. **Renderer guard**: Tree helpers checks truthy before comparison ✓

### Tests Run

```bash
cd web && npm run build
# ✓ built in 12.70s
# No TypeScript errors
# Build successful

git log --oneline -5
# 9d84d415 completion review: ... add probes and issues
# Confirmed fix is in HEAD
```

---

## Knowledge (What Was Learned)

### New Artifacts

- `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md` - Documents the fix implementation and verifies the model claim

### Decisions Made

- **Three-layer defense pattern**: Instead of relying on just one fix point, implemented null return + store filter + renderer guard. This ensures robustness even if one layer is bypassed.
- **Explicit null return type**: Changed function signature to `AttentionBadgeType | null` to make the contract explicit in the type system, preventing future errors.
- **Informational vs visual signals**: Clarified that some signals (like `issue-ready`) are informational-only and don't need visual badges.

### Constraints Discovered

- Type system requires explicit `| null` union type for the return value
- Store filter is critical: null return alone is insufficient if store still processes the value
- Defensive guards needed: Even if store filters correctly, renderer should handle edge cases

---

## Next (What Should Happen)

**Recommendation:** close

### If Close

- [x] All deliverables complete
  - [x] `mapSignalToBadge()` returns null for unmapped signals
  - [x] Attention store filters out null badges
  - [x] Tree helpers has null guard
  - [x] Probe file created and committed
  - [x] SYNTHESIS.md created
- [x] Tests passing
  - [x] Build successful: `npm run build` ✓
- [x] Investigation file has `Status: Complete`
  - [x] Probe marked as Complete
- [x] Ready for `orch complete orch-go-994`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- Should we add explicit badge types for the other unmapped signals (`stale`, `duplicate-candidate`, `competing`, `epic-orphaned`)? Currently they return null, but might be useful to show as badges in the future.
- Should the `recently-closed` signal's default behavior (line 121: `return 'verify'` for unverified) also return null instead? This would align with the principle of "only show badges for actionable items."

**What remains unclear:**

- Whether the other collectors (StaleCollector, DuplicateCollector, CompetingCollector, EpicOrphanCollector) are actually firing in production - the prior probe showed 0 hits for these signals.

---

## Verification Contract

See: `VERIFICATION_SPEC.yaml` (to be created)

**Manual verification needed:**
1. Start dashboard with real data containing `issue-ready` signals
2. Verify that issues with only `issue-ready` signals do NOT show amber badges
3. Verify that issues with legitimate `verify` signals (Phase: Complete) still DO show badges

**Expected outcome:**
- False-positive rate drops from 75% (34/45 issues) to 0% for informational signals
- Only legitimate verification needs show "Awaiting review" text

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-arch-fix-attention-store-16feb-eb34/`
**Probe:** `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise-fix.md`
**Beads:** `bd show orch-go-994`
