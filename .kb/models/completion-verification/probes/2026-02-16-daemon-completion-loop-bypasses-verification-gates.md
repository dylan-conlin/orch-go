# Probe: Daemon Completion Loop Bypasses Verification Gates

**Model:** completion-verification
**Date:** 2026-02-16
**Status:** Complete

---

## Question

Does the daemon's completion path bypass the verification gates (explain-back, behavioral verification, discovered work disposition, checkpoint enforcement) that `orch complete` enforces? If so, is the VerificationTracker compensating for missing gates rather than being a real safety mechanism?

---

## What I Tested

Traced both code paths by reading the source files:

**Daemon path:** `pkg/daemon/completion_processing.go` → `ProcessCompletion()`
**CLI path:** `cmd/orch/complete_cmd.go` → `runComplete()`

Compared which verification gates each path enforces.

```bash
# Identified gate calls in daemon path
grep -n "VerifyCompletionFull\|checkpoint\|ExplainBack\|gate1\|gate2\|PromptDiscoveredWork\|RunExplainBack\|RecordGate2" pkg/daemon/completion_processing.go

# Identified gate calls in CLI path
grep -n "VerifyCompletionFull\|checkpoint\|ExplainBack\|gate1\|gate2\|PromptDiscoveredWork\|RunExplainBack\|RecordGate2" cmd/orch/complete_cmd.go

# Confirmed explain-back/gate2 only exist in CLI
grep -rl "RunExplainBackGate\|RecordGate2" pkg/daemon/ cmd/orch/
```

---

## What I Observed

### Gate Comparison: Daemon vs CLI

| Gate | `orch complete` (CLI) | Daemon `ProcessCompletion()` | Gap? |
|------|----------------------|-------------------------------|------|
| Phase: Complete check | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| SYNTHESIS.md | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Test evidence | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Visual verification | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Git diff | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Build verification | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Constraint verification | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Skill output verification | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| Decision patch limit | ✅ via `VerifyCompletionFull` | ✅ via `VerifyCompletionFull` | No |
| **Explain-back (gate1)** | ✅ `RunExplainBackGate()` | ❌ **Not called** | **YES** |
| **Behavioral verification (gate2)** | ✅ `RecordGate2Checkpoint()` | ❌ **Not called** | **YES** |
| **Checkpoint enforcement** | ✅ `HasGate1Checkpoint` / `HasGate2Checkpoint` blocks completion | ❌ **Not checked** | **YES** |
| **Discovered work disposition** | ✅ Interactive prompts for follow-up items | ❌ **Not called** | **YES** |
| **Liveness check** | ✅ Warns if agent still running | ❌ **Not called** | **YES** |
| Escalation model | ✅ (implicit via gate structure) | ✅ `DetermineEscalationFromCompletion()` | No |
| **Issue close** | ✅ `verify.CloseIssue()` | ❌ Labels `daemon:ready-review` only | No (by design) |
| **Verification heartbeat** | ✅ `control.Ack()` | ❌ Not called | **YES** |

### Critical Finding: Daemon Does NOT Close Issues

The daemon's `ProcessCompletion()` (line 260-264) does **NOT** close the beads issue. Instead it adds a `daemon:ready-review` label:

```go
if err := verify.AddLabel(agent.BeadsID, "daemon:ready-review"); err != nil {
    result.Error = fmt.Errorf("failed to mark ready for review: %w", err)
    return result
}
```

This means the daemon is a **triage layer**, not a completion layer. It:
1. Runs `VerifyCompletionFull()` (same automated gates as CLI)
2. Runs `DetermineEscalationFromCompletion()` to check if auto-completion is safe
3. If escalation is `EscalationNone/Info/Review` → labels `daemon:ready-review`
4. If escalation is `EscalationBlock/Failed` → leaves issue as-is, requires human

The actual closing still requires `orch complete` (with explain-back gates).

### Escalation Model Acts as Safety Valve

The escalation model at line 242 prevents daemon auto-labeling when:
- Verification fails (`EscalationFailed`)
- Visual approval needed (`EscalationBlock`)

But it ALLOWS labeling for:
- Knowledge-producing skills with recommendations (`EscalationReview`)
- Clean completions (`EscalationNone`)
- Large scope changes (`EscalationInfo`)

### The Gap That Remains

Even though the daemon doesn't close issues directly, it **labels** them as `daemon:ready-review`. This means:
1. Daemon labels issue without explain-back, gate2, or discovered work gates
2. Orchestrator sees `daemon:ready-review` label
3. Orchestrator runs `orch complete` which enforces explain-back + gate2
4. **The full gate pipeline runs when the orchestrator completes**

The design is intentionally two-phase: daemon triages (automated gates), orchestrator completes (human gates).

### VerificationTracker's Role

The VerificationTracker (line 268-275) counts daemon auto-completions and pauses after N completions to force human verification. Given the daemon doesn't actually close issues, the tracker is compensating for a different risk: **review backlog accumulation**, not gate bypass. If daemon labels 20 issues as ready-review without any human actually running `orch complete`, the tracker pauses to prevent unbounded accumulation.

---

## Model Impact

- [x] **Confirms** invariant: The daemon does NOT bypass automated verification gates (phase_complete, synthesis, test_evidence, visual, git_diff, build, constraint) — it runs the same `VerifyCompletionFull()` as `orch complete`
- [x] **Confirms** invariant: The daemon uses escalation model to prevent labeling when human approval is needed (EscalationBlock/Failed)
- [x] **Extends** model with: The daemon's completion path is a **two-phase design** — daemon runs automated gates and labels `daemon:ready-review`, but does NOT close issues. The explain-back (gate1), behavioral verification (gate2), discovered work disposition, and checkpoint enforcement gates are **intentionally deferred** to `orch complete`, which the orchestrator must still run. The VerificationTracker compensates for review backlog accumulation risk, not for missing gates.
- [x] **Extends** model with: Six gates are CLI-only (explain-back, gate2, checkpoint enforcement, discovered work disposition, liveness check, verification heartbeat). These are all inherently interactive/human gates that cannot be automated by the daemon.

---

## Notes

- The `daemon:ready-review` label is the bridge between the two phases. Without the orchestrator running `orch complete`, issues stay open indefinitely — the daemon cannot close them.
- The VerificationTracker is best understood as a **review pace governor**: it ensures human verification keeps up with daemon triage rate, preventing a scenario where 50 issues are labeled ready-review but none are actually reviewed.
- The model's concern about "daemon completion loop bypassing gates" is partially validated but the risk is lower than expected: the daemon is a triage/labeling layer, not a closing layer. The full gate pipeline still runs during `orch complete`.
- However, if someone builds a shortcut that watches for `daemon:ready-review` and auto-runs `orch complete --force` or `--skip-explain-back`, the gates WOULD be bypassed. The safety depends on the orchestrator being a real human-in-the-loop.
