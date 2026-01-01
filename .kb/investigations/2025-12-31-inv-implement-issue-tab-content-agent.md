## Summary (D.E.K.N.)

**Delta:** Implemented full Issue tab with rich issue details, markdown description, parent/child relationships, and comments timeline showing phase reports.

**Evidence:** API endpoint `/api/beads/issue?id=<beads-id>` added to serve.go; Svelte component updated with full Issue tab UI including header with status/priority/labels, markdown-rendered description, relationship display, and timeline with phase highlighting.

**Knowledge:** Beads issue data is available via verify.GetIssue() and beads.Client.Show(); comments can be parsed for Phase:/BLOCKED:/QUESTION: patterns for timeline display.

**Next:** Visual verification needed; then close issue.

---

# Investigation: Implement Issue Tab Content Agent

**Question:** How to implement rich Issue tab content in agent detail pane?

**Started:** 2025-12-31
**Updated:** 2025-12-31
**Owner:** Agent og-feat-implement-issue-tab-31dec
**Phase:** Complete
**Next Step:** Visual verification
**Status:** Complete

---

## Findings

### Finding 1: API Structure for Issue Details

**Evidence:** verify.Issue struct contains ID, Title, Description, Status, IssueType, CloseReason. beads.Issue has Priority, Labels, Dependencies, CreatedAt/UpdatedAt/ClosedAt. Comments are fetched separately via verify.GetComments().

**Source:** pkg/verify/check.go:20-28, pkg/beads/types.go:138-160

**Significance:** Need to combine verify.GetIssue() with beads.Client.Show() to get all fields, and separately fetch comments.

### Finding 2: Comment Parsing Patterns

**Evidence:** Comments contain Phase: patterns for agent progress tracking. Also BLOCKED: and QUESTION: patterns for timeline display. ParsePhaseFromComments() in verify/check.go already handles this.

**Source:** pkg/verify/check.go:75-100

**Significance:** Can reuse pattern detection for rich timeline display with visual indicators.

### Finding 3: Cross-Project Support

**Evidence:** Prior knowledge indicates beads comments must be fetched from agent's project directory, not orchestrator's current directory. GetCommentsWithDir() supports this.

**Source:** SPAWN_CONTEXT.md prior knowledge section

**Significance:** API endpoint needs project parameter for cross-project agent visibility.

---

## Synthesis

**Key Insights:**

1. **Data Aggregation** - Full issue details require combining multiple API calls (GetIssue + client.Show() + GetComments)

2. **Pattern Recognition** - Comments can be parsed client-side or server-side for Phase/BLOCKED/QUESTION highlighting

3. **Relationship Extraction** - Dependencies JSON can contain parent/child relationships with dependency_type field

**Answer to Investigation Question:**

Implemented by: 1) Adding /api/beads/issue endpoint that aggregates issue data and comments, 2) Adding TypeScript types and fetchIssueDetails function, 3) Replacing Issue tab with rich UI showing header, description, relationships, and timeline.

---

## References

**Files Examined:**
- cmd/orch/serve.go - Added handleBeadsIssue endpoint
- pkg/verify/check.go - GetIssue, GetComments patterns
- pkg/beads/types.go - Issue and Comment structures
- web/src/lib/stores/agents.ts - Added IssueDetail types and fetch function
- web/src/lib/components/agent-detail/agent-detail-panel.svelte - Updated Issue tab

**Commands Run:**
```bash
# Verify Go compilation
go build ./cmd/orch/...
```

---

## Investigation History

**2025-12-31 02:11:** Investigation started
- Initial question: How to implement Issue tab content with comments timeline
- Context: SPAWN_CONTEXT.md task to show issue header, description, relationships, and phase reports

**2025-12-31 02:18:** Implementation complete
- Added API endpoint, TypeScript types, and UI components
- Status: Complete pending visual verification
