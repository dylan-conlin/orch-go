# Core Method Spec

**Purpose:** Define the parts of orch-go's method that are sacred, strongly defaulted, and negotiable. This is the operational companion to the product-boundary decisions.

**Related decisions:**
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`
- `.kb/decisions/2026-03-26-open-boundary-opinionated-core.md`

---

## Summary

orch-go is not just a system for running agents. It is a system for converting agent work into durable, legible understanding.

That requires a method, not just infrastructure.

Some parts of the system should be flexible so adoption is low-risk:
- models
- backends
- clients
- integrations

Other parts must remain strongly shaped, or the product collapses into a generic toolkit:
- threads
- synthesis
- knowledge placement
- review/completion discipline
- epistemic status
- distinction between execution output and durable knowledge

This guide defines that boundary.

---

## Sacred

These are core to the product's identity. If they are removed or made fully optional, the system stops being itself.

### 1. Threads Are The Primary Organizing Artifact

Work belongs to a thread, question, or line of thought.

Why:
- tasks and chats are execution-native structures
- threads provide continuity across runs, sessions, and artifacts
- they give learning a spine

Without this:
- the system regresses into issue tracking plus agent logs
- work accumulates, but understanding does not

### 2. Synthesis Is Required To Translate Output Into Understanding

Raw output is not equivalent to comprehension.

Synthesis exists to answer:
- what changed in our understanding?
- what emerged from the work?
- what matters now?
- what follows next?

Why:
- agents produce outputs naturally
- humans need translation, not just accumulation

Without this:
- the system becomes artifact-rich and orientation-poor

### 3. Durable Knowledge Must Be Structurally Placed

The system must have explicit rules for where knowledge goes:
- thread
- decision
- claim/model
- workspace-local artifact
- ephemeral output

Why:
- agents externalize by default
- without attractors, valuable knowledge ends up scattered or lost

Without this:
- the project fills with markdown and transcripts that do not compose

### 4. Completion Means More Than "Something Was Produced"

Completion includes:
- execution result
- verification or evidence
- clear epistemic status
- correct placement of what matters

Why:
- output alone is not trustworthy understanding

Without this:
- the system rewards motion over learning

### 5. Uncertainty Must Be First-Class

The system must preserve:
- unresolved questions
- contested claims
- confidence levels
- partial evidence
- explicit uncertainty

Why:
- compressing uncertainty into false closure destroys trust
- multi-agent systems overstate understanding unless uncertainty has structure

Without this:
- the system looks confident faster than it becomes reliable

### 6. Execution Output And Durable Knowledge Are Different Things

Execution output is what happened.
Durable knowledge is what should still matter later.

Examples of execution output:
- patches
- raw notes
- transcripts
- temporary run artifacts

Examples of durable knowledge:
- validated constraints
- decisions
- clarified tensions
- updated claims
- thread movement

Why:
- not everything produced should be promoted
- not everything learned should remain buried in output artifacts

Without this:
- the KB becomes noisy, or the system fails to compound

---

## Strong Defaults

These are not metaphysically sacred, but changing them should require strong justification.

### Thread-First Reading Surface

The primary human-facing surface should increasingly be thread-centric rather than execution-centric.

### Briefs/Synthesis As The Async Comprehension Surface

The default expectation is that important work gets translated into concise comprehension artifacts, not just preserved as raw evidence.

### Evidence Before Promotion

Promotion from output to durable knowledge should require enough support to justify it.

### Review As Epistemic Quality Control

Review is not only about correctness. It is also about:
- legibility
- placement
- epistemic status
- what the system should now believe

### Method Before Throughput

If there is tension between raw speed and preserving cumulative understanding, the default bias should be toward preserving understanding.

---

## Negotiable

These can vary without dissolving the product.

### Infrastructure Choices

- model provider
- execution backend
- client surface
- API integrations
- notification channels

### Policy Tuning

- review cadence
- some gate thresholds
- team-specific routing preferences
- operational defaults for different environments

### Presentation Layer Variations

- dashboard layout
- summary density
- visualization style

These can change as long as they continue to express the core method rather than bypass it.

---

## Dangerous Configurability

These are the kinds of flexibility requests that sound harmless but would hollow out the product.

### "Can threads be optional?"

Danger:
- falls back to flat work tracking
- destroys continuity

### "Can synthesis be skipped by default?"

Danger:
- produces outputs without understanding
- creates accumulation without orientation

### "Can teams define their own placement rules freely?"

Danger:
- removes common knowledge geometry
- makes artifacts non-composable across users and time

### "Can completion just mean artifact exists?"

Danger:
- rewards generation, not learning
- weakens trust in what the system says is done

### "Can uncertainty/evidence tracking be hidden or turned off?"

Danger:
- encourages false certainty
- makes the system look cleaner while becoming less honest

---

## Practical Test For Future Decisions

When considering a new feature, customization, or simplification:

### Step 1: Is this boundary or core?

- Boundary: portability, integration, client/backend flexibility
- Core: how work becomes understanding

### Step 2: If boundary, optimize for adoption

Ask:
- does this lower lock-in risk?
- does this improve portability or inspectability?
- does this make adoption more additive?

### Step 3: If core, optimize for method integrity

Ask:
- does this preserve the epistemic discipline?
- does this make understanding more cumulative?
- does this keep the product from dissolving into a toolkit?

---

## Short Version

Open the infrastructure edges.

Keep the comprehension method sharp.

The product wins when users can bring their own models and tools, but cannot accidentally remove the discipline that makes work cumulative.
