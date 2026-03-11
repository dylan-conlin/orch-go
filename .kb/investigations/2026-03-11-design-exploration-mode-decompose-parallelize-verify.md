---
title: "Design: Exploration Mode — Parallel Decomposition with Judge Synthesis"
status: complete
created: 2026-03-11
beads_id: orch-go-fauck
skill: architect
---

# Design: Exploration Mode — Parallel Decomposition with Judge Synthesis

## TLDR

Exploration mode adds a **decompose → parallelize → judge → synthesize** loop to investigation and architect skills. It's upstream of enforcement: agents explore freely in isolation, existing gates filter promotion to code. Implementation is a new `explore` spawn mode (not a new skill) that wraps existing skills with fan-out/fan-in orchestration.

## Orientation

**Problem:** Our harness is optimized for maintenance (coordination failure prevention on living codebases) but lacks exploration capability (compliance maximization on bounded problems). Investigation and architect skills currently run as single agents — they can't decompose a hard question into independently-attackable subproblems.

**Prior art:** Cursor's First Proof Challenge (Mar 2026) — general-purpose harness solved a research-grade math problem via decompose/parallelize/verify/iterate. Their claim: "prompts matter more than architecture." Our observation: not contradictory. They optimize for exploration (bounded problems), we optimize for maintenance (living systems). We need both.

**Constraint:** No codebase writes in v1. Analysis only.

---

## Fork 0: New Skill vs New Spawn Mode

**Decision: New spawn mode, not a new skill.**

Rationale: Investigation and architect skills already define *what* to analyze. Exploration mode defines *how* to attack the analysis — parallel decomposition is an execution strategy, not a domain behavior. This follows the established principle: "Skills own domain behavior, spawn owns orchestration infrastructure."

```bash
# Usage (proposed)
orch spawn --explore investigation "How does the daemon handle concurrent spawns?"
orch spawn --explore architect "Design token refresh architecture"
```

`--explore` modifies spawn to use decomposition orchestration instead of single-agent execution.

---

## Architecture

### Flow

```
orch spawn --explore investigation "question"
         │
         ▼
┌─────────────────────────────────────────────┐
│  1. DECOMPOSER (single agent)               │
│     Skill: investigation or architect        │
│     Input: original question                 │
│     Output: N independent subproblems        │
│     Constraint: subproblems must be          │
│     independently answerable                 │
└──────────────┬──────────────────────────────┘
               │ fan-out (parallel spawn)
               ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│ Worker 1 │ │ Worker 2 │ │ Worker N │
│ Sub-Q 1  │ │ Sub-Q 2  │ │ Sub-Q N  │
│ (invest) │ │ (invest) │ │ (invest) │
└────┬─────┘ └────┬─────┘ └────┬─────┘
     │             │             │
     └──────┬──────┘─────────────┘
            │ fan-in (wait for all)
            ▼
┌─────────────────────────────────────────────┐
│  3. JUDGE (single agent)                    │
│     Input: all sub-findings                 │
│     Evaluates: correctness, relevance,      │
│     consistency, coverage gaps              │
│     Output: verdict per sub-finding +       │
│     identified gaps                         │
└──────────────┬──────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────┐
│  4. SYNTHESIZER (single agent)              │
│     Input: sub-findings + judge verdicts    │
│     Output: unified analysis document       │
│     NOT concatenation — compositional       │
│     understanding                           │
└─────────────────────────────────────────────┘
```

### Agent Roles

| Role | Count | Skill | Purpose |
|------|-------|-------|---------|
| Decomposer | 1 | Same as parent (investigation/architect) | Break question into subproblems |
| Worker | N (2-5) | Same as parent | Answer one subproblem |
| Judge | 1 | New: `exploration-judge` | Evaluate sub-findings quality |
| Synthesizer | 1 | Same as parent | Compose findings into understanding |

### Key Design Decisions

**Workers use the same skill as parent.** An `--explore investigation` spawns investigation workers. An `--explore architect` spawns architect workers. The skill defines domain expertise; exploration mode defines execution topology.

**Judge is a new skill, but minimal.** The judge role doesn't exist in our current skill set. It evaluates *quality of findings*, not *domain content*. It needs its own behavioral grammar: what makes a finding correct? relevant? sufficient?

**Decomposer and Synthesizer are the parent skill with modified prompts.** They don't need new skills — they need modified SPAWN_CONTEXT that frames the task as "decompose this" or "synthesize these."

---

