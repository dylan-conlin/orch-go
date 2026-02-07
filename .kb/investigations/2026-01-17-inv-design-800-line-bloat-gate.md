<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** 800-line bloat gate should be a third hotspot type in `orch hotspot`, not a separate system in `orch learn`.

**Evidence:** Existing `orch hotspot` has fix-density and investigation-cluster types; adding bloat-size as third type follows established patterns. `.kb/models/extract-patterns.md` explicitly states "800-line gate informs Hotspot Detection in orch learn."

**Knowledge:** Bloat is a coherence signal (like fix-density), not a context gap (like learning.go tracks). Different systems serve different purposes: hotspot = static code analysis, learn = context gap tracking during spawns.

**Next:** Implement `bloat-size` hotspot type in `cmd/orch/hotspot.go` with severity thresholds.

**Promote to Decision:** Issue created: orch-go-21084 (bloat control decision)

---

# Investigation: Design 800-Line Bloat Gate

**Question:** How should `orch learn` detect and respond to files exceeding 800 lines (the bloat threshold)?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implement feat-049 (bloat-size hotspot type)
**Status:** Complete

---

## Findings

### Finding 1: Bloat is a coherence signal, not a context gap

**Evidence:**
- `.kb/models/extract-patterns.md:66-68`: "800 lines is the heuristic limit where 'Context Noise' begins to degrade agent reasoning. When a file hits this limit, it triggers a Sub-domain Extraction."
- `~/.kb/principles.md` (Coherence Over Patches): "When fixes accumulate in the same area, the problem isn't insufficient fixing - it's a missing coherent model."
- Current codebase has 28 files exceeding 800 lines, with spawn_cmd.go at 2380 lines.

**Source:**
- `.kb/models/extract-patterns.md:64-68`
- `~/.kb/principles.md` lines 422-463
- `find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | xargs wc -l | sort -rn`

**Significance:** Bloat is about code coherence (too much in one place), not about missing knowledge (what `orch learn` tracks). This means bloat detection belongs with `orch hotspot` (which detects fix-density and investigation-cluster), not with gap tracking in learning.go.

---

### Finding 2: orch hotspot already has extensible architecture for new detection types

**Evidence:**
- `cmd/orch/hotspot.go` defines `Hotspot` struct with `Type` field (currently "fix-density" or "investigation-cluster")
- `HotspotReport` aggregates multiple hotspot types in `Hotspots []Hotspot`
- `analyzeFixCommits()` and `analyzeInvestigationClusters()` are independent analyzers that feed into the same report
- API endpoint at `cmd/orch/serve_hotspot.go` exposes hotspots to dashboard

**Source:**
- `cmd/orch/hotspot.go:72-80` (Hotspot struct)
- `cmd/orch/hotspot.go:94-138` (runHotspot aggregating multiple analyzers)
- `cmd/orch/serve_hotspot.go:1-65` (API handler)

**Significance:** Adding a third analyzer (`analyzeBloatFiles()`) follows the established pattern. No new infrastructure needed - just a new function that returns `[]Hotspot` with `Type: "bloat-size"`.

---

### Finding 3: Existing exclusion patterns can be reused

**Evidence:**
- `cmd/orch/hotspot.go:28-33` defines `defaultExclusions` for data/config files
- `shouldCountFileWithExclusions()` (lines 233-258) handles test files, generated files, documentation, and config files
- These exclusions are appropriate for bloat detection too (test files are expected to be long)

**Source:**
- `cmd/orch/hotspot.go:28-33` (defaultExclusions)
- `cmd/orch/hotspot.go:233-258` (shouldCountFileWithExclusions)

**Significance:** No need to reinvent exclusion logic. Same patterns that determine "should this file count for fix-density" also determine "should this file count for bloat."

---

### Finding 4: Code extraction patterns already document remediation

**Evidence:**
- `.kb/guides/code-extraction-patterns.md` provides complete workflow for Go, Svelte, and TypeScript extraction
- Pattern: Extract shared utilities first → Domain handlers → Sub-domain infrastructure
- Target size: 300-800 lines per file after extraction
- Line count benchmarks show 40-70% reductions are achievable

