# Meta-Orchestrator Mental Models

**Purpose:** Mental models that help the human meta-orchestrator maintain clarity while coordinating a system that evolves faster than verification allows.

**When to reference:** When feeling overwhelmed. When the system feels opaque. When you notice yourself compensating instead of observing. When making triage decisions.

**Source:** Synthesized from fs.blog mental models library (Shane Parrish), mapped to meta-orchestration context. Jan 24, 2026.

---

## The Map is Not the Territory

Your mental model of the system is a map. The code, agents, and their actual behavior are the territory. The map drifts.

**The test:** "When did I last verify this belief against the territory?"

**What this means:**
- Your understanding of how spawn works may be stale
- The dashboard shows a map, not reality (agents can be dead while appearing alive)
- SYNTHESIS.md files are maps written by agents about their own work
- Principles.md is a map of what matters - verify it still matches

**What this rejects:**
- "I know how this works" (when did you last verify?)
- "The dashboard says healthy" (dashboard is a map)
- "The agent said it's fixed" (agent report is a map)

**Recalibration trigger:** After any system spiral, major refactor, or extended absence - your map is stale.

---

## Circle of Competence

Know the boundary between what you understand and what you're guessing about.

**The test:** "Am I triaging from understanding, or from pattern-matching?"

**What this means:**
- Some areas you understand deeply (spawn flow, beads lifecycle)
- Some areas you're guessing (OpenCode internals, Claude API behavior)
- Triage decisions in your circle: act confidently
- Triage decisions outside your circle: spawn investigation first

**What this rejects:**
- Triaging complex work in unfamiliar areas without investigation
- Assuming the system surfaced all relevant context
- Deferring to AI recommendations without checking your own experience

**The meta-orchestrator trap:** The system presents information confidently. You pattern-match and triage. But you're outside your circle - you're guessing, not deciding.

**Recovery:** "Do I have direct experience here, or am I trusting the frame I was given?"

---

## Activation Energy

The initial effort required to overcome inertia. Strategic paths have higher activation energy than tactical paths.

**The test:** "Am I choosing tactical because it's better, or because it's easier to start?"

**What this means:**
- Spawning `systematic-debugging` feels easier than `architect`
- "Quick fix" has lower activation energy than "understand the pattern"
- The temptation to tactical IS the activation energy barrier to strategic
- Strategic-First Orchestration exists because activation energy biases toward tactical

