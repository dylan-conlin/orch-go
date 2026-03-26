# Session Synthesis

**Agent:** og-inv-stall-tracker-pkg-26mar-0ab2
**Issue:** orch-go-acohg
**Duration:** 2026-03-26 → 2026-03-26
**Outcome:** success

---

## TLDR

I traced the stall tracker from `pkg/daemon/stall_tracker.go` through `orch status`, `/api/agents`, and the attention collector. The key finding is that the 3 minute token-stall threshold currently measures one long gap between samples, so normal 30 second polling does not accumulate into a stall warning.

## Plain-Language Summary

The tracker is supposed to notice when an agent keeps running without spending more tokens. What it actually notices is whether the current sample arrived at least 3 minutes after the previous identical sample, because every call resets the saved timestamp. That means the warning mostly reflects missed polls, while the visible `STALLED` state downstream is still only advisory and shares a flag with a separate 15 minute phase-timeout path.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go` - empirical probe for tracker timing behavior
- `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/SYNTHESIS.md` - orchestration synthesis artifact
- `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/BRIEF.md` - Dylan-facing comprehension brief
- `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/VERIFICATION_SPEC.yaml` - verification evidence contract

### Files Modified
- `.kb/investigations/2026-03-26-inv-stall-tracker-pkg-daemon-stall.md` - investigation with findings, synthesis, uncertainty, and recommendations

### Commits
- Pending local commit for investigation artifacts

---

## Evidence (What Was Observed)

- `pkg/daemon/stall_tracker.go:58` stores a fresh snapshot before evaluating the unchanged-token path, so the saved timestamp is reset on every `Update` call.
- `cmd/orch/serve_agents_status.go:225` configures the shared tracker at 3 minutes, while comments in the same area describe the dashboard poll cadence as every 30 seconds.
- `go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go` showed unchanged 1s/2s/3s updates never stalled, while a 4 second unchanged gap did stall.
- `cmd/orch/status_cmd.go:353` and `cmd/orch/serve_agents_handlers.go:436` are the primary token-stall callers; `pkg/attention/stuck_collector.go:121` only escalates the signal for human review.
- `go test ./pkg/daemon ./cmd/orch ./pkg/attention` currently fails in unrelated repo tests, but compile-only checks for `./pkg/daemon` and `./cmd/orch` now pass.

### Tests Run
```bash
go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go
# PASS: unchanged sub-threshold samples stayed false; one 4s gap returned stalled=true

go test ./pkg/daemon ./cmd/orch ./pkg/attention
# FAIL: unrelated existing test failures in cmd/orch and pkg/daemon

go test ./cmd/orch -run '^$'
# PASS: ok github.com/dylan-conlin/orch-go/cmd/orch [no tests to run]

go test ./pkg/daemon -run '^$'
# PASS: ok github.com/dylan-conlin/orch-go/pkg/daemon [no tests to run]

go test ./pkg/attention
# PASS: ok github.com/dylan-conlin/orch-go/pkg/attention
```

---

## Architectural Choices

No architectural choices - task was investigation only. The recommendation is to route any fix through architect because the affected code is in a hotspot and spans multiple components.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-stall-tracker-pkg-daemon-stall.md` - answered how the tracker computes stall state and where the signal flows

### Decisions Made
- Route follow-up through architect rather than direct implementation because the findings cross tracker semantics, dashboard surfacing, and token type boundaries.

### Constraints Discovered
- `IsStalled` currently conflates token stagnation and phase stagnation, so downstream behavior depends on a mixed signal.
- Full repo tests remain noisy in unrelated areas, but the stall-tracker-related packages compile cleanly with tests skipped.

### Externalized via `kb quick`
- No new `kb quick` entry - investigation artifact is the primary knowledge capture.

---

## Verification Contract

See `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/VERIFICATION_SPEC.yaml` for exact commands, observed outcomes, and remaining manual validation.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-cig47`
**Skill:** `architect`
**Context:**
```text
The tracker currently resets its snapshot timestamp on every Update, so 30 second polling does not accumulate toward the 3 minute stall threshold. Follow-up should define the real stall contract, separate token-stall from phase-stall, and resolve the execution/opencode token type split before implementation.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- When did `pkg/daemon/stall_tracker.go` switch to `execution.TokenStats`, and was there an intended migration plan for the remaining `opencode` callers?
- Should the system treat token-stall and phase-stall as separate API fields rather than one overloaded `IsStalled` flag?

**Areas worth exploring further:**
- Live dashboard polling cadence versus SSE-driven alternatives for stall timing
- Attention queue impact once token-stall becomes cumulative

**What remains unclear:**
- Whether any hidden remediation path outside the traced files consumes `is_stalled`

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** investigation
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/`
**Investigation:** `.kb/investigations/2026-03-26-inv-stall-tracker-pkg-daemon-stall.md`
**Beads:** `bd show orch-go-acohg`
