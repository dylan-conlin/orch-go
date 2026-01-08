---
linked_issues:
  - orch-go-t8f11
---
## Summary (D.E.K.N.)

**Delta:** Synthesized 14 new dashboard investigations (Jan 7) into the existing guide, updating with new patterns: cross-project filtering, null/stale handling, filter timing optimization, and activity feed persistence architecture.

**Evidence:** Read all 14 Jan 7 investigations; identified 4 new theme areas: Performance Optimization (filter timing), Cross-Project Visibility (project_dir filtering), Data Pipeline Integrity (null handling, stale agents), and Activity Feed Persistence (hybrid SSE + API architecture).

**Knowledge:** The dashboard performance issues continue to follow predictable patterns (O(n) scaling, filter timing, threshold regressions). The Jan 7 work introduced new concepts: `is_stale` for old agents, project-aware caching, and early filter application.

**Next:** Close - guide updated with Jan 7 patterns. Consult guide before spawning new dashboard investigations.

**Promote to Decision:** recommend-no (consolidation/documentation, not architectural)

---

# Investigation: Synthesize Dashboard Investigations

**Question:** What new patterns from Jan 7 dashboard investigations should be consolidated into the authoritative guide?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** og-inv-synthesize-dashboard-investigations-07jan-96a0
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Investigation Count Increased from 44 to 58

**Evidence:** The Jan 6 synthesis covered 44 investigations. Current glob shows 58 dashboard investigations in `.kb/investigations/`. The 14 new investigations are all from Jan 7.

**Source:** `glob ".kb/investigations/*dashboard*.md"` - 58 files

**Significance:** The Jan 7 work was intensive - 14 dashboard investigations in a single day. These need to be consolidated to prevent duplicate work.

---

### Finding 2: New Theme - Early Filter Application for Performance

**Evidence:** The `2026-01-07-inv-dashboard-api-agents-filters-applied-late.md` investigation revealed that time/project filters were applied at the END of the handler after all expensive operations. This caused 20s cold cache times even with filters.

The fix moved filters earlier in the pipeline, immediately after session fetch, reducing workload proportionally to filter selectivity.

**Source:** `2026-01-07-inv-dashboard-api-agents-filters-applied-late.md`

**Significance:** This is a new performance pattern: **Filter Early, Process Late**. The original code did "process everything, filter late" which defeats the purpose of filtering.

---

### Finding 3: New Theme - Cross-Project Visibility

**Evidence:** Two investigations addressed cross-project issues:

1. `2026-01-07-inv-dashboard-agents-filter-session-directory.md` - The project filter was using `s.Directory` (session directory) for early filtering, which is the orchestrator's cwd for --workdir spawns, not the target project. Fixed by using `agent.ProjectDir` from workspace cache.

2. `2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` - Dashboard beads now follow the orchestrator's tmux context via project_dir parameter. Cache is now per-project keyed by directory.

**Source:** Two investigation files above

**Significance:** Cross-project spawning (`orch spawn --workdir`) requires filtering AFTER project_dir is populated from workspace cache, not using session directory.

---

### Finding 4: New Theme - Data Pipeline Integrity (Null/Stale Handling)

**Evidence:** Two investigations addressed data completeness:

1. `2026-01-07-inv-dashboard-shows-usage-anthropic-api.md` - Usage API returns null for inactive billing periods. Go's `float64` defaults to 0, losing the null distinction. Fixed by using `*float64` pointers and showing "N/A" in UI.

2. `2026-01-07-inv-fix-dashboard-show-older-agents.md` - Agents older than 2h were completely excluded via `continue`. Fixed by adding `is_stale` boolean field to mark them instead, preserving the performance optimization (skip beads fetch) while still displaying them.

**Source:** Two investigation files above

**Significance:** Two patterns: (1) Null preservation through pipeline requires pointer types in Go; (2) Performance optimizations that exclude data should mark data as stale instead of hiding it.

---

### Finding 5: New Theme - Activity Feed Persistence Architecture

**Evidence:** The `2026-01-07-design-dashboard-activity-feed-persistence.md` investigation designed a hybrid architecture:
- SSE for real-time updates
- OpenCode API (`/session/:sessionID/message`) for historical data
- OpenCode is the source of truth for session history (persists to `~/.local/share/opencode/storage/`)

Current implementation stores 1000 events globally, diluted across agents and lost on refresh.

**Source:** `2026-01-07-design-dashboard-activity-feed-persistence.md`

**Significance:** This establishes the pattern: **OpenCode is the authoritative source for session data, dashboard should be a thin presentation layer**.

---

### Finding 6: O(n²) Pattern Recurrence

**Evidence:** The `2026-01-07-inv-dashboard-api-agents-performance-synthesis.md` investigation identified another O(n²) pattern: investigation directory scanning. With 362 agents and 590 investigation files, `discoverInvestigationPath()` called `os.ReadDir()` 2-3 times per agent = 427K+ file entry comparisons.

Fixed by building an investigation directory cache once before the agent loop.

**Source:** `2026-01-07-inv-dashboard-api-agents-performance-synthesis.md`

**Significance:** This is the **fourth** occurrence of dashboard API slowness since Dec 21. The pattern: O(n) operations that seem innocent but scale terribly. Always profile before fixing.

