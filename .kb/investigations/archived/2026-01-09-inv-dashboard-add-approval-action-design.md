<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard approval action should extend existing ReviewState infrastructure and use the `✅ APPROVED` format recognized by visual verification gates.

**Evidence:** Codebase analysis shows approval system already exists in pkg/verify/visual.go with humanApprovalPatterns, orch complete --approve uses this format, and .review-state.json already tracks workspace metadata.

**Knowledge:** Reusing existing patterns (ReviewState, approval format, API structure) is better than inventing new ones; button in agent detail panel is most discoverable UX option for MVP.

**Next:** Implement in sequence: (1) extend ReviewState struct, (2) add POST /api/approve endpoint, (3) add approve button to agent detail panel, (4) show approval status in UI.

**Promote to Decision:** recommend-no - Implementation decision, not architectural.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Dashboard Add Approval Action Design

**Question:** How should we add an approval action to the dashboard for design work?

**Started:** 2026-01-09 15:30
**Updated:** 2026-01-09 15:30
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Begin implementation (backend first, then frontend)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Approval System Already Exists for Visual Verification

**Evidence:** The codebase has a comprehensive visual verification system in pkg/verify/visual.go that includes:
- humanApprovalPatterns: Regex patterns to detect approval markers like `✅ APPROVED`, `UI APPROVED`, `human_approved: true`
- VerifyVisualVerification function that checks for both evidence AND human approval
- complete_cmd.go has --approve flag that adds approval comment: `✅ APPROVED - Visual changes reviewed and approved by orchestrator`

**Source:** 
- /Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go:158-169 (humanApprovalPatterns)
- /Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go:830-947 (VerifyVisualVerification)
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:32 (completeApprove flag)

**Significance:** We don't need to invent a new approval pattern - we should use the existing `✅ APPROVED` format that the visual verification system already recognizes. The dashboard approval action should create the same format that `orch complete --approve` creates.

---

### Finding 2: Dashboard Agent Detail Panel Has Tabs Based on Agent Status

**Evidence:** The agent detail panel (agent-detail-panel.svelte) shows different tabs depending on agent status:
- Active agents: only Activity tab
- Completed agents: Synthesis and Investigation tabs
- Abandoned agents: only Investigation tab
The panel uses a slide-out design at 80-85% width with backdrop and escape-key handling.

**Source:** 
- /Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/agent-detail/agent-detail-panel.svelte:11-56 (tab logic)
- /Users/dylanconlin/Documents/personal/orch-go/web/src/lib/components/agent-detail/agent-detail-panel.svelte:178-264 (panel structure)

