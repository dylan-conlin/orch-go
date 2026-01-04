<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** pkg/verify has grown to 14 files (~3,200 LOC) with 8 distinct verification domains mixed into check.go (980 lines), creating architectural debt.

**Evidence:** Analysis of check.go shows it contains: beads API wrapper (500+ LOC), synthesis parsing (100+ LOC), completion verification, tier handling, and comment batch operations - distinct domains that should be separate packages.

**Knowledge:** The package evolved organically around `orch complete` needs; extraction should group by domain (beads, synthesis, completion), preserve existing test coverage, and maintain backward compatibility via type aliases.

**Next:** Implement phased extraction: Phase 1 (beads wrapper), Phase 2 (synthesis parsing), Phase 3 (completion orchestration), Phase 4 (domain-specific verifiers).

---

# Investigation: pkg/verify/check.go Structure Analysis and Extraction Plan

**Question:** How should the 980-line check.go be refactored to separate verification domains and improve maintainability?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Package Contains 8 Distinct Verification Domains

**Evidence:** Analysis of pkg/verify/ reveals these distinct domains:

| Domain | Files | LOC | Purpose |
|--------|-------|-----|---------|
| **Beads API** | check.go (partial) | ~500 | RPC/CLI wrapper for beads operations |
| **Phase Tracking** | phase_gates.go | 186 | Phase extraction and gate verification |
| **Synthesis** | check.go (partial) | ~100 | SYNTHESIS.md parsing |
| **Visual** | visual.go | 415 | Web change detection, visual evidence |
| **Test Evidence** | test_evidence.go | 349 | Code change detection, test patterns |
| **Build** | build_verification.go | 236 | Go build verification |
| **Git Diff** | git_diff.go | 269 | SYNTHESIS vs git diff verification |
| **Constraints** | constraint.go | 252 | SPAWN_CONTEXT constraint verification |
| **Skill Outputs** | skill_outputs.go | 234 | Skill manifest verification |
| **Context Risk** | context_risk.go | 183 | Token usage monitoring |
| **Escalation** | escalation.go | 325 | Completion escalation logic |
| **Review** | review.go, review_state.go | 540 | Agent review formatting |
| **Repro** | repro.go | 114 | Bug reproduction extraction |
| **Attempts** | attempts.go | 347 | Fix attempt tracking |

**Source:** `pkg/verify/*.go` (14 files, ~3,200 total LOC)

**Significance:** The package has grown significantly but retains reasonable modularity in files - the main issue is check.go which mixes beads wrapper, synthesis parsing, and completion verification.

---

### Finding 2: check.go Contains Three Separate Concerns

**Evidence:** Analysis of check.go (980 lines) reveals three distinct concerns:

1. **Beads API Wrapper (lines 45-979, ~600 LOC)**
   - `GetComments()`, `GetCommentsWithDir()`, `FallbackCommentsWithDir()`
   - `CloseIssue()`, `UpdateIssueStatus()`, `RemoveTriageReadyLabel()`
   - `GetIssue()`, `GetIssuesBatch()`, `ListOpenIssues()`
   - `GetCommentsBatch()`, `GetCommentsBatchWithProjectDirs()`
   - All use RPC-first with CLI fallback pattern

2. **Synthesis Parsing (lines 163-340, ~180 LOC)**
   - `Synthesis` struct with D.E.K.N. fields
   - `ParseSynthesis()` with section extraction
   - Helper functions: `extractHeaderField()`, `extractSection()`, `extractSectionWithVariant()`
   - `extractRecommendation()`, `extractNextActions()`, `parseActionItems()`, `extractBoldSubsection()`

3. **Completion Verification (lines 386-575, ~190 LOC)**
   - `VerificationResult` struct
   - `VerifySynthesis()`, `VerifyCompletion()`, `VerifyCompletionWithTier()`
   - `VerifyCompletionFull()` - orchestrates all domain verifiers
   - `ReadTierFromWorkspace()`

**Source:** `pkg/verify/check.go:45-979`

**Significance:** These three concerns have different change frequencies and consumers. Beads wrapper is stable infrastructure; synthesis parsing changes with protocol; completion verification integrates domain verifiers.

---

