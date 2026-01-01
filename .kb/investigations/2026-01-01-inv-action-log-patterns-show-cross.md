<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Action-log patterns show cross-project noise because the global ~/.orch/action-log.jsonl file is loaded without project filtering.

**Evidence:** Code inspection shows patterns.LoadLog() returns all events; handlePatterns API and GenerateBehavioralPatternsContext had no project filtering.

**Knowledge:** Pattern filtering by project_dir at both API and spawn-context levels eliminates cross-project noise while preserving global patterns when desired.

**Next:** Close - fix implemented with tests passing, frontend updated to pass projectDir filter.

---

# Investigation: Action Log Patterns Show Cross-Project Noise

**Question:** Why do action-log patterns in the dashboard show patterns from other projects (cross-project noise)?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None - fix implemented
**Status:** Complete

---

## Findings

### Finding 1: Global Action-Log Without Project Filtering

**Evidence:** The action-log.jsonl file is stored globally at `~/.orch/action-log.jsonl` and contains actions from ALL projects. The `LoadLog()` function in `pkg/patterns/analyzer.go:150-185` reads all events without any project filtering.

**Source:** 
- `pkg/patterns/analyzer.go:134-145` - LogPath() returns global path
- `pkg/patterns/analyzer.go:147-185` - LoadLog() reads all events

**Significance:** This is the root cause - pattern detection considers all events regardless of which project they came from.

---

### Finding 2: API Endpoint Had No Project Filter

**Evidence:** The `/api/patterns` handler in `cmd/orch/serve.go:4005-4057` called `log.DetectPatterns()` without any filtering capability.

**Source:** `cmd/orch/serve.go:4005-4057`

**Significance:** Even if the frontend wanted project-specific patterns, there was no API mechanism to request them.

---

### Finding 3: Spawn Context Also Lacked Filtering

**Evidence:** `GenerateBehavioralPatternsContext` in `pkg/spawn/context.go:963-1019` loaded patterns globally, meaning SPAWN_CONTEXT.md could show warnings about files from unrelated projects.

**Source:** `pkg/spawn/context.go:963-1019`

**Significance:** Agents spawned in orch-go would see warnings about files in price-watch, creating confusion.

---

## Synthesis

**Key Insights:**

1. **Intentionally Global, Display-Time Filtering** - The action-log is intentionally global for cross-session learning, but surfacing should be project-aware. The fix filters at display time (API/dashboard, spawn context) rather than at storage time.

2. **Path-Based Matching** - Events can be filtered by checking if Target path or WorkspaceDir starts with the project directory. This simple prefix matching is effective.

3. **Graceful Fallback** - When a project has no patterns, the system falls back to global patterns rather than showing nothing.

**Answer to Investigation Question:**

Cross-project noise occurs because the global action-log was loaded and displayed without project filtering. The fix adds:
1. `?project=/path` query parameter to `/api/patterns` API
2. `DetectPatternsForProject(projectDir)` method in patterns package  
3. `GenerateBehavioralPatternsContextForProject(workspace, projectDir)` function
4. Frontend reactive fetch when project filter changes

---

## Structured Uncertainty

**What's tested:**

- ✅ eventMatchesProject correctly filters by target path (verified: unit tests pass)
- ✅ DetectPatternsForProject returns only matching patterns (verified: unit tests pass)
- ✅ API accepts ?project parameter (verified: code review, build success)
- ✅ Frontend patterns store accepts projectDir parameter (verified: TypeScript compiles)

**What's untested:**

- ⚠️ End-to-end behavior in actual dashboard (not run - servers not started)
- ⚠️ Performance impact of filtering large action logs (not benchmarked)

**What would change this:**

- Finding would be wrong if some events use relative paths instead of absolute paths
- Finding would be wrong if WorkspaceDir is stored differently than expected

---

## Implementation Recommendations

**Purpose:** Document the implemented fix.

### Implemented Approach

**Project-Scoped Pattern Filtering** - Filter patterns at display time using path prefix matching

**What was implemented:**
1. Added `eventMatchesProject()` helper in `pkg/patterns/analyzer.go`
2. Added `DetectPatternsForProject(projectDir)` in `pkg/patterns/analyzer.go`
3. Added `detectRepeatedEmptyReadsForProject()` and `detectRepeatedErrorsForProject()` methods
4. Updated `/api/patterns` handler to accept `?project` query parameter
5. Updated patterns TypeScript store to accept `projectDir` parameter
6. Updated `+page.svelte` to derive `currentProjectDir` from `projectFilter` and pass to patterns fetch
7. Updated `NeedsAttention` component to accept and use `projectDir` prop
8. Updated `GenerateBehavioralPatternsContextForProject()` in spawn context

**Trade-offs:**
- Filtering happens on every request (no caching) - acceptable for current scale
- Fallback to global patterns when project has none - maintains utility

---

## References

**Files Modified:**
- `pkg/patterns/analyzer.go` - Added project filtering methods
- `pkg/patterns/analyzer_test.go` - Added tests for project filtering
- `cmd/orch/serve.go` - Updated handlePatterns to accept project param
- `pkg/spawn/context.go` - Added GenerateBehavioralPatternsContextForProject
- `web/src/lib/stores/patterns.ts` - Added projectDir parameter
- `web/src/routes/+page.svelte` - Added currentProjectDir derivation and reactive fetch
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Added projectDir prop

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/go build ./...

# Test patterns package
/opt/homebrew/bin/go test ./pkg/patterns/... -v

# Test spawn package
/opt/homebrew/bin/go test ./pkg/spawn/... -run TestGenerateBehavioralPatterns -v
```

---

## Investigation History

**2026-01-01 08:00:** Investigation started
- Initial question: Why do action-log patterns show cross-project noise?
- Context: Bug identified in prior investigation (.kb/investigations/2025-12-30-inv-investigate-recent-bugs-attention-panel.md)

**2026-01-01 08:15:** Root cause identified
- Global action-log loaded without filtering
- API had no project parameter
- Spawn context also unfiltered

**2026-01-01 09:00:** Fix implemented
- Added filtering at patterns, API, spawn, and frontend layers
- Tests passing
- Ready for review
