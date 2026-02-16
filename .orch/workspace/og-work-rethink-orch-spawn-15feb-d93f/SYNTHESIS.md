# Session Synthesis

**Agent:** og-work-rethink-orch-spawn-15feb-d93f
**Issue:** orch-go-82eg
**Duration:** 2026-02-15T22:30 → 2026-02-15T23:00
**Outcome:** success

---

## Plain-Language Summary

The spawn command conflates three independent axes—model, backend, and visibility—into entangled flags and auto-override logic. `--opus` is really `--model opus --backend claude --tmux` in disguise. The `claude` backend always forces tmux (no headless claude spawns possible). Infrastructure keyword detection overrides backend choice when `--backend` isn't explicitly set, but the real pain point that triggered this issue was a naming collision: `--mode` controls implementation mode (tdd/direct), not backend, so `--mode opencode` silently did nothing while infrastructure detection forced claude. This document proposes a clean orthogonal design where each axis is independent, auto-overrides become advisory warnings, and the `--opus` shortcut is deprecated.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.

---

## TLDR

Audited all spawn flags and their interactions. Found 4 specific conflation points. Proposed orthogonal 3-axis design (model/backend/visibility) with advisory-only infrastructure detection and a clear migration path from current flags.

---

## Current Flag Audit

### All Spawn Flags (spawn_cmd.go:37-65)

| Flag | Type | Current Behavior |
|------|------|-----------------|
| `--model` | string | Model alias or full spec. Auto-selects backend: opus→claude, sonnet→opencode |
| `--backend` | string | "claude" or "opencode". Highest priority for backend selection |
| `--opus` | bool | **Conflated**: implies model=opus + backend=claude + visibility=tmux |
| `--tmux` | bool | Opt-in tmux window. Only works when backend≠claude (claude always uses tmux) |
| `--headless` | bool | Explicit headless. Overrides all visibility decisions including claude backend |
| `--inline` | bool | Blocking TUI in current terminal |
| `--attach` | bool | Implies `--tmux`, attaches after spawn |
| `--mode` | string | **Naming collision**: controls implementation mode (tdd/direct), NOT backend |

### Backend Selection Priority (spawn_cmd.go:837-892)

```
1. --backend flag          (explicit, highest priority)
2. --opus flag             (implies claude)
3. isInfrastructureWork()  (forces claude, prints message about tmux)
4. --model auto-select     (opus→claude, sonnet→opencode)
5. project config          (spawn_mode setting)
6. default                 (claude)
```

### Dispatch Logic (spawn_cmd.go:994-1024)

```
1. --inline?           → runSpawnInline()     [opencode TUI in terminal]
2. --headless?         → runSpawnHeadless()   [opencode HTTP API]
3. SpawnMode="claude"? → runSpawnClaude()     [ALWAYS tmux + claude CLI]
4. --tmux/orchestrator?→ runSpawnTmux()       [opencode TUI in tmux]
5. default             → runSpawnHeadless()   [opencode HTTP API]
```

### The 4 Conflation Points

#### 1. `--opus` = model + backend + visibility (3 axes collapsed)

`spawn_cmd.go:44,210,852-854`:
```go
spawnOpus bool // "Use Opus via Claude CLI in tmux (implies claude backend + tmux mode)"
```
One flag controls three independent concerns. There's no way to say "use opus model but let me choose the backend/visibility myself."

#### 2. Claude backend = always tmux (backend ↔ visibility)

`spawn_cmd.go:1008-1011`:
```go
if cfg.SpawnMode == "claude" {
    return runSpawnClaude(...)  // Always creates tmux window
}
```
`runSpawnClaude()` calls `spawn.SpawnClaude()` which calls `tmux.EnsureWorkersSession()` + `tmux.CreateWindow()`. There is no code path for headless claude CLI usage. The `--headless` flag is checked BEFORE the claude dispatch, so `--headless --backend claude` routes to `runSpawnHeadless()` which uses the OpenCode API—silently ignoring the claude backend.

#### 3. Infrastructure detection is mandatory, not advisory (spawn_cmd.go:855-876)

When `isInfrastructureWork()` returns true and `--backend` is not explicitly set, it forces `backend = "claude"` and prints "auto-applying escape hatch (--backend claude --tmux)". The keyword list is broad—includes "opencode", "dashboard", "agents.ts", "main.go", "SPAWN_CONTEXT", and any `cmd/orch` or `pkg/*` path.

