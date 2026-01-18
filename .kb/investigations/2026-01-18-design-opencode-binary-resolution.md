<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** orch-go has three inconsistent opencode binary resolution patterns; hardcoded shell commands fail when ~/.bun/bin not in PATH despite valid symlink.

**Evidence:** Grep found Pattern A (OPENCODE_BIN env var), Pattern B (hardcoded "opencode" in spawn_cmd.go:405, doctor.go:511), Pattern C (PATH-only); beads client has working ResolveBdPath() with env var → PATH → known locations precedent.

**Knowledge:** Inconsistency is root cause, not just missing one search location; proven pattern exists (ResolveBdPath); minimal PATH environments are expected context for orchestration (launchd, daemon).

**Next:** Implement pkg/binutil with ResolveBinary(name, envVar, searchPaths) following env var → PATH → known locations order; migrate all opencode and bd resolution to use it; fix shell commands to interpolate resolved path.

**Promote to Decision:** recommend-yes - This establishes the binary resolution pattern for orch-go (architectural constraint: never rely on PATH alone in orchestration context)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Opencode Binary Resolution

**Question:** How should orch-go reliably find the opencode binary when spawning agents, given that ~/.bun/bin is not in the inherited PATH?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** architect agent (og-arch-orch-spawn-find-18jan-e99d)
**Phase:** Complete
**Next Step:** None (ready for implementation via feature-impl)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Inconsistent opencode binary resolution patterns

**Evidence:** The codebase uses three different patterns for finding opencode:

1. **Pattern A - OPENCODE_BIN environment variable:** Used in `pkg/tmux/tmux.go:265-267` and `pkg/opencode/client.go:70-75`
   ```go
   opencodeBin := "opencode"
   if bin := os.Getenv("OPENCODE_BIN"); bin != "" {
       opencodeBin = bin
   }
   ```

2. **Pattern B - Hardcoded in shell commands:** Used in `cmd/orch/spawn_cmd.go:405` and `cmd/orch/doctor.go:511`
   ```go
   cmd := exec.Command("sh", "-c", "ORCH_WORKER=1 opencode serve --port 4096 </dev/null >/dev/null 2>&1 &")
   ```

3. **Pattern C - PATH-only lookup:** Used in `pkg/tmux/tmux.go:427`
   ```go
   cmd := exec.Command("opencode", args...)
   ```

**Source:** Grep results for `opencode.*exec` across *.go files

**Significance:** Pattern B (hardcoded in shell) is the failure point. It relies entirely on PATH lookup with no fallback to known locations or OPENCODE_BIN environment variable. Even if OPENCODE_BIN is set, the shell command doesn't use it.

---

### Finding 2: Successful precedent exists for bd binary resolution

**Evidence:** The beads client has a working resolution pattern in `pkg/beads/client.go:42-81`:

```go
func ResolveBdPath() (string, error) {
    // 1. Try PATH first via exec.LookPath
    path, err := exec.LookPath("bd")
    if err == nil {
        return filepath.Abs(path)
    }
    
    // 2. Check known locations
    home := os.Getenv("HOME")
    for _, searchPath := range bdSearchPaths {
        expanded := strings.Replace(searchPath, "$HOME", home, 1)
        if _, err := os.Stat(expanded); err == nil {
            return expanded, nil
        }
    }
    
    return "", fmt.Errorf("bd executable not found")
}
```

Search paths include: `$HOME/bin`, `$HOME/go/bin`, `$HOME/.bun/bin`, `$HOME/.local/bin`, `/usr/local/bin`, `/opt/homebrew/bin`

**Source:** `pkg/beads/client.go:31-81`

**Significance:** This pattern solves exactly the problem we're facing. It tries PATH first (fast path), then falls back to checking known installation locations. The same approach can be applied to opencode binary resolution.

---

### Finding 3: OpenCode symlink exists at documented location

**Evidence:** 
```bash
$ ls -la ~/.bun/bin/opencode
lrwxr-xr-x  1 dylanconlin  staff  104 Jan  9 08:16 /Users/dylanconlin/.bun/bin/opencode -> /Users/dylanconlin/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode
```

The symlink exists and is valid. `~/.bun/bin` is in the user's PATH:
```bash
$ echo $PATH | tr ':' '\n' | grep -E 'bun|local'
/Users/dylanconlin/.bun/bin
/Users/dylanconlin/.local/bin
/usr/local/bin
```

**Source:** Direct shell commands, CLAUDE.md documentation

**Significance:** The problem isn't that the symlink doesn't exist - it's that processes spawned by orch-go inherit a minimal PATH that doesn't include `~/.bun/bin`. This is documented in CLAUDE.md as a known issue with launchd and other minimal PATH environments.

