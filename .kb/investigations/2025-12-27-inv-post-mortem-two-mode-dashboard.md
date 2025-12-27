---
linked_issues:
  - orch-go-o22b
---
## Summary (D.E.K.N.)

**Delta:** Agent shipped broken UI feature because validation was build-only; SSR hydration bug made toggle non-functional despite passing build.

**Evidence:** Original code checked `typeof window !== 'undefined'` at module load time (runs during SSR), never re-ran during hydration; toggle buttons rendered but clicking didn't change view content.

**Knowledge:** SvelteKit stores that use browser APIs must defer initialization to `onMount()` via an explicit `init()` function; build/typecheck success does not validate runtime behavior.

**Next:** Add browser-testing requirement to UI feature completion gates; consider adding MCP Playwright for automated smoke tests.

---

# Investigation: Post-Mortem Two-Mode Dashboard

**Question:** Why did the two-mode dashboard toggle fail despite the agent claiming complete and having commits?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Post-mortem investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Agent did not perform browser testing

**Evidence:** 
- SYNTHESIS.md "Tests Run" section shows only `bun run build` and `bun run check`
- "What's untested" section explicitly states: "Visual fit at 666px constraint (not browser-tested, but uses existing responsive grid)"
- No screenshots, browser console output, or visual verification evidence in workspace

**Source:** 
- `.orch/workspace/og-work-dashboard-two-modes-27dec/SYNTHESIS.md` lines 41-48

**Significance:** The agent's validation was limited to TypeScript compilation and Vite build, neither of which can detect runtime behavior issues like SSR hydration bugs.

---

### Finding 2: SSR hydration bug in store initialization

**Evidence:** 
Original code from commit `c3743202`:
```javascript
// Load initial value from localStorage AT MODULE LOAD TIME
let initialValue: DashboardMode = 'operational';
if (typeof window !== 'undefined') {  // False during SSR
    const stored = localStorage.getItem(STORAGE_KEY);
    // ...
}
const { subscribe, set, update } = writable<DashboardMode>(initialValue);
```

The fix (uncommitted):
```javascript
import { browser } from '$app/environment';

// Create store with default, then initialize in onMount
const store = writable<DashboardMode>('operational');

// Add init() function
init: () => {
    if (browser) {
        const stored = getStoredMode();
        store.set(stored);
    }
}
```

**Source:** 
- `git show c3743202 -- web/src/lib/stores/dashboard-mode.ts`
- `git diff -- web/src/lib/stores/dashboard-mode.ts`

**Significance:** In SvelteKit, module code runs during SSR where `window` is undefined. The store instance persists through hydration, so the localStorage check never executes in the browser. This is a common SvelteKit pitfall that only manifests at runtime.

---

### Finding 3: Beads issue confirms user-observed failure

**Evidence:** 
Issue `orch-go-8uoh` filed after agent completion:
> "The ⚡ Ops / 📦 History buttons are rendered but clicking them doesn't change the view content. dashboardMode store exists and buttons call dashboardMode.set() but conditional rendering isn't switching."

**Source:** 
- `bd show orch-go-8uoh`

**Significance:** The failure mode was visible to end users immediately - buttons appeared but didn't work. This was not an edge case or race condition; it was a complete functional failure that any browser test would have caught.

---

### Finding 4: Skill-specified validation was insufficient

**Evidence:** 
The agent was spawned with `design-session` skill which does not require browser testing. The SPAWN_CONTEXT.md stated validation requirements as:
- Update Status field when done
- Create SYNTHESIS.md
- `bd comment` for progress updates

No UI-specific validation gates were specified.

**Source:** 
- `.orch/workspace/og-work-dashboard-two-modes-27dec/SPAWN_CONTEXT.md` lines 175-205

**Significance:** The spawn context did not include UI-specific validation requirements. For UI features, "build passes" is necessary but not sufficient validation.

---

### Finding 5: Agent acknowledged untested areas but proceeded anyway

**Evidence:** 
From investigation file:
> "**What's untested:**
> - ⚠️ Visual fit at 666px constraint (not browser-tested, but uses existing responsive grid)
> - ⚠️ SSE updates work correctly in Operational mode (should work - same stores)"

The agent documented uncertainty but marked the task complete without escalating.

**Source:** 
- `.kb/investigations/2025-12-27-inv-dashboard-two-modes-operational-default.md` lines 91-97

**Significance:** The agent had enough self-awareness to document untested areas but lacked a gate preventing completion without testing critical functionality.

---

## Synthesis

**Key Insights:**

1. **Build verification ≠ Behavior verification** - TypeScript compilation and Vite build success verify syntactic correctness and dependency resolution, but cannot validate runtime behavior, especially SSR-related issues.

2. **SSR is a common SvelteKit pitfall** - Stores that access `window`, `localStorage`, or other browser APIs must use explicit initialization patterns (e.g., `onMount`/`init()` with `browser` check) rather than module-level code.

3. **Missing validation gate for UI features** - The orchestration system's completion gates don't distinguish between code-only and UI work. UI features need browser verification as a hard requirement.

4. **Self-documented uncertainty didn't prevent completion** - The agent correctly identified "not browser-tested" as a risk but had no mechanism to force validation before marking complete.

**Answer to Investigation Question:**

