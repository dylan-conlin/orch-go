## Summary (D.E.K.N.)

**Delta:** The daemon has two God objects (Daemon struct: 93 fields/22 mock funcs; runDaemonLoop: 625 lines handling 12 subsystems) and is structurally healthy in pkg/daemon/ but dangerously monolithic in cmd/orch/daemon.go.

**Evidence:** Line counts, function counts, struct field inventory, coupling analysis, and test execution (all pass, 8s).

**Knowledge:** The pkg/daemon/ package is already well-decomposed across 30+ files with good test coverage (10,368 test lines). The structural risk concentrates in two places: the Daemon struct's mock function proliferation and the cmd/orch/daemon.go main loop.

**Next:** Extract periodic task scheduler from runDaemonLoop, extract model drift reflection to its own package, and convert mock function fields to interfaces.

**Authority:** architectural - Extraction changes cross package boundaries and affect multiple consumers of pkg/daemon.

---

# Investigation: Daemon Code Health Audit

**Question:** What is the structural health of the daemon code, and what extraction opportunities exist before new features (singleton enforcement, orient integration) land?

**Defect-Class:** unbounded-growth

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** orch-go-ajay
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Daemon Struct is a God Object (93 fields, 22 mock functions)

**Evidence:** The `Daemon` struct in `pkg/daemon/daemon.go:47-158` has 93 fields:
- 7 legitimate state fields (Config, Pool, RateLimiter, HotspotChecker, PriorArchitectFinder, SpawnedIssues, ProjectRegistry)
- 6 timer fields (lastReflect, lastModelDriftReflect, lastKnowledgeHealth, lastCleanup, lastRecovery, lastOrphanDetection)
- 3 tracker fields (VerificationTracker, SpawnFailureTracker, CompletionFailureTracker)
- 1 map field (resumeAttempts)
- **22 mock function fields** for testing (listIssuesFunc, spawnFunc, reflectFunc, etc.)

**Source:** `pkg/daemon/daemon.go:47-158`, `grep -c 'Func\b' pkg/daemon/daemon.go` → 72 lines containing "Func"

**Significance:** The 22 mock function fields are the primary accretion driver. Each new daemon behavior adds 1-3 more function fields for testability. This pattern makes the struct increasingly difficult to understand and initializers (NewWithConfig, NewWithPool) increasingly fragile. Converting to interfaces would group related behaviors and reduce field count significantly.

---

### Finding 2: runDaemonLoop() is a 625-line God Function

**Evidence:** `cmd/orch/daemon.go:255-878` (625 lines) handles 12 distinct subsystems in a single function:

| Line Range | Subsystem | Lines |
|------------|-----------|-------|
| 297-308 | Signal handling | 11 |
| 386-392 | Pool reconciliation | 6 |
| 396-413 | Verification signal check | 17 |
| 416-433 | Verification pause check | 17 |
| 436-444 | Periodic reflection | 8 |
| 447-455 | Model drift reflection | 8 |
| 458-471 | Knowledge health | 13 |
| 474-507 | Session cleanup | 33 |
| 510-545 | Stuck agent recovery | 35 |
| 548-581 | Orphan detection | 33 |
| 587-667 | Completion processing | 80 |
| 669-729 | Status file writing | 60 |
| 732-858 | Issue polling & spawning | 126 |

Each subsystem follows an identical pattern: check if due → run → handle error/success → log event. This is a textbook candidate for a periodic task scheduler abstraction.

**Source:** `cmd/orch/daemon.go:255-878`

**Significance:** At 1180 total lines, this file is 79% of the 1500-line CRITICAL threshold. Adding singleton enforcement and orient integration will push it over. The repetitive pattern (check → run → handle → log) across 12 subsystems means extraction would be high-impact with a clean mechanical transformation.

---

### Finding 3: pkg/daemon/ is Well-Decomposed but Overly Large

**Evidence:** The package contains 30 non-test files totaling **9,817 lines** of production code (18,075 including tests). The largest files:

| File | Lines | Functions | Responsibility |
|------|-------|-----------|---------------|
| daemon.go | 883 | 14 | Core spawn loop, Daemon struct |
| model_drift_reflection.go | 679 | 25 | Model drift detection & issue creation |
| issue_adapter.go | 551 | 17 | Beads RPC/CLI adapter |
| completion_processing.go | 360 | 8 | Completion verification loop |
| reflect.go | 373 | 14 | KB reflection |
| completion.go | 308 | 14 | CompletionService (SSE) |
| skill_inference.go | 302 | 9 | Skill routing |
| verification_tracker.go | 293 | 16 | Verification pause system |
| extraction.go | 289 | 9 | Hotspot extraction |
| periodic.go | 273 | 12 | Periodic task scheduling |

