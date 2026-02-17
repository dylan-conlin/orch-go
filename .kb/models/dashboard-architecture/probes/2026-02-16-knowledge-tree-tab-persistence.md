# Probe: Knowledge-Tree Tab Persistence

**Status:** Complete
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
const EXPANSION_STATE_KEY = 'knowledge-tree-expansion'
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

## Code Review Verification (2026-02-17)

### Implementation Analysis

Reviewed implementation in commits `068b8915` and `c86c2025`:

**Design Decisions:**

1. **State mechanism:** URL hash primary + localStorage fallback
   - Rationale: Hash enables bookmarking, localStorage caches preference
   - Trade-off: Slightly more complex, but better UX
2. **Load priority:** hash → localStorage → default

   - Rationale: URL is source of truth (web standards)
   - Correct: Yes ✅

3. **Browser navigation:** hashchange listener with full view switching

   - Rationale: Expected browser back/forward behavior
   - Implementation: Disconnects old SSE, loads new data, reconnects
   - Correct: Yes ✅

4. **SSR safety:** typeof window checks throughout

   - Rationale: SvelteKit SSR requires window guards
   - Coverage: loadInitialView, saveView, event listeners, cleanup
   - Correct: Yes ✅

5. **Legacy fallback:** 'work' → 'knowledge' redirect
   - Rationale: Work tab removed, but preserve old bookmarks/localStorage
   - Correct: Yes ✅

**Implementation Quality:**

- ✅ Correct load order (hash → localStorage → default)
- ✅ Saves to both storages on change
- ✅ Browser back/forward integration
- ✅ SSR safe with window guards
- ✅ Error handling (try/catch localStorage)
- ✅ Proper cleanup (removeEventListener in onDestroy)
- ✅ Follows existing patterns (EXPANSION_STATE_KEY precedent)

**Verification Status:**

- Code review: ✅ Passed
- Manual browser test: ⚠️ Requires Dylan verification (Glass unavailable)

### Browser Verification Required

Dylan should manually verify:

1. **Refresh persistence:**

   - Navigate to /knowledge-tree
   - Switch to Timeline tab
   - Refresh → should stay on Timeline ✓

2. **URL bookmarking:**

   - Navigate to /knowledge-tree#timeline
   - Should load directly to Timeline tab ✓

3. **Browser back/forward:**

   - Switch tabs (knowledge → timeline)
   - Press back button → should return to knowledge ✓
   - Press forward → should return to timeline ✓

4. **Legacy redirect:**
   - Navigate to /knowledge-tree#work
   - Should show knowledge view ✓

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

4. **Implementation Pattern**:
   - Load priority: URL hash → localStorage → default
   - Save on change: Both hash and localStorage
   - SSR safety: typeof window guards required throughout
   - Event cleanup: removeEventListener in onDestroy

**Confirms** model claim that localStorage is the established pattern for UI state persistence in dashboard views.

**Status:** Complete (pending manual browser verification)
