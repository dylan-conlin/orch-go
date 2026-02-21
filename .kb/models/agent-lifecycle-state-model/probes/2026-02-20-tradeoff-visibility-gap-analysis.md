# Probe: Architectural Tradeoff Visibility Gap

**Model:** Agent Lifecycle State Model
**Date:** 2026-02-20
**Status:** Complete
**Beads:** orch-go-1158

## Question

The agent-lifecycle-state-model documents the 6-week registry drift cycle as a failure mode but frames it purely as a state management problem ("reconciliation is where every bug lives"). The model does NOT document the meta-failure: **the orchestrator never knew the tradeoff existed until after 6 weeks of debugging.** Does the model need a new invariant about tradeoff surfacing, or is this a gap in a different part of the system?

Specifically testing the model's claim: "Multiple sources must be reconciled - No single source has complete truth; query engine joins with reason codes." This is stated as an engineering invariant. But the deeper invariant — that architectural tradeoffs made by agents must be visible to the non-code-reading orchestrator — is absent from the model.

## What I Tested

### Test 1: Is the tradeoff documented anywhere the orchestrator would see it?

Searched all SYNTHESIS.md files, bd comment history, and investigation files for when the registry cache tradeoff was first introduced. The registry existed from initial commit (Dec 18 Go rewrite). The investigation `2025-12-21-synthesis-registry-evolution-and-orch-identity.md` documents the tradeoff beautifully: "Registry = fast but stale / Direct query = slower but always correct." But this investigation was produced *after* drift was already causing pain.

**Evidence:** The tradeoff was documented as a finding, not as a declared upfront choice. No SYNTHESIS.md or bd comment from the Dec 18-20 period says "chose registry cache pattern; risk: drift."

### Test 2: Does orch complete check for tradeoff documentation?

Read `pkg/verify/check.go` via the surfacing audit agent. `VerifySynthesis()` checks file existence and non-emptiness (`info.Size() > 0`). The Knowledge > Decisions Made section in SYNTHESIS.md is never parsed by any gate. The explain-back gate checks for non-empty text but has no structured field for "what was sacrificed."

**Evidence:** No verification gate reads or surfaces tradeoff content from SYNTHESIS.md to the orchestrator.

### Test 3: Does SPAWN_CONTEXT inject architectural constraints that would make tradeoffs visible?

The SPAWN_CONTEXT.md template (pkg/spawn/context.go) injects: task description, skill guidance, prior knowledge from `kb context`, constraints, prior decisions, and models. However, the injected model summaries don't include "pressure points" or "when this breaks" sections. Models include "Why This Fails" but this describes failure modes of the *model's domain*, not architectural tension points that feature requests might trigger.

**Evidence:** An agent spawned to "add agent caching for faster status" would receive the agent-lifecycle-state-model summary, which says "No persistent lifecycle caches" as a constraint. This constraint DOES appear in the PRIOR KNOWLEDGE injection. But prior to Feb 18 2026, this constraint didn't exist in the model — the model documented registry as an existing pattern, not as a banned one.

### Test 4: Do models include "pressure points" where feature requests conflict with architecture?

Read all 11 models in `.kb/models/`. None include a "pressure points" section. The closest is the "Constraints" section, which documents technical constraints (e.g., "Sessions go idle for many reasons"). But constraints describe *what is*, not *what would break if you changed it*. A "pressure points" section would say: "Adding any persistent cache will recreate the drift cycle — see 6-week history."

**Evidence:** Models document architecture but not architectural fragility. The "No Local Agent State" principle in principles.md is the closest to a pressure point, but it's a broad principle, not model-specific tension.

## What I Observed

1. **The tradeoff was well-documented — but only retroactively.** The Dec 21 synthesis identified "fast but stale vs slow but correct" as the core tension. But this was discovered through pain, not declared at design time.

2. **No mechanism exists to surface tradeoffs from worker → orchestrator at the moment of choice.** SYNTHESIS.md has a "Decisions Made" section, but no gate reads it. bd comments carry phase status, not architectural content. The explain-back gate checks comprehension of what was built, not what was sacrificed.

3. **The architect skill IS well-designed for tradeoff capture** — fork navigation with explicit `Trade-off accepted:` fields. But this only fires when an architect agent is spawned. Feature-impl agents, investigation agents, and other workers make tradeoffs silently.

4. **Models lack "pressure points" that would flag feature-architecture conflicts.** The model format has Summary, Core Mechanism, Why This Fails, Constraints, Evolution, References. None of these are designed to say "if you're asked to do X, the architectural cost is Y."

5. **Seven tradeoff classes recur in this codebase:** (1) cache vs direct query, (2) speed vs correctness for bulk ops, (3) simplicity vs completeness for state, (4) velocity vs verification, (5) spawn mode selection, (6) persistence boundary dependencies, (7) deduplication strategies. Of these, #1 and #4 caused the most damage and were documented *after* the damage.

## Model Impact

**EXTENDS the model.** The agent-lifecycle-state-model accurately describes the four-layer architecture and the registry elimination. But it's missing a critical dimension: **the model documents what happened but not how the orchestrator could have known earlier.**

Proposed extension: Models should include a **"Pressure Points"** section that names the specific feature requests or operational changes that would violate the model's invariants. This would transform models from "here's how it works" to "here's how it works AND here's what breaks it."

The model's invariant #7 ("No persistent lifecycle caches") IS a pressure point, but it was added in Feb 2026 after the damage. The question this probe raises: **how do pressure points get into models before the cycle plays out?**

This is not a failure of the agent-lifecycle-state-model specifically — it's a gap in the model format itself across all 11 models.