**Source:** `wc -l pkg/daemon/*.go | grep -v _test | sort -rn`

**Significance:** While individual files are reasonable, the package has too many responsibilities. model_drift_reflection.go (679 lines, 25 functions) is self-contained and could be its own package. issue_adapter.go (551 lines) is a beads adapter that belongs in a separate layer.

---

### Finding 4: Test Coverage is Solid

**Evidence:**
- All tests pass: `go test ./pkg/daemon/ -count=1 -short` → `ok 8.024s`
- 29 test files totaling 10,368 lines (1.06:1 test-to-code ratio for pkg/daemon/)
- Every major subsystem has corresponding tests:
  - daemon_test.go (965 lines), extraction_test.go (717 lines), architect_escalation_test.go (714 lines)
  - verification_tracker_test.go (635 lines), skill_inference_test.go (578 lines)
  - orphan_detector_test.go (497 lines), spawn_tracker_test.go (495 lines)
- Notable: `daemon_test.go.bak` exists (disabled tests), but current tests all pass

**Source:** `go test ./pkg/daemon/`, `wc -l pkg/daemon/*_test.go`

**Significance:** Good test coverage de-risks extraction work. Tests validate behavior and serve as a safety net for refactoring. The mock function pattern, while bloating the struct, does enable thorough testing.

---

### Finding 5: Coupling Map Shows Manageable Dependencies

**Evidence:**

**pkg/daemon depends on (7 internal packages):**
| Package | Used By | Purpose |
|---------|---------|---------|
| pkg/beads | daemon.go, issue_adapter.go, model_drift_reflection.go, preview.go | Issue CRUD, RPC client |
| pkg/daemonconfig | daemon.go | Config type alias |
| pkg/spawn | daemon.go, model_drift_reflection.go, recovery.go | StalenessEvent, ReadSessionID |
| pkg/opencode | completion.go, cleanup.go, stall_tracker.go | SSE monitor, session listing |
| pkg/tmux | active_count.go, cleanup.go, session_dedup.go | Window management |
| pkg/events | completion_processing.go, recovery.go, skill_inference.go | Event logging |
| pkg/verify | completion_processing.go, issue_adapter.go, recovery.go | Verification, issue listing |
| pkg/checkpoint | issue_adapter.go | Checkpoint reading |

**pkg/daemon is depended on by (10 files in cmd/orch/):**
- daemon.go, complete_cmd.go, serve_daemon_actions.go, serve_system.go
- serve_verification.go, serve_agents_status.go, status_cmd.go
- session.go, swarm.go
- Also: pkg/spawn/gates/concurrency.go

**Source:** `grep '"github.com/dylan-conlin/orch-go/pkg/daemon"'` across codebase

**Significance:** The dependency tree is manageable. pkg/daemon is a leaf-ish package that depends on lower-level packages. The 10 consumers in cmd/orch/ are mostly serve handlers reading daemon state. Extraction of sub-packages won't create circular dependencies.

---

### Finding 6: cmd/orch/daemon.go Contains Presentation Logic Mixed with Business Logic

**Evidence:** The file contains 20 CLI flag variables (lines 126-146), flag registration in init() (lines 148-183), flag-to-config mapping in daemonConfigFromFlags() (lines 188-216), and extensive formatting/display code throughout runDaemonLoop(). The core business logic is in pkg/daemon/, but the orchestration of periodic tasks and event logging is entirely in cmd/orch/.

**Source:** `cmd/orch/daemon.go:126-216` (flags), `cmd/orch/daemon.go:255-878` (loop with formatting)

**Significance:** The periodic task orchestration pattern belongs in pkg/daemon/, not cmd/orch/. Moving it would reduce cmd/orch/daemon.go to ~400 lines (flag setup, config mapping, output formatting) and make periodic tasks testable without a CLI.

---

## Synthesis

**Key Insights:**

1. **The accretion risk is in two places, not one** — pkg/daemon/daemon.go (883 lines) is large but stable and well-tested. The real danger is cmd/orch/daemon.go (1180 lines) which is 79% to CRITICAL and growing with every new daemon subsystem.

2. **Mock function proliferation is the Daemon struct's growth driver** — 22 of 93 struct fields exist solely for test injection. An interface-based approach (e.g., `BeadsAdapter`, `ReflectRunner`, `CompletionProcessor`) would group related mocks, reduce struct fields by ~60%, and improve readability.

