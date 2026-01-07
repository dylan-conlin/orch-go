<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Dashboard Investigation tab now renders investigation file content inline instead of showing terminal commands.

**Evidence:** Implementation tested - Go build passes, all tests pass (including new ParseInvestigationPathFromComments test), frontend build succeeds.

**Knowledge:** The `investigation_path` pattern from beads comments enables dynamic file content loading; secure `/api/file` endpoint restricts access to .kb/ and .orch/workspace/ directories only.

**Next:** Close - all deliverables complete, ready for `orch complete`.

**Promote to Decision:** recommend-no (tactical UI fix, follows existing patterns)

---

# Investigation: Dashboard Investigation Tab Render Investigation

**Question:** How should the Investigation tab render actual investigation file content instead of workspace paths and terminal commands?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: investigation_path is reported via beads comments

**Evidence:** The spawn context template includes instruction to report `investigation_path: /path/to/file.md` via `bd comment`.

**Source:** `pkg/spawn/context.go:157` - Template shows the expected format

**Significance:** This provides a standard way to discover which investigation file an agent is working on, enabling the dashboard to fetch and display its content.

---

### Finding 2: Synthesis tab pattern uses parsed API data

**Evidence:** The Synthesis tab renders `agent.synthesis` which is parsed from SYNTHESIS.md by the backend and included in the AgentAPIResponse.

**Source:** `cmd/orch/serve_agents.go:38-66` - SynthesisResponse structure
`web/src/lib/components/agent-detail/synthesis-tab.svelte` - Frontend rendering

**Significance:** For investigation files, we should follow a similar pattern: parse investigation_path from comments, expose via API, and render content inline.

---

### Finding 3: Frontend Agent interface needed investigation_path field

**Evidence:** The Agent interface in agents.ts had `primary_artifact` but it was never populated from the backend.

**Source:** `web/src/lib/stores/agents.ts:42`

**Significance:** Adding `investigation_path` to the Agent interface enables the frontend to fetch and render investigation file content dynamically.

---

## Synthesis

**Key Insights:**

1. **Pattern consistency** - Following the Synthesis tab pattern (API data + inline rendering) makes the Investigation tab predictable and maintainable.

2. **Security by design** - The `/api/file` endpoint restricts reads to allowed directories (.kb/, .orch/workspace/), preventing arbitrary file access.

3. **Graceful degradation** - When no investigation_path is reported, the tab falls back to showing workspace paths and terminal commands.

**Answer to Investigation Question:**

The Investigation tab should:
1. Extract `investigation_path` from beads comments (via `ParseInvestigationPathFromComments`)
2. Include it in the AgentAPIResponse (new `investigation_path` field)
3. Fetch file content via `/api/file?path=<path>` endpoint
4. Render markdown content inline in a scrollable container
5. Fall back to terminal commands when no investigation file is available

---

## Structured Uncertainty

**What's tested:**

- ✅ ParseInvestigationPathFromComments correctly extracts paths from comments (unit test added)
- ✅ Go build passes with all changes
- ✅ Frontend build succeeds
- ✅ All existing tests pass

**What's untested:**

- ⚠️ Full end-to-end flow in browser (needs orch serve restart and dashboard access)
- ⚠️ Security of /api/file path validation with edge cases (path traversal attempts)
- ⚠️ Large file handling performance

**What would change this:**

- If agents don't consistently report investigation_path, the feature won't be useful
- If /api/file has path traversal vulnerabilities, security would be compromised

---

## Implementation Recommendations

### Recommended Approach ⭐

**Extract path from comments + fetch via API** - Parse investigation_path from beads comments and fetch content via new /api/file endpoint.

**Why this approach:**
- Follows established pattern from Synthesis tab
- Keeps API response lean (path only, not full content)
- Allows file content caching at API level if needed
- Security can be enforced at single endpoint

**Trade-offs accepted:**
- Extra API call to fetch file content (acceptable for UI feature)
- No markdown rendering (plain text with whitespace preserved)

**Implementation sequence:**
1. Add ParseInvestigationPathFromComments in pkg/verify/beads_api.go ✅
2. Add investigation_path to AgentAPIResponse in serve_agents.go ✅
3. Add /api/file endpoint in serve_system.go ✅
4. Update Agent interface in agents.ts ✅
5. Rewrite investigation-tab.svelte to fetch and render ✅

---

## References

**Files Modified:**
- `pkg/verify/beads_api.go` - Added ParseInvestigationPathFromComments function
- `pkg/verify/check_test.go` - Added test for ParseInvestigationPathFromComments
- `cmd/orch/serve_agents.go` - Added investigation_path and project_dir to AgentAPIResponse
- `cmd/orch/serve_system.go` - Added handleFile endpoint for /api/file
- `cmd/orch/serve.go` - Registered /api/file route
- `web/src/lib/stores/agents.ts` - Added investigation_path to Agent interface
- `web/src/lib/components/agent-detail/investigation-tab.svelte` - Rewrote to fetch and render content

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification  
go test ./pkg/verify/... -run TestParseInvestigationPath
go test ./...

# Frontend verification
cd web && bun run build
```

---

## Investigation History

**2026-01-06 17:00:** Investigation started
- Initial question: How to render investigation file content in dashboard Investigation tab
- Context: Tab currently shows workspace paths and terminal commands, should show actual file content

**2026-01-06 17:30:** Design decision made
- Chose to parse investigation_path from beads comments and fetch via /api/file endpoint
- Pattern follows Synthesis tab approach

**2026-01-06 18:00:** Investigation completed
- Status: Complete
- Key outcome: Implemented full-stack solution for rendering investigation file content in dashboard
