# Probe: daemon.go Extraction Completeness

**Date:** 2026-02-19
**Status:** Complete
**Model:** daemon-autonomous-operation
**Trigger:** orch-go-1120 — daemon.go still 1516 lines despite Jan 4 plan targeting 200-300

## Question

The daemon model claims extractions should reduce daemon.go to ~200-300 lines of orchestration. The Jan 4 investigation identified P0/P1/P2 extractions. Are those complete? If so, why is daemon.go still 1516 lines?

## What I Tested

### Test 1: P0/P1 extraction file existence
```bash
ls pkg/daemon/rate_limiter.go      # P0 - exists (Jan 4)
ls pkg/daemon/skill_inference.go   # P0 - exists (Feb 18)
ls pkg/daemon/issue_queue.go       # P0 - exists (Feb 12)
ls pkg/daemon/active_count.go      # P1 - exists (Feb 18)
ls pkg/daemon/issue_adapter.go     # P1 - exists (Feb 18)
ls pkg/daemon/completion_processing.go # P1 - exists (Feb 18)
```

### Test 2: daemon.go line count
```bash
wc -l pkg/daemon/daemon.go
# 1516 lines
```

### Test 3: daemon.go content analysis
Analyzed remaining content in daemon.go (1516 lines):
- Daemon struct + constructors: ~155 lines (72-225)
- NextIssue/NextIssueExcluding: ~120 lines (227-348)
- expandTriageReadyEpics: ~65 lines (350-414)
- Capacity convenience methods: ~95 lines (416-511)
- Preview + rejection checking: ~170 lines (513-689)
- Once/OnceExcluding spawn logic: ~310 lines (691-1004)
- OnceWithSlot spawn logic: ~220 lines (1006-1229)
- Periodic: reflection/cleanup/recovery: ~265 lines (1231-1494)
- Run loop: ~20 lines (1496-1516)

## What I Observed

1. **All P0 and P1 extractions from Jan 4 plan are complete.** Every file identified in the investigation exists and contains the expected code.

2. **daemon.go grew from 1363 to 1516 lines** despite extractions, because new features were added post-Jan 4:
   - Recovery system (ShouldRunRecovery, RunPeriodicRecovery, etc.)
   - Enhanced dedup (session dedup, content-aware dedup, fresh status check)
   - Epic expansion (expandTriageReadyEpics)
   - Extraction gate (hotspot-driven auto-extraction spawning)
   - Completion failure tracking

3. **The "accretion gravity" pattern** is visible: extracted code was replaced by new code at a faster rate than extraction occurred.

4. **Three new P0-equivalent extraction targets** exist:
   - `periodic.go` (~263 lines) - Self-contained periodic task runners
   - `preview.go` (~170 lines) - Preview/rejection/formatting
   - `capacity.go` (~95 lines) - Pool/rate-limit convenience methods
   Total: ~528 lines extractable → daemon.go from 1516 to ~988

5. **Remaining hard core** (~988 lines) is:
   - Daemon struct + constructors (~155 lines)
   - Issue selection logic (~185 lines)
   - Spawn orchestration (Once/OnceExcluding/OnceWithSlot) (~530 lines)
   - Run loop (~20 lines)
   The spawn methods contain massive duplication between OnceExcluding and OnceWithSlot.

## Model Impact

**Extends model claim:** "After extraction, daemon.go should become ~200-300 lines of pure orchestration"

The model's target of 200-300 lines is unreachable with extraction alone. Even after all extractable methods move out, ~988 lines remain. The core problem is:
- The Daemon struct has 35+ fields (many are test mock functions)
- OnceExcluding and OnceWithSlot are ~530 lines combined with ~60% code duplication
- Reaching 200-300 lines requires **refactoring** (dedup Once variants, extract mock functions to options pattern) not just **extraction** (moving methods to new files)

**Confirms model claim:** Extraction pattern works and is proven (all original P0/P1 files are functional).

**New invariant candidate:** daemon.go exhibits "accretion gravity" — new features accumulate in the god file faster than extraction removes them. Extraction alone is insufficient; architectural gates (like the >1500 line hotspot checker) are needed to prevent re-accumulation.

## Post-Extraction Results

Executed three new P0-equivalent extractions:
- `periodic.go` (245 lines) - reflection, cleanup, recovery periodic task management
- `preview.go` (211 lines) - preview, rejection checking, formatting
- `capacity.go` (103 lines) - pool/rate-limit convenience methods, reconciliation

**daemon.go: 1516 → 944 lines** (-572 lines, 38% reduction)

Verification: `go build ./cmd/orch/ && go vet ./... && go test ./pkg/daemon/...` — all pass.

Remaining 944 lines are:
- Daemon struct + constructors (~155 lines)
- Issue selection: NextIssue/NextIssueExcluding + expandTriageReadyEpics (~185 lines)
- Spawn orchestration: Once/OnceExcluding/OnceWithSlot/ReleaseSlot (~530 lines)
- Run loop (~20 lines)

The 530 lines of spawn orchestration contain ~60% code duplication between OnceExcluding and OnceWithSlot. Deduplicating these (refactoring, not extraction) would bring daemon.go to ~700-750 lines. Reaching the original 200-300 line target would additionally require extracting the Daemon struct mock functions into an options/functional-options pattern.
