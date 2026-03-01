# Design: extraction.go Self-Hotspot Extraction Plan

**Phase:** Complete
**Date:** 2026-03-01
**Skill:** architect
**Beads:** orch-go-6v3t

---

## Design Question

`pkg/orch/extraction.go` is 1632 lines — a CRITICAL hotspot (>1500 line threshold). Ironically, this file contains the spawn pipeline logic and was itself the product of an earlier extraction from `cmd/orch/spawn_cmd.go`. Prior probe (Feb 19) identified 9 domains at 2011 lines; partial extraction created spawn_modes.go (530 lines) and spawn_helpers.go (148 lines) but left the source file CRITICAL.

**Question:** How should extraction.go be decomposed to reach <500 lines while maintaining clean domain boundaries and avoiding import cycles?

---

## Problem Framing

**Success criteria:**
- extraction.go reduced to <500 lines
- Each extracted file is a cohesive domain (<400 lines)
- No circular dependencies introduced
- All tests pass after extraction
- Test files co-locate with extracted source

**Constraints:**
- `pkg/orch` is a flat Go package — no import cycles possible within it
- Exported functions must remain accessible as `orch.FunctionName`
- External callers: cmd/orch/spawn_cmd.go and cmd/orch/rework_cmd.go
- Tests in extraction_test.go (453 lines) must move with their functions

**Scope:**
- IN: Identify extraction boundaries, target files, function mappings, test redistribution
- OUT: Actual code moves (follow-up feature-impl work)

---

## Exploration

### Current File Structure (pkg/orch/)

| File | Lines | Purpose |
|------|-------|---------|
| extraction.go | 1632 | Spawn pipeline steps (CRITICAL hotspot) |
| spawn_modes.go | 530 | Spawn dispatch + mode implementations |
| completion.go | 265 | Completion backlog detection |
| spawn_helpers.go | 148 | Shared helpers (shortID, stripANSI, etc.) |
| flags.go | 45 | Flag constants |

### Function Map (39 functions in extraction.go)

Grouped by logical domain:

**Domain A: Types (~80 lines, 28-107)**
- `SpawnInput` struct
- `SpawnContext` struct
- `ResolvedSpawnResult` struct
- `GapCheckResult` struct

**Domain B: Spawn Inference (~240 lines, 111-346)**
- `InferSkillFromIssueType` — maps issue types to skills
- `DetermineSpawnTier` — resolves tier from flags/config/task
- `inferTierFromTask` — heuristic tier detection from task text
- `containsAny` — string matching utility
- `validateModeModelCombo` — validates backend+model compatibility
- `inferSkillFromBeadsIssue` — infers skill from issue labels/title
- `inferMCPFromBeadsIssue` — extracts MCP server from issue labels

**Domain C: Pre-flight Gates (~210 lines, 348-554)**
- `RunPreFlightChecks` — orchestrates all pre-spawn validation
- `buildArchitectVerifier` — validates architect issue references
- `buildArchitectFinder` — searches for prior architect reviews
- `FindPriorArchitectReview` — queries beads for closed architect issues
- `extractSearchTerms` — builds search terms from file paths
- `isArchitectIssue` — checks if issue has architect skill

**Domain D: Project/Skill Setup (~75 lines, 556-632)**
- `ResolveProjectDirectory` — resolves workdir to absolute path
- `LoadSkillAndGenerateWorkspace` — loads skill content + generates workspace name

**Domain E: Beads Lifecycle (~180 lines, 634-715 + 1169-1224 + 1618-1632)**
- `SetupBeadsTracking` — determines beads ID, manages issue lifecycle, detects duplicates
- `determineBeadsID` — resolves beads ID from flag/auto-create/no-track
- `CreateBeadsIssue` — creates new beads issue via RPC or CLI
- `resolveShortBeadsID` — resolves short IDs to full format

