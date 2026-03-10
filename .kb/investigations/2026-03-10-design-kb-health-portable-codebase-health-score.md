# Design: Portable Codebase Health Score for kb-cli

**Date:** 2026-03-10
**Status:** Complete
**Beads:** orch-go-lyh3i
**Skill:** feature-impl (design phase)

---

## Problem

The health score currently lives in orch-go (`pkg/health/health.go` + `cmd/orch/doctor_health.go`). External users of knowledge-physics cannot measure their own codebase health because:

1. `orch health` depends on `bd` (beads) for issue metrics
2. `orch health` depends on `orch hotspot` for fix-density analysis
3. Gate coverage checks orch-specific gates (spawn, completion, pre-commit hooks)
4. Snapshot storage assumes `~/.orch/health-snapshots.jsonl`

For knowledge-physics publication, external users need `kb health` — a zero-dependency command that works in any git repo.

---

## Prior Work

| Source | Finding |
|--------|---------|
| `pkg/health/health.go` | 5-dimension score: gate coverage, accretion control, fix:feat balance, hotspot control, bloat % |
| `.kb/plans/2026-03-10-harness-health-improvement.md` | Phase 0 constraint: calibration must work for external users |
| `.kb/models/knowledge-physics/model.md` | "Harness engineering is substrate governance" — health metrics are substrate-independent |
| knowledge-physics probes | kb-cli public release audit identified 7 blocking changes; health is one integration gap |

---

## Design: `kb health`

### Principle: Subset, Not Fork

`kb health` provides a **strict subset** of `orch health`. The dimensions that require orch infrastructure (gate coverage, beads issue tracking) are excluded. The dimensions that are language-agnostic and git-only are kept.

```
orch health (5 dimensions, orch-specific)
├── Gate coverage          ← EXCLUDED (orch-specific: spawn gates, completion gates)
├── Accretion control      ← INCLUDED (file line counts)
├── Fix:feat balance       ← INCLUDED (git log parsing)
├── Hotspot control        ← INCLUDED (fix-density from git log + file size)
└── Bloat percentage       ← INCLUDED (bloated/total ratio)

kb health (4 dimensions, portable)
├── Accretion control      ← file line counts, configurable threshold
├── Fix:feat balance       ← git log conventional commits
├── Hotspot control        ← fix-density + bloat hotspots
└── Bloat percentage       ← bloated/total source file ratio
```

### Score Formula

4 dimensions, each 0-25 points (total 0-100):

```
accretion  = 25 * max(0, 1 - bloatedFiles / accretionThreshold)
fixFeat    = 25 * max(0, 1 - fixFeatRatio / 3.0)
hotspot    = 25 * max(0, 1 - hotspotCount / hotspotThreshold)
bloatPct   = 25 * max(0, 1 - bloatedFiles / totalSourceFiles)

score = accretion + fixFeat + hotspot + bloatPct
```

**Threshold scaling** (same as orch, proven in orch-go-shmvp):
- `accretionThreshold = max(20, 10% of source files)`
- `hotspotThreshold = max(15, 5% of source files)`

### Grade Scale

Same as orch: A (90+), B (80-89), C (65-79), D (50-64), F (<50).

### Per-Project Configuration

Config file: `.kb/health.yaml` (optional, sane defaults without it)

```yaml
# .kb/health.yaml — project-specific health score configuration
thresholds:
  source_bloat: 800       # Lines before a source file is "bloated"
  test_bloat: 2000        # Lines before a test file is "bloated"
  fix_feat_window: 28     # Days of git history for fix:feat ratio
  fix_density: 5          # Fix commits before file is a hotspot

exclude_dirs:             # Additional dirs to skip (beyond defaults)
  - "generated"
  - "migrations"

exclude_patterns:         # Glob patterns to exclude
  - "*.pb.go"
  - "*.gen.ts"

# Language detection is automatic (by file extension).
# Override if needed:
# languages:
#   - go
#   - typescript
```

