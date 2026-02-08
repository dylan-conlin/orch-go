<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented a Gap Detection Layer for the spawn system that analyzes KB context results and warns about missing or sparse context before spawning agents.

**Evidence:** Created pkg/spawn/gap.go with GapAnalysis struct, AnalyzeGaps function, and context quality scoring (0-100). Integrated into both runPreSpawnKBCheck and gatherKBContext. All 25 tests pass.

**Knowledge:** Gap detection uses thresholds: <2 matches = sparse context warning, quality score weights constraints highest (25 pts), then decisions (15 pts), then investigations (10 pts). Critical gaps (no context) always warn; warning-level gaps only warn when quality <30%.

**Next:** The Gap Detection Layer is complete. The remaining layers (Failure Surfacing, System Learning Loop) can be implemented as follow-up work.

**Confidence:** High (85%) - Implementation tested thoroughly, but real-world effectiveness will need observation.

---

# Investigation: Gap Detection Layer

**Question:** How to implement the first layer of the Pressure Visibility System - detecting when orchestrator/agent lacks context it should have at spawn time?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-feat-gap-detection-layer-25dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: KB Context Already Has Infrastructure for Gap Detection

**Evidence:** The existing `KBContextResult` struct captures all matches with their types (constraint, decision, investigation, guide). The `KBContextFormatResult` already has truncation tracking. The key gap was *analyzing* these results for coverage quality, not just formatting them.

**Source:** 
- `pkg/spawn/kbcontext.go:42-68` - KBContextMatch and KBContextResult structs
- `pkg/spawn/kbcontext.go:60-68` - KBContextFormatResult has truncation info

**Significance:** Building on existing infrastructure minimizes risk. Gap detection becomes an analysis layer on top of existing data structures.

---

### Finding 2: Sparse Results Need Thresholds and Quality Scoring

**Evidence:** Defined two thresholds:
- `MinMatchesForGapDetection = 2` - Below this is sparse
- `HighConfidenceMatchThreshold = 5` - Above this is good coverage

Created quality scoring (0-100):
- Base: 10 pts per match (max 50)
- Constraints: 15 pts + 5 per extra (max 25) - highest priority
- Decisions: 10 pts + 3 per extra (max 15)
- Investigations: 5 pts + 2 per extra (max 10)

**Source:** 
- `pkg/spawn/gap.go:12-19` - Threshold constants
- `pkg/spawn/gap.go:152-192` - calculateContextQuality function

**Significance:** Quality scoring enables nuanced gap detection - not just "has context / no context" but "how good is the context coverage?" This enables tiered warnings.

---

### Finding 3: Gap Types Enable Specific Guidance

**Evidence:** Defined four gap types:
- `GapTypeNoContext` - No KB context found (critical)
- `GapTypeSparseContext` - Few matches found (warning)
- `GapTypeNoConstraints` - Context exists but no constraints (info)
- `GapTypeNoDecisions` - Context exists but no decisions (info)

Each gap type has specific suggestions:
- No context: "Consider running 'kb context' manually to verify, or add relevant kn entries"
- Sparse: "Agent may need to discover context during work"
- No constraints: "If there are constraints for this area, add them via 'kn constrain'"

**Source:** 
- `pkg/spawn/gap.go:21-39` - GapType constants
- `pkg/spawn/gap.go:75-130` - AnalyzeGaps function

**Significance:** Specific gap types enable targeted suggestions, helping users understand *what* is missing and *how* to fix it, not just that something is missing.

---

## Synthesis

**Key Insights:**

1. **Quality scoring enables graduated responses** - Instead of binary "has context / no context", the 0-100 quality score allows for nuanced messaging. Critical gaps (no context) always warn, but info-level gaps only surface when quality is low.

2. **Gap visibility respects user workflow** - Warnings go to stderr to be visible but not interfere with spawn output. Gap summaries in spawn context ensure agents know about limitations. The `ShouldWarnAboutGaps()` method prevents warning fatigue by only warning on significant gaps.

3. **Integration points are minimal and non-breaking** - Modified two functions (`runPreSpawnKBCheck` in main.go, `gatherKBContext` in skill_requires.go) to call gap analysis after KB context lookup. No structural changes to existing data flow.

