# Decision: OpenCode Binary Resolution Patterns

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigation 2026-01-18-design-opencode-binary-resolution.md

## Summary

Unified binary resolution via `pkg/binutil` package following pattern: OPENCODE_BIN env var → PATH lookup → known locations (`~/.bun/bin`, `~/.local/bin`, `/opt/homebrew/bin`, etc). Eliminates three inconsistent patterns across codebase. Applies to both `opencode` and `bd` binary resolution.

## The Problem

Three inconsistent opencode binary resolution patterns in codebase:

**Pattern A - OPENCODE_BIN environment variable:**
```go
opencodeBin := "opencode"
if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
    opencodeBin = bin
}
```
Used in: `pkg/tmux/tmux.go:265-267`, `pkg/opencode/client.go:70-75`

**Pattern B - Hardcoded in shell commands:**
```go
cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 ...")
```
Used in: `cmd/orch/spawn_cmd.go:405`, `cmd/orch/doctor.go:511`

**Pattern C - PATH-only lookup:**
```go
cmd := exec.Command("opencode", args...)
```
Used in: `pkg/tmux/tmux.go:427`

**Pain point:** Pattern B (hardcoded in shell) fails when `~/.bun/bin` not in PATH. Symlink exists at `~/.bun/bin/opencode` but processes spawned by orch-go (especially via launchd daemon) inherit minimal PATH that doesn't include user-specific directories.

## The Decision

### Create pkg/binutil with Unified Resolution

**New package:** `pkg/binutil/binutil.go`

**Core function:**
```go
func ResolveBinary(name string, envVarName string, searchPaths []string) (string, error)
```

**Resolution order:**
1. **Check environment variable** (e.g., OPENCODE_BIN) - Explicit user override takes precedence
2. **Try exec.LookPath()** - Fast path when binary is in PATH
3. **Check known locations** - Fallback for minimal PATH environments
4. **Return error with searched paths** - Clear error message listing all locations tried

**Search paths for opencode:**
- `$HOME/bin/opencode`
- `$HOME/.bun/bin/opencode`
- `$HOME/.local/bin/opencode`
- `$HOME/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode`
- `/usr/local/bin/opencode`
- `/opt/homebrew/bin/opencode`

### Migrate All Resolution to binutil

**Files to modify:**

1. **pkg/beads/client.go** - Replace `ResolveBdPath()` with `binutil.ResolveBinary("bd", "BD_BIN", bdSearchPaths)`

2. **pkg/opencode/client.go** - Add `ResolveOpencodePath()` using `binutil.ResolveBinary("opencode", "OPENCODE_BIN", opencodeSearchPaths)`

3. **pkg/tmux/tmux.go** - Replace inline checks with `binutil.ResolveBinary()`

4. **cmd/orch/spawn_cmd.go:405** - Replace hardcoded "opencode" with resolved path:
   ```go
   opencodePath, err := opencode.ResolveOpencodePath()
   if err != nil { return err }
   cmd := exec.Command("sh", "-c", fmt.Sprintf("ORCH_WORKER=1 %s serve --port 4096 ...", opencodePath))
   ```

5. **cmd/orch/doctor.go:511** - Same interpolation fix as spawn_cmd.go

6. **cmd/orch/attach.go:68-70** - Replace inline check with binutil call

## Why This Design

### Principle: PATH-Only Resolution is Unreliable in Orchestration Contexts

Processes spawned by orch-go (via launchd daemon, minimal environments) inherit restricted PATH that doesn't include user-specific directories like `~/.bun/bin`. Relying solely on PATH means system breaks in exactly the environments where orchestration is most needed.

### Proven Pattern: ResolveBdPath() Already Works

The beads client (`pkg/beads/client.go:42-81`) demonstrates working solution: try PATH first (fast), then check known locations (reliable). Same problem, proven solution.

### Constraint: Never Rely on PATH Alone in Orchestration Context

From CLAUDE.md "CLI PATH Fix" section: OpenCode server inherits minimal PATH excluding ~/go/bin, ~/.local/bin, ~/bin, /opt/homebrew/bin. This is expected context for orchestration.

### Principle: Coherence Over Patches

Having three different resolution patterns (env var, hardcoded shell, PATH-only) creates fragility. Some code paths work while others fail. Unified utility enforces consistency.

## Trade-offs

**Accepted:**
- Small refactor required (30 minutes estimated)
- New package adds one file (~100 lines)
- One extra env var check (~nanoseconds overhead)

**Rejected:**
- Quick patch (only fix two failing shell commands): Doesn't address pattern inconsistency, technical debt accumulates
- Environment variable propagation only: Requires every environment to set OPENCODE_BIN, brittle, doesn't help fresh installs
- Known locations first (vs PATH first): PATH check is fast when env var not set, maintain performance

## Constraints

1. **OPENCODE_BIN takes precedence** - Explicit user override via env var documented in CLAUDE.md
2. **Resolution at startup, cache result** - Binary location unlikely to change during process lifetime
3. **Clear error messages** - List all searched locations when binary not found
4. **Symlink resolution** - `filepath.Abs()` resolves symlinks by default, document this behavior

## Implementation Notes

**What to implement first:**
1. Create `pkg/binutil/binutil.go` with `ResolveBinary()` function
2. Add comprehensive tests (env var override, PATH lookup, known locations fallback, error messages)
3. Migrate opencode resolution (most critical, frequent failures)
4. Migrate bd resolution (already works, but unify pattern)

**Resolution sequence for opencode:**
```go
// 1. Check OPENCODE_BIN env var
if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
    return filepath.Abs(bin)
}

// 2. Try PATH
if path, err := exec.LookPath("opencode"); err == nil {
    return filepath.Abs(path)
}

// 3. Check known locations
for _, searchPath := range opencodeSearchPaths {
    expanded := os.ExpandEnv(searchPath)  // Handle $HOME
    if _, err := os.Stat(expanded); err == nil {
        return expanded, nil
    }
}

// 4. Error with searched paths
return "", fmt.Errorf("opencode not found. Searched: PATH, %v. Ensure opencode installed or set OPENCODE_BIN", searchPaths)
```

**Things to watch out for:**
- Shell escaping: Ensure proper quoting if path contains spaces (unlikely but possible)
- Symlink handling: Document that Abs() resolves symlinks
- Windows compatibility: Use `%USERPROFILE%` if Windows support planned
- Race conditions: Call at init time and cache vs. re-resolve each time

**Success criteria:**
- Headless spawn works without `~/.bun/bin` in PATH
- Error message lists searched locations
- All resolution patterns unified (grep shows consistent binutil use)
- Existing functionality preserved (all spawn modes work)
- Test simulates minimal PATH, verifies opencode found via known locations

## References

**Investigation:**
- `.kb/investigations/2026-01-18-design-opencode-binary-resolution.md` - Root cause and design

**Files:**
- `cmd/orch/spawn_cmd.go:405` - Hardcoded "opencode" in shell (failure point)
- `cmd/orch/doctor.go:511` - Second hardcoded instance
- `pkg/beads/client.go:31-81` - Proven pattern (ResolveBdPath)
- `pkg/tmux/tmux.go:265-267` - OPENCODE_BIN env var pattern
- `pkg/opencode/client.go:70-75` - getOpencodeBin() helper

**Documentation:**
- `CLAUDE.md` - Documents PATH issues and opencode setup
- "CLI PATH Fix (via ~/.bun/bin symlinks)" section - Documents PATH limitations

**Principles:**
- Coherence Over Patches - Don't accumulate inconsistent workarounds
- Pragmatism - Fast path for common case (binary in PATH) before fallbacks
