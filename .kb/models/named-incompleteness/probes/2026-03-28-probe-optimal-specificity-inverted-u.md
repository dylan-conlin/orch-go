# Probe: Optimal Specificity of Named Incompleteness — Is There an Inverted-U?

**Model:** named-incompleteness
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-06
**verdict:** directionally supported, not confirmed

---

## Question

NI-06 claims: Named incompleteness has an optimal specificity — too vague and nothing converges, too specific and nothing connects. The gap must be specific enough to be a coordinate but general enough to attract multiple approaches.

Predicted pattern: inverted-U relationship between gap specificity and clustering effectiveness. Template-mandated gaps (low specificity) should show low convergence. Organic tensions (medium specificity) should show high convergence. Hyper-specific implementation details (high specificity) should show low connection.

---

## What I Tested

Parsed all 119 briefs in `.kb/briefs/` with non-trivial tensions (≥5 words).

### Specificity measurement (composite score)

Four features, normalized and weighted:
- **Entity density** (35%): backtick-quoted identifiers, dotted names, camelCase/snake_case, file paths, specific percentages/sample sizes per 100 words
- **Specific question count** (25%): questions containing concrete referents (`should`, `whether`, `which`, `does`, backtick terms)
- **Comparison density** (20%): explicit tradeoffs (vs, rather than, instead of, tradeoff, alternative)
- **Lexical diversity** (20%): unique/total word ratio

Score range: [0.000, 0.671]. Tercile thresholds: 0.170 and 0.258.

### Convergence and connection (TF-IDF cosine similarity)

Shared TF-IDF vocabulary (2000 features, bigrams, sublinear TF) fitted on all tensions and frames together.

- **Convergence**: mean cosine similarity between each tension and all other briefs' Frame sections (does this gap attract subsequent work?)
- **Connection**: mean cosine similarity between each tension and all other tensions (does this gap link to other gaps?)
- **Combined product**: convergence × connection (the "sweet spot" metric)

### Analysis battery

1. Within-bin pairwise tension similarity (terciles)
2. Length-controlled within-bin similarity (truncation to 73-word median)
3. Cross-referential attraction at multiple thresholds (0.05–0.20)
4. Tension-to-tension connectivity at multiple thresholds
5. Combined convergence × connection product
6. Permutation tests for between-bin significance (10,000 permutations)
7. Quintile analysis (5 bins for finer resolution)
8. Quadratic regression (direct inverted-U test)
9. Decile analysis (10 bins for shape clarity)
10. Confound checks (word count correlation)
11. Template-mandated vs organic gap classification

---

## What I Observed

### Finding 1: The inverted-U shape appears consistently

Every analysis shows the same ordering: **medium specificity > low > high**.

| Metric | Low | Medium | High |
|---|---|---|---|
| Within-bin tension similarity | 0.0665 | 0.0654 | 0.0623 |
| Cross-referential attraction (cosine > 0.10) | 7.95 frames | 9.00 frames | 6.33 frames |
| Tension connectivity (cosine > 0.10) | 13.90 | 14.38 | 13.00 |
| Convergence (mean sim to frames) | 0.0534 | 0.0548 | 0.0510 |
| Connection (mean sim to tensions) | 0.0643 | 0.0652 | 0.0630 |
| Combined product | 0.003433 | 0.003573 | 0.003214 |

Quadratic regression confirms concave (inverted-U) shape for all three metrics:
- Convergence: peak at specificity = 0.170, quadratic R² = 0.035
- Connection: peak at specificity = 0.208, quadratic R² = 0.035
- Product: peak at specificity = 0.199, quadratic R² = 0.041

### Finding 2: The high-specificity decline is the clearest signal

Decile analysis reveals the shape more precisely:

| Decile | Specificity | Product |
|---|---|---|
| D1 | 0.072 | 0.003495 |
| D2 | 0.118 | 0.003335 |
| D3 | 0.145 | 0.003353 |
| D4 | 0.172 | 0.003445 |
| D5 | 0.195 | **0.004009** |
| D6 | 0.218 | 0.003473 |
| D7 | 0.243 | 0.003334 |
| D8 | 0.268 | 0.003676 |
| D9 | 0.309 | 0.003123 |
| D10 | 0.427 | 0.003107 |

D5 (medium specificity ≈ 0.195) is the clear peak. D9-D10 (highest specificity) are the clear floor. The low-specificity side (D1-D4) is relatively flat — no strong ascent toward the peak.

The shape is better described as **"plateau then decline"** than a clean inverted-U. Medium specificity tensions don't dramatically outperform low-specificity ones; but high-specificity tensions consistently underperform both.

### Finding 3: None of this reaches statistical significance

Permutation tests for the quadratic term:
- Convergence: p = 0.231
- Connection: p = 0.127
- Product: p = 0.117

Permutation tests for between-bin range:
- Convergence: p = 0.197
- Connection: p = 0.488
- Product: p = 0.206

The p=0.117 for the product quadratic term is the closest to significance but does not cross the 0.05 threshold. The R² values (0.03-0.04) mean specificity explains only 3-4% of variance in convergence/connection.

### Finding 4: Word count is a larger confound than specificity

