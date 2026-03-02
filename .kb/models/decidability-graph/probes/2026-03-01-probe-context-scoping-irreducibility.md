# Probe: Context-Scoping Irreducibility Test

**Model:** decidability-graph
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The model claims: "The hierarchy exists because of context-scoping, not capability. Workers CAN answer framing questions if given the right context." (Section: "The Irreducible Function: Context Scoping", lines 77-101)

Three sub-questions:
1. Do workers reach good conclusions on framing-adjacent questions when given proper context?
2. Do they fail in ways that suggest capability limits beyond context?
3. Is orchestrator synthesis qualitatively different from what well-contexted workers produce?

---

## What I Tested

Examined 800+ archived workspace SYNTHESIS.md files and 40+ investigation/probe artifacts. Identified and evaluated 12+ cases where workers handled framing-adjacent questions (design judgments, premise challenges, synthesis, strategic recommendations).

### Test 1: Workers Handling Framing Questions (8 strong cases)

**Case A: Code Review Gate Design** (`.kb/models/completion-verification/probes/2026-02-25-probe-code-review-gate-design.md`)
- Worker skill: investigation (probe mode)
- Question: "Should we add agent code review to the completion pipeline?"
- What happened: Worker REJECTED the question's premise. Concluded "The completion pipeline isn't broken because nobody reads the diff" — the framing itself contained a hidden assumption. Instead recommended expanding execution-based gates (go vet) over adding judgment-based gates. Developed a three-type gate taxonomy (execution/evidence/judgment) that the model didn't have.
- Context loaded: Model claims about verification levels, principles (Provenance, Gate Over Remind), prior decisions (phased adversarial verification)

**Case B: Artifact Taxonomy Evolution** (`og-arch-design-artifact-taxonomy-14feb-ee48`)
- Worker skill: architect
- Question: "Should probes become the universal evidence-gathering primitive?"
- What happened: Worker evaluated 4 alternatives, designed migration strategy, and produced the key framing insight: "The artifact name IS the thinking tool — 'probe' demands 'probe WHAT?' while 'investigation' allows 'look into X.'" Correctly escalated final decision to orchestrator.
- Context loaded: Prior taxonomy work, existing probe/investigation infrastructure

**Case C: Dashboard Oscillation / Tmux Liveness** (`.kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md`)
- Worker skill: architect
- Question: "Two fixes didn't work. Is the entire approach fundamentally wrong?"
- What happened: Worker diagnosed that tmux liveness violated the "two-lane decision" (tmux owns presentation, not state). Recommended REVERTING two prior implementations and using phase-based liveness from beads comments instead. Cited "Coherence Over Patches" principle — 3 fix attempts = structural issue.
- Context loaded: Two-lane architectural decision, prior fix attempts

**Case D: Investigation Skill Diagnosis** (`og-inv-diagnose-investigation-skill-06jan-7b60`)
- Worker skill: investigation
- Question: "Why does the investigation skill have a 32% completion rate?"
- What happened: Worker reframed: it's a data quality artifact, not a skill quality issue. After filtering test spawns (71%) and skill/task mismatches (14%), true rate is ~91%. Classic premise challenge.
- Context loaded: Full workspace archive data, beads completion records

**Case E: Post-Mortem Analysis** (`og-inv-post-mortem-analyze-27feb-1b90`)
- Worker skill: investigation
- Question: "What caused communication breakdown across 3 sessions?"
- What happened: Worker performed CROSS-SESSION synthesis (reading 3 session transcripts totaling 2000+ lines), categorized 21 failures into 7 categories, evaluated whether the agreements system addresses root causes, and discovered that 5 of 7 failure categories are behavioral issues that agreements can't catch.
- Context loaded: 3 full session transcripts, agreement system design

**Case F: Spawn Prompt Quality Audit** (`.kb/investigations/2026-02-13-inv-audit-spawn-prompt-quality-vs-outcomes.md`)
- Worker skill: investigation
- Question: "What spawn prompt characteristics correlate with agent success?"
- What happened: Worker analyzed 227 spawns and made the strategic recommendation: "Don't fix manual spawns — strengthen the daemon path." Found 9.4x completion gap between daemon-routed (65.9%) vs manual (7.0%) and concluded workflow beats prompt structure.
- Context loaded: Full events.jsonl data, spawn metadata

