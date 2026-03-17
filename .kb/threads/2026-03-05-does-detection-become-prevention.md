---
title: "When does detection become prevention?"
status: resolved
created: 2026-03-05
updated: 2026-03-17
resolved_to: "Answer emerged empirically: detection becomes prevention when the failure class recurs enough to justify a gate. Both threads progressed — dual-authority stays detection (migration-scan), cross-project moved to prevention (identity layer + command fixes)."
---

# When does detection become prevention?

## 2026-03-05

Deploy or Delete has a named principle, a model, and a detection mechanism (orch doctor --migration-scan, orch-go-2r254). But detection is informational — it finds dual authorities after they exist. The real question: when does detection become prevention? Three escalation points identified: (1) daemon pre-spawn check — scan before spawning into an area, (2) completion gate — flag when agent changes introduce new dual authority, (3) doctor --watch auto-issue creation. Options 1-2 are actual gates; option 3 is still informational. This parallels the cognitive gaps insight — agents treat silence as permission. The migration scan detects aftermath; a gate prevents creation.

Cross-project boundary is a recurring failure surface. In one session: bd comments fails cross-repo, thread creation lands in wrong project, follower follows wrong project, orchestrator abandons live agents because they appear as phantoms from another project's perspective. Every tool assumes local project = complete picture. This isn't individual bugs — it's a missing cross-project identity layer. The system has no concept of 'this agent is mine but lives elsewhere.' Detection happened (we filed 4 bugs). Prevention would be: all orch commands that affect agent lifecycle check for cross-project ownership before acting destructively.

## Resolution (2026-03-17)

The question answered itself through 12 days of work on both threads:

**Dual-authority (Deploy or Delete):** Stayed at detection level. `orch doctor --migration-scan` shipped (orch-go-2r254) and works well as informational tooling. No pre-spawn or completion gate was added — the failure class hasn't recurred enough to justify gate overhead. The principle itself (documented in `.kb/global/.principlec/`) provides sufficient guidance for agents making decisions.

**Cross-project identity:** Moved from detection to prevention. Architect design completed (orch-go-jpsun), then 6+ fixes shipped: auto-track cross-project spawns (orch-go-7tubh), groups-aware project discovery (orch-go-22i8x), cross-project orch complete fixes (orch-go-6fm9r), --workdir flag for threads (orch-go-wrq9j). A `pkg/identity/` consolidation is in progress. This moved to prevention because the failure class was frequent (20+ Class 4 defects) and destructive (abandoning live agents).

**The pattern:** Detection becomes prevention when (1) the failure class recurs frequently enough that informational-only is insufficient, AND (2) the cost of a gate is lower than the cost of continued failures. Cross-project met both criteria; dual-authority met neither. Not every detection needs to become a gate — some stabilize as advisory tooling.
