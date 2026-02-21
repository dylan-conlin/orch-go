## Summary (D.E.K.N.)

**Delta:** The verification infrastructure is far more sophisticated than the completion-verification model describes (14 gates vs claimed 3), all gates are wired and tested, but verification stops at "does evidence exist in comments?" — never actually executing tests or smoke runs.

**Evidence:** Full audit of pkg/verify/ (43 files, 100+ functions), completion pipeline (14 gates traced), daemon verification (IsPaused wired, VerificationTracker operational), test evidence (22 true-positive + 11 false-positive patterns with 116 test cases).

**Knowledge:** The verification spectrum has a clear boundary: strong at artifact-existence and comment-pattern levels, absent at test-execution and behavioral levels. The only gate that runs something real is `go build`. The completion-verification model is ~60% stale and needs rewriting.

**Next:** Rewrite the completion-verification model from this audit's inventory. Then decide whether to add test-execution gates (running `go test` during `orch complete`) as the next verification level.

**Authority:** architectural - Model rewrite crosses knowledge boundaries; adding test-execution gates is an architectural decision affecting completion latency and daemon throughput.

---

# Investigation: Verification Infrastructure End-to-End Audit

**Question:** What verification features exist in orch-go, which are actually wired into real flows, and what levels of the verification spectrum are covered vs missing?

**Defect-Class:** configuration-drift

**Started:** 2026-02-20
**Updated:** 2026-02-20
**Owner:** Claude (codebase-audit skill)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| `.kb/models/completion-verification.md` (model, 2026-01-14) | Contradicts (model significantly stale) | Yes - against source code | 3 deleted files, 3 vs 14 gates, wrong pseudocode |
| `~/orch-knowledge/kb/models/control-plane-bootstrap.md` (model, 2026-02-15) | Confirms (enforcement theater pattern) | Yes - applies to verification spectrum gap | None |
| `~/orch-knowledge/kb/models/verifiability-first-development.md` (model, 2026-02-03) | Extends (audit maps spectrum) | Yes - maps paradigm to concrete gates | None |
| `.kb/models/completion-verification/probes/2026-02-13-friction-gate-inventory-all-subsystems.md` | Extends | Partially (consistent with 14-gate finding) | None |

---

## Finding 1: 14 Verification Gates, All Wired and Tested

**Evidence:** Complete inventory of pkg/verify/ reveals 14 distinct verification gates, all called from `VerifyCompletionFull()` in `check.go`, each with dedicated test files:

| # | Gate | File | What It Checks | Skippable | Auto-Skip Conditions |
|---|---|---|---|---|---|
| 1 | Phase Complete | beads_api.go | Agent reported "Phase: Complete" via beads comment | Yes | Untracked agent |
| 2 | Synthesis | check.go | SYNTHESIS.md exists and has content | Yes | Knowledge skill, light tier, orchestrator |
| 3 | Test Evidence | test_evidence.go | Actual test execution output in comments (not vague claims) | Yes | Non-implementation skill, markdown-only changes |
| 4 | Visual Verification | visual.go | Web/ changes have visual evidence + human approval | Yes | No web changes, approval present |
| 5 | Git Diff | git_diff.go | SYNTHESIS.md file claims match actual git changes | Yes | No file claims |
| 6 | Build | build_verification.go | `go build ./...` passes | Yes | Non-Go project, no Go file changes |
| 7 | Constraint | constraint.go | SPAWN_CONTEXT.md constraints satisfied | Yes | No constraints defined |
| 8 | Phase Gates | phase_gates.go | Required phases reported (from SPAWN_CONTEXT.md) | Yes | No phases required |
| 9 | Skill Output | skill_outputs.go | Required deliverable files exist (from skill.yaml) | Yes | Skill has no outputs |
| 10 | Decision Patch Limit | decision_patches.go | <3 patches to same decision | Yes | Under limit |
| 11 | Accretion | accretion.go | No >50 lines added to files >800/1500 lines | Yes | Growth under threshold |
| 12 | Explain-Back (Gate1) | complete_cmd.go + checkpoint | Human explains what was built | Yes | Tier 3 |
| 13 | Behavioral (Gate2) | complete_cmd.go + checkpoint | Human confirms behavior verified | No (--verified flag) | Tier 2/3 |
| 14 | Handoff Content | check.go | SESSION_HANDOFF.md has non-placeholder TLDR + Outcome | Yes | Worker tier |

