## Summary (D.E.K.N.)

**Delta:** Dashboard theme toggle enhanced from simple button to dropdown menu with Light/Dark/System options.

**Evidence:** 8 Playwright tests pass, visual verification screenshots captured showing dropdown in both light and dark modes.

**Knowledge:** bits-ui 1.8.0 uses GroupHeading instead of Label, and RadioItem children snippet receives {checked} parameter.

**Next:** Close - implementation complete.

---

# Investigation: Dashboard Add Theme Selection System

**Question:** How to enhance the dashboard theme toggle to provide explicit Light/Dark/System selection options?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing theme infrastructure was solid

**Evidence:** Theme store (`theme.ts`) already supported light/dark/system values with localStorage persistence and system preference detection.

**Source:** `web/src/lib/stores/theme.ts:4-84`

**Significance:** No changes needed to the store - only the UI component needed updating.

---

### Finding 2: bits-ui 1.8.0 API differs from older versions

**Evidence:** Type errors revealed that `LabelProps` doesn't exist and `RadioIndicator` is not exported. The library uses `GroupHeading` and passes `{checked}` to children snippet.

**Source:** `node_modules/bits-ui/dist/bits/dropdown-menu/exports.d.ts`

**Significance:** Required creating compatible wrapper components that match the current bits-ui API.

---

### Finding 3: No existing dropdown-menu component in the codebase

**Evidence:** Only button, card, badge, and tooltip UI components existed in `web/src/lib/components/ui/`.

**Source:** Directory listing of `web/src/lib/components/ui/`

**Significance:** Created full dropdown-menu component set: content, item, group-heading, separator, radio-group, radio-item.

---

## Synthesis

**Key Insights:**

1. **bits-ui provides solid foundation** - The library handles accessibility, keyboard navigation, and styling out of the box. Just needed wrapper components with proper types.

2. **Minimal theme store changes** - The existing store design anticipated multiple theme options, making the UI update straightforward.

3. **Pattern consistency** - Following the tooltip component pattern (re-exporting bits-ui primitives with styled wrappers) kept the implementation consistent.

**Answer to Investigation Question:**

Enhanced theme toggle by creating a DropdownMenu component set and updating theme-toggle.svelte to use a radio group selection. The implementation respects the 666px dashboard width constraint and follows established UI patterns.

---

## Structured Uncertainty

**What's tested:**

- ✅ Theme selection changes localStorage (verified: Playwright test)
- ✅ Dark class applied/removed from HTML element (verified: Playwright test)
- ✅ Theme persists across page reload (verified: Playwright test)
- ✅ Dropdown closes on Escape key (verified: Playwright test)
- ✅ Visual appearance in both light and dark modes (verified: screenshots)

**What's untested:**

- ⚠️ System preference change while app is open (not tested in CI)
- ⚠️ Mobile touch interactions (desktop Chrome only in tests)

**What would change this:**

- bits-ui API changes in future versions could break wrapper components
- If system theme detection needs real-time updates, may need additional event listeners

---

## Implementation Recommendations

**Recommended Approach ⭐**

**Dropdown menu with radio group** - Provides clear, explicit theme selection with visual feedback.

**Why this approach:**
- Users can see all available options at once
- Radio indicator shows current selection
- Follows established UI patterns (shadcn-svelte style)

**Trade-offs accepted:**
- Slightly more complex than simple toggle
- Additional component files needed

**Implementation sequence:**
1. Create dropdown-menu UI components (content, item, radio-group, radio-item)
2. Update theme-toggle to use dropdown with radio selection
3. Update tests for new UI pattern

---

## References

**Files Examined:**
- `web/src/lib/stores/theme.ts` - Existing theme store
- `web/src/lib/components/theme-toggle/theme-toggle.svelte` - Original toggle
- `web/src/lib/components/ui/tooltip/` - Pattern reference for bits-ui wrappers

**Commands Run:**
```bash
# Type checking
npm run check

# Run tests
npx playwright test tests/dark-mode.spec.ts
```

---

## Investigation History

**2025-12-26 17:30:** Investigation started
- Initial question: How to add theme selection dropdown to dashboard

**2025-12-26 17:44:** Implementation complete
- Status: Complete
- Key outcome: Theme dropdown with Light/Dark/System options, 8 tests passing
