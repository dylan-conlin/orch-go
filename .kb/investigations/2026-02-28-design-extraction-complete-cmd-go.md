## Summary (D.E.K.N.)

**Delta:** complete_cmd.go (2,267 lines) contains 7 distinct responsibility clusters that map to 4 extraction phases — cmd/orch file splits for CLI plumbing, then pkg/completion for reusable completion logic.

**Evidence:** Function-level analysis shows runComplete() at 1,063 lines orchestrating: identifier resolution (108 lines), verification gate execution (250 lines), discovered work disposition (74 lines), advisory surfacing (40 lines), lifecycle transition (90 lines), post-lifecycle operations (140 lines), and telemetry/events (103 lines). Helper functions at lines 1432-2267 (~835 lines) have zero coupling to CLI flags.

**Knowledge:** The right extraction is NOT a monolithic pkg/completion/ package. It's a 4-phase split: (1) SkipConfig to pkg/verify, (2) post-lifecycle helpers to cmd/orch/complete_postlifecycle.go, (3) completion telemetry/advisory to cmd/orch/complete_telemetry.go, (4) runComplete() decomposition into a pipeline of typed phases in cmd/orch/complete_pipeline.go.

**Next:** Create 4 implementation issues (one per phase) with dependency chain. Phase 1 is independent; phases 2-4 can proceed in parallel after phase 1.

**Authority:** architectural - Cross-package extraction affecting verify, agent, and cmd/orch boundaries

---

# Investigation: Design Extraction of complete_cmd.go

