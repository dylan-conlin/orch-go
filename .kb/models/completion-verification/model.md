# Model: Completion Verification Architecture

**Domain:** Completion / Verification / Quality Gates
**Last Updated:** 2026-03-26
**Synthesized From:** 31 investigations, 26 probes, completion.md guide, end-to-end infrastructure audit (Feb 20, 2026), code review gate design (Feb 25, 2026)

---

## Summary (30 seconds)

Completion verification operates through **14 gates** organized into **4 verification levels** (V0–V3) and classified by **3 gate types** (execution, evidence, judgment). Each level is a strict superset of the one below: V0 (Acknowledge) checks only that the agent reported completion; V1 (Artifacts) adds deliverable and constraint checks; V2 (Evidence) adds test evidence, build, and git diff checks; V3 (Behavioral) adds visual verification and human observation gates. The three gate types define what kind of verification each gate provides: **execution-based** gates produce provenance (binary pass/fail from running code), **evidence-based** gates check claims against patterns (anti-theater detection), and **judgment-based** gates require human comprehension. The verification level is determined at spawn time from skill type and issue type, stored in AGENT_MANIFEST.json, and flows through to `orch complete`. **Targeted bypasses** (`--skip-{gate} "reason"`) remain as an escape hatch for edge cases, but well-configured spawns should require zero skip flags — this invariant is currently violated by a known conflict between light tier and V2 level (see "Why This Fails" §5). The daemon runs the same `VerifyCompletionFull()` pipeline with threshold-based pause to prevent unchecked auto-completion.

---

## Core Mechanism

### The 14 Gates

| # | Gate | Constant | Type | What It Checks |
|---|------|----------|------|----------------|
| 1 | Phase Complete | `phase_complete` | Evidence | Agent reported "Phase: Complete" via beads comment |
| 2 | Synthesis | `synthesis` | Evidence | SYNTHESIS.md exists and is non-empty (skipped for light tier and knowledge-producing skills) |
| 3 | Handoff Content | `handoff_content` | Evidence | SESSION_HANDOFF.md has TLDR & Outcome filled (orchestrator tier only) |
| 4 | Skill Output | `skill_output` | Evidence | Required skill outputs exist (from skill.yaml `outputs.required`) |
| 5 | Phase Gates | `phase_gate` | Evidence | Required skill phases were reported in beads comments |
| 6 | Constraint | `constraint` | Evidence | Constraint patterns from SPAWN_CONTEXT match actual files |
| 7 | Decision Patch Limit | `decision_patch_limit` | Evidence | Decision patch count within limits |
| 8 | Test Evidence | `test_evidence` | Evidence | Evidence of actual test execution in beads comments (anti-theater detection) |
| 9 | Git Diff | `git_diff` | Evidence | Git changes match SYNTHESIS.md claims |
| 10 | Build | `build` | Execution | Project compiles (`go build ./...`) — the only unfakeable gate |
| 11 | Accretion | `accretion` | Evidence | File size growth within limits; pre-existing bloat (file was already >1500 before agent) downgrades to WARNING; agent-caused threshold crossing blocks |
| 12 | Visual Verification | `visual_verification` | Evidence | Screenshot/Playwright evidence for web/ changes, with risk assessment |
| 13 | Explain-Back | `explain_back` | Judgment | Orchestrator explains what was built and why (gate1/comprehension) |
| 14 | Behavioral | `behavioral` | Judgment | Human confirms behavior was observed working (gate2, V3 only) |

**Key design property:** Gates are structurally independent but functionally level-selective. Each gate can fail independently, and all applicable gates must pass (or be explicitly skipped). The verification level determines which subset of gates fires.

**Source:** `pkg/verify/check.go` — constants at top, `VerifyCompletionFull()` orchestrates all gates

### Gate Type Taxonomy

The 14 gates fall into three types based on what kind of verification they provide:

