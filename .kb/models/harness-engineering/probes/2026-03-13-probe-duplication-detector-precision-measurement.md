# Probe: Duplication Detector Precision — Assumed 0% FP, Never Validated

**Model:** harness-engineering
**Date:** 2026-03-13
**Status:** Complete

---

## Question

The harness-engineering model claims (Invariant 7): "Enforcement without measurement is theological; enforcement with measurement is empirical." The duplication detector runs on every completion, assumes 0% false positives, but has never been validated. What is the actual precision?

---

## What I Tested

Extracted all 67 `duplication.detected` events from `~/.orch/events.jsonl` (259 total match occurrences across 54 unique function pairs). Manually classified each unique pair by reading both functions in the codebase:

- **True Positive (TP):** Shared logic that could/should be extracted into a common helper
- **False Positive (FP):** Intentionally parallel, coincidentally similar, or extraction would worsen code

```bash
# Extract all duplication events
grep '"duplication.detected"' ~/.orch/events.jsonl | wc -l
# → 67 events

# Count unique pairs with frequency
grep '"duplication.detected"' ~/.orch/events.jsonl | python3 -c "
import json, sys
from collections import Counter
pairs = Counter()
for line in sys.stdin:
    for m in json.loads(line)['data']['matches']:
        pairs[tuple(sorted([m['func_a'], m['func_b']]))] += 1
for (a,b), c in pairs.most_common(): print(f'{c:3d}x  {a} <-> {b}')
"
# → 54 unique pairs, 259 total occurrences
```

For each unique pair, read both functions and classified as TP or FP based on whether extraction is warranted.

---

## What I Observed

### Precision Results

| Metric | Value |
|--------|-------|
| Total match occurrences | 259 |
| True Positives | 164 (63.3%) |
| False Positives | 90 (34.7%) |
| Unclassified (borderline) | 5 (1.9%) |
| **Precision (TP / (TP + FP))** | **64.6%** |
| Unique TP pairs | 33 |
| Unique FP pairs | 15 |
| Unique pair precision | 68.8% |

### FP Root Cause Breakdown

| Category | Occurrences | % of FP | Example |
|----------|-------------|---------|---------|
| Different semantics | 42 | 47% | `parseBeadsIDs ↔ parseBeadsLine` — different parsing, similar AST shape |
| Structural coincidence | 36 | 40% | `(Logger).Log ↔ WriteCheckpoint` — both write JSONL, different domains |
| Self-match | 6 | 7% | `inferSkillFromBeadsIssue ↔ inferSkillFromBeadsIssue` — same function matched twice |
| Opposite operations | 6 | 7% | `CloseIssue ↔ GetComments` — complementary ops sharing CLI boilerplate |

### Top FP Pairs by Occurrence

| Pair | Occurrences | Category |
|------|-------------|----------|
| `(Logger).Log ↔ WriteCheckpoint` | 23 | Structural coincidence |
| `handleOrphanDetectionResult ↔ handleRecoveryResult` | 10 | Structural coincidence |
| `getWebChangesFromRecentCommits ↔ hasGoChangesInRecentCommits` | 7 | Different semantics |
| `GenerateEcosystemContext ↔ loadOrchSkillContent` | 6 | Different semantics |
| `GenerateEcosystemContext ↔ collectReflectSuggestions` | 6 | Different semantics |
| `parseBeadsIDs ↔ parseBeadsLine` | 6 | Different semantics |
| `DeriveInvariantThreshold ↔ DeriveVerificationThreshold` | 6 | Different semantics |

### Notable TP Findings

The detector correctly identifies real duplication. The top TP clusters:

1. **Beads RPC wrapper duplication** (35+ occurrences): `AddLabel`, `removeLabel`, `addReworkComment`, `UpdateIssueAssignee`, `UpdateIssueStatus`, etc. — all implement identical RPC-first-with-CLI-fallback pattern. Extractable to a generic `callBeadsRPC()` helper (~60 lines savings).

2. **Template ensure duplication** (15 occurrences): `EnsureProbeTemplate`, `EnsureSynthesisTemplate`, `EnsureSessionHandoffTemplate`, `EnsureFailureReportTemplate` — identical stat → mkdir → write pattern. One `ensureTemplate(dir, file, content)` helper would replace all four.