The concrete pain point: spawning an agent to verify a fix TO OpenCode's SessionProcessor required exercising the OpenCode API code path. But mentioning "OpenCode" in the task description triggered infrastructure detection, which forced the claude backend, which bypassed the code path we needed to test.

#### 4. `--mode` naming collision

`spawn_cmd.go:42,208`:
```go
spawnMode string // Implementation mode: tdd or direct
```
vs
```go
spawnBackendFlag string // Spawn backend: claude or opencode
```

Users familiar with the older codebase (or the `--mode` convention from docs) try `--mode opencode` expecting backend selection. This silently sets implementation mode to "opencode" (meaningless value), while the backend remains on its default path. This is likely what happened in the orch-go-1nh7 incident.

---

## Proposed Design: Orthogonal 3-Axis Model

### Axis 1: Model (`--model`)
**What LLM to use.** No side effects.

```
--model opus      → anthropic/claude-opus-4-5-20251101
--model sonnet    → anthropic/claude-sonnet-4-5-20250929
--model flash     → google/gemini-3-flash-preview
--model pro       → google/gemini-2.5-pro
```

**Changes from current:**
- `--model` no longer auto-selects backend (removed the opus→claude, sonnet→opencode coupling)
- `--opus` flag deprecated → prints deprecation notice, maps to `--model opus --backend claude`
- Model is PURELY about LLM selection

### Axis 2: Backend (`--backend`)
**How to run the session.** Independent of model and visibility.

```
--backend claude    → Uses Claude CLI binary (Max subscription auth)
--backend opencode  → Uses OpenCode HTTP API (server at :4096)
```

**Default:** `claude` (unchanged from current)

**Changes from current:**
- Backend no longer implies visibility (claude ≠ tmux)
- Backend no longer auto-selected from model choice
- `--backend` remains highest priority, no auto-override can bypass it

### Axis 3: Visibility (`--visibility` or keep existing flags)

**How to observe the session.** Independent of backend.

Two options for the flag interface:

**Option A: Unified `--visibility` flag** (cleaner, breaking)
```
--visibility headless  → No UI, returns immediately (default for workers)
--visibility tmux      → Tmux window (default for orchestrators)
--visibility inline    → Blocking TUI in current terminal
```

**Option B: Keep existing boolean flags** (backward compatible)
```
--headless  → No UI (default for workers)
--tmux      → Tmux window
--inline    → Blocking TUI
--attach    → Implies --tmux, attaches after spawn
```

**Recommendation: Option B** (keep existing flags). The boolean flags are well-understood, backward compatible, and more unix-like. The `--visibility` flag adds no expressiveness since the three modes are mutually exclusive already.

### Supported Combinations (6 valid pairs)

| Backend | Visibility | Implementation | Status |
|---------|-----------|----------------|--------|
| claude | headless | Pipe context to `claude` CLI, capture exit code | **NEW** (needs impl) |
| claude | tmux | Current `runSpawnClaude()` | Exists |
| claude | inline | `claude` CLI in current terminal | **NEW** (needs impl) |
| opencode | headless | Current `runSpawnHeadless()` | Exists |
| opencode | tmux | Current `runSpawnTmux()` | Exists |
| opencode | inline | Current `runSpawnInline()` | Exists |

The key new capability: **headless claude** enables using Claude CLI backend for batch processing without tmux windows. This matters for daemon-driven spawns that want Claude CLI auth but don't need visual monitoring.

### Infrastructure Advisory (Replace Auto-Override)

**Current:** Infrastructure detection forces `--backend claude --tmux`, silently overriding the user's intent.

**Proposed:** Infrastructure detection becomes advisory-only:

```
⚠️  Infrastructure keywords detected in task description.
    Recommend: --backend claude --tmux
    Reason: agents working on infrastructure can kill themselves if spawned via OpenCode API.
    Override: --no-infra-warn to suppress this warning.
```

**Implementation:**
- `isInfrastructureWork()` returns the same bool
- When true AND no explicit `--backend` flag: print advisory, apply default (claude + tmux)
- When true AND explicit `--backend` flag: print advisory only, respect user's choice
- `--no-infra-warn` suppresses the advisory entirely

**Why advisory-only:** The orch-go-1nh7 incident proved that sometimes infrastructure work NEEDS the opencode path (to test it). Forcing claude defeats the purpose. The user should be warned but empowered to override.

