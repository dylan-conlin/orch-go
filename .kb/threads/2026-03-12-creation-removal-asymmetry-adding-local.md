---
title: "Creation/removal asymmetry — adding is local, removing is global, and this is substrate-independent"
status: resolved
created: 2026-03-12
updated: 2026-03-18
resolved_to: "Measurement-driven retirement is the Layer -1. Gates get removed when empirical evidence proves ceremony (bypass rates, FP rates, zero true positives). The asymmetry is confirmed and substrate-independent — adding a gate is one commit, removing requires weeks of measurement, investigation, and cross-cutting analysis."
---

# Creation/removal asymmetry — adding is local, removing is global, and this is substrate-independent

## 2026-03-12

The deepest and least explored observation in the accretion work. Adding is a single-agent action: create a file, add a column, ship an endpoint, write an investigation. Removing requires coordinating with unknown dependents: who imports this? who reads this table? who calls this API? who built on this finding? The asymmetry is structural — it depends on the relationship between action scope (local) and consequence scope (distributed), not on the substrate. External evidence: 73% feature flags never removed (FlagShark), 85% shared drive data is dark/ROT (Veritas), 39% of orgs can't inventory their APIs (APIsec). In orch-go: 5/6 extraction commits added net lines, new bloated files emerge as fast as old ones are extracted. This may be the one place where substrate-independence is genuinely earned, because the mechanism is action-theoretic (local creation vs global removal), not substrate-specific. The recursive question: coordination mechanisms themselves are subject to creation/removal asymmetry — gates accrete too. 15 gates added, 1 removed (health score). What's the Layer -1 that removes gates that have become ceremony? Ostrom's answer: nested enterprises and meta-governance. Our Layer 4 (gates that generate gates) is half the answer — we also need gates that retire gates.

## 2026-03-18 — Resolution: Measurement-Driven Retirement as Layer -1

The project answered its own question in the 6 days since this thread opened. Three gate removals occurred, each following the same pattern:

1. **Health score spawn gate** (removed Mar 11): Never fired — formula calibrated to pass existing state. Advisory gate that always printed "✓" = false assurance. Decision: remove entirely, keep metric as diagnostic only.

2. **Self-review completion gate** (removed Mar 13): 71 events, 0 true positives, 79% FP rate, 44 bypasses (highest in system). Dominant FP (`fmt.Print` in CLI project) structurally unfixable with regex. Decision: remove entirely.

3. **Accretion gates** (downgraded Mar 17): 55 firings, 2 blocks, both bypassed in seconds (100% bypass rate). Zero quality difference between enforced/bypassed cohorts. Hotspot reduction (12→3 files) driven by daemon event responses, not by gate blocks. Decision: convert from blocking to advisory.

**The Layer -1 is measurement-driven retirement.** The mechanism: gates emit events → events accumulate into measurable outcomes (bypass rates, FP rates, true positive counts) → periodic audits surface gates where blocking adds friction but zero behavioral change → decision to remove/downgrade.

**The asymmetry is confirmed and quantified:** Adding a gate = one commit, one decision. Removing a gate = weeks of data collection (71+ events), retrospective audit, evidence document, decision document, cross-cutting code removal. The cost ratio is roughly 10:1 (removal:creation), which explains why gates accrete — the activation energy for removal far exceeds creation.

**What makes retirement work:** The crucial ingredient is *event emission*. Gates that emit events when they fire create the measurement trail needed for later retirement. Gates that silently block create no evidence base — they can neither justify their existence nor be argued against. The design implication: every gate should emit, even if it doesn't block, because emission is what makes retirement possible.

**Substrate-independence confirmed:** The same asymmetry appears in the external evidence (feature flags, APIs, shared drives) and in the project's own gate lifecycle. The mechanism is always the same: creation is local and cheap, removal requires global coordination with unknown dependents (who relies on this gate? what behavior changes if we remove it?). The only difference is the coordination mechanism — orch-go uses event-driven measurement, organizations use audits and deprecation processes, but both are solving the same problem: making the cost of removal legible enough to act on.
