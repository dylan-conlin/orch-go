---
title: "Measurement command identity crisis — 10 commands, 3 namespaces, no caller telemetry, who uses what?"
status: resolved
created: 2026-03-19
updated: 2026-03-28
resolved_to: "culled: operational debt, not a thread"
---

# Measurement command identity crisis — 10 commands, 3 namespaces, no caller telemetry, who uses what?

## 2026-03-19

10 measurement commands across orch harness (audit/report/gate-effectiveness/snapshot), orch health, orch doctor (+ 7 flags), orch stats. Key findings: (1) daemon calls Go functions directly, bypassing CLI entirely; (2) no telemetry on command invocation — don't know if harness report has ever been run; (3) Dylan can't keep track of what's what; (4) orch health and orch doctor --health are identical. Proposed consolidation: doctor = operations (is it running?), health = health (is it healthy?, absorbs harness audit/report/gate-effectiveness), harness = control plane actions only (init/lock/unlock/verify). Released telemetry investigation (orch-go-kwoka) to measure actual usage before consolidating.