The toggle failed because of an SSR hydration bug: the store read from localStorage at module load time (during SSR where window is undefined), and never re-read during browser hydration. The agent validated only with build/typecheck (which passed) but never loaded the page in a browser. The spawn context and skill didn't require browser testing for UI features, so the agent's completion was technically valid per existing gates but functionally broken.

---

## Structured Uncertainty

**What's tested:**

- ✅ Original commit contained SSR bug (verified: `git show c3743202` shows `typeof window !== 'undefined'` at module level)
- ✅ Fix exists but uncommitted (verified: `git diff` shows `init()` function added)
- ✅ Agent only ran build/typecheck (verified: SYNTHESIS.md "Tests Run" section)
- ✅ User observed broken toggle (verified: beads issue `orch-go-8uoh`)

**What's untested:**

- ⚠️ Whether fix actually works (not browser-tested in this investigation)
- ⚠️ Whether similar SSR bugs exist in other stores
- ⚠️ Whether automated Playwright tests would have caught this

**What would change this:**

- If the toggle actually worked in browser despite the apparent bug, root cause would be different
- If there was hidden browser testing evidence not in workspace, agent validation would be less at fault

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add browser-testing requirement to UI feature completion gates** - UI features must include browser verification evidence (screenshot, console output, or Playwright test results) before marking complete.

**Why this approach:**
- Directly addresses root cause: agent didn't test in browser
- Low overhead: one additional step for UI work
- High signal: would have caught this bug immediately

**Trade-offs accepted:**
- Adds time to UI feature completion
- Requires agents to have browser access (MCP or manual)

**Implementation sequence:**
1. Update `feature-impl` skill to detect UI work and require browser evidence
2. Add `--mcp playwright` recommendation for UI features in spawn context
3. Add "Browser Verification" section to SYNTHESIS.md template for UI features

### Alternative Approaches Considered

**Option B: Add SSR lint rules**
- **Pros:** Prevents class of bugs at build time
- **Cons:** Only catches known patterns; doesn't validate actual behavior
- **When to use instead:** As complementary defense, not replacement

**Option C: Mandatory Playwright for all UI features**
- **Pros:** Automated, reproducible
- **Cons:** High overhead for small changes; requires infrastructure
- **When to use instead:** For complex UI features or when reliability is critical

**Rationale for recommendation:** Browser testing is the minimal sufficient validation. It's less overhead than full Playwright automation but catches 90%+ of visible failures.

---

### Implementation Details

**What to implement first:**
- Document "UI features require browser verification" as constraint
- Add to `feature-impl` skill validation checklist
- Include in spawn context for dashboard/UI work

**Things to watch out for:**
- ⚠️ SvelteKit stores using `localStorage`, `window`, or other browser APIs need `onMount` initialization
- ⚠️ `typeof window !== 'undefined'` at module level runs during SSR, not hydration
- ⚠️ Use `browser` from `$app/environment` instead of manual checks

**Areas needing further investigation:**
- Audit other stores in `web/src/lib/stores/` for similar SSR bugs
- Evaluate Playwright MCP for automated smoke tests

**Success criteria:**
- ✅ No UI feature completes without browser verification evidence
- ✅ Similar SSR bugs caught before merge
- ✅ Clear documentation of SvelteKit SSR patterns

---

## References

**Files Examined:**
- `.orch/workspace/og-work-dashboard-two-modes-27dec/SPAWN_CONTEXT.md` - Original task and skill guidance
- `.orch/workspace/og-work-dashboard-two-modes-27dec/SYNTHESIS.md` - Agent's claimed work and validation
- `.kb/investigations/2025-12-27-inv-dashboard-two-modes-operational-default.md` - Agent's investigation file
- `web/src/lib/stores/dashboard-mode.ts` - Store implementation with bug
- `web/src/routes/+page.svelte` - Page using the store

**Commands Run:**
```bash
# Check git history
git log --oneline -20
git show c3743202 --stat
git show c3743202 -- web/src/lib/stores/dashboard-mode.ts

# Check current state vs commit
git diff -- web/src/lib/stores/dashboard-mode.ts
git status

# Check beads issues
bd show orch-go-jb0j
bd show orch-go-8uoh
bd list --all | grep -i dashboard
```

**Related Artifacts:**
- **Issue:** orch-go-jb0j - Original dashboard two-modes task (closed)
- **Issue:** orch-go-8uoh - Bug: toggle doesn't switch views (in_progress)
- **Workspace:** `.orch/workspace/og-work-dashboard-two-modes-27dec/` - Agent workspace

---

## Investigation History

**2025-12-27 11:44:** Investigation started
- Initial question: Why did the toggle fail despite agent claiming complete?
- Context: Spawned as post-mortem after user observed broken toggle

**2025-12-27 12:XX:** Found root cause
- SSR hydration bug: store initialized at module load, not during hydration
- Agent validated build only, no browser testing
- Fix exists but uncommitted

**2025-12-27 12:XX:** Investigation completed
- Status: Complete
- Key outcome: UI features need browser verification gate; SSR is a common SvelteKit pitfall

---

## Self-Review

- [x] Real test performed (examined git history, diffs, workspace artifacts)
- [x] Conclusion from evidence (traced bug from symptom through code to root cause)
- [x] Question answered (why toggle failed despite "complete" status)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED
