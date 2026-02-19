# Probe: Project config spawn_mode explicitness

**Model:** daemon-autonomous-operation
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does backend resolution treat project config `spawn_mode` as explicit only when the key is present in `.orch/config.yaml`, so a project config without `spawn_mode` does not override an explicit user backend?

---

## What I Tested

[Command run, code examined, or experiment performed — not just code review]

```bash
go test ./pkg/orch -run TestDetermineSpawnBackend -v
```

---

## What I Observed

- `TestDetermineSpawnBackend_ProjectConfigWithoutSpawnModeDoesNotOverrideUserBackend` passed, confirming a project config without `spawn_mode` no longer overrides explicit user backend selection.
- No regressions in other `DetermineSpawnBackend` tests.

---

## Model Impact

- [ ] **Confirms** invariant: [which one]
- [ ] **Contradicts** invariant: [which one] — [what's actually true]
- [x] **Extends** model with: backend resolution now treats project `spawn_mode` as explicit only when the YAML key is present, so user backend preferences are not overridden by project defaults

---

## Notes

[Any additional context, caveats, or follow-up questions]