### Finding 3: Existing Domain Files Have Clear Boundaries

**Evidence:** The already-extracted domain files follow a consistent pattern:

```
visual.go:
  - IsSkillRequiringVisualVerification()
  - VisualVerificationResult struct
  - VerifyVisualVerification()
  - VerifyVisualVerificationForCompletion() -> *VisualVerificationResult

test_evidence.go:
  - IsSkillRequiringTestEvidence()
  - TestEvidenceResult struct
  - VerifyTestEvidence()
  - VerifyTestEvidenceForCompletion() -> *TestEvidenceResult

build_verification.go:
  - IsSkillRequiringBuildVerification()
  - BuildVerificationResult struct
  - VerifyBuild()
  - VerifyBuildForCompletion() -> *BuildVerificationResult
```

Each domain verifier follows the pattern:
1. Skill whitelist/blacklist maps
2. `IsSkillRequiring<Domain>()` function
3. Domain-specific result struct
4. Main verification function
5. `Verify<Domain>ForCompletion()` wrapper for use in `VerifyCompletionFull()`

**Source:** `pkg/verify/visual.go`, `pkg/verify/test_evidence.go`, `pkg/verify/build_verification.go`

**Significance:** The extraction pattern is already established. New extractions should follow this pattern for consistency.

---

### Finding 4: check.go's VerifyCompletionFull is the Integration Point

**Evidence:** `VerifyCompletionFull()` (lines 400-505) orchestrates all domain verifiers:

```go
func VerifyCompletionFull(beadsID, workspacePath, projectDir, tier string) (VerificationResult, error) {
    // First run standard verification
    result, err := VerifyCompletionWithTier(beadsID, workspacePath, tier)
    
    // Then call each domain verifier:
    constraintResult, _ := VerifyConstraintsForCompletion(...)
    phaseGateResult, _ := VerifyPhaseGatesForCompletion(...)
    skillOutputResult, _ := VerifySkillOutputsForCompletion(...)
    visualResult := VerifyVisualVerificationForCompletion(...)
    testEvidenceResult := VerifyTestEvidenceForCompletion(...)
    gitDiffResult := VerifyGitDiffForCompletion(...)
    buildResult := VerifyBuildForCompletion(...)
    
    // Merge results
    return result, nil
}
```

**Source:** `pkg/verify/check.go:400-505`

**Significance:** This function is the natural home for the completion package. It should own the integration logic while delegating to domain packages.

---

### Finding 5: Beads Types and pkg/beads Have Overlap

**Evidence:** check.go defines:
```go
type Comment = beads.Comment  // Alias
type Issue struct { ... }     // Local struct with subset of fields
```

pkg/beads already provides `beads.Issue` and `beads.Comment`. The verify package adds wrapper functions that handle RPC-first with CLI fallback and cross-project directory support.

**Source:** `pkg/verify/check.go:24-36`, `pkg/beads/client.go`

**Significance:** The beads wrapper in verify should either:
1. Stay in verify as internal implementation detail, OR
2. Move to pkg/beads as higher-level convenience functions

Option 1 is simpler; Option 2 creates tighter coupling but better cohesion.

---

### Finding 6: Comment-Related Functions Have Caching/Batching Logic

**Evidence:** check.go contains sophisticated comment handling:

```go
GetCommentsBatch(beadsIDs []string) map[string][]Comment
GetCommentsBatchWithProjectDirs(beadsIDs []string, projectDirs map[string]string) map[string][]Comment
```

These use:
- Semaphore for concurrent RPC calls (maxConcurrent = 20)
- Mutex-protected map for thread-safe writes
- Project directory grouping for efficient RPC client reuse

**Source:** `pkg/verify/check.go:856-979`

**Significance:** This is non-trivial infrastructure that deserves its own package or clear domain within the beads wrapper.

---

## Synthesis

**Key Insights:**

1. **check.go is a god file** - It has grown to contain three distinct domains (beads wrapper, synthesis parsing, completion verification) that have different change frequencies and consumers. The 980 lines exceed typical file size guidelines.

2. **The extraction pattern is proven** - Domain-specific verifiers (visual, test_evidence, build_verification, constraint, phase_gates, skill_outputs, git_diff) already follow a consistent pattern. Extracting from check.go should follow this pattern.

