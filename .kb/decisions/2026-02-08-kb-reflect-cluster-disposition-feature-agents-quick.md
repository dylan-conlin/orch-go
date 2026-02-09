# Decision: kb reflect Cluster Disposition for feature/agents/quick

**Date:** 2026-02-08
**Status:** Accepted
**Authority:** Knowledge hygiene and synthesis process

---

## Context

`kb reflect --type synthesis` surfaced three top clusters:

- `feature` (5 investigations)
- `agents` (4 investigations)
- `quick` (4 investigations)

These clusters were generated from lexical overlap in investigation titles. They are useful as a *signal* but not as a final synthesis boundary. Before disposition, claims were checked against current primary evidence:

- `pkg/spawn/config.go` still sets `"feature-impl": TierLight`
- `cmd/orch/spawn_validation.go` now fails closed on decision-check errors
- `cmd/orch/status_statedb.go` still enumerates all `workers-*` tmux sessions/windows in fallback discovery
- `cmd/orch/serve.go` still defines `DefaultServePort = 3348`

---

## Decision

### 1) Agents cluster is already consolidated by canonical decision

The four `agents` investigations are superseded by:

- `.kb/decisions/2026-02-07-agent-completion-lifecycle-separation.md`

No new agent-lifecycle decision is needed from this reflect pass.

### 2) Feature cluster must be split by lineage, not treated as one topic

The `feature` cluster mixes three distinct threads:

1. **Light-tier SYNTHESIS behavior** (`feature-impl` default light tier)
2. **skillc load-bearing implementation work**
3. **Decision-gate test/fail-open investigation history**

Disposition:

- Keep these as separate lineages tied to their canonical decisions.
- Mark redundant investigations with `Superseded-By` links.
- Do not create a single "feature" model; it is a lexical umbrella, not a coherent mechanism.

### 3) Quick cluster is demoted from investigation-level to probe/quick-level for fact checks

The `quick` cluster contains mostly single-claim checks (fact lookup or one-off validation).

Disposition:

- Keep `.kb/investigations/archived/2026-01-21-inv-audit-kb-quick-entries-stale.md` as the canonical hygiene audit.
- Treat one-off fact checks as **probes** (when model-scoped) or `kb quick` entries (when tactical), not full investigations.
- Mark redundant quick-test investigations as superseded.

### 4) Closure rule for redundant investigations

When an investigation is redundant after consolidation:

1. Add/update `Superseded-By` with canonical decision path.
2. Ensure D.E.K.N. `Next` no longer implies unresolved action for that file.
3. Keep file in archived location for provenance.

---

## Consequences

**Positive:**

- Reflect clusters now produce concrete dispositions instead of repeated investigation churn.
- Canonical decision lineage is clearer for future `kb context` lookups.
- Quick fact checks move to cheaper artifact types (probe/quick) with less overhead.

**Negative:**

- Some historical investigation recommendations remain unimplemented and must be tracked via decisions/issues, not the archived file itself.
- Cluster names in reflect output may still look broad (lexical), requiring orchestrator triage.

---

## Synthesized From

- `.kb/investigations/archived/2025-12-24-inv-feature-impl-agents-completing-without.md`
- `.kb/investigations/archived/2025-12-26-inv-feature-impl-agents-not-producing.md`
- `.kb/investigations/archived/2026-01-14-inv-feature-register-friction-guidance-links.md`
- `.kb/investigations/archived/2026-01-14-inv-feature-skillc-warns-load-bearing.md`
- `.kb/investigations/archived/2026-01-28-inv-implement-test-feature.md`
- `.kb/investigations/archived/2025-12-22-inv-40-agents-showing-as-active.md`
- `.kb/investigations/archived/2026-01-03-inv-agents-going-idle-without-phase.md`
- `.kb/investigations/archived/2026-01-08-inv-25-28-agents-not-completing.md`
- `.kb/investigations/archived/2026-02-04-inv-agents-own-declaration-via-bd.md`
- `.kb/investigations/archived/2026-01-10-inv-quick-test-default-port-orch.md`
- `.kb/investigations/archived/2026-01-19-inv-quick-test-read-claude-md.md`
- `.kb/investigations/archived/2026-01-21-inv-audit-kb-quick-entries-stale.md`
- `.kb/investigations/archived/2026-01-27-inv-quick-test-verify-coaching-plugin.md`

---

## Related Decisions

- `.kb/decisions/2026-02-07-agent-completion-lifecycle-separation.md`
- `.kb/decisions/2026-01-08-load-bearing-guidance-data-model.md`
- `.kb/decisions/2026-01-28-decision-gate.md`
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md`
