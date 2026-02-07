## Summary (D.E.K.N.)

**Delta:** There are no remaining production Go functions over 300 lines, but architecture risk has shifted to several 1k+ files, repeated HTTP handler boilerplate, a small set of now-unreferenced helpers from recent extractions, and major coverage gaps in spawn/serve/daemon critical paths.

**Evidence:** I verified function sizes with an AST-based parser, measured file lengths, ran `go test ./... -cover`, drilled into `cmd/orch` function-level coverage with `go tool cover -func`, and checked candidate dead helpers with cross-repo symbol search.

**Knowledge:** The 21260 extraction work reduced god functions effectively, but did not yet consolidate serve-layer duplication or add regression coverage around the extracted execution paths.

**Next:** Create focused follow-up issues for (1) serve JSON/HTTP helper extraction, (2) dead helper cleanup from extraction boundaries, and (3) coverage hardening for spawn/serve_system/daemon loops.

**Authority:** architectural - Recommendations cross multiple command modules and testing boundaries and need orchestrator-level sequencing.

---

# Investigation: Architecture Code Health Audit Focusing

**Question:** What NEW architecture/code-health risks remain after epic 21260, specifically across >300-line god functions, >1000-line files, duplicated extraction opportunities, dead code from recent extractions, and critical-path test coverage gaps?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/extract-patterns.md` | extends | yes | none |
| Epic `orch-go-21260` outcome (serve_agents/daemon/spawn_cmd/stats_cmd extraction) | confirms | yes | none |

---

## Findings

### Finding 1: No remaining production god functions over 300 lines

**Evidence:** AST-based function span analysis across all `.go` files found no non-test function above 300 lines; longest production function observed is `handleAttention` at 260 lines.

**Source:** `go run /tmp/longfuncs.go` (custom `go/parser` scan), `cmd/orch/serve_attention.go:111`.

**Significance:** The explicit epic target to break up god functions appears achieved; future architecture work should prioritize file-level modularity and duplication over giant single functions.

---

### Finding 2: Eight production files still exceed 1000 lines

**Evidence:** Non-test files over 1000 lines:
- `cmd/orch/serve_system.go` (1446)
- `pkg/beads/client.go` (1313)
- `cmd/orch/review.go` (1225)
- `pkg/spawn/context.go` (1196)
- `pkg/tmux/tmux.go` (1159)
- `cmd/orch/stats_cmd.go` (1128)
- `cmd/orch/kb.go` (1097)
- `cmd/orch/complete_pipeline.go` (1017)

**Source:** Python line-count scan over non-test `.go` files.

**Significance:** The main maintainability bottleneck has shifted from oversized functions to oversized modules, especially in command/server orchestration surfaces where behavior is still centralized.

---

### Finding 3: High-ROI duplication remains in HTTP serve handlers

**Evidence:**
- `w.Header().Set("Content-Type", "application/json")` appears 100 times across `cmd/orch/serve*.go`.
- `if r.Method != http.MethodGet` guard appears 9 times in `cmd/orch/serve_system.go`.
- `handleConfig*` and `handleDaemonConfig*` flows duplicate load/update/save/respond lifecycle with only type/validation differences.

**Source:** `grep` scans in `cmd/orch/serve*.go`, inspection of `cmd/orch/serve_system.go:727`, `cmd/orch/serve_system.go:831`.

**Significance:** A small response helper layer (`jsonOK`, `jsonErr`, method router helper, config-update helper) would remove repeated error/encoding boilerplate and reduce bug-surface for future endpoint additions.

---

### Finding 4: Dead helpers exist near extraction boundaries

**Evidence:** Symbol search shows definition-only helpers (no call sites) in command code:
- `cmd/orch/serve_agents_status.go:74` `getProjectAPIPort`
- `cmd/orch/spawn_usage.go:211` `checkAndAutoSwitchAccount`
- `cmd/orch/spawn_validation.go:35` `runPreSpawnKBCheck`
- `cmd/orch/spawn_validation.go:654` `logDecisionOverride`

**Source:** `grep` across `cmd/orch` for each symbol; only declaration hits (or comment references) found.

**Significance:** These appear to be orphaned by extraction/rewiring work; keeping them increases cognitive load and risks stale behavior assumptions.

---

### Finding 5: Critical-path coverage is still low where architecture risk is highest

**Evidence:**
- Package coverage: `cmd/orch` 23.4%, `pkg/daemon` 54.2%, `pkg/tmux` 29.4%, `pkg/beads` 40.6%.
- In extracted paths, key runtime flows are 0%: `runSpawnWithSkillInternal`, `runSpawnHeadlessWithClient`, `runSpawnTmuxWithClient` (`cmd/orch/spawn_cmd.go`), `runSpawnPipeline` phases in `cmd/orch/spawn_pipeline.go`, and most `serve_system.go` handlers.
- Daemon loop/refactor files remain largely uncovered (`cmd/orch/daemon_loop.go` and multiple daemon command helpers at 0%).

**Source:** `go test ./... -cover`, `go test ./cmd/orch -coverprofile=/tmp/cmd_orch.cover`, `go tool cover -func=/tmp/cmd_orch.cover`.

**Significance:** Architecture refactors landed, but regression net is thin around orchestration-critical execution paths; this is the highest practical reliability risk.

---

## Synthesis

**Key Insights:**

1. **Extraction succeeded on function bloat** - The >300-line function target is effectively closed in production code.
2. **Complexity migrated rather than disappeared** - Module-level concentration and endpoint duplication now dominate maintenance cost.
3. **Coverage debt is now the gating risk** - The most behavior-dense paths (spawn/serve_system/daemon) remain minimally tested post-extraction.

**Answer to Investigation Question:**

The biggest NEW architecture/code-health opportunities are not additional god-function splits; they are (a) reducing 1k+ file concentration in command/server modules, (b) extracting repeated HTTP response + method-handling patterns, (c) removing orphan helpers left after extraction rewires, and (d) adding targeted coverage in spawn/serve_system/daemon critical paths. This conclusion is directly supported by AST-based function sizing, file-size baselines, duplicate-pattern counts, symbol reference checks, and package/function-level coverage output.

---

## Structured Uncertainty

**What's tested:**

- ✅ No production function >300 lines (verified via `go/parser` function span scan).
- ✅ 8 non-test `.go` files exceed 1000 lines (verified via repository line-count scan).
- ✅ Serve-layer JSON/method boilerplate duplication is high (verified via `grep` counts and file inspection).
- ✅ Multiple extraction-adjacent helpers are definition-only in `cmd/orch` (verified via symbol search).
- ✅ Critical command/server paths show low or zero coverage (verified via `go test -cover` + `go tool cover -func`).

**What's untested:**

- ⚠️ Whether any definition-only helpers are intentionally retained for imminent feature flags (not confirmed with owners).
- ⚠️ Exact extraction plan/effort for each 1k+ file (not decomposed into concrete PR slices).
- ⚠️ Runtime performance impact of current duplication (no benchmark/profiling performed).

**What would change this:**

- If hidden reflective/dynamic call paths reference the flagged helpers, dead-code findings would need revision.
- If near-term roadmap requires keeping certain oversized modules stable, extraction priority may shift.
- If broader integration coverage exists outside `go test ./cmd/orch`, risk ranking of low-covered functions could drop.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extract shared serve JSON/method helpers and apply to `serve_system` + sibling handlers | architectural | Cross-file API style change across command server boundary |
| Remove or wire unreferenced extraction-adjacent helpers | implementation | Localized cleanup with clear compiler/test validation |
| Add coverage-first tests for spawn pipeline, serve system handlers, daemon loop execution paths | architectural | Test strategy spans multiple components and mocking boundaries |

### Recommended Approach ⭐

**Coverage-First Hardening + Incremental Serve Refactor** - Stabilize behavior with focused tests on critical paths, then extract common HTTP handler primitives and clean dead helpers.

**Why this approach:**
- Reduces regression risk before touching broad handler abstractions.
- Targets the highest-risk low-coverage files identified in this audit.
- Converts repeated patterns into reusable primitives with measurable line-count reduction.

**Trade-offs accepted:**
- Slightly slower short-term cleanup because tests come before broad refactor.
- Some >1000-line files remain temporarily until guarded by stronger tests.

**Implementation sequence:**
1. Add tests around `spawn_pipeline` and `serve_system` currently-0% paths.
2. Introduce `serve` response helpers and method dispatch helpers; migrate one file first (`serve_system.go`).
3. Remove confirmed-unreferenced helpers (`getProjectAPIPort`, `checkAndAutoSwitchAccount`, `runPreSpawnKBCheck`, `logDecisionOverride`) or wire explicitly if truly intended.

### Alternative Approaches Considered

**Option B: Immediate large-file extraction first**
- **Pros:** Faster file-size reduction.
- **Cons:** Higher break risk due to current low coverage in affected paths.
- **When to use instead:** If immediate parallelization of ownership by module is more urgent than short-term stability.

**Option C: Dead-code-only cleanup sprint**
- **Pros:** Fast wins, low merge friction.
- **Cons:** Leaves major duplication and coverage risk untouched.
- **When to use instead:** If the team needs a very short maintenance window before larger initiatives.

**Rationale for recommendation:** Option A best balances reliability and maintainability by first strengthening safety nets, then reducing duplication and module bloat.

---

### Implementation Details

**What to implement first:**
- Add function-level tests for `cmd/orch/spawn_cmd.go` and `cmd/orch/spawn_pipeline.go` 0%-covered execution branches.
- Add handler tests for `cmd/orch/serve_system.go` method/validation/error branches.
- Introduce a shared JSON responder helper used by at least one high-duplication serve file.

**Things to watch out for:**
- ⚠️ Handler refactors can subtly change status codes or JSON error shape.
- ⚠️ Spawn path tests require careful mocking of account/opencode/beads behavior to avoid brittle integration coupling.
- ⚠️ Dead-helper deletion may break undocumented workflows if external scripts rely on side effects.

**Areas needing further investigation:**
- Exact decomposition strategy for `pkg/beads/client.go` and `pkg/tmux/tmux.go`.
- Whether `serve_system.go` should split by domain (focus/config/daemon/files) or by transport concern.
- Existing expectations for JSON error payload consistency across dashboard endpoints.

**Success criteria:**
- ✅ `cmd/orch` package coverage materially increases in `spawn*` + `serve_system` files.
- ✅ Definition-only orphan helpers are removed or gain real call paths with tests.
- ✅ Repeated JSON/method boilerplate count in `serve*.go` decreases from current baseline.

---

## References

**Files Examined:**
- `cmd/orch/serve_system.go` - duplicated handler lifecycle patterns and module concentration
- `cmd/orch/serve_agents_status.go` - extraction-adjacent dead helper candidate
- `cmd/orch/spawn_usage.go` - unreferenced auto-switch helper candidate
- `cmd/orch/spawn_validation.go` - unreferenced wrapper/log helper candidates
- `cmd/orch/spawn_cmd.go` - critical execution path coverage gaps
- `cmd/orch/spawn_pipeline.go` - critical pipeline coverage gaps

**Commands Run:**
```bash
# Verify project context
pwd

