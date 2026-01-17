# SYNTHESIS: Dashboard Follow-Orchestrator Multi-Project Filtering

**Agent:** og-debug-dashboard-follow-orchestrator-14jan-2226
**Issue:** orch-go-7dtqe
**Skill:** systematic-debugging
**Date:** 2026-01-14
**Duration:** ~45 minutes

---

## TLDR

Fixed incomplete multi-project filtering that allowed price-watch agents to show in orch-go dashboard. Root cause: (1) frontend buildFilterQueryString ignored includedProjects array, (2) backend filterByProject only accepted single filter string, (3) filtering logic used ProjectDir (spawner location) instead of Project field (target project).

**Outcome:** success

---

## What Was Built

### 1. Frontend: Multi-Project Serialization

**File:** `web/src/lib/stores/context.ts`
**Change:** Updated `buildFilterQueryString` to serialize `includedProjects` array as comma-separated values in the `project` URL parameter.

**Before:**
```typescript
if (state.project) {
    params.set('project', state.project);
}
```

**After:**
```typescript
if (state.includedProjects && state.includedProjects.length > 0) {
    params.set('project', state.includedProjects.join(','));
} else if (state.project) {
    params.set('project', state.project);
}
```

**Why:** The orchestrator context tracks 6 included projects (orch-go, orch-cli, beads, kb-cli, orch-knowledge, opencode) but only the primary project was being passed to the API.

---

### 2. Backend: Comma-Separated Parsing

**File:** `cmd/orch/serve_filter.go`
**Change:** Updated `parseProjectFilter` to split on comma and return `[]string` instead of `string`.

**Signature change:**
```go
// Before: func parseProjectFilter(r *http.Request) string
func parseProjectFilter(r *http.Request) []string
```

**Implementation:** Splits param value on comma, trims whitespace, filters empty values.

---

### 3. Backend: Array-Based Filtering

**File:** `cmd/orch/serve_filter.go`
**Change:** Updated `filterByProject` to accept `[]string` filters and return true if projectDir matches ANY filter.

**Signature change:**
```go
// Before: func filterByProject(projectDir, filter string) bool
func filterByProject(projectDir string, filters []string) bool
```

**Logic:** Iterates through filters, checks both full path and extracted project name.

---

### 4. Backend: Filter by Project Name Not ProjectDir

**File:** `cmd/orch/serve_agents.go:983-1006`
**Change:** Use agent.Project field for filtering instead of agent.ProjectDir.

**Critical insight:** Cross-project agents spawned with `--workdir` have:
- `ProjectDir`: Spawner's working directory (e.g., `/Users/dylan/orch-go`)
- `Project`: Target project name (e.g., `"pw"`)

Filtering by ProjectDir caused false matches - all agents spawned from orch-go would match `orch-go` filter regardless of their actual target project.

**Solution:** Filter by `agent.Project` with fallback to `extractProjectName(agent.ProjectDir)` for agents without Project field.

---

### 5. Test Coverage

**File:** `cmd/orch/serve_filter_test.go`
**Added:** Test cases for multi-project scenarios:
- Comma-separated parsing (`"orch-go,orch-cli,beads"` → `[]string{"orch-go", "orch-cli", "beads"}`)
- Whitespace handling (`"orch-go, orch-cli , beads"`)
- Empty value filtering (`"orch-go,,beads"`)
- Multi-filter matching (matches ANY filter in array)

**All tests passing:** 15 test cases covering single and multi-project filtering.

---

## Decisions Made

### Use Project Field Not ProjectDir for Filtering

**Decision:** Filter agents by `agent.Project` instead of `agent.ProjectDir`.

**Rationale:**
- ProjectDir reflects where the agent's workspace is stored (spawner's cwd)
- Project reflects what the agent is actually working on (target project)
- For cross-project coordination, we want to filter by "what's being worked on" not "where it was spawned from"

**Trade-off:** Requires agents to have Project field populated. Fallback to ProjectDir extraction handles legacy agents.

---

### Comma-Separated URL Parameter vs Separate Params

**Decision:** Use single `?project=a,b,c` instead of multiple `?project=a&project=b&project=c`.

**Rationale:**
- RESTful standard pattern
- No API schema change (reuses existing param name)
- Simple frontend serialization (array.join(','))
- Simple backend parsing (strings.Split)

**Trade-off:** Can't have project names with commas (acceptable - our naming is kebab-case).

---

## Challenges & Solutions

### Challenge: URL Encoding in Tests

**Issue:** Test with `"?project=orch-go, orch-cli , beads"` failed with `malformed HTTP version` error.

**Root cause:** httptest.NewRequest interprets unencoded spaces as HTTP version separator.

**Solution:** URL-encode spaces in test query strings (`%20`).

---

### Challenge: Filtering Wrong Field

**Issue:** Initial implementation filtered by ProjectDir, which still showed pw agents.

**Root cause:** Cross-project agents have ProjectDir pointing to spawner location, not target.

**Solution:** Switch to filtering by Project field (discovered via curl smoke test).

---

## Follow-Up Items

None - issue is fully resolved.

---

## Files Changed

1. `web/src/lib/stores/context.ts` - Frontend filter serialization
2. `cmd/orch/serve_filter.go` - Backend parsing and filtering logic
3. `cmd/orch/serve_agents.go` - Filter application using Project field
4. `cmd/orch/serve_filter_test.go` - Test coverage for multi-project scenarios
5. `.kb/investigations/2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md` - Investigation artifact

---

## Verification

### Smoke Tests Performed

```bash
# Test 1: Price-watch agent excluded from orch-go filter
curl 'https://localhost:3348/api/agents?project=orch-go,orch-cli,beads&since=12h' -k
# ✅ Result: No pw agents in response

# Test 2: Price-watch agent included in pw filter
curl 'https://localhost:3348/api/agents?project=pw&since=12h' -k
# ✅ Result: pw-feat-implement-material-category agent present
```

### Unit Tests

```bash
go test ./cmd/orch -v -run "TestParseProjectFilter|TestFilterByProjectDir"
# ✅ PASS: 15/15 tests passing
```

---

## Next Actions

**For orchestrator:**
1. Run `orch complete orch-go-7dtqe` to verify and close issue
2. No follow-up work needed - fix is complete and tested

---

## Session Metadata

**Investigation artifact:** `.kb/investigations/2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md`
**Commits:** 1 (all changes in single atomic commit)
**Tests added:** 9 new test cases for multi-project filtering
**Smoke tests:** 2 manual curl tests confirming fix
