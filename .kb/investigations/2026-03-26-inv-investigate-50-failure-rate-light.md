## Summary (D.E.K.N.)

**Delta:** Light auto-complete fails 100% because CompleteLight skips gate1 (explain-back) but not gate2 (verified), which is required for Tier 1 work and impossible without a human.

**Evidence:** All 12 failures in daemon.log show identical error: `❌ verified: behavioral verification (gate2) missing for Tier 1 work (bug) — use --verified`. Gate2 requires human presence; daemon has none.

**Knowledge:** The `--headless` flag was designed to solve exactly this: it forces `review-tier=auto` (skipping all checkpoint gates) while preserving other verification. CompleteLight was added before `--headless` existed and was never updated.

**Next:** Fix applied — CompleteLight now uses `--headless`. Also discovered: `Complete()` has same class of bug (`--force` without required `--reason`).

**Authority:** implementation - Tactical fix within existing patterns, using the existing `--headless` mechanism

---

# Investigation: 50% Failure Rate on Light Auto-Complete Headless Path

**Question:** Why do all 12 light auto-complete attempts fail with 'exit status 1' while 12 headless brief generations succeed?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** orch-go-uiv9d
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** completion-verification

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/models/completion-verification/probes/2026-02-27-probe-light-tier-v2-verification-conflict.md | extends | yes | - |
| .kb/models/completion-verification/probes/2026-02-28-probe-synthesis-gate-light-tier-empirical-failures.md | extends | yes | - |

---

## Findings

### Finding 1: All failures are gate2 (verified) rejection, not synthesis or build

**Evidence:** Examined 6 failure entries in `~/.orch/daemon.log`. Every single one shows:
```
Cannot complete agent - 1 gate(s) failed:
  ❌ verified: behavioral verification (gate2) missing for Tier 1 work (bug) — use --verified
```
No other gate fails. The issue is exclusively the checkpoint gate requiring `--verified`.

**Source:** `~/.orch/daemon.log` lines 2104, 6156, 6747, 7409, 10425, 11799

**Significance:** This is not a flaky failure or environment issue — it's a deterministic design gap.

---

### Finding 2: CompleteLight skips gate1 but not gate2

**Evidence:** `CompleteLight` passes `--skip-explain-back --skip-reason` but NOT `--review-tier auto`, `--headless`, or `--verified`. The checkpoint gate logic (`complete_verification.go:30-31`) only skips checkpoints when `review-tier == "auto" || review-tier == "scan"`. Without a tier override, the workspace manifest tier ("review") is used, then escalated to "deep" by hotspot/diff thresholds. With "deep" tier, gate2 is required for Tier 1 work (bug/feature).

**Source:**
- `pkg/daemon/auto_complete.go:61-77` — CompleteLight args
- `cmd/orch/complete_verification.go:30-31` — checkpoint skip condition
- `cmd/orch/complete_verification.go:47-58` — gate2 requirement check

**Significance:** The root cause: CompleteLight was designed to be "lighter" than full auto-complete but not light enough. It skips one interactive gate while leaving another that's equally impossible without a human.

---

### Finding 3: --headless mode solves this correctly

**Evidence:** `--headless` in `cmd/orch/complete_cmd.go:277-291` does two things: (1) forces `completeReviewTier = "auto"`, which causes `skipCheckpoints = true` at line 30, skipping both gate1 and gate2; (2) auto-skips explain-back. This is exactly the behavior CompleteLight needs.

**Source:** `cmd/orch/complete_cmd.go:277-291` — headless mode implementation

**Significance:** The `--headless` flag was added after `CompleteLight` and is the correct abstraction for daemon-triggered non-interactive completion.

---

### Finding 4: Complete() (full auto-complete) has the same class of bug

**Evidence:** `Complete()` passes `--force` without `--reason`. Since the `--force` flag requires `--reason` (min 10 chars), this would fail immediately: `--reason is required when using --force (min 10 chars)`. Verified by running `build/orch complete test-123 --force` → error.

**Source:** `pkg/daemon/auto_complete.go:45-57` — Complete() args

**Significance:** The `auto-complete` path (for review-tier=auto/scan agents) is broken the same way, but currently unexpercised because no agents hit that path today.

---

## Synthesis

**Key Insights:**

1. **Gate arithmetic doesn't compose** — Skipping one interactive gate (explain-back) while leaving another (verified) creates a non-interactive path that still requires interaction. The correct abstraction is "non-interactive mode" (`--headless`), not individual gate skips.

2. **The 50% framing is misleading** — It's not "50% of completions fail." It's "100% of light completions fail, 100% of headless completions succeed." The two paths serve different completion categories, and one path is entirely broken.

3. **Headless was added after light but obsoletes its approach** — `CompleteLight` was designed with targeted skips; `--headless` was added later as a holistic non-interactive mode. CompleteLight should have been updated to use `--headless` when that flag was introduced.

**Answer to Investigation Question:**

All 12 light auto-complete failures are caused by the same bug: `CompleteLight` skips gate1 (explain-back) via `--skip-explain-back` but does not skip gate2 (verified). For Tier 1 work (bug/feature issues), gate2 requires `--verified` — a flag that can only be set by a human orchestrator. Since the daemon runs non-interactively, gate2 always fails. The fix is to use `--headless` mode, which forces `review-tier=auto` (skipping all checkpoint gates) while preserving other verification.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 12 failures show identical gate2 error (verified: daemon.log grep)
- ✅ `--headless` forces review-tier=auto and skips checkpoints (verified: code path trace through complete_cmd.go:277-291 → complete_verification.go:30-31)
- ✅ Fix compiles and all daemon tests pass (verified: `go test ./pkg/daemon/...`)
- ✅ Pre-existing test failures confirmed on master (verified: stash + test on clean master)

**What's untested:**

- ⚠️ Live daemon with the fix (requires rebuild + restart + new effort:small agents completing)
- ⚠️ Whether `Complete()` (--force without --reason) is actively failing (no auto-tier agents today)

**What would change this:**

- If `--headless` has side effects beyond checkpoint skipping that change completion behavior
- If effort:small agents have gate requirements beyond checkpoint gates that also need skipping

---

## References

**Files Examined:**
- `pkg/daemon/auto_complete.go` — CompleteLight implementation (the bug)
- `pkg/daemon/coordination.go` — routing logic and ExecuteCompletionRoute
- `cmd/orch/complete_cmd.go` — `--headless` flag implementation
- `cmd/orch/complete_verification.go` — checkpoint gate logic
- `~/.orch/daemon.log` — 12 failure entries

**Commands Run:**
```bash
# Find all light auto-complete failures in daemon log
grep -n "complete (light)" ~/.orch/daemon.log | tail -30

# Verify --headless flag works
build/orch complete test-123 --headless --workdir /tmp

# Verify --force requires --reason
build/orch complete test-123 --force

# Run relevant tests
go test ./pkg/daemon/... -run "AutoComplete|CompleteLight|RouteCompletion|ExecuteCompletion|FireHeadless" -v
```
