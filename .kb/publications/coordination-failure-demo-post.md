# Two AI Agents, One Task, Zero Chance: A Coordination Failure Demo You Can Run in 5 Minutes

*Why upgrading your model won't fix your multi-agent merge conflicts — and what will.*

---

I gave the same coding task to two AI agents. Both completed it perfectly. Then I tried to merge their work.

```
Auto-merging pkg/display/display.go
CONFLICT (content): Merge conflict in pkg/display/display.go
Auto-merging pkg/display/display_test.go
CONFLICT (content): Merge conflict in pkg/display/display_test.go
Automatic merge failed
```

Both agents scored 6/6 on every dimension — correct code, passing tests, no regressions, matching spec. Individually flawless. Together, unusable.

I ran this experiment 10 times. The conflict rate was 100%. Fisher's exact test: **p = 1.0**. Not "statistically significant." Identical.

Then I tried upgrading the model. Maybe the cheaper model (Haiku) was the problem. Maybe Opus — Anthropic's most capable model — would handle it differently.

It didn't. Both models: 100% individual success, 100% merge conflict. Same insertion point, same structural failure, every single time. Twenty agent runs, zero successful merges.

This isn't a bug. It's the default behavior of any system where multiple AI agents work on shared code without coordination infrastructure. And if you're running more than a handful of agents, you're probably already experiencing it.

---

## The Experiment

The setup is deliberately simple. I wanted to isolate the coordination failure from everything else — no complex tasks, no ambiguous specs, no room for the agents to make mistakes.

**Task:** Add a `FormatBytes(bytes int64) string` function to a Go package. Convert byte counts to human-readable strings (512 → "512 B", 1024 → "1.0 KiB", 1048576 → "1.0 MiB"). Write tests. Don't break anything.

**Setup:** Two isolated git worktrees from the same commit. Each agent gets the exact same prompt. They can't see each other's work.

**Models tested:** Claude Haiku 4.5 and Claude Opus 4.5. One is the cheapest model in the family. The other is the most capable.

**Scoring rubric (6 dimensions):**

| Dimension | What It Measures |
|-----------|-----------------|
| Completion | Did the agent produce changes? |
| Compilation | Does `go build` pass? |
| New tests | Do the new tests pass? |
| No regression | Do existing tests still pass? |
| File discipline | Only expected files modified? |
| Spec match | Function matches the required signature? |

### Results: Individual Performance

Both models scored 6/6 in all 10 trials. Twenty runs, zero individual failures.

| Model | Trials | Score | Mean Duration |
|-------|--------|-------|---------------|
| Haiku | 10/10 | 6/6 every trial | 39.1s |
| Opus | 10/10 | 6/6 every trial | 44.0s |