## Fork 1: What Does "Verification" Mean for Design Exploration?

This is the hardest design question. In code/math, verification means "run and check output." In design exploration, there's no executable.

**Answer: Verification is multi-dimensional assessment, not binary pass/fail.**

The judge evaluates each sub-finding on:

| Dimension | Question | Signal |
|-----------|----------|--------|
| **Grounding** | Does this claim cite specific code/docs/evidence? | Ungrounded claims are likely hallucinated |
| **Consistency** | Do sub-findings contradict each other? | Contradictions signal either wrong findings or genuinely contested territory |
| **Coverage** | Does the set of sub-findings cover the original question? | Gaps signal the decomposition missed something |
| **Relevance** | Does this finding address the subproblem it was assigned? | Drift means the worker solved a different problem |
| **Actionability** | Could someone act on this finding? | Vague findings ("it depends") are low value |

**The judge produces a structured verdict, not a pass/fail:**

```yaml
sub_findings:
  - id: worker-1
    verdict: accepted
    grounding: high  # cites specific files and line numbers
    notes: ""
  - id: worker-2
    verdict: contested
    grounding: medium
    notes: "Contradicts worker-1 on token lifetime. Both cite different code paths."
  - id: worker-3
    verdict: rejected
    grounding: low  # no code citations, appears to be general knowledge
    notes: "Claims about OAuth2 spec but doesn't check our implementation."
gaps:
  - "No sub-finding addressed error handling in the refresh flow"
  - "Token revocation not covered by any subproblem"
```

**Contested findings are the most valuable output.** They identify genuine complexity that a single agent would have papered over with false certainty.

---

## Fork 2: How Does Orchestrator Role Change?

**Current:** Orchestrator spawns workers, completes them, synthesizes. Linear.

**With exploration:** Orchestrator becomes the exploration coordinator. The decompose → parallelize → judge → synthesize loop is a *spawned orchestrator* that runs the loop autonomously.

**Implementation:** `--explore` spawns a specialized orchestrator agent (not a worker) that:
1. Decomposes the question (itself, as first action)
2. Spawns N workers in parallel
3. Waits for all workers
4. Spawns judge
5. Spawns synthesizer (or synthesizes itself)
6. Produces final analysis document

This fits the existing spawned orchestrator pattern exactly. The exploration orchestrator:
- Gets its own workspace
- Spawns/completes workers
- Produces SESSION_HANDOFF.md with the synthesis
- Waits for level above to complete

```
Meta-Orchestrator (Dylan or AI)
    │
    ├── orch spawn --explore investigation "question"
    │       │
    │       └── Exploration Orchestrator (spawned)
    │               │
    │               ├── Worker 1 (sub-question)
    │               ├── Worker 2 (sub-question)
    │               ├── Worker 3 (sub-question)
    │               ├── Judge (evaluates)
    │               └── Synthesizer (composes)
    │
    └── orch spawn feature-impl "implement findings"  # downstream
```

---

## Fork 3: Cost Model — Bounding Token Spend

Exploration is token-expensive. A single investigation becomes 1 (decomposer) + N (workers) + 1 (judge) + 1 (synthesizer) = N+3 agents.

**On Claude Max subscription (flat rate):** Cost is rate-limit slots, not dollars. Each exploration consumes N+3 concurrent agent slots from the 5-agent default limit.

**Proposed bounds:**

| Parameter | Default | Override |
|-----------|---------|----------|
| Max subproblems | 3 | `--explore-breadth N` |
| Max depth (iterative refinement) | 1 (no iteration in v1) | Future: `--explore-depth N` |
| Worker concurrency | 3 (all parallel) | Bounded by `--max-agents` |
| Total agent budget | 6 (3 workers + decomposer + judge + synthesizer) | Derived from breadth |

**Rate limit interaction:** The exploration orchestrator checks remaining rate-limit headroom before spawning workers. If headroom < N workers, it reduces breadth to fit. This reuses the existing proactive rate limit monitoring in spawn.

**5-minute rule interaction:** The 5-minute rule ("orchestrator should decide within 5 minutes") doesn't apply to exploration orchestrators. They're workers in meta-orchestrator's frame. The meta-orchestrator's 5-minute rule still applies to deciding *whether* to spawn exploration.

---

## Fork 4: Isolation and Promotion

**Exploration is upstream of enforcement.** This means:

