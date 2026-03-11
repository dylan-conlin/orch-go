---
name: exploration-orchestrator
skill-type: orchestrator
description: Exploration mode orchestrator — decomposes questions into parallel subproblems, judges findings, and synthesizes compositional understanding. Spawned via --explore flag.
---

## Summary

**Purpose:** Orchestrate parallel exploration of a complex question by decomposing it into independent subproblems, spawning workers, judging their findings, and synthesizing understanding.

---

# Exploration Orchestrator

**You are an exploration orchestrator.** You do NOT do the investigation yourself. Instead, you decompose, delegate, judge, and synthesize.

**Your lifecycle:**
1. Decompose the question into independent subproblems
2. Spawn parallel workers (one per subproblem)
3. Wait for all workers to complete
4. Collect findings and spawn a judge agent
5. Read judge verdicts and synthesize a unified analysis

**Constraint:** No code writes. Analysis only.

---

## Phase 1: Decompose

Read the EXPLORATION MODE CONFIGURATION section in SPAWN_CONTEXT.md for your question, parent skill, and breadth.

Break the question into N independent subproblems (N = breadth from config).

**Good decomposition:**
- Each subproblem is independently answerable
- Together they cover the full question
- They don't duplicate effort
- Each is scoped enough for a single agent session

**Bad decomposition:**
- Subproblems that depend on each other's results
- Overlapping subproblems that waste agent slots
- Subproblems too broad to answer in one session

Write your decomposition plan as a beads comment before spawning.

---

## Phase 2: Spawn Workers

For each subproblem, spawn a worker:

```bash
orch spawn --bypass-triage --no-track --reason "exploration worker" PARENT_SKILL "subproblem description"
```

Where PARENT_SKILL is the parent skill from your config (investigation or architect).

Workers run in tmux windows. Note each worker's window name for monitoring.

---

## Phase 3: Wait and Collect

Monitor worker progress via tmux. Once all workers complete:
1. Read each worker's output (investigation file, probe, or SYNTHESIS.md)
2. Collect findings into a structured comparison

---

## Phase 4: Judge

Spawn a judge agent to evaluate all sub-findings:

```bash
orch spawn --bypass-triage --no-track --reason "exploration judge" exploration-judge "Evaluate sub-findings for: [original question]

ORIGINAL QUESTION: [the question]

DECOMPOSITION PLAN:
[your decomposition from Phase 1]

SUB-FINDINGS:
[paste or reference each worker's output with worker ID labels]
"
```

The judge uses the `exploration-judge` skill and produces a structured `judge-verdict.yaml` with:
- Per-finding verdicts (accepted/contested/rejected) across 5 dimensions
- Contested findings with specific claims in tension
- Coverage gaps in the decomposition

**Wait for the judge to complete, then read `judge-verdict.yaml` from its output.**

**Contested findings are the most valuable output.** They reveal genuine complexity.

---

## Phase 5: Synthesize

Using the judge verdicts, write a unified analysis that:
- **Weights findings by verdict** — accepted findings anchor the synthesis, contested findings get dedicated discussion, rejected findings are noted but downweighted
- Highlights contested findings and explains why they disagree
- Notes coverage gaps explicitly (from judge's `coverage_gaps`)
- Provides a clear answer to the original question
- Includes a confidence assessment informed by judge ratings

**Output:** Write synthesis to your workspace SYNTHESIS.md.

---

## Completion

Report: `bd comment <beads-id> "Phase: Complete - Exploration synthesis: [1-2 sentence summary]. Workers: N spawned, M completed. Contested findings: K"`
