# Session Synthesis

**Agent:** og-inv-post-mortem-two-27dec
**Issue:** orch-go-o22b
**Duration:** 2025-12-27 → 2025-12-27
**Outcome:** success

---

## TLDR

Post-mortem on two-mode dashboard agent found root cause: SSR hydration bug where store initialized at module load (during SSR) but never re-initialized during browser hydration. Agent validated only build/typecheck, never loaded in browser. Fix exists but uncommitted. Key recommendation: UI features need browser verification gate.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-post-mortem-two-mode-dashboard.md` - Full post-mortem investigation

### Files Modified
- None (investigation-only task)

### Commits
- Pending (investigation file needs to be committed)

---

## Evidence (What Was Observed)

- Original agent's SYNTHESIS.md "Tests Run" showed only `bun run build` and `bun run check`, no browser testing
- Agent explicitly documented "not browser-tested" in "What's untested" section but completed anyway
- Git history shows SSR bug: `typeof window !== 'undefined'` check at module level runs during SSR, not hydration
- Uncommitted fix adds `init()` function called in `onMount()` with `browser` import from `$app/environment`
- Beads issue `orch-go-8uoh` confirms user-observed failure: "buttons rendered but clicking doesn't change view content"

### Tests Run
```bash
# Verified original commit content
git show c3743202 -- web/src/lib/stores/dashboard-mode.ts
# Shows typeof window check at module level

# Verified uncommitted fix exists
git diff -- web/src/lib/stores/dashboard-mode.ts
# Shows init() function added, browser import from $app/environment

# Confirmed agent's validation was build-only
# Read SYNTHESIS.md - only bun run build and bun run check documented
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-post-mortem-two-mode-dashboard.md` - Complete root cause analysis

### Decisions Made
- UI features require browser verification evidence before marking complete
- SvelteKit stores using browser APIs must use `onMount`/`init()` pattern with `browser` from `$app/environment`

### Constraints Discovered
- Build/typecheck success does NOT validate runtime behavior (especially SSR)
- `typeof window !== 'undefined'` at module level runs during SSR, never during hydration
- Agent self-documented uncertainty ("not browser-tested") but had no gate to force validation

### Externalized via `kn`
- Will run `kn constrain` after completing investigation

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file complete with D.E.K.N. summary
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-o22b`

### Follow-up Work Identified
1. **Update feature-impl skill** to require browser verification for UI features
2. **Add MCP Playwright recommendation** to spawn context for dashboard work
3. **Audit other stores** in `web/src/lib/stores/` for similar SSR bugs
4. **Commit the fix** for dashboard-mode.ts (currently uncommitted)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Are there other stores with similar SSR bugs? (audit recommended)
- Would Playwright MCP have caught this automatically?
- Should there be a lint rule for SvelteKit SSR patterns?

**Areas worth exploring further:**
- Automated smoke testing for UI features using MCP Playwright
- Adding "UI Feature" detection to spawn context for automatic gate application

**What remains unclear:**
- Whether the uncommitted fix is fully correct (not browser-tested in this investigation)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-inv-post-mortem-two-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-post-mortem-two-mode-dashboard.md`
**Beads:** `bd show orch-go-o22b`
