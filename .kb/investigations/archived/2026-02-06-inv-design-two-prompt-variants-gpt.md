<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed two concrete SPAWN_CONTEXT.md prompt variants for GPT-5.3-codex to extract doctor.go (2369 lines) into category-based files.

**Evidence:** Analyzed doctor.go structure, identified 7 natural extraction units by check category, mapped shared dependencies, and produced two ready-to-use prompt files.

**Knowledge:** doctor.go splits cleanly by check category: types+main (~320 lines), liveness checks (~580 lines), correctness checks (~330 lines), sessions cross-ref (~270 lines), config drift (~220 lines), watch+daemon (~430 lines), fix+start helpers (~220 lines). Shared types (ServiceStatus, DoctorReport) and flags must be extracted first.

**Next:** Run the experiment - feed each variant to GPT-5.3-codex and compare extraction quality, correctness, and adherence to the code-extraction-patterns guide.

**Authority:** implementation - Prompt design within established experiment framework, no architectural impact

---

# Investigation: Design Two Prompt Variants for GPT-5.3-codex Code Extraction

**Question:** What are two effective alternative prompt designs for GPT-5.3-codex to extract doctor.go (2369 lines) into separate files split by check category?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation             | Relationship | Verified | Conflicts |
| ------------------------- | ------------ | -------- | --------- |
| N/A - novel investigation | -            | -        | -         |

**Relationship types:** extends, confirms, contradicts, deepens

---

## Findings

### Finding 1: doctor.go has 7 natural extraction units by check category

**Evidence:** Analyzing the file structure reveals clear domain boundaries:

| Unit                    | Lines | Content                                                                                                                                                                                                                                                                                           |
| ----------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `doctor.go` (main)      | ~320  | Types (ServiceStatus, DoctorReport, BinaryStatus), flags, cobra command, `runDoctor()`, `printDoctorReport()`                                                                                                                                                                                     |
| `doctor_liveness.go`    | ~580  | `checkOpenCode`, `checkOrchServe`, `checkWebUI`, `checkOvermindServices`, `checkBeadsDaemon`, `checkStaleBinary`, `checkEcosystemBinary`, `checkAllEcosystemBinaries`, `checkStalledSessions`                                                                                                     |
| `doctor_correctness.go` | ~330  | `checkBeadsIntegrity`, `checkRegistryReconciliation` + `RegistryReconcileResult` type, `checkDockerBackend`                                                                                                                                                                                       |
| `doctor_sessions.go`    | ~270  | `SessionsCrossReferenceReport`, `runSessionsCrossReference`, `loadSessionRegistry`, `isSessionInRegistry`, `printSessionsCrossReferenceReport`                                                                                                                                                    |
| `doctor_config.go`      | ~220  | `ConfigDrift`, `ConfigDriftReport`, `runConfigDriftCheck`, `checkPlistDrift`, `parsePlistValues`, `DocDebtReport`, `runDocDebtCheck`                                                                                                                                                              |
| `doctor_watch.go`       | ~430  | `runDoctorWatch`, `runHealthCheckWithNotifications`, `countUnhealthy`, `DoctorDaemonConfig`, `DoctorDaemonIntervention`, `DoctorDaemonLogger`, `runDoctorDaemon`, `runDaemonHealthCycle`, `killOrphanedViteProcesses`, `killLongRunningBdProcesses`, `restartCrashedServices`, `parseElapsedTime` |
| `doctor_install.go`     | ~120  | `getDoctorPlistPath`, `runDoctorInstall`, `runDoctorUninstall`, doctorInstallCmd, doctorUninstallCmd                                                                                                                                                                                              |

**Source:** `cmd/orch/doctor.go` - full 2369-line analysis

**Significance:** Clean split boundaries mean the prompt can give precise line ranges and function lists. The shared types in the main file must be extracted first (per code-extraction-patterns guide).

---

### Finding 2: Shared dependencies create extraction ordering constraints

**Evidence:** Several types and variables are used across multiple extraction units:

- `ServiceStatus` struct: used by all check functions
- `DoctorReport` struct: used by `runDoctor()`, `printDoctorReport()`, watch mode, daemon mode
- `BinaryStatus`, `EcosystemBinariesStatus`: used by staleness checks only
- Flag variables (`doctorFix`, `doctorVerbose`, etc.): used across all modes
- `DefaultWebPort`, `DefaultServePort` constants: used by multiple checks

