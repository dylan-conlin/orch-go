## Summary (D.E.K.N.)

**Delta:** The daemon's 37+ files decompose cleanly into compliance (18 files), coordination (8 files), and infrastructure (8 files), with only 3 files exhibiting genuine entanglement — and the primary entanglement site is `OnceExcluding()` in daemon.go, which interleaves compliance gates with coordination routing in a single 150-line method.

**Evidence:** Full audit of every exported function in pkg/daemon/ against compliance/coordination classification criteria. The 5-layer spawn gate pipeline (spawn_gate.go) is already pure compliance. Skill inference (skill_inference.go) is already pure coordination. The entanglement is concentrated in daemon.go:OnceExcluding (extraction→escalation→spawn pipeline), issue_selection.go:NextIssueExcluding (filtering+sorting+boosting), and completion_processing.go:ProcessCompletion (verification+routing+auto-complete).

**Knowledge:** The entangled functions share state through the Daemon struct — specifically SpawnedIssues, HotspotChecker, VerificationTracker, and AutoCompleter. The minimal interface between compliance and coordination is: compliance needs (skill, issue) to validate; coordination needs (allowed: bool) from compliance. This is already partially realized in the SpawnPipeline pattern.

**Next:** Recommend architect follow-up to design the split implementation: extract a `ComplianceGateway` interface and a `CoordinationRouter` interface from the Daemon struct, with `OnceExcluding` becoming a thin orchestrator that calls both in sequence.

**Authority:** architectural — Cross-component restructuring that changes daemon internal boundaries; requires synthesis of spawn/completion/selection subsystems.

---

# Investigation: Compliance/Coordination Boundary Map for Daemon Architecture

**Question:** Where exactly are compliance (prevents bad outcomes) and coordination (improves outcomes) responsibilities entangled in the daemon, and what is the minimal interface to separate them?

**Started:** 2026-03-13
**Updated:** 2026-03-13
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/threads/2026-03-12-compliance-coordination-bifurcation-as-models.md | extends | Yes — daemon code confirms the entanglement described | None |
| spawn_gate.go refactor (SpawnPipeline extraction) | confirms | Yes — this was the first compliance extraction and it's clean | None |

---

## Finding 1: File-Level Classification (37 non-test files)

**Evidence:** Every exported function in pkg/daemon/ was audited and classified.

### COMPLIANCE (18 files) — Prevents bad outcomes

| File | Functions | What it prevents |
|------|-----------|-----------------|
| spawn_gate.go | SpawnPipeline, 5 gate types, 1 advisory | Duplicate spawns (6-layer dedup) |
| spawn_tracker.go | SpawnedIssueTracker (8 methods) | Race-condition duplicates during status update lag |
| session_dedup.go | HasExistingSession*, HasExistingTmuxWindow* | Duplicate spawns against running sessions |
| completion_dedup.go | CompletionDedupTracker (3 methods) | Re-processing same Phase: Complete |
| verification_tracker.go | VerificationTracker (10 methods) | Runaway auto-completions without human review |
| verification_retry.go | VerificationRetryTracker (4 methods) | Infinite verification retry loops |
| completion_failure_tracker.go | CompletionFailureTracker (5 methods) | Spawning when completion processing is broken |
| rate_limiter.go | RateLimiter (4 methods) | Hourly spawn rate overflow |
| beads_circuit_breaker.go | BeadsCircuitBreaker (5 methods) | Lock cascade during beads instability |
| invariants.go | InvariantChecker (4 methods) | Ghost agents, counter overflow, scope expansion |
| orphan_detector.go | RunPeriodicOrphanDetection | Orphaned in_progress issues (no running agent) |
| orphan_lifecycle.go | RunLifecycleOrphanRecovery | Orphaned agents (lifecycle label-based) |
| phase_timeout.go | RunPeriodicPhaseTimeout | Unresponsive agents (advisory) |
| auto_complete.go | OrcCompleter, IsEffortSmall | Verification gate enforcement (shells to orch complete) |
| pool.go | WorkerPool (8 methods) | Exceeding max concurrent agents |
| capacity.go | AvailableSlots, AtCapacity, ReconcileActiveAgents | Pool-actual divergence |
| stop.go | StopDaemon | Clean daemon shutdown |
| bdcmd.go | runBdCommand* (timeout wrappers) | bd lock pileup |

