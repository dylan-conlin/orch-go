# Decision: Thread/Comprehension Layer Is The Primary Product

**Date:** 2026-03-26
**Status:** Accepted
**Enforcement:** strategic-direction
**Deciders:** Dylan
**Extends:**
- `.kb/decisions/2026-02-28-atc-not-conductor-orchestrator-reframe.md`
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`

## Decision

`orch-go` is primarily the **thread/comprehension/coordination layer**, not the execution layer.

Execution remains necessary, but it is substrate. The product's center of gravity is the system that turns agent work into durable, legible understanding:

- threads as the primary organizing artifact
- synthesis and briefs as the async comprehension surface
- claims/models/decisions as structured knowledge accumulation
- routing/review/verification as support for trustworthy understanding

The product should be positioned and evolved around this layer, not around owning model backends, session transport, or agent client UX.

## Context

The project was discovered through building, not pre-specified from the start. That produced real capability but also strategic sprawl:

- README and architecture docs still describe orch-go primarily as an orchestration CLI
- the dashboard is still dominated by work graph and execution monitoring views
- large amounts of code exist in spawn/daemon/verify/backend plumbing
- recent threads identify a different center: what matters is not merely spawning agents, but making what agents learn compound over time

Two recent threads clarified the shift:

1. **Threads as primary artifact** — threads are the spine; work exists in service of questions, not the other way around
2. **Research extraction / OpenClaw migration** — execution plumbing is portable and increasingly commoditized; the differentiated value is above it

This means the project has moved from search to consolidation. The question is no longer "what are we building?" in the broad sense. The question is "which layer is core, and which layers are enabling substrate?"

## Product Boundary

### Core

These are primary investments and should define the product story:

- thread graph / thread-centric reading surfaces
- synthesis, briefs, and comprehension queue
- knowledge placement and artifact attractors
- claims, models, and structured accumulation of learned knowledge
- routing, completion, and verification where they improve trust in understanding
- human control surfaces that answer:
  - what changed?
  - what was learned?
  - what remains open?

### Substrate

These are necessary but not identity-defining:

- spawn plumbing
- model/backend routing
- session transport and execution clients
- Claude Code / OpenClaw / OpenAI / other backend integrations
- dashboard surfaces whose purpose is execution observability rather than comprehension

The system should be open and portable at this layer. Dependency on any one execution path is a risk, not a moat.

### Adjacent But Separable

These are valuable, but should not define the product center:

- coordination benchmark / empirical research harness
- model benchmarking and provider strategy work
- lower-level execution experiments and platform migration studies

Where possible, these should be extracted or treated as adjacent assets rather than allowed to dominate core product scope.

## Why This Decision

### 1. This is where the differentiated value actually is

Many tools can run agents. Few systems treat agent-produced knowledge as a first-class, composable resource. That is the strongest non-generic idea in orch-go.

### 2. Lower layers are unstable

Model leadership changes. Client quality changes. Access policies change. Session transports and wrappers change. The layer above them becomes more valuable precisely when the lower layers are volatile.

### 3. Current sprawl is a discovery artifact, not a reason to preserve equal investment

The execution-heavy system was the scaffolding that revealed the stronger product thesis. That scaffolding earned the insight, but it does not automatically deserve to remain the product identity.

### 4. This gives a deletion criterion

Future work can be evaluated by a simple question:

**Does this make the system better at turning agent output into durable, legible understanding?**

If yes, it is likely core. If not, it must justify itself as substrate.

## Implications

### Product

- Reframe top-level docs and positioning around coordination + comprehension
- Stop describing orch-go primarily as an orchestration CLI
- Treat thread-centric UX as the destination, not a side feature

### Architecture

- Keep the execution layer portable and replaceable
- Avoid over-owning backend/client infrastructure unless it is strategically necessary
- Prefer seams that let the methodology survive provider/client churn

### Prioritization

Bias new investment toward:

- thread surfaces
- synthesis delivery and review flows
- comprehension queue ergonomics
- knowledge composition
- legibility of learning and uncertainty

Raise the bar for work that primarily improves:

- spawn plumbing
- execution-specific dashboard surfaces
- backend-specific optimizations

### Positioning

The project should increasingly be described as:

- a coordination and knowledge layer for multi-model agent work
- a system that makes agent output compound into understanding
- a portable control plane above volatile models and clients

Not primarily as:

- a spawn CLI
- a model router
- an execution backend

## What This Does Not Mean

- Execution plumbing can be deleted immediately
- orchestration and verification stop mattering
- backend migrations are irrelevant

It means those concerns are subordinate to the primary product goal. They are enabling layers, not the center of identity.

## Follow-Through

This decision creates pressure for four follow-on moves:

1. Rewrite the top-level story (README, architecture overview, product framing)
2. Promote thread-first product surfaces in the UI and workflow
3. Map current systems into keep / substrate / adjacent categories
4. Use this boundary to decide what to de-emphasize, extract, or stop building

## Auto-Linked Investigations

- .kb/investigations/2026-03-26-inv-rewrite-readme-around-thread-comprehension.md
