<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Swarm dashboard has 6 test failures and 1 code bug - tests expect wrong counts, use wrong ports, and there's a keying bug causing stale data on sort.

**Evidence:** Playwright tests: 13 passed, 6 failed. Code review found agent grid uses index key instead of agent.id at `+page.svelte:299`.

**Knowledge:** Tests must match UI options (5 status values not 4), use playwright baseURL config, and avoid ambiguous selectors. React/Svelte keying with indexes causes stale data bugs.

**Next:** Created epic orch-go-mhec with 5 child issues. 4 marked triage:ready (clear fixes), 1 marked triage:review (Svelte 5 consistency - lower priority).

**Confidence:** High (90%) - all issues verified through test runs and code inspection.

---

# Investigation: Audit Swarm Dashboard Web UI

**Question:** What bugs and issues exist in the swarm dashboard web UI at http://localhost:5188?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** issue-creation agent
**Phase:** Complete
**Next Step:** None - issues created for implementation
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Status filter test expects 4 options but UI has 5

**Evidence:** 
```
Error: expect(locator).toHaveCount(expected) failed
Locator: getByTestId('status-filter').locator('option')
Expected: 4
Received: 5
```

**Source:** 
- `web/tests/filtering.spec.ts:22` - Test expects 4 options
- `web/src/routes/+page.svelte:253-259` - UI has 5 options (All, Active, Idle, Completed, Abandoned)

**Significance:** Test was written before 'idle' status was added. Simple fix: update expected count from 4 to 5.

---

### Finding 2: Duplicate 'Clear' button causes selector ambiguity

**Evidence:**
```
Error: strict mode violation: getByRole('button', { name: 'Clear' }) resolved to 2 elements:
1) Filter bar Clear button
2) Empty state 'Clear filters' button
```

**Source:** 
- `web/tests/filtering.spec.ts:89-97` - Test uses ambiguous selector
- `web/src/routes/+page.svelte:287` - Filter bar Clear button
- `web/src/routes/+page.svelte:305` - Empty state Clear filters button

**Significance:** When filters are active, both elements are visible. Test needs more specific selector or UI needs data-testid.

---

### Finding 3: Race-condition tests use hardcoded port instead of baseURL

**Evidence:**
```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:5188/
```

**Source:**
- `web/tests/race-condition.spec.ts:15,43,58,73` - All use `http://localhost:5188`
- `web/playwright.config.ts:11` - Config has `baseURL: 'http://localhost:4173'`

**Significance:** All 4 race-condition tests fail. Other tests use relative paths (`page.goto('/')`) which work correctly.

---

### Finding 4: Agent grid uses index as key instead of agent.id

**Evidence:**
```svelte
{#each filteredAgents as agent, idx (idx)}
  <AgentCard {agent} />
{/each}
```

**Source:** `web/src/routes/+page.svelte:299`

**Significance:** This is a classic keying anti-pattern. When agents are sorted differently, Svelte reuses DOM elements with stale data because the keys (0, 1, 2...) remain the same while the underlying data changes positions.

---

### Finding 5: Svelte 5 runes inconsistently mixed with Svelte 4

**Evidence:**
- `+layout.svelte`: Uses `$derived.by`, `$props` (Svelte 5)
- `theme-toggle.svelte`: Uses `$derived` (Svelte 5)
- `+page.svelte`: Uses `$:` reactive statements (Svelte 4)
- Stores: Use Svelte 4 writable/derived

**Source:** Code review of all .svelte files and stores

**Significance:** Lower priority - current state works because each component is internally consistent. But inconsistency could cause issues if patterns are copied between files (see prior 0-agents bug from mixing).

---

## Synthesis

**Key Insights:**

1. **Test-UI drift** - Tests were written for an earlier version of the UI. Status filter added 'idle' option but test wasn't updated.

2. **Port configuration mismatch** - Race-condition tests hardcode a development port (5188) instead of using playwright's configured baseURL (4173).

3. **DOM recycling bug** - Using array index as key in each blocks causes stale data when sorting/filtering changes element order.

**Answer to Investigation Question:**

The dashboard has 6 test failures and 1 code bug that needs fixing:
- 2 filtering test failures (status count mismatch, Clear button ambiguity)
- 4 race-condition test failures (hardcoded wrong port)
- 1 code bug (index-based keying causes stale data)

All issues are straightforward to fix with clear solutions documented in the child issues.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All issues were verified through running tests and code inspection. The fixes are straightforward.

**What's certain:**

- ✅ 6 tests fail with specific, verifiable errors
- ✅ Status filter has 5 options (verified in code)
- ✅ Race-condition tests use wrong port (verified in code vs config)
- ✅ Agent grid uses idx as key (verified in code)

**What's uncertain:**

- ⚠️ Whether the keying bug has caused visible issues for users (no reports found)
- ⚠️ Whether Svelte 5 mixing will cause future issues (works currently)

**What would increase confidence to 95%:**

- Manual testing of the keying bug (sort agents, verify data staleness)
- User feedback on any UI glitches observed

---

## Implementation Recommendations

### Recommended Approach

**Fix all 4 triage:ready issues first** - They're straightforward and will get tests passing.

**Implementation sequence:**
1. Fix race-condition tests (use `/` instead of hardcoded URL) - orch-go-mhec.3
2. Fix status filter test (expect 5 options) - orch-go-mhec.1
3. Fix Clear button selector (use filter-bar scoped selector) - orch-go-mhec.2
4. Fix agent grid keying (use agent.session_id || agent.id) - orch-go-mhec.4

**Why this order:** Tests first (quick wins), then code bug (higher impact).

---

## References

**Files Examined:**
- `web/src/routes/+page.svelte` - Main dashboard component
- `web/src/lib/stores/agents.ts` - Agent store
- `web/src/lib/stores/agentlog.ts` - Agentlog store
- `web/src/lib/components/agent-card/agent-card.svelte` - Agent card component
- `web/src/lib/components/theme-toggle/theme-toggle.svelte` - Theme toggle
- `web/src/lib/stores/theme.ts` - Theme store
- `web/src/routes/+layout.svelte` - Layout component
- `web/tests/filtering.spec.ts` - Filtering tests
- `web/tests/dark-mode.spec.ts` - Dark mode tests
- `web/tests/stats-bar.spec.ts` - Stats bar tests
- `web/tests/race-condition.spec.ts` - Race condition tests
- `web/playwright.config.ts` - Playwright configuration

**Commands Run:**
```bash
# Run playwright tests
cd web && npx playwright test

# Verify API running
curl -s http://127.0.0.1:3348/api/agents | head -20

# Verify dashboard running
curl -s http://localhost:5188 | head -100
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-debug-dashboard-shows-0-agents-despite-api-returning-209.md` - Prior Svelte 5 runes issue
- **Epic:** `orch-go-mhec` - Swarm Dashboard Bug Fixes (Audit Dec 2025)

---

## Investigation History

**2025-12-23 21:27:** Investigation started
- Initial question: What bugs exist in swarm dashboard?
- Context: Spawned from orch-go-4k8n to audit dashboard

**2025-12-23 21:30:** Ran playwright tests
- 13 passed, 6 failed
- Identified 2 filtering failures, 4 race-condition failures

**2025-12-23 21:35:** Code review
- Found keying bug at +page.svelte:299
- Found Svelte 5/4 mixing inconsistency

**2025-12-23 21:45:** Investigation completed
- Created epic orch-go-mhec with 5 child issues
- Final confidence: High (90%)
- Status: Complete
- Key outcome: 6 test failures + 1 code bug documented with clear fix paths