# Create investigation
kb create investigation architecture-code-health-audit-focusing

# Find >1000-line files
python3 -c "..."

# Measure function spans via AST parser
go run /tmp/longfuncs.go

# Full test + coverage baseline
go test ./... -cover

# Function-level coverage in cmd/orch
go test ./cmd/orch -coverprofile=/tmp/cmd_orch.cover
go tool cover -func=/tmp/cmd_orch.cover

# Dead helper symbol checks
go run golang.org/x/tools/cmd/deadcode@latest ./cmd/orch
rg <symbol> cmd/orch
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Model:** `.kb/models/extract-patterns.md` - prior extraction approach baseline
- **Workspace:** `.orch/workspace/og-audit-architecture-code-health-07feb-6692/` - spawn workspace for this audit

---

## Investigation History

**[2026-02-07 07:45]:** Investigation started
- Initial question: Identify post-21260 architecture/code-health opportunities across bloat, duplication, dead code, and tests
- Context: Orchestrator requested architecture-focused code health audit

**[2026-02-07 08:03]:** Evidence collection completed
- Ran size/function/coverage/deadcode scans and validated candidate dead helpers via symbol search

**[2026-02-07 08:10]:** Investigation completed
- Status: Complete
- Key outcome: God functions are no longer the main issue; highest ROI now is coverage hardening, serve-layer deduplication, and extraction-adjacent dead helper cleanup