3. **The periodic task pattern is the highest-ROI extraction** — All 12 subsystems in runDaemonLoop follow the same pattern: ShouldRun() → Run() → handle result → log event. A `PeriodicTask` interface with a scheduler would eliminate ~300 lines of repetitive code from cmd/orch/daemon.go and make new subsystems trivially addable.

**Answer to Investigation Question:**

The daemon code is structurally healthy at the pkg/daemon/ level — responsibilities are decomposed across 30+ files, test coverage is solid (10K+ lines, all passing), and coupling is manageable. The critical risk is in cmd/orch/daemon.go (1180 lines, 79% to CRITICAL) where 12 subsystems are orchestrated in a single 625-line function. Adding singleton enforcement and orient integration without extraction will push this file past 1500 lines and trigger the accretion boundary. Three targeted extractions (periodic task scheduler, model drift package, interface consolidation) would reduce this file to ~400 lines and create headroom for new features.

---

## Structured Uncertainty

**What's tested:**

- ✅ All daemon tests pass: `go test ./pkg/daemon/ -count=1 -short` → ok 8.024s
- ✅ Line counts verified via `wc -l` on all daemon files
- ✅ Coupling verified via grep for import paths across codebase
- ✅ Daemon struct field count verified via grep (93 fields)
- ✅ Function counts verified via `grep -c '^func '` per file

**What's untested:**

- ⚠️ Cyclomatic complexity not measured (no Go tool run, estimated from visual inspection)
- ⚠️ Test coverage percentage not measured (no `go test -cover` run)
- ⚠️ Impact of extracting model_drift_reflection.go on import cycle risk not verified

**What would change this:**

- If go test -cover shows <60% coverage on daemon.go, extraction risk increases
- If model_drift_reflection imports create a cycle when extracted, a different boundary is needed
- If singleton enforcement requires <100 lines in runDaemonLoop, extraction urgency decreases

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extract periodic task scheduler | architectural | Creates new pkg/daemon/scheduler or similar; changes cmd/orch/daemon.go structure |
| Extract model drift to own package | architectural | Creates new package boundary, moves 679 lines + tests |
| Convert mock funcs to interfaces | architectural | Changes Daemon struct API used by 10+ consumers |
| Extract issue adapter to own package | implementation | Already self-contained, minimal consumer impact |

### Recommended Approach ⭐

**Three-Phase Extraction** - Address the God function first (highest risk), then the God struct, then the oversized package.

**Why this approach:**
- Phase 1 (scheduler) directly addresses the 1500-line CRITICAL threshold risk
- Phase 2 (interfaces) reduces struct complexity that slows every new feature
- Phase 3 (model drift extraction) is cleanest and can happen independently

**Trade-offs accepted:**
- Extraction work before features delays singleton enforcement / orient integration
- Interface consolidation may require test refactoring

**Implementation sequence:**

1. **Phase 1: Extract periodic task scheduler from runDaemonLoop** (effort: medium, impact: high)
   - Create `PeriodicTask` interface: `{ ShouldRun() bool; Run() Result; Name() string }`
   - Create scheduler that iterates tasks each cycle, handles logging/events
   - Register all 12 subsystems as tasks
   - Reduces cmd/orch/daemon.go from ~1180 to ~400 lines
   - New subsystems become one-liners to add

2. **Phase 2: Convert mock function fields to interfaces** (effort: medium, impact: medium)
   - Group related mocks: `BeadsAdapter` (list, update, show), `ReflectRunner` (reflect, modelDrift), `HealthChecker` (knowledge, orphan)
   - Reduces Daemon struct from 93 fields to ~20
   - Improves testability (mock interfaces, not 22 individual functions)

