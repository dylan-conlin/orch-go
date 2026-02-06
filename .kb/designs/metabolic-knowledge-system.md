# Metabolic Knowledge System

**Status:** Conceptual design
**Date:** 2026-02-05
**Context:** Emerged from retrospective on orch experiment - the system captured knowledge but couldn't curate it. Capture without curation = accumulation = noise.

---

## Core Concept

Knowledge has a lifecycle. It enters, gets processed, proves its value (or doesn't), consolidates with related knowledge, and eventually fades. The system actively manages this lifecycle rather than just accumulating.

**Design principle:** The system should get denser, not bigger. Mass converts to density through compression.

**The insight:** A knowledge system needs metabolism, not just memory.

| Function | What it does |
|----------|--------------|
| Intake | Knowledge enters |
| Digestion | Raw → extractable insight |
| Absorption | Proven knowledge → working set |
| Consolidation | Many pieces → fewer, denser pieces |
| Expulsion | Knowledge leaves the system |

The orch experiment had intake and partial retrieval. It lacked digestion, consolidation, and expulsion. Everything stayed in the stomach.

---

## Knowledge States

```
                    ┌─────────────────────────────────────────┐
                    │                                         │
                    ▼                                         │
┌───────┐    ┌──────────┐    ┌──────────┐    ┌───────────┐   │
│ INTAKE│───▶│   RAW    │───▶│ VALIDATED│───▶│  ACTIVE   │───┤
└───────┘    └──────────┘    └──────────┘    └───────────┘   │
                  │               │               │          │
                  │               │               │          │
                  ▼               ▼               ▼          │
             ┌─────────────────────────────────────────────┐ │
             │              ARCHIVED                       │ │
             └─────────────────────────────────────────────┘ │
                                                             │
                              ┌───────────┐                  │
                              │ CANONICAL │◀─────────────────┘
                              └───────────┘
                              (consolidated)
```

| State | Meaning | Location |
|-------|---------|----------|
| **Raw** | Just captured, unverified | Working set (temporary) |
| **Validated** | Cited or used, confirmed useful | Working set |
| **Active** | Regularly consulted | Working set |
| **Canonical** | Authoritative on topic, consolidated from many | Working set |
| **Archived** | Preserved but not active | Archive (out of working set) |

---

## Stage 1: Intake

**What it is:** Knowledge enters the system.

**Trigger:** Something is learned - a failure, a discovery, a decision made, a question answered.

**Process:**
```
Event occurs
    │
    ▼
Capture with minimal friction
    │
    ▼
Extract the nugget (REQUIRED)
    │
    ▼
Raw artifact created
```

**The Nugget Requirement:**

Every capture must include a single transferable sentence. Not a summary - the *portable insight* someone else would need.

| Artifact Type | Nugget Example |
|---------------|----------------|
| Investigation | "OpenCode SSE events don't include session state - must poll separately" |
| Decision | "Chose polling over websockets because SSE already saturates connection limit" |
| Constraint | "Never run bd close from workers - bypasses verification" |

**Output:** Raw artifact with extracted nugget.

**Health signals:**
- Capture happens in-the-moment (not reconstructed later)
- Nuggets are useful when retrieved in isolation
- Friction is low enough that capture actually happens

---

## Stage 2: Validation

**What it is:** Knowledge proves it's useful (or doesn't).

**Trigger:** Time passes. Knowledge either gets used or doesn't.

**Process:**
```
Raw artifact exists
    │
    ├──▶ Gets cited by other knowledge ──▶ VALIDATED
    │
    ├──▶ Gets retrieved and applied ──▶ VALIDATED
    │
    ├──▶ N days pass, no citations ──▶ Flag for review
    │
    └──▶ Found to be wrong ──▶ ARCHIVED (or deleted)
```

**Validation signals:**
- Citation by another artifact
- Retrieved via query and actually used
- Referenced in a decision or commit

**Output:** Status change (Raw → Validated) or flag for review.

**Health signals:**
- Valuable knowledge gets validated naturally
- Wrong knowledge gets caught and removed
- No manual "approval" process needed

---

## Stage 3: Activation

**What it is:** Validated knowledge becomes part of the working set that's actively consulted.

**Trigger:** Knowledge proves repeatedly useful.

**Process:**
```
Validated artifact
    │
    ├──▶ Cited 3+ times ──▶ ACTIVE
    │
    ├──▶ Retrieved repeatedly ──▶ ACTIVE
    │
    └──▶ Remains cited 1-2 times ──▶ Stay VALIDATED
```

**What "Active" means:**
- Loaded into context by default (for relevant queries)
- Part of the working set an agent/human consults
- Higher retrieval priority than Raw/Validated

**Output:** Knowledge promoted to Active status.

**Health signals:**
- Working set is actually consulted
- Active knowledge is correct (not contradicted by experience)
- Promotion happens through use, not manual curation