**Source:** `pkg/verify/check.go:VerifyCompletionFullWithComments()` (lines 300-460), each gate's dedicated source file, and corresponding `*_test.go` files.

**Significance:** The system is far more sophisticated than the model claims. All 14 gates are implemented, integrated into the completion flow, have test coverage, and support targeted bypass with audit trail. This is NOT enforcement theater — it's working infrastructure.

---

## Finding 2: Completion-Verification Model Is ~60% Stale

**Evidence:**

| Model Claim | Reality | Status |
|---|---|---|
| "Three independent gates (Phase, Evidence, Approval)" | 14 gates in tier-aware pipeline | **Wrong** |
| Source: `pkg/verify/phase.go` | Deleted. Phase logic in `beads_api.go` | **Wrong** |
| Source: `pkg/verify/evidence.go` | Deleted. Split into `test_evidence.go` + `visual.go` | **Wrong** |
| Source: `pkg/verify/cross_project.go` | Deleted. Integrated into `check.go` | **Wrong** |
| Pseudocode: `strings.Contains("Phase: Complete")` | Uses regex `ParsePhaseFromComments()` | **Wrong** |
| Pseudocode: `containsImageURL()` for evidence | Uses 22 framework-specific patterns | **Wrong** |
| Evolution stops at Phase 6 (Jan 14, 2026) | Phase 7 (Feb 2026) removed 3 noise gates | **Incomplete** |
| No mention of daemon verification | Daemon runs full verification + VerificationTracker | **Missing** |
| Tier table: light/full/orchestrator | Correct but incomplete (missing auto-skip details) | **Partial** |
| "Gates are independent and cumulative" | Confirmed — all checked, all reported | **Correct** |
| "Knowledge work surfaces for review" | Confirmed — escalation model works | **Correct** |

**Source:** `.kb/models/completion-verification.md` compared against current `pkg/verify/` codebase.

**Significance:** The model is useful as historical context (evolution section) but misleading as current documentation. Anyone reading the model would think there are 3 gates checking 3 things; the reality is 14 gates checking 14 things with tier-aware routing and anti-theater mechanisms.

---

## Finding 3: The Verification Spectrum Has a Clear Boundary

**Evidence:** Mapping each gate to the verification spectrum:

| Spectrum Level | Gates Covering It | Strength |
|---|---|---|
| **Agent claims done** | Phase Complete, Synthesis, Handoff Content | Strong |
| **Artifacts exist** | Skill Output, Constraint, Phase Gates | Strong |
| **Evidence of testing exists** | Test Evidence (22 patterns, 11 false-positive filters) | Strong |
| **Binary compiles** | Build Verification (`go build ./...`) | Strong (actually executes) |
| **Claims match reality** | Git Diff (SYNTHESIS vs actual changes) | Strong |
| **Human comprehends** | Explain-Back (Gate1) | Strong (unfakeable) |
| **Human verifies behavior** | Behavioral (Gate2, Tier 1 only) | Strong (unfakeable) |
| **Tests actually pass** | None — checks for evidence, doesn't run tests | **Missing** |
| **Smoke test passes** | None | **Missing** |
| **Live e2e passes** | None (Playwright evidence checked, not executed) | **Missing** |
| **Adversarial resilience** | None — agent could fabricate test output patterns | **Missing** |

**The boundary:** Everything above "Tests actually pass" is covered. Everything at or below is not. The system verifies that agents CLAIM to have tested, not that tests actually passed.

