# Session Synthesis

**Agent:** og-inv-work-graph-tree-16feb-ff6d
**Issue:** orch-go-991
**Outcome:** success

---

## Plain-Language Summary

The work-graph tree shows amber "Awaiting verification" badges on 75% of open issues because of a `default: return 'verify'` fallback in the frontend's `mapSignalToBadge()` function. The backend attention API returns `issue-ready` signals for all 34 ready-to-work issues, but the frontend badge mapper has no case for `issue-ready` (or 4 other signal types), so they all silently become "Awaiting verification" badges. The fix is to change the default to return `null` instead of `'verify'`, so only signals with explicit badge mappings render badges.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

Key outcome: Root cause identified — `web/src/lib/stores/attention.ts:138` has `default: return 'verify'` which converts all unmapped signal types into false verification badges.

---

## Delta (What Changed)

### Files Created
- `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise.md` - Probe documenting root cause and data flow

### Files Modified
- None (investigation only)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **Live API test**: `/api/attention` returns 60 items: 34 `issue-ready`, 26 `recently-closed`
- **Tree overlap**: 34 of 45 open issues (75%) have attention signals, all `issue-ready`
- **All 34 `issue-ready` signals** fall through to `default: return 'verify'` in `mapSignalToBadge()`
- **0 `recently-closed` signals** match open tree nodes (closed issues aren't in open scope)
- **5 unmapped signal types**: `issue-ready`, `stale`, `duplicate-candidate`, `competing`, `epic-orphaned`
- **6 mapped signal types**: `recently-closed`, `likely-done`, `stuck`, `unblocked`, `verify`, `verify-failed`

### Data Flow Traced
```
BeadsCollector → Signal: "issue-ready"
  → /api/attention → item.signal = "issue-ready"
    → mapSignalToBadge() → default case → 'verify'
      → attention store signals map
        → +page.svelte attachBadges() → node.attentionBadge = 'verify'
          → amber "Awaiting verification" badge rendered
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The attention system has 11 collectors producing 11 distinct signal types, but the badge mapper only handles 6
- The `default` case acts as a catch-all that silently converts informational signals into verification alerts
- The `issue-ready` signal (from BeadsCollector) is the dominant noise source because ready issues are the most common attention signal

### Externalized via kb
- Probe file documents the finding permanently

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, fix is separate scope)

### Recommended Fix (out of scope for this session)
Change `attention.ts:138` from `default: return 'verify'` to `default: return null as any` (or adjust the return type). Then in the attention store's `mapSignalToBadge()` caller, skip adding signals with null badges to the signals map. This preserves all legitimate badges while eliminating noise.

### If Spawn Follow-up
**Issue:** Fix attention badge noise — change mapSignalToBadge default to null
**Skill:** feature-impl
**Context:**
```
Root cause: web/src/lib/stores/attention.ts:138 default returns 'verify' for unmapped signals.
Fix: Return null for unmapped signals, filter nulls in attention store before adding to signals map.
See probe: .kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise.md
```

---

## Unexplored Questions

- Should `issue-ready` get its own badge type (e.g., green "Ready to spawn") or no badge at all?
- Should the attention API filter out non-badge-worthy signals to reduce payload, or is it valuable for other consumers?
- The secondary issue of recently-closed defaulting to `unverified` may cause noise in future views that show closed issues

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-work-graph-tree-16feb-ff6d/`
**Probe:** `.kb/models/dashboard-architecture/probes/2026-02-16-attention-badge-verify-noise.md`
**Beads:** `bd show orch-go-991`
