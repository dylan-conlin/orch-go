# Decomposition Quality Experiment Results

Generated: Tue Mar 24 12:17:38 PDT 2026

## Hypothesis

Decomposition quality (task description + file structure) predicts conflict rate. Best decomposition (anchored-sectioned) should approach 0% conflict without any coordination primitives.

## Merge Results by Condition

| Condition | Trials | Conflicts | Clean Merge | Build Fail | Semantic Fail | No Change | Conflict % |
|-----------|--------|-----------|-------------|------------|---------------|-----------|------------|
| bare-flat | 10 | 10 | 0 | 0 | 0 | 0 | 100% |
| rich-flat | 10 | 10 | 0 | 0 | 0 | 0 | 100% |
| anchored-flat | 10 | 10 | 0 | 0 | 0 | 0 | 100% |
| bare-sectioned | 10 | 8 | 2 | 0 | 0 | 0 | 80% |
| anchored-sectioned | 10 | 6 | 4 | 0 | 0 | 0 | 60% |

## Individual Agent Success Rates

| Condition | Agent | Avg Score | Perfect (6/6) | Trials |
|-----------|-------|-----------|---------------|--------|
| bare-flat | a | 6.0/6 | 10/10 | 10 |
| bare-flat | b | 6.0/6 | 10/10 | 10 |
| rich-flat | a | 6.0/6 | 10/10 | 10 |
| rich-flat | b | 6.0/6 | 10/10 | 10 |
| anchored-flat | a | 6.0/6 | 10/10 | 10 |
| anchored-flat | b | 6.0/6 | 10/10 | 10 |
| bare-sectioned | a | 6.0/6 | 10/10 | 10 |
| bare-sectioned | b | 6.0/6 | 10/10 | 10 |
| anchored-sectioned | a | 6.0/6 | 10/10 | 10 |
| anchored-sectioned | b | 6.0/6 | 10/10 | 10 |

## Duration Summary

- **bare-flat**: Agent A avg=43s, Agent B avg=42s
- **rich-flat**: Agent A avg=42s, Agent B avg=40s
- **anchored-flat**: Agent A avg=45s, Agent B avg=44s
- **bare-sectioned**: Agent A avg=39s, Agent B avg=46s
- **anchored-sectioned**: Agent A avg=38s, Agent B avg=44s

## Anchoring Analysis

Where did each agent's changes land? Lower variance = stronger anchoring.

### bare-flat

- Trial 1: A=[92] B=[92]
- Trial 10: A=[92] B=[92]
- Trial 2: A=[92] B=[92]
- Trial 3: A=[92] B=[92]
- Trial 4: A=[92] B=[92]
- Trial 5: A=[92] B=[92]
- Trial 6: A=[92] B=[92]
- Trial 7: A=[92] B=[92]
- Trial 8: A=[92] B=[92]
- Trial 9: A=[92] B=[92]

Agent A anchoring: mean=92, variance=0 (N=10)
Agent B anchoring: mean=92, variance=0 (N=10)

### rich-flat

- Trial 1: A=[92] B=[92]
- Trial 10: A=[92] B=[92]
- Trial 2: A=[92] B=[92]
- Trial 3: A=[92] B=[92]
- Trial 4: A=[92] B=[92]
- Trial 5: A=[92] B=[92]
- Trial 6: A=[92] B=[92]
- Trial 7: A=[92] B=[92]
- Trial 8: A=[92] B=[92]
- Trial 9: A=[92] B=[92]

Agent A anchoring: mean=92, variance=0 (N=10)
Agent B anchoring: mean=92, variance=0 (N=10)

### anchored-flat

- Trial 1: A=[92] B=[92]
- Trial 10: A=[92] B=[92]
- Trial 2: A=[92] B=[92]
- Trial 3: A=[92] B=[92]
- Trial 4: A=[92] B=[92]
- Trial 5: A=[92] B=[92]
- Trial 6: A=[92] B=[92]
- Trial 7: A=[92] B=[92]
- Trial 8: A=[92] B=[92]
- Trial 9: A=[92] B=[92]

Agent A anchoring: mean=92, variance=0 (N=10)
Agent B anchoring: mean=92, variance=0 (N=10)

### bare-sectioned

- Trial 1: A=[10] B=[10]
- Trial 10: A=[10] B=[10]
- Trial 2: A=[10] B=[10]
- Trial 3: A=[10] B=[10]
- Trial 4: A=[10] B=[10]
- Trial 5: A=[10] B=[5]
- Trial 6: A=[10] B=[10]
- Trial 7: A=[10] B=[10]
- Trial 8: A=[10] B=[10]
- Trial 9: A=[10] B=[10]

Agent A anchoring: mean=10, variance=0 (N=10)
Agent B anchoring: mean=9, variance=2 (N=10)

### anchored-sectioned

- Trial 1: A=[10] B=[10]
- Trial 10: A=[10] B=[10]
- Trial 2: A=[10] B=[10]
- Trial 3: A=[10] B=[10]
- Trial 4: A=[10] B=[10]
- Trial 5: A=[10] B=[10]
- Trial 6: A=[10] B=[10]
- Trial 7: A=[10] B=[10]
- Trial 8: A=[10] B=[10]
- Trial 9: A=[10] B=[10]

Agent A anchoring: mean=10, variance=0 (N=10)
Agent B anchoring: mean=10, variance=0 (N=10)


## Comparison with Prior Data

| Experiment | Condition | Conflict Rate | N |
|------------|-----------|---------------|---|
| Prior (Mar 10) | no-coord additive | 100% | 20 |
| Prior (Mar 10) | placement additive | 0% | 20 |
| Prior (Mar 23) | no-coord modification | 0% | 40 |
| This (decomp) | bare-flat | 100% | 10 |
| This (decomp) | rich-flat | 100% | 10 |
| This (decomp) | anchored-flat | 100% | 10 |
| This (decomp) | bare-sectioned | 80% | 10 |
| This (decomp) | anchored-sectioned | 60% | 10 |

## Interpretation Guide

- If bare-flat ~100%: Confirms baseline (replicates prior no-coord)
- If anchored-sectioned ~0%: Decomposition quality CAN eliminate coordination need
- If gradient is monotonic (C1>C2>C3>C4>C5): Relationship is continuous
- If anchored-flat alone ~0%: File structure doesn't matter, task descriptions suffice
- If anchored-sectioned still >50%: Decomposition hypothesis is wrong
