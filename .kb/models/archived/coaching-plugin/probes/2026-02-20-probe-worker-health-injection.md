# Probe: Worker health detection and metric injection

**Model:** coaching-plugin
**Date:** 2026-02-20
**Status:** Complete

---

## Question

Do worker sessions get detected via `session.metadata.role === "worker"` (only cached true), and do worker-specific health metrics get emitted/injected from `tool.execute.after`? Which worker metrics are injected?

---

## What I Tested

Reviewed coaching plugin implementation and searched for worker health metrics and detection logic.

```bash
rg "detectWorkerSession|trackWorkerHealth|context_usage|tool_failure_rate|time_in_phase|commit_gap" .opencode/plugins/coaching.ts
```

---

## What I Observed

- `detectWorkerSession(sessionId, session)` checks `session?.metadata?.role === "worker"`, caches only `true`, and returns `false` otherwise; cache is never set to `false`.
- Worker path is selected inside `tool.execute.after` by calling `detectWorkerSession(sessionId, input.session)` and short-circuits orchestrator metrics when true.
- Worker health metrics are emitted inside `trackWorkerHealth(...)` and injected via `injectHealthSignal(...)` for:
  - `tool_failure_rate` (3+ consecutive failures)
  - `context_usage` (estimated tokens; every 50 tools or over threshold)
  - `time_in_phase` (every 30 tools when >5 minutes, with warnings at 15+)
  - `commit_gap` (every 30 tools when >10 minutes, warnings at 30+)
- Worker sessions also emit `accretion_warning` metrics on large file edits (>800 lines) and inject coaching messages (`accretion_warning`, `accretion_strong`) during `edit`/`write` tool calls.

---

## Model Impact

- [x] **Confirms** invariant: Worker detection uses session metadata role and only caches `true` results.
- [x] **Extends** model with: Worker health metrics emitted in `trackWorkerHealth` include `tool_failure_rate`, `context_usage`, `time_in_phase`, `commit_gap`, plus `accretion_warning` for large file edits with injected warnings.

---

## Notes

- Evidence: `.opencode/plugins/coaching.ts` (worker detection at `detectWorkerSession`, worker metrics in `trackWorkerHealth`, accretion path in `tool.execute.after`).
