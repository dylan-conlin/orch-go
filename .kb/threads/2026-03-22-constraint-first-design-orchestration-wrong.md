---
title: "Constraint-first design — when orchestration is the wrong answer"
status: resolved
created: 2026-03-22
updated: 2026-03-22
resolved_to: ".kb/threads/2026-03-22-coordination-self-awareness-system-that.md"
---

# Constraint-first design — when orchestration is the wrong answer

## 2026-03-22

autoresearch (48k stars, 16 days) proves that tight constraint surfaces (1 file, 1 metric, 5-min runs, keep/discard via git) eliminate the need for orchestration machinery entirely. Meanwhile orch-go has 41 hotspots, 61% stale decisions, and hooks that block agents from committing their own work. The question this inverts: instead of asking 'what governance do we add next,' ask 'which problems could be constrained so the machinery isn't needed?' Connected to governance accretion failure mode (2026-03-21 session insight) and open-loop systems thread. First test case: the harness domain — OpenSCAD agents keep hitting hook blockers. Could tighter problem constraints (like autoresearch's 1-file surface) replace the gate stack?
