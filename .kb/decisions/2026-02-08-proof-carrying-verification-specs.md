# Decision: Proof-Carrying Verification Specs for Agent Work

**Date:** 2026-02-08
**Status:** Accepted
**Context:** Verification evidence currently depends on prose/heuristics, which creates a human bottleneck for batch completion and overnight verification.
**Resolves:** orch-go-21492

## Decision

Adopt **proof-carrying verification specs** as a first-class completion artifact. Each relevant worker session must emit an executable verification contract that allows orchestrator tools to run deterministic checks without reconstructing intent from SYNTHESIS prose.

This is introduced as a **complement** to existing completion gates first, then tightened by rollout policy.

---

## 1) Artifact Placement (Decision 1)

Use **both** workspace and beads, with clear ownership:

1. **Canonical artifact (operational):**
   - `VERIFICATION_SPEC.yaml` in workspace root.
   - Used by `orch complete` and `orch verify --batch` as the executable source of truth.

2. **Persistent mirror (audit):**
   - Completion posts a compact beads comment containing:
     - spec version
     - method counts by tier
     - normalized command list hash (sha256)
     - pass/fail summary and runtime
   - This preserves proof lineage even if workspace is archived/pruned.

Rationale: workspace is best for execution fidelity; beads is best for durable audit/search.

---

## 2) Minimal Schema (Decision 2)

Adopt this minimal v1 schema:

```yaml
version: 1
scope:
  beads_id: orch-go-21492
  workspace: og-arch-design-proof-carrying-08feb-66d3
  skill: architect
verification:
  - id: verify-cli-health
    method: cli_smoke # cli_smoke | integration | browser | manual | static
    tier: light       # light | full | orchestrator
    command: "orch health --json"
    cwd: "."
    timeout_seconds: 30
    expect:
      exit_code: 0
      stdout_contains:
        - "daemon_status"
  - id: verify-ui-dashboard
    method: browser
    tier: full
    command: "glass assert --url http://localhost:4000 --contains 'Operational'"
    timeout_seconds: 45
    expect:
      exit_code: 0
  - id: human-signoff
    method: manual
    tier: full
    manual_steps:
      - "Open dashboard in browser"
      - "Confirm agent cards update in <2s"
    expect:
      human_approval_required: true
```

Rules:
- `method`, `tier`, and either `command` or `manual_steps` are required.
- `expect.exit_code` defaults to `0` when omitted for command methods.
- `manual` steps are non-executable and require explicit completion annotation.
- `static` is for non-runtime checks (lint/schema/format/assertions).

---

## 3) Emission + Exit Integration (Decision 3)

Use a **hybrid integration**:

1. **Worker-base / spawn protocol update (authoring path):**
   - Add required instruction: agent must author/update `VERIFICATION_SPEC.yaml` before `Phase: Complete`.
   - Spawn template pre-populates a skeleton based on skill + tier.

2. **Completion hook enforcement (execution path):**
   - `orch complete` parses and validates spec.
   - Runs executable entries for applicable tier.
   - Writes beads digest mirror.
   - Fails verification when required entries fail.

3. **SYNTHESIS extension (narrative path):**
   - Add a short `Verification Contract` section in SYNTHESIS linking spec file and key outcomes.
   - SYNTHESIS remains human-readable context; spec remains machine-executable contract.

Rationale: worker-base alone cannot guarantee compliance; completion hook alone creates poor authoring UX; SYNTHESIS alone is not executable.

---

## Relationship to Existing Gates

Proof-carrying specs **complement** existing Phase/Evidence/Approval gates in rollout phase.

Migration target:
- Phase stays authoritative for lifecycle signaling.
- Evidence/visual/test gates move from regex heuristics toward contract-backed checks when spec coverage is present.
- Existing gates remain fallback for legacy/missing specs during migration.

---

## Batch Composition (`orch verify --batch`)

`orch verify --batch` should:
1. Discover candidate completed workspaces/issues.
2. Load each `VERIFICATION_SPEC.yaml`.
3. Execute executable entries (`cli_smoke`, `integration`, `browser`, `static`) in isolated workers.
4. Mark `manual` entries as pending unless human approval token is present.
5. Emit per-item closure result and aggregate pass-rate report.

Output should include deterministic replay metadata:
- spec hash
- commands run
- expectations checked
- failed step IDs

---

## Missing Spec Policy

Use staged rollout:

1. **Phase A (grace, advisory):** Missing spec logs warning, completion still allowed.
2. **Phase B (selective blocking):** Missing spec blocks for implementation skills (`feature-impl`, `systematic-debugging`, `reliability-testing`) on full tier.
3. **Phase C (default required):** Spec required for all non-light tracked spawns unless explicitly skipped with targeted bypass reason.

This prevents immediate friction spikes while converging to proof-carrying as default behavior.

---

## Consequences

### Positive
- Reduces manual mental reconstruction during verification.
- Enables reliable overnight batch checks from executable contracts.
- Creates durable audit link between what was claimed and what was run.

### Trade-offs
- Authoring overhead for agents (mitigated by prefilled templates).
- Schema drift risk (mitigated by versioned parser + strict validation).
- Manual method entries still require human bottleneck for subjective checks.

### Non-goals
- Replacing all human approval for UI work.
- Eliminating SYNTHESIS artifacts.
- Inferring verification from prose alone.

---

## Implementation Plan

1. Add schema parser/validator (`pkg/verify/proofspec.go`).
2. Add spawn-time skeleton generation by skill/tier.
3. Add completion-time execution and digest emission.
4. Add batch runner integration (`orch verify --batch`).
5. Roll out missing-spec policy by phases A -> B -> C.

---

## References

- `.kb/models/completion-verification.md`
- `.kb/guides/completion-gates.md`
- `cmd/orch/complete_gates.go`
- `pkg/verify/test_evidence.go`
- `pkg/verify/visual.go`
- `pkg/spawn/context_template.go`
