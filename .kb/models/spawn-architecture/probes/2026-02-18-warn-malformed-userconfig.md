# Probe: Warn on malformed user config during spawn

**Model:** spawn-architecture
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does the spawn pipeline surface user config load failures so malformed ~/.orch/config.yaml does not silently ignore default_model/backend preferences?

---

## What I Tested

Added targeted tests that set HOME to a temp dir with malformed ~/.orch/config.yaml and call the spawn helper loaders.

```bash
go test ./cmd/orch -run TestLoadUserConfig -v
```

---

## What I Observed

Both tests passed, and the warning string included the config path plus the backend/default_model hint.

---

## Model Impact

- [x] **Confirms** invariant: spawn pipeline warns on user config load failures
- [ ] **Contradicts** invariant: spawn pipeline silently ignores user config load failures
- [x] **Extends** model with: warning includes config path and fallback note

---

## Notes

Tests validate the warning formatting and load paths; full CLI spawn warning is covered by the same helper usage.