### Validation Warnings (Not Errors)

| Combination | Warning |
|-------------|---------|
| opencode + opus | "opus auth may fail through OpenCode API. Recommend: --backend claude" |
| claude + flash | "Flash is a pay-per-token model. Claude CLI backend uses Max subscription. Consider: --backend opencode" |

These are warnings, not hard errors. The user may have legitimate reasons for unusual combinations.

---

## Fork: `--mode` Naming Collision

### Fork: Should `--mode` be renamed to avoid confusion?

**Options:**
- A: Rename `--mode` to `--impl-mode` (clear, breaks scripts)
- B: Rename `--mode` to `--approach` (clearer semantic, breaks scripts)
- C: Keep `--mode` but add validation (warn if value is "claude"/"opencode")
- D: Remove `--mode` entirely (tdd is the only meaningful value)

**Substrate says:**
- Constraint: "orch-go-1nh7 incident" — `--mode opencode` silently did nothing
- Principle: Premises before solutions — is `--mode` even used?

**Recommendation:** Option C (keep but validate). Add:
```go
if spawnMode == "opencode" || spawnMode == "claude" {
    return fmt.Errorf("--mode controls implementation approach (tdd/direct), not backend.\n  Did you mean: --backend %s", spawnMode)
}
```
This catches the common mistake without breaking existing usage of `--mode tdd` or `--mode direct`.

---

## Migration Path

### Phase 1: Non-Breaking (Implement First)

1. **Add validation to `--mode`**: Reject "claude"/"opencode" values with helpful message
2. **Make infrastructure detection advisory**: Print warning instead of forcing backend when `--backend` is explicitly set
3. **Decouple dispatch logic**: Stop routing `SpawnMode=="claude"` directly to `runSpawnClaude()`. Instead, resolve backend and visibility independently, then dispatch

### Phase 2: Deprecation Notices

4. **Deprecate `--opus`**: Add deprecation warning: "Use `--model opus --backend claude` instead"
5. **Remove model→backend auto-selection**: Stop auto-selecting backend based on `--model` value. Print advisory instead: "Opus model works best with claude backend. Use --backend claude if not already set."

### Phase 3: New Capabilities

6. **Implement headless claude**: Support `--backend claude --headless` (pipe context, no tmux)
7. **Implement inline claude**: Support `--backend claude --inline` (claude CLI in current terminal)

### Timing

Phase 1 can ship immediately (pure behavior fixes, no new flags). Phase 2 adds deprecation notices (non-breaking, just informational). Phase 3 adds new capabilities (requires `spawn.SpawnClaudeHeadless()` implementation).

---

## Revised `determineSpawnBackend()` (Pseudocode)

```go
func determineSpawnBackend(resolvedModel, task, beadsID, projectDir) (backend string, err error) {
    // 1. Explicit flag: highest priority, always respected
    if backendFlag != "" {
        backend = backendFlag
        // Advisory only - warn but don't override
        if isInfrastructureWork(task, beadsID) && backend != "claude" {
            fmt.Println("⚠️  Infrastructure work detected. Claude backend recommended.")
        }
        return backend, validate(backend)
    }

    // 2. Deprecated --opus flag (with deprecation notice)
    if opusFlag {
        fmt.Println("⚠️  --opus is deprecated. Use: --model opus --backend claude")
        return "claude", nil
    }

    // 3. Infrastructure advisory (apply default, don't force)
    if isInfrastructureWork(task, beadsID) && !noInfraWarn {
        fmt.Println("⚠️  Infrastructure work detected. Using claude backend + tmux.")
        fmt.Println("   Override with: --backend opencode")
        return "claude", nil  // Default, not forced
    }

    // 4. Project config
    if projCfg != nil && projCfg.SpawnBackend != "" {
        return projCfg.SpawnBackend, nil
    }

    // 5. Default
    return "claude", nil
}
```

Key changes from current:
- **Removed**: model→backend auto-selection (step 4 in current)
- **Changed**: infrastructure detection from mandatory to advisory
- **Changed**: explicit `--backend` flag ALWAYS wins (currently it does, but the naming collision masked this)

## Revised `dispatchSpawn()` (Pseudocode)