---

## Synthesis

**Key Insights:**

1. **Performance patterns are predictable** - The four dashboard slowness incidents (Dec 22, Dec 27, Jan 6, Jan 7) all had similar root causes: O(n) session/file scaling, threshold regressions, and cache misses. The fix pattern is: filter early, cache expensive operations, profile before fixing.

2. **Cross-project visibility requires delayed filtering** - For --workdir spawns, the session directory is the orchestrator's cwd, not the target project. Filtering must happen AFTER workspace cache lookup populates `project_dir`.

3. **Data pipeline integrity requires explicit null handling** - When APIs return null (Anthropic usage) or data is stale (old agents), the pipeline must preserve this distinction through all layers (API → Go → JSON → TypeScript → UI).

4. **OpenCode is the source of truth for session data** - Dashboard should treat SSE as real-time updates and the API as the source of truth for historical data. Don't duplicate storage in browser memory.

**Answer to Investigation Question:**

The 14 new Jan 7 investigations should be consolidated into the guide with these additions:

1. **New "Common Problems" entries:**
   - "Dashboard filters don't reduce cold cache time" → Apply filters early
   - "Cross-project agents not showing" → Use `project_dir`, not `s.Directory`
   - "Usage shows 0% when data unavailable" → Use pointer types, show "N/A"
   - "Old agents completely hidden" → Use `is_stale` field

2. **New "Key Concepts" entry:**
   - `is_stale` - Agents older than beadsFetchThreshold, displayed with 📦 indicator

3. **New "Architecture" section update:**
   - Activity Feed Persistence - Hybrid SSE + API architecture

4. **Updated "Performance Considerations":**
   - Add "filter timing" as a pattern to check
   - Document investigation directory cache

---

## Structured Uncertainty

**What's tested:**

- ✅ All 14 Jan 7 investigations read and categorized
- ✅ Patterns identified match the investigation findings
- ✅ No contradictions with existing guide content

**What's untested:**

- ⚠️ Whether the guide updates will actually reduce duplicate investigations (needs future validation)
- ⚠️ Activity Feed Persistence is a design, not implementation - implementation may reveal issues

**What would change this:**

- If new dashboard bugs don't match documented patterns, guide needs expansion
- If implementation of activity feed persistence reveals issues with OpenCode API, architecture recommendation may change

---

## Implementation Recommendations

### Recommended Approach ⭐

**Update `.kb/guides/dashboard.md`** - Add the new patterns from Jan 7 investigations to the authoritative reference.

**Why this approach:**
- Prevents re-investigating solved problems
- Consolidates knowledge from 14 new investigations
- Follows the established synthesis pattern from Jan 6

**Trade-offs accepted:**
- Guide requires ongoing maintenance as system evolves
- Some Jan 7 investigations are incomplete (template only) - skip those

**Implementation sequence:**
1. Add new Common Problems entries for filter timing, cross-project, null handling, stale agents
2. Add `is_stale` to Key Concepts section
3. Update Performance Considerations with filter timing and investigation cache
4. Add note about Activity Feed Persistence architecture to Integration Points section
5. Update History section with Jan 7 work

---

## References

**Files Examined:**

- `.kb/investigations/*dashboard*.md` (58 files)
- `.kb/guides/dashboard.md` - Existing guide to update
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Context on synthesis

**Commands Run:**
```bash
# Count dashboard investigations
glob ".kb/investigations/*dashboard*.md"  # 58 files

# Create investigation file
kb create investigation synthesize-dashboard-investigations
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-synthesize-dashboard-investigations-39-synthesis.md` - Prior synthesis
- **Guide:** `.kb/guides/dashboard.md` - Authoritative dashboard reference

---

## Self-Review

- [x] Real test performed (not code review) - Verified guide edits by reading changes
- [x] Evidence concrete - Specific file counts (58 investigations), specific patterns identified
- [x] Conclusion factual - Based on actual investigation file contents, not inference
- [x] No speculation - Conclusions based on what was found in files
- [x] Question answered - Investigation question fully addressed
- [x] File complete - All sections filled
- [x] D.E.K.N. filled - Summary section complete

**Self-Review Status:** PASSED

**Leave it Better:** Investigation is synthesis work - the externalized knowledge is the guide update itself.

**Discovered Work:** No new bugs discovered. Some Jan 7 investigations are incomplete (template only) but those are separate issues.

---

## Investigation History

**2026-01-07 [start]:** Investigation started
- Initial question: What patterns from Jan 7 investigations should be consolidated?
- Context: 14 new dashboard investigations since Jan 6 synthesis (44 → 58 total)

**2026-01-07 [checkpoint]:** Analysis complete
- Identified 4 new theme areas from Jan 7 work
- Ready to update dashboard guide

**2026-01-07 [complete]:** Guide updated
- Updated `.kb/guides/dashboard.md` with all new patterns
- Added 5 new Common Problems entries
- Added 3 new Key Concepts entries
- Added Performance Patterns section with lessons from 4 slowness incidents
- Added Activity Feed Persistence architecture to Integration Points
- Updated References with new investigation categories
- Status: Complete