**Answer to Investigation Question:**

Gap detection at spawn time is implemented through the `AnalyzeGaps` function which:
1. Examines KB context results for total matches and type distribution
2. Calculates a quality score (0-100) based on weighted contributions
3. Identifies specific gap types (no context, sparse, no constraints, no decisions)
4. Provides formatted warnings and summaries for visibility

The implementation is integrated into both the legacy `runPreSpawnKBCheck` path and the skill-driven `gatherKBContext` path, ensuring all spawns benefit from gap detection.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The implementation is thoroughly tested (25 tests covering all gap types, quality scoring, formatting, and threshold behavior). The code compiles and builds successfully. However, real-world effectiveness hasn't been observed yet.

**What's certain:**

- ✅ Gap detection correctly identifies no-context and sparse-context situations
- ✅ Quality scoring algorithm produces sensible scores (verified via tests)
- ✅ Integration into spawn flow works (tested build, all tests pass)
- ✅ Warning formatting produces clear, actionable messages

**What's uncertain:**

- ⚠️ Whether the thresholds (2 for sparse, 30 for low quality) are optimal
- ⚠️ Whether gap warnings will be noticed in practice or cause warning fatigue
- ⚠️ How agents will behave when spawned with gap warnings in context

**What would increase confidence to Very High (95%+):**

- Observe gap detection in real spawn workflows
- Tune thresholds based on actual usage patterns
- Add metrics to track gap frequency and types

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Gap Detection Layer is complete** - The implementation provides:
- `AnalyzeGaps()` function for analyzing KB context results
- Quality scoring (0-100) with type-weighted contributions
- Formatted warnings for stderr display
- Gap summaries for spawn context inclusion

**Next steps for the full Pressure Visibility System:**
1. **Failure Surfacing** - Make gaps visible and harder to ignore (beyond stderr warnings)
2. **System Learning Loop** - Convert observed gaps into mechanism improvements

### Implementation Details

**What was implemented:**

1. New file: `pkg/spawn/gap.go`
   - GapType, GapSeverity enums
   - Gap, GapAnalysis, MatchStatistics structs
   - AnalyzeGaps, calculateContextQuality functions
   - FormatGapWarning, FormatGapSummary methods

2. New file: `pkg/spawn/gap_test.go`
   - 25 test cases covering all functionality

3. Modified: `cmd/orch/main.go`
   - `runPreSpawnKBCheck` calls AnalyzeGaps and displays warnings

4. Modified: `pkg/spawn/skill_requires.go`
   - `gatherKBContext` calls AnalyzeGaps and displays warnings

**Success criteria met:**

- ✅ Gap detection at spawn time implemented
- ✅ Quality scoring provides nuanced assessment
- ✅ Warnings displayed on stderr
- ✅ Gap summaries included in spawn context for low-quality situations
- ✅ All tests pass

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` - KB context structures and formatting
- `pkg/spawn/skill_requires.go` - Skill-driven context gathering
- `cmd/orch/main.go` - Spawn command and runPreSpawnKBCheck
- `.kb/investigations/2025-12-25-inv-pressure-over-compensation-surfacing-mechanisms.md` - Prior investigation defining the 3-layer system

**Commands Run:**
```bash
# Build verification
go build ./...

# Test suite
go test ./pkg/spawn/... -v
go test ./... # Full suite
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Artifact structure
- **Investigation:** `.kb/investigations/2025-12-25-inv-pressure-over-compensation-surfacing-mechanisms.md` - Defined the 3-layer Pressure Visibility System

---

## Investigation History

**2025-12-25 ~17:00:** Investigation started
- Initial question: How to implement gap detection at spawn time?
- Context: Spawned as first layer of Pressure Visibility System

**2025-12-25 ~17:15:** Found existing infrastructure
- KB context already has all needed data in KBContextResult
- Decision: Build analysis layer on top of existing structures

**2025-12-25 ~17:30:** Implemented gap.go
- GapAnalysis with quality scoring
- Four gap types with specific guidance

**2025-12-25 ~17:45:** Integrated into spawn flow
- Modified runPreSpawnKBCheck and gatherKBContext
- All tests passing

**2025-12-25 ~18:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Gap Detection Layer implemented and tested
