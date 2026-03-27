# Brief: orch-go-7jfi8

## Frame

Three issues spawned today — "timeout budget trade-off", "stall duration abstraction", "regression coverage" — all produced agents that had nothing to orient from beyond a title. The agents were blind because the issues were created by workers mid-task (title only, no description) and the daemon spawned them faithfully. The orchestrator's enrichment protocol (write good descriptions, add kb context, tag with skill labels) exists, but these two creation paths — worker `bd create` and architect auto-create — bypass it entirely.

## Resolution

I expected to find a missing mechanism and instead found a layering problem. The spawn pipeline already gathers kb context at spawn time (`runPreSpawnKBCheckFull`), already extracts FRAME comments from beads, already builds an ORIENTATION_FRAME from the issue description. The gap is that nothing feeds these mechanisms when issues are created outside the orchestrator's conversational flow.

The highest-leverage fix turned out to be architect auto-create in `complete_architect.go`. When an architect completes and auto-creates an implementation issue, it includes the TLDR and next actions from the SYNTHESIS but never runs `kb context` to gather relevant constraints and decisions. The architect had this knowledge during its session — it just doesn't carry it forward. One function change in `buildImplementationDescription()` — run kb context with the next action keywords, include the matches — and the implementing agent gets a rich orientation frame instead of a bare summary.

For worker `bd create` issues, the right fix is the kb context timeout bug (orch-go-k6c0v). Once that's fixed, the spawn-time mechanism already compensates using title keywords. Adding a thin-issue advisory in the daemon gives observability into how often this happens.

## Tension

The worker-base skill guidance actively models description-free issue creation — the `bd create` examples in `discovered-work.md` show title-only commands with no `-d` flag. Updating this guidance requires governance escalation (worker-base is protected). But the deeper question is whether workers creating discovered work mid-task can meaningfully enrich their issues, or whether this is always going to be spawn-time's job. The architect auto-create fix is clean because the source material is rich. Worker discovered-work is fundamentally different — the worker noticed something in passing. How much context can you reasonably extract from "noticed while working on something else"?
