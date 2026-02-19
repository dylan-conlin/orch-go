# Probe: extraction.go Hotspot Analysis

**Model:** Code Extraction Patterns (`../.kb/models/extract-patterns.md`)
**Date:** 2026-02-19
**Status:** Complete
**Triggered By:** extraction.go is #1 hotspot — 2011 lines, 22 commits in 28 days

---

## Question

Does extraction.go follow the model's predicted failure pattern for large files? Specifically:
1. Does it exceed the 800-line gate?
2. Does it contain identifiable "Cohesive Extraction Units"?
3. Is the Phase-based Extraction strategy (Shared → Domain → Sub-domain) applicable?

---

## What I Tested

**Static analysis of `pkg/orch/extraction.go` (2011 lines):**
- Mapped all function definitions with line ranges
- Identified responsibility domains by import dependencies and logical grouping
- Counted commits in last 28 days: `git log --oneline --since="2026-01-22" -- pkg/orch/extraction.go`
- Analyzed coupling between domains via shared types and cross-function calls
- Checked existing sibling files (completion.go, flags.go) for extraction patterns already in use

**Build and test verification:**
- `go build ./cmd/orch/` — passes
- `go test ./pkg/orch/...` — passes (0.010s)

---

## What I Observed

### 1. 800-Line Gate: CONFIRMED — 2.5x over limit

At 2011 lines, extraction.go is **2.5x the 800-line threshold**. The model predicts this creates "Context Noise" degrading agent performance — confirmed by the 22 commits in 28 days, many of which are fixes to previous fixes (revert-then-fix pattern visible in git log).

### 2. Cohesive Extraction Units: CONFIRMED — 9 distinct domains

| # | Domain | Lines | Line Count | Key Functions |
|---|--------|-------|------------|---------------|
| 1 | Types & Constants | 1-99 | 99 | SpawnInput, SpawnContext, ResolvedSpawnResult, GapCheckResult, regexes |
| 2 | Skill/Tier Inference | 101-221 | 121 | InferSkillFromIssueType, DetermineSpawnTier, inferTierFromTask, parseSessionScope, containsAny |
| 3 | Account Management | 223-290 | 68 | CheckAndAutoSwitchAccount |
| 4 | Backend Resolution | 292-844, 1823-1896 | 621 | validateModeModelCombo, inferSkillFromBeadsIssue, inferMCPFromBeadsIssue, DetermineSpawnBackend, isInfrastructureWork, IsInfrastructureWork |
| 5 | Pre-flight & Validation | 351-593 | 243 | RunPreFlightChecks, ResolveProjectDirectory, LoadSkillAndGenerateWorkspace, SetupBeadsTracking, ResolveAndValidateModel |
| 6 | Spawn Settings & Context | 595-725 | 131 | ResolveSpawnSettings, GatherSpawnContext, ExtractBugReproInfo, BuildUsageInfo |
| 7 | Config Building & Dispatch | 743-999 | 257 | LoadDesignArtifacts, BuildSpawnConfig, ValidateAndWriteContext, DispatchSpawn |
| 8 | Spawn Mode Implementations | 1130-1572 | 443 | runSpawnInline, runSpawnHeadless, startHeadlessSession, runSpawnTmux, runSpawnClaude |
| 9 | Helpers & Utilities | 1001-1128, 1574-2011 | 424 | stripANSI, formatSessionTitle, addGapAnalysisToEventData, formatContextQualitySummary, printSpawnSummaryWithGapWarning, runPreSpawnKBCheck, checkGapGating, recordGapForLearning, readDesignArtifacts, extractDesignNotes, truncate, resolveShortBeadsID, checkWorkspaceExists, determineBeadsID, CreateBeadsIssue |

### 3. Phase-based Strategy: EXTENDS model — `package orch` has different constraints than `package main`

The model's Phase 1 ("extract shared utilities first") assumes `package main` where all files share visibility. `pkg/orch` is an *exported package* — extraction here means:
- Exported functions stay accessible via `orch.FunctionName`
- No import changes needed for callers (all in same package)
- BUT unexported helpers become visible to new files in same package automatically

This is actually *easier* than the cmd/orch extractions documented in the model because there are no circular dependency risks within a flat package.

---

## Model Impact

**CONFIRMS:**
- The 800-Line Gate constraint — extraction.go at 2.5x threshold shows severe degradation (22 commits/28 days, fix-on-fix pattern)
- Cohesive Extraction Units exist — 9 clear domains with measurable boundaries
- Phase-based strategy applies (Shared → Domain → Sub-domain)

**EXTENDS:**
- The model doesn't specifically address `pkg/` package extraction (only `cmd/orch/` and Svelte). pkg/orch extraction is *simpler* because it's a flat package with no circular dependency risk
- New pattern: **Spawn Mode extraction** — 4 mode implementations (inline/headless/tmux/claude) each ~80-150 lines, sharing identical event logging boilerplate. This is a "Domain Handler" extraction where the shared infrastructure (event logging, summary printing) should be extracted first
- The "fix-on-fix" anti-pattern (visible in git log: fix→revert→fix→fix) is a stronger signal than raw line count. The model should note that **high churn rate + high line count = extraction emergency**, not just either alone

**CONTRADICTS:** Nothing — findings align with model predictions.
