# Architect Investigation: First-Class Episodic Memory Pipeline

**Date:** 2026-02-09
**Status:** Complete
**Type:** architect
**Beads:** orch-go-7l6l7

**Question:** How should orch implement a first-class action-memory artifact class that captures execution episodes (action + outcome + evidence), validates memories before reuse, and applies clear retention policy across spawn, command, verification, and completion boundaries?

**Delta:** orch already captures useful execution signals (`events.jsonl`, `ACTIVITY.json`, gate-skip TTL), but there is no unified, validated episodic memory object that can be safely reused in future prompts.

---

## Summary (D.E.K.N.)

**Delta:** Introduce a new `action-memory` artifact class with explicit schema, provenance, confidence, and expiry. Build a lifecycle pipeline that extracts at spawn/command/verification/completion, validates before prompt reuse, and retains entries by tier (short, medium, long).

**Evidence:** Current code writes disjoint event streams at each lifecycle boundary (`cmd/orch/spawn_cmd.go`, `cmd/orch/send_cmd.go`, `cmd/orch/complete_gates.go`, `cmd/orch/complete_pipeline.go`), plus archival activity export (`cmd/orch/complete_export.go`, `pkg/activity/export.go`) and TTL precedent in gate skip memory (`pkg/verify/gate_skip_memory.go`).

**Knowledge:** Orch's gap is not missing signals; it is missing canonicalization and trust gating. The architecture should treat episodic memory as a derived artifact from trusted events, not raw model text, to resist prompt injection and stale context.

**Next:** Implement in three phases: (1) schema + extractor, (2) validation gate + injection integration, (3) retention + promotion workflow into durable KB artifacts.

---

## Findings

### Finding 1: Lifecycle signals already exist, but are fragmented

**Evidence:**

- Spawn emits `session.spawned` from every backend path (`cmd/orch/spawn_cmd.go`).
- Command interactions emit command-specific events such as `session.send` (`cmd/orch/send_cmd.go`).
- Verification emits structured failures and bypasses (`cmd/orch/complete_gates.go`, `cmd/orch/complete_verify.go`).
- Completion emits `agent.completed` and exports `ACTIVITY.json` before archival (`cmd/orch/complete_pipeline.go`, `cmd/orch/complete_export.go`).

**Significance:** The data needed for episodic memory exists today, but each channel is consumed independently. This causes weak recall and inconsistent reuse.

---

### Finding 2: Activity export already contains tool-level evidence pointers

**Evidence:** `pkg/activity/export.go` captures message parts including tool invocations, input, output, timestamps, and session IDs, and `cmd/orch/serve_agents_activity.go` can replay this from archived workspaces.

**Significance:** `ACTIVITY.json` is an ideal evidence substrate for episodic memories because pointers can reference concrete tool calls (`messageID`, `part.id`, timestamp), not paraphrased narrative.

---

### Finding 3: Orch already has TTL and auto-prune precedent

**Evidence:** `pkg/verify/gate_skip_memory.go` persists gate skip decisions with explicit expiry (`GateSkipDuration = 2h`) and prune-on-read behavior.

**Significance:** Retention policy for episodic memory can reuse this pattern: explicit expiry field, lazy prune on read, deterministic expiration semantics.

---

### Finding 4: Current context injection has no episodic reuse gate

**Evidence:** Spawn context generation (`pkg/spawn/context.go`) injects KB context and task metadata but has no retrieval layer for validated action episodes.

**Significance:** Without a validation-before-reuse gate, any future episodic injection risks replaying stale or tainted memories directly into prompts.

---

## Recommendation

### 1. Add a first-class `action-memory` artifact class

Define a canonical episodic object that is derived from trusted lifecycle events.

```json
{
  "id": "am_01J...",
  "project": "orch-go",
  "workspace": "og-feat-...",
  "session_id": "ses_xxx",
  "beads_id": "orch-go-7l6l7",
  "boundary": "verification",
  "action": {
    "type": "gate_check",
    "name": "phase_complete",
    "input": "beads_id=orch-go-7l6l7"
  },
  "outcome": {
    "status": "success",
    "summary": "Phase: Complete detected from beads comments"
  },
  "evidence": {
    "kind": "events_jsonl",
    "pointer": "~/.orch/events.jsonl#offset=123456",
    "timestamp": 1765632031,
    "hash": "sha256:..."
  },
  "confidence": 0.92,
  "expires_at": "2026-02-10T18:00:00Z",
  "created_at": "2026-02-09T18:00:00Z",
  "validated_at": null,
  "validation_state": "pending"
}
```

Required fields from task:

- `action`
- `outcome`
- `evidence pointer`
- `confidence`
- `expiry`

Additional required system fields: `boundary`, `project/workspace/session`, `validation_state`, `hash`.

---

### 2. Extraction hooks by lifecycle boundary

Use deterministic extraction points where data is already authoritative:

1. **Spawn boundary**
   - Hook after spawn event write (`session.spawned`) in `cmd/orch/spawn_cmd.go`.
   - Emit memory: "agent spawned with skill/model/workspace".

2. **Command boundary**
   - Hook command-specific emitters (`session.send`, `phase` updates) and add a root-level command wrapper in `cmd/orch/main.go` (`PersistentPreRunE`/`PersistentPostRunE`) for uniform command start/finish episodes.
   - Emit memory: "command attempted", "command succeeded/failed", duration, target.