**Source:**
- `.kb/guides/code-extraction-patterns.md` (entire document)
- `.kb/guides/code-extraction-patterns.md:293-309` (Line Count Benchmarks table)

**Significance:** Bloat detection recommendations can reference the existing guide. No need to generate extraction instructions - just point to the guide.

---

## Synthesis

**Key Insights:**

1. **System boundary clarity** - `orch learn` tracks context gaps during spawns (missing knowledge). `orch hotspot` detects code health signals (fix churn, investigation clusters, now bloat). These are different concerns with different data sources and different remediation paths.

2. **Architectural fit** - Adding bloat-size as a third hotspot type requires ~100 lines of new code (one analyzer function + CLI flag). Adding it to learning.go would require a new data model, tracking mechanism, and suggestion generator - much more complexity for no additional value.

3. **Severity-based recommendations** - The 800-line threshold is just the "gate" where action is needed. Above 1500 lines likely requires architect involvement (structural redesign), while 800-1500 can often be handled by feature-impl with extraction patterns guide.

**Answer to Investigation Question:**

The 800-line bloat gate should NOT be implemented in `orch learn`. Instead, it should be added as a third hotspot type ("bloat-size") in `orch hotspot`. This:
- Follows established patterns (reuses Hotspot struct, analyzer pattern, exclusion logic)
- Keeps system boundaries clean (hotspot = code health, learn = context gaps)
- Enables severity-based recommendations (different thresholds → different skills)
- Integrates with existing dashboard via /api/hotspot endpoint

---

## Structured Uncertainty

**What's tested:**
- ✅ orch hotspot already supports multiple hotspot types (verified: read hotspot.go, saw fix-density and investigation-cluster)
- ✅ File exclusion logic exists and is appropriate (verified: read shouldCountFileWithExclusions)
- ✅ 28 files in codebase exceed 800 lines (verified: ran find/wc -l)
- ✅ Extraction patterns guide exists with proven benchmarks (verified: read code-extraction-patterns.md)

**What's untested:**
- ⚠️ Optimal severity thresholds (800/1500 are heuristics, may need tuning)
- ⚠️ Whether bloat detection should run at spawn time vs standalone command
- ⚠️ Dashboard integration needs (may want bloat indicator in stats bar)

**What would change this:**
- If spawn-time gating is needed (block spawn to bloated file) - would need learning.go integration
- If project-specific thresholds are common - would need per-project config
- If git history affects bloat severity (files getting bigger vs stable) - would need trend analysis

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Add bloat-size as third hotspot type in orch hotspot**

**Why this approach:**
- Follows established pattern (Finding 2: existing architecture supports it)
- Maintains clean system boundaries (Finding 1: bloat is coherence, not context gap)
- Reuses exclusion logic (Finding 3: same files to exclude)
- Links to existing remediation docs (Finding 4: extraction patterns guide exists)

**Trade-offs accepted:**
- Bloat isn't tracked over time like context gaps (no trend analysis)
- No spawn-time gating (bloat detection is after-the-fact)

**Implementation sequence:**
1. Add `--bloat-threshold` flag to hotspot command (default: 800)
2. Add `analyzeBloatFiles()` function returning `[]Hotspot` with `Type: "bloat-size"`
3. Add severity-based recommendations:
   - 800-1500 lines: "Recommend extraction - see .kb/guides/code-extraction-patterns.md"
   - >1500 lines: "CRITICAL: Recommend architect session for structural redesign"
4. Include in existing hotspot output (CLI and API)

### Alternative Approaches Considered

**Option A: Add to orch learn as BloatEvent**
- **Pros:** Would enable trend tracking, recurrence detection
- **Cons:** Bloat is static file property, not spawn-time observation; requires new data model
- **When to use instead:** If spawn-time blocking on bloat becomes a requirement

**Option B: Standalone orch bloat command**
- **Pros:** Complete separation of concerns
- **Cons:** Creates third tool for code health; users have to remember multiple commands
- **When to use instead:** If bloat detection has significantly different use cases than hotspot

**Rationale for recommendation:** Option C (extend hotspot) follows Compose Over Monolith principle - one tool for code health signals, with multiple detection types. This is exactly how Unix utilities work (grep searches, ls lists, find finds - one tool per concept, not one tool per variant).