**Case G: LIGHT/V2 Verification Conflict** (`.kb/investigations/2026-02-27-design-light-tier-v2-verification-conflict-resolution.md`)
- Worker skill: investigation
- Question: "Do tier and verification level systems fundamentally conflict?"
- What happened: Worker reframed: "Not a tier vs level choice, but completing an incomplete migration." V0-V3 system was designed to replace tier but migration wasn't finished. Explicitly rejected "enshrine tier as shadow authority."
- Context loaded: Both tier and verification level documentation

**Case H: Orchestrator Diagnostic Mode** (`.kb/investigations/2026-02-27-design-orchestrator-diagnostic-mode.md`)
- Worker skill: investigation
- Question: "How do orchestrators get time-limited code access without sliding into implementer behavior?"
- What happened: Worker identified "the slide pattern" with predictable trajectory (legitimate trace 0-5m → scope creep 5-10m → collapse 10m+). Designed 6 anti-slide safeguards. Key insight: "Coaching injection prevents slide better than time alone."
- Context loaded: Frame guard constraints, coaching plugin architecture

### Test 2: Worker Failures Traceable to Context, Not Capability

Searched for cases where workers failed at framing despite having proper context (which would suggest capability limits). Found **zero cases** where failure was attributable to reasoning inability rather than context gaps.

All failures found trace to one of:
- **Context not loaded**: Worker didn't have the relevant model/decision/investigation
- **Wrong context loaded**: Mis-scoped spawn (e.g., orch-go-dlw9: architect spawned in wrong repo, missing skillc context)
- **Skill/task mismatch**: Worker given investigation skill for implementation task (5 cases in diagnosis investigation)

**Notable non-failure**: The architect who placed skill tools in orch-go instead of skillc (orch-go-dlw9) — traced to 5-link failure chain where the orchestrator's repo-specific framing pre-committed the answer. The architect's REASONING was sound; the orchestrator's SCOPING was wrong.

### Test 3: Orchestrator Synthesis vs. Worker Synthesis Comparison

**Orchestrator synthesis examples examined:**
- Registry investigations synthesis (11 investigations → detected architectural drift)
- Daemon investigations synthesis (7 investigations → behavioral constraints for daemon)
- Model creation/updates (decidability-graph, orchestrator-session-lifecycle)

**Worker synthesis examples examined:**
- CLI synthesis (16 investigations → authoritative guide, skill: feature-impl)
- Status synthesis (12 investigations → incremental guide update, skill: kb-reflect)
- Post-mortem synthesis (3 sessions → 7-category taxonomy, skill: investigation)

---

## What I Observed

### Finding 1: Workers Handle Framing Questions Well When Context Is Adequate

All 8 cases above demonstrate workers successfully:
- Challenging premises (Cases D, G: reframing the question itself)
- Designing against cognitive failure patterns (Case H: the slide pattern)
- Rejecting plausible solutions for better ones (Case A: rejecting code review gate)
- Making strategic recommendations (Case F: "strengthen daemon path, not manual path")
- Performing cross-artifact synthesis (Case E: 3 sessions, 21 failures, 7 categories)

**No evidence of capability limits beyond context.** Workers consistently produced insights that were framing-level, not just factual, when given:
1. The model claims relevant to the question
2. Prior decisions and principles as substrate
3. Concrete data to test against (code, events, workspaces)

### Finding 2: Every Framing Failure Traces to Context, Not Capability

The orch-go-dlw9 case (architect in wrong repo) is the strongest test: the architect's reasoning about skillc tool placement was sound in every respect except that the orchestrator framed the question inside orch-go, implicitly pre-committing to that repo. The 5-link failure chain starts with "orchestrator framed question inside orch-go" — a context-scoping error.

### Finding 3: Orchestrator Synthesis IS Qualitatively Different, But Not Because of Reasoning

Orchestrator synthesis differs from worker synthesis in two structural ways:

**A. Aggregation position**: Orchestrators are the convergence point for multiple agents' outputs. The registry synthesis (11 investigations → drift detection) required seeing all 11 outputs simultaneously. No single worker is positioned to see this pattern because workers are scoped to individual tasks.

