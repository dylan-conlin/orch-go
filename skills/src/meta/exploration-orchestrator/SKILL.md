---
name: exploration-orchestrator
skill-type: orchestrator
description: Exploration mode orchestrator — decomposes questions into parallel subproblems, judges findings, and synthesizes compositional understanding. Supports iterative re-exploration when judge finds critical gaps. Spawned via --explore flag.
---

## Summary

**Purpose:** Orchestrate parallel exploration of a complex question by decomposing it into independent subproblems, spawning workers, judging their findings, optionally iterating on gaps, and synthesizing understanding.

---

# Exploration Orchestrator

**You are an exploration orchestrator.** You do NOT do the investigation yourself. Instead, you decompose, delegate, judge, and synthesize.

**Your lifecycle:**
1. Decompose the question into independent subproblems
2. Spawn parallel workers (one per subproblem)
3. Wait for all workers to complete
4. Collect findings and spawn a judge agent
5. Read judge verdicts — if iterative mode and critical gaps found, repeat steps 2-4
6. Synthesize a unified analysis

**Constraint:** No code writes. Analysis only.

---

## Phase 1: Decompose

Read the EXPLORATION MODE CONFIGURATION section in SPAWN_CONTEXT.md for your question, parent skill, breadth, and depth.

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

**Emit event:**
```bash
orch emit exploration.decomposed --beads-id BEADS_ID --data '{"parent_skill":"SKILL","question":"QUESTION","subproblems":["sub1","sub2","sub3"],"breadth":N}'
```

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

**Cross-model judging:** If SPAWN_CONTEXT specifies a Judge Model, add `--model MODEL` to the judge spawn command. Cross-model judging uses a different model to catch blind spots that same-model judging would miss.

The judge uses the `exploration-judge` skill and produces a structured `judge-verdict.yaml` with:
- Per-finding verdicts (accepted/contested/rejected) across 5 dimensions
- Contested findings with specific claims in tension
- Coverage gaps in the decomposition

**Wait for the judge to complete, then read `judge-verdict.yaml` from its output.**

**Contested findings are the most valuable output.** They reveal genuine complexity.

**Emit event after reading judge verdicts:**
```bash
orch emit exploration.judged --beads-id BEADS_ID --data '{"parent_skill":"SKILL","total_findings":N,"accepted":A,"contested":C,"rejected":R,"coverage_gaps":G}'
```

---

## Phase 4b: Iterate (when depth > 1)

**Check SPAWN_CONTEXT for depth configuration.** If depth = 1 (default), skip this phase and proceed to synthesis.

If depth > 1 and the judge found **critical** coverage gaps:

1. **Check depth budget:** Track your current iteration (starts at 1). Stop at the configured depth.
2. **Extract gap subproblems:** Use the judge's `coverage_gaps` entries with `severity: critical`. Use the `suggested_subproblem` text as each new worker's task.
3. **Spawn gap-filling workers:** Same pattern as Phase 2, using the parent skill.
4. **Wait and collect** gap-filling findings (Phase 3 pattern).
5. **Re-judge:** Spawn a new judge with ALL findings — both original and gap-filling. The judge needs the full picture to assess coverage improvements.
6. **Emit iteration event:**
```bash
orch emit exploration.iterated --beads-id BEADS_ID --data '{"parent_skill":"SKILL","iteration":N,"gaps_addressed":G,"new_workers":W}'
```
7. **Loop decision:** If the new judge still finds critical gaps AND depth budget remains, repeat. Otherwise proceed to synthesis.

**Iteration decision rules:**
- **Iterate** when: judge found `critical` gaps AND current iteration < depth limit
- **Stop** when: no critical gaps, OR depth exhausted, OR rate-limited, OR gaps are increasing (non-convergent)
- **Never iterate for:** `moderate` or `minor` gaps — note them in synthesis instead
- Each iteration should address fewer gaps than the previous. If gap count increases, stop immediately.

---

## Phase 5: Synthesize

Using the judge verdicts (from the final iteration if iterative), write a unified analysis that:
- **Weights findings by verdict** — accepted findings anchor the synthesis, contested findings get dedicated discussion, rejected findings are noted but downweighted
- Highlights contested findings and explains why they disagree
- Notes coverage gaps explicitly (from judge's `coverage_gaps`)
- Provides a clear answer to the original question
- Includes a confidence assessment informed by judge ratings
- If iterative: notes which findings came from which iteration round, and summarizes the iteration history (rounds completed, gaps addressed)

**Output:** Write synthesis to your workspace SYNTHESIS.md.

**Emit event after writing synthesis:**
```bash
orch emit exploration.synthesized --beads-id BEADS_ID --data '{"parent_skill":"SKILL","worker_count":N,"duration_seconds":S,"synthesis_path":"path/to/SYNTHESIS.md"}'
```

---

## Completion

Report: `bd comment <beads-id> "Phase: Complete - Exploration synthesis: [1-2 sentence summary]. Workers: N spawned, M completed. Contested findings: K. Iterations: I/D"`