3. **Literal copies** (14 occurrences): `hasGoChangesInRecentCommits` / `HasGoChangesInRecentCommits`, `addApprovalComment` / `addBeadsComment` — identical functions in different files.

4. **Lifecycle method duplication** (15 occurrences): `Abandon`, `Complete`, `ForceComplete` — three methods with 97.4% similarity. Extractable shared transition logic.

### Allowlist Gap

The existing `.dupdetectignore` only covers `(Logger).Log*` and `(EventLoggerAdapter).Log*`. The `(Logger).Log ↔ WriteCheckpoint` FP (23 occurrences — largest single FP source) is NOT suppressed because `WriteCheckpoint` doesn't match `(Logger).Log*`. The allowlist suppresses same-type pairs only.

### Self-Match Bug

The detector flags `inferSkillFromBeadsIssue ↔ inferSkillFromBeadsIssue` (same function name, 100% similarity) — this appears to be the same function appearing in both the "modified" and "corpus" partitions of `CheckModifiedFilesProject`. This is a detector bug, not a content issue.

---

## Model Impact

- [x] **Confirms** invariant 7: "Enforcement without measurement is theological" — the duplication detector was assumed to have 0% FP but actually has 35% FP rate. Without measuring precision, we were generating noise at a rate indistinguishable from signal.

- [x] **Confirms** invariant 2 (Gate Calibration Death Spiral): While the detector is advisory (not blocking), the 35% FP rate produces alert fatigue. 23 of 67 events (34%) contain the same `(Logger).Log ↔ WriteCheckpoint` FP, meaning the orchestrator sees the same false signal repeatedly, training them to ignore duplication advisories entirely.

- [x] **Extends** model with: **FP taxonomy for AST-based detection** — four root cause categories with distinct remediation paths:
  - *Different semantics* (47%): raise threshold or add structural heuristics (e.g., penalize pairs where function signatures differ significantly)
  - *Structural coincidence* (40%): expand allowlist patterns or add cross-domain filters
  - *Self-match* (7%): deduplicate functions before comparison (detector bug)
  - *Opposite operations* (7%): filter pairs where function names share a root but differ by operation verb

- [x] **Extends** model with: **Precision measurement design** — three implementation paths (see Notes), with allowlist-based suppression logging as the lowest-ceremony option that leverages existing infrastructure.

---

## Notes

### Recommended Precision Improvement Path

**Immediate (no code change):** Expand `.dupdetectignore` to cover the 15 known FP pairs. This alone would raise precision from 65% to ~95% for currently-known pairs. Estimated suppressions:

```
# Structural coincidence — JSONL writers
WriteCheckpoint

# Structural coincidence — daemon periodic handlers
handle*Result

# Different semantics — different git diff parsing
getWebChangesFromRecentCommits

# Different semantics — different parsing functions
parseBeads*

# Different semantics — threshold lookups
Derive*Threshold

# Different semantics — ecosystem context generators
GenerateEcosystemContext

# Opposite operations — beads CLI wrappers with different operations
CloseIssue

# Self-match suppression needs code fix, not allowlist
```

**Short-term (code fix):** Fix self-match bug in `CheckModifiedFilesProject` — deduplicate functions that appear in both modified and corpus partitions before comparison.

**Medium-term (measurement surface):** Add `duplication.suppressed` event type. When a pair matches the allowlist, log it with the pattern that matched. This creates passive precision measurement:

```
precision = detected / (detected + suppressed)
```

Monitor this ratio over time. If precision stays high (>90%), the detector is well-calibrated. If it drops, new FP patterns are emerging.

### Precision vs. Recall Tradeoff

At 0.85 threshold, precision is 65%. Raising threshold to 0.90 would eliminate many "different semantics" FPs (42 occurrences) but would also miss some true duplication (e.g., `Fallback*` methods at 92-96% similarity). The current threshold is weighted toward recall; if the goal is actionable signal, raising to 0.90 + expanding the allowlist would be more effective than threshold change alone.

### Connection to Harness Audit (Mar 12)

The hook infrastructure audit (probe: 2026-03-12) found "zero observability" across all hooks. This probe finds the same pattern in the duplication detector: enforcement exists but measurement doesn't. The detector is more visible (produces printed output) but equally unmeasured (no tracking of detection accuracy or action taken on detections).
