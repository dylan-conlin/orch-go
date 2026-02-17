# Probe: Attention Badge "Awaiting verification" Noise

**Model:** dashboard-architecture
**Date:** 2026-02-16
**Status:** Complete

---

## Question

Why does the work-graph tree show amber "Awaiting verification" badges on nearly every open issue? Is the attention store returning real signals, or are they all defaulting to 'verify'?

---

## What I Tested

1. Traced the full data flow: backend collectors → `/api/attention` → attention store → `mapSignalToBadge()` → `getAttentionBadge()` → badge rendering in work-graph-tree.svelte
2. Hit the live attention API and graph API to measure actual signal distribution and overlap

```bash
# Signal distribution from /api/attention
curl -sk 'https://localhost:3348/api/attention' | python3 -c "..."
# Result: 60 total items: issue-ready (34), recently-closed (26)

# Overlap between attention signals and tree nodes
curl -sk 'https://localhost:3348/api/beads/graph?scope=open' | python3 -c "..."
# Result: 45 open issues, 34 have attention signals → 75% get badges
```

3. Audited `mapSignalToBadge()` in `web/src/lib/stores/attention.ts:110-140` against all 11 signal types emitted by collectors in `pkg/attention/`

---

## What I Observed

### Root Cause: `default: return 'verify'` in mapSignalToBadge()

The frontend `mapSignalToBadge()` function has explicit cases for 6 signal types but a `default` fallback that returns `'verify'`. Five collector signal types hit this default:

| Signal | Collector | Expected Badge | Actual Badge |
|--------|-----------|----------------|--------------|
| `issue-ready` | BeadsCollector | Should be "Ready" or none | `verify` ("Awaiting verification") |
| `stale` | StaleCollector | Should be "Stale" or none | `verify` ("Awaiting verification") |
| `duplicate-candidate` | DuplicateCollector | Should be "Duplicate?" or none | `verify` ("Awaiting verification") |
| `competing` | CompetingCollector | Should be "Competing" or none | `verify` ("Awaiting verification") |
| `epic-orphaned` | EpicOrphanCollector | Should be "Orphaned" or none | `verify` ("Awaiting verification") |

### Quantitative Impact

- **34 of 45 open issues (75%)** receive a badge
- **All 34** are `issue-ready` signals that hit the default → `verify` path
- **0 recently-closed** signals match open tree nodes (correct — closed issues aren't in the tree)
- No `stale`, `duplicate-candidate`, `competing`, or `epic-orphaned` signals were present at test time, but they would also produce false `verify` badges if they fired

### Secondary Issue: recently-closed items default to unverified

In `serve_attention.go:298-303`, ALL recently-closed items get `verification_status: "unverified"` before checking the verifications log. In `mapSignalToBadge()`, unverified recently-closed items map to `verify`. This means any closed issue without an explicit verification record also shows "Awaiting verification" (though these don't appear in the open-issues tree, they would appear in any view showing closed issues).

### Data Flow Summary

```
BeadsCollector.Collect() → Signal: "issue-ready"
    → /api/attention response → item.signal = "issue-ready"
        → attention.ts mapSignalToBadge() → default case → 'verify'
            → attention store signals.set(issueId, { badge: 'verify', reason: ... })
                → +page.svelte attachBadges() → node.attentionBadge = 'verify'
                    → work-graph-tree.svelte renders amber "Awaiting verification"
```

### The Intended Behavior (from code comments + badge config)

The badge system was designed for 9 distinct signal types:
- **verify** (amber): Issues where Phase: Complete was reported, needs `orch complete` — genuine verification needs
- **decide** (purple): Investigation has recommendation needing decision
- **escalate** (yellow): Question needs human judgment
- **likely_done** (blue): Commits suggest completion
- **recently_closed** (gray): Recently closed and verified
- **unblocked** (green): Blocker just closed, now actionable
- **stuck** (orange): Agent stuck >2h
- **crashed** (red): Agent crashed
- **verify_failed** (red): Verification failed during auto-completion

The `issue-ready` signal was never intended to show a badge. It's an Actionability signal ("this issue is ready to spawn"), not a verification signal. The BeadsCollector predates the badge system — when the badge system was wired up, `mapSignalToBadge()` was given an unsafe default that mapped unknown signals to `verify` instead of returning `null`/`undefined`.

---

## Model Impact

- [x] **Extends** model with: The attention badge system has a design flaw where `mapSignalToBadge()` defaults unknown signal types to `verify`, causing 75% of open issues to show false "Awaiting verification" badges. The root cause is the `default: return 'verify'` case in `attention.ts:138`, combined with the BeadsCollector emitting `issue-ready` signals that have no badge mapping. The fix is either (a) change the default to `return undefined`/filter out unmapped signals, or (b) add explicit badge types for all 11 signal types, or (c) filter out non-badge-worthy signals before they reach the frontend.

---

## Notes

- The `issue-ready` signal is the dominant contributor (34/34 badges) because the other unmapped signal types (stale, duplicate-candidate, competing, epic-orphaned) had 0 hits at the time of testing. In a more active project with older issues, stale/duplicate signals would compound the noise further.
- Reverting commit b3adb9fc would remove the badge rendering, but the underlying signal mapping issue would remain — a ticking bomb for whenever badges are re-enabled.
- The cleanest fix: `mapSignalToBadge()` should return `null` for unmapped signals (change `default: return 'verify'` to `default: return null` or similar), and the attention store should filter out null-badge signals before adding them to the signals map. This preserves all legitimate badges while eliminating noise from informational-only signals.
