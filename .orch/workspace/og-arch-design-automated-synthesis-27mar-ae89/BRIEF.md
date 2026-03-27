# Brief: orch-go-rhwly

## Frame

When you read 37 briefs in a sitting and some are transformative while others just say "I fixed the thing," the question forms naturally: could the system tell you which ones to read first? Right now the comprehension queue sorts by time. The good briefs — the ones that change how you think about the system — are scattered between task reports. This design asks whether the difference between a composing-knowledge brief and a task-completion brief is mechanically detectable, and whether that detection can drive reading order.

## Resolution

It turns out the Synthesis parser already extracts exactly the fields that distinguish good from mediocre briefs — evidence specificity, model connections, causal reasoning, open questions. The quality checks in debrief already detect connective language ("because," "which means") and flag action-verb-only summaries ("Added X, Fixed Y"). The infrastructure exists; it's just not wired to ordering.

The design is three extensions to existing code, not a new subsystem: compute 6 quality signals from parsed SYNTHESIS.md at completion time, embed them as YAML frontmatter in the generated brief, sort the comprehension queue by signal count within each state tier. A brief that cites evidence, connects to models, reasons causally, and surfaces open questions gets read before a brief that just records what was done.

The hardest choice was score representation. The knowledge accretion model's own warning about "formula-shaped sentences" — numbers that look like measurements but have no units — applied directly. And the HyperAgents paper gave the nudge: their self-modified selection mechanism didn't outperform the handcrafted one. So: boolean signals with a simple count, not a weighted score. If calibration data arrives later through brief feedback, we can test whether weighting helps. The HyperAgents evidence predicts it won't.

## Tension

The design assumes signal count has discriminating power — that briefs actually spread across 0/6 to 6/6. If 80% of briefs score 5 or 6, the ordering adds nothing. This is an empirical question that can't be answered by design analysis. One way to find out: run `ComputeSynthesisQuality` retroactively on the 50 existing briefs and look at the distribution. If it's bimodal (task reports cluster low, composition briefs cluster high), the design works. If it's uniform-high, the signals are too easy to trigger and need recalibration.
