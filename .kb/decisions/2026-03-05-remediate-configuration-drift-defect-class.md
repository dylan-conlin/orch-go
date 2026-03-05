# Decision: Remediate Configuration-Drift Defect Class

**Date:** 2026-03-05
**Status:** Proposed
**Issue:** orch-go-98m6a
**Supersedes:** None (extends `.kb/models/drift-taxonomy/model.md`)

## Context

Over the past 30 days, 18+ investigations have traced to a single root cause: **code assumes configuration that doesn't exist yet**. These aren't random bugs — they share a structural pattern:

1. A new system is designed and partially implemented
2. The old system remains operational
3. Both claim authority over the same concern
4. The gap between design and deployment creates silent failures

This is **incomplete migration debt** — not configuration "drift" in the traditional sense. True drift is two copies diverging over time. What we have is incomplete deployment: new systems built in code but never fully wired, leaving dual authorities.

## The Inventory: 11 Incomplete Migrations

### Tier 1: Active Harm (causing bugs now)

| # | Migration | Old Authority | New Authority | Gap | Evidence |
|---|-----------|--------------|---------------|-----|----------|
| 1 | **Skill CLI references** | Removed commands (`orch frontier`, `orch reap`, `bd comment`) | New commands (`orch status`, `orch clean --orphans`, `bd comments add`) | 69 stale references across 28 skills | audit 2026-03-05 |
| 2 | **Daemon config surface** | Scattered `Config{}` construction (4 sites) | `pkg/daemonconfig/FromUserConfig()` | 15+ fields fall through to hardcoded defaults instead of `config.yaml` | `convert.go:30-48` |
| 3 | **Config symlink validation** | Manual checking | `check-config-symlinks.sh` | Script exists but NOT wired to SessionStart hook | `settings.json` missing entry |

### Tier 2: Silent Wrong (producing plausible but incorrect results)

| # | Migration | Old Authority | New Authority | Gap | Evidence |
|---|-----------|--------------|---------------|-----|----------|
| 4 | **Model staleness** | `.kb/models/*/model.md` (hand-authored) | Spawn-time staleness annotations | ~50% models stale; annotations surface staleness but don't prevent stale context from being served | probe 2026-02-20 |
| 5 | **Orchestrator skill content** | Models + guides (`.kb/models/`, `.kb/guides/`) | Orchestrator skill (compiled from models/guides) | 19 drift items between skill and its sources | audit 2026-01-15 |
| 6 | **Reference docs** | Pre-Go-rewrite CLI docs | Actual `orch --help` output | `reference/orch-commands.md` has 15+ broken entries | audit 2026-03-05 |

### Tier 3: Latent Risk (will cause harm when conditions change)

| # | Migration | Old Authority | New Authority | Gap | Evidence |
|---|-----------|--------------|---------------|-----|----------|
| 7 | **Plist generation** | Hand-edited `com.orch.daemon.plist` | `pkg/daemonconfig/plist.go` (code-generated) | No `orch config generate plist` command to regenerate from config | design 2026-01-08 |
| 8 | **Defect class gates** | Ad-hoc bug fixes | Named defect classes (0-7) | Classes named but no `orch doctor --defect-scan` or spawn-time checking | catalogue 2026-03-03 |
| 9 | **Scanner scope allowlists** | Open-ended scanning (walk all dirs) | `ScanScope` allowlist pattern | Pattern designed, only 1 scanner uses it | issue orch-go-jauu |
| 10 | **`beads.DefaultDir` elimination** | Global `DefaultDir` (cross-project bleed) | Explicit `projectDir` parameter | Designed in defect class work, not started | catalogue Class 4 |
| 11 | **Cross-project agreements** | Implicit interface contracts | `kb agreements` YAML contracts | 5 agreements for ~20+ interfaces | agreements system |

## Root Cause Analysis

The recurring pattern is:

```
Design investigation → Decision → Partial implementation → Ship partial → New work begins → Old partial forgotten
```

