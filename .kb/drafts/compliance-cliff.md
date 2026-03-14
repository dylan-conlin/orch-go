---
title: "The Compliance Cliff: Why Agent Frameworks Are Solving the Wrong Problem"
description: "Most AI agent tooling is compliance tooling — guardrails, structured output, retry logic. As models improve, compliance value approaches zero. Coordination value doesn't. We built a system that bifurcates them."
pubDate: 2026-03-14
draft: true
---

*Most AI agent tooling is compliance tooling. As models improve, its value approaches zero. Coordination tooling goes the other direction.*

---

## The Ratio

Look at any AI agent framework — LangChain, CrewAI, AutoGen, your company's internal one. Count the lines of code. I'll bet the breakdown looks something like this:

- **~80% compliance:** Guardrails, structured output parsing, retry logic, output validation, prompt templating, error recovery, format enforcement
- **~20% coordination:** Task routing, work selection, deduplication, resource allocation, inter-agent awareness

This ratio made sense in 2024. Models were unreliable. Getting an agent to follow instructions at all was the hard problem. You needed retry loops because tool calls failed 30% of the time. You needed structured output parsers because JSON came back malformed. You needed guardrails because the agent would wander off-task.

But here's the thing about compliance tooling: **it's solving a problem that models are actively eliminating.**

Every model generation makes compliance cheaper. GPT-3.5 needed elaborate retry chains. GPT-4 needed them less. Claude Opus rarely needs them at all. Structured output went from "parse and pray" to a native API feature. Tool calls went from fragile to reliable. The trajectory is clear: compliance cost trends toward zero.

Coordination is the opposite. As you add more capable agents running in parallel on shared systems, coordination becomes *harder*, not easier. More capable agents produce more code per session, create more potential conflicts, and generate more candidate work than any single agent can contextualize. The coordination problem scales with capability. It doesn't resolve with it.

Most frameworks have the ratio backwards. They're over-invested in the thing that's getting easier and under-invested in the thing that's getting harder.

---

## The Value Trajectories

This isn't abstract. I run 50+ autonomous AI agents per day on a shared Go codebase. I've watched these two trajectories diverge in real time over 12 weeks.

**Compliance value is falling.** My agents complete tasks correctly 96% of the time for feature implementation, 100% for investigation. The guardrails I built 8 weeks ago — phase enforcement, synthesis requirements, verification gates — are mostly overhead now. The agents do the right thing without being forced. When I measure which compliance gates actually block bad behavior versus which ones add ceremony to already-correct behavior, the answer is uncomfortable: most of them are ceremony.

**Coordination value is rising.** The harder problems are all coordination problems: Which of the 15 ready issues should the daemon spawn next? Are issues #47 and #52 actually the same work described differently? Should the daemon route this task through architect review or spawn directly? When a skill has a 30% success rate, should the system keep trying it or reallocate those slots to higher-yield skills?

These questions don't get easier with better models. A more capable agent still can't see the 14 other agents working on the same codebase. A more capable agent still can't deduplicate its task against work completed yesterday. Individual capability doesn't compose into collective coordination without infrastructure.

This is the compliance cliff: the point where your compliance investment stops returning value, and you realize you've under-built the coordination layer that actually matters.

---

## The Bifurcation

We hit this cliff and decided to do something structural about it. The daemon — our autonomous agent orchestrator — had compliance and coordination entangled in every method. Gate checks mixed with priority scoring. Verification logic mixed with routing decisions. You couldn't relax compliance without breaking coordination, and you couldn't extend coordination without navigating compliance.

So we split them. Seven commits in a single session, building from structural extraction through a full OODA refactor:

```
c9ea4a3  refactor: extract compliance/coordination from entangled daemon methods
cd3a7038 feat: add Learning Store — aggregate events into per-skill metrics
2629e085 feat: add ComplianceConfig type with per-spawn resolution
c554d6d5 feat: add allocation profile — skill-aware slot scoring
aebe1d80 feat: wire ComplianceConfig into daemon compliance gates
46f2fb00 feat: add measurement feedback loop + work graph
5bb7745f feat: restructure daemon into OODA poll cycle
```