**Source:** Analysis of all 14 gates' implementation logic, cross-referenced with the verifiability-first development model's "instrument flying" paradigm.

**Significance:** This is the verification vocabulary gap the task set out to find. The system has excellent "existence verification" (do artifacts exist?) and good "anti-theater verification" (are claims specific enough to be credible?) but no "execution verification" (did the thing actually run?). The only execution gate is `go build`.

---

## Finding 4: Daemon Verification Is Fully Operational

**Evidence:**

1. **Full verification before auto-marking:** `ProcessCompletion()` in `pkg/daemon/completion_processing.go` runs `VerifyCompletionFull()` before marking issues as `daemon:ready-review`.

2. **Verification tracker wired into loop:** `daemon.go` lines 342-380 check `IsPaused()` BEFORE spawning. Default threshold: 3 unverified completions triggers pause.

3. **State persistence across restarts:** `SeedFromBacklog()` reads checkpoint file on startup, diffs against completed issues. Counter reflects reality across daemon restarts.

4. **Signal files for human bridge:** `~/.orch/daemon-verification.signal` (written by `orch complete`), `~/.orch/daemon-resume.signal` (written by `orch daemon resume`).

5. **Escalation model routes decisions:** `EscalationNone/Info/Review` → auto-mark ready. `EscalationBlock/Failed` → require human intervention.

6. **4-layer dedup defense:** Session-level, content-aware, fresh-status, primary persistent dedup prevent duplicate spawns.

7. **Completion failure tracking:** After 3+ consecutive completion processing failures, daemon pauses spawning.

**Source:** `cmd/orch/daemon.go`, `pkg/daemon/daemon.go`, `pkg/daemon/completion_processing.go`, `pkg/daemon/verification_tracker.go`

**Significance:** The daemon is NOT autonomously closing work without checks. This addresses the control-plane bootstrap model's concern about enforcement theater. The VerificationTracker was built through the bootstrap sequence (daemon off → build mechanism → verify → activate), and this audit confirms it's operational.

---

## Finding 5: Anti-Theater Mechanisms Are Strongest in Test Evidence

**Evidence:** `test_evidence.go` contains the most sophisticated anti-theater design in the system:

**22 true-positive patterns** across 5 frameworks (Go, Node.js, Python, Rust, Playwright) requiring:
- Framework-specific output format (`ok package 0.123s`, `Tests: 15 passed`)
- Quantifiable counts (`15 tests passed`, NOT just `tests passed`)
- Timing information (`2.3s`, `0.123s`)

**11 false-positive patterns** explicitly rejecting enforcement theater:
- `tests pass` / `all tests pass` (bare claims without counts)
- `tests should pass` / `tests will pass` (expectations, not evidence)
- `verified tests pass` / `confirmed tests pass` (meta-claims)
- `tests passing` / `tests are passing` (state claims)

**116+ test cases** verifying both true and false positive detection.

**Source:** `pkg/verify/test_evidence.go` (patterns), `pkg/verify/test_evidence_test.go` (tests)

**Significance:** This is the system's best defense against the enforcement theater anti-pattern from the control-plane-bootstrap model. An agent can't just say "tests pass" — it must provide output that looks like actual test framework output. However, it's still gameable: an agent could write `go test ./... - PASS (47 tests in 2.3s)` without running tests. The anti-theater is pattern-based, not execution-based.

---

## Finding 6: Coaching Plugin Monitoring Gap

**Evidence:** The coaching plugin (`.opencode/plugin/coaching.ts`) detects 8 behavioral patterns including frame collapse, analysis paralysis, and worker health metrics (tool failure rate, context usage, time-in-phase, commit gap).

**However:** Detection only works for OpenCode API spawns (`session.metadata.role='worker'` set via HTTP header). Claude CLI/tmux spawns (the "escape hatch" for critical infrastructure work) bypass HTTP session creation entirely.

