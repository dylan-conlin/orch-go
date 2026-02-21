# Probe: Verification Infrastructure End-to-End Audit

**Model:** completion-verification
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The model claims verification operates through "three independent gates" (Phase, Evidence, Approval) and references specific source files (`pkg/verify/phase.go`, `pkg/verify/evidence.go`, `pkg/verify/cross_project.go`). Does the current codebase match these claims? Is the model's understanding of verification gates current?

Additionally: Are all verification features wired into real flows, or is some of it enforcement theater (code exists but never fires in production)?

---

## What I Tested

Full architectural audit of all verification code across four parallel investigations:

1. **pkg/verify/ inventory** - Read all 43 files, cataloged 100+ exported functions, traced callers
2. **Completion pipeline** - Read complete_cmd.go, complete_pipeline.go, complete_verify.go, traced execution flow
3. **Daemon verification** - Read daemon.go, completion_processing.go, verification_tracker.go, traced auto-close flow
4. **Test evidence detection** - Read test_evidence.go, analyzed 22 true-positive and 11 false-positive patterns, verified caller chain

```bash
# Key verification commands
grep -r "pkg/verify" cmd/orch/ --include="*.go" -l  # Find all callers
grep -r "VerifyCompletionFull\|VerifyTestEvidence\|IsPaused" cmd/orch/ pkg/daemon/ --include="*.go"
find pkg/verify/ -name "*_test.go" | wc -l  # Count test files
```

---

## What I Observed

### Model Staleness: Critical

The model (last updated 2026-01-14) has significant drift from current code:

**Deleted files referenced by model:**
- `pkg/verify/phase.go` - No longer exists. Phase checking is in `beads_api.go`
- `pkg/verify/evidence.go` - No longer exists. Split into `test_evidence.go` + `visual.go`
- `pkg/verify/cross_project.go` - No longer exists. Cross-project logic integrated into `check.go`

**Model claims 3 gates. Code has 14:**

| Model's Gates | Actual Gates (14 total) |
|---|---|
| Phase Gate | 1. Phase Complete |
| Evidence Gate | 2. Synthesis, 3. Test Evidence, 4. Git Diff, 5. Build Verification |
| Approval Gate | 6. Visual Verification (with approval) |
| Not in model | 7. Constraint, 8. Phase Gate (skill-required phases), 9. Skill Output, 10. Decision Patch Limit, 11. Accretion, 12. Explain-Back (Gate1), 13. Behavioral (Gate2), 14. Handoff Content |

**Model pseudocode doesn't match reality:**
- Model shows `strings.Contains(comment.Text, "Phase: Complete")` - actual uses regex `Phase:\s*(\w+)` via `ParsePhaseFromComments()`
- Model shows `containsImageURL()` for evidence - actual uses 22 framework-specific patterns with false-positive rejection
- Model's tier table is incomplete (missing "light" tier auto-skip for synthesis)

### Gates That Are Wired and Working (14/14)

Every gate found in the codebase is:
1. **Implemented** - Has code with logic (not stubs)
2. **Integrated** - Called from `VerifyCompletionFull()` in `check.go`
3. **Tested** - Has dedicated test file with multiple test cases
4. **Skippable** - Each has `--skip-{gate}` flag with required reason

**Strongest gates (anti-theater design):**
- **Test Evidence**: 22 true-positive patterns + 11 false-positive patterns that explicitly reject vague claims ("tests pass" without counts). This is the clearest anti-theater mechanism.
- **Git Diff**: Compares SYNTHESIS.md claims against actual git changes. Catches fabricated delta sections.
- **Build Verification**: Actually runs `go build ./...`. Observable, unfakeable.

### Daemon Verification: Fully Wired

Contrary to potential concerns about autonomous runaway:
- `ProcessCompletion()` runs `VerifyCompletionFull()` before marking anything
- `VerificationTracker.IsPaused()` IS wired into the daemon loop (line 342-380 of daemon.go)
- Threshold-based pause (default 3) prevents unchecked auto-completion
- `SeedFromBacklog()` persists state across daemon restarts
- Signal files (`~/.orch/daemon-verification.signal`) bridge human verification to daemon awareness

