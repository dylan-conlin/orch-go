# Synthesis: Automatic Account Distribution for Claude CLI Spawns

**Beads:** orch-go-1214
**Skill:** architect
**Date:** 2026-02-24

## Summary

Designed automatic account distribution for Claude CLI spawns across two Max accounts (work 20x, personal 5x) with tier-aware capacity routing.

## Key Design Decisions

1. **Schema:** Add `tier`, `role`, `config_dir` fields to accounts.yaml (static config, not API-derived)
2. **Algorithm:** Work-first with personal spillover — primary absorbs bulk, spillover activates when primary is rate-limited
3. **Architecture:** Account added to ResolvedSpawnSettings via `resolveAccount()` with full provenance tracking (same pattern as model/backend/tier resolution)
4. **Injection:** `CLAUDE_CONFIG_DIR` env var per-process via `BuildClaudeLaunchCommand` — proven mechanism from `claude-personal` alias
5. **Caching:** 5-minute TTL capacity cache to avoid token rotation on every spawn
6. **Guidance:** Extend `orch account list` with `[RECOMMENDED]` indicator

## Architecture Impact

- ~265 lines across 7 files + tests
- Follows existing spawn resolution precedence pattern (CLI > config > heuristic > default)
- Two independent auth mechanisms confirmed: OpenCode OAuth (global) vs CLAUDE_CONFIG_DIR (per-process)
- Existing `ShouldAutoSwitch` is OpenCode-only; this design covers Claude backend (now the default)

## Artifacts Produced

- **Investigation:** `.kb/investigations/2026-02-24-design-automatic-account-distribution-claude-cli.md`
- **Probe:** `.kb/models/orchestration-cost-economics/probes/2026-02-24-probe-automatic-account-distribution-design.md`

## Recommended Implementation

Phased approach:
- **Phase 1:** Schema + CLI flag (`--account`) + resolveAccount with CLI source + BuildClaudeLaunchCommand configDir param
- **Phase 2:** CapacityCache + heuristic routing (work-first, personal-spillover)
- **Phase 3:** `orch account list` recommendation + spawn output logging

## Discovered Work

- Implementation of Phase 1 (schema + CLI flag + env var injection)
- Implementation of Phase 2 (capacity cache + heuristic routing)
- Implementation of Phase 3 (account list recommendation + logging)
