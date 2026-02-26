# Model: Completion Verification Architecture

**Domain:** Completion / Verification / Quality Gates
**Last Updated:** 2026-02-20
**Synthesized From:** 31 investigations, 14 probes, completion.md guide, end-to-end infrastructure audit (Feb 20, 2026)

---

## Summary (30 seconds)

Completion verification operates through **14 gates** organized into **4 verification levels** (V0–V3). Each level is a strict superset of the one below: V0 (Acknowledge) checks only that the agent reported completion; V1 (Artifacts) adds deliverable and constraint checks; V2 (Evidence) adds test evidence, build, and git diff checks; V3 (Behavioral) adds visual verification and human observation gates. The verification level is determined at spawn time from skill type and issue type, stored in AGENT_MANIFEST.json, and flows through to `orch complete`. **Targeted bypasses** (`--skip-{gate} "reason"`) remain as an escape hatch for edge cases, but well-configured spawns should require zero skip flags. The daemon runs the same `VerifyCompletionFull()` pipeline with threshold-based pause to prevent unchecked auto-completion.

---

## Core Mechanism

### The 14 Gates

| # | Gate | Constant | What It Checks |
|---|------|----------|----------------|
| 1 | Phase Complete | `phase_complete` | Agent reported "Phase: Complete" via beads comment |
| 2 | Synthesis | `synthesis` | SYNTHESIS.md exists and is non-empty (skipped for light tier and knowledge-producing skills) |
| 3 | Handoff Content | `handoff_content` | SESSION_HANDOFF.md has TLDR & Outcome filled (orchestrator tier only) |
| 4 | Skill Output | `skill_output` | Required skill outputs exist (from skill.yaml `outputs.required`) |
| 5 | Phase Gates | `phase_gate` | Required skill phases were reported in beads comments |
| 6 | Constraint | `constraint` | Constraint patterns from SPAWN_CONTEXT match actual files |
| 7 | Decision Patch Limit | `decision_patch_limit` | Decision patch count within limits |
| 8 | Test Evidence | `test_evidence` | Evidence of actual test execution in beads comments (anti-theater detection) |
| 9 | Git Diff | `git_diff` | Git changes match SYNTHESIS.md claims |
| 10 | Build | `build` | Project compiles (`go build ./...`) — the only unfakeable gate |
| 11 | Accretion | `accretion` | File size growth within limits (accretion boundary enforcement) |
| 12 | Visual Verification | `visual_verification` | Screenshot/Playwright evidence for web/ changes, with risk assessment |
| 13 | Explain-Back | `explain_back` | Orchestrator explains what was built and why (gate1/comprehension) |
| 14 | Behavioral | `behavioral` | Human confirms behavior was observed working (gate2, V3 only) |

**Key design property:** Gates are structurally independent but functionally level-selective. Each gate can fail independently, and all applicable gates must pass (or be explicitly skipped). The verification level determines which subset of gates fires.

**Source:** `pkg/verify/check.go` — constants at top, `VerifyCompletionFull()` orchestrates all gates

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

**Source:** `cmd/orch/complete_cmd.go:SkipConfig`, `getSkipConfig()`, `logSkipEvents()`

### Daemon Verification Integration

The daemon runs the same verification pipeline as manual `orch complete`:

1. `ProcessCompletion()` calls `VerifyCompletionFull()` before marking anything
2. `VerificationTracker.IsPaused()` checks threshold-based pause (default: 3 failures before pause)
3. `SeedFromBacklog()` persists tracker state across daemon restarts
4. Signal file `~/.orch/daemon-verification.signal` bridges human verification to daemon awareness

**Source:** `pkg/daemon/daemon.go` (ProcessCompletion, line ~342-380), `pkg/daemon/verification_tracker.go`

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

**Key insight:** The system is strong at "does it exist?" verification but weak at "did it actually execute?" verification. The only unfakeable signal is `go build`.

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

---

## References

**Guide:**
- `.kb/guides/completion.md` — Procedural guide (commands, workflows, troubleshooting)
- `.kb/guides/completion-gates.md` — Gate-specific reference

**Investigations:**
- `.kb/investigations/2026-02-20-audit-verification-infrastructure-end-to-end.md` — 14-gate inventory (authoritative)
- `.kb/investigations/2026-02-20-inv-architect-verification-levels.md` — V0-V3 levels design
- `.kb/decisions/2026-02-20-verification-levels-v0-v3.md` — Verification levels decision record

**Probes:**
- `probes/2026-02-20-probe-verification-infrastructure-audit.md` — Full infrastructure audit
- `probes/2026-02-20-probe-verification-levels-design.md` — Levels design probe

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
