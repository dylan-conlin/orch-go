<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Svelte 5 runes mode was incompatible with legacy `$:` reactive syntax and `$` store auto-subscription, causing the dashboard to show 0 agents despite the store containing 209 agents.

**Evidence:** Component logs showed `$agents.length = 0` in derived/reactive statements while store debugging showed 209 agents; error message confirmed "`$:` is not allowed in runes mode"; removing all runes (`$state`, `$derived`, `$effect`) and using pure Svelte 4 syntax fixed the issue immediately.

**Knowledge:** In Svelte 5, using any rune (like `$state`) triggers "runes mode" for the entire component, which disables legacy Svelte 4 reactive syntax (`$:`); mixing both causes silent reactivity failures where stores appear empty to the component.

**Next:** Fix committed; consider either fully migrating to Svelte 5 runes or standardizing on Svelte 4 syntax across all components to avoid future mixing issues.

**Confidence:** Very High (98%) - root cause proven with immediate fix verification.

---

# Investigation: Dashboard Shows 0 Agents Despite API Returning 209

**Question:** Why does the dashboard show 0 agents when the API returns 209 agents and the store contains the data?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** systematic-debugging agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (98%)

---

## Findings

### Finding 1: API and Store Working Correctly

**Evidence:** 
- API endpoint `/api/agents` returns 209 agents (verified with `curl`)
- Store fetch logs showed "Store now contains: 209 agents"
- Store debugging in browser console confirmed 209 agents in store

**Source:** 
- `curl http://127.0.0.1:3333/api/agents | jq '. | length'` returned 209
- Browser console logs from `web/src/lib/stores/agents.ts:94`
- Network tab showed successful 200 response with agent data

**Significance:** The backend and data layer were functioning correctly; the issue was in the UI layer's reactivity, not data fetching.

---

### Finding 2: Component Showing 0 Agents Despite Store Having Data

**Evidence:**
- Component logs showed: `[+page] filteredAgents recomputing - $agents.length: 0`
- Store debugging showed: `Store now contains: 209 agents`
- These logs appeared simultaneously, proving a disconnect between store and component

**Source:**
- Vite dev server logs in `/tmp/vite-new.log`
- Added debug logging to `web/src/routes/+page.svelte:42`

**Significance:** This proved the issue was a reactivity problem - the component wasn't seeing store updates despite the store being updated.

---

### Finding 3: Svelte 5 Runes Mode Incompatibility

**Evidence:**
- Component had `let statusFilter = $state('all')` declarations
- Browser showed error: "`$:` is not allowed in runes mode, use `$derived` or `$effect` instead"
- Component mixed Svelte 5 runes with Svelte 4 `$:` syntax and `$` store auto-subscription
- Removing all runes immediately fixed the issue

**Source:**
- Error overlay in browser at `web/src/routes/+page.svelte:29`
- Component source code showed mixed syntax
- Svelte 5 documentation on runes mode

**Significance:** Svelte 5's runes mode is all-or-nothing - using any rune disables legacy reactive syntax, causing silent failures when mixed.

---

## Synthesis

**Key Insights:**

1. **Svelte 5 Runes Trigger Mode Switch** - Using even a single `$state`, `$derived`, or `$effect` declaration puts the entire component into "runes mode", which disables all Svelte 4 reactive features including `$:` declarations and `$` store auto-subscription.

2. **Silent Reactivity Failures** - When runes mode is enabled but legacy syntax is used, Svelte doesn't always show errors immediately; some reactive statements compile but don't actually trigger on changes, leading to stale data in the UI.

3. **All-or-Nothing Migration** - Svelte 5 components must be either fully runes-based or fully legacy syntax; mixing causes unpredictable reactivity behavior and is not supported.

**Answer to Investigation Question:**

The dashboard showed 0 agents because the component was in Svelte 5 runes mode (triggered by `$state` declarations) while using legacy `$:` reactive syntax and `$` store subscriptions. In runes mode, these legacy features don't work, causing the component to see an empty array despite the store containing 209 agents. The fix was to remove all runes and use pure Svelte 4 syntax throughout the component.

---

## Confidence Assessment

**Current Confidence:** Very High (98%)

**Why this level?**

The root cause was definitively proven through systematic debugging, immediate fix verification, and direct observation of the error message. The fix was tested and confirmed working with visual verification showing all 209 agents in the dashboard.

**What's certain:**

- ✅ Svelte 5 runes mode was triggered by `$state` declarations
- ✅ Legacy `$:` syntax doesn't work in runes mode
- ✅ Removing runes immediately fixed the issue (tested and verified)
- ✅ Dashboard now displays all 209 agents correctly

**What's uncertain:**

