<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Stats bar shows "50 ready" but there's no way to see what those issues are. Recommend expandable stats approach that reveals issue list inline.

**Evidence:** Current beads indicator in +page.svelte:370-391 only shows count with tooltip. bd ready returns full issue data with title, priority, type. Daemon indicator already shows capacity details.

**Knowledge:** Dashboard follows progressive disclosure pattern (Active/Recent/Archive). Expandable sections are proven pattern. Orchestrator needs quick triage visibility without switching context.

**Next:** Implement expandable beads section - click "50 ready" to reveal issue list inline with spawn/triage actions.

**Confidence:** High (85%) - Pattern matches existing CollapsibleSection usage; API already returns full issue data via bd ready --json.

---

# Investigation: Dashboard Queue Visibility Stats Bar

**Question:** Where should backlog visibility live in the dashboard? Stats bar shows "50 ready" but no way to see what those issues are.

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None - recommendation ready for implementation
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current Stats Bar Shows Only Aggregate Counts

**Evidence:** The beads indicator in `+page.svelte:370-391` displays:
- Ready issue count as large number
- Blocked count in red if > 0
- Tooltip with breakdown (ready/blocked/open counts)

No ability to see individual issues or their details.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:370-391`

**Significance:** This is the gap - orchestrator sees "50 ready" but must leave dashboard context to run `bd ready` to see the actual queue.

---

### Finding 2: Full Issue Data Available via API

**Evidence:** `bd ready --json` returns full issue objects including:
- id, title, description
- priority, issue_type
- labels (including `triage:ready` vs `triage:review`)
- created_at, updated_at

The serve.go already uses `bd stats --json` for aggregate counts. Adding `bd ready` data would require a new endpoint or enhanced `/api/beads`.

**Source:** `~/bin/bd ready --json` output; `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:1299-1349`

**Significance:** The data exists and is accessible. This is a UI/UX problem, not a data problem.

---

### Finding 3: Dashboard Already Has Progressive Disclosure Pattern

**Evidence:** The Swarm Map uses `CollapsibleSection` component for Active/Recent/Archive sections. Each section:
- Shows count in header badge
- Expands/collapses to show items
- Persists state in localStorage
- Has visual differentiation by variant

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte:576-626`

**Significance:** An expandable queue section would be consistent with existing UX patterns. Users already understand this interaction model.

---

### Finding 4: Daemon Indicator Already Shows Queue Relationship

**Evidence:** The daemon indicator (`+page.svelte:394-428`) shows:
- `ready_count` - issues ready to process
- Capacity metrics (used/max/free slots)
- Last poll/spawn timestamps