**Domain F: Model/Settings Resolution (~75 lines, 717-792)**
- `ResolveAndValidateModel` — resolves aliases, validates flash restriction
- `ResolveSpawnSettings` — centralized spawn settings resolver

**Domain G: KB Context & Gap Analysis (~320 lines, 794-873 + 1268-1442)**
- `GatherSpawnContext` — gathers KB context with gap analysis
- `runPreSpawnKBCheck` — thin wrapper
- `runPreSpawnKBCheckFull` — full KB check with keyword extraction, filtering, formatting
- `checkGapGating` — blocks spawn if context quality too low
- `recordGapForLearning` — records gaps for pattern detection
- `extractPrimaryModelPath` — extracts model path from format result

**Domain H: Backend & Infrastructure (~170 lines, 908-1009 + 1444-1517)**
- `DetermineSpawnBackend` — complex routing: flags > model > project config > user config > infra detection
- `isInfrastructureWork` — detects infrastructure work via keywords
- `IsInfrastructureWork` — exported wrapper

**Domain I: Config Building & Validation (~170 lines, 1030-1167 + 1229-1266)**
- `BuildSpawnConfig` — constructs spawn.Config from SpawnContext
- `ValidateAndWriteContext` — validates context size, atomic spawn Phase 1
- `checkWorkspaceExists` — prevents overwriting existing sessions
- `dirExists` — filesystem check utility

**Domain J: Design Artifacts (~100 lines, 1011-1027 + 1519-1606)**
- `LoadDesignArtifacts` — reads design artifacts from workspace
- `readDesignArtifacts` — reads mockup, prompt, synthesis from workspace
- `extractDesignNotes` — extracts TLDR/Knowledge from SYNTHESIS.md
- `extractSection` — generic section extraction from markdown

**Domain K: Small Helpers (~50 lines, scattered)**
- `CheckAndAutoSwitchAccount` (68 lines) — account management
- `ExtractBugReproInfo` (14 lines) — bug reproduction info
- `BuildUsageInfo` (13 lines) — usage info builder
- `truncate` (6 lines) — string truncation

### Coupling Analysis

**Types (Domain A)** are the shared foundation:
- `SpawnInput` → used by RunPreFlightChecks, DispatchSpawn (spawn_modes.go)
- `SpawnContext` → used by BuildSpawnConfig, external callers
- `GapCheckResult` → used by runPreSpawnKBCheckFull, GatherSpawnContext

**Cross-domain calls within extraction.go:**
- `DetermineSpawnBackend` calls `isInfrastructureWork` and `validateModeModelCombo` (all in Domain H/B)
- `RunPreFlightChecks` calls `buildArchitectVerifier`, `buildArchitectFinder` (all in Domain C)
- `GatherSpawnContext` calls `runPreSpawnKBCheckFull`, `checkGapGating`, `recordGapForLearning` (all in Domain G)
- `SetupBeadsTracking` calls `determineBeadsID` (both in Domain E)
- `ValidateAndWriteContext` calls `checkWorkspaceExists` (both in Domain I)
- `LoadDesignArtifacts` calls `readDesignArtifacts` (both in Domain J)

**Key observation:** Cross-domain coupling is minimal. Each domain's internal functions call each other, but domains rarely call across boundaries. This confirms clean extraction boundaries.

### Duplicate Function Discovery

`isInfrastructureWork` exists in TWO locations:
1. `pkg/orch/extraction.go:1454` (with exported wrapper at line 1515)
2. `cmd/orch/spawn_cmd.go:917` (separate copy in package main)

The cmd/orch copy is called at spawn_cmd.go:664. The pkg/orch copy is called by rework_cmd.go:207 via `orch.IsInfrastructureWork`. This duplication should be eliminated.

---

## Synthesis

### Fork 1: Extraction Granularity

**Options:**
- A: Extract all 7 domains (types + 6 domain files) → ~330 lines remaining
- B: Extract only largest 4 domains → ~700 lines remaining
- C: Merge related small domains → fewer files but larger