---

## Synthesis

**Key Insights:**

1. **PATH-only resolution is fundamentally unreliable in orchestration contexts** - Processes spawned by orch-go (especially via launchd daemon or in minimal environments) inherit a restricted PATH that doesn't include user-specific directories like `~/.bun/bin`. Relying solely on PATH means the system breaks in exactly the environments where orchestration is most needed.

2. **Successful pattern exists and is proven** - The `ResolveBdPath()` function demonstrates a working solution: try PATH first (fast), then check known locations (reliable). This pattern is already solving the same problem for the `bd` binary.

3. **Inconsistency creates fragility** - Having three different resolution patterns (OPENCODE_BIN env var, hardcoded shell commands, PATH-only) means some code paths work while others fail. The failure surfaces in critical paths: spawning agents and ensuring OpenCode server is running.

**Answer to Investigation Question:**

orch-go should use a unified binary resolution utility that mirrors the `ResolveBdPath()` pattern: (1) try `exec.LookPath()` first for speed, (2) check known installation locations including `~/.bun/bin`, (3) provide clear error messages listing searched locations when not found. This resolves the current failure while maintaining performance in normal PATH environments. The solution should be extracted to a common utility (`pkg/binutil` or similar) to serve both opencode and bd resolution, enforcing consistency across the codebase.

---

## Structured Uncertainty

**What's tested:**

- ✅ **Symlink exists at ~/.bun/bin/opencode** (verified: `ls -la ~/.bun/bin/opencode` shows valid symlink)
- ✅ **Current shell has ~/.bun/bin in PATH** (verified: `echo $PATH | grep bun` returns path)
- ✅ **bd resolution pattern exists and follows PATH → known locations** (verified: read `pkg/beads/client.go:42-81`)
- ✅ **Multiple opencode resolution patterns exist** (verified: grep found 3 different patterns across codebase)
- ✅ **Hardcoded shell commands exist** (verified: spawn_cmd.go:405 and doctor.go:511 have literal "opencode" string)

**What's untested:**

- ⚠️ **Proposed solution will work in launchd/daemon context** (not spawned agent in minimal PATH to verify resolution works)
- ⚠️ **Performance impact of 6 os.Stat() checks negligible** (assumed based on typical filesystem performance, not benchmarked)
- ⚠️ **Error message with searched paths will help users debug** (design hypothesis, not user-tested)
- ⚠️ **Caching resolved path won't cause issues** (assumed resolution at startup is sufficient, not tested with binary updates during runtime)

**What would change this:**

- **Finding would be wrong if** symlink doesn't actually resolve to valid binary (tested: symlink target exists)
- **Finding would be wrong if** PATH-first order causes problems (unlikely: if binary in PATH, that's user's intentional setup)
- **Recommendation would change if** binary location is truly unpredictable (e.g., installed in random directories) - would need different strategy like config file with path

---

## Design Forks (Decision Navigation)

### Fork 1: Code Organization - Where should binary resolution live?

**Options:**
- A: Create common utility package (`pkg/binutil`)
- B: Duplicate resolution in each package (opencode, tmux)
- C: Only fix opencode package, leave others as-is

**Substrate says:**
- **Principle (DRY/Coherence Over Patches):** Same logic appearing in multiple places creates maintenance burden and divergence over time
- **Model (Spawn Architecture):** Multiple spawn modes (headless, tmux, claude backend) all need to find opencode - common utility enables consistency
- **Evidence:** Finding 1 shows three different patterns already causing fragility

**RECOMMENDATION:** Option A - Create `pkg/binutil` package

**Trade-off accepted:** Slightly more upfront work (new package, migrate both bd and opencode)
**When this would change:** If we only ever need this for one binary (but we already need it for two: bd and opencode)

---

### Fork 2: Search Order - What sequence should binary resolution follow?

**Options:**
- A: PATH first → OPENCODE_BIN env var → known locations (fast path optimization)
- B: OPENCODE_BIN env var → PATH → known locations (explicit override priority)
- C: Known locations first → PATH (predictability over speed)

**Substrate says:**
- **Principle (Pragmatism):** Fast path for common case (binary in PATH) before fallbacks
- **Model (Existing bd pattern):** PATH first, then known locations - proven to work
- **Constraint:** OPENCODE_BIN is documented in CLAUDE.md as the override mechanism

**RECOMMENDATION:** Option B - OPENCODE_BIN env var → PATH → known locations