**Source:** `.opencode/plugin/coaching.ts`, `cmd/orch/serve_coaching.go`

**Significance:** The escape hatch — designed for critical work when the primary path is unstable — has NO behavioral monitoring. This is an inversion of what you'd want: normal work is monitored, critical work isn't. The coaching plugin is architecturally unable to monitor tmux-based agents.

---

## Finding 7: Preview/Dry-Run Capabilities Exist at Three Levels

**Evidence:**

1. **`orch daemon preview`** — Shows next spawnable issue with inferred skill, model, rejection reasons. Does not spawn.
2. **`orch daemon run --dry-run`** — Runs full issue selection and verification logic but makes no beads changes.
3. **`orch daemon once`** — Spawns exactly one issue and exits (step-by-step mode).

All three are implemented in `cmd/orch/daemon.go` and `pkg/daemon/preview.go`.

**Source:** `cmd/orch/daemon.go` (lines 58-84, 775-932), `pkg/daemon/preview.go` (lines 47-124)

**Significance:** The daemon has three levels of cautious operation. These are NOT theater — they're real preview/dry-run capabilities that allow humans to inspect what the daemon would do before it does it.

---

## Synthesis

**Key Insights:**

1. **The verification system is real, not theater.** 14 gates, all wired, all tested, all skippable with audit trail. The targeted bypass system (replacing blanket --force) shows mature engineering — specific gates can be bypassed while others still run, with logged reasons.

2. **The model is dangerously stale.** Anyone consulting the completion-verification model would significantly underestimate the system. The model says 3 gates; reality is 14. The model references deleted files. This is itself a form of "organizational drift" — the documentation that describes the verification system doesn't verify against reality.

3. **The verification spectrum has a sharp boundary at "execution."** Every gate checks for evidence or artifacts. Only `go build` actually executes something. This means the system trusts agents to report honestly about test execution. The anti-theater mechanisms (false positive filters) raise the bar from "say anything" to "say something specific enough to be credible" — but they don't close the gap to "prove it by running the tests."

4. **The daemon verification solves the bootstrap problem.** VerificationTracker with IsPaused, SeedFromBacklog, signal files, and threshold-based pause is operational. The control-plane-bootstrap model's concerns are addressed in the current codebase.

5. **The coaching plugin has an architectural blindspot.** The escape hatch (Claude CLI/tmux) — designed for critical infrastructure work — is the ONE path that lacks behavioral monitoring. This creates an inversion where the most important work has the least oversight.

**Answer to Investigation Question:**

The verification infrastructure in orch-go is substantially more sophisticated than documented. It covers:
- **Working end-to-end:** 14 verification gates, daemon verification tracker, escalation model, targeted bypasses, anti-theater test evidence detection
- **Partially working:** Coaching plugin (operational for OpenCode spawns, blind to tmux spawns)
- **Theater:** None found — every implemented feature is wired and tested
- **Missing:** Test execution verification (running tests, not just checking for evidence), smoke test capability, e2e execution, adversarial resilience

The verification vocabulary gap is clear: the system needs a way to express "tests were run and passed" vs "agent claims tests were run and passed." The current system only verifies the latter.

---

## Structured Uncertainty

**What's tested:**

- All 14 gates traced from implementation through integration to test coverage
- Daemon verification tracker traced through daemon loop, signal files, and backlog seeding
- Test evidence false-positive patterns verified against test_evidence_test.go (116+ cases)
- Model staleness verified by comparing claimed source files against actual filesystem

**What's untested:**

- Whether the verification tracker actually pauses in production (only confirmed via code path analysis, not behavioral observation)
- Whether the coaching plugin's worker detection chain works end-to-end in current deployment
- Whether any agent has successfully gamed the test evidence gate (fabricating credible output)
- Performance impact of running all 14 gates during completion (latency not measured)

**What would change this:**

