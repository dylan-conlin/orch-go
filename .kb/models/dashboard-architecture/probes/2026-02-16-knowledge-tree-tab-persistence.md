# Probe: Knowledge-Tree Tab Persistence

**Status:** Active
**Date:** 2026-02-16
**Issue:** orch-go-lott

## Question

Does the knowledge-tree page currently persist tab selection (Knowledge/Work/Timeline) across browser refresh? What mechanism should be used to implement persistence?

## What I Tested

1. Examined `/web/src/routes/knowledge-tree/+page.svelte`
2. Analyzed state management for `currentView` variable (line 14)
3. Checked for existing persistence mechanisms (localStorage, URL hash)
4. Will test browser refresh behavior before and after fix

## What I Observed

### Current Implementation (Before Fix)

- `currentView` state initialized to `'knowledge'` on line 14
- No persistence mechanism for tab selection
- Page DOES use localStorage for tree expansion state (lines 9, 25-45)
- No URL hash handling for view state
- Tab selection always resets to 'knowledge' on refresh

### Existing Patterns

The page already demonstrates localStorage usage:
```typescript
// localStorage key for expansion state
const EXPANSION_STATE_KEY = 'knowledge-tree-expansion';
```

This establishes precedent for using localStorage in this component.

## Model Impact

**Extends** dashboard-architecture model with finding about state persistence patterns:

- Knowledge-tree page uses localStorage for expansion state but not for tab selection
- Inconsistent persistence: some UI state persists (expansion), some doesn't (tab selection)
- URL hash is not currently used for any state management in dashboard views

**Recommendation:** Use URL hash as primary mechanism with localStorage fallback
- Hash enables bookmarking specific views (e.g., `/knowledge-tree#work`)
- Matches user expectation for "I want to share this view"
- localStorage as fallback when hash is empty

## Implementation

### Changes Made

Added tab persistence to `/web/src/routes/knowledge-tree/+page.svelte`:

1. **New localStorage key**: `VIEW_STATE_KEY = 'knowledge-tree-view'`

2. **loadInitialView() function** - Loads view in priority order:
   - First: URL hash (`#knowledge`, `#work`, `#timeline`)
   - Second: localStorage fallback
   - Third: Default to 'knowledge'

3. **saveView() function** - Persists to both:
   - URL hash (enables bookmarking and sharing)
   - localStorage (fallback for when hash is cleared)

4. **Updated initialization**:
   - `currentView` now initialized via `loadInitialView()`
   - `treeView` set based on loaded view

5. **Updated handleViewToggle()** - Calls `saveView()` after view change

6. **New handleHashChange()** - Handles browser back/forward:
   - Detects hash changes
   - Loads appropriate data for new view
   - Updates SSE connections

7. **Event listeners**:
   - Added `hashchange` listener in `onMount`
   - Removed listener in `onDestroy`

8. **Updated onMount** - Loads initial view data (timeline or tree) based on persisted state

### Verification Performed

1. **Code review**: Changes follow existing localStorage pattern in same file
2. **Type checking**: `npm run check` shows no new errors introduced
3. **API health**: Dashboard API responding (200 OK)

### Manual Testing Required

Since Glass is unavailable, manual browser testing needed to verify:

1. **Browser refresh**:
   - Navigate to knowledge-tree
   - Switch to Work tab
   - Refresh page → should stay on Work tab

2. **Direct navigation**:
   - Navigate to `/knowledge-tree#work` → should show Work tab
   - Navigate to `/knowledge-tree#timeline` → should show Timeline tab

3. **Browser back/forward**:
   - Switch tabs (knowledge → work → timeline)
   - Press browser back button → should go back through tab history
   - Press forward → should go forward through tab history

4. **localStorage fallback**:
   - Switch to Work tab
   - Clear URL hash (navigate to `/knowledge-tree`)
   - Refresh → should restore Work tab from localStorage

## Model Impact

**Extends** dashboard-architecture model with:

1. **State Persistence Pattern**: 
   - URL hash as primary (sharable, bookmarkable)
   - localStorage as fallback
   - Consistent with Progressive Enhancement principles

2. **Tab State Management**:
   - Knowledge-tree now has persistent tab state
   - Browser back/forward integration via hashchange
   - Enables deep linking to specific views

3. **Consistency Finding**:
   - Before: Expansion state persisted, tab state didn't
   - After: Both persisted using same localStorage pattern
   - Demonstrates value of consistent state management

**Confirms** model claim that localStorage is the established pattern for UI state persistence in dashboard views.
