### Coherence Over Patches

When fixes accumulate in the same area, the problem isn't insufficient fixing - it's a missing coherent model. Escalate to architect before the next patch.

**The test:** "Is this the 3rd fix to the same file this week?"

**What this means:**

- 5+ fix commits to same file signals structural issues, not bad luck
- 10+ conditions in single logic block signals missing design
- Multiple investigations on same topic without resolution signals unclear model
- The next patch will make things worse, not better

**What this rejects:**

- "One more fix should do it" (the fix is correct, the approach is wrong)
- "We don't have time to redesign" (you don't have time NOT to)
- "Each agent fixed their bug correctly" (local correctness, global incoherence)

**The failure mode:** Each patch is locally correct. Agent A fixes the null check. Agent B fixes the edge case. Agent C fixes the timing issue. Agent D fixes the duplicate check. The file now has 10+ conditions scattered across 350 lines. Each fix was real. The code is incoherent.

**Signals that trigger this:**

| Signal | Threshold | Action |
|--------|-----------|--------|
| Fix commits to same file | 5+ in 4 weeks | Recommend architect |
| Conditions in logic block | 10+ | Needs coherent model |
| Investigations on topic | 3+ without Guide | Needs synthesis |
| Same bug pattern recurring | 2+ variations | Structural issue |
