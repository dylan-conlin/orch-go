<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** main.go contains 10 command groups with shared utilities - can be split into 6 domain files plus 2 utility files without circular imports.

**Evidence:** Analyzed 4964 lines, mapped 26 command definitions, 85+ functions, and identified 4 categories of shared state/helpers.

**Knowledge:** The existing pattern (daemon.go, review.go, focus.go) shows the correct approach: keep commands + run* functions together in domain files, share utilities via shared.go.

**Next:** Implement Phase 1 (spawn.go, status.go, shared.go) to validate approach, then continue with remaining domains.

---

# Investigation: Map Main Go Command Dependencies

**Question:** How should the 4964-line cmd/orch/main.go be split into separate files/packages to improve maintainability while avoiding circular imports?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Self-Review

- [x] Real test performed (not code review) - Ran `go build`, `wc -l`, `grep` commands
- [x] Conclusion from evidence (not speculation) - Based on actual code analysis
- [x] Question answered - Clear split strategy provided
- [x] File complete - All sections filled
- [x] D.E.K.N. filled - Summary section complete
- [x] NOT DONE claims verified - N/A (investigation only)

**Self-Review Status:** PASSED

---

## Findings

### Finding 1: Initial Scope - main.go is 4964 lines, cmd/orch/ is 26,176 lines total

**Evidence:** 
- main.go: 4964 lines (god object containing most command implementations)
- cmd/orch/ total: 26,176 lines across ~45 files
- Already separated: serve.go (2921), review.go (1079), handoff.go (898), kb.go (745), changelog.go (769), daemon.go (559), focus.go (340+), init.go (448), learn.go (455), patterns.go (591), session.go (509+), servers.go (350+), swarm.go (667), tokens.go (352), reconcile.go (370+)

**Source:** `wc -l cmd/orch/*.go | sort -n`

**Significance:** About 15 command files have already been extracted following a consistent pattern. main.go still contains the core commands (spawn, status, complete, send, tail, question, abandon, work, clean, account, port, retries, version).

---

### Finding 2: Commands in main.go can be grouped into 6 logical domains

**Evidence:** Analysis of the 26 cobra.Command definitions in main.go:

| Domain | Commands | Lines Est. | Dependencies |
|--------|----------|------------|--------------|
| **Spawn** | spawn, work | ~750 | beads, spawn pkg, skills, events, opencode, tmux, model |
| **Agent Ops** | send, tail, question, abandon | ~550 | opencode, tmux, events, beads, verify |
| **Status** | status | ~500 | opencode, tmux, beads, verify, account, usage |
| **Complete** | complete | ~450 | verify, beads, events, spawn, changelog detection |
| **Clean** | clean | ~300 | opencode, tmux, events, verify |
| **Account** | account (list/switch/add/remove), usage | ~250 | account pkg, usage pkg |
| **Port** | port (allocate/list/release/tmuxinator) | ~200 | port pkg, tmux pkg |
| **Utility** | version, retries | ~150 | verify pkg |

**Source:** `grep -n "var.*Cmd = &cobra.Command" cmd/orch/main.go`

**Significance:** Clear domain boundaries exist. Each group has cohesive dependencies and could be extracted following the existing pattern.

---

### Finding 3: Shared state and utilities are concentrated in 4 categories

**Evidence:** 

**1. Global flags (lines 35-183):**
```go
var serverURL string  // Used by 8+ commands
var spawnSkill, spawnIssue, spawnPhases, ... // 20+ spawn flags
var statusJSON, statusAll, statusProject // status flags
var completeForce, completeReason, ... // complete flags
```

**2. Shared utility functions (used across domains):**
```go
extractBeadsIDFromTitle(title string) string           // Used in: status, send, tail, question, abandon, complete, focus, handoff, doctor
extractSkillFromTitle(title string) string             // Used in: status
extractBeadsIDFromWindowName(name string) string       // Used in: status
extractSkillFromWindowName(name string) string         // Used in: status
extractProjectFromBeadsID(beadsID string) string       // Used in: status
findWorkspaceByBeadsID(projectDir, beadsID string)     // Used in: tail, question, abandon, complete, status
resolveSessionID(serverURL, identifier string) string  // Used in: send
truncate(s string, maxLen int) string                  // Used in: spawn, status
formatDuration(d time.Duration) string                 // Used in: status (note: defined elsewhere, reused)
```

**3. Shared types:**
```go
type SwarmStatus struct { ... }      // Used by status
type AccountUsage struct { ... }     // Used by status  
type AgentInfo struct { ... }        // Used by status
type StatusOutput struct { ... }     // Used by status
type GapCheckResult struct { ... }   // Used by spawn
type CleanableWorkspace struct { ... } // Used by clean
type headlessSpawnResult struct { ... } // Used by spawn
```

**4. Shared initialization (multiple init() functions):**
- rootCmd.AddCommand() calls in main init()
- Flag registration in command-specific init()s

