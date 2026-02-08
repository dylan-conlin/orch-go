## Summary (D.E.K.N.)

**Delta:** Integrated hotspot detection into orch spawn to warn when tasks target high-churn areas (5+ fix commits in 28 days).

**Evidence:** Tests pass for path extraction, hotspot matching, and warning formatting. Integration added to runSpawnWithSkill() that runs hotspot analysis on spawn.

**Knowledge:** Task descriptions can be parsed for file paths using regex, then cross-referenced with git history to surface areas needing architect review.

**Next:** None - implementation complete, ready for use.

---

# Investigation: Integrate Hotspot Detection Into Orch

**Question:** How to integrate hotspot detection into orch spawn so warnings appear when spawning to high-churn areas?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Task descriptions contain parseable file paths

**Evidence:** Created regex pattern that extracts paths like `cmd/orch/spawn.go`, `pkg/daemon/`, and paths in quotes from task descriptions. Test cases verify extraction works for common task formats.

**Source:** `cmd/orch/hotspot.go:394-433` - `extractPathsFromTask()` function

**Significance:** Enables automatic detection of which files/areas a task targets, without requiring explicit user input.

---

### Finding 2: Hotspot matching requires different strategies per type

**Evidence:** Fix-density hotspots need exact file or directory matching, while investigation-cluster hotspots match on topic keywords in paths. Implementation uses type-specific logic.

**Source:** `cmd/orch/hotspot.go:439-464` - `matchPathToHotspots()` function

**Significance:** Multi-strategy matching improves accuracy by handling file-based and topic-based hotspots appropriately.

---

### Finding 3: Integration is non-blocking by design

**Evidence:** Warning prints to stderr but doesn't block spawn. This matches the design spec: "Warning only, not blocking."

**Source:** `cmd/orch/spawn_cmd.go:516-525` - hotspot check integration point

**Significance:** Orchestrators get information to decide whether to proceed or spawn architect first, without being blocked.

---

## Synthesis

**Key Insights:**

1. **Path extraction is sufficient for file-based hotspots** - No need for complex code analysis; regex-based path extraction captures the vast majority of file references in task descriptions.

2. **Warning placement matters** - By running the check early in spawn workflow (after concurrency check, before main logic), the warning appears before the user sees spawn output.

3. **Reusing existing hotspot analysis** - The `RunHotspotCheckForSpawn()` function reuses `analyzeFixCommits()` and `analyzeInvestigationClusters()` from the existing `orch hotspot` command.

**Answer to Investigation Question:**

Hotspot detection is integrated by:
1. Running hotspot analysis on spawn (reusing existing `analyzeFixCommits` and `analyzeInvestigationClusters`)
2. Extracting file paths from task description via regex
3. Matching extracted paths against detected hotspots
4. Printing a warning box with architect recommendation if matches found

The warning is non-blocking, allowing the spawn to proceed while surfacing the information.

---

## Structured Uncertainty

**What's tested:**

- ✅ Path extraction works for common formats (verified: 7 test cases pass)
- ✅ Hotspot matching handles exact, directory, and topic matches (verified: 4 test cases pass)
- ✅ Warning formatting produces readable output (verified: test confirms structure)
- ✅ Integration doesn't break existing spawn tests (verified: `go test ./cmd/orch/...` passes)

**What's untested:**

- ⚠️ Performance impact on spawn (not benchmarked - git log analysis adds latency)
- ⚠️ Edge cases with very large repos (not tested at scale)
- ⚠️ Whether orchestrators will act on warnings (behavioral assumption)

**What would change this:**

- If git analysis is too slow, may need caching
- If false positive rate is high, may need tunable thresholds
- If warnings are ignored, may need gating (blocking) mode

---

## Implementation Recommendations

**Purpose:** Document the completed implementation for future reference.

### Implemented Approach ⭐

**Task path extraction + hotspot cross-reference** - Extract file paths from task description, cross-reference with git history analysis, warn if matches found.

**Why this approach:**
- Simple and effective - regex pattern covers most task formats
- Reuses existing code - no duplication of hotspot analysis
- Non-blocking - respects orchestrator autonomy

**Trade-offs accepted:**
- Some latency from git analysis (acceptable for spawn workflow)
- May miss hotspots not referenced in task text (acceptable - explicit is better)

---

### Implementation Details

**What was implemented:**
- `extractPathsFromTask()` - regex-based path extraction from task text
- `matchPathToHotspots()` - matching with type-specific strategies
- `checkSpawnHotspots()` - orchestrates extraction and matching
- `formatHotspotWarning()` - generates warning box with architect recommendation
- `RunHotspotCheckForSpawn()` - main entry point, runs analysis and returns warning

**Integration point:**
- Added to `runSpawnWithSkill()` in `spawn_cmd.go` after concurrency check
- Warning prints to stderr, doesn't block spawn

**Success criteria:**
- ✅ Warning appears when spawning to hotspot area
- ✅ Warning recommends architect spawn
- ✅ Spawn proceeds normally after warning

---

## References

**Files Modified:**
- `cmd/orch/hotspot.go` - Added spawn integration functions
- `cmd/orch/hotspot_test.go` - Added tests for new functions
- `cmd/orch/spawn_cmd.go` - Added hotspot check to spawn workflow

**Commands Run:**
```bash
# Run tests
go test -v -run "TestExtractPaths|TestMatchPath|TestCheckSpawn|TestFormatHotspot" ./cmd/orch/

# Verify all tests pass
go test ./cmd/orch/...
```

**Related Artifacts:**
- **Design:** `.kb/investigations/2026-01-04-design-patch-density-architect-escalation.md` - Original design investigation
- **Epic:** `orch-go-yz3d` - Parent epic for hotspot detection

---

## Investigation History

**2026-01-04:** Investigation started
- Initial question: How to integrate hotspot detection into spawn?
- Context: Design investigation recommended spawn integration as Phase 2

**2026-01-04:** Implementation complete
- Status: Complete
- Key outcome: Hotspot warnings integrated into spawn workflow, tests passing
