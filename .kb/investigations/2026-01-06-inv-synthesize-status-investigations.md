<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Synthesized 10 status investigations (Dec 20 - Jan 5) into single authoritative guide at `.kb/guides/status.md`.

**Evidence:** Read all 10 investigations, identified 5 major evolution themes (stale sessions, performance, liveness detection, title format, cross-project), consolidated into comprehensive guide with architecture, troubleshooting, and constraints sections.

**Knowledge:** `orch status` evolved through multiple fixes: (1) x-opencode-directory header caused 200+ stale sessions, (2) sequential bd calls caused 11s latency, (3) messages endpoint needed for processing detection, (4) session titles needed `[beads-id]` pattern, (5) cross-project needed three-strategy lookup.

**Next:** Close - guide created, future agents should read guide before investigating status issues.

**Promote to Decision:** recommend-no (synthesis consolidation, not new architectural decision)

---

# Investigation: Synthesize Status Investigations

**Question:** What patterns emerged from 10 status-related investigations, and how should they be consolidated into an authoritative reference?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent og-feat-synthesize-status-investigations-06jan-3efe
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Stale Session Problem (Dec 20-22)

**Evidence:** Three investigations (`inv-enhance-status-command-swarm-progress.md`, `inv-investigate-orch-status-showing-stale.md`, `inv-orch-status-showing-stale-sessions.md`) documented the same root issue: calling `ListSessions(projectDir)` with `x-opencode-directory` header returned ALL historical sessions (200-300+) instead of just in-memory sessions (2-4).

**Source:** 
- `2025-12-20-inv-enhance-status-command-swarm-progress.md` - Added swarm metrics
- `2025-12-21-inv-investigate-orch-status-showing-stale.md` - Four-layer architecture discovery
- `2025-12-21-inv-orch-status-showing-stale-sessions.md` - Fix: `ListSessions("")` without header

**Significance:** Identified that OpenCode has separate in-memory and disk storage layers. The fix (`ListSessions("")`) is now stable and documented.

---

### Finding 2: Performance Bottleneck (Dec 23)

**Evidence:** `inv-orch-status-takes-11-seconds.md` traced 11+ second latency to sequential subprocess calls: 3 `bd` calls × 37 agents × ~100ms each = ~11 seconds.

**Source:** `2025-12-23-inv-orch-status-takes-11-seconds.md`

**Significance:** Fixed with batch/parallel fetching: `GetIssuesBatch()`, `ListOpenIssues()`, `GetCommentsBatch()`. Result: 12.2s → ~1s (11x improvement).

---

### Finding 3: Liveness Detection (Dec 23)

**Evidence:** Two investigations (`inv-orch-status-can-detect-active.md`, `inv-orch-status-shows-active-agents.md`) established that:
1. OpenCode sessions have no `status` field
2. SSE busy/idle events have false positives during normal operation
3. Messages endpoint is authoritative: `finish: null` + `completed: 0` = actively generating

**Source:**
- `2025-12-23-inv-orch-status-can-detect-active.md` - Messages endpoint discovery
- `2025-12-23-inv-orch-status-shows-active-agents.md` - Session title format fix

**Significance:** `IsSessionProcessing()` now uses messages endpoint. Session titles must include `[beads-id]` for matching.

---

### Finding 4: Cross-Project Visibility (Jan 5)

**Evidence:** `debug-fix-orch-status-showing-different.md` found that beads comments were looked up using current working directory instead of the agent's actual project.

**Source:** `2026-01-05-debug-fix-orch-status-showing-different.md`

**Significance:** Three-strategy project directory resolution now handles cross-project agents:
1. Session.Directory (if valid)
2. Workspace lookup from current project
3. Derive from beads ID prefix

---

### Finding 5: Consistent Patterns Across Investigations

**Evidence:** All 10 investigations followed similar debugging patterns:
- Check OpenCode session count vs expected
- Check beads issue state
- Trace through data flow from sources to output

**Source:** All 10 investigations

