<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 12 status investigations should be consolidated: prior synthesis (Jan 6) created `.kb/guides/status.md` covering 10 investigations; 2 new investigations (Jan 6-7) need to be integrated.

**Evidence:** Read all 12 investigations. Jan 6 synthesis produced comprehensive guide. Two new findings: (1) orch complete missing session deletion, (2) SESSION METRICS added to status. Neither is yet in guide.

**Knowledge:** Status guide is authoritative but 2 days stale. Incremental update is cheaper than re-synthesis. Archive candidates: older stale investigations superseded by guide.

**Next:** Update `.kb/guides/status.md` with two new findings; archive 8 older investigations fully superseded by guide.

**Promote to Decision:** recommend-no (maintenance consolidation, not architectural)

---

# Investigation: Synthesize Status Investigations (12)

**Question:** Do the 12 status investigations require synthesis, or is existing synthesis sufficient with incremental updates?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-work-synthesize-status-investigations-08jan-ea49
**Phase:** Complete
**Next Step:** None (proposals ready for orchestrator review)
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Prior Synthesis Already Exists (Jan 6)

**Evidence:** `.kb/investigations/2026-01-06-inv-synthesize-status-investigations.md` synthesized 10 investigations and created `.kb/guides/status.md` (308 lines, comprehensive reference).

**Source:** 
- `2026-01-06-inv-synthesize-status-investigations.md` - synthesis investigation
- `.kb/guides/status.md` - authoritative guide created

**Significance:** The heavy lifting is done. This synthesis task only needs to handle 2 new investigations, not re-do all 12.

---

### Finding 2: Two New Investigations Since Jan 6 Synthesis

**Evidence:** 
1. `2026-01-06-inv-orch-status-shows-completed-agents.md` - Identified `orch complete` doesn't delete OpenCode sessions, causing completed agents to appear in status until 30-min window expires
2. `2026-01-07-inv-orch-status-surface-drift-metrics.md` - Added SESSION METRICS section showing time in session, last spawn, spawn count

**Source:** Files dated after Jan 6 synthesis

**Significance:** These two findings need to be added to the guide for completeness.

---

### Finding 3: Investigations Have Varying Archival Readiness

**Evidence:** Analysis of 12 investigations:

| Investigation | Disposition |
|--------------|-------------|
| 2025-12-20-inv-enhance-status-command-swarm-progress.md | Superseded by guide Section "Key Evolution #1" |
| 2025-12-21-inv-investigate-orch-status-showing-stale.md | Superseded by guide Section "Stale Session Problem" |
| 2025-12-21-inv-orch-status-showing-stale-sessions.md | Superseded by guide Section "Stale Session Problem" |
| 2025-12-22-debug-orch-status-stale-sessions.md | Superseded by guide Section "Key Evolution #2" |
| 2025-12-22-inv-update-orch-status-use-islive.md | Incomplete template - never finished |
| 2025-12-23-inv-orch-status-can-detect-active.md | Superseded by guide Section "Active Detection" |
| 2025-12-23-inv-orch-status-shows-active-agents.md | Superseded by guide Section "Title Format" |
| 2025-12-23-inv-orch-status-takes-11-seconds.md | Superseded by guide Section "Performance" |
| 2025-12-24-inv-fix-status-filter-test-expects.md | Already resolved, minimal content |
| 2026-01-05-debug-fix-orch-status-showing-different.md | Superseded by guide Section "Cross-Project Visibility" |
| 2026-01-06-inv-orch-status-shows-completed-agents.md | NEW - needs guide integration |
| 2026-01-07-inv-orch-status-surface-drift-metrics.md | NEW - needs guide integration |

**Source:** Review of each investigation file

**Significance:** 8 investigations are fully superseded by the guide. 2 are new (need integration). 1 synthesis investigation is meta. 1 is incomplete/orphaned.

---

## Synthesis

**Key Insights:**

1. **Incremental update, not re-synthesis** - The Jan 6 synthesis created a comprehensive guide. The 2 new investigations represent incremental additions, not fundamental changes.

2. **Two new topics for guide:**
   - **Session Cleanup on Complete:** `orch complete` should delete OpenCode sessions (like `orch abandon` does) - pattern from `abandon_cmd.go:165-174`
   - **Session Drift Metrics:** SESSION METRICS section added showing orchestrator session state

3. **Archive candidates exist** - 8 investigations are fully captured in the guide with no unique value remaining. Archiving reduces future agent confusion.

**Answer to Investigation Question:**

The 12 investigations do NOT require full re-synthesis. The existing `.kb/guides/status.md` is comprehensive. Required actions:
1. Update guide with 2 new findings (session cleanup, drift metrics)
2. Archive 8 superseded investigations
3. Keep 2026-01-06 synthesis investigation as reference
4. Disposition unclear for incomplete 2025-12-22-inv-update-orch-status-use-islive.md

---

## Structured Uncertainty

**What's tested:**

- ✅ Prior synthesis exists and is comprehensive (verified: read full 308-line guide)
- ✅ Two investigations are dated after synthesis (verified: file dates)
- ✅ Supersession analysis is accurate (verified: compared each investigation D.E.K.N. to guide sections)

