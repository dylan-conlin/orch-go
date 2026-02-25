# Probe: Hotspot Bloat Scanner Build Output Exclusions

**Status:** Complete
**Model:** Completion Verification Architecture
**Date:** 2026-02-25
**Issue:** orch-go-1229

## Question

Does the bloat scanner in `analyzeBloatFiles` correctly exclude build output directories, or does it produce false positives that feed into the spawn gate and block legitimate work?

**Model claim being tested:** The accretion enforcement system (bloat detection → spawn gate) should only flag actual source code, not build output, tool files, or vendored dependencies.

## What I Tested

1. **Read `analyzeBloatFiles`** (hotspot.go:500) — the `filepath.Walk` directory skip logic
2. **Read `shouldCountFileWithExclusions`** (hotspot.go:280) — the path-based filter used by both bloat and fix-density analysis
3. **Ran `orch hotspot --json`** before fix to observe false positives
4. **Created test filesystem** with large files in 10 build output directories + 1 source directory
5. **Ran full test suite** (51 hotspot tests) after fix

## What I Observed

### Before Fix
- `analyzeBloatFiles` only skipped 3 directories: `.git`, `node_modules`, `vendor`
- `shouldCountFileWithExclusions` only checked `vendor/` prefix and `/generated/` substring
- `.opencode/plugin/coaching.ts` (1570 lines) appeared as CRITICAL bloat hotspot
- `.svelte-kit/output/server/chunks/index4.js` and similar build output appeared as hotspots
- These false positives propagated to `RunHotspotCheckForSpawn` and blocked spawns

### After Fix
- `skipBloatDirs` map: 13 directories (`.git`, `node_modules`, `vendor`, `.svelte-kit`, `dist`, `build`, `__pycache__`, `.next`, `.nuxt`, `.output`, `.opencode`, `.orch`, `.beads`)
- `buildOutputPrefixes` slice: 12 path prefixes (same dirs + `public/assets/`)
- Zero false positives from build output in `orch hotspot --json` output
- All 51 existing + new tests pass

### Hotspot Types Affected
| Type | Affected? | Why |
|------|-----------|-----|
| bloat-size | YES (primary bug) | Uses `filepath.Walk` on actual filesystem |
| fix-density | YES (secondary) | Uses `shouldCountFile` on git log paths |
| investigation-cluster | No | Only scans `.kb/investigations/` |
| coupling-cluster | No | Uses git log (build dirs are gitignored) |

## Model Impact

**Extends** the completion-verification model's understanding of accretion enforcement:

- **New invariant:** Bloat detection must exclude build output directories at two layers: filesystem walk (directory skip) AND path filtering (prefix check). The Walk skip is for efficiency; the prefix check is defense-in-depth for git-log-based paths.
- **Self-referential failure mode:** When accretion gates produce false positives on non-source files, they can block the very spawns needed to fix them. This is a class of bug worth monitoring for in any gate system.
- **Coverage gap:** The `--exclude` CLI flag only supports suffix-based globs (`*.json`), not directory prefixes. Directory-based exclusions must be maintained in code.