**SUBSTRATE:**
- Model: 800-Line Gate says extraction triggers at 800 lines
- Guide: "Target ~300-800 lines per file"
- Principle: Session Amnesia — smaller files are more comprehensible per-session

**RECOMMENDATION:** Option A — extract all 7 domains. 330 lines remaining is well under the 800-line gate, and each extracted file stays under 400 lines. The domains are cleanly separable with minimal cross-coupling.

**Trade-off accepted:** 7 new files adds to the pkg/orch directory, but each is a self-contained domain that any agent can comprehend independently.

### Fork 2: What stays in extraction.go

**Options:**
- A: Keep as "extraction.go" with pipeline setup functions
- B: Rename to "spawn_pipeline.go" to better describe content

**SUBSTRATE:**
- Decision: File naming should describe content (from code-extraction guide)
- The name "extraction.go" suggests hotspot extraction logic, but the file contains spawn pipeline steps

**RECOMMENDATION:** Option B — rename to `spawn_pipeline.go`. The current name is misleading; it was created when spawn logic was "extracted" from spawn_cmd.go, but the name implies this file contains extraction/hotspot analysis logic.

**Trade-off accepted:** Rename means updating imports — but since this is a package-internal file (not imported by path), only test files reference it directly.

### Fork 3: Handling the isInfrastructureWork duplicate

**Options:**
- A: Keep both copies (avoid touching spawn_cmd.go)
- B: Remove cmd/orch copy, have it call orch.IsInfrastructureWork
- C: Move canonical copy to spawn_backend.go during extraction

**SUBSTRATE:**
- Principle: Session Amnesia — duplicates diverge silently
- Guide: "Extract shared utilities first"

