<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** extraction.go (2011 lines) contains 9 distinct responsibility domains that should be extracted into focused modules. This is the #1 hotspot in orch-go with 22 commits in 28 days showing a fix-on-fix anti-pattern.

**Evidence:** Static analysis reveals clear domain boundaries: Types/Constants (1-99), Skill/Tier Inference (101-221), Account Management (223-290), Backend Resolution (292-844 + 1823-1896), Pre-flight/Validation (351-593), Spawn Settings/Context (595-725), Config Building/Dispatch (743-999), Spawn Mode Implementations (1130-1572), Helpers/Utilities (1001-1128 + 1574-2011).

**Knowledge:** The package already has extraction precedent (completion.go, flags.go). Extraction within `pkg/orch` is simpler than `cmd/orch/` extractions because it's a flat package with no circular dependency risk — all files share package-level visibility.

**Next:** Implement Phase 0 extraction (spawn_modes.go, spawn_helpers.go) as immediate wins — these are self-contained with no caller changes needed.

---

# Investigation: extraction.go Structure Analysis and Extraction Plan

**Question:** What responsibility domains exist in extraction.go and how should they be extracted into focused modules following the daemon.go extraction pattern?

**Started:** 2026-02-19
**Updated:** 2026-02-19
**Owner:** Worker Agent (og-arch-orientation-frame-extraction-19feb-6cf9)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: extraction.go Contains 9 Distinct Responsibility Domains

**Evidence:** Line-by-line analysis reveals these domains:

| # | Domain | Lines | Line Count | Coupling Level |
|---|--------|-------|------------|----------------|
| 1 | Types & Constants | 1-99 | 99 | High (shared by all) |
| 2 | Skill/Tier Inference | 101-221 | 121 | Low (pure functions) |
| 3 | Account Management | 223-290 | 68 | Low (self-contained) |
| 4 | Backend Resolution | 292-306, 743-896 | ~175 | Medium (uses model, config) |
| 5 | Pre-flight & Validation | 351-593 | 243 | Medium (uses gates, verify, beads) |
| 6 | Spawn Settings & Context | 595-725 | 131 | Medium (uses spawn pkg) |
| 7 | Config Building & Dispatch | 864-999 | 136 | Medium (uses SpawnContext) |
| 8 | Spawn Mode Implementations | 1130-1572 | 443 | Medium (uses opencode, tmux, events) |
| 9 | Helpers & Utilities | 1001-1128, 1574-2011 | ~424 | Mixed (some pure, some with dependencies) |

**Source:** `pkg/orch/extraction.go:1-2011`

**Significance:** The domains have identifiable boundaries. Domain 8 (Spawn Modes) is the largest at 443 lines and contains 4 mode implementations with duplicated event logging boilerplate — the highest extraction ROI.

---

### Finding 2: Spawn Mode Implementations Share Massive Boilerplate

**Evidence:** The four spawn mode functions share nearly identical patterns:

| Mode Function | Lines | Pattern Elements |
|---------------|-------|-----------------|
| `runSpawnInline` | 1130-1221 (92) | Create session → send prompt → event log → gap warning → summary |
| `runSpawnHeadless` | 1223-1314 (92) | Create session → retry → event log → gap warning → summary |
| `runSpawnTmux` | 1360-1511 (152) | Ensure session → create window → send prompt → event log → gap warning → summary |
| `runSpawnClaude` | 1513-1572 (60) | SpawnClaude → event log → gap warning → summary |
| `startHeadlessSession` | 1316-1358 (43) | (helper for runSpawnHeadless) |

All four modes:
1. Build event data maps with identical fields (skill, task, workspace, beads_id, spawn_mode, model, no_track, skip_artifact_check)
2. Call `addGapAnalysisToEventData` and `addUsageInfoToEventData`
3. Call `printSpawnSummaryWithGapWarning`
4. Print nearly identical spawn summary blocks

**Source:** `pkg/orch/extraction.go:1130-1572`

**Significance:** This is the highest-value extraction target. A shared `logSpawnEvent` function and `printSpawnSummary` helper would eliminate ~100 lines of duplication AND make future additions to event data automatic across all modes.

