---
title: "Plan lifecycle — when and how plans complete"
status: open
created: 2026-03-05
updated: 2026-03-05
resolved_to: ""
---

# Plan lifecycle — when and how plans complete

## 2026-03-05

Plans currently have no formal lifecycle — freetext Status field, no completion mechanism, no filtering from orient/status. Synthesis-as-comprehension plan is 'Implemented' but still surfaces everywhere.

Plans vs threads: threads accumulate forming insight (understanding), plans navigate decisions and phase execution (coordination). Different end states — threads resolve into artifacts (models, decisions), plans end when their work ships.

Plans probably just need a 'completed' state, not a resolution pointer. They're coordination artifacts, not understanding artifacts. When work ships, the plan is done. Understanding produced along the way already lives in models, decisions, and threads — the plan doesn't need to point to them because beads issues connect everything.

Open question: should orch plan have a complete command? Or is manual frontmatter edit sufficient? Given throughput (1500+ spawns), even low-frequency operations benefit from CLI commands over manual edits.