**RECOMMENDATION:** Option C — move to spawn_backend.go (it's part of backend determination domain), then clean up the cmd/orch duplicate as a separate follow-up task. The extraction itself should not modify files outside pkg/orch to keep the change set focused.

### Fork 4: Test file distribution

**Options:**
- A: Split extraction_test.go into domain-specific test files
- B: Keep extraction_test.go and add new test files for extracted code

**SUBSTRATE:**
- Guide: "Tests follow handlers" — test files should co-locate with source

**RECOMMENDATION:** Option A — split test files. Each domain file gets its own test file. The current extraction_test.go maps cleanly:
- `TestDetermineSpawnBackend_*` (7 tests) → spawn_backend_test.go
- `TestIsArchitectIssue`, `TestExtractSearchTerms` → spawn_preflight_test.go
- `TestDetermineSpawnTier_*` → spawn_inference_test.go

---

## Recommendations

⭐ **RECOMMENDED:** Full 7-domain extraction with rename

### Implementation Plan

**Phase 1: Types first (must be first — other files depend on these types)**

Create `spawn_types.go` (~80 lines):
```
SpawnInput struct
SpawnContext struct
ResolvedSpawnResult struct
GapCheckResult struct
```

**Phase 2: Domain extractions (can be done in parallel after Phase 1)**

1. **`spawn_inference.go`** (~240 lines):
   - InferSkillFromIssueType, DetermineSpawnTier, inferTierFromTask
   - containsAny, validateModeModelCombo
   - inferSkillFromBeadsIssue, inferMCPFromBeadsIssue

2. **`spawn_preflight.go`** (~210 lines):
   - RunPreFlightChecks
   - buildArchitectVerifier, buildArchitectFinder
   - FindPriorArchitectReview, extractSearchTerms, isArchitectIssue

3. **`spawn_kb_context.go`** (~320 lines):
   - GatherSpawnContext
   - runPreSpawnKBCheck, runPreSpawnKBCheckFull
   - checkGapGating, recordGapForLearning
   - extractPrimaryModelPath

4. **`spawn_backend.go`** (~170 lines):
   - DetermineSpawnBackend
   - isInfrastructureWork, IsInfrastructureWork

5. **`spawn_beads.go`** (~180 lines):
   - SetupBeadsTracking, determineBeadsID
   - CreateBeadsIssue, resolveShortBeadsID

6. **`spawn_design.go`** (~100 lines):
   - LoadDesignArtifacts, readDesignArtifacts
   - extractDesignNotes, extractSection

**Phase 3: Test redistribution**

Split extraction_test.go:
- `spawn_backend_test.go` — 7 DetermineSpawnBackend tests (~200 lines)
- `spawn_preflight_test.go` — isArchitectIssue + extractSearchTerms tests (~115 lines)
- `spawn_inference_test.go` — DetermineSpawnTier tests (~40 lines)
- Remaining test stubs for new domains can be added later

**Phase 4: Rename + cleanup**
- Rename `extraction.go` → `spawn_pipeline.go`
- Rename `extraction_test.go` (after splitting) → `spawn_pipeline_test.go` (if anything remains)

### What remains in spawn_pipeline.go (~330 lines)

The "thin orchestrator layer" — pipeline steps that compose the spawn flow:
- `ResolveProjectDirectory` — resolve working directory
- `LoadSkillAndGenerateWorkspace` — load skill + generate workspace
- `CheckAndAutoSwitchAccount` — auto-switch accounts (single cohesive function, 68 lines)
- `ResolveAndValidateModel` — resolve model aliases
- `ResolveSpawnSettings` — centralized settings resolution
- `ExtractBugReproInfo` — extract bug info (14 lines)
- `BuildUsageInfo` — build usage info (13 lines)
- `BuildSpawnConfig` — construct spawn config (67 lines)
- `ValidateAndWriteContext` — validate + atomic write (67 lines)
- `checkWorkspaceExists` — workspace validation (26 lines)
- `dirExists` — filesystem utility (7 lines)
- `truncate` — string utility (6 lines)

### Expected outcome

| File | Lines | Domain |
|------|-------|--------|
| spawn_pipeline.go (née extraction.go) | ~330 | Core pipeline steps |
| spawn_types.go | ~80 | Shared types |
| spawn_inference.go | ~240 | Skill/tier/model inference |
| spawn_preflight.go | ~210 | Pre-flight validation gates |
| spawn_kb_context.go | ~320 | KB context + gap analysis |
| spawn_backend.go | ~170 | Backend routing + infra detection |
| spawn_beads.go | ~180 | Beads issue lifecycle |
| spawn_design.go | ~100 | Design artifact reading |

All files under 400 lines. Total code unchanged (just redistributed).

### Discovered work: cmd/orch/spawn_cmd.go duplicate cleanup

The duplicate `isInfrastructureWork` in cmd/orch/spawn_cmd.go should be removed and the call at line 664 changed to `orch.IsInfrastructureWork`. This is a separate follow-up task (different package, different change set).

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves recurring hotspot violations for extraction.go (2+ probes)

**Suggested blocks keywords:**
- extraction.go
- pkg/orch extraction
- spawn pipeline decomposition

---

## Acceptance Criteria

- [ ] Each extracted file compiles: `go build ./cmd/orch/`
- [ ] All tests pass: `go test ./pkg/orch/...`
- [ ] extraction.go (renamed) is under 500 lines
- [ ] No file exceeds 400 lines
- [ ] `orch hotspot` no longer flags extraction.go as CRITICAL

## File Targets (for implementation agent)

**Create:** spawn_types.go, spawn_inference.go, spawn_preflight.go, spawn_kb_context.go, spawn_backend.go, spawn_beads.go, spawn_design.go, spawn_backend_test.go, spawn_preflight_test.go, spawn_inference_test.go

**Modify:** extraction.go (remove extracted code, rename to spawn_pipeline.go)

**Delete:** extraction_test.go (contents distributed to domain test files)

## Out of Scope

- Modifying cmd/orch/spawn_cmd.go (isInfrastructureWork duplicate cleanup)
- Adding new tests beyond redistribution
- Changing function signatures or behavior