### COORDINATION (8 files) — Improves outcomes

| File | Functions | What it improves |
|------|-----------|-----------------|
| skill_inference.go | InferSkillFromIssue, InferModelFromSkill | Skill/model routing accuracy |
| focus_boost.go | applyFocusBoost, matchFocusToProject | Priority alignment with strategic focus |
| issue_queue.go | interleaveByProject, roundRobinByProject, SpawnableLabelsFor | Cross-project fairness, priority ordering |
| architect_escalation.go | CheckArchitectEscalation, isImplementationSkill | Hotspot-aware skill routing |
| extraction.go | CheckExtractionNeeded, InferTargetFilesFromIssue | Pre-extraction for bloated files |
| diagnostic.go | ClassifyFailureMode, RunDiagnostics | Failure mode classification for triage |
| hotspot.go | CheckHotspotsForIssue, FormatHotspotWarnings | Hotspot visibility |
| preview.go | Preview, CountSpawnable | What-if visibility for operators |

### INFRASTRUCTURE (8 files) — Neither compliance nor coordination

| File | Functions | Purpose |
|------|-----------|---------|
| interfaces.go | All daemon interfaces | Dependency injection contracts |
| project_resolution.go | ProjectRegistry delegation | Backward-compatible re-exports |
| status.go | WriteStatusFile, ReadStatusFile | Daemon status persistence |
| log.go | DaemonLogger | Dual-stream logging |
| health_signals.go | ComputeDaemonHealth | Health signal derivation |
| utilization.go | GetUtilizationMetrics | Utilization analytics |
| hotspot_checker.go | GitHotspotChecker | Hotspot data provider |
| pidlock.go | PID lock management | Process singleton |

### PERIODIC MAINTENANCE (5 files) — Monitoring/knowledge

| File | Functions | Purpose |
|------|-----------|---------|
| periodic.go | Scheduler + periodic runners | Schedule management |
| reflect.go | RunReflection, RunModelDriftReflection | Knowledge reflection |
| knowledge_health.go | KnowledgeHealth check/create | Knowledge accumulation monitoring |
| agreement_check.go | AgreementCheck check/create | Agreement enforcement |
| beads_health.go | BeadsHealth collect/store | Issue database health |
| friction_accumulator.go | FrictionAccumulator scan/store | Friction tracking |
| artifact_sync.go | ArtifactSync analyze/create | Artifact drift detection |
| recovery.go | GetActiveAgents, ResumeAgentByBeadsID | Stuck agent recovery |
| stall_tracker.go | StallTracker (4 methods) | Token progress detection |
| question_detector.go | QuestionDetection | Pending question surfacing |

### ENTANGLED (3 files) — Does both

| File | Functions | Entanglement type |
|------|-----------|------------------|
| **daemon.go** | OnceExcluding | Compliance checks (verification pause, completion health, rate limit) interleaved with coordination decisions (skill inference, model inference, extraction routing, architect escalation) |
| **issue_selection.go** | NextIssueExcluding | Compliance filtering (skip spawned, skip blocked, skip in_progress, label filter, dependency check) interleaved with coordination sorting (focus boost, priority sort, project interleaving) |
| **completion_processing.go** | ProcessCompletion, CompletionOnce | Compliance verification (VerifyCompletionFull, escalation check, retry budget) interleaved with coordination routing (effort-based auto-complete, tier-based auto-complete, label management) |
| **spawn_execution.go** | spawnIssue | Compliance gates (SpawnPipeline, pool slot, status update, rollback) interleaved with coordination routing (account resolution, auto-complete on orphan) |

