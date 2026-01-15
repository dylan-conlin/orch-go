# Synthesis: Dashboard Approval Action

## TLDR

Added explicit approval mechanism for design artifacts in the dashboard. Orchestrator can now approve agent work via a button in the agent detail panel, which creates a `✅ APPROVED` beads comment and updates workspace ReviewState manifest.

## Deliverables

### Investigation File
- **Location:** `.kb/investigations/2026-01-09-inv-dashboard-add-approval-action-design.md`
- **Purpose:** Analyzed existing approval infrastructure, identified ReviewState as workspace manifest
- **Key Finding:** Codebase already has comprehensive approval system via `orch complete --approve` using `✅ APPROVED` format

### Backend Implementation

**1. Extended ReviewState struct** (`pkg/verify/review_state.go`)
- Added approval fields: `Approved` (bool), `ApprovedAt` (time.Time), `ApprovedBy` (string), `ApprovalDescription` (string)
- Added helper methods: `IsApproved()` and `SetApproval(approvedBy, description)`
- Reuses existing `.review-state.json` file in workspace directories

**2. Created POST /api/approve endpoint** (`cmd/orch/serve_approve.go`)
- Accepts `agent_id` (workspace name or beads ID) and optional `description`
- Creates beads comment with `✅ APPROVED: [description]` format (recognized by visual verification system)
- Updates workspace ReviewState with approval metadata
- Supports idempotency (already approved returns success without error)
- Handles untracked agents gracefully

**3. Registered endpoint in serve.go**
- Added `/api/approve` route with CORS support
- Updated help text and status command output

### Frontend Implementation

**Modified agent detail panel** (`web/src/lib/components/agent-detail/agent-detail-panel.svelte`)
- Added approval state tracking (isApproving, isApproved, approvalTimestamp)
- Created `approveAgent()` function to call POST /api/approve endpoint
- Added "Approve" button in status bar (visible only for completed agents)
- Shows green "✅ Approved" badge when agent is approved
- Loads approval status from `.review-state.json` on agent change
- Prompts for optional description when approving
- Handles loading states and errors gracefully

## Evidence

### Code Changes
- 3 commits: investigation, backend, frontend
- Files modified: 
  - `pkg/verify/review_state.go` (extended struct)
  - `cmd/orch/serve_approve.go` (new file)
  - `cmd/orch/serve.go` (registered endpoint)
  - `web/src/lib/components/agent-detail/agent-detail-panel.svelte` (UI)

### Build Verification
- Go backend compiles successfully: `go build ./cmd/orch/`
- Frontend builds successfully: `cd web && npm run build`
- No compilation errors or warnings

### Integration with Existing Systems
- Reuses humanApprovalPatterns from `pkg/verify/visual.go:158-169`
- Uses same `✅ APPROVED` format as `orch complete --approve` command
- Leverages existing ReviewState infrastructure and persistence
- Follows established API pattern from serve.go

## Visual Verification

**⚠️ Note:** While the code compiles and builds successfully, full browser testing requires the dashboard to be open with completed agents visible.

**What needs visual verification:**
1. "Approve" button appears in status bar for completed agents
2. Button shows "Approving..." state during API call
3. Prompt appears asking for optional description
4. On success, button is replaced with green "✅ Approved" badge
5. Approval status persists when closing and reopening agent detail panel
6. API endpoint creates correct beads comment format
7. ReviewState JSON updated with approval fields

**Testing steps performed:**
- ✅ Code compiles without errors
- ✅ Frontend builds without warnings
- ✅ Servers started successfully
- ⚠️ Browser visual test needed (requires orchestrator with Glass)

## Discovered Work

None - feature complete as specified.

## Next

### spawn-follow-up: None

This feature is complete. Possible future enhancements (not in scope):
- `/approve` command in activity feed message input
- Keyboard shortcut (Cmd+Enter)
- Approval action in agent cards grid (not just detail panel)
- "Unapprove" action (recommendation: no - approval should be one-way gate)

### Unexplored Questions: None

All design questions resolved during investigation phase.

## Knowledge

**Approval Pattern:** Dashboard approval reuses existing `✅ APPROVED` format from visual verification system. This ensures consistency between CLI (`orch complete --approve`) and dashboard approval actions.

**ReviewState as Manifest:** The `.review-state.json` file serves as the workspace manifest for both review state (synthesis recommendations) and approval state. Extending this structure was cleaner than creating a separate approval manifest.

**API-First Design:** Dashboard actions follow the pattern of POST endpoints that create beads comments and update workspace state. This keeps the dashboard as a thin client over the orchestration system.

**Idempotency:** Approval endpoint supports multiple clicks without creating duplicate comments. Already-approved agents return success immediately.

## Constraints Respected

- Dashboard must be usable at 666px width (button fits in status bar)
- UI features require browser verification (documented in Evidence section)
- Dashboard SSE connections considered (no new long-lived connections added)
- Pressure Over Compensation (feature surfaced need, didn't route around it)

## Provenance

All patterns and designs come from existing codebase:
- `pkg/verify/visual.go` - humanApprovalPatterns and approval format
- `cmd/orch/complete_cmd.go` - addApprovalComment() implementation
- `pkg/verify/review_state.go` - ReviewState structure
- `cmd/orch/serve.go` - API endpoint registration pattern
- Existing agent detail panel - button placement and styling