The design principle is simple: **compliance produces signals, coordination consumes them.** The interface between them is narrow — three methods. Compliance says "allowed" or "blocked." Coordination decides what to do with the allowed slots.

This means you can turn compliance *down* without touching coordination. And that's exactly what the system does.

---

## The Compliance Dial

The core mechanism is a four-level compliance dial that can be set per skill, per model, or per (skill, model) combination:

```go
type ComplianceLevel int

const (
    ComplianceStrict    ComplianceLevel = iota  // All gates active
    ComplianceStandard                          // Relaxed for proven combos
    ComplianceRelaxed                           // Significantly reduced overhead
    ComplianceAutonomous                        // Safety-only mechanisms
)
```

Each level derives concrete thresholds:

| Level | Verification Pause | Architect Escalation | Synthesis Required | Phase Enforcement |
|-------|-------------------|---------------------|-------------------|-------------------|
| Strict | Every 3 auto-completes | Yes | Yes | Required |
| Standard | Every 8 | Yes | Yes | Required |
| Relaxed | Every 20 | No | No | Advisory |
| Autonomous | Disabled | No | No | Advisory |

Resolution follows a precedence chain — combo overrides skill overrides model overrides default:

```go
func (c *ComplianceConfig) Resolve(skill, model string) ComplianceLevel {
    if level, ok := c.Combos[model+"+"+skill]; ok {
        return level  // "opus+feature-impl" → specific level
    }
    if level, ok := c.Skills[skill]; ok {
        return level  // "investigation" → skill-wide level
    }
    if level, ok := c.Models[model]; ok {
        return level  // "opus" → model-wide level
    }
    return c.Default  // global default
}
```

The interesting part isn't the config — it's what drives the dial.

---

## The Learning Store

The system measures its own compliance overhead by aggregating agent lifecycle events into per-skill metrics:

```go
type SkillLearning struct {
    SpawnCount           int
    TotalCompletions     int
    SuccessCount         int
    AbandonedCount       int
    SuccessRate          float64
    AvgDurationSeconds   int
    VerificationFailures int
    GateHitRates         map[string]*GateStats
}
```

Every spawn, completion, abandonment, gate evaluation, and verification outcome gets recorded to an append-only event log. The Learning Store recomputes from this log periodically — no local state, no cache, just a functional computation over the event stream.

This gives the system a per-skill success rate. Feature implementation: 96%. Investigation: 100%. Systematic debugging: lower. The numbers are real and they update as the system runs.

---

## Auto-Adjustment

The compliance dial moves itself. When a skill sustains an 80%+ success rate across 10+ completions, the system suggests a one-level downgrade:

```go
const (
    MinSamplesForDowngrade         = 10
    DowngradeSuccessRateThreshold  = 0.80
)

func SuggestDowngrades(cfg *ComplianceConfig, learning *LearningStore) []DowngradeSuggestion {
    for skill, sl := range learning.Skills {
        sampleSize := sl.TotalCompletions + sl.AbandonedCount
        if sampleSize < MinSamplesForDowngrade {
            continue
        }
        if sl.SuccessRate < DowngradeSuccessRateThreshold {
            continue
        }
        currentLevel := cfg.Resolve(skill, "")
        if currentLevel >= ComplianceAutonomous {
            continue
        }
        // Step one level at a time — never jump from Strict to Autonomous
        suggestions = append(suggestions, DowngradeSuggestion{
            Skill:          skill,
            SuggestedLevel: currentLevel + 1,
            SuccessRate:    sl.SuccessRate,
        })
    }
}
```

Three design choices matter here:

**One level at a time.** The system steps from Strict to Standard, never from Strict to Autonomous. Each level gets its own measurement window. This prevents a lucky streak from eliminating all oversight.

**Safety asymmetry.** The system only suggests downgrades, never upgrades. Adding compliance back requires human judgment — "this skill is failing, add more oversight" is a decision with context that automated systems shouldn't make alone. Removing compliance when evidence supports it is lower-risk and can be automated.

