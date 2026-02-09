# Probe: Are completion gates catching defects or generating bypass noise after targeted skips?

**Model:** `.kb/models/completion-verification.md`
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Since targeted `--skip-{gate}` bypasses were introduced (2026-01-14), what is the real usage mix of `--force`, targeted skips, and clean completions, and does that behavior indicate useful gate enforcement or noisy gates being bypassed?

---

## What I Tested

**Command/Code:**

```bash
python3 - <<'PY'
# Parsed ~/.orch/events.jsonl since 2026-01-14 for:
# - agent.completed (forced vs non-forced)
# - verification.bypassed (gate + reason)
# - spawn.triage_bypassed vs daemon.spawn
# Correlated bypass -> completion by beads_id within 15 minutes.
PY

sqlite3 ".beads/beads.db" "SELECT COUNT(*) ... FROM comments WHERE text LIKE '%Phase: Complete%' ..."
git log --since="2026-01-14" --pretty=format:"%s"
```

**Environment:**

- Repo: `orch-go`
- Time window: `2026-01-14T00:00:00Z` to now
- Sources: `~/.orch/events.jsonl`, `.beads/beads.db` comments/events, `git log`

---

## What I Observed

**Output:**

```text
instrumented_total 671
forced 112 pct 16.7
targeted_skip 107 pct 15.9
clean 452 pct 67.4
missing_forced_field 296

pre_2026-01-14: forced 931 / 1278 instrumented = 72.8%
post_2026-01-14: forced 112 / 671 instrumented = 16.7%

verification_failed_events 239
resolution_after_failure:
  resolved_with_targeted_skip 92
  resolved_clean 28
  resolved_with_force 8
  no_completion_seen 79

verification.bypassed by gate:
  test_evidence 184
  synthesis 174
  phase_complete 47

top bypass reason:
  "docs-only change, no tests needed" = 322 / 488 bypass events

spawn.triage_bypassed 1463
daemon.spawn 600
ratio 2.44
weekly ratio trend: 3.63 -> 3.67 -> 2.45 -> 1.71 -> 1.06

beads comments local range: 2026-02-05..2026-02-09 (1167 comments)
phase-complete comments: 299
phase-complete comments mentioning --force: 2
phase-complete comments mentioning --skip-*: 5

stability.jsonl bypass keyword hits: 0
```

**Key observations:**

- `--force` dropped sharply after targeted bypass rollout (72.8% -> 16.7% on instrumented completions), so blanket bypass behavior is materially reduced.
- Targeted bypass usage is substantial and concentrated in `test_evidence` and `synthesis`; the dominant reason (`docs-only change, no tests needed`) indicates recurring false-positive friction for doc-only or non-code work.
- Among verification failures with a later completion, targeted skip resolution (`92`) far exceeds force resolution (`8`) and clean resolution (`28`), suggesting gates are often bypassed selectively rather than fixed.
- Manual spawns (`spawn.triage_bypassed`) still exceed daemon spawns overall (2.44x) but trend strongly downward week-over-week, approaching parity recently.
- `~/.orch/stability.jsonl` contains no bypass telemetry; completion bypass observability lives in `events.jsonl` and partial beads comments.

---

## Model Impact

**Verdict:** extends — targeted bypasses reduced blanket `--force`, but gate noise remains concentrated in specific gates.

**Details:**
This confirms the model's claim that targeted bypasses replaced blunt forcing in practice, and updates the prior force-rate mental model: current post-rollout force usage is ~16.7% (instrumented), not ~55%+. However, bypass reasons and failure-resolution patterns show meaningful noise in `test_evidence` and `synthesis` gates for docs/non-code paths, so completion verification still catches issues but also generates bypass-prone friction that should be tuned gate-by-gate.

**Confidence:** High — based on direct event-log computation, gate-level reason distributions, and cross-checks against beads comments and git history metadata.
