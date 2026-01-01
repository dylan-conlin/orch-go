---
Question: Why does form state clear when clicking other elements in Price Watch config editor?
Status: Complete
Confidence: medium
Resolution-Status: Not Reproducible
Tags: [glass, sveltekit, stimulus, form-state, price-watch]
---

# TLDR

Could not reproduce the reported issue. Form state (Config Name, Description) persists correctly when clicking quantity buttons, competitor toggles, material items, and filter dropdowns. The Rails/Stimulus-based form handles state correctly.

## What I Tried

1. **Typed in Config Name field** - "Test Config Name"
2. **Clicked quantity button (250)** - State preserved, 250 became selected
3. **Clicked competitor toggle (OSH Cut)** - State preserved, OSH Cut became deselected
4. **Clicked material item (ABS Black .125")** - State preserved, material selected (count updated to 1)
5. **Changed material category filter (Aluminum)** - State preserved, material list filtered
6. **Typed in material search ("steel")** - State preserved, materials filtered

## What I Observed

- Config Name field retained "Test Config Name" through all interactions
- All toggles and selections work correctly
- Form state is managed by Stimulus controller using JavaScript Sets (selectedMaterials, selectedGeometries, selectedCompetitors, selectedQuantities)
- No page re-renders observed during interactions
- The Turbo Stream subscription (`turbo_stream_from "collection_runs"`) only updates table cells, not the modal

## Test Performed

Manual testing via Glass browser automation:
1. Navigated to http://localhost:3338/collection_runs
2. Clicked "Create Config" button to open modal
3. Typed in Config Name field
4. Clicked various toggles and buttons
5. Verified form state was preserved after each interaction
6. Took screenshots to document state

## Root Cause Analysis

**The issue could not be reproduced.** Possible explanations:

1. **Already fixed** - The current codebase handles form state correctly
2. **Specific interaction pattern** - The original issue may require a specific sequence of actions not tested
3. **Timing-dependent** - Race condition with Turbo Stream updates that occurs intermittently
4. **Glass vs real user** - Glass event dispatching may differ from real user interaction in ways that avoid triggering the bug
5. **Different page/component** - The issue may be in a different form not investigated

## Code Analysis

The form uses:
- **Rails ERB template**: `_config_editor.html.erb` with Stimulus data attributes
- **Stimulus controller**: `config_editor_controller.js` managing state with JavaScript Sets
- **Input binding**: Native HTML inputs with Stimulus `data-action` attributes
- **State management**: Client-side only, no server round-trips on toggle clicks

The architecture appears sound for form state preservation:
- Input values stored in controller class properties
- Re-renders only affect specific DOM regions (material list, geometry list)
- Name/Description inputs are not touched during list re-renders

## Conclusion

Unable to reproduce. The form state management appears to work correctly. If the issue recurs, would recommend:
1. Recording the exact sequence of actions that causes the problem
2. Checking browser console for JavaScript errors
3. Testing with different browsers
4. Checking if Turbo Stream broadcasts are coinciding with form interactions

## Next Steps

- Close as not reproducible
- If issue recurs, create new investigation with specific reproduction steps
