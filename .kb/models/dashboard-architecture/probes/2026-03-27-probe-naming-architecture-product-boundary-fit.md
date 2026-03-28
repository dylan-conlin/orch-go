# Probe: Naming Architecture — Does Name Structure Fit Product Boundary?

**Date:** 2026-03-27
**Model:** dashboard-architecture
**Status:** Complete
**Question:** Does the prior naming investigation's recommended architecture (separate product name from repo/CLI, Kenning as candidate) actually fit the product boundary as defined by the five-elements thread and comprehension-first decision?

## What I Tested

Testing three claims from the prior investigation (orch-go-sg39y) against the product boundary substrate:

1. **Claim: "Kenning" is the right semantic territory** — Does a name from the "understanding/knowledge" category best match a product whose identity is five elements (threads, briefs, tensions, shape, resistance)?

2. **Claim: Product name should separate from repo/CLI name** — Is this the right architecture for a product that is still pre-v1 and has zero external users?

3. **Claim: The name should evoke comprehension/synthesis/continuity** — Does the five-elements thread change what the name should evoke? The product is now "a thinking surface," not just "comprehension."

## What I Observed

### Observation 1: Kenning's Semantic Fit Strengthened by Five-Element Evolution

The product's identity evolved from "comprehension layer" (product decision, Mar 26) to "thinking surface with five elements" (thread, Mar 27). The five elements are:
1. What you're thinking about (threads)
2. What was learned (briefs)
3. What's still open (tensions)
4. What kind of move is this? (shape classification)
5. Is this working or fighting you? (resistance signal)

Tested whether "kenning" (compound metaphor / "to know, to perceive") still fits this expanded identity:

- **Elements 1-3** ("what do I know?"): Kenning's etymology ("to know") maps directly. Confirmed.
- **Elements 4-5** ("what should I do?"): Kenning's compound-metaphor meaning (composing simpler concepts into richer understanding) maps to the synthesis operation, but shape-classification and resistance are more about *perception* than *composition*. Kenning's secondary meaning ("to perceive, to recognize") actually covers this. The Old Norse *kenna* means both "to know" and "to perceive/recognize."

The five-element evolution doesn't weaken the name — it actually reveals that kenning's double meaning (know + perceive) maps to the product's double function (know + decide).

### Observation 2: The Premise "This Product Needs a Name Now" Deserves Testing

Applied the Premise Before Solution principle. The question "What should we name this?" presupposes naming is the right next action. Tested:

**Evidence against naming now:**
- Zero external users. The first public touchpoint is a blog post (wedge investigation), not a product launch.
- The method guide (must-ship per release bundle) is Phase 3 of the consolidation plan. The content-first surface is Phase 1. Naming for the method guide is premature if Phase 1 isn't done.
- Domain/package names are perishable assets, but kenning's registries are unclaimed (confirmed by fresh collision check: kenning.sh, kenning.so, npm, Go all clear, Antmicro still at 91 stars). The window isn't closing.

**Evidence for naming now:**
- The blog post (best wedge) needs to reference *something*. "The orch-go system" reinforces the orchestration framing that the product decision explicitly rejects.
- Dylan's narrative packaging thread shows he needs a handle to tell the story. Mechanism descriptions ("multi-agent orchestration with comprehension queues") don't land. A name provides the hook.
- A working name, even unproven, gives the project identity coherence while the surfaces are being built.

**Conclusion:** The product needs a *working name* now, not a *locked name*. The distinction matters: a working name can appear in blog drafts, internal docs, and conversations. It doesn't need domain registration, package claims, or README rewrites until the method guide ships.

### Observation 3: The Naming Architecture Has Three Layers, Not Two

The prior investigation identified product name vs repo/CLI name. But the product boundary decision and five-elements work reveal a third layer:

| Layer | Example | When it matters | Change cost |
|-------|---------|-----------------|-------------|
| **Method name** | "Kenning" (the approach) | Blog post, method guide, talks | Low — it's a word in text |
| **Product name** | "Kenning" (the tool) | README, website, package registry | Medium — needs consistency |
| **Code name** | orch-go / orch | Go imports, CLI binary, scripts | High — 392 files + muscle memory |

The prior investigation treated method name and product name as one thing. But they can diverge. The method (how you think about agent work) might have a different name than the tool (what you install). Example: "Getting Things Done" is the method, "OmniFocus" is the tool.

This opens a fourth option the prior investigation didn't consider:

**Option D: Name the method, defer naming the product.**
- The blog post introduces the *method*: "Kenning: compound understanding from AI agent work"
- The tool stays `orch` for now. The method guide describes the approach, links to `orch-go` as the reference implementation
- If the method name sticks, the product can adopt it later. If not, the method was still named correctly

This is lower-risk than Option B→C because it doesn't conflate the method's reputation with the tool's adoption trajectory.

### Observation 4: Collision Status Reconfirmed (Fresh Check)

Ran fresh collision check against all Kenning registries:

| Asset | Status (original) | Status (today) | Changed? |
|-------|-------------------|----------------|----------|
| Antmicro kenning (GitHub) | 91 stars | 91 stars | No |
| kenning.sh | Available | Available | No |
| kenning.so | Available | Available | No |
| npm `kenning` | Available | Available | No |
| Go `kenning` | Available | Available | No |
| PyPI `kenning` | Unclear | 404 (may be Antmicro's; needs manual check) | Ambiguous |
| GitHub `kenning` user | N/A | Taken (personal user, not product) | New info |

The namespace window is not closing. No competitive pressure to register immediately.

## Model Impact

### Extends Dashboard Architecture Model

The dashboard-architecture model now has two rendering modes (metadata vs content) and a product identity triangle (threads, briefs, tensions). This probe adds a **naming architecture** dimension:

**New claim:** The product has three naming layers — method name, product name, and code name — that can diverge without conflict at v1. The method name is the first to face external audiences (via blog post and method guide). The product name faces external audiences only when the tool ships. The code name faces only developers who build or contribute. Treating all three as one name creates unnecessary coupling and premature commitment.

### Confirms Minimum Comprehension Surface Finding

The comprehension surface probe found the product feel comes from content-first rendering. This naming probe confirms that the *verbal* identity should also be content-first: name the method (what you do) before naming the tool (what you install). This mirrors the surface finding: show the text before showing the counts.

### Extends Product Boundary Decision

The product boundary decision says "stop describing orch-go primarily as an orchestration CLI." The naming architecture provides the positive replacement: describe the *method* by name, reference the *tool* as its implementation. This is more actionable than "don't say orchestration" because it gives writers a word to use instead.
