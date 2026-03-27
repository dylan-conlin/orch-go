# Probe: Brief Composition as Clustering Stage Instance

**Date:** 2026-03-26
**Model:** Signal-to-Design Loop
**Status:** Complete
**Investigation:** `.kb/investigations/2026-03-26-design-brief-composition-layer.md`

## Question

Does brief composition fit cleanly as a Stage 3 (clustering) instance of the signal-to-design loop? The model claims clustering requires "explicit, machine-readable clustering key" and ranks resolution: "explicit tag > threshold count > lexical proximity." Briefs have no explicit tags — does the loop still work?

## What I Tested

1. **Examined 61 briefs** in `.kb/briefs/` — total 943 lines. Checked for structural metadata, tags, categories. Found: no clustering-friendly metadata beyond the Frame/Resolution/Tension structure.

2. **Verified 4 manually-identified clusters** from today's orchestrator session against the model's clustering requirements:
   - Identity gap cluster (8 briefs): orch-go-pityd, orch-go-wgkj4, orch-go-ey4py, orch-go-bw0y6, orch-go-vo51p, orch-go-z4h7s, + 2 others
   - Epistemic dishonesty cluster (5 briefs): orch-go-o5uih, orch-go-k6c0v, orch-go-fsikn, orch-go-n4uwb, orch-go-z1pkh
   - Model-routing-as-boundary (6 briefs)
   - Production-exceeding-comprehension (3 briefs)

3. **Tested clustering resolution hierarchy** against brief content:
   - Explicit tag: NOT AVAILABLE — briefs have no tags
   - Threshold count: PARTIALLY AVAILABLE — same words appear across cluster members ("identity," "substrate," "16%/72%")
   - Lexical proximity: AVAILABLE — shared terminology, shared reference to same decision/thread

4. **Compared Tension sections** across cluster members to test if tensions cluster better than frames.

## What I Observed

### Observation 1: Briefs are a novel signal type — they cluster differently than investigations

The model's existing instances (defect-class → named agreements, investigation → skill redesign) use **explicit metadata tags** for clustering. Briefs have no tags. But briefs have something the other signal types don't: a **constrained Tension section** that surfaces unresolved questions.

When I compared Frame sections vs Tension sections for clustering quality:
- **Frame clustering:** High lexical overlap within identity-gap cluster ("product decision," "comprehension layer," "substrate," "execution plumbing"). But also high false-positive risk — many briefs mention "comprehension" without being in the same cluster.
- **Tension clustering:** Lower lexical overlap but higher semantic precision. The identity-gap tensions all ask variations of "will changing the framing actually change behavior?" The epistemic-dishonesty tensions all ask "is this looseness intentional or unfinished?" These are recognizably the same question from different angles.

### Observation 2: The model's resolution hierarchy holds but the ranking inverts for composition

The model says: explicit tag > threshold > lexical proximity. For brief composition:
- Lexical proximity on Tension sections is actually **more useful** than keyword threshold on Frame sections
- The reason: Frame sections describe what happened (diverse vocabulary), but Tension sections ask what's unresolved (convergent vocabulary)

This doesn't contradict the model — it extends it. The model's hierarchy assumes generic signals. Briefs are structured signals with a section (Tension) that naturally converges when briefs share an underlying gap.

### Observation 3: Composition is Stage 3, but with a Stage 4 boundary condition

The model says Stage 4 (synthesis) "requires authority — someone must decide 'this cluster means something and warrants design.'" The comprehension artifacts thread adds a constraint: synthesis must not FEEL complete, or it kills the reactive moment.

This means composition must do Stage 3 (form clusters) but must STOP before completing Stage 4 (synthesize meaning). The digest artifact format — clusters with draft proposals, not summaries — is structurally designed to stop at this boundary. The epistemic label makes the boundary explicit.

### Observation 4: Missing capture metadata would improve clustering

The model's "Failure Mode 2: Clustering Resolution Too Low" says: "Clustering relies on natural language similarity (lexical proximity) instead of explicit metadata. The fix: explicit, enumerated clustering keys."

This predicts that keyword-only clustering will eventually be too noisy. The design should anticipate adding a lightweight clustering field to briefs — perhaps a `relates_to:` field in the brief template that references thread slugs. This would move from lexical proximity to explicit tag, the model's preferred resolution.

## Model Impact

### Confirmed
- **Stage structure holds.** Brief composition maps cleanly to Stage 3 (clustering). The five-stage loop accurately describes the gap: Stages 1-2 are live (briefs generated and accumulated), Stage 3 is missing (no clustering), so Stages 4-5 can't fire systematically.
- **Failure Mode 2 predicts the risk.** Without explicit metadata, lexical clustering will produce false positives. The model correctly identifies this as the main quality risk.
- **Failure Mode 3 applies.** "Synthesis Without Authority" — if the digest is generated but no one acts on it, it's a dashboard effect. The session-start integration addresses this by putting the digest in the orchestrator's conversation.

### Extended
- **Tension sections as natural clustering key.** The model doesn't account for structured signal types where one section (Tension) naturally converges while another (Frame) is diverse. This is a new finding: when signals have a "what's unresolved" section, that section clusters better than the "what happened" section. The resolution hierarchy should note: "For structured signals with unresolved-question sections, cluster on the question, not the narrative."
- **Stage 3/4 boundary as a design parameter.** The model treats clustering and synthesis as sequential stages. The comprehension design reveals that the BOUNDARY between them is a design choice — how much of Stage 4 to include in the automated step is a function of trust in the automated actor and the value of human participation. This is a new axis the model doesn't describe.

### No contradictions found.
