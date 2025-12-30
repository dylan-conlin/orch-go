<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard artifact-viewer was showing "Artifact not found" error for agents without SYNTHESIS.md; now shows graceful "No synthesis produced" with close_reason fallback.

**Evidence:** Browser-verified: clicking on agent og-feat-orch-go-systematic-30dec (which has close_reason but no SYNTHESIS.md) now shows "No synthesis produced" and "Completion Summary" card instead of error.

**Knowledge:** The artifact-viewer's conditional order matters - check for content first, then check for "no tabs at all" case, then handle errors for individual tabs.

**Next:** Close - fix implemented and browser-verified.

---

# Investigation: Dashboard Shows Error Agents Complete

**Question:** Why does the dashboard show 'Artifact not found' error for agents that complete without SYNTHESIS.md, and how should it gracefully handle this case?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent og-debug-dashboard-shows-error-30dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Root cause in artifact-viewer.svelte conditional order

**Evidence:** The artifact-viewer.svelte file (lines 164-167) checked for `artifacts.get(activeTab)?.error` before checking for the "no available tabs" case. This meant when an agent had no synthesis file, the error condition triggered first, showing "Synthesis not available" with the raw error message "Artifact not found: open /path/SYNTHESIS.md: no such file".

**Source:** `web/src/lib/components/artifact-viewer/artifact-viewer.svelte:164-167`

**Significance:** The conditional order was incorrect - when no artifacts are available at all (empty `availableTabs`), we should show a graceful message, not an error.

---

### Finding 2: close_reason fallback existed but wasn't integrated

**Evidence:** The agent-detail-panel.svelte already had a fallback section (lines 371-377) that displayed close_reason when no synthesis existed:
```svelte
{#if $selectedAgent.status === 'completed' && !$selectedAgent.synthesis && $selectedAgent.close_reason}
  <div class="border-t p-4">
    <h3>Completion Summary</h3>
    <p>{$selectedAgent.close_reason}</p>
  </div>
{/if}
```
However, this appeared BELOW the ArtifactViewer which was already showing an error.

**Source:** `web/src/lib/components/agent-detail/agent-detail-panel.svelte:371-377`

**Significance:** The close_reason data was available but the UI didn't integrate it properly with the "no synthesis" case.

---

### Finding 3: Test case confirms fix

**Evidence:** Agent `og-feat-orch-go-systematic-30dec` (beads: orch-go-4i7f):
- Status: completed
- Has close_reason: "action-log plugin fix verified working"
- No SYNTHESIS.md file (verified via API returns error)
- After fix: Dashboard shows "No synthesis produced" + "Completion Summary" card with close_reason

**Source:** Browser screenshot of dashboard after clicking on agent

**Significance:** Confirms the fix works as intended for the exact use case described in the spawn context.

---

## Synthesis

**Key Insights:**

1. **Conditional order matters** - UI components should check for "nothing to show" before checking for errors on specific items.

2. **Pass data where it's needed** - Rather than having duplicate fallback sections, pass the close_reason to the component that handles the "no content" case.

3. **Light-tier completions are valid** - Some agents complete verification-only work without producing synthesis files. The UI should handle this gracefully.

**Answer to Investigation Question:**

The dashboard showed "Artifact not found" because the artifact-viewer.svelte checked for errors before checking if there were any artifacts at all. The fix reorders the conditionals and passes close_reason to artifact-viewer so it can display a graceful "Completion Summary" when no synthesis exists.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes (verified: `bun run build` succeeds)
- ✅ Agent without SYNTHESIS.md shows "No synthesis produced" (verified: browser screenshot)
- ✅ close_reason displays in styled card when synthesis missing (verified: browser screenshot)

**What's untested:**

- ⚠️ Agents with synthesis + close_reason (should show synthesis, not close_reason)
- ⚠️ Agents with neither synthesis nor close_reason (should show simple "no synthesis" message)

**What would change this:**

- Finding would be wrong if agents WITH synthesis also show the close_reason card (they shouldn't)

---

## Implementation Recommendations

**Purpose:** Document the implemented solution.

### Recommended Approach ⭐

**Show graceful fallback with close_reason** - When no synthesis exists, show "No synthesis produced" with an optional close_reason card.

**Why this approach:**
- Maintains UX consistency - no raw error messages
- Uses available data (close_reason) meaningfully
- Matches existing design patterns in the dashboard

**Trade-offs accepted:**
- Removed duplicate close_reason section from agent-detail-panel (DRY improvement)
- Added prop to artifact-viewer (minor API change)

**Implementation sequence:**
1. Reorder conditionals in artifact-viewer.svelte
2. Add closeReason prop to artifact-viewer
3. Pass close_reason from agent-detail-panel
4. Remove duplicate fallback section

### Implementation Details

**What was implemented:**
- `artifact-viewer.svelte`: Added `closeReason` prop, reordered conditionals, added styled card for close_reason
- `agent-detail-panel.svelte`: Pass closeReason prop to ArtifactViewer, removed duplicate fallback section

**Success criteria:**
- ✅ No "Artifact not found" errors shown to user
- ✅ Completion summary displayed when available
- ✅ Build passes

---

## References

**Files Modified:**
- `web/src/lib/components/artifact-viewer/artifact-viewer.svelte` - Added closeReason prop, reordered conditionals
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Pass closeReason, removed duplicate fallback

**Commands Run:**
```bash
# Build verification
bun run build

# API test
curl "http://localhost:3348/api/agents/artifact?workspace=og-feat-orch-go-systematic-30dec&type=synthesis"
```

**Related Artifacts:**
- **Issue:** orch-go-ytdp - Dashboard shows error for agents that complete without SYNTHESIS.md
- **Workspace:** `.orch/workspace/og-debug-dashboard-shows-error-30dec/`

---

## Investigation History

**2025-12-30 15:45:** Investigation started
- Initial question: Why does dashboard show error for agents without SYNTHESIS.md?
- Context: Agent orch-go-4i7f completed with close_reason but no synthesis

**2025-12-30 15:50:** Root cause identified
- Found conditional order issue in artifact-viewer.svelte
- Found existing close_reason fallback that wasn't integrated

**2025-12-30 16:00:** Fix implemented and browser-verified
- Status: Complete
- Key outcome: Dashboard now shows graceful message with close_reason fallback
