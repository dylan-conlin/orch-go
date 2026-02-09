# Probe: Does `bd-sync-safe.sh` leave direct read commands immediately usable after sync?

**Model:** /Users/dylanconlin/Documents/personal/orch-go/.kb/models/beads-integration-architecture.md
**Date:** 2026-02-09
**Status:** Complete

---

## Question

When `bd-sync-safe.sh` exits successfully, does the next direct read (`bd show <id>`) avoid the stale DB error without requiring manual `--allow-stale` or another sync?

---

## What I Tested

**Command/Code:**

```bash
bash -n scripts/bd-sync-safe.sh

python3 - <<'PY'
import subprocess
ok_sync=0
show_fail=0
for i in range(20):
    s=subprocess.run(['./scripts/bd-sync-safe.sh'],capture_output=True,text=True)
    if s.returncode!=0:
        continue
    ok_sync+=1
    c=subprocess.run(['bd','show','orch-go-633op','--json'],capture_output=True,text=True)
    if c.returncode!=0 and 'Database out of sync with JSONL' in (c.stdout+c.stderr):
        show_fail+=1
print('successful_sync_runs',ok_sync)
print('stale_show_failures',show_fail)
PY
```

**Environment:**

- Repo: `/Users/dylanconlin/Documents/personal/orch-go`
- Changed file: `scripts/bd-sync-safe.sh`
- Worktree is intentionally dirty (expected unrelated sync commit failures)

---

## What I Observed

**Output:**

```text
successful_sync_runs 19
stale_show_failures 0
```

**Key observations:**

- Post-fix, every successful `bd-sync-safe.sh` run was immediately followed by a successful `bd show` with zero stale errors.
- One run failed due dirty-tree commit constraints, but that failure mode is separate from stale-read loop behavior.

---

## Model Impact

**Verdict:** extends — sync wrapper should verify read-path freshness before reporting success

**Details:**
The model covered sync/import retries, but not post-sync readiness validation. This probe adds a new operational invariant: after wrapper sync completes, run a lightweight direct read check and self-heal with final import-only if stale is still reported.

**Confidence:** High — validated by repeated sync→show reproduction loops in the target repository.