```go
func dispatchSpawn(input, cfg, ...) error {
    // Step 1: Resolve visibility (independent of backend)
    visibility := resolveVisibility(input.Inline, input.Headless, input.Tmux,
                                     input.Attach, cfg.IsOrchestrator)

    // Step 2: Dispatch on backend × visibility matrix
    switch cfg.SpawnMode {
    case "claude":
        switch visibility {
        case "inline":  return runSpawnClaudeInline(...)   // NEW
        case "tmux":    return runSpawnClaude(...)          // Existing
        case "headless": return runSpawnClaudeHeadless(...) // NEW
        }
    case "opencode":
        switch visibility {
        case "inline":  return runSpawnInline(...)    // Existing
        case "tmux":    return runSpawnTmux(...)      // Existing
        case "headless": return runSpawnHeadless(...) // Existing
        }
    }
}

func resolveVisibility(inline, headless, tmux, attach, isOrchestrator bool) string {
    if inline { return "inline" }
    if headless { return "headless" }
    if tmux || attach { return "tmux" }
    if isOrchestrator { return "tmux" }
    return "headless"
}
```

Key change: The dispatch is now a clean 2×3 matrix (backend × visibility) instead of the current tangled if/else chain.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-work-rethink-orch-spawn-15feb-d93f/SYNTHESIS.md` - This design document

### Files Modified
- None (design only, no code changes)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- `spawn_cmd.go:852-854`: `--opus` flag sets backend to claude without model or visibility options
- `spawn_cmd.go:1008-1011`: `cfg.SpawnMode == "claude"` unconditionally routes to tmux
- `spawn_cmd.go:855-876`: Infrastructure detection forces claude backend at priority 3 (below explicit flag but above model auto-selection)
- `spawn_cmd.go:877-888`: Model auto-selects backend (opus→claude, sonnet→opencode)
- `spawn_cmd.go:42`: `spawnMode` flag name collides semantically with backend selection
- `spawn_cmd.go:2044-2102`: Infrastructure keyword list is broad, matches common terms like "dashboard", "main.go"
- `pkg/spawn/claude.go:12-70`: `SpawnClaude()` always creates tmux window, no headless path

### Tests Run
```bash
# No tests run - this is a design-only session
```

---

## Knowledge (What Was Learned)

### Decisions Made
- **Keep boolean visibility flags** (`--tmux`, `--headless`, `--inline`) over a unified `--visibility` flag — backward compatible, more unix-like
- **Make infrastructure detection advisory** — the exact situation that motivated this issue (orch-go-1nh7) required the user to override infrastructure detection
- **Deprecate `--opus` rather than remove** — gives users time to migrate scripts
- **Phase implementation in 3 steps** — non-breaking fixes first, then deprecations, then new capabilities

### Constraints Discovered
- `SpawnClaude()` in `pkg/spawn/claude.go` has no headless path — implementing headless claude requires new code in that package
- The `--mode` flag (tdd/direct) is rarely used but can't be removed without audit of downstream consumers

### Externalized via `kb quick`
- (see below)

---

## Next (What Should Happen)

**Recommendation:** close (design complete)

### If Close
- [x] All deliverables complete (SYNTHESIS.md with audit, design, migration path)
- [x] No tests needed (design-only session)
- [x] Ready for orchestrator review

### Follow-up Implementation Issues
The orchestrator should create implementation issues for:

1. **Phase 1**: Add `--mode` validation, make infrastructure advisory, decouple dispatch
2. **Phase 2**: Deprecate `--opus`, remove model→backend auto-selection
3. **Phase 3**: Implement headless claude, inline claude

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does the daemon spawn path interact with this? The daemon uses `runSpawnWithSkillInternal` with `daemonDriven=true` which skips triage but uses the same backend selection. Does the daemon need independent backend configuration?
- Should `runSpawnTmux()` (opencode in tmux) use the opencode `--attach` mode or the HTTP API? Currently it uses `--attach` mode — this is orthogonal to the backend/visibility split but worth noting.

**What remains unclear:**
- Whether headless claude (piping context to CLI, capturing output) is reliable enough for daemon use. The current claude CLI assumes interactive terminal. May need `--print` or similar flag.

*(These are out of scope per the task definition but should inform Phase 3 implementation.)*

---

## Session Metadata

**Skill:** design-session
**Model:** opus
**Workspace:** `.orch/workspace/og-work-rethink-orch-spawn-15feb-d93f/`
**Beads:** `bd show orch-go-82eg`
