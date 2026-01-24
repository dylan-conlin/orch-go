## Summary (D.E.K.N.)

**Delta:** Added /api/errors endpoint for error pattern analysis - provides total errors, time-windowed counts (24h/7d), recent errors, and recurring patterns with remediation suggestions.

**Evidence:** All 8 new tests pass; endpoint returns valid JSON with error events from events.jsonl.

**Knowledge:** Error events use two types: session.error (with error message) and agent.abandoned (with beads_id, reason, workspace); pattern detection groups similar errors by normalized message.

**Next:** Close - implementation complete with tests.

**Confidence:** High (90%) - All tests pass, follows established API patterns from /api/gaps and /api/agentlog.

---

# Investigation: Add /api/errors Endpoint for Error Pattern Analysis

**Question:** How to implement an /api/errors endpoint that provides error pattern analysis from agent events?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent (feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Error Events Structure in events.jsonl

**Evidence:** Two error-related event types exist:
- `session.error` with `Data["error"]` containing error message
- `agent.abandoned` with `Data["beads_id"]`, `Data["reason"]`, `Data["agent_id"]`, `Data["workspace_path"]`

**Source:** ~/.orch/events.jsonl, pkg/events/logger.go:19-20

**Significance:** These provide the raw data for error pattern analysis. Agent abandoned events are the primary source of actionable errors.

---

### Finding 2: Existing API Pattern for Event Analysis

**Evidence:** /api/gaps and /api/agentlog provide similar event-based analysis:
- /api/gaps reads from gap-tracker.json
- /api/agentlog reads from events.jsonl with time-based filtering

**Source:** cmd/orch/serve.go:1722-1779 (handleGaps), 968-1004 (handleAgentlog)

**Significance:** Established pattern to follow - read JSONL, filter by event type, provide summary statistics and recent items.

---

### Finding 3: Error Pattern Detection Approach

**Evidence:** Implemented pattern detection by:
1. Normalizing error messages (truncate to 100 chars, trim whitespace)
2. Counting occurrences of each normalized pattern
3. Tracking affected beads IDs per pattern
4. Providing remediation suggestions based on keyword matching

**Source:** cmd/orch/serve.go (normalizeErrorMessage, suggestRemediation functions)

**Significance:** Enables dashboard to show recurring errors and suggest remediation actions.

---

## Implementation Details

**API Response Structure (ErrorsAPIResponse):**
- `total_errors` - Total error events in events.jsonl
- `errors_last_24h` - Errors in last 24 hours
- `errors_last_7d` - Errors in last 7 days
- `abandoned_count` - Total agent.abandoned events
- `session_errors` - Total session.error events
- `recent_errors` - Last 20 error events (most recent first)
- `patterns` - Recurring error patterns (2+ occurrences) with remediation suggestions
- `by_type` - Breakdown by error type

**Helper Functions:**
- `extractSkillFromAgentID()` - Maps agent ID prefixes (og-feat, og-debug, etc.) to skill names
- `normalizeErrorMessage()` - Truncates/trims messages for pattern matching
- `containsString()` - String slice membership check
- `suggestRemediation()` - Keyword-based remediation suggestions

**Tests Added:**
- TestHandleErrorsMethodNotAllowed
- TestHandleErrorsJSONResponse
- TestErrorsAPIResponseJSONFormat
- TestExtractSkillFromAgentID
- TestNormalizeErrorMessage
- TestContainsString
- TestSuggestRemediation
- TestHandleErrorsWithTestData

---

## References

**Files Examined:**
- cmd/orch/serve.go - Main API server implementation
- pkg/events/logger.go - Event types and logging

**Commands Run:**
```bash
# View recent events
cat ~/.orch/events.jsonl | tail -50

# Run tests
go test ./cmd/orch/... -v -run "Error"
```
