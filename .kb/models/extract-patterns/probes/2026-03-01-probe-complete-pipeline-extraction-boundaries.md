# Probe: complete_pipeline.go Extraction Boundaries

**Model:** Code Extraction Patterns (`/Users/dylanconlin/Documents/personal/orch-go/.kb/models/extract-patterns/model.md`)
**Status:** Complete
**Date:** 2026-03-01
**Agent:** og-arch-complete-pipeline-go-01mar-363e

---

## Question

Does `complete_pipeline.go` (970 lines) follow the model's extraction hierarchy, and what are the natural cohesive extraction units within it?

## Model Claims Being Tested

1. **800-Line Gate:** Files >800 lines trigger extraction (model: "800 lines is the heuristic limit where Context Noise begins to degrade agent reasoning")
2. **Cohesive Extraction Units:** "Identifying groups of functions, types, and helpers that share a single infrastructure substrate"
3. **Phase-based Extraction:** Shared utilities → Domain handlers → Sub-domain infrastructure
4. **Target ~300-800 lines per file** (from guide)
5. **Package main Convenience:** File splitting within `package main` requires no import changes

## What I Tested

### 1. File Structure Analysis

Ran line counts and analyzed function boundaries:

```
complete_pipeline.go (970 lines total):
  Types (CompletionTarget, VerificationOutcome, AdvisoryResults): 26 lines
  resolveCompletionTarget():  136 lines (lines 54-189)
  executeVerificationGates(): 274 lines (lines 191-464) ← LARGEST
  runCompletionAdvisories():  196 lines (lines 466-661)
  executeLifecycleTransition(): 264 lines (lines 663-926)
  applySkipFilters():          43 lines (lines 928-970)
```

### 2. Coupling Analysis

Analyzed cross-file dependencies:

```
executeVerificationGates calls:
  - addApprovalComment (complete_postlifecycle.go)
  - applySkipFilters (same file)
  - state.GetLiveness, events.NewLogger, verify.* (packages)
  - completeForce, completeExplain, etc. (package vars in complete_cmd.go)

runCompletionAdvisories calls:
  - verify.ParseSynthesis, verify.CollectDiscoveredWork (packages)
  - verify.FindProbesForWorkspace (packages)
  - verify.FormatArchitecturalChoicesForCompletion (packages)
  - RunKnowledgeMaintenance (knowledge_maintenance.go)
  - orch.RunExplainBackGate (pkg/orch/)
  - orch.RecordGate2Checkpoint (pkg/orch/)
  - buildVerificationChecklist (complete_checklist.go)
  - RunHotspotAdvisoryForCompletion (complete_hotspot.go)
  - RunModelImpactAdvisory (complete_model_impact.go)
  - RunSynthesisCheckpoint (complete_synthesis.go)
  - UpdateHandoffAfterComplete (session_handoff.go)
  - completeForce, completeExplain, etc. (package vars)

executeLifecycleTransition calls:
  - maybeAutoCreateImplementationIssue (complete_architect.go)
  - collectCompletionTelemetry (complete_postlifecycle.go)
  - collectAccretionDelta (complete_postlifecycle.go)
  - activity.ExportToWorkspace (pkg/activity)
  - exportOrchestratorTranscript (complete_postlifecycle.go)
  - buildLifecycleManager (lifecycle_adapters.go)
  - hasGoChangesInRecentCommits (complete_postlifecycle.go)
  - runAutoRebuild (complete_postlifecycle.go)
  - detectNewCLICommands (complete_postlifecycle.go)
  - detectNotableChangelogEntries (complete_checklist.go)
  - events.NewLogger (packages)
  - invalidateServeCache (complete_postlifecycle.go)
```

### 3. Complete Subsystem File Distribution (current)

```
complete_cmd.go:            285 lines (CLI definition + thin orchestrator)
complete_pipeline.go:       970 lines (4 pipeline phases) ← TARGET
complete_actions.go:         52 lines (archiveWorkspace - dead code)
complete_checklist.go:      206 lines (checklist + changelog UI)
complete_cleanup.go:         43 lines (tmux cleanup)
complete_postlifecycle.go:  541 lines (post-lifecycle helpers)
complete_architect.go:      164 lines (auto-create impl issue)
complete_hotspot.go:        184 lines (hotspot advisory)
complete_model_impact.go:   238 lines (model impact advisory)
complete_synthesis.go:      120 lines (synthesis checkpoint)
complete_trust.go:           58 lines (trust calibration)
                           -----
Total:                    2,861 lines across 11 files
```

## What I Observed

### Confirms Model Claims

1. **800-Line Gate triggers extraction need:** At 970 lines, `complete_pipeline.go` is 21% above the 800-line threshold. The file has been modified 5 times since Feb 1, 2026, meaning it's actively accumulating changes. This confirms the model's prediction that >800-line files attract more fixes (fix-density 8).

2. **Cohesive extraction units exist:** The 4 pipeline phases have distinct responsibilities:
   - `resolveCompletionTarget` = identity resolution (filesystem + beads)
   - `executeVerificationGates` = gate checks + liveness (verification subsystem)
   - `runCompletionAdvisories` = advisory dispatching (informational outputs)
   - `executeLifecycleTransition` = state transition + side effects (lifecycle subsystem)

3. **Package main convenience holds:** All functions are in `package main`. Extraction to new files requires zero import changes. All cross-references are to functions in other `cmd/orch/*.go` files (same package) or `pkg/` packages.

### Extends Model

4. **Pipeline files need different extraction heuristics than monolithic files:** The model's hierarchy (shared utilities → domain handlers → sub-domain) was designed for monolithic `main.go`/`serve.go` extraction. `complete_pipeline.go` doesn't have shared utilities to extract first—it has 4 pipeline phases that are sequentially composed but internally cohesive. The extraction pattern here is "extract by pipeline phase" rather than "shared first, then domains."

5. **Prior extraction created the current structure:** The file was already decomposed from `runComplete()` into 4 typed phases (Feb 28, 2026 refactor). The current issue is that all 4 phases are in a single file, making it the largest file in the complete subsystem. The next extraction step is splitting phases into separate files.

6. **Coupling cluster metric (74) reflects the advisory fan-out:** `runCompletionAdvisories` calls 10+ different advisory functions scattered across 6+ files. This high coupling is inherent to the advisory dispatcher pattern—it's a coordination point, not a sign of poor design. Extracting advisories to their own file doesn't reduce coupling, but it does reduce the cognitive surface of `complete_pipeline.go`.

## Model Impact

**Confirms:**
- 800-line gate accurately predicts extraction need
- Cohesive extraction units within pipeline phases
- Package main convenience enables friction-free extraction

**Extends:**
- Model should document "pipeline phase extraction" as a distinct pattern alongside the existing hierarchy (shared → domain → sub-domain)
- Pipeline phases are extracted by responsibility boundary, not by shared-utility-first ordering
- Advisory dispatcher functions inherently have high coupling cluster scores; this is structural, not pathological

**Recommendation:**
Extract `executeVerificationGates` + `applySkipFilters` to `complete_verification.go` (~330 lines) and `executeLifecycleTransition` to `complete_lifecycle.go` (~270 lines), leaving `complete_pipeline.go` at ~370 lines with types + resolve + advisories.