**What this rejects:**
- "I'll do strategic next time" (you won't - same activation energy)
- "This one is simple enough for tactical" (the 5th time you've said this)
- Treating effort-to-start as signal of correctness

**The discipline:** When you feel resistance to strategic approach, that's the activation energy. Push through it for patterned areas.

---

## Second-Order Thinking

Consider downstream effects, not just immediate outcomes.

**The test:** "What happens after this spawn completes?"

**What this means:**
- Spawning 5 agents creates 5 completion reviews
- Each completion may surface follow-up work
- Verification backlog accumulates faster than you can clear it
- Today's spawns are tomorrow's triage load

**What this rejects:**
- "I'll spawn these and review later" (later is now, plus new spawns)
- Measuring productivity by spawn count (ignoring verification debt)
- "The agents will handle it" (you're the bottleneck, not them)

**The math:** If you spawn 5 agents/hour and verify 2/hour, you accumulate 3 unverified agents/hour. By end of day: 24 unverified. This is Verification Bottleneck as second-order effect.

---

## The Red Queen Effect

Continuous adaptation necessary just to maintain position.

**The test:** "Am I improving, or just keeping up?"

**What this means:**
- The system generates complexity faster than you eliminate it
- Standing still = falling behind (investigation backlog grows)
- Synthesis isn't optional luxury - it's required to maintain understanding
- Each session without synthesis increases map/territory drift

**What this rejects:**
- "I'll synthesize when things calm down" (they won't)
- "We're making progress" (relative to what baseline?)
- Treating hygiene as deferrable

**The trap:** Feeling productive while actually losing ground. Spawn, complete, spawn, complete - but understanding degrades, backlog grows, map drifts.

---

## Local vs Global Maxima

Adequate solutions that prevent discovering optimal ones.

**The test:** "Is this fix good enough that I'll stop looking, but not actually good?"

**What this means:**
- Each tactical fix finds a local maximum
- Local maxima feel like progress (the bug is fixed!)
- Architect seeks global maxima (the design that prevents the bug class)
- Coherence Over Patches is the escape from local maxima

**What this rejects:**
- "It works now" (local maximum achieved)
- "We can refactor later" (you found the local maximum, you'll stay there)
- 5th fix to same file (you're hill-climbing, not exploring)

**Visual:**
```
Global Maximum ────────────────────────► (architect finds this)
                    ╱
Local Maximum ─────╱                     (tactical fixes find these)
                  ╱
    ───────────╱
```

---

## Catalyst Model

The orchestrator accelerates change without being changed.

**The test:** "Am I catalyzing work, or doing work?"

**What this means:**
- Catalysts speed up reactions without being consumed
- Orchestrator speeds up agent work without doing agent work
- If you're reading code to understand implementation → you're the reagent, not catalyst
- Your job: reduce activation energy for agents, provide context, verify outcomes

**What this rejects:**
- "Let me quickly check the code" (you're now the reagent)
- "I'll just fix this one thing" (catalyst became reactant)
- Doing investigation work instead of spawning it

**The discipline:** Catalysts don't change. If you're changing (writing code, doing investigation), you've left the catalyst role.

---

## Margin of Safety

Buffer protecting against unexpected stress.

**The test:** "If verification takes longer than expected, do I have margin?"

**What this means:**
- Don't spawn at the rate you can theoretically verify
- Build slack: if you can verify 3/hour, spawn 2/hour
- Margin absorbs: complex completions, interruptions, discovery work
- No margin = any surprise creates backlog

**What this rejects:**
- "I can handle 5 agents" (under ideal conditions)
- Running at capacity as default state
- Treating margin as wasted capacity

**The math:** Margin = (Verification Capacity - Spawn Rate) × Time. Negative margin = growing backlog.

---

## Churn

Some component replacement maintains health; excess damages stability.

**The test:** "Is this agent failure healthy learning, or system instability?"

**What this means:**
- Some agent abandonment is normal (wrong approach, scope too big)
- Healthy churn: <20% abandonment, learnings captured
- Unhealthy churn: >40% abandonment, same failures recurring
- Zero churn is suspicious (not attempting hard things)

**What this rejects:**
- "Every abandon is a problem" (some is healthy)
- "High completion rate = good" (might mean avoiding hard work)
- Ignoring churn patterns (same failure type recurring)

**Calibration:**

| Churn Rate | Interpretation |
|------------|----------------|
| <10% | Possibly not challenging enough |
| 10-25% | Healthy exploration |
| 25-40% | Check for systemic issues |
| >40% | System unstable, pause and investigate |

---

## Critical Mass

Threshold where gradual change becomes explosive.

**The test:** "Is this cluster big enough to synthesize?"

**What this means:**
- 15+ investigations on topic = critical mass for model creation
- Below threshold: individual findings
- Above threshold: pattern visible, model possible
- Synthesis converts accumulated mass into structured understanding

**What this rejects:**
- Synthesizing too early (not enough mass)
- Ignoring accumulated mass (15+ investigations sitting idle)
- Treating each investigation as isolated

**Current thresholds:**

| Artifact Type | Critical Mass | Action |
|---------------|---------------|--------|
| Investigations on topic | 15+ | Create model |
| Fixes to same file | 5+ | Spawn architect |
| Failures of same type | 3+ | Pattern investigation |

---

## Multiply by Zero

Single failure negates all other efforts.

**The test:** "What single point of failure would make everything else worthless?"

**What this means:**
- If dashboard can't see agent death → all monitoring is × 0
- If verification doesn't happen → all agent work is × 0
- If beads sync fails → all issue tracking is × 0
- Observation Infrastructure principle exists because of multiply-by-zero risk

**What this rejects:**
- "Most of the system works" (zero multiplies everything)
- Tolerating known observation gaps
- "We'll fix that later" for critical-path failures

**The discipline:** Find the zeros. Fix them first. Everything else is rearranging deck chairs.

---

## Applying These Models

**Session start:** Circle of Competence (what do I actually understand today?)

**Triage decisions:** Map vs Territory (when did I verify this?), Activation Energy (am I avoiding strategic?)

**Spawn decisions:** Second-Order Thinking (what's the downstream load?), Margin of Safety (do I have capacity?)

**Mid-session:** Red Queen (am I keeping up or falling behind?), Churn (healthy or unstable?)

**Completion review:** Local vs Global Maxima (is this fix good enough to stop, but not actually good?)

**Session end:** Critical Mass (anything ready to synthesize?), Multiply by Zero (any observation gaps?)

---

## Cross-References

**Principles that derive from these models:**

| Mental Model | Principle | Relationship |
|--------------|-----------|--------------|
| Map ≠ Territory | Evidence Hierarchy | Artifacts are maps, code is territory |
| Second-Order Thinking | Verification Bottleneck | Downstream effects of spawn rate |
| Local vs Global Maxima | Coherence Over Patches | Tactical finds local, architect finds global |
| Activation Energy | Strategic-First Orchestration | Tactical bias is activation energy problem |
| Feedback Loops | Pain as Signal | System self-correction via injected friction |
| Multiply by Zero | Observation Infrastructure | Visibility gaps are zeros |

**Related guides:**
- `~/.kb/principles.md` - Foundational principles (derived from these models)
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Operational orchestrator guidance
