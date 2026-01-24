<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Browser caches Vite dev assets across server restarts; vite.config.ts lacks cache-control headers to force fresh fetches.

**Evidence:** Minimal vite config with no cache headers; SvelteKit defaults don't prevent browser caching in dev mode; commit 934f5eeb added model badges but didn't address dev cache invalidation.

**Knowledge:** Vite dev server restarts don't automatically trigger browser cache invalidation; browser maintains cached JS/CSS until explicit hard refresh (Cmd+Shift+R).

**Next:** Add cache-control headers to vite dev server config to prevent browser caching during development.

**Promote to Decision:** recommend-no (tactical fix for dev experience, not architectural)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Bug Model Badges Not Visible

**Question:** Why do model badges not appear in dashboard after vite dev server restart without browser hard refresh?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** architect agent (orch-go-lai6h)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Vite config lacks cache-control headers for dev server

**Evidence:** The vite.config.ts file only configures SvelteKit plugin and proxy settings. No server.headers configuration exists to control browser caching behavior during development.

**Source:** web/vite.config.ts:1-24

**Significance:** Without explicit cache-control headers, browsers apply default caching strategies which can cache JavaScript bundles across dev server restarts, causing stale UI to persist until hard refresh.

---

### Finding 2: Model visibility feature added without dev cache consideration

**Evidence:** Commit 934f5eeb added model badge display to agent cards (lines 561-572 in agent-card.svelte), but the feature required hard refresh to become visible after vite restart.

**Source:** git show 934f5eeb, web/src/lib/components/agent-card/agent-card.svelte:561-572

**Significance:** The UI change was correct, but the dev environment wasn't configured to handle asset invalidation on server restart, creating friction during development.

---

### Finding 3: SvelteKit adapter-static with no dev-specific cache config

**Evidence:** svelte.config.js uses @sveltejs/adapter-static for build output, but doesn't configure dev-specific caching behavior. Vite v6.4.1 is in use but with minimal configuration.

**Source:** web/svelte.config.js:1-23, package.json vite version

**Significance:** The static adapter is for production builds; dev server behavior is controlled by vite.config.ts, which needs explicit cache headers to prevent browser caching during development.

---

## Synthesis

**Key Insights:**

1. **Dev server restarts break HMR assumptions** - When vite restarts, the HMR WebSocket reconnects, but the browser still trusts its cached assets. Vite assumes HMR can patch changes, but after a full server restart, cached bundles are stale and HMR can't know what changed.

2. **Browser cache is the bottleneck** - The issue isn't with vite's HMR mechanism itself, but with the browser's default caching strategy. Without explicit no-cache headers for dev assets, browsers cache JavaScript bundles and serve them from cache even after server restart.

3. **Production vs development caching needs** - Production builds benefit from aggressive caching (fingerprinted assets, long max-age). Development needs the opposite: no caching to ensure every server restart delivers fresh assets.

**Answer to Investigation Question:**

Model badges don't appear after vite restart because the browser serves cached JavaScript bundles instead of fetching the new bundle from the restarted server. The vite.config.ts lacks cache-control headers to prevent this caching during development. The fix is to add `server.headers` configuration with `Cache-Control: no-store` for dev mode, forcing the browser to always fetch fresh assets.

---

## Structured Uncertainty

**What's tested:**

- ✅ Vite config lacks cache headers (verified: read web/vite.config.ts, no server.headers present)
- ✅ Model badge code exists and is correct (verified: read agent-card.svelte:561-572, conditional rendering on agent.model)
- ✅ Issue introduced in commit 934f5eeb (verified: git show 934f5eeb shows model badge addition)

**What's untested:**

- ⚠️ Adding cache-control headers will fix the issue (hypothesis - needs browser testing after implementation)
- ⚠️ Issue only affects model badges (may affect other dynamic UI updates after server restart)
- ⚠️ Hard refresh consistently works around the issue (assumed from bug report, not independently verified)

**What would change this:**

- Finding would be wrong if browser still caches after no-cache headers are added
- Finding would be wrong if the issue is actually in the API data flow rather than browser caching
- Finding would be wrong if SvelteKit has a different mechanism for dev cache invalidation that we're not using

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add cache-control headers to vite dev server** - Configure vite.config.ts to send Cache-Control: no-store headers for all dev server responses.

**Why this approach:**
- Directly addresses root cause (browser caching dev assets)
- Standard Vite configuration pattern (server.headers is documented API)
- Zero impact on production builds (only affects dev server)
- Prevents ALL dev cache issues, not just model badges

