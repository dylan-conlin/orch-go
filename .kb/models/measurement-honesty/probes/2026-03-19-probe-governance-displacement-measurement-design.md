# Probe: Governance Displacement Measurement Design — Deny Counts Without Displacement Tracking Create False Confidence

**Model:** measurement-honesty
**Date:** 2026-03-19
**Status:** Complete

---

## Question

The measurement-honesty model's invariant #2 says "Absent negative signal ≠ positive signal." The governance hook infrastructure has zero observability (F1 from hook audit, 2026-03-12). Even if we add deny counting, a deny count without displacement tracking is an incomplete measurement that creates false confidence ("governance works!") while hiding the architectural cost. What measurement approaches are honest about what they actually measure?

Specific model claims under test:
- **Invariant #1:** "A metric that cannot go red provides no information." — Does a deny-count-only metric structurally prevent negative signal?
- **Invariant #2:** "Absent negative signal ≠ positive signal." — Zero displacement tracking makes governance look perfect.
- **Invariant #6:** "Delete before fixing." — Should we avoid building displacement measurement if we can't make it honest?
- **The False Confidence Problem (§Core Mechanism):** "The displacement effect: when a dashboard shows [metrics], the operator treats the system as measured."

---

## What I Tested

### 1. Current hook event observability

```bash
# Check for ANY hook-related event types in events.jsonl
grep -c "hook" ~/.orch/events.jsonl
# Result: matches exist but ALL are in spawn.gate_decision or agent completion reasons — zero hook.deny events

# Count governance-specific spawn gate decisions
grep "spawn.gate_decision.*governance" ~/.orch/events.jsonl | python3 -c "import sys, json; lines=[json.loads(l) for l in sys.stdin]; print(f'Total: {len(lines)}')"
# Result: Total: 13

# Check for hook.deny, hook.allow, hook.invocation — any hook lifecycle event
grep -cE '"type":"hook\.' ~/.orch/events.jsonl
# Result: 0
```

### 2. Governance hook deny output format

```bash
# Read the hook's deny path
tail -30 ~/.orch/hooks/gate-governance-file-protection.py
```

The hook outputs `{"hookSpecificOutput": {"permissionDecision": "deny", "permissionDecisionReason": "GOVERNANCE FILE PROTECTION: Workers cannot modify governance infrastructure..."}}` to stdout and exits. It writes **nothing** to events.jsonl, no log file, no checkpoint. The only artifact of a deny is the text shown to the agent inside Claude Code's conversation.

### 3. Can the hook emit events?

```bash
# Check if orch emit supports arbitrary event types
grep -A5 'emitCmd.*Run' cmd/orch/emit_cmd.go
```

`orch emit <event-type>` exists and supports `--beads-id`, `--reason`, `--data`. A hook could shell out to `orch emit hook.deny --data '{"hook":"governance","file":"pkg/spawn/gates/foo.go"}'` — but this adds ~200ms latency per deny (Go binary startup). The hook currently targets <50ms latency.

Alternative: The Python hook could append directly to `~/.orch/events.jsonl` — it's just JSONL append. The Go logger uses `os.O_APPEND` which is atomic for small writes on POSIX. Python's `open(path, 'a')` with a single `write()` call is equally safe.

### 4. Existing completion-time infrastructure for file auditing

```bash
# Read the accretion delta collector
grep -n "collectAccretion\|parseNumstat" cmd/orch/complete_postlifecycle.go
```

`collectAccretionDelta()` already runs at completion time, reads the agent manifest for git baseline, runs `git diff --numstat baseline..HEAD`, and emits `accretion.delta` with per-file line counts. This is the natural instrumentation point for a displacement check — we already know every file the agent touched.

### 5. Known displacement example

```bash
# The scs-sp-8dm case: agent blocked from pkg/spawn/gates/, wrote to pkg/orch/spawn_preflight.go
grep -n "concurrency" pkg/orch/spawn_preflight.go
```

Confirmed: `pkg/orch/spawn_preflight.go` lines 30-41 contain a concurrency check (`ConcurrencyCheck` type, `activeCount >= maxAgents` gate). The comment says "Concurrency gate reinstated for manual spawns (scs-sp-8dm: manual + daemon spawns in same window exceeded capacity)." This is displaced gate logic that architecturally belongs in `pkg/spawn/gates/` but was placed in `pkg/orch/` because the governance hook blocked writing to gates/.

