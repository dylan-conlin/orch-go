# Probe: Does `orch-dashboard restart` auto-start daemon by default?

**Model:** `.kb/models/daemon-autonomous-operation/model.md`
**Date:** 2026-02-09
**Status:** Complete

---

## Question

Model claim: daemon runs continuously via launchd and is operationally independent from dashboard service restarts. Does `orch-dashboard restart` currently auto-start `orch daemon run` under overmind, and can default startup be changed to avoid unintended autonomous spawns?

---

## What I Tested

**Command/Code:**
```bash
orch-dashboard restart && overmind status

# Patch ~/bin/orch-dashboard to add default:
#   --ignored-processes daemon
# and opt-in override:
#   ORCH_DASHBOARD_START_DAEMON=1

orch-dashboard restart && overmind status
ORCH_DASHBOARD_START_DAEMON=1 orch-dashboard restart && overmind status
```

**Environment:**
- Host: macOS (local orch environment)
- Dashboard script: `/Users/dylanconlin/bin/orch-dashboard`
- Project dir: `/Users/dylanconlin/Documents/personal/orch-go`

---

## What I Observed

**Output:**
```text
Before patch (default restart):
PROCESS   PID       STATUS
api       48559     running
daemon    48560     running
doctor    48561     running

After patch (default restart):
ℹ Daemon auto-start disabled (default)
PROCESS   PID       STATUS
api       50291     running
doctor    50292     running

After patch with opt-in env:
ℹ Daemon auto-start enabled via ORCH_DASHBOARD_START_DAEMON=1
PROCESS   PID       STATUS
api       54141     running
daemon    54143     running
doctor    54150     running
```

**Key observations:**
- Pre-fix behavior reproduced: dashboard restart started daemon automatically.
- Post-fix default behavior excludes daemon while keeping api+doctor running.
- Daemon startup remains available as explicit opt-in via `ORCH_DASHBOARD_START_DAEMON=1`.

---

## Model Impact

**Verdict:** extends — daemon should not auto-start from dashboard restart by default.

**Details:**
The probe confirms a drift from model intent (daemon autonomy/launchd lifecycle) existed in current startup behavior. Defaulting `orch-dashboard` to ignore daemon restores separation: dashboard restart handles dashboard services, while daemon startup is explicit. This adds a practical invariant: dashboard restart must be safe from unintended issue spawning unless opt-in is provided.

**Confidence:** High — direct before/after reproduction against the real startup path with explicit process table evidence.
