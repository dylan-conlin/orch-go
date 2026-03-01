# Probe: extraction.go Self-Hotspot — Extraction Plan

**Model:** Code Extraction Patterns (`.kb/models/extract-patterns/model.md`)
**Date:** 2026-03-01
**Status:** Complete
**Triggered By:** extraction.go at 1632 lines — CRITICAL hotspot. Prior probe (2026-02-19) confirmed 9 domains at 2011 lines; partial extraction (spawn_modes.go, spawn_helpers.go) reduced to 1632 but still 2x the 800-line gate.

---

## Question

After partial extraction, does extraction.go still contain identifiable Cohesive Extraction Units? Specifically:
1. How many extraction domains remain after spawn_modes.go and spawn_helpers.go were extracted?
2. What is the optimal extraction strategy to reach <500 lines?
3. Does the `pkg/orch` package constraint (exported types, no circular dependency risk) alter the model's Phase-based strategy?

---

## What I Tested

**Static analysis of `pkg/orch/extraction.go` (1632 lines):**
- Mapped all 39 function definitions with line ranges
- Identified responsibility domains by logical grouping and coupling
- Traced all external callers via grep for `orch.FunctionName` patterns
- Verified existing sibling files (spawn_modes.go:530, spawn_helpers.go:148, completion.go:265, flags.go:45)
- Checked test coverage: extraction_test.go (453 lines) covers DetermineSpawnBackend, isArchitectIssue, DetermineSpawnTier, extractSearchTerms

**Coupling analysis:**
- External callers: cmd/orch/spawn_cmd.go and cmd/orch/rework_cmd.go
- Both call exported functions individually (pipeline steps, not orchestrated chain)
- Internal coupling: types (SpawnInput, SpawnContext) used by most functions
- Discovered duplicate: `isInfrastructureWork` exists in BOTH pkg/orch/extraction.go AND cmd/orch/spawn_cmd.go (separate implementations)

---

## What I Observed

### 1. 800-Line Gate: Still 2x over after partial extraction

Prior probe (Feb 19): 2011 lines, 9 domains
Current (Mar 1): 1632 lines after extracting spawn modes (530 lines) and helpers (148 lines)
Remaining: 7 distinct extraction domains within the file

### 2. Cohesive Extraction Units: 7 domains identified

| # | Domain | Lines | Line Count | Target File | Key Functions |
|---|--------|-------|------------|-------------|---------------|
| 1 | Types | 28-107 | ~80 | `spawn_types.go` | SpawnInput, SpawnContext, ResolvedSpawnResult, GapCheckResult |
| 2 | Spawn Inference | 111-346 | ~240 | `spawn_inference.go` | InferSkillFromIssueType, DetermineSpawnTier, inferTierFromTask, containsAny, inferSkillFromBeadsIssue, inferMCPFromBeadsIssue, validateModeModelCombo |
| 3 | Pre-flight Gates | 348-554 | ~210 | `spawn_preflight.go` | RunPreFlightChecks, buildArchitectVerifier, buildArchitectFinder, FindPriorArchitectReview, extractSearchTerms, isArchitectIssue |
| 4 | KB Context & Gap Analysis | 794-873, 1268-1442 | ~320 | `spawn_kb_context.go` | GatherSpawnContext, runPreSpawnKBCheck, runPreSpawnKBCheckFull, checkGapGating, recordGapForLearning, extractPrimaryModelPath |
| 5 | Backend & Infrastructure | 908-1009, 1444-1517 | ~170 | `spawn_backend.go` | DetermineSpawnBackend, isInfrastructureWork, IsInfrastructureWork |
| 6 | Beads Lifecycle | 634-715, 1169-1224, 1618-1632 | ~180 | `spawn_beads.go` | SetupBeadsTracking, determineBeadsID, CreateBeadsIssue, resolveShortBeadsID |
| 7 | Design Artifacts | 1011-1027, 1519-1606 | ~100 | `spawn_design.go` | LoadDesignArtifacts, readDesignArtifacts, extractDesignNotes, extractSection |

