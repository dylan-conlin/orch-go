# Friction Ledger Guide

**Purpose:** Capture orchestration friction incidents in real time with enough structure to detect repeats and synthesize reliably.

**Last verified:** 2026-02-08

---

## What To Capture

Every friction event must include four fields:

1. **Symptom** - what failed or resisted progress
2. **Impact** - why it matters (time, reliability, quality, velocity)
3. **Evidence path** - concrete artifact path (investigation, log, screenshot, repro note)
4. **Linked issue** - beads issue ID where follow-up work is tracked

---

## Command Workflow

### 1) Log incident immediately

```bash
orch friction log \
  --symptom "Duplicate spawn race" \
  --impact "Consumed extra daemon slot" \
  --evidence .kb/investigations/2026-02-08-dup-spawn.md \
  --issue orch-go-21409
```

Behavior:
- validates issue exists
- validates evidence path exists
- appends entry to `.orch/friction-ledger.jsonl`
- emits `friction.logged` event
- comments linked issue for visibility

### 2) Review recent events

```bash
orch friction list
orch friction list --issue orch-go-21409
```

### 3) Detect repeated patterns

```bash
orch friction summary
```

---

## Storage + Query

- **Ledger file:** `.orch/friction-ledger.jsonl` (project-local)
- **Event stream:** `~/.orch/events.jsonl` with event type `friction.logged`

Project-local ledger keeps evidence paths and issue references tied to the codebase where friction occurred.

---

## Operational Protocol

Use this as default cadence:

1. Log friction when observed (do not defer).
2. During triage/review, run `orch friction summary`.
3. If a symptom repeats (2+), create or update a synthesis/investigation artifact.
4. Link synthesis output back to affected issues.

---

## Example Entry Quality

Good:

- Symptom: `Spawn blocked by decision keyword false positive`
- Impact: `Daemon stalled overnight queue; 4 ready issues skipped`
- Evidence: `.kb/investigations/2026-02-08-inv-daemon-decision-block-false-positive.md`
- Linked issue: `orch-go-21011`

Bad:

- Symptom: `it broke`
- Impact: `annoying`
- Evidence: `none`
- Linked issue: `(missing)`