**Significance:** The approval button should appear when viewing completed agents (since that's when visual verification is needed). It should be placed in the header area or status bar where other agent actions/info are displayed.

---

### Finding 3: Dashboard API Pattern Uses Handlers in serve.go

**Evidence:** The serve.go file registers HTTP endpoints with corsHandler wrapper:
- Pattern: `mux.HandleFunc("/api/xxx", corsHandler(handleXxx))`
- Existing POST endpoint example: `/api/issues` for creating beads issues (handleIssues)
- Comments are added via beads commands, not direct API calls

**Source:**
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:219-328 (endpoint registration)
- /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:277-278 (POST /api/issues example)

**Significance:** We should create a `POST /api/approve` endpoint that:
1. Takes agentID and optional description
2. Creates beads comment with `✅ APPROVED: [description]` format
3. Updates workspace manifest if needed
4. Returns success/failure to UI

---

### Finding 4: ReviewState JSON Serves as Workspace Manifest

**Evidence:** Workspaces have a .review-state.json file (pkg/verify/review_state.go) that tracks:
- reviewed_at: timestamp when reviewed
- workspace_id: workspace directory name
- beads_id: associated beads issue ID
- acted_on/dismissed: arrays for recommendation tracking
- light_tier_acknowledged: boolean for light-tier completion

Example from og-feat-fix-skill-constraint-23dec/.review-state.json:
```json
{
  "reviewed_at": "2026-01-03T14:19:56.330316-08:00",
  "workspace_id": "og-feat-fix-skill-constraint-23dec",
  "beads_id": "orch-go-b1j3",
  "light_tier_acknowledged": true
}
```

**Source:**
- /Users/dylanconlin/Documents/personal/orch-go/pkg/verify/review_state.go:14-39 (ReviewState struct)
- /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-fix-skill-constraint-23dec/.review-state.json

**Significance:** The ReviewState file is the "workspace manifest" mentioned in the task. We should extend it to include approval fields (approved: bool, approved_at: time.Time, approved_by: string, approval_description: string).

---

## Synthesis

**Key Insights:**

1. **Reuse Existing Approval Infrastructure** - The codebase already has a comprehensive approval system via `orch complete --approve`, which uses the `✅ APPROVED` format recognized by visual verification gates. We should reuse this pattern rather than inventing a new one.

2. **ReviewState is the Workspace Manifest** - The .review-state.json file already tracks workspace review state (reviewed_at, workspace_id, beads_id). We should extend this structure to include approval fields (approved, approved_at, approved_by, approval_description) rather than creating a separate manifest file.

3. **API-First Design** - The dashboard follows an API-first pattern where all actions go through /api/* endpoints in serve.go. The approval action should follow this pattern with a POST /api/approve endpoint that creates the beads comment and updates the workspace manifest.

4. **Button in Agent Detail Panel is MVP** - Of the three UX options (button, /approve command, keyboard shortcut), a button in the agent detail panel is most discoverable and easiest to implement as MVP. The button should appear in the header/status bar area where agent actions are displayed.

**Answer to Investigation Question:**

We should add an approval action to the dashboard by:
1. Extending ReviewState struct with approval fields
2. Creating POST /api/approve endpoint in serve.go
3. Adding an "Approve" button to the agent detail panel header (visible for completed agents with visual changes)
4. Using the existing `✅ APPROVED: [description]` format for beads comments
5. Updating the workspace .review-state.json with approval metadata

This approach reuses existing infrastructure (approval patterns, ReviewState, API patterns) and provides a simple, discoverable UX for the ui-design-session workflow.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Extend ReviewState + Add API Endpoint + Button UI** - Add approval tracking to the existing ReviewState infrastructure, expose it via a new /api/approve endpoint, and add an "Approve" button to the agent detail panel.

**Why this approach:**
- Reuses existing ReviewState file and infrastructure (no new file types to manage)
- Follows established API pattern from serve.go (consistent with other dashboard actions)
- Leverages existing approval format recognized by visual verification system
- Provides discoverable UI without requiring users to learn new commands

**Trade-offs accepted:**
- Deferring /approve command and keyboard shortcuts (can add later if needed)
- Not adding approval UI to agent cards (only in detail panel - simpler, less clutter)
- Using beads comments as source of truth (requires beads daemon, but matches existing pattern)

**Implementation sequence:**
1. **Backend: Extend ReviewState** - Add approval fields to struct, update Load/Save functions
2. **Backend: Add API endpoint** - Create POST /api/approve handler in serve.go
3. **Frontend: Add approval button** - Add button to agent detail panel header with click handler
4. **Frontend: Visual feedback** - Show approval status in UI (badge, timestamp)
5. **Integration: Test with real agent** - Verify approval creates comment + updates manifest

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
1. ReviewState struct extension (adds approval fields)
2. POST /api/approve endpoint (backend logic to create comment + update manifest)
3. Approve button in agent detail panel (UI component)
4. Visual approval status display (show approved state in UI)

**Things to watch out for:**
- ⚠️ **Error handling:** API endpoint must handle cases where beads issue doesn't exist (untracked agents)
- ⚠️ **Race conditions:** Multiple approval clicks should be idempotent (don't create duplicate comments)
- ⚠️ **Cross-project agents:** Need to handle agents spawned with --workdir in other projects
- ⚠️ **Permission model:** Currently no auth - anyone with dashboard access can approve (acceptable for single-user MVP)
- ⚠️ **Approval description:** Need UI to capture optional description from user (input field or prompt)

**Areas needing further investigation:**
- Should approval be allowed for non-completed agents? (Recommendation: only completed agents)
- Should there be an "unapprove" action? (Recommendation: no - approval is one-way gate)
- Should approval status show in agent cards grid? (Recommendation: defer - detail panel is enough)

**Success criteria:**
- ✅ Clicking "Approve" button creates beads comment with `✅ APPROVED` format
- ✅ ReviewState JSON updated with approved: true, approved_at: timestamp
- ✅ Agent detail panel shows approval status (badge or indicator)
- ✅ Visual verification gate in `orch complete` recognizes dashboard approval
- ✅ No errors when approving agents in different projects (cross-project support)

---

## References

**Files Examined:**
- pkg/verify/visual.go:158-169 - humanApprovalPatterns showing approval markers
- pkg/verify/visual.go:830-947 - VerifyVisualVerification function
- pkg/verify/review_state.go:14-39 - ReviewState struct definition
- cmd/orch/complete_cmd.go:32,240-248 - addApprovalComment implementation
- cmd/orch/serve.go:219-328 - API endpoint registration pattern
- web/src/lib/components/agent-detail/agent-detail-panel.svelte:11-56 - Tab logic and panel structure

**Commands Run:**
```bash
# Find approval-related code
grep -r "approved" --include="*.go" | head -10

# Find workspace manifest references
grep -r "manifest" --include="*.go" | grep -i workspace

# Find ReviewState usage
grep -r "ReviewState\|reviewState" --include="*.go"

# Check workspace directory structure
ls -la .orch/workspace/og-feat-fix-skill-constraint-23dec/
```

**External Documentation:**
- None needed - all patterns found in existing codebase

**Related Artifacts:**
- **Workspace:** .orch/workspace/og-feat-dashboard-add-approval-09jan-463e - Current implementation workspace

---

## Investigation History

**2026-01-09 15:30:** Investigation started
- Initial question: How should we add an approval action to the dashboard for design work?
- Context: Task requires explicit approval mechanism for design artifacts (ui-design-session workflow)

**2026-01-09 15:45:** Found existing approval infrastructure
- Discovered humanApprovalPatterns in pkg/verify/visual.go
- Found orch complete --approve command using `✅ APPROVED` format
- Identified ReviewState as workspace manifest

**2026-01-09 16:00:** Investigation completed
- Status: Complete
- Key outcome: Extend ReviewState + add API endpoint + button UI approach recommended