**Reasoning:** 
1. OPENCODE_BIN is explicitly set in CLAUDE.md setup - should take precedence as user's explicit choice
2. PATH check is still fast when env var not set
3. Known locations are final fallback for minimal PATH environments

**Trade-off accepted:** One extra env var check (~nanoseconds)
**When this would change:** If performance profiling shows env var check is measurable bottleneck (unlikely)

---

### Fork 3: Hardcoded Shell Commands - How to fix spawn_cmd.go:405 and doctor.go:511?

**Options:**
- A: Interpolate resolved path into shell command string
- B: Set OPENCODE_BIN env var, let shell use it with explicit check
- C: Eliminate shell wrapper, use exec.Command directly with resolved path

**Substrate says:**
- **Principle (Coherence):** Eliminate special cases and magic strings
- **Model (Spawn Architecture):** Existing shell wrapper handles backgrounding and redirection
- **Finding 1:** Hardcoded "opencode" in shell string is the direct failure point

**RECOMMENDATION:** Option A - Interpolate resolved path into shell command

**Why:**
1. Minimal change to working backgrounding/redirection logic
2. Explicit and visible - the resolved path is right there in the command
3. No reliance on environment variable passing through shell

**Trade-off accepted:** Slightly longer command strings in logs
**When this would change:** If shell command becomes more complex and needs refactoring anyway

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Unified Binary Resolution with Common Utility** - Create `pkg/binutil` package with `ResolveBinary(name string, searchPaths []string) (string, error)` that follows OPENCODE_BIN env → PATH → known locations order.

**Why this approach:**
- Solves the immediate bug (opencode not found in minimal PATH)
- Eliminates pattern inconsistency (Finding 1) by providing single source of truth
- Reuses proven pattern from bd resolution (Finding 2)
- Makes binary resolution testable and maintainable in one place

**Trade-offs accepted:**
- Small refactor required to migrate existing code (estimated: 30 minutes)
- New package adds one file to codebase (~100 lines)

**Implementation sequence:**
1. **Create `pkg/binutil/binutil.go`** - Extract and generalize the resolution pattern from beads, respecting env var override
2. **Migrate beads client** - Replace `ResolveBdPath()` with `binutil.ResolveBinary("bd", bdSearchPaths)`  
3. **Migrate opencode resolution** - Replace all opencode lookups with `binutil.ResolveBinary("opencode", opencodeSearchPaths)`
4. **Fix shell commands** - Update spawn_cmd.go:405 and doctor.go:511 to interpolate resolved path
5. **Add tests** - Test PATH precedence, env var override, known location fallback, error message clarity

### Alternative Approaches Considered

**Option B: Quick patch - only fix the two failing shell commands**
- **Pros:** Minimal code change, fast to implement
- **Cons:** Doesn't address Finding 1 (pattern inconsistency), technical debt accumulates, next failure will have same root cause
- **When to use instead:** Emergency hotfix when time-critical and proper fix can follow

**Option C: Environment variable propagation only**
- **Pros:** No code changes, configuration-based
- **Cons:** Requires every environment to set OPENCODE_BIN (launchd plist, shell configs, etc.), doesn't help fresh installs, brittle
- **When to use instead:** When binary location is truly unpredictable and can't be in standard paths

**Rationale for recommendation:** Option A eliminates the root cause (inconsistent resolution + PATH-only approach) rather than patching symptoms. The pattern is already proven (Finding 2), and the cost is low relative to ongoing maintenance of three different approaches.

---

### Implementation Details

**File Targets:**

**New files to create:**
- `pkg/binutil/binutil.go` - Common binary resolution utility
  - `ResolveBinary(name string, envVarName string, searchPaths []string) (string, error)`
  - Implements: env var check → PATH lookup → known locations → error with searched paths
- `pkg/binutil/binutil_test.go` - Unit tests for resolution logic

**Files to modify:**
- `pkg/beads/client.go:42-81` - Replace `ResolveBdPath()` with `binutil.ResolveBinary()` call
- `pkg/opencode/client.go:70-75` - Add `ResolveOpencodePath()` function using binutil
- `pkg/tmux/tmux.go:265-267, 293-295, 320-322` - Replace inline checks with `binutil.ResolveBinary()`
- `cmd/orch/spawn_cmd.go:405` - Replace hardcoded "opencode" with resolved path interpolation
- `cmd/orch/doctor.go:511` - Replace hardcoded "opencode" with resolved path interpolation
- `cmd/orch/attach.go:68-70` - Replace inline check with binutil call

**What to implement first:**
1. **Create `pkg/binutil` package** - Foundation for everything else, enables testing in isolation
2. **Add comprehensive tests** - Cover env var override, PATH lookup, known locations fallback, error messages
3. **Migrate one package** - Start with opencode (most critical) to validate pattern works

