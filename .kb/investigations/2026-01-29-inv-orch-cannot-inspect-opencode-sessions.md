## Summary (D.E.K.N.)

**Delta:** Orch tooling (tail, status) can inspect non-orch-spawned sessions by adding `--session` flag to `orch tail` and including untracked sessions in `orch status`.

**Evidence:** Code analysis shows `orch tail` requires beads ID (line 33-35), and `orch status` filters sessions without beads IDs (line 284-288). Both tools have the underlying API capability but lack the UI path.

**Knowledge:** The OpenCode client already supports `GetMessages(sessionID)` and `ListSessions("")` - the gap is purely in the CLI interface, not the underlying infrastructure.

**Next:** Implement `orch tail --session <session-id>` flag and add untracked session visibility to `orch status`.

**Promote to Decision:** recommend-no - This is a tactical enhancement, not an architectural change.

---

# Investigation: Orch Cannot Inspect Non-Orch-Spawned OpenCode Sessions

**Question:** How can orchestrators inspect OpenCode sessions that weren't spawned via `orch spawn`?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Agent (architect skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: `orch tail` requires beads ID, has no direct session access

**Evidence:** 
```go
// tail_cmd.go:33-35
var tailCmd = &cobra.Command{
    Use:   "tail [beads-id]",
    Args: cobra.ExactArgs(1),
```

The tail command strictly requires a beads ID argument. Even though the underlying code can fetch messages via `client.GetMessages(sessionID)` (line 69), there's no path for users to provide a session ID directly.

**Source:** `cmd/orch/tail_cmd.go:33-35, 69`

**Significance:** Users with a session ID from the OpenCode dashboard cannot use `orch tail` without first finding the associated beads ID (which may not exist for manually started sessions).

---

### Finding 2: `orch status` filters out sessions without beads IDs

**Evidence:**
```go
// status_cmd.go:284-288
beadsID := extractBeadsIDFromTitle(s.Title)
if beadsID == "" {
    continue
}
```

Sessions without a beads ID in their title are completely invisible to `orch status`. This includes:
- Interactive sessions started via `opencode` directly
- Sessions from other projects without beads integration
- Sessions with custom titles

**Source:** `cmd/orch/status_cmd.go:284-288`

**Significance:** Orchestrators have no visibility into these sessions through orch tooling, despite them being visible in the OpenCode dashboard.

---

### Finding 3: OpenCode API supports direct session access

**Evidence:**
```go
// opencode/client.go:353-374
func (c *Client) GetSession(sessionID string) (*Session, error) { ... }

// opencode/client.go:625-646
func (c *Client) GetMessages(sessionID string) ([]Message, error) { ... }
```

The OpenCode client already has full support for:
1. Fetching session details by ID
2. Fetching all messages for a session
3. Listing all sessions (with or without directory filter)

**Source:** `pkg/opencode/client.go:353-374, 625-646`

**Significance:** The infrastructure exists - only the CLI interface needs enhancement.

---

### Finding 4: Session titles follow predictable patterns for orch-managed sessions

**Evidence:**
```go
// shared.go:28-37
func extractBeadsIDFromTitle(title string) string {
    // Look for "[beads-id]" pattern
    if start := strings.LastIndex(title, "["); start != -1 {
        if end := strings.LastIndex(title, "]"); end != -1 && end > start {
            return strings.TrimSpace(title[start+1 : end])
        }
    }
    return ""
}
```

Orch-managed sessions have titles ending with `[beads-id]`. Non-orch sessions have freeform titles. This distinction can be used to label sessions as "tracked" vs "untracked" in status output.

**Source:** `cmd/orch/shared.go:28-37`

**Significance:** We can visually distinguish orch-managed from ad-hoc sessions.

---

## Synthesis

**Key Insights:**

1. **Gap is UI, not infrastructure** - The OpenCode API already supports all required operations (get session, get messages, list sessions). The limitation is purely in the CLI argument parsing and filtering logic.

2. **Two complementary fixes needed** - Users need both:
   - A way to tail specific sessions by ID (`orch tail --session`)
   - Visibility into all sessions in status (`orch status` showing untracked)

3. **Untracked sessions are legitimate** - Not all coding sessions need beads tracking. Interactive exploration, quick tests, and cross-project work are valid use cases for sessions without beads integration.