---

### Finding 3: Backend Resolution Is Split Across File

**Evidence:** Backend/infrastructure detection logic is spread non-contiguously:

| Function | Lines | Purpose |
|----------|-------|---------|
| `validateModeModelCombo` | 292-306 | Backend+model validation |
| `DetermineSpawnBackend` | 743-844 | Main backend resolution (100 lines) |
| `isInfrastructureWork` | 1823-1891 | Infrastructure keyword matching |
| `IsInfrastructureWork` | 1893-1896 | Exported wrapper |

These are logically one domain but scattered across 600+ lines of the file.

**Source:** Various locations in extraction.go

**Significance:** Extracting these into `spawn_backend.go` would make the backend resolution logic self-contained and easier to reason about. The 100-line `DetermineSpawnBackend` function is where most of the fix-on-fix commits have landed.

---

### Finding 4: Package Already Has Extraction Precedent

**Evidence:** Existing extracted files in `pkg/orch/`:

| File | Lines | Domain |
|------|-------|--------|
| `completion.go` | 266 | Completion backlog + explain-back gate |
| `completion_test.go` | - | Tests for completion.go |
| `flags.go` | 46 | Mode flag registration |
| `flags_test.go` | - | Tests for flags.go |
| `extraction.go` | 2011 | Everything else (the "god file") |
| `extraction_test.go` | 331 | Tests for extraction.go |

**Source:** `ls pkg/orch/*.go`

**Significance:** completion.go and flags.go prove the extraction pattern works in this package. extraction.go is the remaining accumulation file that was meant to be temporary (see original package comment: "extracted from cmd/orch/spawn_cmd.go").

---

### Finding 5: Callers Import Only 4 Files

**Evidence:** Files that import `pkg/orch`:

| Caller | Functions Used | Domain |
|--------|---------------|--------|
| `cmd/orch/spawn_cmd.go` | SpawnInput, SpawnContext, all pipeline functions | Spawn pipeline (primary) |
| `cmd/orch/rework_cmd.go` | SpawnInput, SpawnContext, most pipeline functions | Spawn pipeline (rework) |
| `cmd/orch/complete_cmd.go` | RunExplainBackGate, RecordGate2Checkpoint | Completion (already in completion.go) |
| `cmd/orch/serve_agents_status.go` | AgentInfo, DetectCompletionBacklog | Completion (already in completion.go) |

**Source:** `grep -r "pkg/orch" cmd/orch/`

**Significance:** spawn_cmd.go and rework_cmd.go are the only callers of extraction.go functions. Since all files are in the same package (`orch`), extraction won't require any caller changes — exported symbols stay accessible package-wide.

---

### Finding 6: Helper Functions Are a Mix of Pure and Effectful

**Evidence:** The helpers/utilities domain (424 lines) contains two distinct types:

**Pure functions (no side effects, easy to extract):**
- `stripANSI` (3 lines)
- `formatSessionTitle` (5 lines)
- `formatContextQualitySummary` (42 lines)
- `extractPrimaryModelPath` (6 lines)
- `extractDesignNotes` (17 lines)
- `extractSection` (23 lines)
- `truncate` (5 lines)
- `resolveShortBeadsID` (15 lines)
- `containsAny` (7 lines)
- `parseSessionScope` (13 lines)

**Effectful functions (I/O, external calls):**
- `addGapAnalysisToEventData` (24 lines) — mutates map
- `addUsageInfoToEventData` (15 lines) — mutates map
- `printSpawnSummaryWithGapWarning` (16 lines) — writes to stderr
- `runPreSpawnKBCheck` / `runPreSpawnKBCheckFull` (77 lines) — runs kb CLI
- `checkGapGating` (24 lines) — reads gap state
- `recordGapForLearning` (33 lines) — writes to tracker file
- `readDesignArtifacts` (36 lines) — reads filesystem
- `checkWorkspaceExists` (26 lines) — reads filesystem
- `determineBeadsID` (19 lines) — calls beads
- `CreateBeadsIssue` (31 lines) — calls beads RPC/CLI

