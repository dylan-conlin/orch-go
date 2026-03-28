# Probe: Do Tensions Cluster Tighter Than Conclusions in orch-go's Brief Corpus?

**Model:** named-incompleteness (generative-systems-are-organized-around)
**Date:** 2026-03-28
**Status:** Complete
**claim:** NI-01
**verdict:** disconfirms (with extension)

---

## Question

NI-01 claims: "Systems that accumulate named gaps self-organize. Systems that accumulate conclusions require external triage. The named gap is the compositional signal."

The model's information-theoretic argument says gaps are "specific coordinates in possibility space" while conclusions are "generic." Spatial prediction: tension/question atoms from briefs should have higher within-topic cosine similarity than conclusion/finding atoms from the same briefs — because tensions converge on shared coordinates while resolutions scatter.

---

## What I Tested

Parsed all 103 briefs in `.kb/briefs/`. Every brief has consistent Frame/Resolution/Tension sections.

### Analysis 1: Global pairwise similarity (TF-IDF cosine)
- Vectorized tensions and resolutions with shared TF-IDF vocabulary (2000 features, bigrams, sublinear TF)
- Computed all pairwise cosine similarities within each set

### Analysis 2: Within-topic similarity (KMeans topic assignment)
- Clustered on Frame text (topic assignment independent of measured sections)
- Measured within-topic cosine similarity for tensions vs resolutions
- Tested at k=5, 7, 10, 15 for robustness

### Analysis 3: Length-controlled replication
- Tensions average 73 words; resolutions average 160 words
- Control 1: Truncated resolutions to match tension word count distribution
- Control 2: Binary TF-IDF (term presence only, removes frequency effects)
- Control 3: Random 73-word window from each resolution

### Analysis 4: Specificity measures
- Gini coefficient of TF-IDF weights
- Max TF-IDF weight per document
- Entropy of TF-IDF distribution
- Lexical diversity (unique/total words)
- Cross-document term sharing rate

---

## What I Observed

### Clustering: Resolutions cluster tighter (pre-control)

| Metric | Tensions | Resolutions | Difference |
|---|---|---|---|
| Global mean cosine sim | 0.0272 | 0.0479 | -0.0207 |
| k=5 within-topic | 0.0362 | 0.0596 | -0.0234 |
| k=7 within-topic | 0.0400 | 0.0645 | -0.0245 |
| k=10 within-topic | 0.0448 | 0.0734 | -0.0286 |
| k=15 within-topic | 0.0487 | 0.0830 | -0.0343 |

Every analysis: resolutions cluster tighter. Effect size: medium (d ≈ -0.43). In the paired analysis, 74% of brief pairs show R>T. Zero clusters (out of 37 across all k values) show T>R except one marginal case at k=15.

### Length control nearly eliminates the clustering gap

| Control | Tension sim | Resolution sim | Difference | p-value |
|---|---|---|---|---|
| Truncated to match | 0.0292 | 0.0306 | -0.0014 | 0.95 |
| 73-word window | 0.0294 | 0.0298 | -0.0005 | 0.79 |
| Binary TF-IDF | 0.0274 | 0.0465 | -0.0192 | 1.00 |

**Most of the clustering gap is a text-length artifact.** After controlling for length (controls 1 and 3), the difference is essentially zero and non-significant. Binary TF-IDF still shows R>T because vocabulary breadth is length-correlated.

### Specificity: Tensions ARE more specific (strongly)

| Measure | Tensions | Resolutions | p-value |
|---|---|---|---|
| Gini coefficient | 0.9851 | 0.9716 | < 0.001 |
| Max TF-IDF weight | 0.2984 | 0.2533 | < 0.001 |
| Entropy | 5.014 | 5.945 | < 0.001 |
| Lexical diversity | 0.830 | 0.748 | < 0.001 |
| Non-zero features | 35.0 | 67.8 | < 0.001 |

All specificity measures are significant at p < 0.001. Tensions genuinely concentrate their weight on fewer, more specific terms. They use more unique vocabulary relative to their total word count.

### Distinctive vocabulary

Tension-distinctive terms: "worth," "question," "judgment," "watching," "worth watching," "wants," "need," "approach" — these are the language of open inquiry, pointing forward.