Why the partial is never completed:
1. **No tracking mechanism for "deployment complete"** — beads issues track the implementation, not the deployment/wiring
2. **New work obscures incomplete old work** — 20+ investigations/week means the incomplete migration scrolls off attention
3. **Partial implementations work well enough** — the daemon config compiles, the symlink script exists, the skill deploys. The gap is invisible.
4. **No structural gate preventing incomplete deployment** — nothing asks "is this migration actually finished?"

## Remediation Plan

### Phase 1: Complete Active-Harm Migrations (feature-impl, 2-3h)

**1a. Fix skill CLI references** (highest ROI — blocks every worker agent)
- Global `bd comment ` → `bd comments add ` across `skills/src/`
- Remove 5 dead command references (`orch frontier/reap/health/stability/friction`)
- Fix 7 wrong flag syntaxes
- Remove 4 non-existent skill references
- `skillc deploy` to push to `~/.claude/skills/`

**1b. Wire config symlink validation**
- Add `$HOME/.orch/hooks/check-config-symlinks.sh` to `settings.json` SessionStart hooks
- Verify via `cc personal` launch

**1c. Complete daemon config FromUserConfig**
- Add `config.yaml` backing for: `MaxSpawnsPerHour`, `SpawnDelay`, `CleanupEnabled`, `CleanupInterval`, `CleanupAgeDays`, `RecoveryEnabled`, `RecoveryInterval`, `RecoveryIdleThreshold`, `KnowledgeHealthEnabled`
- Update `pkg/userconfig/` with matching accessor methods

### Phase 2: Structural Prevention (architect + feature-impl)

**2a. "Migration complete" checklist gate**
- When closing a design investigation that recommends implementation, the close comment must include a `MIGRATION_STATUS` block:
  ```
  MIGRATION_STATUS:
    designed: [what was designed]
    implemented: [what code was written]
    deployed: [what config/hooks/skills were wired]
    remaining: [what's still incomplete, or "none"]
  ```
- If `remaining != "none"`, create a follow-up issue with `triage:ready` label

**2b. Stale reference detection in skillc**
- `skillc lint` command that checks all `bd`, `orch`, `kb` command references in skill sources against actual CLI help output
- Run as part of `skillc deploy` pipeline
- This is the structural gate for Tier 1 (skill content drift)

### Phase 3: Reduce Migration Surface (ongoing)

**3a. Eliminate reference docs as authority** — `reference/orch-commands.md` should be auto-generated from `orch --help`, not hand-maintained. One source of truth.

**3b. Orchestrator skill recompilation** — the skill should be compiled from models/guides with staleness checking, not manually authored then checked for drift.

## Decision

1. **Frame accepted:** Configuration drift in this system is primarily incomplete migration debt, not copy divergence
2. **Remediate Tier 1 first** — active harm, highest ROI, can be done in a single feature-impl session
3. **Add MIGRATION_STATUS gate** — prevents future incomplete migrations from being forgotten
4. **skillc lint for structural prevention** — catches the highest-volume drift class (skill content) at deploy time

## Trade-offs

- **MIGRATION_STATUS adds friction to closing investigations** — accepted because the cost of incomplete migration (18 investigations) far exceeds the cost of writing 4 lines
- **skillc lint requires CLI help parsing** — accepted because the alternative (manual checking) has a 50%+ miss rate
- **Not auto-fixing model staleness** — accepted per drift taxonomy constraint: models require synthesis, auto-fix creates review debt faster than it can be processed

## Implementation Routing

| Phase | Skill | Priority | Blocked By |
|-------|-------|----------|------------|
| 1a | feature-impl | P1 | nothing |
| 1b | feature-impl | P1 | nothing |
| 1c | feature-impl | P2 | nothing |
| 2a | worker-base skill update | P2 | 1a (needs updated skill template) |
| 2b | feature-impl | P2 | nothing |
| 3a | feature-impl | P3 | nothing |
| 3b | architect | P3 | nothing |