**Source:** `grep -n "^func " cmd/orch/main.go`, code analysis of function call sites

**Significance:** The shared utilities are the main challenge for splitting. They need to either:
1. Live in a shared.go file (same package, no import issues)
2. Move to pkg/ (requires exporting, more refactoring)

---

### Finding 4: Existing extracted files follow a consistent pattern

**Evidence:** Examining daemon.go, focus.go, review.go patterns:

```go
// daemon.go - 559 lines
package main

import (...)

var daemonCmd = &cobra.Command{...}
var daemonRunCmd = &cobra.Command{...}
// ... more subcommands

var (
    // Daemon-specific flags
    daemonDelay int
    daemonDryRun bool
    ...
)

func init() {
    daemonCmd.AddCommand(daemonRunCmd)
    // flag registration
}

func runDaemonLoop() error {...}
func runDaemonOnce() error {...}
// ... more run* functions
```

**Pattern observed:**
1. File contains cobra Command definitions + their init() + their run* implementations
2. Domain-specific flags are defined in the file
3. Root command registration happens in main.go init()
4. Utility functions that serve ONLY that domain stay in the file
5. Cross-domain utilities could be extracted to shared.go

**Source:** `head -150 cmd/orch/daemon.go`, `head -100 cmd/orch/focus.go`

**Significance:** The pattern is already established and working. Following it ensures consistency.

---

### Finding 5: Import dependencies don't create circular import risk

**Evidence:** All commands import from pkg/ packages, never from each other:
- `github.com/dylan-conlin/orch-go/pkg/account`
- `github.com/dylan-conlin/orch-go/pkg/beads`
- `github.com/dylan-conlin/orch-go/pkg/events`
- `github.com/dylan-conlin/orch-go/pkg/model`
- `github.com/dylan-conlin/orch-go/pkg/opencode`
- `github.com/dylan-conlin/orch-go/pkg/port`
- `github.com/dylan-conlin/orch-go/pkg/question`
- `github.com/dylan-conlin/orch-go/pkg/session`
- `github.com/dylan-conlin/orch-go/pkg/skills`
- `github.com/dylan-conlin/orch-go/pkg/spawn`
- `github.com/dylan-conlin/orch-go/pkg/tmux`
- `github.com/dylan-conlin/orch-go/pkg/usage`
- `github.com/dylan-conlin/orch-go/pkg/verify`
- `github.com/spf13/cobra`

