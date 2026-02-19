# Probe: User default_model treated as explicit for backend

**Model:** spawn-architecture
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does a user-configured `default_model` prevent infra detection from forcing the claude backend when no CLI `--model` flag is set?

---

## What I Tested

```bash
go test ./pkg/orch -run TestDetermineSpawnBackend_UserDefaultModelPreventsInfraOverride
```

---

## What I Observed

Before the fix, the test failed with infra detection forcing the claude backend even when `default_model` was set in user config. After updating `DetermineSpawnBackend` to treat `default_model` as explicit, the test passed (backend no longer forced to claude).

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [x] **Extends** model with: user-config `default_model` is treated as an explicit model choice for backend selection, so infra escape hatch becomes advisory instead of forcing claude.

---

## Notes

Test failure reproduced the bug; fix validated by passing test.
