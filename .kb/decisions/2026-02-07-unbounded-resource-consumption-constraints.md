# Decision: Permanent Constraints Against Unbounded Resource Consumption

**Date:** 2026-02-07
**Status:** Accepted
**Deciders:** Dylan, Orchestrator
**Context:** Post-mortem of system reliability crisis — 5 failure modes sharing one root cause (unbounded resource consumption without lifecycle management). 779 investigations in ~2 months failed to elevate symptom-level fixes to pattern-level diagnosis.

## Problem

Every major reliability failure in the orch ecosystem shares the same DNA: a resource is created (goroutine, subprocess, cache entry, memory allocation) without a corresponding lifecycle that bounds its consumption and ensures its cleanup. This class of defect shipped 5 times because nothing in the development process — agent prompts, code review, linting, investigation synthesis — checks for it.

## Decision

Adopt 5 permanent constraints that make "unbounded resource consumption" a defect category that cannot ship.

## Constraints

### C1: Every goroutine, subprocess, and cache must have a bounded lifetime

- No `go func()` without a `context.Context` that cancels
- No `exec.Command` without `context.WithTimeout`
- No `map` used as cache without max size and eviction
- **Enforcement:** Custom `golangci-lint` linter rule
- **Scope:** All Go code in orch-go, orch-cli

### C2: Process creation requires process cleanup

- Any `exec.Command().Start()` must register the process for SIGTERM-on-parent-shutdown
- Use `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` for process group management
- Orphan processes are a P0 defect
- **Enforcement:** Pre-commit hook (grep for `exec.Command` without cleanup pattern)
- **Scope:** All Go code that spawns external processes

### C3: Agent spawn prompts include resource audit scope

- Every `orch spawn` touching infrastructure code must include in the prompt:
  > "Audit all resources this component creates (goroutines, subprocesses, connections, caches) and ensure each has bounded lifetime and cleanup on shutdown."
- **Enforcement:** Template change to SPAWN_CONTEXT.md generation
- **Scope:** All infrastructure-touching spawns (not pure business logic)

### C4: Caches require max-size at construction

- No cache may be created without specifying maximum entry count
- `NewCache()` without a size parameter is a compilation error by API design
- Constructor signature: `NewCache(maxSize int, ttl time.Duration)`
- **Enforcement:** API design (constructor won't compile without args) + code review
- **Scope:** All in-memory caches

### C5: Weekly resource-class investigation synthesis

- Every Friday: query `kb context` for resource-related terms (subprocess, goroutine, cache, memory, process, connection, OOM, zombie)
- If 3+ investigations share a resource category → escalate to architectural constraint
- **Enforcement:** Manual until `kb reflect` ships, then automated detection
- **Scope:** All investigations across all projects

## Consequences

### Positive
- "Unbounded resource consumption" becomes a structurally detectable defect
- Agents are prompted to think about resource lifecycle, not just feature correctness
- Pattern clustering becomes visible before crisis, not after
- New code cannot introduce the most common failure mode without tripping a gate

### Negative
- C1 linter will generate false positives on intentionally-long-lived goroutines (mitigate with `//nolint:boundedlifetime` annotation)
- C3 adds ~1 sentence to spawn prompts (negligible cost)
- C5 is manual overhead until `kb reflect` automation ships

### Risks
- Constraints without enforcement atrophy. C1 and C2 must ship as automated checks within 1 week or they're decorative.
- C5 depends on discipline. If skipped for 2 weeks, the pattern blindness returns.

## References

- **Root investigation:** `.kb/investigations/2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md`
- **Prior:** `.kb/investigations/2026-02-07-inv-opencode-server-memory-leak-4gb.md`
- **Prior:** `.kb/investigations/2026-01-22-inv-strategic-audit-daemon-reliability-multiple.md`
- **Prior:** `.kb/investigations/2026-01-26-inv-analyze-local-share-opencode-crash.md`
