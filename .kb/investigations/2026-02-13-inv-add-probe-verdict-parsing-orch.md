## Summary (D.E.K.N.)

**Delta:** Added probe verdict parsing to orch complete — probes in .kb/models/*/probes/ are now scanned and their Model Impact verdicts surfaced in completion output.

**Evidence:** 11 tests pass covering both structured ("**Verdict:** extends — ...") and checkbox ("- [x] **Confirms** invariant: ...") probe formats; FindProbesForWorkspace correctly filters by spawn time.

**Knowledge:** Probes use two verdict formats (structured and checkbox); matching probes to workspaces via spawn_time comparison is reliable and avoids git dependency.

**Next:** Close — implementation complete.

**Authority:** implementation — adds output to existing complete pipeline, no architectural changes.

---

# Investigation: Add Probe Verdict Parsing to orch complete

**Question:** How to surface probe verdicts during orch complete so orchestrator can merge model updates?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Two probe verdict formats exist

**Evidence:** PROBE.md template uses checkbox format (`- [x] **Confirms** invariant: ...`), real probes like completion-verification use structured format (`**Verdict:** extends — ...`).

**Source:** `pkg/spawn/probes.go:162-203` (template), `.kb/models/completion-verification/probes/2026-02-09-friction-bypass-analysis-post-targeted-skips.md` (real probe)

**Significance:** Parser must handle both formats to work with existing probes.

### Finding 2: Spawn time provides reliable workspace-probe matching

**Evidence:** Workspace `.spawn_time` file contains RFC3339 timestamp. Comparing probe file modification time against spawn time correctly identifies probes created during agent session.

**Source:** `cmd/orch/complete_cmd.go:1636-1642` (existing spawn time reading pattern)

**Significance:** Avoids git dependency, works with archived workspaces, handles concurrent agents correctly.

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` — completion pipeline, insertion point for probe verdicts
- `pkg/verify/synthesis_parser.go` — existing parsing patterns for synthesis sections
- `pkg/verify/check.go` — verification gate infrastructure
- `pkg/spawn/probes.go` — probe template, model name extraction utilities
- `.kb/models/completion-verification/probes/2026-02-09-*.md` — real probe format

**Files Created:**
- `pkg/verify/probe_verdict.go` — ProbeVerdict type, ParseProbeVerdict, FindProbesForWorkspace, FormatProbeVerdicts
- `pkg/verify/probe_verdict_test.go` — 11 tests covering both verdict formats, workspace matching, and output formatting
