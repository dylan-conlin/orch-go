---
title: "Model-driven strategy — models generate inquiry, confirmed inquiry generates direction, not models → implementation"
status: subsumed
created: 2026-03-19
updated: 2026-03-28
resolved_to: ""
---

# Model-driven strategy — models generate inquiry, confirmed inquiry generates direction, not models → implementation

## 2026-03-19

Models should determine strategy but remain falsifiable. Current state: models are reference material consulted after the fact. Strategy happens separately via symptom → triage → spawn. The displacement synthesis and career pivot both proved that reading models together generates better direction than ad-hoc deliberation. Three structural elements: (1) Model health as orient input — surface claim confidence, probe recency, cross-model tensions alongside throughput. (2) Claim-driven backlog generation — unconfirmed claims in high-impact models generate probe pressure via daemon, not implementation pressure. (3) Model conflict as priority signal — tensions between models are higher-signal than any P2 issue. Key constraint: models drive inquiry, not implementation. Flow is models → questions → investigations → evidence → strategy. The displacement finding is the cautionary tale: wrong knowledge in skill docs WAS already driving agent strategy and producing phantom constraints. Prescriptive models raise accuracy stakes — same property that makes knowledge transfer powerful makes wrong models dangerous. Epistemic integrity = models trustworthy enough to drive strategy without investigation for every decision.

Design crystallized. Three consumption points for claims.yaml: (1) Orient surfaces edges — tensions (cross-model conflicts), stale claims (old validation + recent activity in area), unconfirmed claims in active models. Not a claim dump — shows where understanding is weakest relative to where we're working. (2) Daemon generates probes demand-driven — stale claim + model referenced in recent spawns/decisions = probe issue. Dormant model stale claims don't trigger. (3) Orchestrator completion pipeline connects findings to claim graph — every completion confirms, extends, or contradicts existing claims. Claim lifecycle: investigation finds X → updates claim status/evidence/tensions. Bootstrap: start with 4 models (architectural-enforcement, measurement-honesty, agent-trust-enforcement, skill-content-transfer) — most actively referenced, clearest invariants. Build orient + daemon consumption on these, expand as pattern proves.

## Auto-Linked Investigations

- .kb/investigations/2026-02-13-inv-restore-models-probes-go-infrastructure.md
- .kb/investigations/2026-01-18-inv-investigate-gemini-text-mimicking-tool-calls.md
- .kb/investigations/archived/2026-01-16-inv-test-global-models-guides-context.md
- .kb/investigations/archived/2026-01-16-inv-test-models-guides-context.md
- .kb/investigations/synthesized/system-learning-loop/2025-12-25-inv-fix-orch-learn-act-generate.md

Implications session. Six concrete changes: (1) Session start becomes research briefing not task queue — orient surfaces where understanding is weakest relative to active work. (2) Priority becomes derived from claim health × dependency, not manually assigned. (3) Daemon shifts from task dispatcher to epistemic maintenance — autonomously probes stale claims in active models. (4) Completion review becomes claim update loop — every probe updates the graph. (5) Orchestrator role shifts from triage-dominant to synthesis-dominant. (6) KB ecosystem gets a backbone — claims are the connective tissue that turns a pile of artifacts into a directed graph of understanding. Three things get harder: knowledge debt becomes visible (uncomfortable but honest), model authorship demands falsifiability (not all insights are claims), self-referential tension becomes the defining question (system derives strategy from its own understanding). Dylan's frame: 'this is where we shine the light on the system itself' — knowledge debt visibility is the natural progression of epistemic integrity work, not a cost.

Session shipped the full pipeline in one sitting. 88 claims across 8 models. Orient shows knowledge edges live (2 tensions, 2 stale-in-active, 1 unconfirmed core). Daemon probe generation wired (2h interval). Completion pipeline updates claims.yaml from probe verdicts. First manual probe released (AE-02). Plan triage (13→3) was a natural consequence — dead plans were false-confidence notifications. Phase 3 (model-driven strategy) is now operational, not designed.

Tension-cluster pipeline designed, built, and wired in one session. Hub-based clustering (3+ claims from 2+ models sharing tension target) triggers architect spawn. Daemon periodic task scans every 24h, max 1 issue per cycle. ARCHITECT_OUTPUT.yaml parser in queue as final piece. The backlog is now partially self-generating: claim graph tensions produce architect issues that produce implementation issues. Pattern mirrors synthesis (3+ inv → model) exactly — same threshold logic, different input atoms.
