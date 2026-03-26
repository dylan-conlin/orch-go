# Session Synthesis

**Agent:** `og-inv-daemon-decide-whether-26mar-ccf5`
**Issue:** `orch-go-zaeiu`
**Duration:** 2026-03-26 -> 2026-03-26
**Outcome:** blocked

---

## Plain-Language Summary

I traced how the daemon decides whether to start work on its own or leave it for human/orchestrator review. The key split is that spawning is blocked or allowed by queue gates and per-issue filters before any agent launches, while most orchestrator handoff happens later, after a worker finishes and the daemon labels the issue `daemon:ready-review` unless the work qualifies for auto-complete.

## TLDR

The daemon only auto-spawns issues that survive cycle-level gates, issue-level compliance, and routing checks. It usually defers to the orchestrator after completion rather than before spawn, with `daemon:ready-review` as the normal handoff and `triage:review` as the escalation path for repeated verification failures.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-daemon-decide-whether-auto-spawn.md` - Investigation tracing the daemon decision tree.
- `.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/VERIFICATION_SPEC.yaml` - Verification contract for this session.
- `.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/SYNTHESIS.md` - Session synthesis for orchestration review.
- `.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/BRIEF.md` - Short human-facing brief.

### Files Modified
- None.

### Commits
- Pending local commit for this investigation session.

---

## Evidence (What Was Observed)

- `CheckPreSpawnGates()` blocks the whole cycle on verification pause, completion-health failure, comprehension throttle, or rate limit before `Decide()` considers any issue (`pkg/daemon/compliance.go:25`, `pkg/daemon/ooda.go:119`).
- `Decide()` defers test-like work behind implementation siblings, then applies `CheckIssueCompliance()` to reject unsatisfied items and select the first passing issue (`pkg/daemon/ooda.go:149`, `pkg/daemon/sibling_sequencing.go:60`, `pkg/daemon/compliance.go:109`).
- `RouteIssueForSpawn()` keeps routing autonomous by swapping in extraction work or escalating to `architect` for hotspot matches instead of handing the issue to the orchestrator (`pkg/daemon/coordination.go:37`, `pkg/daemon/architect_escalation.go:78`).
- `RouteCompletion()` defaults completed work to `label-ready-review`, which becomes a `daemon:ready-review` orchestrator handoff unless the agent is light/auto/scan tier (`pkg/daemon/coordination.go:147`, `pkg/daemon/completion_processing.go:355`).

### Tests Run
```bash
go test ./pkg/daemon -run 'Test(CheckPreSpawnGates|CheckIssueCompliance|Decide|ShouldDeferTestIssue|CheckArchitectEscalation|RouteCompletion|ProcessCompletion)'
```

---

## Verification Contract

See `.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/VERIFICATION_SPEC.yaml`.

Key outcomes:
- Targeted daemon decision-tree tests passed.
- Investigation cites the exact codepaths for gates, filtering, routing, spawn, and orchestrator handoff.

---

## Architectural Choices

No architectural choices - task was within existing patterns.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-daemon-decide-whether-auto-spawn.md` - Canonical trace of spawn versus orchestrator handoff.

### Decisions Made
- The correct explanation is a two-stage model: pre-spawn autonomy plus post-completion orchestrator review.

### Constraints Discovered
- The phrase "defer to orchestrator" is misleading if treated as a pre-spawn-only branch because the default orchestrator handoff lives in completion routing.

### Externalized via `kb quick`
- `kb quick decide "Daemon auto-spawn requires cycle gates plus per-issue compliance; orchestrator handoff is mainly completion routing" --reason "Verified by tracing pkg/daemon OODA, compliance, routing, and completion codepaths plus targeted tests on 2026-03-26"`

---

## Next (What Should Happen)

**Recommendation:** resume

### If Resume
- Commit is blocked by unrelated pre-commit compilation failures in `cmd/orch`.
- Investigation and workspace artifacts are authored and ready to commit once the repo builds again.
- Resume by re-staging this session's four files plus the investigation file and retrying the same commit.

---

## Unexplored Questions

- How often live daemon behavior diverges from this static decision tree because of beads failures, retries, or multi-project queue interactions.
- Whether the dashboard should show this tree explicitly so Dylan can see why an item was skipped or handed to review.

---

## Friction

Tooling - pre-commit compilation gate failed on unrelated repository changes, so the investigation cannot be committed yet.

---

## Session Metadata

**Skill:** `investigation`
**Model:** `openai/gpt-5.4`
**Workspace:** `.orch/workspace/og-inv-daemon-decide-whether-26mar-ccf5/`
**Investigation:** `.kb/investigations/2026-03-26-inv-daemon-decide-whether-auto-spawn.md`
**Beads:** `bd show orch-go-zaeiu`
