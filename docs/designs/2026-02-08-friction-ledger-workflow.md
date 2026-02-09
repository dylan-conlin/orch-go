# Friction Ledger Workflow

**Date:** 2026-02-08  
**Status:** Implemented  
**Owner:** orch-go-21409

## Problem

Recurring orchestration failures are currently scattered across beads comments, ad-hoc notes, and investigation files. We can detect symptoms late (after multiple retries), but we do not have a single lightweight workflow that captures each friction event in real time with enough structure to synthesize patterns.

This causes three operational gaps:

1. **Missed repetition signal**: same failure mode appears across multiple issues before becoming obvious.
2. **Weak traceability**: incidents are not consistently linked to evidence artifacts.
3. **Slow synthesis prep**: preparing synthesis requires reconstructing event history manually.

## Goals

Define and implement a lightweight protocol that captures each friction event with four required fields:

- **Symptom** (what failed)
- **Impact** (why it matters)
- **Evidence path** (where proof lives)
- **Linked issue** (where follow-up is tracked)

And make repeated failures visible via built-in summary output.

## Protocol

### Capture (real-time)

When friction is observed, record it immediately:

```bash
orch friction log \
  --symptom "Duplicate spawn race" \
  --impact "Consumed one extra daemon slot" \
  --evidence .kb/investigations/2026-02-08-dup-spawn.md \
  --issue orch-go-21409
```

What this does:

1. Validates the linked issue exists.
2. Validates the evidence path exists.
3. Appends a JSONL entry to `.orch/friction-ledger.jsonl`.
4. Emits `friction.logged` event to `~/.orch/events.jsonl`.
5. Adds a beads comment to the linked issue for immediate visibility.

### Review (active sessions)

List recent incidents:

```bash
orch friction list
orch friction list --issue orch-go-21409
```

### Synthesize (pattern detection)

Surface repeated symptoms:

```bash
orch friction summary
```

Summary groups incidents by normalized symptom and reports count, latest impact, latest evidence, and linked issues.

## Data Model

Stored per entry in `.orch/friction-ledger.jsonl`:

```json
{
  "id": "fr-20260208-193015.123456789",
  "timestamp": "2026-02-08T19:30:15.123456789Z",
  "symptom": "Duplicate spawn race",
  "impact": "Consumed one extra daemon slot",
  "evidence_path": ".kb/investigations/2026-02-08-dup-spawn.md",
  "linked_issue": "orch-go-21409"
}
```

## Why This Is Lightweight

- Single command to capture an event.
- Four required fields only.
- Uses existing artifacts (beads + investigations) as evidence targets.
- JSONL append model keeps storage simple and grep-friendly.
- No new database or migration required.

## Validation

Implementation is covered by tests in:

- `pkg/friction/ledger_test.go`
- `cmd/orch/friction_cmd_test.go`

And manual smoke commands:

```bash
go test ./pkg/friction
go test ./cmd/orch -run 'Friction|NormalizeEvidencePath|FrictionLedgerPath'
```
