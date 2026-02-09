# Probe: Can `bd-sync-safe.sh` recover from hash-mismatch import stalls without manual kill/retry?

**Model:** /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

When `bd sync` enters the `"JSONL content differs from last sync"` import-first path and exceeds a timeout budget, can the wrapper script automatically recover by running explicit `--import-only` and then retrying sync?

---

## What I Tested

**Command/Code:**

```bash
# 1) Script smoke in real repo
BD_SYNC_SAFE_TIMEOUT_SECONDS=60 ./scripts/bd-sync-safe.sh

# 2) Deterministic timeout simulation with stubbed bd binary
PATH="/tmp/bd-sync-safe-test/bin:$PATH" BD_SYNC_SAFE_TIMEOUT_SECONDS=1 ./scripts/bd-sync-safe.sh

# 3) Verify retry sequence was executed
python3 - <<'PY'
from pathlib import Path
print(Path('/tmp/bd-sync-safe-test/calls.log').read_text())
PY
```

**Environment:**

- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Changed file: `scripts/bd-sync-safe.sh`
- Stubbed `bd` forced first sync call to emit hash-mismatch lines and sleep past timeout

---

## What I Observed

**Output (key lines):**

```text
bd-sync-safe: command timed out after 1s: bd sync --no-daemon --sqlite --no-pull --no-push
→ JSONL content differs from last sync
→ Importing JSONL first to prevent stale DB from overwriting changes...
→ Timed out in JSONL hash-mismatch import path; retrying with explicit import-only...
→ Importing from JSONL...
Import complete: 0 created, 0 updated, 1 unchanged
→ Exporting pending changes to JSONL...
✓ Sync complete

calls.log:
sync --no-daemon --sqlite --no-pull --no-push
sync --no-daemon --sqlite --import-only
sync --no-daemon --sqlite --no-pull --no-push
```

---

## Model Impact

**Verdict:** extends — local wrapper resilience for CLI fallback latency spikes

**Details:**
The model already captures RPC-first fallback behavior and staleness friction. This probe adds an operational mitigation at the orch layer: bounded sync timeout plus targeted retry only when hash-mismatch import path is detected. This removes the manual kill/retry loop for the known failure mode while preserving normal sync behavior.

**Confidence:** High — verified with deterministic timeout simulation and call-order evidence.