### 6. The "63 hook denials" claim

```bash
# Search for where the 63 denials number comes from
grep "63\|hook.*denial" ~/.orch/events.jsonl | head -3
```

The number "63 hook denials" from the architectural-displacement thread has **no events.jsonl evidence**. Zero `hook.deny` events exist. The number may come from manual transcript review or estimation. This is itself evidence of F1 (zero observability) — we don't even know how the denial count was produced.

### 7. Governance warning vs runtime deny gap

The `spawn.gate_decision` with `gate_name:"governance"` fires at spawn time based on text-matching the task description against protected path strings. This is a **warning** — "the task mentions governance files." It is NOT evidence that the runtime hook actually denied an Edit/Write. An agent could receive the spawn-time warning but never attempt to write to a protected file. The spawn-time warning and the runtime deny are different events measuring different things.

---

## What I Observed

### Finding 1: Complete absence of hook-level events

There are zero event types in `pkg/events/logger.go` for hook invocations. The 38 defined event types cover spawns, completions, verifications, services, explorations, and commands — but nothing for hooks. The hooks exist entirely outside the event system.

### Finding 2: Two separate observability gaps

| Gap | What's Missing | Consequence |
|-----|---------------|-------------|
| **Deny counting** | No `hook.deny` event. Hook writes nothing. | Cannot answer "how often does governance block agents?" |
| **Displacement tracking** | No post-deny behavior correlation. | Cannot answer "when governance blocks, does the agent put code in the wrong place?" |

These are independent gaps. Closing gap 1 (deny counting) does NOT close gap 2 (displacement). A system with perfect deny counting but no displacement tracking would produce the exact false confidence the task describes: "governance blocked 63 times, therefore governance is working" — when in reality those 63 denials may have displaced code into wrong packages 63 times.

### Finding 3: `orch emit` path is viable but has latency cost

The `orch emit` command takes ~200ms (Go binary startup). For a hook that targets <50ms latency, this is 4x overhead. Direct Python append to events.jsonl is ~0ms additional (file already opened for JSON parse). The tradeoff: emit via `orch emit` (standardized, schema-validated, slower) vs direct JSONL append (fast, fragile to schema drift).

### Finding 4: Completion-time displacement detection is architecturally natural

The accretion delta collector (`complete_postlifecycle.go:468-541`) already:
1. Reads agent manifest (git baseline, spawn time)
2. Gets full file diff (all files touched by agent)
3. Parses per-file line additions/removals
4. Emits `accretion.delta` event

Adding displacement detection here means: for each file in the agent's diff, check if it's in a "sibling" package of a governance-protected path AND the spawn event had a governance warning. This is ~10 lines of logic added to an existing pipeline step.

### Finding 5: Displacement is not binary — four outcomes after deny

When a governance hook denies an agent, four things can happen:

| Outcome | Description | Measurable? |
|---------|-------------|------------|
| **Compliance** | Agent reports via `bd comments add` and moves on | Yes — look for CONSTRAINT/DISCOVERED patterns in beads comments |
| **Displacement** | Agent puts the code in a wrong-but-similar location | Partially — heuristic matching against governance-adjacent packages |
| **Abandonment** | Agent gives up on that aspect of the task | Partially — absence of commits touching the relevant subsystem |
| **Invisible workaround** | Agent finds a completely different approach | No — cannot distinguish from legitimate alternative approach |

Any measurement that claims to fully capture "what happened after deny" would need to account for all four outcomes. Measuring only displacement would miss compliance and abandonment, creating a different kind of false confidence.

---

## Measurement Design Proposal

### Phase 1: Get the Denominator (~10 lines Python, ~0 lines Go)

**What:** Add direct JSONL append to `gate-governance-file-protection.py` when it denies.

**Event schema:**
```json
{
  "type": "hook.deny",
  "timestamp": 1773961173,
  "data": {
    "hook_name": "governance-file-protection",
    "file_path": "pkg/spawn/gates/concurrency.go",
    "matched_pattern": "pkg/spawn/gates/",
    "tool_name": "Edit",
    "context": "worker",
    "beads_id": "orch-go-abc123",
    "session_id": "session-xyz"
  }
}
```