3. **Verification boundary**
   - Hook verification results in `cmd/orch/complete_gates.go` and gate skips in `cmd/orch/complete_verify.go`.
   - Emit memory: gate outcomes, bypass decisions, transient retries.

4. **Completion boundary**
   - Hook after activity export and before/after `agent.completed` in `cmd/orch/complete_export.go` and `cmd/orch/complete_pipeline.go`.
   - Emit memory: completion reason, evidence availability, cleanup outcomes.

---

### 3. Validation-before-reuse gate (prompt-injection resistant)

Before injecting any episodic memory into spawn context:

1. **Expiry check**: `now < expires_at`.
2. **Provenance allowlist**: source must be trusted (`events.jsonl`, `ACTIVITY.json`, state DB projection) not freeform assistant prose.
3. **Evidence resolution**: pointer must resolve and hash-check.
4. **Scope match**: project + workspace/session compatibility (avoid cross-project leakage unless explicitly requested).
5. **Freshness recheck**: if memory references mutable state (issue status, gate state), re-read canonical source.
6. **Sanitization**: strip prompt-like directives and markdown/code injection payloads from summary text.
7. **Confidence threshold**: reject low-confidence entries (default < 0.7) from automatic injection; keep for UI inspection only.

Output of gate:

- `accepted` -> eligible for prompt injection
- `degraded` -> shown as advisory (not injected)
- `rejected` -> quarantined with reason

---

### 4. Retention policy (short vs long-lived)

Use three retention tiers:

1. **Short-lived (operational episodes)**
   - TTL: 24h
   - Examples: command retries, transient verification failures, liveness prompts.
   - Purpose: immediate continuity between adjacent sessions.

2. **Medium-lived (work-cycle episodes)**
   - TTL: 14d
   - Examples: successful verification patterns, recurring bypass rationale, completion outcomes.
   - Purpose: issue-level and sprint-level reuse.

3. **Long-lived (promoted episodes)**
   - TTL: 90d (or no TTL if promoted)
   - Examples: repeated high-confidence episodes that explain system behavior classes.
   - Promotion path: summarize into `.kb/decisions`/`.kb/models` when they become declarative knowledge.

Pruning behavior:

- prune on read + periodic background prune
- keep tombstone metadata (`id`, `expired_at`, reason) for observability windows

---

## Proposed System Design

### Components

1. `pkg/episodic/schema.go`
   - artifact struct definitions + validation helpers.

2. `pkg/episodic/extractor.go`
   - converts lifecycle events and activity parts into normalized action-memory entries.

3. `pkg/episodic/store.go`
   - append/read/prune APIs (start with JSONL, optional SQLite index later).

4. `pkg/episodic/validate.go`
   - validation-before-reuse gate implementation.

5. `pkg/spawn/context_episodic.go`
   - injects only validated entries into `SPAWN_CONTEXT.md`.

6. `cmd/orch/*` hook adapters
   - spawn/command/verification/completion callsites.

### Storage choice

Start with `~/.orch/action-memory.jsonl` as authoritative episodic ledger:

- aligns with existing `events.jsonl` patterns
- avoids violating `state.db` projection-cache contract
- keeps migration risk low

Optional phase-2 optimization: derived SQLite index for fast query only.

---

## Implementation Plan

### Phase A: Schema + extraction

- add artifact schema and unit tests
- wire spawn + verification + completion hooks
- write entries with evidence pointers only

### Phase B: Validation + injection

- add validator and quarantine reasons
- integrate validated retrieval into spawn context generation
- add CLI inspection command (`orch memory episodes --for <beads-id>`)

### Phase C: Retention + promotion

- TTL prune policy by episode class
- dashboard visibility for rejected/quarantined memories
- promotion helper to create decisions/models from recurring high-confidence episodes

---

## Acceptance Criteria

1. Every spawned worker receives a "recent validated episodes" section when relevant.
2. No memory is injected without resolvable evidence pointer and passing validation gate.
3. Expired entries are not injected and are pruned automatically.
4. Verification bypasses and failures generate queryable episodic entries with reasons.
5. Prompt-injection attempts embedded in activity text are rejected/quarantined.

---

## Evidence (Commands Run)

```bash
pwd
orch phase orch-go-7l6l7 Planning "Designing first-class episodic memory architecture for action outcomes"
orch phase orch-go-7l6l7 Implementing "Drafting architecture for first-class action-memory schema, hooks, validation, and retention"

go test ./pkg/activity -run TestDetectPhaseCompleteAttemptFromEvents
go test ./pkg/verify -run TestWriteAndReadGateSkipMemory
go test ./pkg/verify -run TestPhaseCompleteRecoveryFromActivity
```

Observed:

- activity detection test passed (`pkg/activity`)
- gate skip memory persistence + TTL behavior test passed (`pkg/verify`)
- Phase: Complete recovery from `ACTIVITY.json` test passed (`pkg/verify`)

---

## References

- `cmd/orch/spawn_cmd.go`
- `cmd/orch/spawn_pipeline.go`
- `cmd/orch/send_cmd.go`
- `cmd/orch/phase_cmd.go`
- `cmd/orch/complete_gates.go`
- `cmd/orch/complete_verify.go`
- `cmd/orch/complete_pipeline.go`
- `cmd/orch/complete_export.go`
- `pkg/activity/export.go`
- `pkg/events/logger.go`
- `pkg/verify/gate_skip_memory.go`
- `.kb/investigations/2026-02-08-inv-session-memory-landscape-2026-who.md`