1. Exploration agents read the codebase but write NO code
2. All output is analysis documents (investigation files, design docs)
3. These documents go through existing gates when they inform implementation:
   - Investigation → Architect (routing rule: investigation findings need architect before impl)
   - Architect → feature-impl (normal flow)
   - Hotspot gates still apply to any subsequent implementation

**Workspace isolation:**

```
.orch/workspace/
├── explore-daemon-concurrency-abc123/           # Exploration orchestrator
│   ├── SPAWN_CONTEXT.md
│   ├── SESSION_HANDOFF.md                       # Final synthesis
│   ├── decomposition.yaml                       # Subproblem breakdown
│   └── sub-findings/
│       ├── worker-1-finding.md
│       ├── worker-2-finding.md
│       ├── worker-3-finding.md
│       └── judge-verdict.yaml
```

Workers write to their own workspaces as normal. The exploration orchestrator collects their outputs into sub-findings.

---

## Implementation Plan

### Phase 1: Core Machinery (v1)

1. **`--explore` flag in spawn** — Routes to exploration orchestrator instead of single agent
2. **Exploration orchestrator skill** — Minimal skill that defines the decompose/fan-out/fan-in/judge/synthesize loop
3. **`exploration-judge` skill** — Evaluates sub-findings on grounding, consistency, coverage, relevance, actionability
4. **Decomposition prompt template** — Injected into SPAWN_CONTEXT when agent role is "decomposer"
5. **Synthesis prompt template** — Injected into SPAWN_CONTEXT when agent role is "synthesizer"
6. **Cost bounding** — Breadth limit, rate-limit awareness

### Phase 2: Observability

7. **Dashboard integration** — Show exploration tree (decomposer → workers → judge → synthesizer)
8. **Measurement surface** — Track: subproblem quality, judge agreement rate, synthesis coherence (per the "measurement as first-class harness layer" thread)

### Phase 3: Iteration (v2, future)

9. **Judge-triggered re-exploration** — If judge finds gaps, decomposer creates additional subproblems (depth > 1)
10. **Cross-exploration learning** — Judge verdicts feed into future decomposition quality

### What NOT to Build

- No code generation in exploration mode (v1 constraint)
- No custom UI for exploration (reuse spawned orchestrator dashboard)
- No persistent exploration state (workspaces are sufficient)
- No exploration-specific beads tracking (reuse existing issue lifecycle)

---

## Interaction with Existing Systems

| System | Interaction | Change Needed |
|--------|------------|---------------|
| Spawn gates | Exploration orchestrator passes all gates as an investigation/architect spawn | None — exploration is just a modified orchestrator spawn |
| Hotspot enforcement | Workers are investigation/architect (exempt skills) | None |
| Daemon | Could auto-trigger exploration for complex triage:ready investigations | Future: daemon heuristic for `--explore` |
| Completion verification | Exploration orchestrator completes via `orch complete` | None — standard spawned orchestrator lifecycle |
| kb context | Each worker gets kb context for its subproblem | None — standard spawn behavior |
| Rate limiting | Exploration orchestrator checks headroom before fan-out | Minor: orchestrator needs rate-limit awareness |

---

## Open Questions (For Orchestrator)

1. **Should the judge be a different model?** The two-model experiment decision says "worth running selectively." Judge is a natural candidate — use a different model to catch blind spots. But adds complexity to v1.

2. **Should exploration be available for systematic-debugging?** The design scopes to investigation/architect. Debugging could benefit from parallel hypothesis testing (each worker explores one hypothesis). But debugging writes code, violating the "no writes" constraint.

3. **Iteration depth:** v1 is single-pass (depth=1). When should we add iterative refinement (judge says "gap found" → decomposer adds subproblem → worker explores → re-judge)?

---

## Verification Spec

```yaml
type: design
deliverables:
  - path: .kb/investigations/2026-03-11-design-exploration-mode-decompose-parallelize-verify.md
    description: Design document for exploration mode
    verification: exists and addresses all 4 key design questions from task
evidence:
  - claim: "Exploration mode is a spawn mode, not a new skill"
    basis: "Skills own domain behavior, spawn owns orchestration infrastructure (prior decision)"
  - claim: "Fits existing spawned orchestrator pattern"
    basis: "Decompose/fan-out/fan-in/synthesize maps to orchestrator lifecycle"
  - claim: "No new enforcement needed — exploration is upstream"
    basis: "Investigation and architect are already hotspot-exempt; outputs go through existing gates"
  - claim: "Cost bounded by breadth limit and rate-limit awareness"
    basis: "Reuses existing proactive rate limit monitoring"
```