---

### Implementation Details

**What to implement first:**
- `analyzeBloatFiles()` function (core detection)
- CLI flag `--bloat-threshold` with default 800

**Things to watch out for:**
- ⚠️ Test files can legitimately be long (table-driven tests) - already excluded by shouldCountFile
- ⚠️ Generated files should be excluded - already excluded by shouldCountFile
- ⚠️ Some languages have different norms (800 may be low for some, high for others)

**Areas needing further investigation:**
- Should bloat threshold be configurable per-project in .orch/config.yaml?
- Should dashboard show bloat hotspots differently (file icon vs wrench icon)?
- Should there be a "quick fix" recommendation for obvious extractions?

**Success criteria:**
- ✅ `orch hotspot` shows bloated files with type "bloat-size"
- ✅ Files >800 lines appear, with severity escalation at >1500
- ✅ API endpoint /api/hotspot includes bloat hotspots
- ✅ Recommendations reference code-extraction-patterns.md guide

---

## Feature Definition

**Feature ID:** feat-049

**Title:** Add bloat-size hotspot type to orch hotspot for 800-line gate

**Description:** Extend `orch hotspot` with bloat-size detection. Scan source files (using existing shouldCountFile exclusions), flag files exceeding --bloat-threshold (default: 800 lines), generate severity-based recommendations. 800-1500 lines: recommend extraction with guide reference. >1500 lines: recommend architect session for structural redesign. Include in CLI output and /api/hotspot endpoint.

**File targets:**
- `cmd/orch/hotspot.go` (add flag, analyzeBloatFiles function, integrate into runHotspot)
- `cmd/orch/hotspot_test.go` (add tests for bloat detection)

**Acceptance criteria:**
- `orch hotspot` shows bloat-size hotspots for files >800 lines
- `orch hotspot --bloat-threshold 1000` customizes threshold
- API /api/hotspot includes bloat-size entries
- Recommendations differ by severity (800-1500 vs >1500)

**Priority:** medium

**Skill:** feature-impl

**Estimated effort:** ~2 hours (100-150 lines new code)

---

## References

**Files Examined:**
- `.kb/models/extract-patterns.md` - 800-line gate definition and rationale
- `.kb/guides/code-extraction-patterns.md` - Extraction workflow and benchmarks
- `cmd/orch/hotspot.go` - Current hotspot detection implementation
- `cmd/orch/serve_hotspot.go` - API endpoint for hotspots
- `pkg/spawn/learning.go` - Gap tracking system (to understand boundary)
- `pkg/spawn/gap.go` - Gap analysis types
- `~/.kb/principles.md` - Coherence Over Patches, Compose Over Monolith

**Commands Run:**
```bash
# Count files exceeding 800 lines
find . -name "*.go" -o -name "*.svelte" -o -name "*.ts" | grep -v node_modules | grep -v vendor | xargs wc -l | sort -rn | head -30

# Search for existing bloat/hotspot references
grep -ri "800.?line\|bloat\|hotspot" --include="*.go" --include="*.md"
```

**Related Artifacts:**
- **Model:** `.kb/models/extract-patterns.md` - Defines 800-line gate concept
- **Guide:** `.kb/guides/code-extraction-patterns.md` - Remediation procedures
- **Decision:** (to be created) - Bloat detection belongs in hotspot, not learn

---

## Investigation History

**[2026-01-17 10:30]:** Investigation started
- Initial question: How should orch learn detect and respond to 800-line bloat?
- Context: Spawned from architect design task

**[2026-01-17 11:00]:** Phase 1 Problem Framing complete
- Identified 4 decision forks: detection point, file scope, integration strategy, action recommendation
- Gathered substrate from principles, models, and existing code

**[2026-01-17 11:30]:** Phases 2-3 Exploration and Synthesis complete
- Key insight: Bloat is coherence signal (like fix-density), not context gap (like learning.go tracks)
- Recommendation: Add bloat-size as third hotspot type, not new system

**[2026-01-17 12:00]:** Investigation completed
- Status: Complete
- Key outcome: Design for feat-049 (bloat-size hotspot type) with severity-based recommendations