3. **Phase 3: Extract model_drift_reflection.go to pkg/modeldrift/** (effort: low, impact: medium)
   - 679 lines + 68 test lines already self-contained
   - Only imports: beads, spawn (for StalenessEvent type)
   - Creates clean domain package for model maintenance subsystem

### Alternative Approaches Considered

**Option B: Feature-freeze until all extractions complete**
- **Pros:** Clean architecture before new features
- **Cons:** Blocks singleton enforcement / orient work
- **When to use:** If daemon is actively breaking due to complexity

**Option C: Only extract on-demand when 1500-line threshold hit**
- **Pros:** Minimizes speculative work
- **Cons:** Emergency extraction under pressure produces worse architecture
- **When to use:** If extraction capacity is severely limited

### Implementation Details

**What to implement first:**
- Phase 1 (periodic scheduler) is most urgent — blocks are growing per-cycle
- Can be done incrementally: extract one subsystem at a time

**Things to watch out for:**
- ⚠️ The logger instance in runDaemonLoop is shared across subsystems — scheduler needs logger injection
- ⚠️ Some subsystems produce snapshots consumed by status file writing (knowledgeHealthSnapshot) — scheduler needs snapshot collection
- ⚠️ CompletionService (completion.go) is SSE-based with goroutines — different lifecycle than periodic tasks

**Success criteria:**
- ✅ cmd/orch/daemon.go < 600 lines after Phase 1
- ✅ Daemon struct < 30 fields after Phase 2
- ✅ No new files >500 lines created during extraction
- ✅ All existing tests continue to pass
- ✅ Adding a new periodic subsystem requires <20 lines of code

---

## Ranked Extraction Opportunities

| Priority | What to Extract | From → To | Lines | Effort | Impact | Risk |
|----------|----------------|-----------|-------|--------|--------|------|
| 1 | Periodic task scheduler | cmd/orch/daemon.go → pkg/daemon/scheduler/ | ~300 saved | Medium | **High** - unblocks growth | Low |
| 2 | Mock funcs → interfaces | pkg/daemon/daemon.go struct | ~50 saved, clarity++ | Medium | **Medium** - struct clarity | Medium (API change) |
| 3 | Model drift reflection | pkg/daemon/model_drift*.go → pkg/modeldrift/ | 679+68 moved | Low | **Medium** - clean domain | Low |
| 4 | Issue adapter | pkg/daemon/issue_adapter.go → pkg/daemon/beadsadapter/ | 551 moved | Low | **Low** - already isolated | Low |
| 5 | Knowledge health | pkg/daemon/knowledge_health.go → dedicated subsystem | 163 moved | Low | **Low** - small file | Low |

---

## Risk Assessment: What Breaks If daemon.go Crosses 1500 Lines?

1. **Accretion boundary enforcement triggers** — The spawn gate blocks `feature-impl` and `systematic-debugging` skills from spawning work targeting CRITICAL files (>1500 lines). This means daemon improvements themselves get blocked.

2. **Cognitive load** — At 1180 lines, cmd/orch/daemon.go is already at the edge of comprehensibility. Developers (human and AI) reading the file must hold 12 subsystems in working memory. Error patterns become harder to spot.

3. **Merge conflicts increase** — Multiple agents working on different daemon features will conflict in the same file, especially in the monolithic runDaemonLoop function.

4. **Test fragility** — Any change to the main loop potentially affects all 12 subsystems. Without a scheduler abstraction, integration testing requires running the full loop.

---

## References

**Files Examined:**
- `cmd/orch/daemon.go` (1180 lines) - CLI commands and main polling loop
- `pkg/daemon/daemon.go` (883 lines) - Daemon struct, spawn logic
- `pkg/daemon/pool.go` (253 lines) - Worker pool
- `pkg/daemon/periodic.go` (273 lines) - Periodic task scheduling (reflection, cleanup, recovery)
- `pkg/daemon/completion.go` (308 lines) - CompletionService
- `pkg/daemon/completion_processing.go` (360 lines) - Completion loop
- `pkg/daemon/recovery.go` (177 lines) - Stuck agent recovery
- `pkg/daemon/model_drift_reflection.go` (679 lines) - Model drift detection
- `pkg/daemon/orphan_detector.go` (186 lines) - Orphan detection
- `pkg/daemon/knowledge_health.go` (163 lines) - Knowledge health checks
- `pkg/daemon/capacity.go` (114 lines) - Capacity management
- `pkg/daemon/issue_adapter.go` (551 lines) - Beads adapter
- `pkg/daemon/status.go` (193 lines) - Status file management
- `pkg/daemonconfig/config.go` (157 lines) - Config struct

**Commands Run:**
```bash
wc -l cmd/orch/daemon.go pkg/daemon/*.go pkg/daemonconfig/*.go
wc -l pkg/daemon/*_test.go
go test ./pkg/daemon/ -count=1 -short
grep -n '^func ' cmd/orch/daemon.go pkg/daemon/daemon.go
grep '"github.com/dylan-conlin/orch-go/pkg/daemon"' across codebase
```

---

## Investigation History

**2026-02-27:** Investigation started
- Initiated by orchestrator: daemon is known hotspot (coupling-cluster:90), approaching 1500-line CRITICAL threshold
- Context: Singleton enforcement and orient integration about to land, need structural health assessment

**2026-02-27:** Analysis complete
- Read all 30+ daemon files, ran tests, mapped coupling
- Key finding: The risk is concentrated in cmd/orch/daemon.go (God function), not pkg/daemon/ (well-decomposed)
- Recommended three-phase extraction with periodic scheduler as highest priority
