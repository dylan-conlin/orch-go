# Probe: AE-02 Gate Passability Validation

**Claim:** AE-02 — "Gates must be passable by the gated party — knowledge-producing skills need exemptions"
**Model:** architectural-enforcement
**Date:** 2026-03-19
**Verdict:** CONFIRMED — claim holds; passability invariant is intact and properly scoped

---

## Falsification Condition

> Knowledge-producing agents can complete their work without reading files they would be gated from

## Method

Examined three independent evidence sources:

1. **Code analysis:** Current gate implementations and governance hook logic
2. **Workspace artifacts:** Recent architect/investigation SYNTHESIS files for governance-file read patterns
3. **Event data:** 226 architect/investigation completions (Feb-Mar 2026) for systemic patterns

## Findings

### Finding 1: Governance hook scoping is correct — blocks Write, permits Read

The governance file protection hook (`~/.orch/hooks/gate-governance-file-protection.py:92`) explicitly checks:

```python
if tool_name not in ("Edit", "Write"):
    sys.exit(0)
```

Read operations pass unconditionally. This means knowledge-producing agents can read governance-protected files (`pkg/spawn/gates/*`, `pkg/verify/accretion.go`, etc.) to analyze them without being blocked. The passability invariant holds at the governance hook layer.

### Finding 2: Hotspot gate exemptions are implemented and tested

`pkg/spawn/gates/hotspot.go:28-31` defines blocking skills as only `feature-impl` and `systematic-debugging`. The test at `hotspot_test.go:132` explicitly validates that `architect`, `investigation`, `capture-knowledge`, and `codebase-audit` are exempt:

```go
exemptSkills := []string{"architect", "investigation", "capture-knowledge", "codebase-audit"}
```

The gate is advisory-only (never blocks — `hotspot.go:39` comment: "Advisory only — emits warnings and events but never blocks"), so even blocking skills are technically unblocked. But the exemption distinction still matters for the advisory signal: exempt skills see "Strategic/read-only skill: hotspot advisory noted" instead of the extraction cascade warning.

### Finding 3: Investigation og-inv-design-analysis-hard-19mar-2503 directly confirms READ dependency

The displacement investigation (completed 2026-03-19) documented that it needed to read:
- `~/.orch/hooks/gate-governance-file-protection.py` lines 105-118 (denial message analysis)
- All 4 spawn gate implementations in `pkg/spawn/gates/` (triage, hotspot, agreements, question)
- `pkg/orch/governance.go` lines 45-74 (governance preflight check)

These are all governance-protected paths. The investigation completed successfully because governance hooks block Edit/Write but not Read. If Read were also blocked, this investigation could not have produced its findings about displacement patterns.

### Finding 4: AE-09 (displacement) reinforces AE-02 rather than weakening it

AE-09 added a new invariant: "enforcement without guidance creates displacement." This does NOT change the passability requirement — it extends it. The concern is that deny hooks block Write without saying where to write instead. This is orthogonal to Read access. Knowledge-producing agents need Read to analyze; implementation agents need Write guidance. AE-02 (Read passability) and AE-09 (Write guidance) are complementary, not conflicting.

### Finding 5: Completion gates have implicit passability via skill exemption

The architectural choices verification gate (`pkg/verify/architectural_choices.go:17-23`) exempts investigation and probe skills entirely — they return a passing result without checking. Test evidence gate (`pkg/verify/test_evidence.go:435-451`) exempts markdown-only changes and outside-project changes. Since knowledge-producing agents primarily produce `.md` files, they naturally pass completion gates without needing exemption by skill name.

## Assessment

**The falsification condition is NOT met.** Knowledge-producing agents demonstrably cannot complete their work without reading gated files. The og-inv-design-analysis-hard-19mar-2503 investigation is direct proof: it analyzed governance hooks, spawn gates, and governance preflight code — all governance-protected paths — and would have failed without Read access.

The passability design is two-layered:
1. **Spawn gates:** Exempt knowledge-producing skills from hotspot blocking
2. **Governance hooks:** Block Edit/Write only, permit Read universally

This means passability is achieved through **operation-type scoping** (Read vs Write) rather than **skill-type exemption** at the governance hook level. The hotspot gate uses skill exemption; the governance hook uses operation scoping. Both achieve passability but through different mechanisms.

## Staleness Note

The claim's last_validated date of 2026-02-14 predates:
- The advisory-only gate conversion (gates no longer block at all, making AE-02 easier to satisfy)
- AE-09 displacement work (which extends but doesn't contradict AE-02)
- 226 architect/investigation completions since Feb 2026

The claim remains valid but should note that gates are now advisory, reducing the practical impact of the passability invariant (nothing is actually blocked, so passability is trivially satisfied for spawn gates).