This creates conceptual link between "ready queue" and "daemon processing capacity."

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/daemon.ts`

**Significance:** The queue visibility feature should complement daemon visibility - showing WHAT's ready alongside HOW MUCH capacity exists.

---

## Synthesis

**Key Insights:**

1. **Context Switching is the Problem** - Currently, seeing queue details requires leaving dashboard to run CLI. This breaks orchestrator flow and causes context loss.

2. **Progressive Disclosure is the Pattern** - Dashboard already uses collapsible sections effectively. The same pattern can surface queue details on demand.

3. **Daemon + Queue = Complete Picture** - Showing ready queue alongside daemon capacity tells the full story: "Here's what's waiting AND here's our processing capacity."

**Answer to Investigation Question:**

Queue visibility should live in an **expandable stats section** - clicking the "50 ready" indicator reveals a collapsible queue list below the stats bar. This keeps the stats bar compact by default while enabling drill-down when needed.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
- Pattern matches proven UI (CollapsibleSection already works)
- Data is available via API
- Constraints (666px min width) are known and accommodated
- Only uncertainty is implementation details

**What's certain:**

- ✅ Full issue data available via bd ready --json
- ✅ CollapsibleSection pattern works well in current UI
- ✅ 666px constraint must be respected (known from kn entries)
- ✅ Daemon indicator shows capacity context

**What's uncertain:**

- ⚠️ Exact layout at 666px minimum width
- ⚠️ Whether to show all ready issues or just top 10
- ⚠️ Whether to include inline spawn actions or just link

**What would increase confidence to Very High:**

- Prototype the expandable section at 666px
- Test with real 50+ issue queue
- Get Dylan's feedback on information density

---

## Implementation Recommendations

### Recommended Approach ⭐

**Expandable Queue Section** - Click beads indicator to reveal collapsible queue list below stats bar.

**Why this approach:**
- Consistent with existing progressive disclosure pattern
- No new UI paradigms to learn
- Stats bar remains compact when collapsed
- Queue visible without context switching

**Trade-offs accepted:**
- Adds vertical space when expanded (acceptable - user controls this)
- Need new API endpoint for ready issues (straightforward to implement)

**Implementation sequence:**
1. Add `/api/beads/ready` endpoint that returns `bd ready --json`
2. Create ReadyQueue svelte store similar to beads store
3. Make beads indicator clickable → toggle expanded state
4. Add CollapsibleSection below stats bar for queue items
5. Each queue item shows: title, priority badge, type, labels
6. Optional: Add quick-action buttons (spawn, review)

### Alternative Approaches Considered

**Option B: Sidebar Panel**
- **Pros:** Persistent visibility, doesn't push content down
- **Cons:** Takes horizontal space (violates 666px constraint), requires layout restructure
- **When to use instead:** If dashboard moves to wider minimum width

**Option C: Separate /queue Route**
- **Pros:** Full-page experience, room for rich filtering
- **Cons:** Requires navigation, context switch, defeats purpose
- **When to use instead:** If queue management becomes primary focus

**Option D: Modal/Overlay**
- **Pros:** Full visibility without layout change
- **Cons:** Blocks underlying content, feels heavy for quick glance
- **When to use instead:** If queue needs editing capabilities

**Rationale for recommendation:** Expandable section is lowest friction, reuses existing patterns, and respects constraints. Orchestrator gets quick triage without losing agent visibility.

---

### Implementation Details

**What to implement first:**
- `/api/beads/ready` endpoint (blocks UI work)
- Beads store enhancement to fetch ready issues
- Clickable beads indicator with expanded state

**Things to watch out for:**
- ⚠️ 666px width constraint - test queue items at narrow width
- ⚠️ Performance with 50+ items - consider virtual scrolling or pagination
- ⚠️ Refresh rate - queue changes rarely, 60s poll is fine

**Areas needing further investigation:**
- Should triage:ready and triage:review be visually differentiated?
- Should clicking an issue spawn immediately or show details?
- Should there be filtering within the queue view?

**Success criteria:**
- ✅ Orchestrator can see ready queue without leaving dashboard
- ✅ Works at 666px width without horizontal scroll
- ✅ Collapsed state preserves current compact UX
- ✅ Expanded state shows enough info to make triage decisions

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/routes/+page.svelte` - Main dashboard with stats bar and beads indicator
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go` - API server with beads stats endpoint
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/beads.ts` - Beads store (stats only)
- `/Users/dylanconlin/Documents/personal/orch-go/web/src/lib/stores/daemon.ts` - Daemon store pattern to follow

**Commands Run:**
```bash
# Check ready issues format
~/bin/bd ready

# Check full JSON structure
~/bin/bd ready --json | head -100

# Check stats format
~/bin/bd stats --json
```

**Related Artifacts:**
- **Constraint:** Dashboard must be fully usable at 666px width (from kn entries)
- **Decision:** Dashboard uses progressive disclosure (Active/Recent/Archive sections)
- **Decision:** Dashboard beads stats use bd stats --json API call

---

## Investigation History

**2025-12-26 ~12:00:** Investigation started
- Initial question: Where should backlog visibility live in dashboard?
- Context: Stats bar shows "50 ready" but no way to see actual issues

**2025-12-26 ~12:30:** Context gathered
- Reviewed +page.svelte stats bar implementation
- Reviewed serve.go beads endpoint
- Reviewed bd ready JSON output
- Identified CollapsibleSection pattern

**2025-12-26 ~13:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend expandable queue section below stats bar