3. **VerifyCompletionFull is the orchestrator** - This function naturally belongs in a "completion" domain that coordinates all verifiers. It should not contain the implementation details of individual verifications.

4. **Beads wrapper is infrastructure** - The RPC-first/CLI-fallback pattern with cross-project support is stable infrastructure that multiple packages depend on.

**Answer to Investigation Question:**

check.go should be extracted into three packages:
1. **pkg/verify/beads** (or keep in check.go as internal) - RPC/CLI wrapper functions
2. **pkg/verify/synthesis** (or synthesis.go) - Synthesis parsing
3. **pkg/verify** (check.go reduced) - Completion verification orchestration

The existing domain verifiers should remain as separate files. The key refactoring is making check.go focused on the completion orchestration role, not holding all the pieces.

---

## Structured Uncertainty

**What's tested:**

- ✅ Line counts and domain boundaries (verified: read all 14 source files)
- ✅ VerifyCompletionFull integration pattern (verified: read source code)
- ✅ Existing domain verifier patterns (verified: compared visual.go, test_evidence.go, build_verification.go)

**What's untested:**

- ⚠️ Impact of extraction on import cycles (not validated with Go compiler)
- ⚠️ Performance implications of package restructuring (not benchmarked)
- ⚠️ Test coverage after extraction (tests need to be analyzed)

**What would change this:**

- If beads RPC/CLI wrapper is used outside pkg/verify, it should move to pkg/beads
- If synthesis parsing is used outside pkg/verify, it should be its own package
- If import cycles are discovered, package structure may need adjustment

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Phased In-Place Extraction

**Why this approach:**
- Preserves backward compatibility via type aliases
- Enables incremental validation of each phase
- Minimizes risk of breaking existing consumers

**Trade-offs accepted:**
- More phases = more coordination overhead
- Type aliases add temporary complexity

### Phase 1: Extract Synthesis Parsing (synthesis.go)

**What to extract:**
- `Synthesis` struct and related types
- `ParseSynthesis()` function
- All helper functions: `extractHeaderField()`, `extractSection()`, `extractSectionWithVariant()`, `extractRecommendation()`, `extractNextActions()`, `parseActionItems()`, `extractBoldSubsection()`

**Where:** `pkg/verify/synthesis.go` (new file)

**Why first:**
- Cleanest extraction (no external dependencies beyond stdlib)
- Most self-contained (only needs filepath and os)
- Test file: synthesis_test.go

**Estimated LOC:** ~180 lines extracted

### Phase 2: Consolidate Beads Wrapper (beads.go or keep in check.go)

**What to consolidate:**
- `Comment` type alias
- `Issue` struct (local version)
- `GetComments()`, `GetCommentsWithDir()`, `FallbackCommentsWithDir()`
- `GetPhaseStatus()`, `ParsePhaseFromComments()`, `IsPhaseComplete()`, `HasBeadsComment()`
- `CloseIssue()`, `UpdateIssueStatus()`, `RemoveTriageReadyLabel()`
- `GetIssue()`, `GetIssuesBatch()`, `ListOpenIssues()`
- `GetCommentsBatch()`, `GetCommentsBatchWithProjectDirs()`

**Where:** `pkg/verify/beads.go` (new file) or keep in check.go as "beads section"

**Why second:**
- Dependencies on pkg/beads are already handled
- Natural grouping of RPC/CLI wrapper logic

**Estimated LOC:** ~550 lines extracted (or reorganized)

### Phase 3: Reduce check.go to Completion Orchestration

**What remains:**
- `VerificationResult` struct
- `VerifySynthesis()` (calls synthesis.go functions)
- `VerifyCompletion()`, `VerifyCompletionWithTier()`, `VerifyCompletionFull()`
- `ReadTierFromWorkspace()`
- Pre-compiled regex patterns used only in check.go

**Where:** `pkg/verify/check.go` (reduced)

**Why third:**
- After Phase 1 and 2, check.go becomes focused orchestration
- The remaining ~200 lines is appropriate file size

**Estimated LOC:** ~200 lines remaining