**Source:** `pkg/orch/extraction.go` various locations

**Significance:** Pure functions can be extracted with zero risk. Effectful functions need their dependencies to travel with them.

---

### Finding 7: The Fix-on-Fix Anti-Pattern Is Concentrated in Backend Resolution

**Evidence:** Git log shows a clear pattern in the backend resolution area:

```
a8e340918 fix: remove model-backend conflation in extraction.go
0d344aced Revert "fix: remove model-backend conflation in extraction.go"
807441669 fix: remove model-based backend auto-selection (orch-go-1011)
d9ded28f4 fix: user config now overrides infrastructure escape hatch (orch-go-1045)
71bca206c fix: explicit --model flag now prevents infrastructure escape hatch override
93b97f63a fix: DetermineSpawnBackend now respects user config backend setting
3f421aa71 feat: treat user default_model as explicit backend signal (orch-go-1049)
3d722d3c6 fix: respect explicit spawn_mode only (orch-go-1047)
```

8 of 22 commits (36%) target the backend resolution logic. The `DetermineSpawnBackend` function has been modified, reverted, and re-fixed multiple times.

**Source:** `git log --oneline --since="2026-01-22" -- pkg/orch/extraction.go`

**Significance:** The backend resolution domain is the primary source of churn. Extracting it would isolate the instability to a focused file where changes have clear boundaries and tests can target the specific logic.

---

## Synthesis

**Key Insights:**

1. **Extract by Stability, Not Just Size** — While Spawn Mode Implementations (443 lines) are the largest domain, Backend Resolution (~175 lines scattered) is the most unstable. Both should be prioritized but for different reasons: size vs churn.

2. **Dedup Before Extract** — The 4 spawn mode functions share ~100 lines of identical boilerplate (event logging, summary printing). Deduplicating this boilerplate first reduces the total lines to extract and makes each mode function a cohesive 30-50 line unit.

3. **extraction.go Should Become the Pipeline Orchestrator** — After extraction, extraction.go should shrink to ~400-500 lines containing: types/constants (99 lines), the pipeline functions that spawn_cmd.go calls in sequence, and the dispatch function. This mirrors the daemon.go pattern where the orchestrator imports and coordinates extracted modules.

4. **pkg/orch Extraction Is Simpler Than cmd/orch** — No circular dependency risk, no import changes needed for callers. This is a flat package where file splitting is purely organizational.

**Answer to Investigation Question:**

extraction.go contains 9 responsibility domains that should be extracted into focused modules:

| Priority | New File | Contains | Lines Moved | Reason |
|----------|----------|----------|-------------|--------|
| P0 | `spawn_modes.go` | runSpawnInline, runSpawnHeadless, startHeadlessSession, runSpawnTmux, runSpawnClaude | ~443 | Largest domain, self-contained |
| P0 | `spawn_helpers.go` | formatSessionTitle, addGapAnalysisToEventData, addUsageInfoToEventData, formatContextQualitySummary, printSpawnSummaryWithGapWarning, stripANSI | ~110 | Pure helpers used by spawn modes |
| P1 | `spawn_backend.go` | DetermineSpawnBackend, validateModeModelCombo, isInfrastructureWork, IsInfrastructureWork | ~175 | Highest churn domain, concentrate instability |
| P1 | `spawn_context.go` | GatherSpawnContext, runPreSpawnKBCheck, runPreSpawnKBCheckFull, checkGapGating, recordGapForLearning, extractPrimaryModelPath | ~160 | KB context gathering, self-contained |
| P1 | `spawn_beads.go` | SetupBeadsTracking, determineBeadsID, CreateBeadsIssue, resolveShortBeadsID, inferSkillFromBeadsIssue, inferMCPFromBeadsIssue | ~130 | Beads integration, external dependency |
| P2 | `spawn_design.go` | LoadDesignArtifacts, readDesignArtifacts, extractDesignNotes, extractSection | ~80 | Design workspace helpers |
| P2 | `skill_inference.go` | InferSkillFromIssueType, DetermineSpawnTier, inferTierFromTask, parseSessionScope, containsAny | ~120 | Pure functions, low coupling |

