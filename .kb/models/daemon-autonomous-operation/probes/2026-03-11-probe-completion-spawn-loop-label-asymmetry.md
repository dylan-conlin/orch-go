# Probe: Completion-Spawn Loop Label Asymmetry

**Date:** 2026-03-11
**Model:** daemon-autonomous-operation
**Trigger:** orlcp completed 3x with identical output, spawn loop re-queuing completed issues

## Question

Why did the daemon process the same Phase: Complete 3 times for orlcp, and why were completed issues re-entering the spawn queue?

## What I Tested

Traced the event timeline for orch-go-orlcp across events.jsonl and beads comments:
- 3 `daemon.complete` events with identical blog post completion reason
- Issue had both `triage:ready` AND `daemon:ready-review` labels simultaneously
- Spawn loop `NextIssueExcluding` checked `triage:ready` but never checked `daemon:ready-review`

## What I Observed

**Root cause: Asymmetric label awareness between spawn and completion loops.**

The completion loop filtered out `daemon:ready-review` issues (completion_processing.go:131-143), but the spawn loop (`NextIssueExcluding` in daemon.go) had no equivalent filter. This created a cycle:

1. Completion loop processes Phase: Complete → adds `daemon:ready-review`
2. Spawn loop sees `triage:ready` (still present) → tries to re-spawn
3. If spawn fails or agent completes immediately → issue returns to spawn queue
4. Completion loop sees Phase: Complete again → processes again

Additionally, when `daemon:ready-review` label failed to persist (between completions #1 at 20:34 and #2 at 00:24), there was no fallback mechanism to prevent reprocessing.

## Model Impact

**Confirmed:** The Feb 14 incident probe identified L6 (UpdateBeadsStatus fail-fast) as the primary dedup mechanism. This probe reveals a parallel gap: even with L6 working, the **completion loop** had no dedup equivalent. The spawn pipeline's 7-layer defense only prevents duplicate spawns, not duplicate completions.

**New failure mode documented:** Completion-spawn loop cycle — completed issues with stale `triage:ready` re-enter spawn queue through the spawn loop, which lacks daemon completion label awareness.

**Fix applied:** Three layers:
1. Spawn queue filters `daemon:ready-review` and `daemon:verification-failed`
2. Completion processing removes `triage:ready` after adding `daemon:ready-review`
3. In-memory `CompletionDedupTracker` prevents same Phase: Complete from being processed twice (defense-in-depth for label persistence failures)