Since all cmd/orch/*.go files are in `package main`, they can freely call each other's functions without import statements.

**Source:** Lines 4-33 of main.go, cross-referencing with other command files

**Significance:** Splitting main.go within `package main` (same directory) avoids all circular import risks. The only constraint is that shared utilities must be defined before use (compilation order), but Go handles this automatically within a package.

---

## Synthesis

**Key Insights:**

1. **Clean domain boundaries exist** - The 10 command groups have natural cohesion and can be split without fragmenting related logic.

2. **Shared utilities are the key challenge** - 9 utility functions are used across multiple domains and must be extracted to a shared.go file.

3. **Existing pattern works well** - The daemon.go, focus.go, review.go examples show the proven approach: command + flags + init + run functions in one file.

4. **No circular import risk** - All files stay in `package main`, so cross-file function calls work without imports.

**Answer to Investigation Question:**

Split main.go into 8 files following the existing pattern:
1. `spawn_cmd.go` - spawn, work commands + spawn helpers (~750 lines)
2. `agent_ops.go` - send, tail, question, abandon commands (~550 lines)
3. `status_cmd.go` - status command + display helpers (~500 lines)
4. `complete_cmd.go` - complete command + verification (~450 lines)
5. `clean_cmd.go` - clean command (~300 lines)
6. `account_cmd.go` - account, usage commands (~250 lines)
7. `port_cmd.go` - port commands (~200 lines)
8. `shared.go` - extractBeadsIDFromTitle, findWorkspaceByBeadsID, truncate, etc. (~200 lines)

Keep in main.go:
- main() function
- rootCmd definition
- Global serverURL flag
- Version command (small)
- Retries command (small)
- Root init() registering all commands

This reduces main.go from ~4964 lines to ~400 lines.

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing extracted files (daemon.go, focus.go) compile and work (verified: project builds)
- ✅ All imports are from pkg/ packages, not cross-cmd files (verified: analyzed import statements)
- ✅ Shared utilities have clear call sites (verified: grep for each function name)

**What's untested:**

- ⚠️ Actual line counts after extraction (estimates based on code analysis)
- ⚠️ Build time impact of more files (likely negligible but not measured)
- ⚠️ IDE performance with 25+ files in cmd/orch/ (should be fine but untested)

**What would change this:**

- Finding would be wrong if a utility function has hidden dependencies not identified
- Finding would be wrong if test files have cross-dependencies that complicate the split

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Incremental domain extraction** - Extract one domain at a time, test builds after each, starting with the most self-contained domains.

**Why this approach:**
- Each extraction is independently testable
- Failures are isolated and easy to fix
- Progress is visible and reversible
- Follows the proven pattern from daemon.go extraction

**Trade-offs accepted:**
- Takes longer than a single big refactor
- Multiple PRs or commits instead of one
- Why acceptable: Lower risk, easier review, can pause if issues arise

**Implementation sequence:**
1. **Phase 1: Create shared.go** - Extract 9 shared utility functions first. This establishes the foundation.
2. **Phase 2: Extract spawn_cmd.go** - Largest domain (~750 lines), high value, well-bounded
3. **Phase 3: Extract status_cmd.go** - Complex but self-contained
4. **Phase 4: Extract remaining domains** - agent_ops, complete_cmd, clean_cmd, account_cmd, port_cmd

### Alternative Approaches Considered

**Option B: Single big refactor**
- **Pros:** Done in one PR, no intermediate states
- **Cons:** High risk, hard to review, hard to bisect if bugs introduced
- **When to use instead:** If the team is confident and wants speed over safety

**Option C: Move to cmd/orch/spawn/, cmd/orch/status/ subpackages**
- **Pros:** Stronger encapsulation, clearer boundaries
- **Cons:** Would require exporting functions, changing import paths, major restructure
- **When to use instead:** If the project needs stricter modularity later

**Rationale for recommendation:** Incremental extraction is lower risk and can be paused at any point. The existing pattern (daemon.go, focus.go) proves the approach works.

---

### Implementation Details

**What to implement first:**
1. Create `shared.go` with these functions:
   - `extractBeadsIDFromTitle(title string) string`
   - `extractSkillFromTitle(title string) string`
   - `extractBeadsIDFromWindowName(name string) string`
   - `extractSkillFromWindowName(name string) string`
   - `extractProjectFromBeadsID(beadsID string) string`
   - `findWorkspaceByBeadsID(projectDir, beadsID string) (string, string)`
   - `resolveSessionID(serverURL, identifier string) (string, error)`
   - `truncate(s string, maxLen int) string`
   - `findTmuxWindowByIdentifier(identifier string) (*tmux.WindowInfo, error)`

2. Extract spawn_cmd.go:
   - Move: spawnCmd, workCmd, and all spawn* flags
   - Move: spawn init() function
   - Move: runSpawnWithSkill, runSpawnInline, runSpawnHeadless, runSpawnTmux
   - Move: determineBeadsID, createBeadsIssue, determineSpawnTier
   - Move: checkConcurrencyLimit, checkAndAutoSwitchAccount
   - Move: headlessSpawnResult type, startHeadlessSession, formatSessionTitle
   - Move: gap analysis helpers (checkGapGating, recordGapForLearning, etc.)
   - Move: InferSkillFromIssueType, inferSkillFromBeadsIssue, runWork

**Things to watch out for:**
- ⚠️ Multiple init() functions: Go supports multiple init() per package, but order is undefined. Flag registration order shouldn't matter.
- ⚠️ Variable shadowing: When moving code, ensure local variables don't shadow package-level ones
- ⚠️ Deleted function references: After moving a function, ensure all call sites still compile

**Areas needing further investigation:**
- Test file dependencies (main_test.go may need adjustment)
- Whether any functions should move to pkg/ instead (for external use)

**Success criteria:**
- ✅ `go build ./cmd/orch` succeeds after each extraction phase
- ✅ `go test ./cmd/orch` passes after each extraction phase
- ✅ main.go reduced to <500 lines
- ✅ Each new file is <1000 lines
- ✅ No function appears in multiple files

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Primary target (4964 lines)
- `cmd/orch/daemon.go` - Example of extracted command (559 lines)
- `cmd/orch/focus.go` - Example of extracted command (340+ lines)
- `cmd/orch/review.go` - Example of extracted command (1079 lines)

**Commands Run:**
```bash
# Count lines in all Go files
wc -l cmd/orch/*.go | sort -n

# Find all command definitions
grep -n "var.*Cmd = &cobra.Command" cmd/orch/main.go

# Find all function definitions
grep -n "^func " cmd/orch/main.go

# Find shared utility usage
grep -rn "extractBeadsIDFromTitle\|findWorkspaceByBeadsID" cmd/orch/*.go
```

**Related Artifacts:**
- None directly related; this is a new investigation

---

## Investigation History

**2026-01-03 17:00:** Investigation started
- Initial question: How to split 4964-line main.go into maintainable files
- Context: God object growing, hard to navigate and maintain

**2026-01-03 17:30:** Major findings complete
- Identified 10 command domains
- Mapped shared utilities
- Confirmed no circular import risk
- Validated against existing extracted files

**2026-01-03 18:00:** Investigation completed
- Status: Complete
- Key outcome: Detailed split strategy with 8 new files, incremental implementation plan