**After all extractions, extraction.go should contain (~400-500 lines):**
- Types & constants (SpawnInput, SpawnContext, ResolvedSpawnResult, GapCheckResult) — 99 lines
- Pipeline functions called by spawn_cmd.go:
  - `RunPreFlightChecks` — 43 lines
  - `ResolveProjectDirectory` — 25 lines
  - `LoadSkillAndGenerateWorkspace` — 44 lines
  - `ResolveAndValidateModel` — 36 lines
  - `ResolveSpawnSettings` — 33 lines
  - `ExtractBugReproInfo` — 14 lines
  - `BuildUsageInfo` — 12 lines
  - `BuildSpawnConfig` — 44 lines
  - `ValidateAndWriteContext` — 56 lines
  - `DispatchSpawn` — 31 lines
  - `CheckAndAutoSwitchAccount` — 68 lines
- `truncate`, `dirExists` helpers — 10 lines

---

## Structured Uncertainty

**What's tested:**
- ✅ Line counts are accurate (verified: manual inspection of extraction.go)
- ✅ Build passes after current state (`go build ./cmd/orch/`)
- ✅ Tests pass (`go test ./pkg/orch/...` — 0.010s)
- ✅ Existing extraction pattern works (completion.go, flags.go are functional)
- ✅ Domain boundaries are real (verified: no cross-domain helper calls within proposed extraction units)

**What's untested:**
- ⚠️ Actual extraction won't break callers (callers reference by function name, not file — low risk in same package)
- ⚠️ Test file splitting (extraction_test.go has tests for DetermineSpawnBackend and DetermineSpawnTier — these move with their functions)
- ⚠️ The dedup-before-extract approach for spawn modes (event logging boilerplate) hasn't been prototyped

**What would change this:**
- If spawn mode functions have more coupling than apparent (would require keeping them together)
- If a spawn_cmd.go refactor is planned simultaneously (would change the pipeline API)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Phased Extraction by Stability + Size** — Extract the most self-contained domains first, then domains with external dependencies, finally clean up the orchestrator.

**Why this approach:**
- P0 extractions (spawn_modes + spawn_helpers) remove 553 lines with zero caller changes
- P1 extractions isolate the unstable backend resolution logic, reducing fix-on-fix risk
- Each phase is independently testable: `go build ./cmd/orch/ && go test ./pkg/orch/...`
- Follows the proven pattern from daemon.go extraction and cmd/orch extractions

**Trade-offs accepted:**
- Multiple small commits instead of one big refactor
- extraction.go temporarily imports from sibling files during transition (but all in same package, so no actual imports needed)

**Implementation sequence:**

**Phase 0: Dedup Spawn Mode Boilerplate (Pre-extraction)**
Before extracting spawn modes, create shared helpers:
1. `logSpawnEvent(logger, eventType, sessionID, eventData)` — consolidate event logging
2. `buildSpawnEventData(skillName, task, cfg, spawnMode)` — consolidate event data construction
3. `printSpawnResult(mode, sessionID, cfg, beadsID)` — consolidate summary printing

This reduces each spawn mode function by ~30-40 lines, making them 30-60 lines each.

**Phase 1: Zero-Dependency Extractions (P0)**
1. `spawn_helpers.go` — Extract pure helper functions (formatSessionTitle, stripANSI, format/print helpers)
2. `spawn_modes.go` — Extract all 5 spawn mode functions (now deduplicated)

**Phase 2: Domain Extractions (P1)**
3. `spawn_backend.go` — Extract DetermineSpawnBackend + isInfrastructureWork + validateModeModelCombo
4. `spawn_context.go` — Extract GatherSpawnContext + KB check + gap gating
5. `spawn_beads.go` — Extract SetupBeadsTracking + beads helpers

**Phase 3: Cleanup (P2)**
6. `spawn_design.go` — Extract design artifact helpers
7. `skill_inference.go` — Extract skill/tier inference functions
8. Move tests from extraction_test.go to corresponding test files
9. Rename `extraction.go` → `spawn_pipeline.go` (better reflects its role as orchestrator)

