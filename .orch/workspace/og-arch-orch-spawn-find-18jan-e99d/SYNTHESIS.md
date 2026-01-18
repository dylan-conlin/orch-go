# Session Synthesis

**Agent:** og-arch-orch-spawn-find-18jan-e99d
**Issue:** orch-go-dc8g7
**Duration:** 2026-01-18 (start) → 2026-01-18 (complete)
**Outcome:** success

---

## TLDR

Designed unified binary resolution strategy for orch-go to fix opencode not found in minimal PATH; recommended creating `pkg/binutil` following env var → PATH → known locations pattern.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-18-design-opencode-binary-resolution.md` - Complete design investigation with 3 decision forks navigated, implementation recommendations, and file targets

### Files Modified
- None (design phase only - implementation will be separate agent)

### Commits
- Investigation file committed (pending)

---

## Evidence (What Was Observed)

### Pattern Inconsistency Found
- **Pattern A (env var):** `pkg/tmux/tmux.go:265-267`, `pkg/opencode/client.go:70-75` - Checks `OPENCODE_BIN` environment variable
- **Pattern B (hardcoded):** `cmd/orch/spawn_cmd.go:405`, `cmd/orch/doctor.go:511` - Hardcodes "opencode" in shell commands (failure point)
- **Pattern C (PATH-only):** `pkg/tmux/tmux.go:427` - Uses `exec.Command("opencode")` with no fallback
- **Source:** Grep results from `grep -r 'opencode.*exec\|exec.*opencode' --include="*.go"`

### Proven Precedent Exists
- `pkg/beads/client.go:42-81` implements `ResolveBdPath()` with exact pattern needed:
  1. Try `exec.LookPath("bd")` for PATH lookup
  2. Fall back to known locations: `$HOME/bin`, `$HOME/go/bin`, `$HOME/.bun/bin`, `$HOME/.local/bin`, `/usr/local/bin`, `/opt/homebrew/bin`
  3. Return clear error listing searched locations
- **Source:** Direct code reading of beads client implementation

### Symlink Verified
```bash
$ ls -la ~/.bun/bin/opencode
lrwxr-xr-x  1 dylanconlin  staff  104 Jan  9 08:16 /Users/dylanconlin/.bun/bin/opencode -> /Users/dylanconlin/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode
```
- Symlink exists and target is valid
- `~/.bun/bin` is in user's PATH but not in minimal PATH inherited by spawned processes

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-18-design-opencode-binary-resolution.md` - Design investigation documenting problem, findings, decision forks, and recommendations

### Decisions Made

**Fork 1: Code Organization**
- **Decision:** Create `pkg/binutil` package for common binary resolution
- **Rationale:** Coherence Over Patches principle - same logic in multiple places creates maintenance burden; already need it for both `bd` and `opencode`
- **Trade-off:** Slight upfront refactor cost vs. long-term consistency

**Fork 2: Search Order**  
- **Decision:** OPENCODE_BIN env var → PATH → known locations
- **Rationale:** OPENCODE_BIN is documented in CLAUDE.md as explicit override mechanism, should take precedence; PATH check still fast; known locations are final fallback
- **Trade-off:** One extra env var check (negligible performance cost)

**Fork 3: Shell Command Fix**
- **Decision:** Interpolate resolved path into shell command string
- **Rationale:** Minimal change to working backgrounding/redirection logic; explicit and visible; no environment variable propagation complexity
- **Trade-off:** Slightly longer command strings in logs

### Constraints Discovered
- **Orchestration contexts have minimal PATH** - launchd, daemon, and spawned agents inherit restricted PATH without user-specific directories
- **Pattern inconsistency is fragility** - Three different resolution approaches mean some code paths work while others fail
- **Never rely on PATH alone in orchestration** - Established as architectural constraint for orch-go

### Substrate Consultation
- **Principle:** Coherence Over Patches - 5+ fixes to same area means redesign not another patch
- **Principle:** DRY - Same logic in multiple places creates divergence
- **Model:** Spawn Architecture - Multiple spawn modes (headless, tmux, claude backend) all need to find opencode
- **Precedent:** bd resolution already solves this problem with proven pattern

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Required

**Issue:** Implement unified binary resolution via pkg/binutil
**Skill:** feature-impl
**Context:**
```
Design complete in .kb/investigations/2026-01-18-design-opencode-binary-resolution.md.
Create pkg/binutil package with ResolveBinary(name, envVar, searchPaths) following
env var → PATH → known locations order. Migrate all opencode and bd resolution to use it.
Fix spawn_cmd.go:405 and doctor.go:511 to interpolate resolved path.

Success criteria:
- Headless spawn works without ~/.bun/bin in PATH
- Error message lists searched locations when not found
- All resolution patterns unified (grep verification)

File targets: pkg/binutil/binutil.go (new), pkg/binutil/binutil_test.go (new),
plus 6 files to migrate (see investigation for details).
```

### Promotion Recommendation

**Promote to Decision:** Yes - recommend creating `.kb/decisions/YYYY-MM-DD-binary-resolution-pattern.md`

**Rationale:** This establishes an architectural constraint for orch-go: "Never rely on PATH alone in orchestration contexts." The pattern (env var → PATH → known locations) should be applied to any future binary dependencies (tmux, git, etc.). This is foundational infrastructure that affects all spawn modes.

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should we cache resolved binary paths or re-resolve each time? (Trade-off: performance vs. detecting binary updates/moves)
- Do other CLIs need the same resolution pattern? (tmux, git, bd, kb, orch itself for spawned agents)
- What's the performance impact of 6 os.Stat() checks on known locations? (Likely negligible but not benchmarked)
- Should Windows be supported? (Search paths use $HOME, would need %USERPROFILE% handling)

**Areas worth exploring further:**
- Generalize beyond binary resolution to any external dependency discovery
- Consider config file with custom search paths for enterprise environments with non-standard installations
- Integration with package managers (brew, apt, etc.) to auto-detect installation locations

**What remains unclear:**
- Whether symlink resolution behavior (filepath.Abs resolves symlinks) will affect OPENCODE_BIN pointing to opencode-dev
- Whether runtime resolution (vs. startup caching) has observable performance impact in high-frequency spawn scenarios

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-arch-orch-spawn-find-18jan-e99d/`
**Investigation:** `.kb/investigations/2026-01-18-design-opencode-binary-resolution.md`
**Beads:** `bd show orch-go-dc8g7`