### 3. What stays in extraction.go (~330 lines)

Core pipeline setup functions that compose the spawn pipeline:
- ResolveProjectDirectory, LoadSkillAndGenerateWorkspace (project setup)
- ResolveAndValidateModel, ResolveSpawnSettings (model resolution)
- CheckAndAutoSwitchAccount (account management)
- ExtractBugReproInfo, BuildUsageInfo (small helpers)
- BuildSpawnConfig, ValidateAndWriteContext (config building + atomic writes)
- checkWorkspaceExists, dirExists, truncate (utilities)

### 4. Test distribution

extraction_test.go (453 lines) maps cleanly to extracted domains:
- TestDetermineSpawnBackend_* (7 tests, ~200 lines) → spawn_backend_test.go
- TestIsArchitectIssue (1 test, ~60 lines) → spawn_preflight_test.go
- TestDetermineSpawnTier_* (1 test, ~40 lines) → spawn_inference_test.go
- TestExtractSearchTerms (1 test, ~55 lines) → spawn_preflight_test.go

### 5. Duplicate function discovery

`isInfrastructureWork` exists in TWO places:
- `pkg/orch/extraction.go:1454` (unexported, with exported wrapper at 1515)
- `cmd/orch/spawn_cmd.go:917` (separate package-local copy)

spawn_cmd.go line 664 calls its own local copy, not orch.IsInfrastructureWork.
The rework_cmd.go (line 207) calls orch.IsInfrastructureWork.
This is tech debt — the cmd/orch copy should be removed and callers unified.

---

## Model Impact

**CONFIRMS:**
- The 800-Line Gate — extraction.go at 2x threshold still contains 7 extractable domains
- Cohesive Extraction Units — each domain has clear boundaries (shared types, function groups, coupling patterns)
- Phase-based strategy applies — types first, then domains in parallel

**EXTENDS:**
- **Partial extraction doesn't fix the problem**: spawn_modes.go (530 lines) and spawn_helpers.go (148 lines) were extracted but the source file remained CRITICAL. The model should note that extraction of only 1-2 domains from a 9-domain monolith is insufficient — you need to extract most domains to get below threshold
- **Test co-location works within pkg packages**: extraction_test.go maps cleanly to domain files, confirming the model's "tests follow handlers" pattern works for pkg/ files too
- **Duplicate function anti-pattern**: When code is extracted from cmd/orch to pkg/orch, leftover copies in cmd/orch create divergent duplicates. The model should note: after extraction, grep for remaining copies in the source package

**CONTRADICTS:** Nothing — findings align with model predictions.

---

## Extraction Plan

### Phase 1: Types (must be first)
Extract `spawn_types.go` (80 lines) — SpawnInput, SpawnContext, ResolvedSpawnResult, GapCheckResult

### Phase 2: Domain files (parallel, after types)
1. `spawn_inference.go` (~240 lines) — skill/tier inference, model validation
2. `spawn_preflight.go` (~210 lines) — pre-flight gates, architect verification
3. `spawn_kb_context.go` (~320 lines) — KB context gathering, gap analysis
4. `spawn_backend.go` (~170 lines) — backend routing, infrastructure detection
5. `spawn_beads.go` (~180 lines) — beads issue lifecycle
6. `spawn_design.go` (~100 lines) — design artifact reading

### Phase 3: Test redistribution
Split extraction_test.go → spawn_backend_test.go, spawn_preflight_test.go, spawn_inference_test.go

### Phase 4: Cleanup
- Remove `isInfrastructureWork` duplicate from cmd/orch/spawn_cmd.go
- Rename extraction.go → spawn_pipeline.go (better describes remaining content)

### Expected outcome
- extraction.go/spawn_pipeline.go: ~330 lines (core pipeline)
- 7 new files: ~1300 lines distributed across cohesive domains
- All files under 400 lines