**Question:** How should complete_cmd.go (2,267 lines, #1 hotspot) be decomposed to bring it under the 1,500-line accretion boundary while preserving the completion pipeline's correctness?

**Defect-Class:** unbounded-growth

**Started:** 2026-02-28
**Updated:** 2026-02-28
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None — create implementation issues
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/guides/code-extraction-patterns.md` | extends | Yes — same-package extraction pattern confirmed | None |
| `.kb/guides/completion.md` | extends | Yes — verification architecture matches | None |
| `.kb/guides/completion-gates.md` | extends | Yes — gate reference matches code | None |
| daemon.go extraction (probes/2026-02-19) | confirms | Yes — same pattern of cmd → pkg extraction | None |

---

## Findings

### Finding 1: Seven Distinct Responsibility Clusters in runComplete()

**Evidence:** Line-by-line analysis of `runComplete()` (lines 367-1431, 1,063 lines) reveals these clusters:

| Cluster | Lines | Span | Coupling |
|---------|-------|------|----------|
| 1. Skip config validation | 368-386 | 18 | CLI flags only |
| 2. Identifier resolution (workspace/beads/cross-project) | 388-516 | 128 | shared.go, beads pkg |
| 3. Issue/untracked detection | 517-550 | 33 | verify pkg |
| 4. Verification gate execution + skip filtering | 551-876 | 325 | verify, checkpoint, events |
| 5. Liveness check + interactive prompt | 878-937 | 59 | state, term |
| 6. Discovered work + advisories + knowledge maintenance | 957-1142 | 185 | verify, beads |
| 7. Lifecycle transition + post-lifecycle + telemetry | 1143-1431 | 288 | agent, events, opencode |

**Source:** `cmd/orch/complete_cmd.go:367-1431`

**Significance:** These clusters have clear data boundaries between them. Clusters 1-3 produce identifiers/workspace context. Cluster 4 produces verification results. Clusters 5-6 are gates/advisories. Cluster 7 executes the transition. This maps naturally to a pipeline.

---

### Finding 2: 835 Lines of Helper Functions Have Zero CLI Coupling

**Evidence:** Functions at lines 1432-2267 have NO dependency on CLI flags (the `completeXxx` package-level vars). They accept all inputs as parameters:

| Function | Lines | Category |
|----------|-------|----------|
| `invalidateServeCache()` | 1436-1452 | Post-lifecycle |
| `addApprovalComment()` | 1456-1471 | Beads integration |
| `hasGoChangesInRecentCommits()` | 1475-1505 | Auto-rebuild |
| `detectNewCLICommands()` | 1513-1575 | Auto-rebuild |
| `trackDocDebt()` | 1579-1601 | Auto-rebuild |
| `detectNotableChangelogEntries()` | 1615-1673 | Changelog advisory |
| `isSkillRelevantChange()` | 1676-1700 | Changelog advisory |
| `runAutoRebuild()` | 1704-1710 | Auto-rebuild |
| `restartOrchServe()` | 1718-1789 | Auto-rebuild |
| `looksLikeWorkspaceName()` | 1791-1795 | Identifier resolution |
| `findWorkspaceByNameAcrossProjects()` | 1797-1832 | Identifier resolution |
| `exportOrchestratorTranscript()` | 1837-1913 | Pre-lifecycle |
| `archiveWorkspace()` | 1918-1953 | Lifecycle (dead code — now in LifecycleManager) |
| `collectCompletionTelemetry()` | 1958-1991 | Telemetry |
| `buildVerificationChecklist()` | 1998-2075 | UI output |
| `printVerificationChecklist()` | 2077-2087 | UI output |
| `formatChecklistStatus()` | 2089-2100 | UI output |
| `collectAccretionDelta()` | 2108-2245 | Telemetry |
| `countFileLines()` | 2249-2267 | Utility |

**Source:** `cmd/orch/complete_cmd.go:1432-2267`

**Significance:** These functions can be moved to separate files without ANY change to their signatures. The only question is which file they belong in.

---

### Finding 3: SkipConfig Belongs in pkg/verify

**Evidence:** `SkipConfig` (lines 203-331) is tightly coupled to `verify.Gate*` constants. Every method references gate names from pkg/verify. The struct is a thin mapping layer between CLI flags and verification gates. It has methods `hasAnySkip()`, `skippedGates()`, and `shouldSkipGate()` that purely operate on verify gate constants.

`SkipConfig` is currently consumed by:
1. `runComplete()` — passes to verification functions
2. `logSkipEvents()` — iterates skipped gates
3. `buildVerificationChecklist()` — checks if gates were skipped

Moving SkipConfig to pkg/verify would eliminate the circular coupling where complete_cmd.go reimplements gate-name mapping logic that belongs with the gate definitions.

**Source:** `cmd/orch/complete_cmd.go:203-331`, `pkg/verify/check.go` (gate constants)

**Significance:** This is the clearest extraction candidate — it's already a data type with methods, has no CLI flag dependency (getSkipConfig() bridges flags→struct), and logically belongs with the gates it references.

---

### Finding 4: archiveWorkspace() Is Dead Code

**Evidence:** `archiveWorkspace()` (lines 1918-1953) is defined but never called. The LifecycleManager (`pkg/agent/lifecycle_impl.go`) now handles workspace archival via its `WorkspaceManager.Archive()` effect. The function was superseded when lifecycle transitions were centralized.

**Source:** `cmd/orch/complete_cmd.go:1918-1953`, `pkg/agent/lifecycle_impl.go`

**Significance:** 36 lines of dead code that can be deleted outright. Small win but contributes to the overall reduction.

---

### Finding 5: Post-Lifecycle Operations Form a Cohesive Cluster

**Evidence:** Lines 1280-1431 contain post-lifecycle operations that run AFTER the LifecycleManager transition completes:

1. Remove triage:ready label (1283-1293)
2. Signal daemon verification (1290-1292)
3. Auto-rebuild if Go changes (1299-1338)
4. Detect notable changelog entries (1341-1368)
5. Collect telemetry (1370-1406)
6. Collect accretion delta (1408-1424)
7. Invalidate serve cache (1428)

These are all side-effects that happen AFTER the critical transition. They share a pattern: fire-and-forget with `fmt.Fprintf(os.Stderr, "Warning: ...")` on failure. They could be extracted to `complete_postlifecycle.go` as a single function that takes a context struct.

**Source:** `cmd/orch/complete_cmd.go:1280-1431`

**Significance:** ~150 lines that can move to their own file with a clean interface. The pattern is already established by `complete_hotspot.go` and `complete_model_impact.go`.

---

### Finding 6: Duplicate Skip-Filter Logic for Orchestrator vs Worker Verification

**Evidence:** Lines 639-678 (orchestrator verification) and lines 734-773 (worker verification) contain nearly identical skip-filter logic. Both:
1. Iterate `result.GatesFailed`
2. Check `skipConfig.shouldSkipGate(gate)`
3. Filter errors by matching gate names in error text
4. Call `logSkipEvents()`
5. Update result with filtered data

This is ~40 lines of duplicated code that could be a single function: `applySkipFilters(result *verify.VerificationResult, skipConfig SkipConfig, beadsID, agentName string)`.

**Source:** `cmd/orch/complete_cmd.go:639-678` and `734-773`

**Significance:** Deduplication opportunity that reduces lines AND eliminates a maintenance risk (changes to skip filtering need to be applied twice).

---

### Finding 7: Verification Checklist UI Is Self-Contained

**Evidence:** `verificationChecklistItem` type (line 1993), `buildVerificationChecklist()` (lines 1998-2075), `printVerificationChecklist()` (lines 2077-2087), and `formatChecklistStatus()` (lines 2089-2100) form a complete, self-contained UI rendering unit. They have no external callers outside complete_cmd.go and no dependency on CLI flags.

**Source:** `cmd/orch/complete_cmd.go:1993-2100`

**Significance:** 107 lines of presentation logic that can move to `complete_checklist.go` alongside the existing `complete_hotspot.go` pattern.

---

## Synthesis

**Key Insights:**

1. **Same-package splitting is the primary strategy** — Most of complete_cmd.go is CLI-level orchestration that belongs in `cmd/orch/`. Only SkipConfig truly belongs in a library package (pkg/verify). This matches the code-extraction-patterns guide: "Keep command + flags + init + run + types together" but split by responsibility within the same package.

2. **runComplete() needs pipeline decomposition, not monolithic extraction** — The prior pipeline refactoring (mentioned in MEMORY.md as "completed Feb 2026") was either reverted or never landed for complete_cmd.go. The function needs to be decomposed into typed phases with a thin orchestrator, matching the pattern described in MEMORY.md ("thin orchestrator function calls phase functions with typed I/O").

3. **Four extraction phases minimize risk** — By doing SkipConfig extraction first (independent), then parallel extractions of helpers, the risk of merge conflicts is minimized. Each phase is independently buildable and testable.

**Answer to Investigation Question:**

The extraction should use a 4-phase approach:

**Phase 1: SkipConfig → pkg/verify/skip.go** (~130 lines moved)
- Move SkipConfig type + methods to pkg/verify
- getSkipConfig() stays in complete_cmd.go (bridges CLI flags → type)
- validateSkipFlags() stays in complete_cmd.go (validates CLI flags)
- Create `applySkipFilters()` in pkg/verify to deduplicate orchestrator/worker skip logic

**Phase 2: Post-lifecycle helpers → cmd/orch/complete_postlifecycle.go** (~300 lines moved)
- Auto-rebuild cluster: hasGoChangesInRecentCommits, detectNewCLICommands, trackDocDebt, runAutoRebuild, restartOrchServe
- Telemetry cluster: collectCompletionTelemetry, collectAccretionDelta, countFileLines
- Cache invalidation: invalidateServeCache
- Delete dead code: archiveWorkspace()

**Phase 3: Checklist + changelog → cmd/orch/complete_checklist.go** (~180 lines moved)
- verificationChecklistItem type
- buildVerificationChecklist, printVerificationChecklist, formatChecklistStatus
- NotableChangelogEntry type, detectNotableChangelogEntries, isSkillRelevantChange

**Phase 4: runComplete() pipeline decomposition → cmd/orch/complete_pipeline.go** (~400 lines moved)
- Extract 4 phase functions from runComplete():
  1. `resolveCompletionTarget()` — identifier resolution, workspace lookup, beads ID resolution
  2. `executeVerificationGates()` — all verification + skip filtering
  3. `runCompletionAdvisories()` — discovered work, probes, architectural choices, knowledge maintenance, explain-back
  4. `executeLifecycleTransition()` — pre-lifecycle exports, LifecycleManager.Complete, post-lifecycle operations
- runComplete() becomes a ~100-line orchestrator calling these phases

**Expected result:**
- complete_cmd.go: ~700 lines (command definition + flags + init + thin runComplete orchestrator)
- complete_pipeline.go: ~400 lines (4 phase functions)
- complete_postlifecycle.go: ~300 lines (helpers)
- complete_checklist.go: ~180 lines (UI rendering + changelog)
- pkg/verify/skip.go: ~150 lines (SkipConfig + applySkipFilters)
- Delete: ~36 lines (dead archiveWorkspace)
- Dedup: ~40 lines (skip filter duplication)

Total: from 2,267 lines in one file → ~700 + 400 + 300 + 180 = ~1,580 lines across 4 files + 150 in pkg/verify

---

## Structured Uncertainty

**What's tested:**

- ✅ Every function in complete_cmd.go has been read and categorized by responsibility cluster
- ✅ SkipConfig methods only reference verify.Gate* constants (verified by reading lines 230-311)
- ✅ archiveWorkspace() has no callers (grep confirmed — only the definition exists)
- ✅ Post-lifecycle helpers have no CLI flag dependencies (all accept parameters)
- ✅ Skip-filter logic is duplicated between orchestrator and worker paths (lines 639-678 vs 734-773)

**What's untested:**

- ⚠️ Whether complete_pipeline.go (mentioned in MEMORY.md) was attempted and reverted, or was a different refactoring
- ⚠️ Whether the existing complete_test.go tests will need modification after extraction
- ⚠️ Whether moving SkipConfig to pkg/verify creates import cycles (unlikely — complete_cmd.go already imports verify)

**What would change this:**

- If complete_pipeline.go already exists with typed phases, the Phase 4 work changes to extending rather than creating
- If tests depend on SkipConfig being in main package, Phase 1 ordering changes
- If runComplete() has been simplified since last read, Phase 4 scope shrinks

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| 4-phase extraction of complete_cmd.go | architectural | Cross-package boundary (SkipConfig → pkg/verify) + affects verification pipeline used by all completions |
| Delete archiveWorkspace() dead code | implementation | Confirmed dead code, no external impact |
| Deduplicate skip-filter logic | implementation | Code quality within single function, no behavioral change |

### Recommended Approach ⭐

**4-Phase Incremental Extraction** — Split complete_cmd.go into 4 files (3 in cmd/orch/ + 1 in pkg/verify/) using the established code-extraction-patterns workflow.

**Why this approach:**
- Follows established extraction guide (shared utilities first → domain splits)
- Each phase is independently buildable (`go build ./cmd/orch/` after each)
- SkipConfig extraction to pkg/verify eliminates cross-package coupling anti-pattern
- Pipeline decomposition of runComplete() enables future phase-level testing

**Trade-offs accepted:**
- 4 implementation issues instead of 1 large one — more tracking overhead but safer execution
- complete_cmd.go still ~700 lines after extraction — could be smaller but further splitting would break the "command + run function" cohesion pattern
- Not creating pkg/completion/ — the logic is CLI-specific, not a reusable library

**Implementation sequence:**
1. **Phase 1 (SkipConfig → pkg/verify/skip.go)** — Independent, no inter-file dependencies. Creates the applySkipFilters() helper that Phase 4 will use.
2. **Phase 2 (post-lifecycle → complete_postlifecycle.go)** — Move helper functions. Can start after Phase 1 or in parallel.
3. **Phase 3 (checklist → complete_checklist.go)** — Move UI rendering. Can start after Phase 1 or in parallel.
4. **Phase 4 (pipeline → complete_pipeline.go)** — Decompose runComplete(). Depends on Phase 1 (uses SkipConfig from verify), benefits from Phase 2-3 (fewer helpers in the file).

### Alternative Approaches Considered

**Option B: Monolithic pkg/completion/ package**
- **Pros:** Single destination for all completion logic; clean import path
- **Cons:** Most code is CLI-specific (flag parsing, interactive prompts, tmux operations) — doesn't belong in a library package. Would require passing CLI state as parameters, creating large context structs for no reuse benefit.
- **When to use instead:** If completion logic needs to be called from multiple entry points (daemon, API, CLI). Currently it's CLI-only.

**Option C: Keep everything in cmd/orch/ with just file splits**
- **Pros:** Simplest — no cross-package changes
- **Cons:** SkipConfig stays coupled to verify gate constants via string matching instead of being co-located. Misses the architectural improvement.
- **When to use instead:** If pkg/verify is itself being extracted or redesigned, deferring SkipConfig move avoids churn.

**Rationale for recommendation:** Option A provides the best balance of code organization (SkipConfig belongs with gates), practical splitting (helpers to separate files), and risk management (4 small phases vs 1 large change). Option B over-engineers for a single-caller scenario. Option C misses the SkipConfig coupling fix.

---

### Implementation Details

**What to implement first:**
- Phase 1: SkipConfig extraction — smallest, most self-contained, creates foundation for dedup
- Delete archiveWorkspace() — immediate 36-line win, zero risk

**Things to watch out for:**
- ⚠️ Tab indentation in Go files — use `cat -vet` to verify whitespace before Edit tool operations (per CLAUDE.md)
- ⚠️ complete_test.go may reference SkipConfig directly — tests need updating in Phase 1
- ⚠️ Parallel agent conflicts — check `git status` before starting each phase for unexpected changes
- ⚠️ The `getSkipConfig()` function bridges CLI flags to SkipConfig — it MUST stay in complete_cmd.go (package main has the flag vars)

**Areas needing further investigation:**
- Whether MEMORY.md's "Pipeline Refactoring Pattern" refers to complete_cmd.go or a different file
- Whether complete_test.go coverage is sufficient to validate the extraction

**Success criteria:**
- ✅ complete_cmd.go under 800 lines
- ✅ `go build ./cmd/orch/` passes after each phase
- ✅ `go test ./cmd/orch/...` passes after each phase (no test regressions)
- ✅ `go vet ./cmd/orch/` passes after each phase
- ✅ SkipConfig importable as `verify.SkipConfig` from any consumer
- ✅ No duplicate function definitions across files

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` — Primary extraction target (2,267 lines, full read)
- `cmd/orch/complete_architect.go` — Existing extraction (164 lines)
- `cmd/orch/complete_cleanup.go` — Existing extraction (43 lines)
- `cmd/orch/complete_hotspot.go` — Existing extraction (184 lines)
- `cmd/orch/complete_model_impact.go` — Existing extraction (238 lines)
- `cmd/orch/complete_test.go` — Test patterns (843 lines)
- `cmd/orch/shared.go` — Shared utilities (570 lines)
- `pkg/verify/check.go` — Gate constants and VerificationResult type
- `pkg/orch/completion.go` — Explain-back gate implementation
- `pkg/agent/lifecycle.go` — LifecycleManager interface
- `pkg/checkpoint/checkpoint.go` — Verification checkpoint storage

**Commands Run:**
```bash
wc -l cmd/orch/complete*.go
wc -l pkg/verify/*.go
grep "^func \|^type \|^var " cmd/orch/complete_cmd.go
grep -rl "findWorkspaceByName\|SkipConfig\|archiveWorkspace" cmd/orch/
kb context "extraction completion architecture"
```

**Related Artifacts:**
- **Guide:** `.kb/guides/code-extraction-patterns.md` — Extraction workflow reference
- **Guide:** `.kb/guides/completion.md` — Completion system architecture
- **Guide:** `.kb/guides/completion-gates.md` — Gate reference
- **Decision:** Prior decision on `No Local Agent State` constraint (CLAUDE.md)

---

## Investigation History

**2026-02-28 12:45:** Investigation started
- Initial question: How to decompose complete_cmd.go (2,267 lines) to under 1,500 lines
- Context: #1 hotspot in codebase — the file enforcing code quality is itself the least maintainable

**2026-02-28 13:15:** All 7 responsibility clusters identified, helper coupling analyzed
- Found archiveWorkspace() is dead code
- Found skip-filter duplication between orchestrator/worker paths
- Confirmed SkipConfig belongs in pkg/verify

**2026-02-28 13:30:** Investigation completed
- Status: Complete
- Key outcome: 4-phase extraction plan with dependency chain, expected reduction from 2,267 → ~700 lines in primary file
