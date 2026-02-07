# Investigation: What Happened to the Dashboard Review Section?

**Date:** 2026-01-27  
**Status:** Complete  
**Investigator:** Worker Agent (orch-go-20965)

## Question

Dylan recalls a "Review" section in the dashboard ops view for completed agents awaiting orchestrator review, but it appears to be missing. What happened to it?

## Key Finding

**The "Needs Review" section DOES exist** in the current dashboard. There is confusion between two distinct sections:

1. **"Needs Review"** (exists) - Shows agents at Phase: Complete awaiting `orch complete`
2. **"Pending Reviews"** (removed) - Showed unreviewed synthesis recommendations from agents

## Timeline of Changes

### Dec 26, 2025: Pending Reviews Added
**Commit:** `bbe2e0bc`

Added PendingReviewsSection showing unreviewed synthesis recommendations from completed agents. This section parsed SYNTHESIS.md files and displayed individual action items for orchestrator review.

**Purpose:** Surface actionable recommendations from agent work  
**Location:** Separate section below Ready Queue  
**Data source:** `/api/pending-reviews` endpoint (scanned workspace SYNTHESIS files)

### Dec 30, 2025: Attention-First Redesign
**Commit:** `21ced841`

Major dashboard redesign consolidating attention items:
- Removed Ops/History mode toggle
- Moved pending reviews into unified NeedsAttention panel
- Simplified dashboard layout

### Jan 6, 2026: Needs Review Section Added
**Commit:** `530553ae`

Added **distinct** "Needs Review" section for agents in Phase: Complete:

```typescript
export const needsReviewAgents = derived(agents, ($agents) =>
  $agents.filter((a) => 
    a.status === 'active' && 
    a.phase?.toLowerCase() === 'complete'
  )
);
```

**Purpose:** Show agents awaiting orchestrator review via `orch complete`  
**Location:** Dedicated section with amber/yellow styling  
**Visibility:** Both operational and historical modes

This addressed the "3/3 slots filled" confusion where agents waiting for review consumed capacity markers but weren't truly active.

### Jan 5, 2026: Pending Reviews Removed
**Commit:** `808c9d52`

Removed PendingReviewsSection entirely:

```
fix: remove PendingReviewsSection and skip light-tier processing for performance

- Remove PendingReviewsSection from dashboard (unused)
- Add skipLightTierProcessing flag to pending-reviews API
- Add 7-day recency filter for workspace scanning
- Fixes 15+ second API timeouts with 700+ workspaces
```

**Reason:** Performance problems - 15+ second API timeouts with 700+ workspaces  
**Status:** Marked as "not actively used" (comments in code: web/src/routes/+page.svelte:9, 49, 81, 563)

## Current State (as of Jan 27, 2026)

### What EXISTS

**"Needs Review" Section** - Shows Phase: Complete agents

**Location in code:**
- `web/src/lib/stores/agents.ts` - `needsReviewAgents` derived store
- `web/src/routes/+page.svelte` lines 503-537 (operational mode)
- `web/src/routes/+page.svelte` lines 690-708 (historical mode)

**Appearance:**
- Icon: ✅
- Styling: Amber/yellow border (`border-amber-500/30`)
- Expanded by default (`needsReview: true` in sectionState)
- Help text: "Run `orch complete` to review"

**Derived store logic:**
```typescript
export const needsReviewAgents = derived(agents, ($agents) =>
  $agents.filter((a) => 
    a.status === 'active' && 
    a.phase?.toLowerCase() === 'complete'
  )
);
```

### What was REMOVED

**"Pending Reviews" Section** - Synthesis recommendations (removed Jan 5, 2026)

**Why removed:**
- Performance: 15+ second API timeouts
- Workspace scanning overhead (700+ workspaces)
- Marked as "not actively used"

**Components removed:**
- `web/src/lib/components/pending-reviews-section/` (component still exists in repo, not imported)
- `web/src/lib/stores/pending-reviews.ts` (store still exists, not imported)
- `/api/pending-reviews` endpoint (still exists in serve.go, not called from UI)

## Testing Evidence

### Code verification
```bash
# Needs Review section exists in operational mode
$ grep -n "Needs Review" web/src/routes/+page.svelte
503:        <!-- Needs Review (Phase: Complete, awaiting orch complete) -->
690:                        <!-- Needs Review Section (Phase: Complete, awaiting orch complete) -->

# needsReviewAgents store exists and is used
$ grep -n "needsReviewAgents" web/src/lib/stores/agents.ts
38:export const needsReviewAgents = derived(agents, ($agents) =>

# PendingReviewsSection is commented out
$ grep -n "PendingReviewsSection" web/src/routes/+page.svelte
9:  // PendingReviewsSection removed - not actively used
```

### Dashboard sections breakdown

**Operational Mode** (lines 458-549):
1. Up Next (priority queue)
2. Frontier (decidability state)
3. Questions (blocking questions)
4. **Active Agents** (truly running, excludes needs-review)
5. **Needs Review** (Phase: Complete, awaiting orch complete) ← THIS EXISTS
6. Needs Attention (errors, pending reviews, blocked)
7. Recent Wins (completed in 24h)
8. Ready Queue (collapsed by default)

**Historical Mode** (lines 550-763):
1. Up Next
2. Ready Queue
3. Swarm Map with collapsible sections:
   - **Active** (truly running, excludes needs-review)
   - **Needs Review** (Phase: Complete, awaiting orch complete) ← THIS EXISTS
   - Recent (idle/completed within 24h)
   - Archive (older than 24h)

## Conclusion

**Dylan's memory is correct** - there IS a "Review" section for completed agents awaiting orchestrator review.

**The section is called "Needs Review"** and shows agents at Phase: Complete. It exists in both operational and historical dashboard modes.

**The confusion likely stems from:**
1. Two similar-sounding sections: "Needs Review" (agents) vs "Pending Reviews" (synthesis items)
2. "Pending Reviews" was removed for performance Jan 5, 2026
3. "Needs Review" was added Jan 6, 2026 (one day later)
4. Code comments reference the removal but not the addition

## Recommendation

**No restoration needed** - the desired functionality exists as "Needs Review" section.

**Possible clarifications:**
1. Remove dead code: PendingReviewsSection component/store if truly unused
2. Update documentation: Clarify distinction between agent review vs synthesis review
3. Verify "Needs Review" is working as expected (visual verification)

## Related Files

- `web/src/routes/+page.svelte` - Main dashboard with Needs Review section
- `web/src/lib/stores/agents.ts` - needsReviewAgents derived store
- `web/src/lib/components/pending-reviews-section/` - Removed section (code remains)
- `web/src/lib/stores/pending-reviews.ts` - Removed store (code remains)

## Related Commits

- `bbe2e0bc` - Added Pending Reviews (Dec 26)
- `21ced841` - Attention-first redesign (Dec 30)
- `530553ae` - Added Needs Review (Jan 6)
- `808c9d52` - Removed Pending Reviews (Jan 5)

## Related Investigations

- `.kb/investigations/2025-12-30-inv-dashboard-attention-first-redesign-investigation.md` (referenced but not found)
- `.kb/investigations/2026-01-05-inv-pending-reviews-api-performance.md` (likely exists, not yet verified)
