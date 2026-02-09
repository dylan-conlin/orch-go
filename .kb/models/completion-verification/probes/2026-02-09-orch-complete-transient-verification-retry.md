# Probe: Does `orch complete` recover from a one-shot transient verification failure?

**Model:** `.kb/models/completion-verification.md`
**Date:** 2026-02-09
**Status:** Complete

---

## Question

When verification fails once due to a transient condition (for example dashboard/API reachability), does `orch complete` retry automatically instead of failing immediately?

---

## What I Tested

**Command/Code:**
```bash
go test ./cmd/orch -run 'Test(ShouldRetryVerification|VerifyRegularAgentRetriesTransientGateFailure|VerifyRegularAgentNoRetryOnNonTransientFailure)$'
```

**Environment:**
- Repo: `orch-go`
- Branch: current working branch
- New regression test stubs first verification attempt as transient gate failure, second as success

---

## What I Observed

**Output:**
```text
ok  	github.com/dylan-conlin/orch-go/cmd/orch	0.021s
```

**Key observations:**
- `TestVerifyRegularAgentRetriesTransientGateFailure` passed and asserts verification is attempted twice with a single retry delay.
- `TestVerifyRegularAgentNoRetryOnNonTransientFailure` passed and asserts non-transient failures still fail immediately (no retry loop).

---

## Model Impact

**Verdict:** extends — transient verification failures can be auto-retried without relaxing gate strictness

**Details:**
The completion gate model already distinguishes strict verification requirements. This probe adds a resilience behavior: a one-shot retry on transient verification failures (network/server reachability, transient phase visibility) preserves correctness while reducing false-negative completion failures.

**Confidence:** High — backed by deterministic regression tests in `cmd/orch` and full `go test ./cmd/orch` pass.
