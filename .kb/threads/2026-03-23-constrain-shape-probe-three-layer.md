---
title: "Constrain Shape Probe — three-layer verification model for composed systems"
status: open
created: 2026-03-23
updated: 2026-03-23
resolved_to: ""
---

# Constrain Shape Probe — three-layer verification model for composed systems

## 2026-03-23

Origin: while building connectivity probes for the LED totem sled (CSG intersection tests — zero facets = path clear), realized the probe methodology transferred directly from orch-go's knowledge probe system. Agent independently arrived at claim table, probe modules, gate runner, verdicts — same structure, different apparatus.

### The three layers

Three levels of verification for composed systems, investment priority in this order:

1. **Constrain** (gates) — eliminate impossible/wrong states. Before/during action.
2. **Shape** (attractors) — make correct states likely. During design.
3. **Probe** (composition verification) — verify claims about composed state. After composition.

Gates and attractors work on the action space. Probes work on the composed state. Each layer reduces the surface area the next layer needs to cover.

### What probing actually is

Not a new mechanism. It's integration testing with one specific practice: **name the emergent property explicitly as a claim before testing it**. Most systems fail at composition not because testing is hard, but because nobody wrote down what the composition should produce.

The LED sled had no connectivity table until we built the probes. "Cable path is continuous" was implicit. The knowledge models had no claims section until the probe system required one. The forcing function is the claim, not the test.

Sequence: (1) name the emergent property, (2) write it as explicit claim, (3) build test apparatus. Step 3 is standard integration testing. Steps 1-2 are where most systems fail.

### Transfer test (confirmed)

led-totem-toppers-bc0 built the connectivity gate. Agent independently produced claim table, probe modules, gate runner, verdicts — same structure as orch-go knowledge probes, different apparatus. Methodology transfers as a design template: "list your composition claims, build apparatus, check verdicts."

### Vocabulary, not theory

After stress-testing: this is a vocabulary contribution and design checklist, not a discovery. "Name your composition claims explicitly, then test them" is sound engineering advice. The vocabulary (claim/probe/verdict) is useful as a template when entering a new domain — it gives you a structure for asking "where will composition fail?" But it's not saying anything the testing pyramid doesn't already say.

### What's still useful

The template transfers. The agent built the connectivity gate without knowing about the parallel. When staring at a new domain asking "where will composition fail?", the checklist works: list claims, build apparatus, check verdicts. That's a useful design tool, not a theory.

### Subsumes/informs

- Reframes: compositional correctness gap (HE §8) — the gap is unnamed composition claims, not missing infrastructure
- Extends: coordination model — gate/attractor/probe as three mechanism types
- Informs: narrative-packaging thread — concrete hook (CSG probe rendering), but the story is smaller than initially thought
- Informs: architect skill — implications for what architects should produce (TBD)

## Auto-Linked Investigations

- .kb/investigations/2026-03-25-inv-investigate-antithesis-hegel-property-based.md