- ⚠️ Whether other components in the codebase have similar mixing issues (not investigated)
- ⚠️ Long-term strategy for Svelte 5 migration (out of scope)

**What would increase confidence to 100%:**

- Audit all components for runes/legacy mixing
- Establish team convention for Svelte 5 usage

---

## Implementation Recommendations

### Recommended Approach ⭐

**Standardize on Svelte 4 Syntax** - Remove all Svelte 5 runes from components and use legacy syntax until full migration is planned.

**Why this approach:**
- Svelte 4 syntax is stable and well-understood by the team
- Avoids partial migration issues and mixing problems
- Maintains consistency across all components
- Defers full Svelte 5 migration until it can be done systematically

**Trade-offs accepted:**
- Missing out on Svelte 5 runes benefits (fine-grained reactivity, better TypeScript support)
- Will require future migration work when Svelte 5 is fully adopted

**Implementation sequence:**
1. Remove all `$state`, `$derived`, `$effect` from existing components
2. Document "Svelte 4 syntax only" convention in project CLAUDE.md
3. Plan future Svelte 5 migration as separate project

### Alternative Approaches Considered

**Option B: Fully Migrate to Svelte 5 Runes**
- **Pros:** Access to new Svelte 5 features, better TypeScript support
- **Cons:** Requires rewriting all reactive logic in all components; higher risk; more time investment
- **When to use instead:** When team has capacity for full migration and wants Svelte 5 benefits immediately

**Option C: Convert Stores to Runes-Compatible Format**
- **Pros:** Keeps runes while fixing reactivity
- **Cons:** Complex, requires understanding Svelte 5 store interop; still mixing paradigms
- **When to use instead:** When committed to Svelte 5 but can't rewrite all components yet

**Rationale for recommendation:** Option A (standardize on Svelte 4) is lowest risk, requires minimal changes, and maintains consistency while deferring the complex migration decision.

---

### Implementation Details

**What to implement first:**
- ✅ Already done: Removed runes from `+page.svelte`
- Document Svelte syntax convention in `CLAUDE.md`
- Audit other components for similar issues

**Things to watch out for:**
- ⚠️ Don't introduce new `$state` declarations in future PRs
- ⚠️ Be careful with copy-pasting code from Svelte 5 examples
- ⚠️ Check that HMR works properly after removing runes

**Areas needing further investigation:**
- Survey all `.svelte` files for runes usage
- Determine if any components intentionally use Svelte 5 features
- Plan future Svelte 5 migration timeline

**Success criteria:**
- ✅ Dashboard displays all agents correctly (verified)
- ✅ No console errors related to reactivity
- ✅ Stats bar shows correct counts (33 active, 145 completed, 31 idle)

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Main dashboard component with reactivity issue
- `web/src/lib/stores/agents.ts` - Agent store (working correctly)
- `web/src/lib/components/agent-card/agent-card.svelte` - Agent card component (no issues found)

**Commands Run:**
```bash
# Verify API returns agents
curl -s http://127.0.0.1:3333/api/agents | jq '. | length'

# Check agent status distribution
curl -s http://127.0.0.1:3333/api/agents | jq '[.[].status] | group_by(.) | map({status: .[0], count: length})'

# Restart dev server with clean cache
pkill -f "vite dev" && cd web && rm -rf .svelte-kit node_modules/.vite && bun run dev --host
```

**External Documentation:**
- Svelte 5 Runes Documentation - https://svelte.dev/docs/svelte/what-are-runes
- Svelte 5 Migration Guide - Explains runes mode trigger

**Related Artifacts:**
- **Issue:** orch-go-fwpz - Dashboard shows 0 agents bug report

---

## Investigation History

**2025-12-22 21:38:** Investigation started
- Initial question: Why dashboard shows 0 agents despite API returning 209
- Context: Bug report from user, dashboard appeared broken

**2025-12-22 21:40:** Verified API and store working
- Confirmed API returns 209 agents
- Confirmed store contains 209 agents
- Ruled out backend/data layer issues

**2025-12-22 21:45:** Identified reactivity disconnect
- Component sees `$agents.length = 0`
- Store contains 209 agents
- Narrowed to component reactivity issue

**2025-12-22 21:50:** Discovered runes mode error
- Found "`$:` not allowed in runes mode" error
- Identified mixed Svelte 4/5 syntax as root cause

**2025-12-22 21:55:** Implemented and verified fix
- Removed all Svelte 5 runes from component
- Reverted to pure Svelte 4 syntax
- Verified dashboard shows all 209 agents correctly

**2025-12-22 21:56:** Investigation completed
- Final confidence: Very High (98%)
- Status: Complete
- Key outcome: Root cause identified and fixed; dashboard now functional
