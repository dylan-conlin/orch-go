# Probe: Can daemon infer completion from commit evidence plus idle session?

**Model:** .kb/models/agent-lifecycle-state-model.md
**Date:** 2026-02-08
**Status:** Complete

---

## Question

Model invariant says `session idle != completion`. Does adding a stricter rule (`session idle for threshold` + `successful git commit observed in that same session`) provide a safe completion signal for daemon auto-close without agent-emitted `Phase: Complete`?

---

## What I Tested

**Command/Code:**
```bash
go test ./pkg/daemon && go test ./cmd/orch -run TestDoesNotExist
```

**Environment:**
- Branch/worktree: `orch-go` working copy with daemon completion detection changes
- Detection implementation checks: in-progress issue, workspace exists, session idle >= threshold, `IsSessionProcessing == false`, and session message history contains successful `git commit` tool invocation

---

## What I Observed

**Output:**
```text
ok   github.com/dylan-conlin/orch-go/pkg/daemon   3.053s
ok   github.com/dylan-conlin/orch-go/cmd/orch     0.014s [no tests to run]
```

**Key observations:**
- Completion list now includes two sources: explicit `Phase: Complete` and commit+idle auto-detection for `in_progress` issues.
- For commit+idle detections, daemon backfills a `Phase: Complete - Auto-detected...` comment before verification/close, preserving audit trail.
- Detection is guarded by session-specific commit evidence (parsed from session tool history), not idle alone.

---

## Model Impact

**Verdict:** extends — `session idle != completion`

**Details:**
The invariant still holds in its original form (idle alone is insufficient). The probe supports an extension: idle can become a completion signal only when paired with session-scoped successful commit evidence and non-processing state. This narrows false positives while enabling recovery from context-expired agents that already committed work.

**Confidence:** Medium — validated by package tests and implementation behavior, but full end-to-end daemon runtime validation against live sessions is still pending.