- Running `orch complete` on a real agent and observing each gate fire would confirm behavioral correctness
- A daemon restart during verification tracker pause would test SeedFromBacklog
- An adversarial test (agent fabricating `go test` output) would reveal anti-theater limits

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|---|---|---|
| Rewrite completion-verification model | implementation | Documentation update within existing patterns |
| Add `go test` execution gate | architectural | Affects completion latency, daemon throughput, cross-component impact |
| Fix coaching plugin tmux blindspot | architectural | Requires architectural decision about monitoring escape hatch |
| Add adversarial testing for anti-theater | strategic | Changes verification philosophy from trust-but-verify to zero-trust |

### Recommended Approach: Rewrite Model + Execution Gate Design

**Why this approach:**
- Model rewrite is zero-risk, high-value (documentation matching reality)
- Execution gate design session needed before implementation (latency tradeoffs)
- Coaching plugin fix is a separate concern (different architectural boundary)

**Implementation sequence:**
1. **Rewrite completion-verification model** from this audit's inventory — zero risk, immediate value
2. **Design session for test-execution gate** — evaluate `go test` during `orch complete` vs `go vet` vs both, measure latency impact
3. **Investigate coaching plugin tmux path** — can worker health metrics be emitted via file rather than HTTP?

### Alternative Approaches Considered

**Option B: Add `go test` gate immediately**
- **Pros:** Closes biggest verification spectrum gap
- **Cons:** Latency impact unknown, may slow daemon throughput, needs design
- **When to use instead:** If adversarial gaming of test evidence gate is observed

**Option C: Accept current verification level**
- **Pros:** System already catches most issues, anti-theater is good enough
- **Cons:** Fundamental gap between "evidence exists" and "evidence is real" remains
- **When to use instead:** If verification latency is unacceptable and false positive rate is low

---

### Implementation Details

**What to implement first:**
- Model rewrite (zero-risk, immediate documentation value)

**Things to watch out for:**
- Model rewrite should include all 14 gates with current file references
- Evolution section should be extended through Phase 7
- Daemon verification section should be added

**Success criteria:**
- Model references match actual source files
- Gate count in model matches reality
- New reader of model gets accurate picture of system

---

## References

**Files Examined:**
- `pkg/verify/*.go` (43 files) — Complete verification package
- `cmd/orch/complete_cmd.go` — Completion orchestrator
- `cmd/orch/complete_pipeline.go` — Pipeline phases
- `cmd/orch/complete_verify.go` — SkipConfig
- `cmd/orch/daemon.go` — Daemon main loop with IsPaused check
- `pkg/daemon/daemon.go` — Core daemon logic
- `pkg/daemon/completion_processing.go` — ProcessCompletion with VerifyCompletionFull
- `pkg/daemon/verification_tracker.go` — Threshold pause mechanism
- `.opencode/plugin/coaching.ts` — Behavioral detection
- `cmd/orch/serve_coaching.go` — Worker health metrics API
- `.kb/models/completion-verification.md` — Model being probed
- `~/orch-knowledge/kb/models/control-plane-bootstrap.md` — Enforcement theater model
- `~/orch-knowledge/kb/models/verifiability-first-development.md` — Verification paradigm

**Related Artifacts:**
- **Probe:** `.kb/models/completion-verification/probes/2026-02-20-probe-verification-infrastructure-audit.md`
- **Model:** `.kb/models/completion-verification.md` (needs rewrite)
- **Decision:** `.kb/decisions/2026-02-14-verifiability-first-hard-constraint.md`

---

## Investigation History

**2026-02-20:** Investigation started
- Spawned 4 parallel exploration agents covering pkg/verify/, completion pipeline, daemon verification, test evidence
- All agents returned comprehensive inventories

**2026-02-20:** All parallel audits complete
- 14 gates identified, all wired and tested
- Model staleness confirmed (3 deleted files, 3 vs 14 gates)
- Verification spectrum gap identified (evidence-existence vs test-execution)
- Investigation completed
