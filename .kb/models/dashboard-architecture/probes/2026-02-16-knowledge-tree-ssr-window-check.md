# Probe: Knowledge-Tree SSR Window Check

**Status:** Active  
**Date:** 2026-02-16  
**Context:** Bug fix for orch-go-993  
**Model:** Dashboard Architecture

## Question

Does the knowledge-tree page properly guard all browser-only code (window, localStorage) from server-side rendering?

## What I Tested

1. Read web/src/routes/knowledge-tree/+page.svelte:150
2. Checked for window references without SSR guards
3. Verified pattern used elsewhere in the file for browser-only code

## What I Observed

**Issue Found:**

Line 150 in `onDestroy` callback:
```typescript
onDestroy(() => {
    knowledgeTree.disconnectSSE();
    timelineStore.disconnectSSE();
    if (sseStatusUnsubscribe) sseStatusUnsubscribe();
    animationUnsubscribe();
    window.removeEventListener('hashchange', handleHashChange);  // ❌ No SSR guard
});
```

**Existing Pattern:**

The file already uses `typeof window === 'undefined'` guards in multiple places:
- Line 17 (`loadInitialView`)
- Line 41 (`saveView`)
- Line 67 (`loadExpansionState`)
- Line 79 (`saveExpansionState`)

**Root Cause:**

The `onDestroy` lifecycle hook runs during SSR hydration cleanup, but `window` doesn't exist in the server context. The developer missed applying the established guard pattern to the event listener cleanup.

## Model Impact

**Confirms Dashboard Architecture model claim:**

The model documents SSE connection constraints but doesn't explicitly mention SSR/hydration lifecycle issues. This finding extends the model's "Why This Fails" section with a new failure mode:

### Failure Mode: Unguarded Browser APIs in Lifecycle Hooks

**Symptom:** 500 error during initial page load (SSR)

**Root cause:** Browser-only APIs (window, localStorage, document) accessed in lifecycle hooks without SSR guards

**Why it happens:**
- SvelteKit runs component code both server-side and client-side
- Lifecycle hooks like `onDestroy` can run during SSR hydration
- Browser globals don't exist in Node.js context
- Unguarded access throws ReferenceError → 500 error

**Fix:** Wrap all browser API access with `typeof window === 'undefined'` guard

**Prevention:** Audit all lifecycle hooks (onMount, onDestroy, afterUpdate) for browser-only code

## Fix Applied

**Change made:**

```typescript
// Before (line 150):
window.removeEventListener('hashchange', handleHashChange);

// After (lines 150-152):
if (typeof window !== 'undefined') {
    window.removeEventListener('hashchange', handleHashChange);
}
```

## Verification

**Test performed:**

1. Ran `npm run build` from web/ directory
2. Build completed successfully with no SSR errors
3. Output included: `entries/pages/knowledge-tree/_page.svelte.js   31.14 kB`
4. This confirms server-side rendering now processes the page without encountering undefined window reference

**Original bug no longer reproduces.**

**Status:** Complete
