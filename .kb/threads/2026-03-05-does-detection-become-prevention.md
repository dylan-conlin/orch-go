---
title: "When does detection become prevention?"
status: open
created: 2026-03-05
updated: 2026-03-05
resolved_to: ""
---

# When does detection become prevention?

## 2026-03-05

Deploy or Delete has a named principle, a model, and a detection mechanism (orch doctor --migration-scan, orch-go-2r254). But detection is informational — it finds dual authorities after they exist. The real question: when does detection become prevention? Three escalation points identified: (1) daemon pre-spawn check — scan before spawning into an area, (2) completion gate — flag when agent changes introduce new dual authority, (3) doctor --watch auto-issue creation. Options 1-2 are actual gates; option 3 is still informational. This parallels the cognitive gaps insight — agents treat silence as permission. The migration scan detects aftermath; a gate prevents creation.

Cross-project boundary is a recurring failure surface. In one session: bd comments fails cross-repo, thread creation lands in wrong project, follower follows wrong project, orchestrator abandons live agents because they appear as phantoms from another project's perspective. Every tool assumes local project = complete picture. This isn't individual bugs — it's a missing cross-project identity layer. The system has no concept of 'this agent is mine but lives elsewhere.' Detection happened (we filed 4 bugs). Prevention would be: all orch commands that affect agent lifecycle check for cross-project ownership before acting destructively.