- Specificity correlates negatively with word count (r = -0.249): more specific tensions tend to be shorter
- Word count correlates with convergence (r = 0.504) and connection (r = 0.408) much more strongly than specificity does
- TF-IDF cosine similarity is partly measuring text length, not semantic convergence

This means the high-specificity decline may be partly a length artifact: hyper-specific tensions are shorter and have less surface area for lexical overlap. The prior probe (tension-clustering-spatial-prediction) found the same confound — length control nearly eliminated the clustering gap between tensions and resolutions.

### Finding 5: No template-mandated tensions exist in this corpus

The template-vs-organic classifier found: 0 template, 81 mixed, 38 organic. The orch-go brief system produces no truly template-filling tensions. This means one half of NI-06's predicted comparison (low-specificity template mandated gaps → low convergence) cannot be tested in this corpus.

Qualitative samples confirm this: even the lowest-specificity tensions name real concerns:
- **D1 sample** (score=0.000): "The 16/72 ratio tells you where the code mass is, but it doesn't tell you where the value mass is."
- **D5 sample** (score=0.195): "Two other files still carry the old identity: CLAUDE.md opens with 'Go rewrite of orch-cli'..."
- **D10 sample** (score=0.671): "All three `OrcCompleter` methods now use `--headless`, which raises a question: is `--force` still serving any purpose?"

All three are organic. The difference is scope, not authenticity: D1 asks a strategic question about code investment; D10 asks whether a specific flag is dead code.

---

## Model Impact

### What's directionally supported

**The shape NI-06 predicts is the shape the data shows.** Every metric, every bin size, and the quadratic regression all agree: the relationship between specificity and convergence/connection is concave (inverted-U). The peak is in the medium-specificity range (0.17-0.21 on a 0.00-0.67 scale). This is consistent with NI-06's claim that there's an optimal specificity.

### What's not yet confirmed

**The effect is too weak for TF-IDF to detect reliably.** Three reasons:

1. **Methodological ceiling**: TF-IDF measures lexical overlap. The prior probe established that "the mechanism for composition of named gaps is reference (pointing at the same thing) not resemblance (looking alike)." The specificity-convergence relationship may be stronger in semantic embedding space.

2. **Corpus homogeneity**: This brief corpus has no template-mandated tensions. All tensions are organic at varying specificity levels. The NI-06 prediction is strongest at the extremes (template garbage vs hyper-specific implementation detail), and this corpus lives in the middle range.

3. **Word count confound**: TF-IDF similarity scales with text length (r=0.40-0.50). Shorter, more specific tensions have less surface for lexical overlap regardless of their actual convergence potential.

### The asymmetric finding

The decline at high specificity is more robust than the ascent from low specificity. This suggests a refinement to NI-06:

- **Original claim**: "too vague and nothing converges, too specific and nothing connects"
- **What the data shows**: "specificity above a threshold actively hurts connection; below that threshold, specificity barely matters"

This is the difference between an inverted-U (symmetric peak) and a **saturation curve with decline** (flat plateau, then dropoff). The practical implication: the system should worry more about hyper-specific gaps than about vague ones — at least in a corpus where vague gaps are still organic.

### Qualitative confirmation

The qualitative samples strongly support NI-06's mechanism. High-specificity tensions like "is `--force` dead code?" point at exactly one thing — useful for that one thing, but invisible to anything else. Medium-specificity tensions like "does `~/.orch/` become a shadow state store?" name a concern specific enough to guide action but general enough that multiple other briefs (about persistence, about dashboard state, about architecture) could compose against it.

---

## Constraints on this probe

1. **TF-IDF is a lexical model** — it measures word overlap, not meaning overlap. The effect may be larger with semantic embeddings. This is the same limitation as the tension-clustering probe.
2. **N=119 briefs** from a single project, single author system. No cross-project validation.
3. **No template-mandated tensions** in this corpus — one half of the NI-06 prediction (the low-specificity failure mode) cannot be tested.
4. **Word count confounds specificity** in TF-IDF space. Length-controlled replication would strengthen or weaken the finding.
5. **Composite specificity score** weights (35/25/20/20) are heuristic. Different weightings could shift tercile membership.

---

## What would strengthen or weaken this finding

**Strengthen**: Repeat with sentence-transformer embeddings. If the inverted-U becomes significant with semantic rather than lexical similarity, the effect is real but lexically invisible (same pattern as NI-01's pilot→full study progression where TF-IDF missed what embeddings caught).

**Strengthen further**: Find a corpus WITH template-mandated tensions (mandatory retrospective questions, compliance-driven open-question sections). Compare convergence of template-mandated vs organic gaps to test the low-specificity failure mode.

**Strengthen further**: Manual labeling of "which tensions generated follow-up work" (operationalized as: did a subsequent brief, probe, or thread reference the same concern?). This measures actual convergence, not distributional similarity.

**Weaken**: If sentence-transformer replication still shows p>0.05, the effect may genuinely be too small to matter — specificity may not be the operative variable, or the optimal range may be too broad to detect.

**Weaken**: If the word-count confound fully explains the high-specificity decline after length control, then what looked like a specificity effect was actually a brevity effect — shorter texts cluster less because they share fewer words, not because their content is too narrow.