**Source:** Cross-referencing function signatures and variable usage across `doctor.go`

**Significance:** The "types and flags in main file" pattern from the code-extraction-patterns guide applies directly. GPT-5.3-codex must be told to keep shared types in the main file.

---

### Finding 3: Current SPAWN_CONTEXT template is Claude-centric

**Evidence:** The existing template includes:

- Claude-specific tool references (Task tool, Explore subagent, AskUserQuestion)
- References to `bd comment`, `bd create`, beads tracking
- Investigation skill workflow with D.E.K.N. format
- Worker-base authority delegation patterns
- References to `kb create`, `kb quick` commands

These are all Claude Code ecosystem concepts that GPT-5.3-codex would not understand.

**Source:** SPAWN_CONTEXT.md template from this spawn

**Significance:** Variant B must strip all of these and replace with universal programming concepts. Variant A can keep some structure but needs explicit verification steps instead of relying on Claude's tool ecosystem.

---

## Synthesis

**Key Insights:**

1. **Extraction is well-scoped** - doctor.go has clear category boundaries with minimal cross-cutting concerns. The main challenge is handling shared types correctly, not identifying boundaries.

2. **Two complementary prompt strategies** - Variant A (checklist + gates) tests whether explicit step-by-step instructions with forced verification improve extraction quality. Variant B (model-agnostic) tests whether removing Claude-specific assumptions helps GPT-5.3-codex focus on the actual code task.

3. **Both variants need the same domain knowledge** - The file structure analysis, extraction ordering, and naming conventions are constant. What changes is the instructional frame around that knowledge.

**Answer to Investigation Question:**

Two prompt variants designed and produced as ready-to-use files:

- **Variant A** (`SPAWN_CONTEXT_VARIANT_A.md`): Checklist-driven with explicit verification gates after each extraction step
- **Variant B** (`SPAWN_CONTEXT_VARIANT_B.md`): Model-agnostic rewrite removing all Claude/orchestrator assumptions

---

## Structured Uncertainty

**What's tested:**

- ✅ doctor.go line count and structure verified (read full 2369 lines)
- ✅ Extraction boundaries verified by analyzing function signatures and cross-references
- ✅ Shared dependency analysis verified by tracing type usage across functions

**What's untested:**

- ⚠️ Whether GPT-5.3-codex will follow either prompt correctly (that's the experiment)
- ⚠️ Whether the 7-file split is optimal (could be 5 or 9 depending on preferences)
- ⚠️ Whether GPT-5.3-codex handles Go package-level visibility correctly without explicit instruction

**What would change this:**

- If doctor.go has been modified since this analysis, line ranges would be wrong
- If GPT-5.3-codex has different context window limits, prompts may need truncation

---

## Implementation Recommendations

| Recommendation                               | Authority      | Rationale                                                 |
| -------------------------------------------- | -------------- | --------------------------------------------------------- |
| Run both variants and compare output quality | implementation | Experiment execution within established two-model pattern |

### Recommended Approach ⭐

**Run both variants sequentially** - Feed each SPAWN_CONTEXT to GPT-5.3-codex in separate sessions, then compare: (a) compilation success, (b) test pass rate, (c) adherence to naming conventions, (d) handling of shared types.

### Deliverables

Two files produced in `.orch/workspace/`:

1. `SPAWN_CONTEXT_VARIANT_A.md` - Checklist with verification gates
2. `SPAWN_CONTEXT_VARIANT_B.md` - Model-agnostic rewrite

---

## References

**Files Examined:**

- `cmd/orch/doctor.go` - Full 2369-line source file for extraction
- `cmd/orch/doctor_test.go` - Existing test file
- `.kb/guides/code-extraction-patterns.md` - Extraction patterns guide
- `~/.opencode/skill/feature-impl/SKILL.md` - Feature implementation skill

---

## Investigation History

**2026-02-06:** Investigation started

- Read doctor.go (2369 lines), identified 7 extraction units
- Read code-extraction-patterns guide for established conventions
- Designed two prompt variants

**2026-02-06:** Investigation completed

- Status: Complete
- Key outcome: Two ready-to-use SPAWN_CONTEXT.md files for GPT-5.3-codex experiment