**Significance:** These debugging patterns are now documented in the guide's "Common Problems" section.

---

## Synthesis

**Key Insights:**

1. **Four-layer architecture is fundamental** - OpenCode in-memory, OpenCode disk, orch registry, and tmux windows are independent state sources. Status issues usually involve layer mismatch.

2. **Beads is source of truth for completion** - OpenCode sessions persist indefinitely. Only beads comments/status indicate actual agent completion.

3. **Performance requires batch operations** - O(N) subprocess calls don't scale. All beads operations must be batched.

4. **Title format enables matching** - Session titles with `[beads-id]` pattern enable tmux-to-OpenCode correlation.

5. **Cross-project needs explicit handling** - Beads IDs encode project name, which can be used to derive project directory.

**Answer to Investigation Question:**

The 10 investigations addressed 5 distinct evolution phases of `orch status`:
1. Initial enhancement (swarm metrics, accounts)
2. Stale session filtering (header behavior, 30-min threshold)
3. Performance optimization (batch fetching)
4. Liveness detection (messages endpoint)
5. Cross-project visibility (three-strategy lookup)

These have been consolidated into `.kb/guides/status.md` as the single authoritative reference.

---

## Structured Uncertainty

**What's tested:**

- ✅ Guide covers all 10 investigation topics (verified: enumerated each investigation)
- ✅ Current status_cmd.go implementation matches guide descriptions (verified: read status_cmd.go)
- ✅ Existing status-dashboard.md is complementary, not redundant (verified: compared content)

**What's untested:**

- ⚠️ Guide completeness for future issues (will discover gaps when new bugs arise)
- ⚠️ Guide discoverability (agents may still investigate before reading guide)

**What would change this:**

- New status bugs that don't fit documented patterns would require guide updates
- Changes to OpenCode API behavior could invalidate assumptions

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide created at `.kb/guides/status.md`** - Single authoritative reference synthesizing 10 investigations.

**Why this approach:**
- Follows kb pattern: 10+ investigations → synthesize into guide
- Provides single entry point for future debugging
- Preserves investigation links for deep-dive context

**Trade-offs accepted:**
- Guide may become stale as implementation evolves
- Investigators may still skip reading guide

**Implementation sequence:**
1. ✅ Created comprehensive guide with all 5 evolution themes
2. ✅ Included troubleshooting section for common problems
3. ✅ Cross-referenced existing guides and investigations

---

## References

**Files Examined:**
- `2025-12-20-inv-enhance-status-command-swarm-progress.md`
- `2025-12-21-inv-investigate-orch-status-showing-stale.md`
- `2025-12-21-inv-orch-status-showing-stale-sessions.md`
- `2025-12-22-debug-orch-status-stale-sessions.md`
- `2025-12-22-inv-update-orch-status-use-islive.md`
- `2025-12-23-inv-orch-status-can-detect-active.md`
- `2025-12-23-inv-orch-status-shows-active-agents.md`
- `2025-12-23-inv-orch-status-takes-11-seconds.md`
- `2025-12-24-inv-fix-status-filter-test-expects.md`
- `2026-01-05-debug-fix-orch-status-showing-different.md`
- `cmd/orch/status_cmd.go` - Current implementation

**Commands Run:**
```bash
# Chronicle for evolution timeline
kb chronicle "status"

# Verified guide doesn't duplicate existing content
ls .kb/guides/
```

**Related Artifacts:**
- **Guide Created:** `.kb/guides/status.md` - Authoritative reference
- **Existing Guide:** `.kb/guides/status-dashboard.md` - Dashboard-focused complement

---

## Investigation History

**2026-01-06 16:40:** Investigation started
- Initial question: Synthesize 10 status investigations into guide
- Context: kb synthesis pattern - 10+ investigations → consolidate

**2026-01-06 16:55:** Read all 10 investigations
- Identified 5 major evolution themes
- Noted consistent debugging patterns

**2026-01-06 17:10:** Investigation completed
- Status: Complete
- Key outcome: Created `.kb/guides/status.md` with comprehensive reference