**Trade-offs accepted:**
- Slightly slower dev experience (browser always fetches fresh assets instead of using cache)
- Network overhead for every asset request (acceptable for local dev server)
- Why that's acceptable: Dev experience correctness > speed. Better to always see fresh UI than deal with cache invalidation bugs.

**Implementation sequence:**
1. Add server.headers configuration to vite.config.ts - foundational fix
2. Test with vite server restart + browser refresh (no hard refresh) - verify fix works
3. Document in commit message that this prevents dev cache issues - knowledge transfer

### Alternative Approaches Considered

**Option B: Add timestamp query params for cache busting**
- **Pros:** Forces browser to fetch new assets by changing URLs
- **Cons:** Requires modifying build/serve process; doesn't address root cause (browser still tries to cache); more complex than headers
- **When to use instead:** If cache-control headers don't work due to CDN/proxy interference

**Option C: Document hard refresh requirement in CLAUDE.md**
- **Pros:** No code changes; simple documentation update
- **Cons:** Doesn't fix the issue, just documents the workaround; friction remains; every developer needs to remember manual step
- **When to use instead:** If technical fix is impossible or carries unacceptable trade-offs (neither applies here)

**Option D: Configure HMR to force full reload on reconnect**
- **Pros:** Vite HMR would trigger browser reload when server restarts
- **Cons:** Doesn't prevent browser from serving cached assets; reload still fetches from cache if headers aren't set; harder to configure than headers
- **When to use instead:** As complementary fix after headers are added

**Rationale for recommendation:** Cache-control headers address the root cause with minimal code change and standard Vite API. Other options either work around the symptom (docs) or add complexity without solving the core issue (query params, HMR config).

---

### Implementation Details

**What to implement first:**
- Add `server.headers: { 'Cache-Control': 'no-store' }` to vite.config.ts export
- This is a single object addition to the existing config
- Applies to all dev server responses (JS, CSS, HTML)

**Things to watch out for:**
- ⚠️ Only apply to dev mode (production builds should have normal caching)
- ⚠️ Vite config is TypeScript - ensure proper typing for server.headers
- ⚠️ Test with actual vite restart scenario (not just HMR updates)

**Areas needing further investigation:**
- Whether other UI updates have similar caching issues (not just model badges)
- If SvelteKit has recommended cache configuration for dev mode
- Performance impact of no-cache headers (expected to be minimal for local dev)

**Success criteria:**
- ✅ After vite server restart, browser refresh (not hard refresh) shows updated UI
- ✅ Model badges appear immediately when agents are spawned with model data
- ✅ No manual hard refresh required after any vite restart

---

## References

**Files Examined:**
- web/vite.config.ts:1-24 - Confirmed no cache-control headers configured
- web/svelte.config.js:1-23 - Reviewed adapter configuration (static adapter, no dev cache config)
- web/src/lib/components/agent-card/agent-card.svelte:561-572 - Model badge rendering logic
- web/src/app.html:1-14 - HTML template (no cache meta tags)

**Commands Run:**
```bash
# Check vite version
cd web && npm list vite

# Find model visibility commit
git log --all --oneline --grep="model" --since="2 weeks ago"

# View commit that added model badges
git show 934f5eeb --stat

# Check for service workers
find web -name "service-worker*" -o -name "sw.js"

# Check for cache config
cd web && grep -r "cache" *.config.*
```

**External Documentation:**
- Vite server.headers API - Standard configuration for dev server HTTP headers
- Browser Cache-Control directives - no-store prevents caching entirely

**Related Artifacts:**
- **Commit:** 934f5eeb - Added model visibility feature to dashboard

---

## Investigation History

**2026-01-18 14:30:** Investigation started
- Initial question: Why do model badges not appear without hard refresh after vite restart?
- Context: Bug report from issue orch-go-u5o9w noting model visibility feature requires manual hard refresh

**2026-01-18 14:35:** Identified root cause
- Browser caching dev assets due to missing cache-control headers in vite.config.ts
- No service worker involved, no existing cache configuration

**2026-01-18 14:40:** Analysis phase complete
- Recommendation: Add server.headers with Cache-Control: no-store
- Ready to implement fix

**2026-01-18 14:45:** Implementation complete
- Added Cache-Control: no-store header to vite.config.ts server configuration
- Documented rationale in code comments
- Investigation complete - requires browser testing to verify fix works
