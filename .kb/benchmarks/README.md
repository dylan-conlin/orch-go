# AI Model Benchmarks

**Purpose:** Empirical evidence for model selection decisions in AI orchestration.

## Why Collect Benchmarks?

Model selection guides (`.kb/guides/model-selection.md`) need empirical backing, not vibes. These benchmarks provide:

1. **Same task, multiple models** - Controlled comparison
2. **Preserved states** - Reproducible via git branches
3. **Failure mode analysis** - Understanding WHY models fail
4. **Actionable insights** - Direct implications for `orch spawn` decisions

## Benchmark Format

Each benchmark should include:

```markdown
# Benchmark: {Task Name} - {N} Model Comparison

**Status:** Complete/In Progress
**Created:** YYYY-MM-DD
**Type:** Benchmark / Model Performance Comparison
**Source:** {project where benchmark was run}

## Summary
{Results table with Model, Time, Result, Approach}

## The Problem
{What task were models given?}

## Key Findings
{What did we learn about model behavior?}

## Preserved States
{Git branches or transcripts for reproducibility}

## Model Details
{Per-model breakdown: time, tokens, approach, outcome}

## Orchestration Implications
{How should this inform orch spawn decisions?}
```

## Running New Benchmarks

1. **Choose a well-defined task** - Clear success criteria
2. **Document the prompt** - Exact same prompt to each model
3. **Run sequentially** - Fresh state for each model
4. **Preserve states** - Git branches, transcripts
5. **Analyze failure modes** - Why did models fail, not just that they failed
6. **Extract implications** - What should change in orchestration?

## Benchmarks

| Date | Task | Models | Key Finding |
|------|------|--------|-------------|
| 2026-01-28 | Logout fix | 6 | Codex + DeepSeek succeeded; most failed with frontend-only fixes |

## Discovery

```bash
kb context "benchmark"
kb context "model comparison"
```