---

## Stage 4: Consolidation

**What it is:** Many related pieces become one denser piece.

**Trigger:** Topic density exceeds healthy threshold.

**Signals that trigger consolidation:**
- Query returns >5 results on same topic
- 5+ artifacts share similar nuggets
- Contradictions detected between artifacts
- Someone asks a question that N artifacts partially answer

**Process:**
```
Consolidation triggered
    │
    ▼
Cluster identified (related artifacts)
    │
    ▼
Synthesis work (human or agent)
    │
    ▼
Canonical artifact produced
    │
    ▼
Original artifacts marked "absorbed into [canonical]"
    │
    ▼
Originals moved to ARCHIVED
```

**Canonical artifact types:**

| Source | Becomes |
|--------|---------|
| Multiple investigations on same system | Model |
| Multiple scattered decisions | Architectural pattern |
| Multiple constraints on same topic | Principle |
| Multiple "how to" captures | Guide |

**Output:**
- One Canonical artifact in working set
- N original artifacts in archive (with links preserved)
- Net reduction in working set size

**Health signals:**
- Queries return fewer, more relevant results after consolidation
- New work cites Canonical artifact (not scattered originals)
- Working set stays bounded despite ongoing capture

---

## Stage 5: Expulsion

**What it is:** Knowledge leaves the working set (and possibly the system entirely).

**Triggers:**

| Condition | Action |
|-----------|--------|
| Uncited for N days | Flag → Review → Archive |
| Superseded by consolidation | Archive (preserve for provenance) |
| Contradicted by newer knowledge | Archive with "superseded by X" |
| Found to be wrong | Delete (not archive - remove entirely) |

**Process:**
```
Expulsion triggered
    │
    ├──▶ Superseded/Stale ──▶ ARCHIVED (retrievable)
    │
    └──▶ Wrong ──▶ DELETED (gone)
```

**Archive vs Delete:**
- **Archive:** Knowledge was valid when created but is now superseded or unused. Keep for provenance, historical queries.
- **Delete:** Knowledge was wrong. Remove entirely to prevent misleading future queries.

**Output:**
- Working set gets smaller
- System "forgets" appropriately

**Health signals:**
- Working set doesn't grow indefinitely
- Old knowledge doesn't mislead
- Archival rate roughly matches intake rate (steady state)

---

## The Working Set

At steady state, the working set contains knowledge across states. The specific counts emerge from:

- **Context budget** - How much fits in agent context
- **Retrieval quality** - How many results are useful before noise dominates
- **Citation patterns** - What actually gets used

**The counts aren't the design. The metabolic process is the design. The counts are tuning parameters you discover by running the system.**

Everything outside the working set is archived - retrievable if needed, but not actively loaded or consulted.

---

## Health Metrics

How you know the system is working:

| Metric | Healthy | Unhealthy |
|--------|---------|-----------|
| Query relevance | 1-3 highly relevant results | 10+ results, unclear which matters |
| Citation rate | >50% of Raw → Validated | Most Raw never cited |
| Consolidation rate | Clusters get synthesized | Same topic fragments endlessly |
| Contradiction rate | Contradictions detected and resolved | Contradictory knowledge coexists |
| Working set growth | Bounded (intake ≈ expulsion) | Grows indefinitely |
| Retrieval utility | Retrieved knowledge gets applied | Retrieval feels useless |

---

## What's NOT in This Design

Deliberately omitted (implementation decisions, not design decisions):

- **Specific thresholds** - These emerge from running the system
- **Implementation details** - File formats, CLI commands, storage
- **Integration with orchestration** - This is the knowledge system alone
- **Automation level** - How much is human vs agent vs automatic

---

## Open Questions

1. **Who does consolidation?** Human? Agent? Triggered automatically but executed manually?

2. **What's the archive?** A separate directory? A status flag? How long retained?

3. **How are citations tracked?** Explicit links? Grep for references? Something smarter?

4. **What's the nugget format?** Free text? Structured? Tagged?

5. **How does retrieval change by state?** Does Active rank higher than Validated? How?

---

## Provenance

This design emerged from a retrospective on the orch experiment (Oct 2025 - Feb 2026):

- **What worked:** Capture (kb quick, kb create), retrieval (kb context), principles emerged from friction
- **What failed:** No lifecycle, no curation, 735 investigations accumulated, 50% deemed unnecessary
- **Core insight:** The system was a filing cabinet (memory only). A healthy system needs metabolism (intake, digestion, absorption, consolidation, expulsion).

Related artifacts:
- `.kb/investigations/2026-02-04-inv-what-made-orch-feel-like-job.md`
- `.kb/investigations/2026-02-04-inv-analyze-94kb-orchestrator-skill-claude.md`
- `.kb/investigations/2026-02-04-inv-analyze-checkpoint-rituals-session-start.md`
- `~/.kb/principles.md`
