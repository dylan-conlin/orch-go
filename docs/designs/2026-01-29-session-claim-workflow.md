# Design: Session Claim Workflow

**Problem:** Users can start OpenCode sessions outside of `orch spawn` (e.g., via `opencode` CLI directly, or sessions started before orch integration). These untracked sessions are now visible in `orch status --all` (via orch-go-20988), but cannot be managed with orch tooling like `tail`, `complete`, or `cleanup` using beads IDs. Users must use session IDs directly (e.g., `orch tail --session ses_xxx`), which is inconvenient and breaks the beads-centric workflow.

**Success Criteria:**
- Users can "claim" an untracked session and associate it with a beads issue
- After claiming, `orch tail <beads-id>`, `orch complete <beads-id>`, and other beads-centric commands work with the claimed session
- Claimed sessions appear as tracked in `orch status`
- Workflow is simple and requires minimal user input

## Proposed Changes

### 1. Add `orch claim` command (cmd/orch/claim_cmd.go)

New command to claim an untracked session:

```bash
orch claim <session-id> <beads-id>
```

**Workflow:**
1. Validate session ID exists in OpenCode
2. Validate beads ID exists in beads
3. Check if session is already tracked (has workspace with .session_id)
4. Check if beads issue is already claimed (has workspace)
5. Create workspace directory: `.orch/workspace/<workspace-name>`
6. Write workspace files:
   - `.session_id` - OpenCode session ID
   - `.beads_id` - Beads issue ID
   - `.tier` - Default to "light" (no synthesis required for claimed sessions)
   - `.spawn_time` - Current time
   - `AGENT_MANIFEST.json` - Metadata for the claimed session
7. Optionally update session title to include `[beads-id]` for visibility in `orch status`

**Workspace naming convention:**
- Follow existing pattern: `<project>-claimed-<description>-<date>-<hash>`
- Extract description from session title (first 20 chars, sanitized)
- Example: `og-claimed-add-feature-29jan-a1b2`

### 2. Workspace Files

**AGENT_MANIFEST.json format:**
```json
{
  "workspace_name": "og-claimed-add-feature-29jan-a1b2",
  "skill": "claimed",
  "beads_id": "orch-go-21029",
  "project_dir": "/Users/dylan/orch-go",
  "git_baseline": "<current-git-sha>",
  "spawn_time": "2026-01-29T21:45:00Z",
  "tier": "light",
  "spawn_mode": "claimed",
  "claimed_session_id": "ses_3f30689ebffeBb6HL1yuXUw0l9",
  "claimed_from_title": "Original session title"
}
```

### 3. Session Title Update (Optional)

Update OpenCode session title to include beads ID for visibility:
- Original: "Add new feature"
- Updated: "Add new feature [orch-go-21029]"

**Note:** This requires adding a new method to `pkg/opencode/client.go`:
```go
func (c *Client) UpdateSessionTitle(sessionID, newTitle string) error {
    // PATCH /api/sessions/{sessionID}
    // { "title": "new title" }
}
```

**Decision:** Title update is **optional** - workspace files are sufficient for tracking. We can defer this to a follow-up enhancement if needed.

### 4. Validation Gates

**Pre-claim validation:**
- Session ID exists: `client.GetSession(sessionID)`
- Beads ID exists: `resolveShortBeadsID(beadsID)`
- Session not already claimed: Check if workspace exists with matching `.session_id`
- Beads issue not already claimed: Check if workspace exists with matching `.beads_id`

**Error messages:**
- "Session ses_xxx not found in OpenCode"
- "Session ses_xxx is already claimed (workspace: <workspace-name>)"
- "Beads issue <beads-id> is already claimed (workspace: <workspace-name>)"
- "Beads issue <beads-id> not found"

## Testing Strategy

**Manual testing:**
1. Start an untracked session: `opencode --dir ~/orch-go "add a feature"`
2. Get session ID from `orch status --all`
3. Create beads issue: `bd create "Test claim workflow"`
4. Claim session: `orch claim ses_xxx orch-go-yyyy`
5. Verify: `orch tail orch-go-yyyy` shows session output
6. Verify: `orch status` shows session as tracked (no "untracked" badge)

**Automated tests:**
- Unit test workspace file creation
- Unit test validation logic (session exists, not already claimed)
- Integration test: claim session → tail by beads ID

## Alternatives Considered

### Option A: Auto-claim via interactive prompt in `orch status`
- **Pros:** Seamless UX - user sees untracked session and can claim it immediately
- **Cons:** Complex UX flow, requires terminal interaction, harder to script
- **Decision:** Defer to future enhancement - explicit `orch claim` is simpler to start

### Option B: Update session title without creating workspace
- **Pros:** Minimal state changes
- **Cons:** Doesn't enable beads-centric tooling (`tail`, `complete`) - they require workspace files
- **Decision:** Not viable - workspace is essential for orch tooling

### Option C: Require workspace name as input
- **Pros:** User has full control over workspace naming
- **Cons:** Extra friction, workspace name should be derivable
- **Decision:** Auto-generate workspace name from session title + beads ID

## Implementation Sequence

1. **Phase 1:** Add `orch claim` command with workspace creation (no title update)
2. **Phase 2:** Add validation gates (session exists, not already claimed)
3. **Phase 3:** Add OpenCode title update (optional enhancement)
4. **Phase 4:** Add interactive claim workflow in `orch status` (future)

## Success Metrics

- Users can claim untracked sessions in < 5 seconds
- Claimed sessions work with all beads-centric orch commands
- No manual workspace file creation required
