<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch sessions` command suite (list, search, show) for querying OpenCode session history.

**Evidence:** All three commands tested successfully - list shows sessions from disk storage, search finds matches via API, show displays full message content.

**Knowledge:** OpenCode stores session metadata on disk but message content requires API access; hybrid approach (disk for listing, API for content) works well.

**Next:** Complete - commands are functional and tested. Run `make install` to deploy.

---

# Investigation: Orch Sessions Search Query Opencode

**Question:** How can we query and search OpenCode session history to find past insights and decisions?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode Storage Structure

**Evidence:** OpenCode stores sessions in a hierarchical structure:
- Sessions: `~/.local/share/opencode/storage/session/{projectID}/{sessionID}.json`
- Messages: `~/.local/share/opencode/storage/message/{sessionID}/{messageID}.json`

Session files contain full metadata (id, title, directory, timestamps, summary).
Message files contain only metadata (no actual message content).

**Source:** Manual exploration of `~/.local/share/opencode/storage/`

**Significance:** For listing sessions, we can walk the disk storage. For searching message content, we need the OpenCode API.

---

### Finding 2: OpenCode API Provides Message Content

**Evidence:** The existing `opencode.Client.GetMessages(sessionID)` returns full message content via the `/session/{id}/message` API endpoint. Messages include `Parts` with `Type="text"` containing searchable content.

**Source:** `pkg/opencode/client.go:475-496` and `pkg/opencode/types.go:77-125`

**Significance:** We can leverage the existing OpenCode client for search functionality without needing to decode disk storage.

---

### Finding 3: Hybrid Approach Works Well

**Evidence:** Implemented and tested:
- `orch sessions list` - Walks disk storage, shows 5 sessions in <1s
- `orch sessions search "SPAWN_CONTEXT"` - Found 3 matching sessions with snippets
- `orch sessions show ses_xxx` - Displayed full session with 53 messages

**Source:** End-to-end testing of implemented commands

**Significance:** The hybrid approach (disk for metadata, API for content) balances performance with functionality.

---

## Synthesis

**Key Insights:**

1. **Disk + API hybrid** - Using disk storage for session listing is fast and doesn't require OpenCode running, while API access for message content enables rich search functionality.

2. **Existing infrastructure reuse** - The `opencode.Client` already had `GetMessages` and `ListSessions` methods that could be leveraged.

3. **Simple search is effective** - Plain text search with optional regex handles the use case well without needing full-text indexing.

**Answer to Investigation Question:**

The session search capability was successfully implemented via `orch sessions {list,search,show}` commands. Sessions can be listed from disk storage (works offline), searched via API for message content, and individually viewed in full. The implementation reuses existing OpenCode client infrastructure and provides useful filtering options (date, directory, limit).

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch sessions list` returns sessions sorted by updated time (verified: listed 5 sessions)
- ✅ `orch sessions search` finds matching text in message content (verified: found "SPAWN_CONTEXT" in 3 sessions)
- ✅ `orch sessions show` displays full session details and messages (verified: showed 53 messages)
- ✅ Unit tests pass for sessions package (verified: `go test ./pkg/sessions/...` passes)

**What's untested:**

- ⚠️ Performance with thousands of sessions (not benchmarked, but disk walk should be fast)
- ⚠️ Regex search edge cases (basic patterns tested only)
- ⚠️ Date filtering edge cases around timezone boundaries

**What would change this:**

- If OpenCode changes storage format, disk parsing would need updates
- If API response format changes, message content extraction would need updates

---

## Implementation Recommendations

**Purpose:** The implementation is complete. Document the architecture for future reference.

### Implemented Approach ⭐

**Hybrid Disk + API** - Use disk storage for session metadata listing, API for message content search.

**Why this approach:**
- Disk walk is fast and works without OpenCode running
- API provides rich message content for search
- Reuses existing `opencode.Client` infrastructure

**Trade-offs accepted:**
- Search requires OpenCode to be running (acceptable - common case)
- No full-text index (acceptable - simple search is sufficient for finding past work)

**Implementation structure:**
1. `pkg/sessions/sessions.go` - Store, List, Search, Show methods
2. `cmd/orch/sessions.go` - CLI commands (list, search, show)
3. `pkg/sessions/sessions_test.go` - Unit tests

---

## References

**Files Created:**
- `pkg/sessions/sessions.go` - Session store and search implementation
- `pkg/sessions/sessions_test.go` - Unit tests
- `cmd/orch/sessions.go` - CLI commands

**Files Modified:**
- `cmd/orch/main.go` - Added `sessionsCmd` registration

**Commands Added:**
```bash
orch sessions list [--limit N] [--date YYYY-MM-DD] [--directory path]
orch sessions search [query] [--regex] [--case] [--limit N]
orch sessions show [session-id]
```

---

## Investigation History

**2025-12-27 19:30:** Investigation started
- Initial question: How to query OpenCode session history?
- Context: Orchestrator sessions contain valuable insights but aren't discoverable

**2025-12-27 19:45:** Storage structure explored
- Found disk storage format (metadata in JSON files)
- Found API returns full message content

**2025-12-27 20:00:** Implementation complete
- Created sessions package with List, Search, Show
- Added CLI commands
- All tests passing

**2025-12-27 20:10:** Investigation completed
- Status: Complete
- Key outcome: `orch sessions` command suite implemented and tested
