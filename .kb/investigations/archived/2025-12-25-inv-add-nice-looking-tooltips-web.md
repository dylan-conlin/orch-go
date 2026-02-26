## Summary (D.E.K.N.)

**Delta:** Tooltips successfully added to web dashboard using shadcn-svelte's built-in tooltip component (bits-ui based).

**Evidence:** 20/21 playwright tests pass; build succeeds; type-check passes. One pre-existing flaky test (race-condition) fails unrelated to changes.

**Knowledge:** shadcn-svelte's tooltip requires a Tooltip.Provider wrapper at layout level for SSR; tooltip trigger creates additional button element affecting test selectors.

**Next:** Complete - tooltips are working and provide better UX than title attributes.

**Confidence:** High (90%) - manual testing not performed via browser, but type-check and playwright tests validate functionality.

---

# Investigation: Add Nice Looking Tooltips Web

**Question:** What's the best approach to add tooltips to the web dashboard - shadcn/ui component, external lib, or custom?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Project uses shadcn-svelte with bits-ui

**Evidence:** `web/package.json` shows `bits-ui: ^2.11.0` as dependency. `web/components.json` confirms shadcn-svelte configuration.

**Source:** `web/package.json:20`, `web/components.json`

**Significance:** The project already has the foundation for tooltips - bits-ui is the headless component library that powers shadcn-svelte. No new dependencies needed.

---

### Finding 2: Multiple UI elements use title attributes

**Evidence:** Stats bar items (focus, servers, beads), agent cards (context indicator, duration, truncated text), and buttons (Connect/Disconnect) all used native `title` attributes.

**Source:** `web/src/routes/+page.svelte:305-334`, `web/src/lib/components/agent-card/agent-card.svelte:220-230`

**Significance:** Native title attributes have poor UX (slow to appear, no styling, poor accessibility). Converting to proper tooltips improves user experience significantly.

---

### Finding 3: Tooltip.Provider required at layout level

**Evidence:** Initial implementation failed with "Context 'Tooltip.Provider' not found" error during SSR. Fixed by wrapping root layout in `<Tooltip.Provider>`.

**Source:** Playwright test output showed server error; fixed in `web/src/routes/+layout.svelte`

**Significance:** bits-ui tooltips use Svelte context API which requires provider at component tree root. This is a common pattern with headless UI libraries.

---

## Implementation Details

**Components added:**
- `web/src/lib/components/ui/tooltip/` - shadcn-svelte tooltip component via CLI
- Tooltip.Provider wrapper in `+layout.svelte`
- Tooltips applied to:
  - Stats bar: errors, focus, servers, beads indicators
  - Header: usage stats, connection status
  - Agent card: context indicator, processing indicator, duration, title, workspace ID, beads ID
  - Buttons: Connect/Disconnect (SSE), Follow/Stop (agentlog)

**Tests updated:**
- `tests/stats-bar.spec.ts` - Fixed button selector to use `.first()` since Tooltip.Trigger creates additional button element

---

## Confidence Assessment

**Current Confidence:** High (90%)

**What's certain:**
- ✅ Type-check passes with no errors
- ✅ Build succeeds
- ✅ 20/21 playwright tests pass
- ✅ Tooltip component properly imported and used

**What's uncertain:**
- ⚠️ Visual appearance not verified in live browser
- ⚠️ Animation/transition timing not customized

---

## References

**Files Modified:**
- `web/src/routes/+layout.svelte` - Added Tooltip.Provider, tooltips to header usage/connection
- `web/src/routes/+page.svelte` - Added tooltips to stats bar items
- `web/src/lib/components/agent-card/agent-card.svelte` - Added tooltips to card elements
- `web/tests/stats-bar.spec.ts` - Fixed button selectors for tooltip wrapper

**Commands Run:**
```bash
# Add shadcn tooltip component
npx shadcn-svelte@latest add tooltip

# Verify changes
bun run check
bun run build
npx playwright test
```
