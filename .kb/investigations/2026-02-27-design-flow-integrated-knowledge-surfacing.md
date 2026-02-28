---
status: open
type: design
date: 2026-02-27
triggers: ["session conversation about flow state, model drift, and verification trust"]
---

**Status:** Active

# Design: Flow-Integrated Knowledge Surfacing

## Problem Statement

The orchestration system externalizes understanding into model files, decisions, and knowledge artifacts. But the human operator (Dylan) rarely reads these directly. His mental model is maintained tacitly — through the flow of work (spawning, reviewing completions, making decisions) — not through explicit study.

This creates a layer gap: knowledge exists in the system but doesn't reach the human at the moments they're cognitively engaged. The model files serve agents well (via `kb context`) but don't serve the human directly.

Meanwhile, the system has no awareness of its own throughput patterns, no way to calibrate trust in verification, and verification itself exists as a binary: inspect everything (expensive, breaks flow) or rubber-stamp (cheap, entropy).

## Core Insight

**Flow state is the scarce resource.** Human attention sustains or collapses the entire system. Verification isn't overhead — it's the mechanism that maintains flow. When verification is skipped, mental models drift from reality, decisions degrade, and the spiral begins. When it works, it creates an upward loop: verify → understand → good decision → connected work → flow → want to verify.

The system should surface knowledge *through* the flow, not outside it. And it should help calibrate trust so verification effort is proportional to risk, not uniform.

## Four Design Threads

### 1. Model Surfacing at Engagement Moments

The system has natural moments where the human is already cognitively engaged. These are integration points where knowledge should surface without requiring a context switch:

| Moment | What surfaces | Source |
|--------|--------------|--------|
| **Session start (orientation)** | 2-3 sentence model summaries relevant to current work; drift warnings; session throughput baseline | `bd ready` + model freshness + `events.jsonl` |
| **Before spawn** | Model claims relevant to the task; "last time we tried X it failed because Y" | `kb context` enrichment, `kb quick tried` entries |
| **During completion review** | "This changes your model of X — here's how" or "this contradicts model claim Y" | Agent deliverables cross-referenced with model claims |
| **Decision point** | Relevant constraints, prior decisions, tried-and-failed approaches | `kb quick` entries, `.kb/decisions/` |
| **Frustration/friction** | "You've hit this 3 times — here's the pattern" | Pattern detection across sessions |
| **Session end** | What shifted in understanding today; which models need updating | Delta between session start and end state |

**Design constraint:** Must respect context window limits. Surface summaries and pointers, not full files. 2-3 sentences per model, with path to depth if needed.

**Key question:** How to determine which models are "relevant" at each moment? Candidates:
- Models tagged with areas matching `bd ready` issues
- Models whose probes are stale (>7 days)
- Models referenced in recent completions
- Models matching current focus areas (`orch focus`)

### 2. System Self-Awareness (Throughput & Expectations)

The system tracks events in `~/.orch/events.jsonl` but never reflects on its own patterns. It doesn't know:
- How many issues typically close per session
- What the spawn-to-complete ratio is
- Where agents get stuck (skill-level failure rates)
- How long agents typically take by skill type

If it did, session start orientation could include: "Last 3 sessions averaged 4 completions. 2 agents are currently in-progress and likely to land. 3 issues are ready to spawn."

This turns session start from "what should we do?" into "here's what's ready to land" — grounded expectations, not aspirational ones.

**Implementation shape:** A `stats` or `session-baseline` command that queries `events.jsonl` and produces a session forecast. Could be woven into the orchestrator's session start protocol.

### 3. Calibrated Trust (Beyond Binary Verification)

Current state: verify everything (expensive, breaks flow) or rubber-stamp (cheap, entropy). Neither works.

**Calibrated trust** means verification effort is proportional to risk. Properties that enable this:

| Signal | High trust (light review) | Low trust (heavy review) |
|--------|--------------------------|--------------------------|
| **Model freshness** | Relevant model probed <3 days ago | Model stale >14 days or no model exists |
| **Agent scope** | Single-file, well-scoped, tests pass | Multi-file architectural change |
| **Pattern history** | Skill has high completion rate | Skill has high failure/rework rate |
| **Model consistency** | Agent work doesn't contradict model claims | Agent work contradicts or changes model claims |
| **Orchestrator model freshness** | Orchestrator can articulate what changed | Orchestrator can't articulate the delta |

**The trust indicator:** At completion review, the system could surface a trust tier:
- **Green:** Low risk. Light review sufficient. (Config change, single-file fix, tests pass, fresh model)
- **Yellow:** Moderate risk. Standard review. (Multi-file, touches known hotspot, model partially stale)
- **Red:** High risk. Deep review required. (Architectural, contradicts model, stale model, no prior art)

This lets Dylan modulate attention: rubber-stamp greens, engage on yellows, deep-dive on reds. Flow is preserved because most work should be green/yellow.

### 4. Immersion Without Study

The meta-problem: Dylan needs to be immersed in the knowledge but can't afford to study it separately. The solution isn't "read more files" — it's making the orchestration flow itself the medium through which understanding is maintained.

This means:
- **Orientation IS model review** — session start naturally includes "here's what you believe about X, is that still true?"
- **Completion review IS model update** — "this agent's work means Y about your model of Z"
- **Decision-making IS constraint surfacing** — relevant tried/failed/decided entries appear at the moment of choice
- **The flow IS the immersion** — if these moments work, Dylan's mental model stays current without ever opening a model file

## Open Questions

1. How to measure model freshness meaningfully? Last-probed date is crude. A model could be probed recently but in an area that doesn't matter for current work.
2. What's the right granularity for model summaries at session start? Too much = context bloat. Too little = useless.
3. How does the trust calibration interact with the existing completion review protocol? Replace it, augment it, or gate which tier of review to apply?
4. Should the system track "Dylan's last known understanding" of each model, so it can detect when reality has moved but his model hasn't?
5. How to avoid the meta-trap: building a system to monitor the system that monitors the system. Where does this terminate?

## Relationship to Existing System

- **Extends:** Orchestrator skill (session start/end protocols, completion review)
- **Extends:** `kb context` (currently agent-facing, could become human-facing too)
- **Extends:** `events.jsonl` (add throughput analytics)
- **New capability:** Trust calibration / review tiering
- **New capability:** Model-to-moment surfacing

## Next Steps

This is a design investigation, not an implementation plan. Next steps:
1. Validate these threads with Dylan — do they match the felt experience?
2. Identify which thread has the highest immediate impact (likely: model surfacing at session start + completion review enrichment)
3. Design concrete implementation for highest-impact thread
4. Prototype in orchestrator skill before building tooling
