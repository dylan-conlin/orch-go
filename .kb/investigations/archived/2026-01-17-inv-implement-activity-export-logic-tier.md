<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Activity export logic (Tier 2 persistence) is fully implemented in pkg/activity with integration in orch complete and fallback loading in serve_agents.

**Evidence:** All 5 tests pass; ExportToWorkspace exports on completion; handleSessionMessages falls back to ACTIVITY.json when OpenCode API fails.

**Knowledge:** Two-tier persistence architecture works: OpenCode API for live data, ACTIVITY.json for archival. Export happens post-verification, pre-archive.

**Next:** Close - implementation complete, no further action needed.

**Promote to Decision:** recommend-no - This is tactical implementation of already-decided architecture (Option C from gy1o4.1.5).

---

# Investigation: Implement Activity Export Logic (Tier 2 Persistence)

**Question:** Is the activity export logic for Tier 2 persistence fully implemented and integrated into the orch complete workflow?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker Agent (orch-go-gy1o4.1.7)
**Phase:** Complete
**Next Step:** None - implementation verified complete
**Status:** Complete

**Patches-Decision:** None
**Extracted-From:** Task orch-go-gy1o4.1.7 (from epic gy1o4.1)
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Core Export Package Implemented

**Evidence:** `pkg/activity/export.go` contains three key functions:
- `ExportToWorkspace(sessionID, workspacePath, serverURL)` - fetches messages from OpenCode API, transforms to SSE format, writes ACTIVITY.json
- `TransformMessages(sessionID, messages)` - converts OpenCode messages to SSE-compatible format (maps tool-invocation→tool, filters invalid types)
- `LoadFromWorkspace(workspacePath)` - reads and parses ACTIVITY.json, returns nil for missing files

**Source:**
- `pkg/activity/export.go:60-103` (ExportToWorkspace)
- `pkg/activity/export.go:107-158` (TransformMessages)
- `pkg/activity/export.go:164-181` (LoadFromWorkspace)

**Significance:** The core activity persistence logic exists and follows the SSE-compatible format defined in the design investigation. Events can be loaded from ACTIVITY.json and seamlessly merged with live SSE data.

---

### Finding 2: Integration in orch complete

**Evidence:** `cmd/orch/complete_cmd.go` lines 922-938 call activity export:
```go
// Export activity to ACTIVITY.json for archival (Tier 2 persistence)
// This is done BEFORE deleting the session (needs API access) and BEFORE archiving.
if workspacePath != "" && !isOrchestratorSession {
    sessionFile := filepath.Join(workspacePath, ".session_id")
    if data, err := os.ReadFile(sessionFile); err == nil {
        sessionID := strings.TrimSpace(string(data))
        if sessionID != "" {
            if activityPath, err := activity.ExportToWorkspace(sessionID, workspacePath, serverURL); err != nil {
                fmt.Fprintf(os.Stderr, "Warning: failed to export activity: %v\n", err)
            } else if activityPath != "" {
                fmt.Printf("Exported activity: %s\n", filepath.Base(activityPath))
            }
        }
    }
}
```

**Source:** `cmd/orch/complete_cmd.go:922-938`

**Significance:** Export is correctly positioned in the completion flow:
1. After verification (only exports for agents that pass gates)
2. Before session deletion (needs OpenCode API access)
3. Before archival (writes to active workspace, moves with archive)
4. Non-fatal errors (archival is supplementary)

---

### Finding 3: Fallback Loading in serve_agents

**Evidence:** `cmd/orch/serve_agents.go` implements fallback loading at lines 1485-1502:
```go
if err != nil {
    // OpenCode API failed (session may be deleted/cleaned up).
    // Fall back to ACTIVITY.json if available in the workspace.
    projectDir, _ := os.Getwd()
    workspacePath := findWorkspaceBySessionID(projectDir, sessionID)
    if workspacePath != "" {
        if events := loadActivityFromWorkspace(workspacePath); events != nil {
            // Successfully loaded from ACTIVITY.json
            w.Header().Set("Content-Type", "application/json")
            if encErr := json.NewEncoder(w).Encode(events); encErr != nil {
                http.Error(w, fmt.Sprintf("Failed to encode events: %v", encErr), http.StatusInternalServerError)
            }
            return
        }
    }
    // No fallback available, return original error
    http.Error(w, fmt.Sprintf("Failed to fetch messages: %v", err), http.StatusInternalServerError)
    return
}
```

Supporting functions:
- `findWorkspaceBySessionID()` (lines 1396-1436) - searches both active and archived workspaces
- `loadActivityFromWorkspace()` (lines 1438-1453) - reads and parses ACTIVITY.json

