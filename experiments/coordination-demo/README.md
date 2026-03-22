# Communication Doesn't Produce Coordination

We tested 100 AI agent pairs. Every agent completed its task perfectly. Every agent that could communicate did communicate. Every agent that was told to avoid conflicts acknowledged the instruction.

**Zero coordination. 100% merge conflict rate.**

Then one structural change — telling agents *where* to put their code instead of *about* each other — and the conflict rate dropped to zero.

```
Condition          Success Rate
─────────────────────────────────────────────────
no-coord           0%   (0/20)   Agents work alone
context-share      0%   (0/20)   Agents see each other's tasks
messaging          0%   (0/20)   Agents exchange plans first
gate               0%   (0/20)   Agents must verify no conflicts
placement        100%  (20/20)   Orchestrator assigns insertion points
```

This isn't a capability failure. All 200 agents scored 6/6 on individual task completion. The agents understand the problem, discuss it, and acknowledge the risk — then choose the same insertion point anyway, because it's the semantically correct location.

**Communication is not coordination. Structure is.**

## Reproduce It

```bash
./demo.sh setup      # Check prerequisites (~5 seconds)
./demo.sh run        # Run 1 trial per condition (~10 minutes)
./demo.sh results    # Visualize results
```

For the full 100-trial experiment: `./demo.sh run --full` (~2 hours).

### Prerequisites

- **Go** (1.21+) — agents implement Go functions
- **Claude CLI** (`claude`) — agents are Claude Code instances
- **jq** — for results analysis
- **git** — for worktree-based agent isolation

## The Experiment

Two AI agents work simultaneously on the same Go codebase. Each implements a different function in `pkg/display/display.go`:

| Task Type | Agent A | Agent B |
|-----------|---------|---------|
| Simple | `FormatBytes(int64) string` | `FormatRate(float64) string` |
| Complex | `VisualWidth(string) int` + `PadToWidth` | `FormatTable([]string, [][]string) string` |

Both agents work in isolated git worktrees branched from the same commit. After both finish, we attempt a `git merge`. The merge either succeeds (clean merge + tests pass) or fails (conflict, build failure, or test failure).

Five conditions vary what coordination information the agents receive:

1. **no-coord** — Each agent only sees its own task. No mention of the other agent.
2. **context-share** — Each agent is given the other agent's full task description. "Be aware of where the other agent will insert code."
3. **messaging** — Agents exchange implementation plans via shared files before coding. Both agents' tasks are visible.
4. **gate** — Context sharing plus a mandatory post-implementation step: review your insertion point, predict the other agent's, check for overlap, and revise if conflicting.
5. **placement** — The orchestrator assigns explicit, non-overlapping insertion points. Agent A: "place after `FormatDurationShort`". Agent B: "place after `StripANSI`".

## What We Found

### Communication works. Coordination doesn't.

In the messaging condition, 18 out of 20 trials had **both** agents write coordination plans. The plans consistently say "no conflicts expected" while describing identical insertion points ("after FormatDurationShort"). The agents understand the task, understand the risk, and still collide.

### Gates don't help either.

The gate condition adds a mandatory self-check: "After implementing, review your insertion point against the other agent's likely insertion point. If they overlap, **move your code**." All 20 trials: CONFLICT. Agents perform the check, report "no conflict," and keep their code exactly where it was.

Why? The verification step is subject to the same bias as the original decision. The semantically correct location beats the coordination-safe location at every decision point — including the verification step.

### Structural placement works perfectly.

The only condition that works assigns non-overlapping regions at design time. No runtime decision is required. The agent doesn't choose where to put code — the orchestrator already decided.

## The Framework

These results generalize into four coordination primitives:

| Primitive | What It Does | Key Finding |
|-----------|-------------|-------------|
| **Route** | Agents don't collide — work assigned to non-overlapping regions | The only primitive that matters in this experiment |
| **Sequence** | Work happens in the right order | Not tested here (agents run in parallel) |
| **Throttle** | Velocity doesn't exceed verification bandwidth | Not tested here (2 agents only) |
| **Align** | Agents share a current, accurate model of what "correct" means | Necessary but not sufficient — all agents were aligned on task, none coordinated |

### Gates vs Attractors

Each primitive can be implemented two ways:

- **Gate** (runtime check): An LLM or conditional decides what happens next during execution. Can fail because the decider makes the wrong choice. Example: "check for conflicts before committing."
- **Attractor** (structural constraint): The system's shape routes work at design time. No runtime decision required. Cannot be bypassed. Example: "place after function X."

Across 100 trials and 6 external framework analyses: **every gate-based coordination fails, every attractor-based coordination works.**

| Framework | Works? | Mechanism |
|-----------|--------|-----------|
| CrewAI | No | Gate (manager LLM routing) |
| LangGraph | No | Gate (conditional graph edges) |
| OpenAI Agents SDK | No | Gate (output-mediated handoffs) |
| Anthropic production | **Yes** | **Attractor** (task regions + output formats) |
| autoresearch | **Yes** | **Attractor** (N=1 eliminates coordination) |

### The N=1 Boundary

When only one agent works on a region, all four primitives are trivially satisfied. This is why autoresearch succeeds with radical simplicity — it eliminates coordination rather than solving it. The first architectural question isn't "how should agents coordinate?" but "can we structure the work so they don't have to?"

## Reference Data

The `reference-results/` directory contains the canonical results:
- `summary.json` — Machine-readable combined results from all 100 trials
- Full raw data is in `redesign/results/` (80-trial run: `20260310-174045/`, 20-trial gate extension: `20260322-124035/`)

## Project Structure

```
coordination-demo/
├── demo.sh                  # Three-command entry point
├── visualize.sh             # Results visualization
├── README.md                # This file
├── reference-results/
│   └── summary.json         # Canonical 100-trial results
└── redesign/
    ├── run.sh               # Experiment runner (5 conditions × N trials)
    ├── score.sh             # Individual agent scoring (6-point rubric)
    ├── analyze.sh           # Cross-condition analysis
    ├── prompts/             # Task descriptions for each agent
    │   ├── simple-a.md      # FormatBytes task
    │   ├── simple-b.md      # FormatRate task
    │   ├── complex-a.md     # VisualWidth + PadToWidth task
    │   └── complex-b.md     # FormatTable task
    └── results/             # Raw experiment data
        ├── 20260310-174045/ # 80-trial (4-condition) run
        └── 20260322-124035/ # 20-trial (gate condition) run
```

## Full Model

The complete coordination model, including evidence tables, control theory mapping, external validation, and open questions, is in [`.kb/models/coordination/model.md`](../../.kb/models/coordination/model.md).
