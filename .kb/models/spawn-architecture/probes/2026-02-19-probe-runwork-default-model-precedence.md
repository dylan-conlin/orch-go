# Probe: runWork default_model precedence

**Model:** spawn-architecture
**Date:** 2026-02-19
**Status:** Active

---

## Question

Does the daemon-driven work path treat user default_model as a CLI override, causing project config model to be ignored?

---

## What I Tested

Reviewed runWork behavior and validated spawn resolve precedence test coverage.

```bash
rg "spawnModel" -n cmd/orch/spawn_cmd.go pkg -g"*.go"
git show HEAD:cmd/orch/spawn_cmd.go | rg -n "default_model"
rg -n "default_model" cmd/orch/spawn_cmd.go
go test ./pkg/spawn -run TestResolve_BugClass10_UserDefaultModelNotInjectedAsCLI -count=1
```

---

## What I Observed

- Committed version shows runWork loading user config default_model into spawnModel (git show HEAD:cmd/orch/spawn_cmd.go | rg "default_model").
- Current working tree replaces that block with a note to avoid injecting default_model into spawnModel.
- spawnModel is passed into spawn.ResolveInput as CLI.Model (cmd/orch/spawn_cmd.go:535-539), so injecting user default_model here would elevate it to highest precedence.
- Resolve tests for default_model precedence pass in pkg/spawn (TestResolve_BugClass10_UserDefaultModelNotInjectedAsCLI).

---

## Model Impact

- [ ] **Confirms** invariant: 
- [ ] **Contradicts** invariant: 
- [x] **Extends** model with: Daemon work path previously elevated user default_model to CLI priority before resolve, bypassing project config model precedence.

---

## Notes

This probe focuses on the daemon work path in runWork, not the resolve algorithm itself.