**B. Scope authorization**: Workers create nodes (issues), only orchestrators create blocking edges (dependencies). When a worker discovers "this problem blocks that epic," it can SURFACE the blocking relationship but cannot CREATE it. The authorization to change the work graph is structurally distinct from the reasoning to identify what should change.

Neither of these is about reasoning capability. Both are about structural position in the information flow.

### Finding 4: Worker Synthesis Can Match Orchestrator Quality When Given Equivalent Context

The CLI synthesis (16 investigations → guide, done by feature-impl worker) and post-mortem analysis (3 sessions → 7-category taxonomy, done by investigation worker) are synthesis work that matches orchestrator quality. The difference: orchestrators didn't need to be told "read these 16 investigations and synthesize" — they identified what NEEDED synthesizing. The scoping decision preceded the synthesis work.

---

## Model Impact

- [x] **Extends** model with: context-scoping plus two structural factors

### The extension:

The model claim is **correct as stated**: workers CAN answer framing questions if given the right context, and the hierarchy IS about context-scoping, not capability. 8 concrete cases confirm this with zero counter-examples of capability-limited failure.

However, the model should be extended to recognize that "context-scoping" encompasses **three distinct functions**, not one:

| Function | Description | Example |
|----------|-------------|---------|
| **Knowledge loading** | Deciding what models/decisions/investigations to include in spawn context | Architect gets decidability-graph model → can reason about authority boundaries |
| **Scope authorization** | Deciding what the agent can affect (create nodes but not blocking edges) | Worker discovers a blocking relationship but can only surface it, not create it |
| **Aggregation position** | Being the convergence point where multiple agents' outputs become visible | Orchestrator sees 11 investigation results simultaneously → detects drift pattern |

The current model collapses all three into "context-scoping." This is 80% right — knowledge loading is the most common bottleneck. But scope authorization and aggregation position are structurally irreducible to loading more context into a single agent.

**Specifically:**
- A worker given all 11 registry investigations COULD produce the same drift detection. But deciding those 11 are the relevant set IS the synthesis.
- A worker that identifies "X blocks Y" CAN produce the insight. But authorizing the blocking edge requires orchestrator scope (by design, not capability).

### Verdict: EXTENDS

The model's core claim holds: context-scoping, not capability, explains the hierarchy. But "context-scoping" should be decomposed into knowledge loading (delegatable), scope authorization (structural), and aggregation position (emergent from information flow). The first is what workers lack; the latter two are why orchestrators remain irreducible even when workers have full knowledge context.

---

## Evidence Index

| Case | Worker Skill | Framing Work | Context Quality | Outcome |
|------|-------------|--------------|-----------------|---------|
| A: Code Review Gate | investigation | Rejected premise, designed gate taxonomy | Rich (model + principles + decisions) | Excellent |
| B: Artifact Taxonomy | architect | Named "artifact name IS thinking tool" | Rich (prior taxonomy + infrastructure) | Excellent |
| C: Tmux Liveness | architect | Diagnosed structural wrongness, recommended revert | Rich (two-lane decision + prior fixes) | Excellent |
| D: Investigation 32% | investigation | Reframed as data quality artifact | Rich (workspace archive data) | Excellent |
| E: Post-Mortem | investigation | Cross-session synthesis, 7-category taxonomy | Rich (3 full transcripts) | Excellent |
| F: Spawn Quality | investigation | "Strengthen daemon, not manual" strategic rec | Rich (events data + spawn metadata) | Excellent |
| G: LIGHT/V2 Conflict | investigation | "Incomplete migration, not design conflict" | Rich (tier + level docs) | Excellent |
| H: Diagnostic Mode | investigation | "Slide pattern" cognitive failure model | Rich (frame guard + coaching) | Excellent |

**Failures examined:** 0 capability-limited failures found across 800+ workspaces. All failures trace to context gaps, mis-scoping, or skill/task mismatch.

**Orchestrator comparison:** Orchestrator synthesis is qualitatively different due to aggregation position and scope authorization, not reasoning capability. Workers given equivalent context produce equivalent quality.

---

## Notes

Prior probe (2026-02-09) tested whether orchestrator context-scoping is reducible to a deterministic pipeline. Found ~49% skill divergence in manual spawns, suggesting non-trivial scoping judgment. This probe goes deeper and confirms: the judgment IS the scoping, not the reasoning. Workers reason well; orchestrators scope well. Different jobs, same capability.
