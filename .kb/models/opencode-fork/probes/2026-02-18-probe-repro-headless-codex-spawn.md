# Probe: Repro Headless Codex Spawn

**Model:** opencode-fork
**Date:** 2026-02-18
**Status:** BLOCKED - Spawn concurrency limit reached

---

## Question

Does a headless (OpenCode HTTP API) spawn using `--backend opencode` and `--model codex` successfully create and complete a session, or does it fail during session creation/provider initialization?

---

## What I Tested

Attempted a manual headless spawn using OpenCode HTTP API:

```
orch spawn --bypass-triage --backend opencode --model codex --no-track --light investigation "repro headless codex spawn: respond with OK and then wait"
```

---

## What I Observed

Spawn was blocked before session creation:

```
Error: concurrency limit reached: 9 active agents (max 5)
```

No codex session was created; repro is blocked until concurrency limit is cleared or raised.

---

## Model Impact

No impact yet. The codex headless spawn path was not exercised due to concurrency gating.
