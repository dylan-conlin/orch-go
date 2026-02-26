# Decision: Models Track Architecture, Not Implementation

## Context

After closing the entire Verification System Overhaul epic (5 issues), we noticed `completion-verification.md` wasn't updated. Three orchestrator sessions shipped verification work without mentioning the model.

Question: Is model staleness a problem?

## Decision

**Models track architecture, not implementation.** Update models when concepts change, not when code changes.

### The Test

For each change, ask: "Would an agent reading the model tomorrow be misled about how the system works?"

- If **yes** → update the model (architectural change)
- If **no** → skip (implementation detail)

### Examples from Jan 14 Verification Work

| Change | Architectural? | Model Update? |
|--------|----------------|---------------|
| Zero spawn_time fallback | No - fallback within existing gate | No |
| Markdown-only exemption | No - new exemption within existing gate | No |
| `--skip-{gate}` flags | **Yes** - changes bypass model from all-or-nothing to targeted | **Yes** |
| Verification metrics | No - observability, not architecture | No |

## Alternatives Considered

1. **Update models on every code change** - Rejected: too much overhead, models become implementation docs
2. **Never update models** - Rejected: models become misleading over time
3. **Periodic model audits** - Considered: could complement but doesn't solve real-time awareness

## Consequences

- Orchestrators prompted to reflect on model impact at session end (orch-go-8hdpi)
- Models stay focused on concepts, not implementation details
- Some drift is acceptable if architecture unchanged

## Provenance

Meta-orchestrator session 2026-01-14, discussion with Dylan after reviewing that 3 orchestrator sessions shipped verification work without noting model impact.
