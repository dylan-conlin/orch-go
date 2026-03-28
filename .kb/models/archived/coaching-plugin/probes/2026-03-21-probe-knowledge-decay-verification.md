# Probe: Knowledge Decay Verification — Coaching Plugin

**Date:** 2026-03-21
**Trigger:** 30-day decay check (last probe: 2026-02-20)
**Method:** Code verification against model claims

---

## Claims Verified

### 1. Plugin file size: "1831 lines as of Jan 18"
**Status: STALE — significant refactor occurred**

The plugin was extracted from a single 1831-line `coaching.ts` into 7 files totaling 1545 lines:
- `coaching.ts` (544 lines) — main plugin, hook registration
- `coaching-types.ts` — shared types, constants, utilities
- `coaching-classification.ts` — bash command classification, spawn detection
- `coaching-investigation.ts` — investigation recommendation loading, contradiction detection
- `coaching-injection.ts` — coaching message injection via noReply
- `coaching-detection.ts` — file line count, code file detection, premise skipping
- `coaching-worker-health.ts` — worker health metric tracking

**Impact:** Model's "Primary Evidence" section lists `plugins/coaching.ts` at 1831 lines. This is materially wrong.

### 2. Detection patterns (5 orchestrator)
**Status: PARTIALLY STALE**

| Pattern | Model Claims | Code Reality |
|---------|-------------|--------------|
| `frame_collapse` | Present | Confirmed (coaching.ts:384) |
| `circular_pattern` | Present | Confirmed (coaching.ts:499) |
| `behavioral_variation` | Present | Confirmed (coaching.ts:460) |
| `spawn_without_context` | Present | NOT FOUND in any plugin file |
| `completion_backlog` | Written by Go backend | Confirmed in serve_coaching.go:220-221 |

`spawn_without_context` appears to have been removed. Only 3 orchestrator patterns remain in the plugin code (+ `completion_backlog` in Go backend = 4 total, not 5).

### 3. Worker health metrics (5 patterns)
**Status: CONFIRMED**

All 5 worker health metrics confirmed in `coaching-worker-health.ts`:
- `tool_failure_rate` (line 97)
- `context_usage` (line 118)
- `time_in_phase` (line 142)
- `commit_gap` (line 165)
- `accretion_warning` (coaching.ts:325)

### 4. Removed patterns (action_ratio, analysis_paralysis)
**Status: CONFIRMED**

Zero matches for `action_ratio` or `analysis_paralysis` in any plugin file. Removal is complete.

### 5. Worker detection via session.metadata.role
**Status: CONFIRMED**

Chain of trust intact:
1. `pkg/opencode/client.go:528` — sets `x-opencode-env-ORCH_WORKER` header
2. OpenCode fork `server/routes/session.ts:245-246` — reads header, sets metadata
3. `plugins/coaching.ts:133-141` — `detectWorkerSession()` checks `session?.metadata?.role`
4. One-way caching (only cache true) confirmed in `coaching.ts:135-141`

### 6. Three injection mechanisms
**Status: CONFIRMED**

1. `config.instructions` — used in `orchestrator-session.ts` (backup shows `config.instructions.push(skillPath)`)
2. `client.session.prompt({ noReply: true })` — confirmed in `coaching-injection.ts:108`, `coaching-worker-health.ts:65`
3. `session.created` event hook — confirmed in `coaching.ts:157`

### 7. Metrics persistence path (~/.orch/coaching-metrics.jsonl)
**Status: CONFIRMED (path correct, but file not present on disk)**

`coaching-types.ts:16` defines `METRICS_PATH = join(homedir(), ".orch", "coaching-metrics.jsonl")`. The file doesn't currently exist at `~/.orch/`, which suggests the plugin hasn't been actively running recently (or the `~/.orch/` directory doesn't exist). This is consistent with the coaching system being present but possibly not active in the current development workflow.

### 8. Dashboard visualization
**Status: CONFIRMED**

- `serve_coaching.go` exposes `/api/coaching` endpoint (line 250)
- `web/src/lib/stores/coaching.ts` polls `http://localhost:3348/api/coaching` every 30s
- serve_coaching.go line count: 268 (model says 321 — minor drift)

### 9. JSONL dual writers
**Status: CONFIRMED**

- Plugin writes via `writeMetric()` (coaching-types.ts)
- Go backend writes `completion_backlog` metrics (serve_coaching.go:220-221)

### 10. Observation coupled to intervention (Failure Mode 3)
**Status: STILL ACCURATE**

No evidence of the "architectural fix" (separate daemon) being implemented. Injection is still triggered from within the tool.execute.after hook path.

---

## Summary

**Overall verdict: Model is mostly current but has two material inaccuracies.**

| Finding | Severity | Action Needed |
|---------|----------|---------------|
| Plugin refactored from 1 file (1831 lines) to 7 files (1545 lines) | Medium | Update Primary Evidence section |
| `spawn_without_context` pattern removed | Medium | Update Detection Patterns table (4 orch patterns, not 5) |
| `serve_coaching.go` is 268 lines, not 321 | Low | Update line count reference |
| `~/.orch/coaching-metrics.jsonl` not present on disk | Info | May indicate plugin not actively running |
| All other claims | — | Confirmed accurate |

**Recommended model update:** Medium — update file structure, correct pattern count, update line counts.
