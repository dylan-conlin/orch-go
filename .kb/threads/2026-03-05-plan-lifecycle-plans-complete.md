---
title: "Plan lifecycle — when and how plans complete"
status: resolved
created: 2026-03-05
updated: 2026-03-17
resolved_to: "Manual frontmatter edit is sufficient — system already filters correctly"
---

# Plan lifecycle — when and how plans complete

## 2026-03-05

Plans currently have no formal lifecycle — freetext Status field, no completion mechanism, no filtering from orient/status. Synthesis-as-comprehension plan is 'Implemented' but still surfaces everywhere.

Plans vs threads: threads accumulate forming insight (understanding), plans navigate decisions and phase execution (coordination). Different end states — threads resolve into artifacts (models, decisions), plans end when their work ships.

Plans probably just need a 'completed' state, not a resolution pointer. They're coordination artifacts, not understanding artifacts. When work ships, the plan is done. Understanding produced along the way already lives in models, decisions, and threads — the plan doesn't need to point to them because beads issues connect everything.

Open question: should orch plan have a complete command? Or is manual frontmatter edit sufficient? Given throughput (1500+ spawns), even low-frequency operations benefit from CLI commands over manual edits.

## 2026-03-17 — Resolution

Answered empirically: manual frontmatter edit is sufficient. The initial concern ("no filtering from orient/status") was already addressed — code audit confirms all three consumers filter correctly:

1. **`plan show`** (`cmd/orch/plan_cmd.go`): Defaults to `FilterByStatus(plans, "active")`, shows "No active plans. Use --all to see completed/superseded plans." when empty.
2. **`ScanActivePlans`** (`pkg/orient/plans.go`): Hard-filters `status == "active"` — completed plans never appear in orient output.
3. **Daemon staleness** (`pkg/daemon/plan_staleness.go`): Explicitly calls `FilterByStatus(plans, "active")` before scanning.

The open question about a CLI command resolves to: **not needed**. Plan completion is a low-frequency operation (plans span weeks/months). Manual `**Status:** completed` edit is proportionate. Adding a CLI command would be over-engineering — the filtering infrastructure already works, and there's no completion verification to automate (unlike agent completion which has gates).

Status values are well-defined: `active`, `completed`, `superseded`, `draft`.

## Auto-Linked Investigations

- .kb/investigations/2026-03-26-inv-trace-complete-spawn-lifecycle-cli.md
- .kb/investigations/2026-03-11-design-exploration-orchestrator-lifecycle.md
- .kb/investigations/2026-03-01-design-complete-pipeline-extraction.md