**Source:** Full file-by-file audit of pkg/daemon/*.go

**Significance:** The compliance layer is 18 files (already mostly separated). The coordination layer is 8 files (already mostly separated). Only 3-4 files are genuinely entangled, and they're the core pipeline methods.

---

## Finding 2: Entanglement Details — The Three Hot Spots

### Entanglement Site 1: `daemon.go:OnceExcluding()` (lines 280-423)

**The interleaving:**
```
L284-307: COMPLIANCE — verification pause check, completion health check, rate limit check
L323:     COORDINATION — NextIssueExcluding (itself entangled — see Site 2)
L335-341: COORDINATION — InferSkillFromIssue, InferModelFromSkill
L343-382: COORDINATION+COMPLIANCE — CheckExtractionNeeded (coordination routing triggered by compliance condition: file >1500 lines)
L384-402: COORDINATION+COMPLIANCE — CheckArchitectEscalation (coordination routing triggered by compliance condition: hotspot match)
L404-409: COMPLIANCE — spawnIssue (runs dedup pipeline)
L410-422: METADATA — result decoration
```

**Shared state coupling:**
- `d.VerificationTracker` — compliance (pause check)
- `d.CompletionFailureTracker` — compliance (health check)
- `d.RateLimiter` — compliance (rate limit)
- `d.HotspotChecker` — used by BOTH extraction (compliance: prevent adding to bloated files) AND escalation (coordination: route to architect)
- `d.PriorArchitectFinder` — coordination (skip escalation if architect already reviewed)
- `d.SpawnedIssues` — compliance (dedup)

**What would break if split:** Extraction and escalation share HotspotChecker with the dedup pipeline. The extraction result changes the issue being spawned (replaces it with extraction issue), which flows into the compliance pipeline (spawnIssue). Architect escalation changes the skill, which flows into model inference. These are sequential dependencies — compliance gates must run BEFORE coordination routing can finalize.

### Entanglement Site 2: `issue_selection.go:NextIssueExcluding()` (lines 56-178)

**The interleaving:**
```
L57-71:   COORDINATION — fetch issues, expand epics
L74-76:   COORDINATION — applyFocusBoost (priority boosting)
L79-81:   COORDINATION — priority sort
L86:      COORDINATION — interleaveByProject (round-robin fairness)
L88-174:  COMPLIANCE — filtering loop:
  L99-104:  skip recently spawned (dedup)
  L106-111: skip non-spawnable types (type check)
  L113-118: skip blocked issues (status gate)
  L119-125: skip in_progress (status gate)
  L131-136: skip daemon-labeled issues (completion gate)
  L141-153: label filter with epic child exemption (label gate)
  L155-170: blocking dependency check (dependency gate)
```

**Shared state coupling:**
- `d.SpawnedIssues` — compliance
- `d.Config.Label` — compliance (label filter)
- `d.FocusGoal`, `d.FocusBoostAmount`, `d.ProjectDirNames` — coordination

**What would break if split:** The coordination sorting (focus boost, priority, interleaving) must happen BEFORE the compliance filtering loop, because filtering returns the FIRST passing issue. If you split them into separate passes, you'd need to either (a) run compliance on all issues first then sort survivors, or (b) sort first then filter. Current design does (b) — sort then filter — which is correct because filtering should pick from the best-ordered candidates.

### Entanglement Site 3: `completion_processing.go:ProcessCompletion()` (lines 361-542)

**The interleaving:**
```
L366-396: COMPLIANCE — fetch comments, run verification (VerifyCompletionFullWithComments)
L406-420: COMPLIANCE+COORDINATION — determine escalation level (compliance produces signal, coordination interprets)
L423-437: COMPLIANCE — check verification passed, check escalation allows auto-complete
L454-484: COORDINATION — effort-based auto-complete routing (IsEffortSmall → LightAutoCompleter)
L489-511: COORDINATION — tier-based auto-complete routing (reviewTier == "auto")
L515-537: COMPLIANCE — label management (add ready-review, remove triage labels, record verification)
```

**Shared state coupling:**
- `d.VerificationTracker` — compliance (record completion count)
- `d.AutoCompleter` — coordination (effort/tier routing)
- `verify` package — compliance (verification gates)

**What would break if split:** The escalation level determination connects compliance output (verification result) to coordination input (should we auto-complete?). The `DetermineEscalationFromCompletion` function bridges the two — it takes compliance data and produces a coordination signal.

**Source:** Line-by-line audit of daemon.go, issue_selection.go, completion_processing.go, spawn_execution.go

**Significance:** Entanglement is concentrated in exactly the control flow where compliance decisions feed coordination routing. The pattern is always: compliance produces a signal → coordination consumes that signal to route. This is a natural interface boundary.

---

## Finding 3: Already-Clean Separations

**Pure Compliance (confirmed clean):**
- `spawn_gate.go` — Zero coordination logic. The SpawnPipeline runs gates that return allow/reject. No skill inference, no model selection, no priority logic. Each gate has a typed interface (SpawnGate) with clear FailMode. ✅ Clean.
- `verification_tracker.go` — Pure counter + pause logic. No knowledge of skills, models, or routing. ✅ Clean.
- `rate_limiter.go` — Pure sliding window. No awareness of what's being rate-limited. ✅ Clean.
- `beads_circuit_breaker.go` — Pure exponential backoff. No routing. ✅ Clean.
- `spawn_tracker.go` — Pure dedup cache. Title normalization is compliance (prevent content duplicates), not coordination. ✅ Clean.

**Pure Coordination (confirmed clean):**
- `skill_inference.go` — Zero compliance logic. Hierarchical inference (labels → title → description → type) with model mapping. No gates, no dedup, no verification. ✅ Clean.
- `focus_boost.go` — Pure priority manipulation. No gates. ✅ Clean.
- `diagnostic.go` — Pure classification. No gates, no enforcement. ✅ Clean.

**Hidden coupling found (minor):**
- `architect_escalation.go:CheckArchitectEscalation` uses `InferSkillFromLabels` (coordination) and `HotspotChecker` (shared). The function IS coordination (routing to architect) but is TRIGGERED by a compliance condition (hotspot match). This is acceptable coupling — the hotspot check is an input, not a dependency.
- `extraction.go:CheckExtractionNeeded` similarly uses HotspotChecker as input. The extraction routing IS coordination but serves compliance (prevent adding to >1500-line files). Dual purpose — could live in either layer.

**Source:** spawn_gate.go, skill_inference.go, verification_tracker.go, focus_boost.go, rate_limiter.go

**Significance:** 80%+ of daemon code is already cleanly separated. The split is concentrated in 3-4 control-flow orchestration methods.

---

## Synthesis

**Key Insights:**

1. **The entanglement pattern is always "compliance signal → coordination routing"** — Compliance produces a yes/no/pause signal, coordination consumes it to route work. This is a natural function-call boundary, not a deep structural entanglement. The SpawnPipeline already demonstrates the pattern: compliance gates produce PipelineResult, and the caller (spawnIssue) routes based on Allowed.

2. **The Daemon struct is the coupling medium** — All entangled methods share state through the Daemon struct. Both compliance (SpawnedIssues, VerificationTracker, RateLimiter) and coordination (HotspotChecker, FocusGoal, AutoCompleter) hang off the same struct. The fields themselves are cleanly separated — it's the METHODS that interleave access.

3. **Issue selection has the cleanest potential split** — NextIssueExcluding does coordination (sort) then compliance (filter) in sequence. These could be two separate passes: `d.PrioritizeIssues(issues)` then `d.FilterSpawnable(issues)` with no shared state between them.

**Answer to Investigation Question:**

The daemon has ~37 non-test files. 18 are pure compliance, 8 are pure coordination, 8 are infrastructure, and only 3-4 are entangled. The entanglement occurs at the control-flow level (pipeline methods that interleave compliance checks with coordination routing) rather than at the data or interface level. The minimal interface between the two layers is:

**Compliance needs FROM Coordination:**
- The inferred skill (to validate against hotspot gates)
- The selected issue (to run dedup gates against)
- The verification result (to check completion quality)

**Coordination needs FROM Compliance:**
- `allowed: bool` — can we proceed?
- `paused: bool` — is the daemon paused for verification?
- `atCapacity: bool` — any slots available?
- `verificationPassed: bool` — did the completion pass verification?
- `escalationLevel` — how should we route this completion?

This is a narrow interface. The SpawnPipeline's `PipelineResult{Allowed, RejectedBy, Advisories}` is already the right shape for the spawn path.

---

## Structured Uncertainty

**What's tested:**

- ✅ Every exported function in pkg/daemon/ classified (verified: read all 37+ source files)
- ✅ SpawnPipeline is pure compliance (verified: no coordination logic in spawn_gate.go)
- ✅ InferSkillFromIssue is pure coordination (verified: no gates/dedup in skill_inference.go)
- ✅ OnceExcluding interleaves compliance and coordination (verified: line-by-line audit)

**What's untested:**

- ⚠️ Whether the proposed interface would maintain identical behavior (not implemented)
- ⚠️ Whether periodic tasks (reflect, health checks) would be affected by the split (likely no — they're already independent)
- ⚠️ Performance impact of two-pass filtering in issue_selection (likely negligible — list is small)

**What would change this:**

- Finding that compliance gates need coordination context beyond skill/issue would widen the interface
- Finding that coordinator routing needs intermediate compliance state (not just allowed/rejected) would deepen the coupling
- The auto-complete path in completion_processing already shows this: escalation level is a richer signal than just allowed/rejected

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Split OnceExcluding into compliance gateway + coordination router | architectural | Changes daemon internal boundaries, affects spawn/completion/selection subsystems |
| Extract ComplianceGateway interface | architectural | New abstraction crossing spawn and completion paths |
| Extract CoordinationRouter interface | architectural | New abstraction crossing skill inference, extraction, escalation |

### Recommended Approach ⭐

**Phase-Split Refactoring** — Split each entangled method into a compliance phase and a coordination phase, connected by a typed result struct.

**Why this approach:**
- Matches the existing SpawnPipeline pattern (already proven in spawn_gate.go)
- Each phase can be tested independently (compliance gates don't need skill inference mocks, coordination doesn't need dedup mocks)
- The interface between phases is narrow (see Finding 2)
- Preserves behavioral equivalence (sequential execution, same ordering)

**Trade-offs accepted:**
- Slightly more function calls per spawn cycle (negligible vs. subprocess/HTTP overhead)
- Two structs where one existed before (but the Daemon struct is already 170 lines)

**Implementation sequence:**

1. **Extract compliance pre-checks from OnceExcluding** — Move verification pause, completion health, rate limit checks into a `PreSpawnComplianceCheck() → ComplianceResult` method. This is mechanical extraction with zero behavioral change.

2. **Extract coordination routing from OnceExcluding** — Move skill inference, extraction check, architect escalation into a `RouteIssue(issue) → RoutingDecision` method that returns (skill, model, extractionResult, escalationResult).

3. **Make OnceExcluding a thin orchestrator** — Compliance pre-check → NextIssue (coordination sorting + compliance filtering already partially split) → RouteIssue → spawnIssue (compliance pipeline + execution).

4. **Same pattern for ProcessCompletion** — `VerifyCompletion() → ComplianceVerdict` then `RouteCompletion(verdict, agent) → CompletionAction`.

### Alternative Approaches Considered

**Option B: Package-level split (pkg/daemon/compliance/, pkg/daemon/coordination/)**
- **Pros:** Enforced at the compiler level, impossible to accidentally re-entangle
- **Cons:** Massive refactor, breaks all test files, changes import paths across cmd/orch/
- **When to use instead:** If entanglement keeps recurring after method-level split

**Option C: Interface extraction only (ComplianceGateway + CoordinationRouter interfaces)**
- **Pros:** Maximum flexibility for testing, enables alternative implementations
- **Cons:** Interfaces without structural separation add indirection without preventing re-entanglement
- **When to use instead:** If the primary goal is testability rather than architectural clarity

**Rationale for recommendation:** The phase-split approach matches the existing pattern (SpawnPipeline), requires no import changes, and can be done incrementally (one method at a time).

---

### Implementation Details

**What to implement first:**
- OnceExcluding split — highest traffic path, most entangled, most impactful
- NextIssueExcluding split — straightforward (coordination sort → compliance filter are already sequential)

**Things to watch out for:**
- ⚠️ Extraction routing changes the issue being spawned — the coordination result flows INTO compliance (spawnIssue runs dedup on the replacement issue). This is the tightest coupling point.
- ⚠️ Architect escalation changes the skill, which changes the model — coordination outputs feed back into coordination. This is internal to the coordination layer and should stay together.
- ⚠️ The CompletionOnce loop has its own dedup (VerificationRetryTracker, CompletionDedupTracker) before calling ProcessCompletion — these are compliance checks at the loop level, separate from ProcessCompletion's internal compliance.

**Areas needing further investigation:**
- Whether HotspotChecker should be classified as compliance infrastructure or coordination infrastructure (currently used by both)
- Whether the auto-complete routing in ProcessCompletion should be its own coordination method

**Success criteria:**
- ✅ Each entangled method split into compliance + coordination phases
- ✅ No compilation changes required outside pkg/daemon/
- ✅ All existing tests pass without modification
- ✅ The Daemon struct can be understood as: infrastructure + compliance components + coordination components

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go` — Primary entanglement site (OnceExcluding, Daemon struct)
- `pkg/daemon/issue_selection.go` — Issue filtering + sorting entanglement
- `pkg/daemon/spawn_execution.go` — Spawn pipeline + account routing
- `pkg/daemon/spawn_gate.go` — Already-clean compliance (SpawnPipeline)
- `pkg/daemon/skill_inference.go` — Already-clean coordination
- `pkg/daemon/completion_processing.go` — Completion verification + routing
- `pkg/daemon/extraction.go` — Hotspot extraction (hybrid)
- `pkg/daemon/architect_escalation.go` — Hotspot routing
- `pkg/daemon/interfaces.go` — All daemon interfaces
- `pkg/daemon/focus_boost.go` — Priority boosting (pure coordination)
- `pkg/daemon/hotspot_checker.go` — Hotspot data provider
- All 37+ files in pkg/daemon/ audited via subagent

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-12-compliance-coordination-bifurcation-as-models.md` — Motivating observation about value trajectories
- **Thread:** `.kb/threads/2026-03-13-compliance-coordination-bifurcation-designing-split.md` — Design discussion

---

## Investigation History

**2026-03-13:** Investigation started
- Initial question: Where exactly are compliance and coordination entangled in the daemon?
- Context: Thread observation that compliance tooling decreases in value as models improve, while coordination increases

**2026-03-13:** Classification complete
- All 37+ non-test files classified as compliance/coordination/infrastructure/entangled
- 3 primary entanglement sites identified with line-level detail

**2026-03-13:** Investigation completed
- Status: Complete
- Key outcome: 80%+ of daemon is already cleanly separated; entanglement is in 3 control-flow methods with a narrow interface between layers
