# Session Synthesis

**Agent:** og-arch-architect-stall-tracker-26mar-fee9
**Issue:** orch-go-cig47
**Duration:** 2026-03-26T16:00 → 2026-03-26T16:30
**Outcome:** success

---

## Plain-Language Summary

The stall tracker had two bugs (timestamp reset masking stalls, token type mismatch causing compile failures) — both are already fixed in prior commits. The actual remaining problem is that the `IsStalled` boolean on agent responses gets set by 4 different code paths with different meanings: the dashboard sets it for token stalls, phase stalls, never-started agents, and stale spawns, while the CLI only sets it for token stalls. This means downstream consumers like the attention system can't tell what kind of stall they're looking at. The fix is to add a `StallReason` string field that says *which* stall fired, keeping the boolean for backward compatibility.

---

## TLDR

Architect review of stall tracker found both bugs from the prior investigation are already fixed. The real issue is `IsStalled` being a Defect Class 5 violation — 4 different stall meanings in one boolean. Recommending additive `StallReason` field, created implementation issue orch-go-8e15i.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-architect-stall-tracker-semantics-token.md` — Architect investigation with findings, fork analysis, and recommendations

### Files Modified
- None (pure investigation/design session)

### Commits
- (investigation artifact commit)

---

## Evidence (What Was Observed)

- `go build ./...` succeeds — no compile failures exist (timestamp reset fixed in `c4d4aa496`, type migration in `5062bdf08`)
- `go test -run TestStall ./pkg/daemon/` — 7/7 pass including `FrequentPollingDetectsStall` which validates the timestamp fix
- Dashboard `serve_agents_handlers.go` sets `IsStalled` at lines 187, 208, 213, 444 (4 distinct conditions)
- CLI `status_cmd.go` sets `IsStalled` at line 363 only (token stall)
- `serve_agents_types.go:25` documents `IsStalled` as "same phase for 15+ minutes"
- `status_format.go:108` documents `IsStalled` as "no token progress for 3+ minutes"
- `StuckCollector` in `pkg/attention/stuck_collector.go:47` reads `is_stalled` with no way to distinguish cause

### Tests Run
```bash
go build ./...
# Success (zero errors)

go test -v -count=1 -run TestStall ./pkg/daemon/
# 7/7 PASS (6.7s)
```

---

## Architectural Choices

### Keep IsStalled + add StallReason (over replacing IsStalled with typed enum)
- **What I chose:** Additive `StallReason string` field alongside existing `IsStalled bool`
- **What I rejected:** (A) Replacing `IsStalled` with `StallType string`, (B) Leaving as-is with documentation
- **Why:** Zero breaking change for existing JSON consumers. The boolean remains backward-compatible; the reason string enables new behavior. Restructuring `IsStalled` would break dashboard JS and attention collector without material benefit.
- **Risk accepted:** Mild redundancy (bool + string) — standard "flag + reason" pattern, well-understood.

### Keep stall detection split across daemon and handlers (over consolidation)
- **What I chose:** Leave token stall in `pkg/daemon/stall_tracker.go`, phase stall in `cmd/orch/serve_agents_handlers.go`
- **What I rejected:** Moving all stall logic into a unified `pkg/stall` package
- **Why:** Phase stall depends on beads comment parsing (handler-layer knowledge). Pulling this into `pkg/daemon` would create an upward dependency. The split is architecturally correct — token detection is a library concern, phase detection is an application concern.
- **Risk accepted:** Two packages to understand for full stall picture.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes:
- Investigation documents all 4 IsStalled setter paths with evidence
- Implementation issue created (orch-go-8e15i) with clear scope
- No code changes needed (design-only session)

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-architect-stall-tracker-semantics-token.md` — Full architect investigation

### Decisions Made
- Decision: Additive `StallReason` field (not restructuring) because backward compatibility matters more than schema purity
- Decision: Keep stall detection split (daemon for tokens, handler for phase) because the dependency direction is correct

### Constraints Discovered
- The same JSON field `is_stalled` carries different semantics in CLI vs dashboard responses — Defect Class 5

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation, implementation issue, SYNTHESIS, BRIEF)
- [x] Tests passing (verified)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-cig47`

**Follow-up issue created:** orch-go-8e15i — implementation of StallReason field

---

## Unexplored Questions

- Does the dashboard Svelte UI branch on `is_stalled` directly? (Would determine if UI changes are needed alongside the API change)
- What is the real-world frequency distribution of each stall type? (Production metrics would tell us if this deconflation matters operationally)
- Should the attention collector route differently per stall reason? (e.g., token stalls may be more urgent than phase stalls)

---

## Friction

No friction — smooth session. Prior investigation provided excellent context.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-architect-stall-tracker-26mar-fee9/`
**Investigation:** `.kb/investigations/2026-03-26-inv-architect-stall-tracker-semantics-token.md`
**Beads:** `bd show orch-go-cig47`