**Implementation:** ~10 lines appended to the hook's deny branch. Read `BEADS_ID` and `SESSION_ID` from env vars (already set by spawn).

**What it measures:** "How often does the governance hook deny agent writes, to which files?"
**What it does NOT measure:** What happened after the deny.
**False confidence risk:** MEDIUM if used alone — deny count alone implies governance is working. Must be paired with displacement signal.
**What would make it go red:** N > 0 (it always can). Monotonically increasing count is the expected behavior.

### Phase 2: Completion-Time Displacement Flag (~50 lines Go)

**What:** At `orch complete` time, check if the agent's commits touch governance-adjacent packages when the spawn had a governance warning.

**Event schema:**
```json
{
  "type": "governance.displacement_candidate",
  "timestamp": 1773961173,
  "data": {
    "beads_id": "orch-go-abc123",
    "governance_warning_paths": ["pkg/spawn/gates/", "pkg/verify/"],
    "displaced_files": [
      {"path": "pkg/orch/spawn_preflight.go", "added_lines": 35, "sibling_of": "pkg/spawn/gates/"}
    ],
    "confidence": "heuristic"
  }
}
```

**Heuristic:** Map each governance-protected path to its "sibling" packages:
- `pkg/spawn/gates/` → sibling is `pkg/orch/`, `pkg/spawn/` (non-gates)
- `pkg/verify/` → sibling is `pkg/orch/`, `cmd/orch/`
- `cmd/orch/*_lint_test.go` → sibling is `cmd/orch/` (non-test files)
- `.orch/hooks/` → sibling is `pkg/hook/`

If agent was warned about governance paths at spawn AND commits touch a sibling package with >10 added lines, emit candidate event.

**What it measures:** "Agent was warned about governance paths AND wrote substantial code to adjacent packages."
**What it does NOT measure:** Whether the code was actually displaced governance logic vs. legitimate work in that package.
**False confidence risk:** MEDIUM — false positives when agent legitimately modifies `pkg/orch/` for non-governance reasons.
**What would make it go red:** Any governance-warned agent committing to sibling packages.
**Precision:** UNKNOWN. Must measure before trusting. Expected high FP rate given that most agents modify `pkg/orch/` for legitimate reasons.
**Requires:** Manual review of each candidate to classify as real-displacement vs false-positive.

### Phase 3: (ONLY if Phase 1+2 show volume) Deny Checkpoint Correlation

**Trigger:** Only build if Phase 1 shows >5 denials/week AND Phase 2 shows >50% real-displacement rate in manual review.

**What:** Hook writes a deny checkpoint file (`~/.orch/deny-checkpoints/<beads-id>-<timestamp>.json`). Completion pipeline reads checkpoints and correlates with post-deny commit timestamps to identify which specific commits followed a deny.

**Why defer:** The expected deny volume may not justify the implementation complexity. If governance denials happen <3 times per week, manual transcript review is more honest than automated detection with unknown precision.

### What NOT to Build

**Agent Self-Report:** The current deny message already tells agents to report via `bd comments add`. The scs-sp-8dm example shows agents sometimes DO self-report. But we cannot know the reporting rate (no denominator), so self-report data is anecdotal — useful for case studies, unreliable for counting. Adding more explicit self-report instructions ("DISPLACEMENT:") to the deny message is low-cost but provides no precision guarantee.

**Automated Displacement Classification:** A system that claims "this code IS displaced governance logic" (not just "candidate") would require semantic analysis of the code. Pattern matching (function names like `Check*`, `Validate*`, `Gate*`) would have unknown and likely poor precision. Per model invariant #4, precision must be measured before the metric is operational.

### Measurement Honesty Self-Assessment

