# Probe: Coupling Hotspot Detection Gap in Accretion Enforcement

**Date:** 2026-02-19
**Status:** Complete
**Model:** completion-verification
**Issue:** orch-go-1109

## Question

Does the current accretion enforcement architecture (4-layer design from 2026-02-14 investigation) address cross-surface coupling, or does it only detect single-file accretion?

## What I Tested

1. Ran real co-change analysis on 2,733 commits (90 days) to find cross-surface coupling clusters
2. Checked all 4 enforcement layers from the accretion gravity investigation against coupling scenarios
3. Verified the daemon config problem (12 files for 1 boolean) as ground truth

**Commands run:**
- Git co-change mining script: analyzed pairwise file co-occurrences across 1,212 non-metadata commits
- Cross-surface commit filter: identified 76 commits touching 3+ architectural layers
- Concept cluster extraction: grouped files by path-based keywords

## What I Observed

### Finding 1: Current hotspot system is size-only, completely blind to coupling

The existing `orch hotspot` has 3 detection types:
- `fix-density`: files with many fix commits
- `investigation-cluster`: topics with many investigations
- `bloat-size`: files exceeding line count threshold

None of these detect coupling. The daemon config cluster (25 files, 3 layers) has zero individual files flagged as bloated — every file is under 800 lines. The 4-layer accretion enforcement design from the 2026-02-14 investigation inherits this blindness: spawn gates, completion gates, coaching plugin, and CLAUDE.md boundaries ALL use file size as the signal. Coupling is orthogonal.

### Finding 2: Cross-surface commits are rare (6%) but contain all agent-hostile coupling

Only 76 of 1,212 commits touch 3+ architectural layers. But these 76 commits perfectly identify the 4 concept clusters that cause agent problems: daemon (24), verification (23), spawn (22), agent-status (22). Filtering to cross-surface commits eliminates healthy coupling (test+source pairs) automatically.

### Finding 3: Coupling score validates against known spiral

The daemon config cluster scores 180 on the proposed formula (3 layers × 25 files × 2.4 avg frequency). The tmux cluster scores ~13 (1 layer × 3 files × 4.3 avg frequency). This correctly ranks daemon config as CRITICAL and tmux as noise — matching real-world spiral evidence (daemon config caused 526K token spiral, tmux never caused issues).

### Finding 4: Events lack token tracking for spiral correlation

events.jsonl has 203 abandoned agents with `duration_seconds` and `workspace` but no `token_count`. The 526K spiral is known from the investigation narrative, not from instrumented data. Duration is available only for abandoned agents, not completed ones.

## Model Impact

**Extends** the completion-verification model:

1. **New gap identified:** The accretion enforcement architecture (4 layers from 2026-02-14) addresses file-level bloat but not cross-surface coupling. These are orthogonal failure modes. A file can be small (healthy by accretion metrics) yet part of a high-coupling cluster (hostile to agents). The 4-layer enforcement needs a 5th dimension: coupling detection.

2. **Existing invariant confirmed:** "Post-facto gates waste agent work" (Finding 2 from accretion investigation) applies equally to coupling. An agent that discovers a 12-file touch surface mid-session has already wasted tokens. Prevention (coupling warnings at spawn time) is the only effective intervention.

3. **New invariant proposed:** "Cross-surface coupling is the primary blind spot in accretion enforcement." Current gates catch files growing large (accretion) but not concepts growing wide (coupling). Both cause agent spirals through different mechanisms: accretion via complexity, coupling via discovery cost.

4. **Enhancement needed:** Add token_count to events.jsonl for future spiral correlation. Without instrumented token tracking, we can only detect coupling (static) not spiral risk (dynamic).
