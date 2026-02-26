<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created InvestigationTab component showing workspace path, primary artifact path, and terminal command hints for file access.

**Evidence:** Build successful, component follows ActivityTab patterns, integrates with existing tab infrastructure in agent-detail-panel.svelte.

**Knowledge:** Tab components use Svelte 5 runes ($props, $state, $derived, $effect), agent data available via props from parent, workspace path derived from agent.project_dir + agent.id.

**Next:** Close - component ready for visual verification and integration testing.

**Promote to Decision:** recommend-no (tactical implementation following existing patterns)

---

# Investigation: Create InvestigationTab Component

**Question:** How to create an InvestigationTab component showing workspace path, primary artifact path, and terminal command hints?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing Tab Component Pattern

**Evidence:** ActivityTab.svelte (239 lines) demonstrates the pattern:
- Uses `interface Props { agent: Agent; }` with `$props()` for Svelte 5
- Derives data using `$derived()` for reactive computations
- Uses `$state()` for local UI state (copiedItem, filters)
- Includes clipboard helper with visual feedback

**Source:** `web/src/lib/components/agent-detail/activity-tab.svelte:1-134`

**Significance:** InvestigationTab follows this same pattern for consistency.

---

### Finding 2: Agent Data Structure

**Evidence:** Agent interface includes key fields for investigation display:
- `id` - workspace directory name
- `project_dir` - absolute path to project directory
- `primary_artifact` - optional path to investigation/synthesis file
- `status` - used for conditional display (abandoned badge)

**Source:** `web/src/lib/stores/agents.ts:27-59`

**Significance:** Can derive workspace path as `${agent.project_dir}/.orch/workspace/${agent.id}`

---

### Finding 3: Tab Infrastructure Already Exists

**Evidence:** agent-detail-panel.svelte has:
- Tab type enum including 'investigation' (line 12)
- Tab visibility logic for completed/abandoned agents (lines 24-26)
- Tab navigation with TabButton component (lines 303-307)
- No content implementation for Investigation tab (only placeholder)

**Source:** `web/src/lib/components/agent-detail/agent-detail-panel.svelte:12,24-26,303-307`

**Significance:** Only needed to create InvestigationTab component and wire it into existing tab content area.

---

## Synthesis

**Key Insights:**

1. **Pattern Consistency** - Following ActivityTab patterns ensures maintainability and consistent UX

2. **Workspace Path Construction** - Path is deterministic: `{project_dir}/.orch/workspace/{agent.id}/`

3. **Terminal Command Hints** - Most valuable commands: cd to workspace, ls files, cat SYNTHESIS.md, cat SPAWN_CONTEXT.md, cat primary_artifact

**Answer to Investigation Question:**

Created InvestigationTab.svelte with:
- Workspace path card (copyable)
- Primary artifact path card (copyable, conditional)
- Terminal commands section with 5 quick-copy commands
- Visual feedback on copy actions
- Abandoned badge when relevant

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (verified: npm run build completed without errors)
- ✅ Type checking passes (verified: npm run check shows no errors in new component)
- ✅ Component exports correctly (verified: index.ts updated, imports resolve)

**What's untested:**

- ⚠️ Visual appearance in browser (not verified - needs manual testing)
- ⚠️ Copy to clipboard functionality (not verified - needs browser testing)
- ⚠️ Tab switching behavior (not verified - needs browser testing)

**What would change this:**

- Finding would be incomplete if visual testing reveals layout issues
- Implementation would need adjustment if copy functionality fails in browser

---

## Implementation Recommendations

### Recommended Approach ⭐

**Follow ActivityTab Pattern** - Create component with same structure and conventions.

**Why this approach:**
- Consistent with existing codebase
- Uses established Svelte 5 patterns
- Reuses proven clipboard helper pattern

**Trade-offs accepted:**
- Some code duplication (clipboard helper) - could be extracted to shared utility later
- Inline styles via Tailwind - consistent with rest of dashboard

**Implementation sequence:**
1. Created investigation-tab.svelte with Props interface
2. Added workspace/artifact path cards with copy functionality
3. Added terminal command hints section
4. Exported from index.ts
5. Wired into agent-detail-panel.svelte

---

## References

**Files Examined:**
- `web/src/lib/components/agent-detail/activity-tab.svelte` - Pattern reference
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Integration point
- `web/src/lib/stores/agents.ts` - Agent data structure
- `web/src/lib/components/agent-detail/index.ts` - Exports

**Commands Run:**
```bash
# Type check
npm run check

# Build verification
npm run build
```

---

## Investigation History

**2026-01-06:** Investigation started
- Initial question: How to create InvestigationTab component
- Context: Part of orch-go-akhff epic for dashboard agent detail redesign

**2026-01-06:** Implementation completed
- Created investigation-tab.svelte (187 lines)
- Updated index.ts exports
- Integrated into agent-detail-panel.svelte
- Build verified successful