**Source:** `cmd/orch/serve_agents.go:1485-1502, 1396-1453`

**Significance:** Dashboard can load activity for completed/archived agents even after OpenCode session is deleted. The fallback searches both active and archived workspaces.

---

### Finding 4: Tests Comprehensive and Passing

**Evidence:** `pkg/activity/export_test.go` contains 5 tests:
1. `TestTransformMessages` - verifies message transformation (text, tool-invocation→tool, reasoning)
2. `TestTransformMessages_FiltersInvalidTypes` - verifies invalid types excluded
3. `TestLoadFromWorkspace_FileNotExists` - verifies nil,nil return for missing file
4. `TestLoadFromWorkspace_ValidFile` - verifies correct parsing
5. `TestLoadFromWorkspace_InvalidJSON` - verifies error handling

All tests pass:
```
=== RUN   TestTransformMessages
--- PASS: TestTransformMessages (0.00s)
=== RUN   TestTransformMessages_FiltersInvalidTypes
--- PASS: TestTransformMessages_FiltersInvalidTypes (0.00s)
=== RUN   TestLoadFromWorkspace_FileNotExists
--- PASS: TestLoadFromWorkspace_FileNotExists (0.00s)
=== RUN   TestLoadFromWorkspace_ValidFile
--- PASS: TestLoadFromWorkspace_ValidFile (0.00s)
=== RUN   TestLoadFromWorkspace_InvalidJSON
--- PASS: TestLoadFromWorkspace_InvalidJSON (0.00s)
PASS
```

**Source:** `pkg/activity/export_test.go`, `go test ./pkg/activity/... -v`

**Significance:** Core logic is unit tested. Edge cases (missing files, invalid JSON, type filtering) are covered.

---

## Synthesis

**Key Insights:**

1. **Two-Tier Architecture Implemented** - Tier 1 (OpenCode API) serves live data; Tier 2 (ACTIVITY.json) provides archival. The serve_agents fallback bridges them seamlessly.

2. **Correct Integration Timing** - Export happens post-verification, pre-archive in orch complete. This ensures only verified completions get archived activity, and the file travels with the workspace to archived/.

3. **SSE-Compatible Format** - ACTIVITY.json uses the same format as live SSE events, enabling the dashboard to merge historical and live data without transformation.

**Answer to Investigation Question:**

Yes, the activity export logic is fully implemented. The pkg/activity package provides ExportToWorkspace and LoadFromWorkspace. Integration in orch complete exports on agent completion. The serve_agents handleSessionMessages endpoint falls back to ACTIVITY.json when OpenCode API fails. All tests pass.

---

## Structured Uncertainty

**What's tested:**

- ✅ Message transformation works correctly (verified: TestTransformMessages)
- ✅ Invalid types filtered out (verified: TestTransformMessages_FiltersInvalidTypes)
- ✅ Missing files return nil,nil (verified: TestLoadFromWorkspace_FileNotExists)
- ✅ Valid ACTIVITY.json parses correctly (verified: TestLoadFromWorkspace_ValidFile)
- ✅ Invalid JSON returns error (verified: TestLoadFromWorkspace_InvalidJSON)

**What's untested:**

- ⚠️ End-to-end flow (export → archive → load) not integration tested
- ⚠️ Performance with very large sessions (1000+ events) not benchmarked
- ⚠️ Dashboard UI fallback behavior not visually verified

**What would change this:**

- If OpenCode message format changes, TransformMessages may need updates
- If dashboard requires different event format, ACTIVITY.json structure may need versioning

---

## References

**Files Examined:**
- `pkg/activity/export.go` - Core export and load functions
- `pkg/activity/export_test.go` - Unit tests
- `cmd/orch/complete_cmd.go` - orch complete integration
- `cmd/orch/serve_agents.go` - Fallback loading in API endpoint

**Commands Run:**
```bash
# Run activity package tests
go test ./pkg/activity/... -v
# Result: 5 tests pass
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-activity-feed-persistence-option.md` - Design for Option C (Black Box Recorder)
- **Issue:** `orch-go-gy1o4.1.7` - This implementation task

---

## Investigation History

**2026-01-17 19:30:** Investigation started
- Initial question: Is activity export logic fully implemented?
- Context: Picking up partially completed work from previous agent

**2026-01-17 19:35:** Findings complete
- Found implementation in pkg/activity/export.go
- Found integration in orch complete
- Found fallback loading in serve_agents
- All tests pass

**2026-01-17 19:40:** Investigation completed
- Status: Complete
- Key outcome: Implementation verified complete, no further code changes needed
