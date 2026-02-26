## Summary (D.E.K.N.)

**Delta:** Implemented close_reason fallback for completed agents without SYNTHESIS.md in the dashboard.

**Evidence:** Added CloseReason field to Issue struct, API response includes close_reason, UI shows fallback in agent cards and detail panel.

**Knowledge:** Light-tier spawns don't create SYNTHESIS.md by design; beads close_reason provides equivalent summary information.

**Next:** Close issue - implementation complete with all tests passing.

**Confidence:** High (90%) - Feature implemented, tests pass, but visual validation blocked by orch serve CWD issue.

---

# Investigation: Add Fallback Beads Close Reason

**Question:** How do we show useful completion summaries for light-tier agents that don't have SYNTHESIS.md?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Light-tier spawns skip SYNTHESIS.md by design

**Evidence:** `config.go:31` shows feature-impl defaults to light tier which explicitly skips synthesis generation.

**Source:** pkg/spawn/config.go, `.orch/templates/SYNTHESIS.md`

**Significance:** This is intentional behavior - not a bug. Light-tier agents need an alternative summary source.

---

### Finding 2: Beads close_reason contains equivalent summary information

**Evidence:** `bd show orch-go-9es5` returns close_reason with detailed completion summary: "Stats bar now uses flex-wrap with grouped items..."

**Source:** `bd list --json`, `bd show <id> --json`

**Significance:** close_reason is already populated by agents when completing via `bd close --reason`. This is the natural fallback for light-tier agents.

---

### Finding 3: Workspace beads_id requires SPAWN_CONTEXT.md parsing

**Evidence:** Workspace names like `og-feat-add-focus-drift-24dec` don't contain beads_id. It's stored in SPAWN_CONTEXT.md as "spawned from beads issue: **orch-go-zsuq.2**".

**Source:** `.orch/workspace/*/SPAWN_CONTEXT.md`, `cmd/orch/review.go:167`

**Significance:** Function `extractBeadsIDFromWorkspace` already exists and can be reused for workspace-based agent detection.

---

## Synthesis

**Key Insights:**

1. **Beads close_reason is the natural fallback** - It's already populated by agents during normal completion workflow.

2. **Workspace vs session agent detection** - Active sessions have format "workspace [beads-id]" in title, workspace directories don't. The alreadyIn check needed updating.

3. **Batch fetching enables efficiency** - GetIssuesBatch and GetCommentsBatch allow fetching close_reason for all agents in one call.

**Answer to Investigation Question:**

Light-tier agents can show beads close_reason as the summary in place of synthesis. The implementation adds close_reason to the API response and updates the UI to fall back to it when synthesis is null.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Code changes are complete, all tests pass, and the logic is correct. Minor uncertainty due to not being able to visually validate end-to-end (orch serve CWD issue).

**What's certain:**

- ✅ CloseReason field added to Issue struct
- ✅ API returns close_reason for completed agents
- ✅ UI components show fallback when synthesis is null
- ✅ All Go tests pass
- ✅ TypeScript compiles without errors

**What's uncertain:**

- ⚠️ Visual validation blocked by orch serve running from wrong directory

**What would increase confidence to Very High:**

- End-to-end visual testing with server running from project directory
- Playwright tests for close_reason display

---

## Implementation Recommendations

### Recommended Approach ⭐

**Use beads close_reason as synthesis fallback** - Already implemented.

**Why this approach:**
- Uses existing data (no new agent behavior needed)
- Consistent with how agents already report completion
- Works with existing batch fetching infrastructure

**Trade-offs accepted:**
- close_reason format may be less structured than SYNTHESIS.md
- Depends on agents including useful info in close_reason

**Implementation sequence:**
1. Add CloseReason to Issue struct ✅
2. Add close_reason to API response ✅
3. Fetch from closed issues in batch ✅
4. Update UI components to show fallback ✅

---

## References

**Files Modified:**
- `pkg/verify/check.go` - Added CloseReason to Issue struct
- `cmd/orch/serve.go` - Added close_reason to API, improved workspace detection
- `web/src/lib/stores/agents.ts` - Added close_reason to Agent interface
- `web/src/lib/components/agent-card/agent-card.svelte` - Fallback display logic
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Detail panel fallback
- `web/src/lib/components/collapsible-section/collapsible-section.svelte` - Group summary fallback

**Commands Run:**
```bash
# Build and test
go build ./...
go test ./...
bun run check
```

---

## Investigation History

**2025-12-24 14:46:** Investigation started
- Initial question: How to show completion summaries for light-tier agents?
- Context: Dashboard shows no TLDR for completed agents without SYNTHESIS.md

**2025-12-24 15:00:** Implementation complete
- Final confidence: High (90%)
- Status: Complete
- Key outcome: close_reason fallback implemented across API and UI