### Phase 4: Optional - Consider Package Restructuring

**If needed after Phase 1-3:**
- Evaluate if pkg/verify is doing too much
- Consider subpackages: verify/beads, verify/synthesis, verify/completion
- Or keep flat structure if < 15 files

**Decision criteria:**
- More than 20 files in pkg/verify → consider subpackages
- Import cycles discovered → require restructuring
- External consumers want specific domains → extract to subpackages

---

### Implementation Details

**What to implement first:**
- Phase 1 (synthesis.go) - cleanest extraction, lowest risk

**Things to watch out for:**
- ⚠️ Pre-compiled regex patterns at package level - ensure they move with their functions
- ⚠️ Type aliases may confuse IDE tools - keep them temporary
- ⚠️ Test file organization - may need parallel restructuring

**Areas needing further investigation:**
- Which functions are called from outside pkg/verify (public API surface)
- Test coverage of each domain
- Import cycle analysis after extraction

**Success criteria:**
- ✅ check.go reduced to < 300 lines
- ✅ Each extracted file is cohesive (single domain)
- ✅ All existing tests pass after extraction
- ✅ No new import cycles introduced

---

## File Structure After Extraction

```
pkg/verify/
├── check.go           (~200 LOC) - Completion verification orchestration
├── synthesis.go       (~180 LOC) - SYNTHESIS.md parsing [NEW]
├── beads.go           (~550 LOC) - Beads RPC/CLI wrapper [NEW]
├── phase_gates.go     (186 LOC)  - Phase gate verification
├── constraint.go      (252 LOC)  - Constraint verification
├── skill_outputs.go   (234 LOC)  - Skill output verification
├── visual.go          (415 LOC)  - Visual verification
├── test_evidence.go   (349 LOC)  - Test evidence verification
├── build_verification.go (236 LOC) - Build verification
├── git_diff.go        (269 LOC)  - Git diff verification
├── context_risk.go    (183 LOC)  - Context exhaustion risk
├── escalation.go      (325 LOC)  - Escalation level logic
├── review.go          (415 LOC)  - Agent review formatting
├── review_state.go    (125 LOC)  - Review state persistence
├── repro.go           (114 LOC)  - Bug reproduction extraction
├── attempts.go        (347 LOC)  - Fix attempt tracking
└── *_test.go files    (various)  - Tests for each domain
```

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Primary analysis target (980 lines)
- `pkg/verify/visual.go` - Pattern reference (415 lines)
- `pkg/verify/test_evidence.go` - Pattern reference (349 lines)
- `pkg/verify/build_verification.go` - Pattern reference (236 lines)
- `pkg/verify/constraint.go` - Domain verifier example (252 lines)
- `pkg/verify/phase_gates.go` - Domain verifier example (186 lines)
- `pkg/verify/skill_outputs.go` - Domain verifier example (234 lines)
- `pkg/verify/git_diff.go` - Domain verifier example (269 lines)
- `pkg/verify/context_risk.go` - Risk assessment domain (183 lines)
- `pkg/verify/escalation.go` - Escalation logic domain (325 lines)
- `pkg/verify/review.go` - Review formatting domain (415 lines)
- `pkg/verify/review_state.go` - State persistence (125 lines)
- `pkg/verify/repro.go` - Reproduction extraction (114 lines)
- `pkg/verify/attempts.go` - Attempt tracking (347 lines)

**Commands Run:**
```bash
# File listing
glob pkg/verify/*.go

# Read all source files
# (Multiple read commands executed in parallel)
```

**Related Artifacts:**
- **Prior constraint:** "High patch density in a single area signals missing coherent model" - This investigation responds to that pattern in check.go

---

## Investigation History

**2026-01-04 08:XX:** Investigation started
- Initial question: How to extract/refactor check.go (979 lines)
- Context: Spawned by orchestrator to analyze pkg/verify structure

**2026-01-04 08:XX:** Analysis complete
- Read all 14 source files in pkg/verify
- Identified 8 distinct verification domains
- Mapped check.go structure (3 concerns, 980 lines)

**2026-01-04 08:XX:** Investigation completed
- Status: Complete
- Key outcome: Phased extraction plan with 4 phases, starting with synthesis.go
