# Investigation: signal_count Distribution Across Actual Briefs

**Date:** 2026-03-27
**Issue:** orch-go-iw4aw
**Status:** Complete

---

## Question

The orch-go-rhwly design proposes sorting the comprehension queue by `signal_count` (sum of 6 boolean quality signals). The design itself flags the risk: "If 80% of briefs score 5 or 6, the ordering adds nothing." Does the distribution actually discriminate?

## Method

Measured 6 quality signals across all 70 briefs in `.kb/briefs/`:

| Signal | What it detects |
|--------|----------------|
| `structural` | Frame + Resolution + Tension all populated (>20 chars each) |
| `evidence` | 2+ specific file paths, commits, or test references |
| `model_conn` | References to `.kb/models/` or known model names |
| `connective` | 2+ causal/connective phrases (because, which means, it turns out...) |
| `tension_q` | Tension section contains actual questions (has `?` and >30 chars) |
| `insight` | <30% of content lines are action-verb-only (Added/Fixed/Updated...) |

## Findings

### Distribution: signal_count spreads well (mean 2.9/6)

```
  0/6:   0 (  0.0%)
  1/6:   4 (  5.7%)  ████
  2/6:  20 ( 28.6%)  ████████████████████
  3/6:  27 ( 38.6%)  ███████████████████████████
  4/6:  16 ( 22.9%)  ████████████████
  5/6:   3 (  4.3%)  ███
  6/6:   0 (  0.0%)
```

**The feared 80%-at-5-6 clustering does not exist.** The distribution is roughly normal, centered at 3/6. Low-quality briefs (0-2) are 34%, mid (3) is 39%, high (4-6) is 27%. The sort key has real discriminating power.

### Per-signal prevalence and discriminating power

| Signal | Fired | Prevalence | Discriminating Power |
|--------|-------|-----------|---------------------|
| `structural` | 70/70 | 100% | **USELESS** — every brief has all 3 sections |
| `evidence` | 13/70 | 18.6% | WEAK — fires too rarely, most briefs score 0 |
| `model_conn` | 7/70 | 10.0% | **USELESS** — fires so rarely it barely affects ordering |
| `connective` | 33/70 | 47.1% | **GOOD** — near-perfect 50/50 split |
| `tension_q` | 24/70 | 34.3% | **GOOD** — meaningful split |
| `insight` | 57/70 | 81.4% | WEAK — fires too often, most briefs score 1 |

**Two signals carry the discriminating load:** `connective` and `tension_q`. These two alone produce a 3-way split (0, 1, or 2 of these fired) that captures most of the ordering value.

**Two signals are structurally useless:**
- `structural` fires 100% — it should be dropped or replaced. Brief composition already guarantees Frame/Resolution/Tension structure.
- `model_conn` fires 10% — it barely affects sort order. Only 7 briefs reference models.

**Two signals are weak but contribute at the margins:**
- `evidence` (18.6%) and `insight` (81.4%) are skewed but not fully degenerate. They move a few briefs up or down.

### The 5/6 briefs are the composition briefs

The 3 briefs scoring 5/6 are: orch-go-2mfvw (HyperAgents external validation), orch-go-3tyik, orch-go-c5ha1. Spot-checking orch-go-2mfvw confirms it's a composition-quality brief — it connects findings to models, reasons causally, and surfaces open questions. The scoring tracks actual quality.

## Implications

1. **signal_count works as a sort key.** The distribution has enough spread (1-5 range, roughly normal at 2.9) to meaningfully reorder the comprehension queue. The design can proceed.

2. **Two signals should be replaced or recalibrated:**
   - `structural` (100% prevalence) adds zero information. Replace with something that measures structural *quality* not structural *presence* — e.g., resolution length ratio (resolutions that are substantive vs one-liner).
   - `model_conn` (10% prevalence) is too rare to matter in a 6-signal count. Consider broadening to "cross-reference" (any reference to other briefs, investigations, threads, or decisions — not just models).

3. **A 4-signal version might work better:** Drop `structural` and `model_conn`, keep `evidence`, `connective`, `tension_q`, `insight`. The 4-signal count would range 0-4 with better balance per signal.

4. **The connective+tension_q pair is the core discriminator.** If implementation is phased, start with just these two signals — they produce a 3-tier ordering that captures most of the value with minimal code.

## Addendum: Frontmatter Gap (2026-03-27, orch-go-5qiv1)

**The above analysis measures what signal_count *would* produce. In practice, signal_count is dead in production.**

### The gap

`serve_briefs.go:260` reads `signal_count` from YAML frontmatter via `ParseBriefSignalCount`. But **0 of 82 briefs have YAML frontmatter** — every brief starts with `# Brief:`, not `---`. `ParseBriefSignalCount` returns 0 for all briefs.

### Why no briefs have frontmatter

Two brief generation paths exist:

| Path | File | When Used | Adds Frontmatter? |
|------|------|-----------|-------------------|
| `CopyBrief` | lifecycle_adapters.go:228 | Lifecycle manager during agent cleanup | **No** — copies BRIEF.md verbatim |
| `generateHeadlessBrief` | complete_brief.go:20 | `orch complete --headless` only | **Yes** — computes quality signals from SYNTHESIS.md |

All 82 existing briefs were produced via `CopyBrief`. The `generateHeadlessBrief` path requires SYNTHESIS.md (skipped by light-tier spawns), and most completions go through the lifecycle manager, not `orch complete --headless`.

### Effective sort order today

Since all briefs have `signal_count = 0`, the 3-tier sort in `serve_briefs.go:265-282` collapses to:
1. Unread before read
2. ~~signal_count desc~~ (no-op — all zero)
3. Newest first

### Fix required

The signal computation needs to move from write-time (in `generateHeadlessBrief`) to read-time (in `serve_briefs.go`), or `CopyBrief` needs to inject frontmatter. Without this, the signal infrastructure exists but never fires.