Resolution-distinctive terms: "fix," "file," "turn," "instead," "surprise," "layer," "execution," "code," "tests" — these are the language of concrete implementation, describing what was done.

---

## Model Impact

### What's confirmed

The NI model's claim that **gaps are specific coordinates** — confirmed. Tensions are lexically more concentrated, have lower entropy, and use more specialized vocabulary. Each tension is about ONE specific thing.

### What's disconfirmed

The spatial prediction that **specific coordinates cluster tighter** — disconfirmed. Despite being more specific individually, tensions do NOT have higher pairwise cosine similarity than resolutions. After controlling for text length, neither section type clusters tighter than the other.

### The paradox: specificity ≠ proximity

The model conflates two distinct spatial properties:

1. **Specificity** (each gap points at one coordinate) — ✓ confirmed
2. **Proximity** (those coordinates are near each other) — ✗ disconfirmed

Tensions are specific but they're specific about **different** things. Each tension points to a unique gap in possibility space. "Is this fixable at the skill level or is it a fundamental model limitation?" and "Does `~/.orch/` become a shadow state store?" are both highly specific but point at completely different coordinates. They don't cluster in distributional space because they DON'T share vocabulary — each gap is genuinely unique.

Resolutions share more vocabulary not because they're "at the same coordinate" but because they describe **similar processes**: fixing, implementing, refactoring. The shared process vocabulary creates distributional proximity independent of topic. Two resolutions about completely different features can cluster because both say "the fix was straightforward — thread the parameter through three functions."

### What this means for NI's "self-organization" claim

The model's claim that "systems accumulating named gaps self-organize" may still be correct — but **the mechanism isn't distributional proximity**. "Self-organization" in the NI sense might mean:

- **Graph connectivity**: Tension A creates the context for Tension B (one gap's resolution opens another). This is compositional structure, not vector similarity.
- **Attentional convergence**: Multiple agents converge on the same gap because it's named, not because it's distributionally similar to other gaps.
- **Generative chaining**: The resolution of one specific gap opens specific new gaps — the chain is the self-organization, not the clustering.

TF-IDF cosine similarity measures **distributional proximity** (shared words). The NI model's "coordinates in possibility space" may be about **referential structure** (what a gap points at) rather than distributional similarity (what words it uses).

### Recommended model update

The "spatial" language in NI's information-theoretic argument needs refinement:

- **Current claim**: "Two questions about the same gap converge because the gap is one coordinate."
- **Refined claim**: "Two questions about the same gap compose because the gap creates a **referential link** between them — not because they use similar vocabulary."

The mechanism for composition of named gaps is **reference** (pointing at the same thing) not **resemblance** (looking alike). This is a meaningful distinction that the model should incorporate.

---

## Constraints on this probe

1. **TF-IDF is a bag-of-words model** — it can't capture reference or compositional structure, only lexical overlap. The spatial prediction may hold under semantic embeddings (e.g., sentence transformers) that capture meaning rather than vocabulary.
2. **N=103 briefs** from a single project, single author system. No cross-project validation.
3. **Topic assignment via KMeans on frames** is a noisy proxy for true topic labels.
4. **The "same gap" test requires manual labeling** — which briefs share a tension about the same underlying gap? This probe tested aggregate distributional similarity, not targeted "same-gap" pairs. A manual-label version could still confirm NI-01 if specific gap-sharing pairs have high tension similarity.

---

## What would strengthen or weaken this finding

**Strengthen:** Repeat with sentence-transformer embeddings (semantic, not lexical). If tensions STILL don't cluster tighter, the distributional prediction is robustly wrong. If they DO cluster tighter with semantic embeddings but not TF-IDF, the model's insight is about meaning-level structure not vocabulary-level structure.

**Strengthen further:** Manually identify brief pairs that share a named gap (e.g., multiple briefs about "orchestrator frustration detection" or "brief comprehension UI"). Measure whether same-gap tension pairs have higher similarity than random-gap tension pairs. This directly tests "two questions about the same gap converge."

**Weaken:** If the model's "self-organization" is purely about reference chains (A→B→C) rather than distributional clustering, then this probe's framing was wrong — it tested the wrong kind of spatial structure.
