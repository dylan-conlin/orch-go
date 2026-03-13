## Provenance

These principles emerged from practice, not theory. Each traces to a specific incident.

| Principle | Date | What Broke | Evidence |
|-----------|------|------------|----------|
| **Provenance** | Dec 24, 2025 | Recognized that psychosis involved elaborate externalization anchored to closed loops, not reality | `blog/drafts/2025-12-24-amnesia-as-feature.md` |
| **Provenance** (correlation≠causation) | Jan 23, 2026 | Orchestrator found timeline correlation (OshCut maintenance Jan 17, failures Jan 18), escalated from "correlates" to "root cause confirmed" without evidence of mechanism | `price-watch/2026-01-23-llm-failure-mode-confirmation-bias-example.txt` |
| **Session Amnesia** | Nov 14, 2025 | Investigation on "habit formation" reframed: "we're designing habit formation for agents with amnesia" | `.kb/decisions/2025-11-14-session-amnesia-foundational-constraint.md` |
| **Evidence Hierarchy** | Nov 28, 2025 | Audit agent claimed "feature NOT DONE" by reading stale workspace - feature was actually implemented | `.kb/decisions/2025-11-28-evidence-hierarchy-principle.md` |
| **Gate Over Remind** | Dec 7, 2025 | "Why do I always have to remind you to update CLAUDE.md?" - reminders fail under cognitive load | `.kb/investigations/design/2025-12-07-discuss-potentially-refine-meta-orchestration.md` |
| **Self-Describing Artifacts** | Dec 2025 | Agents edited compiled SKILL.md instead of source files, breaking skillc build | `.kb/decisions/2025-12-21-skillc-architecture-and-principles.md` |
| **Surfacing Over Browsing** | Nov 2025 | Beads and orch independently converged on same pattern (`bd ready`, `orch inbox`) | Observed convergence |
| **Capture at Context** | Jan 14, 2026 | Session ended with empty handoff - gate existed but fired at wrong moment | `.kb/decisions/2026-01-14-capture-at-context.md` |
| **Track Actions, Not Just State** | Dec 27, 2025 | Orchestrator knew tier system, still checked SYNTHESIS.md on light-tier agents repeatedly | `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms.md` |
| **Pain as Signal** | Jan 17, 2026 | Dashboard alerts were stale - agents thrashing without awareness | `plugins/coaching.ts` implementation |
| **Infrastructure Over Instruction** | Jan 17, 2026 | Orchestration felt "forced" - humans had to drive usage of models agents couldn't see | `.kb/decisions/2026-01-17-infrastructure-over-instruction.md` |
| **Asymmetric Velocity** | Jan 14, 2026 | Synthesizing VB, UL, and Trust Calibration revealed common pattern | Synthesis of multiple principles |
| **Verification Bottleneck** | Jan 4, 2026 | Two system spirals (Dec 21: 115 commits; Dec 27-Jan 2: 347 commits) - agents locally correct, system deteriorating | `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` |
| **Understanding Lag** | Jan 14, 2026 | Dec 27-Jan 2 rollback included correct observability - dead agents had ALWAYS been dead | `.kb/decisions/2026-01-14-understanding-lag-pattern.md` |
| **Coherence Over Patches** | Jan 4, 2026 | Dashboard status logic accumulated 10+ conditions from incremental patches - 37% of commits were fixes | `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` |
| **Perspective is Structural** | Jan 5, 2026 | Orchestrator dropped into worker mode despite having full delegation rule in context | Meta-orchestrator session `og-work-meta-orchestrator-session-05jan` |
| **Authority is Scoping** | Jan 19, 2026 | Discovered hierarchy is about context-scoping, not reasoning capability | Decidability Graph work Jan 19 |
| **Escalation is Information Flow** | Jan 5, 2026 | Even with correct structural understanding, escalation carries cultural shame | Meta-orchestrator session |
| **Friction is Signal** | Jan 6, 2026 | Reviewing 5 session handoffs - every insight emerged from friction, not completions | Session handoffs Jan 5-6 |
| **AI Validation Loops** | Jan 21, 2026 | Drafting constraint promotions showed persuasive AI explanations were being treated as sufficient verification | `.kb/investigations/2026-01-21-inv-scan-kb-quick-constraints-promotion.md` |
| **Prompt Pollution** | Jan 21, 2026 | Prompt examples with textual tool-call syntax trained models to mimic tool calls as output text | `.kb/investigations/2026-01-21-inv-scan-kb-quick-constraints-promotion.md` |
| **Defect Class Blindness** | Feb 7, 2026 | 5 failure modes sharing unbounded resource consumption root cause shipped across different components; 779 investigations failed to connect them | `.kb/decisions/2026-02-07-unbounded-resource-consumption-constraints.md` and `.kb/models/system-reliability-feb2026.md` |
| **Evolve by Distinction** | Nov 2025 | Phase/Status conflation, Tracking/Knowledge conflation caused recurring confusion | `.kb/decisions/2025-11-28-evolve-by-distinction.md` |
| **Reflection Before Action** | Dec 2025 | Urge to manually extract recommendations - recognized `kb reflect` was the system solution | `.kb/decisions/2025-12-21-reflection-before-action.md` |
| **Premise Before Solution** | Dec 25, 2025 | Epic created from "how do we evolve skills" without validating premise | `.kb/investigations/2025-12-25-inv-investigate-question-asking-process-strategic.md` |
| **Pressure Over Compensation** | Dec 25, 2025 | About to paste context orchestrator should have known - realized compensating prevents learning | `.kb/decisions/2025-12-25-pressure-over-compensation.md` |
| **Understanding Through Engagement** | Jan 12, 2026 | Orchestrators delegating understanding to architects instead of synthesizing | `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` |
| **Strategic-First Orchestration** | Jan 11, 2026 | Coaching plugin had 8 bugs and 2 abandonments from tactical debugging | `.kb/decisions/2026-01-11-strategic-first-orchestration.md` |
| **Share Patterns Not Tools** | Dec 26, 2025 | skillc verify and orch-go both needed skill output verification | `.kb/decisions/2025-12-26-share-patterns-not-tools.md` |
| **Observation Infrastructure** | Jan 8, 2026 | 11 investigations revealed system performing better than it appeared - 89% completion showed as 72% | `.kb/decisions/2026-01-08-observation-infrastructure-principle.md` |
| **Escape Hatches** | Jan 21, 2026 | Critical orchestration paths lacked an independent fallback when primary infrastructure failed | `.kb/investigations/2026-01-21-inv-scan-kb-quick-constraints-promotion.md` |
| **Redundancy is Load-Bearing** | Mar 1, 2026 | Skill compression (2,185→619 lines) immediately degraded delegation compliance | Grammar Design Discipline synthesis, behavioral testing baseline |
| **Legibility Over Compliance** | Mar 1, 2026 | 14 commits of compliance gates in 18 days showed negligible improvement (39% vs 38% vs 30% bare) | Grammar Design Discipline synthesis, revert spiral investigation |
| **Deploy or Delete** | Mar 5, 2026 | 18 configuration-drift investigations in 30 days all traced to incomplete migrations | Configuration-drift defect class analysis, orchestrator-skill model synthesis |
| **Accretion Gravity** | Feb 2026 | spawn_cmd.go grew from 200 to 2,000 lines across 25 agents, each addition locally correct | Knowledge accretion model synthesis |
| **Gate Over Remind** (measurement evolution) | Mar 12, 2026 | Gates existed for months without measurement — enforcement without measurement is theological | Harness engineering model, thread: "measurement-enforcement pairing" |
| **Creation/Removal Asymmetry** | Mar 12, 2026 | Same accretion pattern across code (spawn_cmd.go), knowledge (1,200 investigations, 91% orphan rate), and config (dual verification systems) | Knowledge accretion model synthesis, thread: "creation/removal asymmetry" |

**The test for new principles:** Can you trace it to a specific failure? If not, it's not a principle yet.
