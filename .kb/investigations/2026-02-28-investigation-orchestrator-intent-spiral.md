# Investigation: Orchestrator Intent Spiral

**Date:** 2026-02-28
**Status:** Open — pattern identified, no fix implemented
**Triggered by:** Post-session analysis of orchestrator session `2026-02-28-222046`

## What Happened

Dylan asked: "compare Playwright MCP vs CLI for UX audits." The orchestrator:

1. Spawned agents with **different skills** (ux-audit vs investigation) — confound #1
2. Fixed skill, but CLI agent was **contaminated by prior audit context** — confound #2
3. Spawned a third pair with controls (`--skip-artifact-check`, different page)
4. Then **undermined its own eval** ("what are we comparing?") listing everything the eval can't measure
5. Dylan had to explain "experiential evaluation" — the orchestrator had been designing a controlled A/B test when Dylan wanted "use the tool, form opinions, report back"

Three spawn/abandon cycles. Multiple beads issues. The orchestrator never caught the fundamental misunderstanding until Dylan explicitly stated it.

## Evidence

- **Session transcript:** `evidence/2026-02-28-orchestrator-intent-spiral/session-transcript.txt`
- **Orchestrator skill at time of failure:** `evidence/2026-02-28-orchestrator-intent-spiral/orchestrator-skill-snapshot.md` (checksum: 326ed539711c, compiled 2026-02-28 20:12:56)

## The Failure Pattern: Cascaded Intent Displacement

Intent passes through multiple translation layers: **human → orchestrator → spawn prompt → skill template → worker agent**. At each layer, the intent is reshaped by the layer's dominant frame. The output is confident execution of the wrong interpretation.

This is NOT information degradation (telephone game). Each layer actively constructs meaning using its own frame and optimizes with full competence. The failure is competence applied to the wrong thing.

### Four distinct failure points

**1. Intent didn't survive the spawn boundary.** "Evaluate" (experience this tool, form opinions) became "audit" (produce findings about a target page). The spawn prompt encoded a method, not the intent.

**2. Skill overrode intent.** The orchestrator's routing table (`EVALUATE UI/UX → ux-audit`) selected ux-audit. Once selected, the skill's methodology dominated the agent's behavior. The spawn prompt became secondary context. The orchestrator even "fixed" the first confound by making BOTH agents use ux-audit — deepening the problem.

**3. No early verification of agent behavior.** The CLI agent wrote a 300-line .cjs script in its first minutes — the moment the eval went wrong. Nobody checked until completion. When the orchestrator did look (`orch tail`), it rationalized the script-writing as valid rather than recognizing the wrong approach.

**4. Orchestrator optimized for rigor over understanding.** After two corrections, the orchestrator got defensive — pre-analyzing every possible confound, listing everything the eval can't measure, offering options that were all variations on controlled comparison. It was solving for "don't get corrected again" rather than "give Dylan what he wants."

### Amplification mechanism: error-correction feedback loop

Each correction from Dylan made the orchestrator more anxious. The response pattern:
1. Immediately agree (sycophancy: "Good catch", "You're right")
2. Over-correct with more elaborate methodology
3. Introduce new complexity or doubt
4. Drift further from original intent

The orchestrator skill's extensive pre-spawn checklists became amplifiers — an anxious orchestrator leans harder into ceremony, which adds more process without fixing comprehension.

## Academic Territory

This pattern sits at the intersection of multiple established fields:

| Term | Field | What it captures |
|------|-------|-----------------|
| **Goal displacement** | Organizational theory (Merton 1940) | Methods become ends — audit methodology replaced evaluation intent |
| **Variety attenuation** | Cybernetics (Ashby/Beer) | Each layer filters by its own model of what matters |
| **Systematically distorted communication** | Critical theory (Habermas) | Structural self-deception — distortion invisible to the distorter |
| **Specification gaming** | AI alignment (Goodhart) | Optimizing the literal spec, not the actual goal |
| **Commander's intent failure** | Military doctrine | "Why" lost, only "what" transmitted across layers |
| **Sensemaking / enactment** | Organizational psychology (Weick) | Layers construct (not discover) meaning from input |
| **Construal level shift** | Cognitive science (Trope & Liberman) | Abstraction strips situated, experiential meaning |

### Synthesis

**"Cascaded specification gaming through frame-dominant variety attenuation."** Each intermediary compresses intent using its own interpretive frame as the codec. The resulting specification is then faithfully optimized by the next layer. The failure is not degradation — it is confident execution of the wrong interpretation.

## Structural Vulnerability in the Orchestrator Skill

The skill's routing table has no path for experiential/exploratory work:

```
BUILD something    → feature-impl
DESIGN decisions   → architect
FIX broken thing   → systematic-debugging
UNDERSTAND         → probe | investigation | research
EVALUATE UI/UX     → ux-audit
```

"Try this tool and tell me what it's like" maps to none of these. The orchestrator forced it into the closest match (ux-audit), and the skill's methodology took over.

More broadly: as skills have gotten heavier (compiled, methodology-dense, context-filling), the cost of a skill mismatch has increased. A powerful skill overrides a weak spawn prompt more aggressively. The spawn prompt — where intent lives — gets proportionally less influence.

## Open Questions

1. Is `--intent` (experience / produce / compare) the right fix, or does it just add another field the orchestrator can fill in wrong?
2. Would "restate intent before spawning" (force plain-language confirmation) catch the gap earlier?
3. Does the skill system need a "no skill" / "freeform" path for work that doesn't fit existing categories?
4. Is the error-correction feedback loop (defensive analysis after corrections) addressable in the skill, or is it an LLM behavioral pattern that can't be gated?
5. How does this relate to the existing entropy spiral pattern? (Same family? Different mechanism?)

## Related

- Entropy spiral investigations: `2026-02-12`, `2026-02-13`, `2026-02-14` — different mechanism (agent self-reinforcing failure) but similar "confident execution of wrong thing" shape
- Orchestrator skill behavioral compliance: `2026-02-24-design-orchestrator-skill-behavioral-compliance.md`
- Orchestrator diagnostic mode: `2026-02-27-design-orchestrator-diagnostic-mode.md`