**What's untested:**

- ⚠️ Whether archiving investigations breaks any citation links
- ⚠️ Whether the two new findings are fully accurate (didn't re-test implementations)

**What would change this:**

- Finding would be wrong if guide has significant gaps not covered by the 10 investigations
- Finding would be incomplete if there are additional status investigations not in the list of 12

---

## Proposed Actions

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `.kb/guides/status.md` | Add Section 6: "Session Cleanup on Complete" | Document that orch complete doesn't delete sessions, fix pattern | [ ] |
| U2 | `.kb/guides/status.md` | Add Section 7: "Session Drift Metrics" | Document new SESSION METRICS section in status output | [ ] |
| U3 | `.kb/guides/status.md` | Update "Last verified" date | Guide being updated | [ ] |
| U4 | `.kb/guides/status.md` | Add two new investigations to Source Investigations table | Complete the reference | [ ] |

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `2025-12-20-inv-enhance-status-command-swarm-progress.md` | Superseded by guide - all content in "Key Evolution #1" | [ ] |
| A2 | `2025-12-21-inv-investigate-orch-status-showing-stale.md` | Superseded by guide - all content in "Stale Session Problem" | [ ] |
| A3 | `2025-12-21-inv-orch-status-showing-stale-sessions.md` | Superseded by guide - all content in "Stale Session Problem" | [ ] |
| A4 | `2025-12-22-debug-orch-status-stale-sessions.md` | Superseded by guide - all content in "Key Evolution #2" | [ ] |
| A5 | `2025-12-22-inv-update-orch-status-use-islive.md` | Incomplete template - never finished, no value | [ ] |
| A6 | `2025-12-23-inv-orch-status-can-detect-active.md` | Superseded by guide - all content in "Active Detection" | [ ] |
| A7 | `2025-12-23-inv-orch-status-shows-active-agents.md` | Superseded by guide - all content in "Title Format" | [ ] |
| A8 | `2025-12-23-inv-orch-status-takes-11-seconds.md` | Superseded by guide - all content in "Performance" | [ ] |
| A9 | `2025-12-24-inv-fix-status-filter-test-expects.md` | Issue was already resolved before agent spawned, minimal content | [ ] |
| A10 | `2026-01-05-debug-fix-orch-status-showing-different.md` | Superseded by guide - all content in "Cross-Project Visibility" | [ ] |

### Keep Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| K1 | `2026-01-06-inv-synthesize-status-investigations.md` | Meta-synthesis investigation - documents why guide was created | [ ] |
| K2 | `2026-01-06-inv-orch-status-shows-completed-agents.md` | New finding to integrate, then archive | [ ] |
| K3 | `2026-01-07-inv-orch-status-surface-drift-metrics.md` | New finding to integrate, then archive | [ ] |

**Summary:** 13 proposals (4 update, 10 archive, 3 keep)
**High priority:** U1-U4 (guide updates before archiving)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Incremental Guide Update + Selective Archive** - Update the guide with 2 new sections, then archive superseded investigations.

**Why this approach:**
- Guide is authoritative reference - keep it current
- Archiving reduces future agent confusion
- Preserves synthesis investigation as meta-reference

**Trade-offs accepted:**
- Archived investigations may lose some nuance
- Archive links from other artifacts would break (acceptable - guide replaces them)

**Implementation sequence:**
1. Update `.kb/guides/status.md` with new sections (U1-U4)
2. Move archived investigations to `.kb/investigations/archived/` (A1-A10)
3. Commit changes

---

## References

**Files Examined:**
- All 12 status investigations listed in spawn context
- `.kb/guides/status.md` - existing authoritative guide
- `2026-01-06-inv-synthesize-status-investigations.md` - prior synthesis

**Commands Run:**
```bash
# Chronicle for evolution timeline
kb chronicle "status"

# List investigations for comparison
ls -la .kb/investigations/ | grep status
```

**Related Artifacts:**
- **Guide:** `.kb/guides/status.md` - Authoritative reference (to be updated)
- **Prior Synthesis:** `.kb/investigations/2026-01-06-inv-synthesize-status-investigations.md`

---

## Investigation History

**2026-01-08:** Investigation started
- Initial question: Do 12 status investigations need synthesis?
- Context: kb reflect flagged 12 investigations on "status" topic

**2026-01-08:** Found prior synthesis exists
- Jan 6 synthesis already created comprehensive guide
- Only 2 new investigations since synthesis

**2026-01-08:** Investigation completed
- Status: Complete
- Key outcome: Incremental update to guide + archive 10 superseded investigations

---

## Self-Review

- [x] Real test performed (not code review) - Read all 12 investigations and compared to guide
- [x] Conclusion from evidence (not speculation) - Based on actual file contents
- [x] Question answered - Explained that incremental update is needed, not re-synthesis
- [x] File complete - All sections filled
- [x] D.E.K.N. filled - Summary section complete
- [x] Proposed Actions section completed with structured proposals

**Self-Review Status:** PASSED
