# Probe: JSONL hash mismatch warnings should be suppressed by default in orch CLI fallback

**Model:** /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Should orch's bd CLI fallback run with `--quiet` by default so routine concurrent-write hash mismatch warnings are suppressed, while still allowing warning visibility when debug mode is enabled?

---

## What I Tested

**Command/Code:**
```bash
go test ./pkg/beads -run 'TestRunBDCommand_AddsQuietByDefault|TestRunBDCommand_DebugModeSkipsQuiet' -v
go test ./pkg/beads -run 'TestPrependSandboxArg|TestPrependSandboxArg_DisablesQuietWhenDebugEnabled' -v
```

**Environment:**
- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Code path under test: `pkg/beads/runBDCommand` + `prependSandboxArg`
- Added assertions that default mode injects `--quiet`, and `ORCH_DEBUG=1` disables quiet injection

---

## What I Observed

**Output:**
```text
=== RUN   TestRunBDCommand_AddsQuietByDefault
--- PASS: TestRunBDCommand_AddsQuietByDefault (0.00s)
=== RUN   TestRunBDCommand_DebugModeSkipsQuiet
--- PASS: TestRunBDCommand_DebugModeSkipsQuiet (0.00s)
PASS

=== RUN   TestPrependSandboxArg
--- PASS: TestPrependSandboxArg (0.00s)
=== RUN   TestPrependSandboxArg_DisablesQuietWhenDebugEnabled
--- PASS: TestPrependSandboxArg_DisablesQuietWhenDebugEnabled (0.00s)
PASS
```

**Key observations:**
- Default fallback command construction now includes `--quiet`, which suppresses non-essential bd warnings.
- When `ORCH_DEBUG` is set, `--quiet` is intentionally not injected, so warning-level diagnostics remain visible.

---

## Model Impact

**Verdict:** extends — fallback CLI output should be mode-aware (quiet by default, verbose in debug)

**Details:**
This extends the model with an operator-noise invariant: in multi-agent concurrency, hash mismatch warnings are expected and should not flood normal workflows. The fallback layer now defaults to quiet mode while preserving investigation fidelity via `ORCH_DEBUG`.

**Confidence:** High — behavior is covered by focused tests asserting both default and debug-mode execution paths.