| Metric | What Would Go Red? | False Confidence Risk | Epistemic Type |
|--------|--------------------|-----------------------|----------------|
| `hook.deny` count | N > 0 (always can) | MEDIUM alone, LOW when paired with displacement | Honest — counts what it counts |
| `governance.displacement_candidate` | Any governance-warned agent touching siblings | MEDIUM (FP from legitimate code) | Noisy signal — requires precision calibration |
| "Displacement rate" (candidates / denials) | Ratio > 0 | LOW — honestly a ratio | Honest-but-misnamed if labeled "displacement" instead of "candidate rate" |
| "Governance is working" (zero displacement) | **Nothing can** (absent signal) | **HIGH — this is the false confidence to avoid** | False confidence — must NOT be displayed |

The last row is the key insight: a dashboard that shows "0 displacement events" after adding Phase 1 + Phase 2 would be FALSE CONFIDENCE unless we can prove the detection has high recall. We cannot prove high recall because invisible workarounds (outcome 4) are structurally undetectable.

**The honest framing:** "We detected N displacement candidates out of M denials. Actual displacement is at least N but could be higher — our detector has unknown recall for invisible workarounds."

---

## Model Impact

- [x] **Confirms** invariant #2 ("Absent negative signal ≠ positive signal"): Governance hooks have zero negative-signal channels. The system cannot distinguish "hooks are preventing displacement" from "we can't see displacement." Zero displacement events = dead channel, not proof of effectiveness. This is structurally identical to the rework-rate example in the model (0 reworks across 817 completions).

- [x] **Confirms** invariant #4 ("Precision must be measured before the metric is operational"): The proposed displacement candidate detector (Phase 2) would have unknown precision. Deploying it without calibration would create a noisy signal that drifts toward false confidence per failure mode #2 (noisy signals become false confidence through familiarity).

- [x] **Extends** model with: **Two-gap independence** — for governance hooks, the observability problem has two independent gaps (deny counting + displacement tracking) where closing one does NOT close the other. A deny counter alone creates a new instance of false confidence: "we blocked N times" implies governance works, but the architectural cost (displacement) remains invisible. The model's taxonomy assumes a single measurement either works or doesn't. In this case, the "measurement" of governance effectiveness requires TWO measurements with different epistemic properties, and the combination is only honest if both are present.

- [x] **Extends** model with: **Structural undetectability** — some displacement outcomes (invisible workarounds) are structurally undetectable regardless of instrumentation quality. This means any displacement metric has an inherent recall ceiling < 100%. The honest response is: label the metric as a floor estimate ("at least N displacements detected"), never a ceiling estimate ("only N displacements occurred"). This pattern may apply to other measurements where the target concept has a component that cannot be observed.

- [x] **Extends** model with: **Latency-honesty tradeoff in hook instrumentation** — adding event emission to hooks creates a tension between measurement honesty (we should measure) and hook performance (measurement adds latency). Direct JSONL append (~0ms additional) trades schema coupling for speed. `orch emit` (~200ms) trades speed for schema validation. This tradeoff doesn't appear in the current model, which assumes measurement can be added without cost.

---

## Notes

### Implementation Priority

Phase 1 (hook deny logging) should be implemented first — it has near-zero cost and closes the most fundamental gap. Phase 2 (displacement candidates) should be built and calibrated before being displayed to operators. Phase 3 should be deferred pending volume data from Phase 1.

### The Volume Question

With only 13 governance warnings at spawn time in the events log, the actual runtime deny volume may be very low. If so, manual transcript review of each denied session is more honest than automated detection with unknown precision. The correct first step is: get the denominator (Phase 1), then decide whether automated detection is worth the precision risk.

### Architectural Recommendation

This investigation identifies a hotspot area (`pkg/events/logger.go`). Adding new event types and displacement detection logic would affect governance-protected files (`pkg/spawn/gates/` adjacency checking) and hotspot files (`pkg/events/`). Per CLAUDE.md routing rules, implementation should go through `architect` before `feature-impl`.

### CONSTRAINT: Cannot verify hook denials empirically

I cannot trigger an actual governance hook deny in this investigation because the hook checks `CLAUDE_CONTEXT=worker` and I am running as a worker — but triggering a deny would mean attempting to edit a governance-protected file, which would block my edit. The hook's deny behavior was verified by the hook infrastructure audit (2026-03-12) via synthetic input. The measurement design in this probe is based on reading the hook code and event infrastructure, not on observing a live deny event.