### What's NOT Verified (Spectrum Gaps)

| Verification Level | Status | Details |
|---|---|---|
| Agent claims completion | Strong | Phase gate + beads comment parsing |
| Artifacts exist | Strong | SYNTHESIS.md, skill outputs, constraints |
| Tests were run | Strong | Anti-theater evidence detection |
| Binary compiles | Strong | `go build ./...` |
| Tests pass | Partial | Checks for test output evidence in comments, but doesn't RUN tests |
| Smoke test | Missing | No automated smoke/integration test execution |
| Live e2e | Missing | Checks for Playwright evidence but doesn't execute |
| Adversarial | Missing | Agent could write "go test ./... - PASS (47 tests)" without running tests |

### Coaching Plugin Coverage Gap

The coaching plugin (behavioral monitoring) only works for OpenCode API spawns. Claude CLI/tmux spawns (the "escape hatch" for critical work) have NO behavioral monitoring. This means:
- Normal work: monitored by coaching plugin
- Critical infrastructure work: unmonitored (exactly when monitoring matters most)

---

## Model Impact

- [x] **Contradicts** invariant: "Three independent gates (Phase, Evidence, Approval)" — Actually **14 gates** organized in a tier-aware pipeline. The model significantly understates the system's sophistication.

- [x] **Contradicts** invariant: Source file references — `pkg/verify/phase.go`, `pkg/verify/evidence.go`, `pkg/verify/cross_project.go` no longer exist. Code has been refactored into `beads_api.go`, `test_evidence.go`, `visual.go`, and integrated into `check.go`.

- [x] **Confirms** invariant: "Gates are independent and cumulative" — Each gate CAN fail independently, and all must pass (or be explicitly skipped). The pipeline checks all gates and reports all failures at once.

- [x] **Confirms** invariant: "Tier-aware verification" — Three tiers (light, full, orchestrator) route to different gate sets. The model's tier table is incomplete but directionally correct.

- [x] **Confirms** invariant: "Knowledge work surfaces for review" — Escalation model surfaces investigation/architect/research for mandatory review. `IsKnowledgeProducingSkill()` drives this.

- [x] **Extends** model with: The verification spectrum gap analysis. System is strong at "does artifact exist?" and "does agent claim completion?" levels but weak at "did the verification actually execute?" level. The anti-theater mechanisms (false positive filters) are excellent for test evidence comments but don't extend to actually executing tests. The unfakeable signal is `go build` — everything else could theoretically be gamed by a sophisticated agent writing the right comment patterns.

- [x] **Extends** model with: Pure-noise gate removal (Phase 7). Model's evolution section stops at Phase 6 (Jan 14, 2026). Phase 7 (Feb 2026) removed 3 gates (agent_running, model_connection, commit_evidence) based on friction analysis of 1,008 bypass events.

- [x] **Extends** model with: Daemon verification is now fully operational. The model doesn't mention the daemon's verification at all. The daemon runs `VerifyCompletionFull()`, has threshold-based pause via `VerificationTracker`, and persists state across restarts.

---

## Notes

**The model needs a major rewrite.** The current model is ~60% stale:
- File references are wrong (3 deleted files)
- Gate count is wrong (3 claimed, 14 actual)
- Pseudocode doesn't match implementation
- Evolution section missing Phase 7
- No mention of daemon verification integration
- No mention of anti-theater mechanisms in test evidence

**Recommendation:** The model should be rewritten from scratch using the inventory produced by this audit as the authoritative source. The 14-gate architecture with tier-aware routing and targeted bypasses is the actual system.

**The verification vocabulary gap** (from the original task framing) can now be scoped: the system has excellent "does it exist?" verification but lacks "did it actually run?" verification. The only gate that executes something real is `go build`. Everything else checks for evidence of execution, not execution itself.