**Creation/removal asymmetry.** This reflects a deeper principle: adding a gate is a local decision (you observe a problem, you add enforcement), but removing a gate requires global context (is anything depending on this enforcement that you can't see?). The auto-adjuster only operates in the safe direction — loosening constraints that measurably aren't needed.

On the first production run, the system computed downgrades for `feature-impl` (96% success, 49 completions) and `investigation` (100% success). The compliance overhead was measurably unnecessary, and the system said so.

---

## The Coordination Layer

With compliance producing binary signals (allowed/blocked) and auto-adjusting based on measurement, the interesting work moves to coordination. This is where the system decides not "did the agent follow instructions" but "given N things we could work on, which combination produces the most value?"

### The OODA Loop

The daemon's poll cycle is structured as four explicit phases:

**Sense** — gather raw signals. Check compliance gates (verification pause, completion health, rate limits). Poll the ready queue. Pure data collection, no decisions.

**Orient** — analyze and contextualize. Expand epics into children. Apply focus boosts for priority projects. Score candidates using the allocation profile. Interleave across projects. This is where the Learning Store feeds back — skill success rates modulate priority scores.

**Decide** — select the next action. Filter through compliance (the narrow interface). Infer skill and model. Route through hotspot detection and architect escalation. Pick the first candidate that passes all checks.

**Act** — execute the decision. Spawn the agent. Record the event. Feed back into the next Sense cycle.

The OODA structure makes the compliance/coordination split legible. Sense and Decide consume compliance signals. Orient is pure coordination. Act is execution. You can read the code and see exactly where each concern lives.

### The Work Graph

Coordination isn't just priority ordering. It's deduplication, conflict detection, and relationship mapping. The Work Graph computes three signals fresh each cycle:

**Title similarity.** Jaccard similarity after tokenization and stop-word removal. Threshold: 0.65. Two issues titled "fix daemon stuck detection timeout" and "fix daemon stuck agent timeout" score 0.78 — flagged as probable duplicates.

**File overlap.** Issues referencing the same source files. If issues #47 and #52 both mention `pkg/daemon/daemon.go`, the system flags the conflict — spawning both concurrently risks merge conflicts.

**Investigation chains.** Issues that reference other issue IDs, indicating follow-up work that should be sequenced, not parallelized.

These signals feed into removal candidates — not automated removal (that requires global context), but surfaced recommendations. The system says "these two look like duplicates" and lets the coordination layer decide.

This is where the creation/removal asymmetry becomes concrete. Any agent can create an issue — that's a local operation, no global context needed. But removing or deduplicating issues requires seeing the full work graph, understanding relationships between tasks, and knowing what other agents are working on. The Work Graph provides the visibility; the human (or a future orchestrator) provides the judgment.

### Allocation Scoring

The allocation profile scores candidate issues using skill success rates from the Learning Store:

```go
// Score = basePriority * (1 - weight + weight * blendedSuccessRate)
// P0 task at 100% success: 5 * 1.2 = 6.0
// P0 task at 0% success:   5 * 0.8 = 4.0
// P2 task at 100% success: 3 * 1.2 = 3.6

multiplier := 1 - SuccessRateWeight + SuccessRateWeight*successRate*2
score := basePriority * multiplier
```

Priority still dominates — a P0 task with 0% skill success (4.0) still outranks a P2 task with 100% success (3.6). But within the same priority band, the system routes work toward skills that are working and away from skills that aren't. Success rate gets blended with a default (0.5) for skills with fewer than 10 samples, preventing small sample overreaction.

The allocation profile doesn't just sort. It transforms the daemon from "process the queue in priority order" to "allocate cognitive resources where they'll produce the most value." That's a coordination function. It doesn't simplify with better models — it becomes more important, because more capable agents make the allocation decision higher-leverage.

---

## The Implication: Cognitive Resource Allocation

If you follow the compliance cliff to its conclusion, you arrive at a system that barely does compliance at all. Models improve. Success rates climb. The dial turns itself down. Gates that fired every third session now fire every twentieth. Synthesis requirements become advisory. Phase enforcement becomes optional.

What remains is a cognitive resource allocator.

Not "did the agent follow instructions" — the agent follows instructions, that's table stakes. But "given 15 things we could work on, 3 models we could use, and 5 agents we could spawn, which combination produces the most understanding per unit time?"

That's a fundamentally different problem than compliance. It requires:

- **Measurement** — per-skill success rates, duration distributions, gate hit rates
- **Deduplication** — detecting that two tasks are the same work described differently
- **Conflict detection** — knowing that two tasks target the same files
- **Relationship mapping** — understanding that task B depends on task A's findings
- **Allocation** — routing work toward proven skill/model combinations

None of these simplify with better models. All of them become more important as you scale agent count and capability.

The agent framework of the future isn't a compliance framework with coordination bolted on. It's a coordination framework with compliance as a dial — a dial that starts high and turns itself down as evidence accumulates.

---

## The Honest Assessment

### What's proven

The structural bifurcation works. Compliance and coordination are cleanly separated in the codebase. The interface between them is narrow (three methods). You can change one without breaking the other. This was validated by the implementation itself — seven commits, each independently testable, no regressions.

The Learning Store computes real metrics from real events. The auto-adjuster produces real downgrade suggestions on first run. The compliance dial moves based on evidence, not configuration.

The OODA structure makes the daemon's behavior legible. You can read the code and understand what happens in each phase. This is an engineering improvement independent of the thesis.

### What's speculative

The claim that compliance value approaches zero as models improve — I have 12 weeks of data from one system. The trajectory is clear in my data (96-100% success rates for two skills), but I haven't run controlled experiments with weaker models to establish the curve. The claim is directional, not quantified.

The claim that coordination value increases — this is structural (more agents = more coordination needed), but I haven't measured coordination ROI directly. I measure compliance ROI (gates that fire vs. gates that are ceremony). I don't yet have a measurement for "value of correct allocation" vs. "value of random allocation."

The auto-adjuster has suggested downgrades but hasn't been running long enough to validate that the downgrades are safe over time. A skill at 96% success under Strict compliance might drop to 85% under Standard if the Strict gates were catching the 4% that would have failed. I don't have enough data to rule this out. The one-level-at-a-time design is meant to make this detectable before it cascades.

### What's next

Accumulate 2-4 weeks of data under the bifurcated architecture. Run the gate effectiveness query to answer empirically whether structural enforcement improves agent quality or whether the 96% rate is intrinsic to the skill+model combination. If the answer is "intrinsic," the compliance cliff arrives faster than expected and the coordination layer becomes the entire system.

---

## Getting Started

If you're building agent orchestration and want to future-proof against the compliance cliff:

1. **Measure your compliance ratio.** Count your lines of compliance code (retry, validation, formatting, guardrails) vs. coordination code (routing, deduplication, allocation, scheduling). If the ratio is 80/20, you're over-indexed on the thing that's getting easier.

2. **Separate the interface.** Compliance produces signals. Coordination consumes them. The interface should be narrow — ideally a single "allowed/blocked" decision that coordination can query without understanding compliance internals.

3. **Build the measurement loop.** You can't auto-adjust what you don't measure. Track per-skill success rates, completion times, and gate hit rates. The Learning Store pattern — functional computation over an append-only event log — is simple and doesn't require a database.

4. **Start with the dial, not the AI.** The auto-adjuster is 80 lines of Go. The thresholds are constants. The logic is "if success rate > 0.80 and sample size > 10, suggest one level down." You don't need ML for this. You need measurement and conservative stepping.

5. **Invest in coordination.** Deduplication, conflict detection, allocation scoring — these are the capabilities that get more valuable as models improve. Build them now, while your compliance tooling is still useful enough to justify its existence.

---

*This is the second post in a series on multi-agent systems. The first — [Harness Engineering](https://dylanconlin.com/posts/harness-engineering/) — covers the compliance side: what breaks when 50 agents commit to the same codebase, and how gates and attractors prevent structural degradation. This post covers what comes after compliance: the coordination layer that decides what work to do, in what combination, with what relationship. The system described here is [orch-go](https://github.com/dylan-conlin/orch-go), running in production on a ~47,600-line Go codebase with 50+ agent sessions/day.*
