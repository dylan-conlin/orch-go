---
title: "Creation/removal asymmetry — adding is local, removing is global, and this is substrate-independent"
status: open
created: 2026-03-12
updated: 2026-03-12
resolved_to: ""
---

# Creation/removal asymmetry — adding is local, removing is global, and this is substrate-independent

## 2026-03-12

The deepest and least explored observation in the accretion work. Adding is a single-agent action: create a file, add a column, ship an endpoint, write an investigation. Removing requires coordinating with unknown dependents: who imports this? who reads this table? who calls this API? who built on this finding? The asymmetry is structural — it depends on the relationship between action scope (local) and consequence scope (distributed), not on the substrate. External evidence: 73% feature flags never removed (FlagShark), 85% shared drive data is dark/ROT (Veritas), 39% of orgs can't inventory their APIs (APIsec). In orch-go: 5/6 extraction commits added net lines, new bloated files emerge as fast as old ones are extracted. This may be the one place where substrate-independence is genuinely earned, because the mechanism is action-theoretic (local creation vs global removal), not substrate-specific. The recursive question: coordination mechanisms themselves are subject to creation/removal asymmetry — gates accrete too. 15 gates added, 1 removed (health score). What's the Layer -1 that removes gates that have become ceremony? Ostrom's answer: nested enterprises and meta-governance. Our Layer 4 (gates that generate gates) is half the answer — we also need gates that retire gates.