### Alternative Approaches Considered

**Option B: Extract by caller alignment**
- Extract based on which functions spawn_cmd.go vs rework_cmd.go call
- **Pros:** Optimizes for caller comprehension
- **Cons:** Both callers use the same pipeline — this doesn't create meaningful separation

**Option C: Move everything to pkg/spawn**
- Merge extraction.go functions into the existing pkg/spawn package
- **Pros:** Eliminates the pkg/orch package entirely
- **Cons:** pkg/spawn is already well-scoped for context generation; adding pipeline orchestration would bloat it. Also, extraction.go imports from pkg/spawn — this would create circular dependencies

### Success Criteria

- ✅ extraction.go < 500 lines after all extractions (rename to spawn_pipeline.go)
- ✅ All tests pass after each extraction phase
- ✅ No public API changes (same `orch.FunctionName` imports work for callers)
- ✅ Each new file is <300 lines and has single responsibility
- ✅ Backend resolution churn isolated to spawn_backend.go
- ✅ `go build ./cmd/orch/ && go vet ./cmd/orch/ && go test ./pkg/orch/...` passes at every step

### Implementation Details

**Things to watch out for:**
- ⚠️ Test file splitting: extraction_test.go tests DetermineSpawnBackend extensively — those tests move to spawn_backend_test.go
- ⚠️ The `headlessSpawnResult` type is only used by runSpawnHeadless/startHeadlessSession — it moves to spawn_modes.go
- ⚠️ `ansiRegex` and `sessionScopeRegex` are package-level vars — they stay accessible regardless of which file defines them
- ⚠️ The P0 dedup step (Phase 0) should be done as a separate commit before file splitting, to keep diffs clean

**What to implement first:**
- Phase 0 (dedup) can be done in a single commit
- Phase 1 (spawn_modes.go + spawn_helpers.go) is the highest-value extraction
- These two phases alone reduce extraction.go by ~553 lines (27%)

---

## References

**Files Examined:**
- `pkg/orch/extraction.go` — Main analysis target (2011 lines)
- `pkg/orch/extraction_test.go` — Tests (331 lines)
- `pkg/orch/completion.go` — Reference extraction (266 lines)
- `pkg/orch/flags.go` — Reference extraction (46 lines)
- `cmd/orch/spawn_cmd.go` — Primary caller
- `cmd/orch/rework_cmd.go` — Secondary caller
- `cmd/orch/complete_cmd.go` — Uses completion.go functions
- `cmd/orch/serve_agents_status.go` — Uses completion.go functions

**Commands Run:**
```bash
wc -l pkg/orch/extraction.go  # 2011
git log --oneline --since="2026-01-22" -- pkg/orch/extraction.go  # 22 commits
go build ./cmd/orch/  # passes
go test ./pkg/orch/...  # passes (0.010s)
```

**Related Artifacts:**
- `.kb/investigations/2026-01-04-inv-design-analyze-pkg-daemon-daemon.md` — daemon.go extraction plan (reference pattern)
- `.kb/guides/code-extraction-patterns.md` — Extraction guide
- `.kb/models/extract-patterns/model.md` — Extraction patterns model
- `.kb/models/extract-patterns/probes/2026-02-19-probe-extraction-go-hotspot-analysis.md` — Companion probe

---

## Investigation History

**2026-02-19 14:30:** Investigation started
- Initial question: Analyze extraction.go structure for extraction planning
- Context: extraction.go at 2011 lines is #1 hotspot with 22 commits in 28 days

**2026-02-19 15:00:** Static analysis complete
- Identified 9 distinct responsibility domains
- Found fix-on-fix anti-pattern concentrated in backend resolution
- Confirmed extraction precedent in package (completion.go, flags.go)

**2026-02-19 15:15:** Investigation completed
- Status: Complete
- Key outcome: Phased extraction plan with P0/P1/P2 priorities
- Highest-value target: spawn modes (443 lines) + spawn helpers (110 lines)