For a well-specified task, the $0.25/MTok model and the $15/MTok model are functionally identical. Haiku is marginally faster (not statistically significant — Welch's t-test, p > 0.05).

### Results: Coordination

Now the interesting part. For each trial, I merged the Haiku branch into the Opus branch.

| Trial | Merge Result |
|-------|-------------|
| 1 | CONFLICT |
| 2 | CONFLICT |
| 3 | CONFLICT |
| 4 | CONFLICT |
| 5 | CONFLICT |
| 6 | CONFLICT |
| 7 | CONFLICT |
| 8 | CONFLICT |
| 9 | CONFLICT |
| 10 | CONFLICT |

Ten trials, ten conflicts. The 95% Clopper-Pearson confidence interval for the true conflict rate is **[69.2%, 100%]** — the lower bound is well above 50%. This isn't an edge case. It's the dominant outcome.

The failure is structural: both agents follow the instruction "place the function after FormatDurationShort" and insert at the exact same line. Git can't auto-merge two different insertions at the same position.

### The Identical Commit Message

Here's the detail that makes the structural nature unmistakable. Both models, working independently, produced the **identical** commit message:

```
feat: add FormatBytes function for human-readable byte formatting
```

Same conventional commit format, same words. Independent convergence. They even think about the work the same way. The predictability that makes each agent individually reliable is exactly what makes them collectively incompatible.

---

## But What About Harder Tasks?

Fair question. Maybe the simple task doesn't reveal real capability differences.

I ran a second experiment with a complex, ambiguous task: build a table renderer across 4 files (modify 2 existing, create 2 new). The spec deliberately left design choices open — column separator style, padding, how to handle rows with mismatched columns.

Both models scored 10/10 on a 10-dimension rubric. The coordination failure was worse:

```
CONFLICT (content): Merge conflict in display.go
CONFLICT (content): Merge conflict in display_test.go
CONFLICT (add/add): Merge conflict in table.go
CONFLICT (add/add): Merge conflict in table_test.go
```

A new conflict type appeared: `add/add` — both agents created the same new files. And a new failure mode emerged that the simple task didn't have: **semantic conflicts**. The agents made incompatible design decisions (Haiku expands tables for extra columns, Opus ignores them). Even if the text-level merge succeeded, the tests would fail against each other's implementations.

Capability differences *did* emerge — Opus anticipated Unicode edge cases the spec didn't mention (using rune counting instead of byte length), and wrote tests that verify actual column alignment positions rather than just checking separators exist. But these differences are invisible to the scoring rubric. Both pass 10/10.

**The takeaway:** model capability matters for code quality but not for coordination. Upgrading the model improves edge-case handling. It doesn't reduce merge conflicts.

---

## Compliance vs. Coordination: The Distinction That Matters

This experiment demonstrates a distinction I think is underappreciated in multi-agent AI systems:

| | Compliance Failure | Coordination Failure |
|---|---|---|
| **What breaks** | Agent doesn't follow instructions | Agents each follow instructions correctly but collectively produce conflicts |
| **Example** | Agent ignores the spec | Both agents implement the spec perfectly at the same insertion point |
| **Fixed by better models?** | Yes | No — made *worse* by faster, more confident agents |
| **Response to scale** | Decreases (smarter agents comply more) | Increases (more agents = more conflicts) |

Most multi-agent failure taxonomies — including Cemri et al.'s MAST framework analyzing 1,600+ traces — categorize failures by symptom rather than by their response to model improvement. Their prescription for inter-agent misalignment: "deeper social reasoning abilities." That's a compliance answer to a coordination question.

The experiment makes this concrete. Both agents demonstrated perfect compliance. Zero failures across 20 runs on 6 dimensions. The failure is entirely in coordination — and it's 100% structural.

This has a practical implication: **the coordination failures you're seeing today won't go away when the next model drops.** In fact, if the new model is faster and more confident, it'll produce more code per session and the conflicts will be worse. We've seen this in production — 30 individually correct commits grew a single file by 892 lines in 60 days. Not from bad agents. From excellent ones.

---

## Reproduce It Yourself

The experiment takes about 5 minutes to run and requires only the Claude CLI. Here's the minimal version:

### Prerequisites

- [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) installed
- A git repository with at least one source file
- A task that requires modifying that file

### Steps

**1. Pick a task.** Something simple and well-specified — add a function, write tests. The task should have a clear insertion point in an existing file.

**2. Create two worktrees from the same commit:**

```bash
BASELINE=$(git rev-parse HEAD)
git worktree add -b agent-a /tmp/agent-a $BASELINE
git worktree add -b agent-b /tmp/agent-b $BASELINE
```

**3. Run both agents with the same prompt:**

```bash
# Terminal 1
cd /tmp/agent-a
claude --dangerously-skip-permissions -p "$(cat task-prompt.md)"

# Terminal 2
cd /tmp/agent-b
claude --dangerously-skip-permissions -p "$(cat task-prompt.md)"
```

**4. Try to merge:**

```bash
cd /tmp/agent-a
git merge agent-b --no-edit
```

**5. Observe the conflict.**

If your task requires modifying the same file at the same insertion point (which is the natural structure of most "add a function" tasks), you'll get a conflict. The agents didn't do anything wrong. The system did.

### For a more rigorous run

We have a [full experiment runner](https://github.com/dylanconlin/orch-go/tree/master/experiments/coordination-demo) that automates N trials with scoring and merge analysis. The scripts handle worktree creation, parallel execution, 6-dimension scoring, and statistical analysis.

```bash
# Run 10 trials (haiku + opus per trial, ~8 min total)
bash experiments/coordination-demo/run.sh 10
```

---

## What Actually Fixes This

If model upgrades don't fix coordination failures, what does? Structural solutions:

**Sequential execution for overlapping files.** When two tasks target the same file, run them one after the other. The second agent sees the first agent's work. This is the simplest solution and eliminates the structural conflict entirely. The cost is throughput — you lose parallelism for file-overlapping work.

**File-level work assignment.** Don't give two agents tasks that modify the same file simultaneously. This requires knowing what files a task will touch before running it, which isn't always possible. But for many tasks, it's predictable enough.

**Structural attractors.** When shared code lands in the same file because no better destination exists, create one. Shared packages with clear names pull code toward them by convention. We extracted `pkg/spawn/backends/` and the source file shrank by 1,755 lines. The attractor relocates the gravitational center so agents stop piling code into the same function.

**Gates.** Pre-commit hooks that block commits to files over a size threshold. Spawn gates that refuse to create agents targeting degraded files. These don't fix the coordination problem directly, but they make accretion visible and create friction that forces the structural conversation.

The pattern: coordination failures require architectural responses, not model-level responses. Better agents don't merge more cleanly. Better architecture makes merging unnecessary.

---

## The Bigger Picture

This experiment is the smallest version of a larger observation. We run 50+ AI agent sessions per day on a single codebase. The coordination failure demonstrated here — two agents, one task, structural conflict — scales predictably. More agents, more tasks, more conflicts. The entropy doesn't asymptote. It accelerates.

We've been writing about this under the frame of [harness engineering](https://dylanconlin.com/blog/harness-engineering) — the discipline of building structural governance for multi-agent codebases. The demo here is the foundational claim: coordination failure is structural, not cognitive, and it doesn't resolve with model improvement.

But coordination failure isn't unique to code. We've observed the same dynamics — agents independently producing correct work that conflicts when integrated — in knowledge systems where agents contribute to shared understanding rather than shared files. The physics appear to be substrate-independent. What changes across substrates is the conflict surface (merge conflicts in code, contradictory claims in knowledge, incompatible schemas in databases), not the underlying dynamic.

That's what we're exploring next. The coordination failure demo is the empirical anchor. The theory is what it points toward.

---

*Data: 20 agent runs (10 Haiku, 10 Opus), 100% individual success, 100% merge conflict, Fisher's exact p=1.0. Plus 1 complex/ambiguous trial (4-file, 10-dimension scoring). Full experiment scripts, results, and scoring rubrics at [experiments/coordination-demo/](https://github.com/dylanconlin/orch-go/tree/master/experiments/coordination-demo). Built with [orch-go](https://github.com/dylanconlin/orch-go), a multi-agent orchestration system running ~50 AI agents/day.*