**Things to watch out for:**
- ⚠️ **Shell escaping:** When interpolating resolved path into shell commands, ensure proper quoting if path contains spaces (unlikely but possible)
- ⚠️ **Symlink handling:** `filepath.Abs()` resolves symlinks by default - document this behavior as it affects OPENCODE_BIN pointing to `opencode-dev`
- ⚠️ **Error message clarity:** Must list all searched locations when binary not found (helps users debug PATH issues)
- ⚠️ **Windows compatibility:** Search paths use `$HOME` - ensure Windows `%USERPROFILE%` handled if Windows support planned
- ⚠️ **Race conditions:** Binary resolution at startup vs runtime - consider calling `ResolveBinary` at init time and caching

**Areas needing further investigation:**
- Whether to cache resolved paths or re-resolve each time (trade-off: performance vs. detecting binary updates/moves)
- Whether other CLIs need resolution (tmux, git, etc.) - if yes, generalize further
- Performance impact of os.Stat() checks on 6 paths - likely negligible but worth noting

**Success criteria:**
- ✅ **Bug reproduction test passes:** Create test that simulates minimal PATH (unset `~/.bun/bin`), verify opencode is still found via known locations
- ✅ **Headless spawn works:** `orch spawn --backend opencode feature-impl "test task"` succeeds without PATH including `~/.bun/bin`
- ✅ **Error message improvement:** When opencode not found, message lists: "opencode not found. Searched: [PATH], [~/.bun/bin/opencode], [other locations]. Ensure opencode is installed or set OPENCODE_BIN."
- ✅ **All resolution patterns unified:** Grep for "opencode" lookups shows consistent use of binutil (no more inline OPENCODE_BIN checks)
- ✅ **Existing functionality preserved:** All existing spawn modes still work (tmux, headless, claude backend)

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:405` - Found hardcoded "opencode" in shell command that fails in minimal PATH
- `cmd/orch/doctor.go:511` - Found second instance of hardcoded "opencode" in shell command
- `pkg/tmux/tmux.go:265-267, 293-295, 320-322` - Found OPENCODE_BIN env var pattern (partial solution)
- `pkg/opencode/client.go:70-75` - Found getOpencodeBin() helper that checks OPENCODE_BIN
- `pkg/beads/client.go:31-81` - Found ResolveBdPath() as proven pattern for binary resolution
- `CLAUDE.md:1-100` - Reviewed documentation of PATH issues and opencode setup

**Commands Run:**
```bash
# Verify opencode symlink exists and target is valid
ls -la ~/.bun/bin/opencode

# Check if ~/.bun/bin is in current PATH
echo $PATH | tr ':' '\n' | grep -E 'bun|local'

# Find all locations where opencode binary is executed
grep -r 'opencode.*exec\|exec.*opencode' --include="*.go"

# Find all uses of exec.LookPath to understand existing resolution patterns
grep -r 'exec\.LookPath\|LookPath' --include="*.go"
```

**External Documentation:**
- Go `exec.LookPath` docs - Standard library function for finding executables in PATH

**Related Artifacts:**
- **SPAWN_CONTEXT.md** - Original bug report: "exec: 'opencode': executable file not found in $PATH"
- **CLAUDE.md** - Documents setup with symlink at ~/.bun/bin/opencode
- **Prior constraint:** "CLI PATH Fix (via ~/.bun/bin symlinks)" section documents known PATH limitations

---

## Investigation History

**2026-01-18 (start):** Investigation started
- Initial question: How should orch-go reliably find the opencode binary when spawning agents?
- Context: Headless spawns failing with "exec: 'opencode': executable file not found in $PATH" despite valid symlink at ~/.bun/bin/opencode

**2026-01-18 (+30 min):** Found three inconsistent resolution patterns
- Pattern A: OPENCODE_BIN env var check (partial solution)
- Pattern B: Hardcoded "opencode" in shell commands (failure point)  
- Pattern C: PATH-only exec.Command (fails in minimal PATH)
- Key insight: Inconsistency is the root cause, not just missing one location

**2026-01-18 (+60 min):** Discovered proven precedent in beads client
- ResolveBdPath() implements exact pattern we need: PATH → known locations
- Same search paths include ~/.bun/bin already
- Evidence that approach works in production

**2026-01-18 (+90 min):** Design synthesis complete
- Recommendation: Create pkg/binutil for unified resolution
- Three design forks navigated with substrate consultation
- Implementation plan ready with file targets and acceptance criteria