**Default skip dirs** (hardcoded, matching orch's `skipBloatDirs`):
`.git`, `node_modules`, `vendor`, `.svelte-kit`, `dist`, `build`, `__pycache__`, `.next`, `.nuxt`, `.output`

**Source file detection** (by extension, language-agnostic):
`.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.rs`, `.java`, `.kt`, `.swift`, `.rb`, `.c`, `.cpp`, `.h`, `.hpp`, `.cs`, `.svelte`, `.vue`

**Test file detection** (by naming convention):
`*_test.go`, `*.test.ts`, `*.test.js`, `*.spec.ts`, `*.spec.js`, `test_*.py`, `*_test.py`, `*_test.rs`

### Data Collection (No External Dependencies)

Every metric is collected using only `os` and `os/exec("git", ...)`:

| Metric | Collection Method | External Dep |
|--------|------------------|-------------|
| Bloated files | `filepath.Walk` + line counting | None |
| Total source files | `filepath.Walk` | None |
| Fix commits (28d) | `git log --since="28 days ago" --pretty=format:%s` | `git` |
| Feat commits (28d) | Same git log, different regex | `git` |
| Fix-density hotspots | `git log --since="28 days ago" --pretty=format:%s --name-only` + group by file | `git` |
| Bloat hotspots | Files > threshold (from bloated files scan) | None |

### CLI Interface

```bash
# Basic usage — run from any git repo with .kb/ directory
kb health

# Machine-readable output
kb health --json

# Specify project directory (default: cwd)
kb health --project-dir /path/to/repo

# Override thresholds inline (one-off, doesn't modify config)
kb health --source-bloat 1000 --test-bloat 3000
```

**Output format (human):**
```
Codebase Health Score
=====================
Score: 72/100 (C)

  Dimension           Points   Detail
  ──────────────────  ──────   ──────
  Accretion control   18.2/25  14 bloated / 30 threshold
  Fix:feat balance    22.5/25  0.3 ratio (28d: 8 fix, 27 feat)
  Hotspot control     15.8/25  19 hotspots / 45 threshold
  Bloat percentage    15.5/25  14 bloated / 312 source files (4.5%)
```

**Output format (JSON):**
```json
{
  "score": 72.0,
  "grade": "C",
  "dimensions": {
    "accretion": {"points": 18.2, "max": 25, "bloated": 14, "threshold": 30},
    "fix_feat": {"points": 22.5, "max": 25, "ratio": 0.3, "fixes": 8, "feats": 27},
    "hotspot": {"points": 15.8, "max": 25, "count": 19, "threshold": 45},
    "bloat_pct": {"points": 15.5, "max": 25, "bloated": 14, "total": 312, "pct": 4.5}
  },
  "config": {
    "source_bloat": 800,
    "test_bloat": 2000,
    "fix_feat_window": 28,
    "scaled_accretion_threshold": 30,
    "scaled_hotspot_threshold": 15
  }
}
```

### Package Structure in kb-cli

```
internal/health/
├── health.go       # ScoreConfig, ComputeScore(), ScoreResult types
├── collect.go      # CollectMetrics(projectDir, config) — file walking, git log
├── config.go       # LoadConfig(kbDir) — reads .kb/health.yaml
└── health_test.go  # Unit tests (all pure computation, no git needed for score tests)

cmd/kb/
└── health.go       # kb health command definition
```

**Key type:**
```go
type ScoreConfig struct {
    SourceBloat     int      // Default 800
    TestBloat       int      // Default 2000
    FixFeatWindow   int      // Default 28 (days)
    FixDensity      int      // Default 5
    ExcludeDirs     []string // Additional dirs to skip
    ExcludePatterns []string // Glob patterns to exclude
}

type Metrics struct {
    BloatedFiles     int
    TotalSourceFiles int
    FixCommits       int
    FeatCommits      int
    HotspotCount     int
}

type ScoreResult struct {
    Score      float64
    Grade      string
    Dimensions map[string]DimensionResult
    Metrics    Metrics
    Config     ScoreConfig
}
```

### Relationship to `orch health`

`orch health` should eventually delegate to `kb health` for the 4 portable dimensions, then add its own 5th dimension (gate coverage). This avoids code duplication.

**Migration path:**
1. Implement `kb health` in kb-cli with the 4-dimension formula
2. `orch health` continues using its current 5-dimension formula
3. Future: `orch health` imports `kb-cli/internal/health` or shells out to `kb health --json` and adds gate coverage on top

This is NOT a breaking change — orch's score will differ from kb's score because orch includes gate coverage. Both are valid for their context.

### Calibration Constraint for Harness Plan

The harness-health-improvement plan (Phase 1: Score Calibration) should ensure that:

1. **Test file threshold separation** works in `kb health` (it does — `TestBloat` config field)
2. **Codebase-scaled thresholds** work in `kb health` (they do — same `max(floor, pct*total)` formula)
3. **Score changes are meaningful** — same exit criteria: ±5 points when 3 source files extracted

The calibration work in orch-go-shmvp (already completed) established the formula; `kb health` adopts it directly.

### What `kb health` Intentionally Excludes

| Excluded | Why |
|----------|-----|
| Gate coverage | orch-specific (pre-commit, spawn, completion gates) |
| Issue metrics (open, blocked, stale, orphaned) | Requires beads (`bd`) |
| Trend tracking / snapshot history | v1 is stateless; add in v2 if demanded |
| Investigation hotspots | Requires `.kb/investigations/` analysis (could add later) |
| Alerts | Requires trend data |

Trend tracking is deferred because external users running `kb health` once don't benefit from longitudinal data. When/if they adopt it as a recurring metric, v2 adds `--track` with local JSONL storage in `.kb/health-snapshots.jsonl`.

---

## Implementation Notes

### Test Strategy

1. **Unit tests for score computation** — pure math, no filesystem
2. **Unit tests for config loading** — temp dir with `.kb/health.yaml`
3. **Integration tests for metric collection** — temp git repo with known files and commits
4. **No mocking of git** — use real `git init` + `git commit` in temp dirs

### Edge Cases

- **No git repo**: Error with "not a git repository"
- **No `.kb/` dir**: Still works — `.kb/health.yaml` is optional
- **No conventional commits**: fix=0, feat=0, ratio=0.0 → full 25 points (no signal, not penalized)
- **Empty repo**: score=100 (nothing to penalize)
- **Monorepo**: `--project-dir` scopes to subdirectory

---

## Decision Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| 4 vs 5 dimensions | 4 (drop gate coverage) | Gates are orch-specific |
| Points per dimension | 25 each | Simpler than 20-each with a missing dimension |
| Config location | `.kb/health.yaml` | Co-located with knowledge base, optional |
| Trend tracking | Deferred to v2 | First user is one-shot; trends need repeat usage |
| Package location | `internal/health/` in kb-cli | Internal to kb binary, not a shared library yet |
| Threshold defaults | Same as orch (800/2000/28d/5 fixes) | Proven calibration from orch-go-shmvp |