**Answer to Investigation Question:**

Orchestrators can inspect non-orch-spawned sessions by:
1. Adding `--session` flag to `orch tail` for direct session ID access
2. Showing untracked sessions in `orch status` (with visual distinction from tracked ones)

Both changes are low-risk, additive enhancements that preserve existing behavior.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode API `GetMessages(sessionID)` works for any valid session ID (verified: existing tail code uses it)
- ✅ OpenCode API `ListSessions("")` returns all in-memory sessions regardless of title format (verified: existing status code uses it)
- ✅ Session titles can be parsed to extract beads ID when present (verified: existing `extractBeadsIDFromTitle` function)

**What's untested:**

- ⚠️ Performance impact of showing all sessions in status (not benchmarked for high session counts)
- ⚠️ User experience of untracked session display format (no user feedback yet)

**What would change this:**

- If OpenCode API changes session listing behavior
- If untracked sessions become a source of confusion (might need opt-in flag)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Additive CLI enhancement** - Add `--session` flag to `orch tail` and show untracked sessions in `orch status`

**Why this approach:**
- Preserves all existing behavior (no breaking changes)
- Minimal code changes (additive flags and conditionals)
- Directly addresses the reported pain point
- Uses existing, tested infrastructure

**Trade-offs accepted:**
- `orch status` output becomes longer with untracked sessions
- Users must know session ID to use `--session` flag (can copy from OpenCode dashboard)

**Implementation sequence:**
1. Add `--session` flag to `orch tail` - highest value, most requested
2. Add untracked sessions to `orch status` - provides discovery mechanism
3. Add `--untracked` flag if needed to filter status output

### Alternative Approaches Considered

**Option B: Automatic session-to-beads mapping**
- **Pros:** Seamless experience, no manual ID copying
- **Cons:** Complex heuristics, potential false positives, requires title parsing assumptions
- **When to use instead:** If session ID copying becomes a significant friction point

**Option C: OpenCode dashboard deep-link integration**
- **Pros:** Single source of truth for session visibility
- **Cons:** Requires browser automation or dashboard API, adds external dependency
- **When to use instead:** If visual inspection is primary use case

**Rationale for recommendation:** Option A is the minimal change that solves the stated problem. Options B and C add complexity without proportional benefit for the current use case.

---

### Implementation Details

**What to implement first:**
1. `orch tail --session <session-id>` flag - core functionality
2. Untracked session visibility in `orch status` - discovery mechanism

**Things to watch out for:**
- ⚠️ Session IDs are ephemeral - don't persist them as identifiers
- ⚠️ Untracked sessions may belong to other projects - show directory in status

**Areas needing further investigation:**
- How to surface session-to-workspace mapping (future enhancement)

**Success criteria:**
- ✅ `orch tail --session ses_xxx` fetches messages for that session
- ✅ `orch status` shows sessions without beads IDs (labeled as untracked)
- ✅ Existing beads-ID-based workflows continue to work unchanged

---

## References

**Files Examined:**
- `cmd/orch/tail_cmd.go` - Current tail command implementation
- `cmd/orch/status_cmd.go` - Current status command implementation
- `cmd/orch/shared.go` - Helper functions for beads ID extraction
- `pkg/opencode/client.go` - OpenCode API client with session methods

**Commands Run:**
```bash
# Find OpenCode session API usage
grep -r "ListSessions\|GetSession\|GetMessages" --include="*.go"
```

**Related Artifacts:**
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Four-layer state architecture
- **Guide:** `.kb/guides/status.md` - Status command documentation

---

## Investigation History

**[2026-01-29]:** Investigation started
- Initial question: How can orchestrators inspect non-orch-spawned OpenCode sessions?
- Context: Users reported inability to use orch tail on sessions started outside orch spawn

**[2026-01-29]:** Code analysis completed
- Found that tail requires beads ID with no direct session access
- Found that status filters out sessions without beads IDs
- Confirmed OpenCode API supports direct session ID operations

**[2026-01-29]:** Implementation completed and verified
- Added `IsUntracked` field to AgentInfo struct  
- Added Phase 3 discovery for sessions without beads ID
- Updated filtering to exclude untracked unless --all
- Updated display to show "untracked" status and truncated session ID
- Verified: `orch status --all` shows untracked sessions, `orch tail --session` works