| Gate Type | Gates | Produces | Provenance? |
|---|---|---|---|
| **Execution-based** | Build (#10) | Binary pass/fail from running code | ✓ Verifiable, deterministic |
| **Evidence-based** | Phase Complete (#1), Synthesis (#2), Handoff Content (#3), Skill Output (#4), Phase Gates (#5), Constraint (#6), Decision Patch Limit (#7), Test Evidence (#8), Git Diff (#9), Accretion (#11), Visual Verification (#12) | Pattern match against claims | Partial — detects theater, not correctness |
| **Judgment-based** | Explain-Back (#13), Behavioral (#14) | Human comprehension/observation | Human only — valid because human takes responsibility |

**Why this taxonomy matters:**

- **Execution-based gates produce truth.** The output is independent of opinion — code compiles or it doesn't. These are the only unfakeable gates. Expanding this type (e.g., `go vet`, `staticcheck`) increases verification strength without adding judgment.
- **Evidence-based gates check claims.** They detect theater (vague "tests pass" vs. concrete output) and verify artifacts exist, but cannot verify correctness. An agent can fabricate framework-specific test output that passes these gates.
- **Judgment-based gates require humans.** Explain-back forces comprehension. Behavioral forces observation. These are valid precisely because a human performs them and takes responsibility for the judgment.

**Excluded type — Agent judgment:** The pipeline deliberately excludes agent-judgment gates (e.g., AI code review). Agent reviewing agent code is a closed loop — same model family, same blind spots, no provenance chain. It produces opinion that neither the orchestrator nor Dylan can independently verify. See decision: `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md`.

**Source:** Investigation `.kb/investigations/2026-02-25-design-code-review-gate-for-completion-pipeline.md`, probe `probes/2026-02-25-probe-code-review-gate-design.md`

### Four Verification Levels (V0–V3)

The 14 gates are organized into four levels, each a strict superset of the level below:

| Level | Name | Gates That Fire | Typical Work |
|-------|------|----------------|--------------|
| **V0** | Acknowledge | Phase Complete | Config changes, README updates, issue creation |
| **V1** | Artifacts | V0 + Synthesis, Handoff Content, Skill Output, Phase Gates, Constraint, Decision Patch Limit | Investigations, architect designs, research, audits |
| **V2** | Evidence | V1 + Test Evidence, Git Diff, Build, Accretion | Feature implementation, bug fixes, debugging |
| **V3** | Behavioral | V2 + Visual Verification, Explain-Back, Behavioral | UI features, user-facing changes, critical behavioral modifications |

**Level determination:** `V_default = max(skill_level, issue_type_level)`

| Skill | Default Level |
|-------|--------------|
| issue-creation | V0 |
| investigation, architect, research, codebase-audit, design-session | V1 |
| feature-impl, systematic-debugging, reliability-testing | V2 |
| (any with web/ changes detected) | auto-elevated to V3 |

| Issue Type | Minimum Level |
|------------|--------------|
| task, question | no minimum |
| investigation, probe | V1 |
| feature, bug, decision | V2 |

**Override:** Orchestrator can declare `--verify-level V0` at spawn time to override defaults.

**Source:** Design in `.kb/investigations/2026-02-20-inv-architect-verification-levels.md`, current implementation in `pkg/verify/check.go`, spawn tier defaults in `pkg/spawn/config.go`

### Phase Gate (Gate 1)

**What:** Verifies agent reported completion via beads comment.

**Implementation:** Uses regex `(?i)Phase:\s*(\w+)(?:\s*[-–—]\s*(.*))?` to parse latest phase from comments. Returns `PhaseStatus{Phase, Summary, Found, PhaseReportedAt}`.

**Source:** `pkg/verify/beads_api.go:ParsePhaseFromComments()`

### Test Evidence Gate (Gate 8)

**What:** Detects actual test execution evidence in beads comments, with anti-theater mechanisms.

**True positive patterns (22):** Framework-specific output with counts — `go test ./... - PASS`, `ok package 0.123s`, `--- PASS: TestName`, `15 passing, 0 failing`, `pytest - 15 passed`, `cargo test - ok`, `playwright test - 5 passed`.

**False positive patterns (11):** Vague claims without evidence — `tests pass` (no count), `all tests pass`, `verified tests pass`, `tests should pass`, `assuming tests pass`, `tests will pass`, `tests are passing`. These are explicitly rejected.

**Exemptions:**
- Markdown-only changes (no test harness needed)
- Files outside project directory (no local test harness available)
- No code changes detected (only config/docs changed)
- Skills excluded: investigation, architect, research, design-session, codebase-audit, issue-creation

**Source:** `pkg/verify/test_evidence.go` — patterns at lines 79-136, `HasTestExecutionEvidence()` for detection, `VerifyTestEvidenceForCompletion()` for the completion flow

### Visual Verification Gate (Gate 12)

**What:** Requires screenshot/Playwright evidence for UI changes, with risk-based assessment.

**Risk assessment heuristics:**
- `WebRiskNone` — No web changes → gate skipped
- `WebRiskLow` — Trivial CSS changes (≤10 lines) → NO verification needed
- `WebRiskMedium` — Component/layout changes → verification REQUIRED
- `WebRiskHigh` — New routes, major UX changes → verification REQUIRED

**Human approval patterns:** `✅ APPROVED`, `UI APPROVED`, `LGTM UI`, `I approve the UI/visual/changes`

**Source:** `pkg/verify/visual.go` — risk assessment at `AssessWebChangeRisk()`, evidence detection at `HasVisualVerificationEvidence()`, approval at `HasHumanApproval()`

### Checkpoint Tiers (Gates 13–14)

**What:** Human comprehension and behavioral verification at completion time.

| Checkpoint Tier | Issue Types | Gate1 (Explain-Back) | Gate2 (Behavioral) |
|----------------|-------------|---------------------|-------------------|
| Tier 1 | feature, bug, decision | Required | Required |
| Tier 2 | investigation, probe | Required | Not required |
| Tier 3 | task, question, other | Not required | Not required |

**Implementation:**
- Gate1: Requires `--explain "text"` on `orch complete` — orchestrator explains what was built
- Gate2: Requires `--verified` flag — orchestrator confirms behavior observed
- Both stored as checkpoint records in `~/.orch/verification-checkpoints.jsonl`

**Source:** `pkg/checkpoint/checkpoint.go:TierForIssueType()`, `RequiresGate2()`

### Tier-Aware Verification

Three workspace tiers route to different verification flows:

| Tier | Artifact Required | Beads Checks | Phase Reporting | Flow |
|------|-------------------|--------------|-----------------|------|
| **light** | None | Yes | Yes | `verifyLight()` — Phase + commits, skips SYNTHESIS.md |
| **full** | SYNTHESIS.md | Yes | Yes | `verifyFull()` — All applicable gates |
| **orchestrator** | SESSION_HANDOFF.md | No | No | `VerifyOrchestratorCompletion()` — Handoff only |

**Implementation:**
```go
tier := readTierFile(workspace)  // reads .tier file in workspace
switch tier {
case "light":
    // Phase + commits, synthesis auto-skipped
case "full":
    // All gates for the verification level
case "orchestrator":
    // SESSION_HANDOFF.md checks only
}
```

**Source:** `pkg/verify/check.go:VerifyCompletionWithTier()`

### Targeted Bypass System

Each gate can be individually bypassed with a required reason (min 10 characters):

```bash
--skip-test-evidence --skip-reason "markdown-only change, no tests applicable"
--skip-build --skip-reason "CI will catch build, local toolchain broken"
--skip-visual --skip-reason "CSS-only change verified via diff review"
```

**Constraint:** Bypass events are logged to `~/.orch/events.jsonl` with gate name, reason, beads ID, and skill for observability. `orch stats` shows pass/fail/bypass rates per gate.

**Pipeline propagation:** Skip flags must propagate to ALL downstream systems that independently enforce the same check. `--skip-phase-complete` bypasses orch's gate but also triggers `bd close --force` — because `bd close` has its own Phase: Complete check. Without propagation, the skip is ineffective (orch proceeds, `bd close` fails anyway). Fixed via `verify.ForceCloseIssue()`.

**Bypass statistics (post targeted-skip rollout, as of Feb 2026):**
- `--force` dropped from 72.8% (pre-rollout) to 1.8% (instrumented post-rollout)
- Remaining bypass noise concentrated in `test_evidence` (5.5:1 bypass:fail) and `synthesis` (5.9:1) — mostly docs-only and knowledge work where gates are structurally inapplicable
- `build` is the healthiest gate: 0.7:1 ratio (more failures than bypasses)
- Three "pure noise" gates removed in Phase 7: `agent_running` (∞:1), `model_connection` (71:1), `commit_evidence` (11.8:1)

**Source:** `cmd/orch/complete_cmd.go:SkipConfig`, `getSkipConfig()`, `logSkipEvents()`, `pkg/verify/beads_api.go:ForceCloseIssue()`

### Daemon Verification Integration

The daemon implements a **two-phase design**: it triages (automated gates) but does NOT close issues. Closing still requires `orch complete`.

**Daemon phase (automated):**
1. `ProcessCompletion()` calls `VerifyCompletionFull()` — same 14-gate pipeline as `orch complete`
2. `DetermineEscalationFromCompletion()` — prevents labeling when human approval needed (`EscalationBlock`, `EscalationFailed`)
3. If escalation allows → labels issue `daemon:ready-review` (does NOT close)
4. `VerificationTracker.RecordCompletion()` — increments counter, may trigger pause

**Human phase (manual, via `orch complete`):**
- Explain-back (gate1), behavioral verification (gate2), discovered work disposition, checkpoint enforcement, and liveness check are **CLI-only** — they require interactive/human involvement and cannot be automated
- The full gate pipeline including human gates fires during `orch complete`

**VerificationTracker (review pace governor):**
- `IsPaused()` checked before each spawn — blocks new work when review backlog exceeds threshold (default: 3)
- `SeedFromBacklog()` persists tracker state across daemon restarts
- **Dual signal mechanism:**
  - `~/.orch/daemon-verification.signal` — written by interactive `orch complete` only (NOT headless, NOT orchestrator sessions, NOT dashboard API), triggers `RecordHumanVerification()` (resets counter)
  - `~/.orch/daemon-resume.signal` — written by `orch daemon resume`, triggers manual unpause
- **Periodic resync (fixed 2026-03-26, orch-go-zem67):** When paused, `checkVerificationPause()` calls `ResyncWithBacklog()` with fresh `ListUnverifiedWork()` results. This auto-unpauses when issues close through non-interactive paths (headless, bd close) that don't write verification signals. Without this, the in-memory counter goes stale and the daemon stays paused with nothing to review.
- **Caller discipline (fixed 2026-03-26):** `WriteVerificationSignal()` must only fire when Dylan is the actor. Headless completions, orchestrator sessions, and dashboard API calls are automated paths that would reset the counter without human review. Structural tests in `verification_tracker_test.go` enforce this invariant.

**Checkpoint source-of-truth:**
- Checkpoint file (`~/.orch/verification-checkpoints.jsonl`) is the source of truth for verification state
- `daemon:ready-review` label is the **view layer** only — closed issues lose the label, but their checkpoints persist
- `CountUnverifiedCompletions()` MUST read checkpoint file AND filter to open issues via `verify.ListOpenIssues()`
- All consumers of verification state MUST use `verify.ListUnverifiedWork()` or `verify.CountUnverifiedWork()` — divergent counting logic caused three code paths to disagree on "unverified" (spawn gate checked closed issues, daemon/review checked open issues)

**Source:** `pkg/daemon/daemon.go` (ProcessCompletion, line ~342-380), `pkg/daemon/verification_tracker.go`, `pkg/daemon/issue_adapter.go` (CountUnverifiedCompletions), `pkg/verify/unverified.go`

### Escalation Model

After verification passes, escalation determines human attention level:

| Level | Meaning | When |
|-------|---------|------|
| `EscalationNone` | Auto-complete silently | Simple tasks, all gates pass |
| `EscalationInfo` | Auto-complete, log for review | Knowledge work without recommendations |
| `EscalationReview` | Auto-complete, queue for mandatory review | Knowledge work with recommendations, non-success outcome |
| `EscalationBlock` | DO NOT auto-complete | Visual needs approval, verification failed |
| `EscalationFailed` | Failure state | Verification gates failed |

**Knowledge-producing skills** (investigation, architect, research, design-session, codebase-audit, issue-creation) always surface for at least `EscalationInfo`.

**Source:** `pkg/verify/escalation.go:DetermineEscalation()`, `IsKnowledgeProducingSkill()`

### Activity Feed Persistence

Completed agent activity remains viewable via hybrid persistent layer:
- **Storage:** Proxied from OpenCode's `/session/:sessionID/messages` API
- **Reconciliation:** Historical messages transformed into SSE-compatible events
- **Caching:** Per-session Map cache in dashboard frontend

**Source:** `cmd/orch/serve_agents.go:handleSessionMessages()`

### Progressive Handoff Updates

`orch complete` triggers interactive prompts for active orchestrator sessions:
1. Standard verification gates run for the worker agent
2. Orchestrator prompted for worker's outcome and key finding
3. Outcome auto-inserted into SESSION_HANDOFF.md Spawns table
4. Beads issue closed only after handoff update (or skip)

**Source:** `cmd/orch/session.go:UpdateHandoffAfterComplete()`

---

## Why This Fails

### 1. Evidence Gate False Positive (Adversarial Agent)

**What happens:** Agent writes "go test ./... - PASS (47 tests)" in a beads comment without actually running tests.

**Root cause:** Test evidence gate checks for evidence *patterns* in comments, not actual test execution. The only gate that executes something real is Build (`go build ./...`).

**Why detection is hard:** The anti-theater patterns catch vague claims ("tests pass") but cannot distinguish fabricated framework-specific output from real output.

**Mitigation:** V3 level adds human behavioral observation. For V2, the Build gate catches compilation failures but not test failures. Future: actually run tests as a gate.

### 2. Visual Verification Evidence Without Approval

**What happens:** Agent passes visual evidence gate by writing "screenshot captured" without actual screenshot. Human approval gate not triggered because risk assessment classified changes as Low.

**Root cause:** Risk assessment heuristics (CSS-only ≤10 lines → Low) can misclassify impactful visual changes.

**Fix:** Override with `--verify-level V3` for known-sensitive UI work. Skill manifest `requires_ui_approval: true` (future).

### 3. Cross-Project Verification Wrong Directory

**What happens:** Verification runs in wrong directory, checks wrong tests, reports false failure.

**Root cause:** SPAWN_CONTEXT.md missing PROJECT_DIR, fallback uses workspace location (orch-go), but agent worked in a different repo.

**Fix:** `orch spawn --workdir` explicitly sets PROJECT_DIR in SPAWN_CONTEXT.md. Verification reads it. Make --workdir mandatory for cross-project spawns.

**Source:** Cross-project logic integrated into `pkg/verify/check.go`

### 4. Coaching Plugin Coverage Gap

**What happens:** Behavioral monitoring (coaching plugin) only works for OpenCode API spawns. Claude CLI/tmux spawns (the "escape hatch" for critical work) have NO behavioral monitoring.

**Implication:** Critical infrastructure work — exactly when monitoring matters most — runs unmonitored.

### 5. Light Tier / V2 Verification Level Conflict (Active)

**What happens:** `feature-impl` defaults to `TierLight` (spawn context says "SYNTHESIS.md NOT required") but also to `VerifyV2` (which includes GateSynthesis). Every `feature-impl` completion fails the synthesis gate because the agent was instructed not to produce the file that verification demands.

**Empirical data:** 3/3 feature-impl completions on Feb 28, 2026 failed the synthesis gate. 100% failure rate for the most common spawn type. Agents followed instructions correctly — the contradiction is in the system.

**Root cause:** The V0-V3 level system was designed to replace the tier system, but the migration was incomplete. SPAWN_CONTEXT template still uses tier for SYNTHESIS.md instructions; `pkg/verify/check.go:550` skips tier-check for synthesis (the legacy path checked `tier != "light"`, the modern level-based path does not). The `GateArchitecturalChoices` gate at line 388 already has `&& tier != "light"` suppression — showing someone hit this class of problem before but didn't generalize the fix.

**Workaround:** Orchestrator must use `--skip-synthesis` for all feature-impl completions. This violates the "zero skip flags for well-configured spawns" design goal.

**Fix options:** (1) Add `tier != "light"` suppression to synthesis gate in level-based path. (2) Align feature-impl default tier to `TierFull` since V2 requires synthesis.

### 6. Hotspot System Blind Spots (Known)

**What happens:** The hotspot system (bloat-size, fix-density, investigation-cluster, coupling-cluster) misses five categories of structural problems. Three are causing real pain today:

1. **Implicit coupling** (CRITICAL): String-literal protocols spanning packages without shared constants — 10 independent skill-name maps across 7 packages; status string filtering gap causing blocked agents to be invisible on dashboard; `synthesis_parser.go` regex `\w+` silently drops "spawn-follow-up" recommendations because hyphens break word-character matching.

2. **Semantic complexity** (HIGH): Files under 800 lines with high goroutine/channel/mutex density (`swarm.go`, `capacity/manager.go`, `serve_agents_cache.go`, `spawn/resolve.go`). Hotspot sees these as small, well-factored files.

3. **Dead features** (HIGH): ~7,900 lines confirmed dead code including `pkg/capacity/` (717 lines), `pkg/shell/` (972 lines), `pkg/certs/` (private key in source control), a 21MB compiled binary tracked in git, and `.bak2` backup files.

4. **Cold spots** (MEDIUM): Frozen infrastructure that may silently degrade (opencode SSE subsystem, `pkg/state/reconcile.go`).

5. **Scattered duplication** (MEDIUM): Beads RPC/CLI fallback pattern hand-written 58 times across 24 files with subtle behavioral inconsistencies.

**Existing bugs found through this analysis:** Blocked agents invisible on dashboard (serve_agents_discovery.go filters missing "blocked"); spawn-follow-up recommendations silently dropped (synthesis_parser.go regex); `beads.DefaultDir` not defer-restored in complete_pipeline.go; private key tracked in source control.

### 7. Human Negative-Feedback Channel Structural Disuse

**What happens:** `orch rework` exists (cmd/orch/rework_cmd.go, 356 lines) and `agent.reworked` events are defined — but in 1,285+ completions, there have been **0 reworks**, **0 agent.rejected events**, and only **11 abandons** (all operational, none quality-based). The human feedback channel is structurally unused.

**Root cause — friction asymmetry:** Rework requires 3 mandatory inputs (beads-id, feedback text, `--bypass-triage`) plus 5 blocking preconditions (closed issue, archived workspace, worker skill only, preflight checks, full context gathering). Compare to `orch work <issue-id>` (re-spawn): 1 input, 0 preconditions. Rational actors always re-spawn rather than rework.

**Root cause — vocabulary gap (partially addressed):** `orch reject` was built (March 2026) as a 1-step negative feedback verb matching friction of `orch complete`. It emits `agent.rejected` events and reopens issues. However, as of March 22 2026, **0 reject events exist in production** — the verb exists but has never been used by human or machine.

**Root cause — auto-completion bypass:** Daemon auto-completes 37% of agents (406/1,102) via `orch complete --force`, which bypasses all interactive gates. These completions enter the learning loop indistinguishable from human-verified work.

**Root cause — consumer-side gap:** Even if rejections were generated, the daemon does not consume `RejectedCount` from the learning store. Only `SuccessRate` affects daemon allocation (±20% priority modulation via allocation.go:107-119). RejectedCount, ReworkCount, VerificationFailures, and VerificationBypasses are all write-only metrics in learning.go — aggregated but never read by any behavioral code path.

**Implication:** The learning loop has no negative signal at any layer. The vocabulary gap is partially addressed (reject verb exists) but the structural gap persists: (1) reject is never triggered automatically, (2) even if triggered, daemon doesn't consume rejection data, (3) the audit system that would generate automatic rejections has a disconnected middle (see §8).

**Source:** Probe `probes/2026-03-20-probe-human-feedback-channel-structural-disuse.md`, probe `probes/2026-03-22-probe-open-loop-infrastructure-code-audit.md`

### 8. Split Commit — Callers Committed Without Implementations

**What happens:** An agent commits files that call new functions/use new types, but does NOT commit the files that define those functions/types. The committed code does not compile. The issue is closed anyway.

**Concrete instance (2026-03-26):** Commit `da9b666b4` (orch-go-r7avo) modified `ooda.go` and `preview.go` to call `RouteModel()` and a 4-arg `RouteIssueForSpawn()`, but the implementations (in `skill_inference.go`, `coordination.go`) were left in the dirty working tree. Build errors: `undefined: RouteModel`, `too many arguments`. Issue was marked closed. Build was broken for an unknown duration.

**Root cause:** The completion pipeline's Build gate (#10) runs `go build` on the *working tree*, not on the *committed code*. When an agent has staged some files and left others dirty, `go build` passes (because all files are present in the working tree) but the committed state is broken. The Build gate verifies workability, not commit integrity.

**Why this wasn't caught:** (1) No CI/CD runs on commits. (2) The Build gate runs against the working tree, not `git stash && go build`. (3) `orch complete` for orch-go-r7avo passed all gates because the agent's working tree compiled fine. (4) Pre-commit hook also builds the working tree, not the staged set.

**Proposed fix:** Either (a) pre-commit hook should `git stash -k` (stash unstaged) before building, or (b) completion verification should check `git diff --name-only HEAD` for compilation-breaking asymmetry (callers committed, implementations not).

**Source:** Probe `probes/2026-03-26-probe-dirty-worktree-closed-issue-reconciliation.md`

### 9. Consumer-Last Construction (Systemic Open Loops)

**What happens:** The completion/daemon pipeline contains **10 concrete open loops** where infrastructure is built on the emission side (events logged, labels added, config fields defined, scheduler tasks registered) but the consumption side (daemon reads signal and changes behavior) is missing. The pipe is built on both ends but not connected in the middle.

**Instances (code-verified, Mar 22 2026):**

| # | Loop | Emission | Missing Consumer |
|---|------|----------|-----------------|
| 1 | Quality audit | periodic_audit.go labels issues `audit:deep-review` | No code spawns audit agents for labeled issues |
| 2 | Accretion response | 513 accretion.delta events emitted | daemon_loop.go:141 has blank wiring (comment only) |
| 3 | Reject → learning | reject_cmd.go emits agent.rejected | RejectedCount never read by daemon |
| 4 | Comprehension queue | comprehension:pending labels added | ComprehensionQuerier never instantiated (always nil) |
| 5 | Verification metrics | VerificationFailures/Bypasses aggregated | Both fields are dead code (never read) |
| 6 | Rework feedback | ReworkCount aggregated in learning.go | Display-only in orient, no daemon routing |
| 7-10 | 11 periodic tasks | Registered in scheduler.go, config fields defined | Never called from daemon_periodic.go |

**Root cause — consumer-last construction:** The system consistently builds infrastructure in this order: (1) emit events/labels, (2) define config/scheduler registration, (3) never return to build the consumer. This is not a testing gap (Phase 3 of a plan) — the consumer code was never written. Each instance follows the same pattern: emitters are built during feature work, consumers are deferred indefinitely.

**Structural evidence:** 24 tasks registered in pkg/daemon/scheduler.go (lines 125-148); only 13 invoked from cmd/orch/daemon_periodic.go. 11 tasks have config fields, scheduler registration, and sometimes implementation functions — but no call site from the main loop. ~2,800 lines of structurally unreachable code.

**Production evidence (as of Mar 22 2026):** 1,285 completions, 513 accretion.delta events, 0 agent.rejected events, 0 audit verdicts, 0 reworks. Every feedback channel except SuccessRate is structurally unused.

**Source:** Probe `probes/2026-03-22-probe-open-loop-infrastructure-code-audit.md`

### 10. No Ownership Reconciliation — Closed Issues Leave Unowned Dirt

**What happens:** An agent closes an issue while tracked dirty files remain in the working tree. Those files are not committed, not transferred to another issue, and not classified as allowed residue. The next agent inherits an opaque dirty worktree.

**Concrete instance (2026-03-26):** 33+ tracked dirty source/docs files across 5 clusters. Cluster 1 (capability routing: `skill_inference.go`, `coordination.go`, `allocation.go`) belongs to closed issue orch-go-r7avo — uncommitted implementations that break the build. Cluster 2 (backend verification removal) belongs to closed issue orch-go-8l4h9. Neither cluster was committed during completion.

**Root cause:** The completion pipeline has no gate that checks "are there tracked dirty files from this agent's work that are unowned?" The 14 existing gates verify the agent's claims (SYNTHESIS, Phase: Complete, test evidence) and the agent's committed work (git_diff, build). None verify what the agent left *uncommitted*.

**Why this matters more than cleanliness:** The invariant should not be "clean worktree" — 99.7% of dirty entries (7,294 of 7,296) are harmless historical `.orch/workspace/` artifacts. The invariant should be "every tracked dirty file is owned by an open issue or belongs to an allowed artifact class." This is a binary check (owned/unowned) that avoids the accretion-gate bypass problem (continuous invariants have 100% bypass rate).

**Proposed fix:** New Gate 15 (`ownership_reconciliation`) at V2+ level. At completion, compare tracked dirty files against: (a) agent's git baseline (ignore pre-existing dirt), (b) artifact class registry (allow local-state, knowledge-backlog), (c) beads issue ownership (allow files owned by other open issues). Fail if unowned post-baseline dirty files remain.

**Source:** Probe `probes/2026-03-26-probe-ownership-based-harness-design-evaluation.md`, design `.kb/investigations/2026-03-26-design-ownership-based-harness-prevention.md`

---

## Constraints

### Why 14 Gates Instead of 1?

**Constraint:** Verification checks 14 independent aspects of "done."

**Implication:** Each gate can fail independently. Failure diagnostics are precise (which gate failed, why).

**This enables:** Targeted bypass per gate, level-based gate selection, data-driven improvement via per-gate metrics
**This constrains:** Cannot simplify to single pass/fail without losing diagnostics

### Why Levels Over Gates as Primary Concept?

**Constraint:** The 14 gates exist but are organized into 4 levels (V0–V3).

**Implication:** Orchestrators think in levels ("this needs V2 verification"), not individual gates.

**This enables:** Zero-flag completions for well-configured spawns, shared vocabulary between human and orchestrator
**This constrains:** Auto-skip logic scattered across 6 files should be consolidated into level-based routing

### Why Build Is Unconditional?

**Constraint:** The Build gate fires for any completion that changed Go files, regardless of verification level.

**Implication:** Even V0 (Acknowledge) runs Build if Go files changed. A broken build should always be caught.

**This enables:** The only unfakeable verification signal (actually executes `go build ./...`)
**This constrains:** Cannot skip Build via verification levels, only via explicit `--skip-build`

### Why Accretion Thresholds Are Language-Agnostic?

**Constraint:** The accretion gate uses uniform 800/1500 line thresholds across all language types.

**Implication:** Both Go and TypeScript files follow similar size distributions — the thresholds correctly flag structural bloat in both languages. Language-specific thresholds are over-engineering for a portfolio that is 95%+ Go/TypeScript.

**Exception:** Generated files (e.g., `*.gen.ts`, `*.pb.go`) must be excluded via pattern matching, not language-specific thresholds. The bloat scanner excludes 13 build output directories (`skipBloatDirs`: `.git`, `node_modules`, `vendor`, `.svelte-kit`, `dist`, `build`, `.opencode`, `.orch`, `.beads`, etc.) and 12 path prefixes. False positives from build artifacts (e.g., `.opencode/plugin/coaching.ts` appearing as CRITICAL) were fixed by expanding this exclusion list.

**Source:** `cmd/orch/hotspot.go:skipBloatDirs`, probe `probes/2026-02-14-language-agnostic-accretion-metrics.md`

### Why No Agent-Judgment Gates?

**Constraint:** The pipeline contains execution, evidence, and judgment gates — but no agent-judgment gates (e.g., AI code review).

**Implication:** Agent reviewing agent code is a closed loop — same model family, same blind spots, no provenance chain. The output is opinion, not evidence.

**Cross-model nuance (Mar 2026):** Cross-model review demonstrably escapes the blind spot loop — a 6-model benchmark showed Codex/DeepSeek found a backend root cause that Opus/Sonnet/GPT-5.2/Gemini missed. However, the provenance objection still applies: cross-model opinions are still opinions, not executable evidence. The correct application is cross-model review as advisory signal (like hotspot warnings), not as a blocking gate. Skill-frame divergence within the same model family also matters: investigation-framed Opus finds bugs that debugging-framed Opus misses, suggesting `orch spawn investigation "review X"` captures some cross-frame value today.

**This enables:** Clean separation between machine verification (execution), structural checks (evidence), and human responsibility (judgment)
**This constrains:** Cannot add AI code review or similar agent-opinion gates without violating provenance. The fix for "nobody reads the diff" is expanding execution-based gates (go vet, staticcheck), not adding agent judgment.

**Source:** Decision `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md`, probe `probes/2026-03-01-probe-cross-model-blind-spots.md`

### Why Knowledge Work Surfaces, Not Auto-Closes?

**Constraint:** Investigation/architect/research agents surface for review even if all gates pass.

**Implication:** Cannot batch-close knowledge work overnight. Orchestrator must review next session.

**This enables:** Knowledge synthesis opportunity, findings integration into mental model
**This constrains:** Cannot batch-close knowledge work without orchestrator review

---

## Verification Spectrum

| Verification Level | What's Checked | What Could Be Gamed |
|---|---|---|
| Agent claims completion | Strong (Phase gate + regex parsing) | Agent could write "Phase: Complete" prematurely |
| Artifacts exist | Strong (file existence checks) | Agent could create empty/placeholder files |
| Tests were run | Strong (anti-theater patterns reject vague claims) | Agent could fabricate framework-specific output |
| Binary compiles | Strong (actually runs `go build`) | Cannot be faked |
| Tests pass | Partial (checks for evidence in comments, doesn't RUN tests) | Agent could write passing output without running |
| Smoke test | Missing | No automated smoke/integration test execution |
| Live e2e | Missing | Checks for Playwright evidence but doesn't execute |
| Adversarial | Missing | No verification against intentionally deceptive agents |

**Key insight:** The system is strong at "does it exist?" verification (evidence-based gates) but weak at "did it actually execute?" verification. Only 1 of 14 gates is execution-based (`go build`). Expanding execution-based gates (go vet, staticcheck, actually running tests) would increase the unfakeable verification surface without adding agent-judgment complexity.

**Visual evidence:** The visual verification gate detects Playwright-based browser tool patterns only. Glass browser automation patterns were removed (Feb 2026) — they were dead code with no functional impact. `visualEvidencePatterns` in `visual.go` now contains Playwright patterns plus generic patterns ("verified in browser", "screenshot", etc.).

---

## Evolution

### Phase 1: Basic Verification (Dec 2025)
Phase gate only. Check for "Phase: Complete" comment, close beads issue. **Gap:** No evidence checking, auto-closed everything.

### Phase 2: Evidence Gate (Dec 26-28, 2025)
Added evidence gate for visual/test proof. **Key insight:** Agents claim "tested, works" without doing it.

### Phase 3: Approval Gate (Dec 29-31, 2025)
Added human approval for UI changes. **Key insight:** Even with evidence, agents can attach wrong screenshot.

### Phase 4: 5-Tier Escalation (Jan 2-4, 2026)
Knowledge-producing work surfaces for review instead of auto-closing. **Key insight:** Auto-closing knowledge work means findings never get synthesized.

### Phase 5: Cross-Project Verification (Jan 5-7, 2026)
Detection of project directory from SPAWN_CONTEXT.md. **Key insight:** Workspace location ≠ work location.

### Phase 6: Targeted Bypasses (Jan 14, 2026)
Replaced blanket `--force` with targeted `--skip-{gate}` flags. 55% of completions had used `--force` due to false positives.

### Phase 7: Pure-Noise Gate Removal (Feb 2026)
Removed 3 gates identified as pure noise through friction analysis of 1,008 bypass events:
- `agent_running` (∞:1 bypass:fail ratio, 183 bypasses, 0 failures)
- `model_connection` (71:1 ratio)
- `commit_evidence` (11.8:1 ratio, redundant with git_diff gate)

### Phase 8: Verification Levels Design (Feb 20, 2026)
Unified three implicit level systems (spawn tier, checkpoint tier, skill-based auto-skips) into four explicit verification levels (V0–V3). Design complete, implementation pending.

### Phase 9: Gate Type Taxonomy (Feb 25, 2026)
Classified all 14 gates into three types (execution/evidence/judgment) based on what kind of verification they provide. Established that agent-judgment gates are excluded by design — the fix for verification gaps is expanding execution-based gates, not adding agent opinion. **Key insight:** The "sharp boundary at execution" (only Build actually runs code) is real and is the right place to expand.

### Phase 10: Accretion Gate Nuance (Feb 24, 2026)
Added pre-existing bloat awareness to accretion gate: when a file was already over threshold before the agent's changes, gate downgrades from ERROR to WARNING. Agents are not penalized for pre-existing code debt they didn't create. Also added `--skip-accretion` flag. Also fixed skip propagation gap: `--skip-phase-complete` now triggers `bd close --force` to bypass `bd`'s independent Phase: Complete check.

### Phase 11: Light Tier / V2 Conflict Identified (Feb 27-28, 2026)
Empirically confirmed that `feature-impl` spawned with V2 + TierLight creates contradictory instructions: agent told SYNTHESIS.md "NOT required" at spawn time, then blocked at completion because V2 includes GateSynthesis. 100% failure rate on Feb 28 (3/3 completions). Root cause: incomplete migration from tier system to level system. Unresolved — fix decision pending.

### Phase 12: Ownership Reconciliation Design (Mar 26, 2026)
Designed Gate 15 (`ownership_reconciliation`) to close the gap between agent completion and worktree state. Key reframe: the invariant is not "clean worktree" but "every tracked dirty file is owned." Close-time enforcement at V2+ level, using agent's git baseline to exclude pre-existing dirt. Supporting changes: artifact class registry, skill text alignment (remove contradictory `git add -A` from feature-impl), build gate hardening (build against committed state, not working tree). Design complete — implementation pending.

---

## References

**Guide:**
- `.kb/guides/completion.md` — Procedural guide (commands, workflows, troubleshooting)
- `.kb/guides/completion-gates.md` — Gate-specific reference

**Investigations:**
- `.kb/investigations/2026-02-20-audit-verification-infrastructure-end-to-end.md` — 14-gate inventory (authoritative)
- `.kb/investigations/2026-02-20-inv-architect-verification-levels.md` — V0-V3 levels design
- `.kb/investigations/2026-02-25-design-code-review-gate-for-completion-pipeline.md` — Gate type taxonomy origin, code review analysis
- `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` — Verification levels decision record
- `.kb/decisions/2026-02-25-no-code-review-gate-expand-execution-verification.md` — No agent-judgment gates, expand execution instead

**Probes:**

*Previously referenced:*
- `probes/2026-02-20-probe-verification-infrastructure-audit.md` — Full infrastructure audit (source of 14-gate model)
- `probes/2026-02-20-probe-verification-levels-design.md` — Levels design probe (source of V0-V3 model)

*Merged 2026-03-06 — all 25 probes:*

| Probe | Verdict | 1-Line Summary |
|-------|---------|----------------|
| `2026-02-09-friction-bypass-analysis-post-targeted-skips.md` | extends | `--force` dropped 72.8% → 16.7%; bypass noise concentrated in test_evidence/synthesis (docs-only friction) |
| `2026-02-13-friction-gate-inventory-all-subsystems.md` | extends | 48 gates across 3 subsystems; only build/git_diff have healthy bypass:fail ratio; three noise gates removed |
| `2026-02-14-language-agnostic-accretion-metrics.md` | confirms | 800/1500 thresholds are language-agnostic; generated file exclusion needed; cross-project aggregation requires project_dir field |
| `2026-02-15-daemon-verification-tracker-checkpoint-enforcement.md` | contradicts (fixed) | Daemon was reading labels instead of checkpoints — closed issues lost label, never counted as unverified; fixed via checkpoint-first counting |
| `2026-02-15-verifiability-first-closure-audit.md` | confirms | All 4 verifiability-first issues have functional work in codebase; enforcement theater resolves via iterative fixing |
| `2026-02-15-verification-tracker-wiring.md` | extends | VerificationTracker was unwired (zero calls); fixed with dual signal mechanism (verification signal + resume signal) |
| `2026-02-15-verificationtracker-backlog-count-mismatch.md` | extends | Daemon counted closed-issue checkpoints; orch review filtered to open-only; fixed by using ListOpenIssues() for filtering |
| `2026-02-16-daemon-completion-loop-bypasses-verification-gates.md` | extends | Daemon is a two-phase triage layer, not a closing layer; 6 gates are CLI-only by design; VerificationTracker governs review pace, not gate bypass |
| `2026-02-16-probe-three-code-paths-verification-state.md` | extends | Three code paths (spawn gate, daemon, review) used incompatible definitions of "unverified"; canonical source `verify.ListUnverifiedWork()` created |
| `2026-02-17-rework-loop-design-for-verification-gaps.md` | extends | No EscalationRework path exists; Block/Failed is a dead-end requiring manual re-spawn with lost context |
| `2026-02-18-probe-entropy-spiral-fix-commit-relevance.md` | extends | 161 fix commits in entropy-spiral branch; 3 still apply cleanly to master; 158 irrelevant due to code divergence |
| `2026-02-19-probe-coupling-hotspot-detection-gap.md` | extends | Accretion enforcement is size-only, blind to coupling; coupling-cluster is orthogonal failure mode (25-file daemon cluster scores 180, CRITICAL) |
| `2026-02-19-probe-accretion-enforcement-gap-analysis.md` | contradicts+extends | Spawn gate blocking is NOT implemented (warning-only, result discarded); CLAUDE.md claim "Spawn gates block feature-impl on CRITICAL files" was aspirational; layers 2-4 fully shipped |
| `2026-02-19-probe-glass-removal-verification.md` | confirms | Glass visual evidence patterns removed; Playwright patterns are sole visual detection mechanism; no functional gap |
| `2026-02-20-probe-verification-infrastructure-audit.md` | contradicts+extends | Prior "3-gate" model contradicted; 14 gates fully wired; coaching plugin only works for OpenCode spawns (not tmux) |
| `2026-02-20-probe-verification-levels-design.md` | contradicts+extends | "All gates fire" claim wrong; gates are level-selective; auto-skip logic scattered across 6 files; build is the only truly unconditional gate |
| `2026-02-24-probe-double-gate-skip-phase-complete-propagation.md` | extends | --skip-phase-complete must propagate to bd close --force; pipeline leak fixed via ForceCloseIssue() |
| `2026-02-24-probe-accretion-gate-preexisting-bloat-skip.md` | extends | Pre-existing bloat (file already >1500 before agent) downgrades to WARNING; agent-caused threshold crossing keeps ERROR |
| `2026-02-25-probe-coupling-cluster-implementation-review.md` | confirms | Coupling-cluster implementation stays within accretion bounds; spawn gates get coupling awareness for free via existing Hotspot type |
| `2026-02-25-probe-hotspot-bloat-scanner-build-output-exclusions.md` | extends | Bloat scanner had only 3 directory exclusions; expanded to 13 directories + 12 path prefixes; false positives from .opencode/plugin and .svelte-kit eliminated |
| `2026-02-25-probe-code-review-gate-design.md` | confirms | Agent code review is a closed loop (same blind spots, no provenance); correct fix is expanding execution gates (go vet, staticcheck); code review would be judgment-based, not execution-based |
| `2026-02-27-probe-light-tier-v2-verification-conflict.md` | contradicts | "Zero skip flags for well-configured spawns" invariant violated; feature-impl=TierLight + V2 creates contradictory synthesis instructions; incomplete tier→level migration |
| `2026-02-28-probe-synthesis-gate-light-tier-empirical-failures.md` | extends | 100% feature-impl failure rate on Feb 28 (3/3); agents correctly followed instructions but verification demanded missing SYNTHESIS.md |
| `2026-03-01-probe-hotspot-blind-spot-analysis.md` | extends | 5 blind spot categories with codebase evidence; 4 existing bugs found including private key in git and blocked agents invisible on dashboard |
| `2026-03-01-probe-cross-model-blind-spots.md` | extends | Cross-model review escapes blind spot loop (6-model benchmark: Codex/DeepSeek found backend root cause that Opus/Sonnet missed); skill-frame divergence within same model also produces different blind spots; provenance objection still applies |
| `2026-03-26-probe-dirty-worktree-closed-issue-reconciliation.md` | extends | Split-commit failure mode: agent committed callers (ooda.go) without implementations (skill_inference.go), broke build. Build gate runs on working tree not committed code — cannot detect this. New "Why This Fails" §8. |

**Models:**
- `.kb/models/completion-lifecycle.md` — Where completion fits in agent lifecycle
- `.kb/models/orchestrator-session-lifecycle/model.md` — How orchestrator completion differs
- `.kb/models/spawn-architecture/model.md` — How SPAWN_CONTEXT.md sets PROJECT_DIR

**Source code:**
- `pkg/verify/check.go` — Main verification entry point, 14 gate constants, `VerifyCompletionFull()`, tier routing
- `pkg/verify/beads_api.go` — Phase comment parsing (`ParsePhaseFromComments()`), beads integration
- `pkg/verify/test_evidence.go` — Test evidence detection with anti-theater patterns
- `pkg/verify/visual.go` — Visual verification with risk assessment and approval patterns
- `pkg/verify/escalation.go` — 5-tier escalation model, `IsKnowledgeProducingSkill()`
- `pkg/checkpoint/checkpoint.go` — Checkpoint tier enforcement (gate1/gate2 by issue type)
- `cmd/orch/complete_cmd.go` — Complete command orchestration, SkipConfig, CLI flags, 24-step completion flow
- `cmd/orch/complete_pipeline.go` — Pipeline phase functions
- `cmd/orch/complete_verify.go` — SkipConfig integration
- `pkg/daemon/daemon.go` — Daemon verification integration (`ProcessCompletion()`)
- `pkg/daemon/verification_tracker.go` — Threshold-based pause for daemon auto-completion
- `pkg/daemon/issue_adapter.go` — `CountUnverifiedCompletions()`, checkpoint-first counting with open-issue filter
- `pkg/verify/unverified.go` — Canonical `ListUnverifiedWork()` / `CountUnverifiedWork()` — all consumers must use this
- `pkg/verify/beads_api.go` — `CloseIssue()`, `ForceCloseIssue()` — propagates phase skip to bd close --force
- `pkg/verify/accretion.go` — Accretion gate with pre-existing bloat detection and net-negative bypass
- `cmd/orch/hotspot.go` — Bloat scanner with 13-directory exclusion list; `skipBloatDirs`, `buildOutputPrefixes`
- `cmd/orch/hotspot_coupling.go` — Coupling-cluster analysis (4th hotspot type, 389 lines, standalone file)

## Probes

- 2026-03-20: Human Feedback Channel Structural Disuse — 0 reworks, 11 operational abandons in 1,102 completions. Friction asymmetry (rework=8 steps, complete=0 steps) creates false ground truth. §7 updated.
- 2026-03-20: Daemon-Driven Random Quality Audit Design — 3-layer structural pipeline: daemon periodic audit selection (weighted toward auto-completed work), spawned audit agent for intent/test/quality review, verdict-to-reject pipeline feeding `agent.rejected` events into learning loop. Key gap found: `learning.go` missing `RejectedCount` field and `agent.rejected` handler — learning loop structurally blind to rejections even after `orch reject` ships. See `.kb/investigations/2026-03-20-inv-design-daemon-driven-random-quality-audit.md`.
- 2026-03-22: Open-Loop Infrastructure Code Audit — 10 concrete open loops found where emission infrastructure exists but consumer/action layer is missing. Pattern named "consumer-last construction": system builds emitters + config first, never returns to build consumer. 513 accretion.delta events with no reader, 11 periodic tasks registered but never invoked, ComprehensionQuerier always nil, RejectedCount/VerificationFailures/VerificationBypasses are dead code. §7 updated (reject exists but 0 production events), §8 added.
- 2026-03-26: Ownership-Based Harness Design Evaluation — Close-time reconciliation gate (Gate 15) designed. Key finding: bypass resistance correlates with invariant type (binary > continuous); ownership is binary, accretion is continuous. Dirty-worktree is not a new defect class but a composition of Class 3 (stale accumulation) + Class 5 (contradictory authority) + Class 0 (scope expansion). Contradictory skill text (`git add -A` in feature-impl vs NEVER in worker-base) is textbook Class 5. New "Why This Fails" §10, Evolution §Phase 12.
- 2026-03-26: WriteVerificationSignal called from non-human paths — `WriteVerificationSignal()` was called from 3 paths (dashboard API single close, batch close, `orch complete`), only one of which is human-initiated. Automated callers reset `completionsSinceVerification` to 0, preventing the daemon from ever reaching its pause threshold. Fixed by removing calls from dashboard API and gating `complete_lifecycle.go` on `!completeHeadless && !target.IsOrchestratorSession`. Structural tests added.

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-03-inv-recover-priority-verification-gates-git.md
- .kb/investigations/archived/2026-01-04-inv-phase-completion-verification-orchestrator-spawns.md
- .kb/investigations/2026-02-14-inv-remove-pure-noise-completion-gates.md
- .kb/investigations/archived/2026-01-10-inv-phase-2-completion-verification.md
- .kb/investigations/archived/2025-12-27-inv-implement-cross-project-completion-adding.md
- .kb/investigations/archived/2026-01-17-inv-enhance-agent-reporting-verification-gates.md
- .kb/investigations/archived/2025-12-23-inv-implement-phase-gates-verification-orch.md
