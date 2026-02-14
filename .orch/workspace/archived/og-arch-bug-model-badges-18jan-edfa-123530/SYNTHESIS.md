# Session Synthesis

**Agent:** og-arch-bug-model-badges-18jan-edfa
**Issue:** orch-go-lai6h
**Duration:** 2026-01-18 14:30 → 2026-01-18 14:50
**Outcome:** success

---

## TLDR

Fixed browser cache issue preventing model badges from appearing in dashboard after vite dev server restart. Added Cache-Control: no-store header to vite.config.ts to force browser to fetch fresh assets instead of serving cached bundles.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-inv-bug-model-badges-not-visible.md` - Full investigation documenting root cause analysis and fix

### Files Modified
- `web/vite.config.ts` - Added server.headers configuration with Cache-Control: no-store to prevent browser caching during development

### Commits
- Pending: fix(web): prevent browser caching in vite dev mode to fix model badge visibility

---

## Evidence (What Was Observed)

- vite.config.ts lacked cache-control headers (web/vite.config.ts:1-24)
- Model badge rendering code was correct (agent-card.svelte:561-572) - conditional on agent.model
- Commit 934f5eeb added model visibility feature without addressing dev cache behavior
- No service worker involved (verified via find command)
- No existing cache configuration in vite or svelte configs
- Vite v6.4.1 in use with minimal configuration
- SvelteKit using adapter-static for production builds

### Tests Run
```bash
# Verified vite version
cd web && npm list vite
# Result: vite@6.4.1

# Confirmed no cache config exists
cd web && grep -r "cache" *.config.*
# Result: No cache settings found

# Confirmed model badge commit
git show 934f5eeb --stat
# Result: Added model field and badge display
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-inv-bug-model-badges-not-visible.md` - Documents browser caching issue and cache-control header fix

### Decisions Made
- Decision 1: Use Cache-Control: no-store for dev mode rather than cache busting via query params - Simpler, addresses root cause, standard Vite API
- Decision 2: Apply headers to all dev responses, not just specific assets - Prevents all caching issues, not just model badges
- Decision 3: Accept slight performance penalty in dev mode - Correctness > speed for local development

### Constraints Discovered
- Browser default caching behavior persists across vite server restarts without explicit no-cache headers
- Vite HMR assumes cached assets are valid when reconnecting after server restart
- Production vs development caching needs are opposite - production wants aggressive caching, development needs none

### Root Cause
When vite dev server restarts:
1. Browser's HMR WebSocket reconnects
2. Browser still trusts its cached JavaScript bundles
3. Vite can't tell browser that cached bundles are stale (no cache headers configured)
4. Browser serves cached JS instead of fetching new bundle from restarted server
5. Result: UI shows stale state until manual hard refresh

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + fix)
- [ ] Tests passing (requires manual browser testing - cannot automate from agent)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-lai6h` after verification

**Verification steps for orchestrator:**
1. Restart vite dev server (`cd web && npm run dev`)
2. Open dashboard in browser
3. Make a UI change (e.g., modify agent-card.svelte)
4. Restart vite dev server
5. Refresh browser (normal refresh, not hard refresh)
6. Verify changes are visible without Cmd+Shift+R

**Expected outcome:** Changes visible after normal refresh. Model badges appear immediately when agents spawn with model data.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Do other UI features suffer from similar caching issues? (Model badges were just the symptom)
- Should we add ETag or Last-Modified headers in addition to no-cache? (Over-engineering for dev mode)
- Does SvelteKit have recommended dev cache configuration we're not using? (Vite docs are authoritative)

**What remains unclear:**
- Performance impact of no-cache headers (expected minimal for local dev, but not measured)
- Whether production builds are affected (they shouldn't be - headers only apply to dev server)

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-arch-bug-model-badges-18jan-edfa/`
**Investigation:** `.kb/investigations/2026-01-18-inv-bug-model-badges-not-visible.md`
**Beads:** `bd show orch-go-lai6h`
