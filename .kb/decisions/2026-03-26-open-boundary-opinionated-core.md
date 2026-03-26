# Decision: Open At The Boundary, Opinionated At The Core

**Date:** 2026-03-26
**Status:** Accepted
**Enforcement:** product-principle
**Deciders:** Dylan
**Extends:**
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`

## Decision

orch-go should be **open at the boundary** and **opinionated at the method core**.

This means:

- lower adoption risk by keeping integration points portable, inspectable, and non-captive
- preserve product distinctiveness by keeping the comprehension/coordination method sharp rather than endlessly configurable

The system should not try to own the entire stack. It should make the stack usable.

## Context

The accepted product boundary is that orch-go is primarily the thread/comprehension layer above unstable models, clients, and execution backends.

That creates a product design problem:

- if the system is too closed at the edges, it looks like another lock-in play
- if the system is too open at the center, it collapses into generic infrastructure

Because the product sits above volatile lower layers, users need confidence that they can adopt it without betting their entire future stack on one vendor, one client, or one execution path. At the same time, the product only matters if it has a real method for turning agent work into durable, legible understanding.

The correct asymmetry is:

- **openness at the edge**
- **discipline at the center**

## Boundary Definition

### Open At The Boundary

These should be open-source, inspectable, or highly portable by default:

- thread / brief / claim / decision artifact formats
- import/export pathways
- backend and provider adapters
- APIs for reading and writing knowledge artifacts
- integration surfaces for existing tools and repos
- research and evaluation outputs that build trust

These are the places where openness lowers fear and increases adoption.

### Opinionated At The Core

These should remain strongly shaped by the product's method:

- threads as the primary organizing artifact
- synthesis as the translation layer from output to understanding
- knowledge placement rules
- review / completion discipline
- explicit treatment of uncertainty, evidence, and unresolved questions
- the distinction between execution output and durable knowledge

These are the places where excessive configurability would dissolve the product into a toolkit.

## Product Implications

### Open-Source vs Held Back

#### Open by default

- artifact schemas
- portability and conversion tooling
- backend/provider adapters
- boundary APIs
- framework-neutral research artifacts

#### More defensibly held back

- the best integrated comprehension/review UX
- hosted collaborative knowledge surfaces
- higher-order routing/ranking/review intelligence
- productized method-enforcement systems

This is not a statement that these must be proprietary forever. It is a statement about where leverage is most likely to live if the product becomes real.

### Configurable vs Fixed

#### Configurable

- model/provider choice
- backend/client choice
- integration destinations
- notification/review preferences
- some team policy thresholds

These are infrastructure choices and local environment preferences.

#### Fixed or strongly defaulted

- thread-first organization
- synthesis expectations
- durable artifact placement
- review/completion discipline
- visibility of uncertainty

These are the epistemic discipline of the product and should not be flattened into "bring your own workflow" mush.

### Onboarding

Day-one adoption should be additive, not replacement.

Users should be able to:

- keep their current model/provider
- keep their current client or coding surface
- adopt the thread/comprehension layer on one repo or one stream of work
- inspect the produced artifacts directly
- feel immediate value through better synthesis, continuity, and legibility

Users should not need to:

- replace their full stack
- understand the full ontology before first value
- trust a black box

## Why This Decision

### 1. The product's layer benefits from fragmentation beneath it

The more volatile the model/client/execution landscape becomes, the more valuable a portable coordination/comprehension layer becomes.

### 2. Adoption risk is mostly boundary risk

Users fear lock-in at the infrastructure edges. That is where openness matters most.

### 3. Product distinctiveness lives in the method

If the core discipline becomes optional or infinitely configurable, the product loses the very thing that makes it non-generic.

## Failure Modes This Rejects

### Too closed at the edges

Symptoms:

- one-provider assumptions
- opaque artifact formats
- forced adoption of one client
- hidden or non-exportable knowledge

Result:

The system is interpreted as another ecosystem trap.

### Too open at the center

Symptoms:

- every workflow concept is optional
- threads/synthesis/review become arbitrary toggles
- the system becomes "generic agent platform + optional notes"

Result:

The system loses product shape and becomes replaceable plumbing.

## Practical Rule

When making product choices, ask:

1. Is this a boundary concern or a method concern?
2. If boundary: can we make it more portable, inspectable, and easier to adopt?
3. If core method: are we preserving clarity and discipline, or dissolving the product?

## Near-Term Pressure

This decision creates pressure for:

1. product positioning that emphasizes portability at the edges
2. artifact and API design that favors inspectability/export
3. onboarding that is additive to existing workflows
4. restraint around making the core method endlessly customizable

## Auto-Linked Investigations

- .kb/investigations/2026-03-26-inv-inventory-current-system-into-core.md
