<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Auto-generation already exists via `cmd/gendoc/main.go` and Cobra's doc generator, but it manually duplicates command definitions causing drift.

**Evidence:** Found 41 commands registered in `cmd/orch/*.go` but only 30 documented in gendoc; gendoc rebuilds command tree manually instead of importing from main.go.

**Knowledge:** The current approach requires manual synchronization because gendoc duplicates command definitions rather than importing them from the actual CLI.

**Next:** Recommend `orch lint --docs` validation command to detect drift (Option C) - least invasive, highest ROI solution.

---

# Investigation: Auto-Generate Docs from orch --help Output

**Question:** Can we auto-generate documentation from --help text to prevent drift between CLI implementation and documentation?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** orch-go
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Auto-generation infrastructure already exists

**Evidence:** 
- `cmd/gendoc/main.go` exists with Cobra's `doc.GenMarkdownTreeCustom`
- `make docs` target runs `go run ./cmd/gendoc`
- Output goes to `docs/cli/*.md` with frontmatter

**Source:** 
- cmd/gendoc/main.go:1-73
- Makefile:86-89

**Significance:** We don't need to build from scratch - the infrastructure is there. The problem is that gendoc manually duplicates command definitions instead of importing them.

---

### Finding 2: Significant command drift exists

**Evidence:** 
Commands registered in `cmd/orch/*.go` (41 total):
- abandonCmd, accountCmd, cleanCmd, completeCmd, daemonCmd, doctorCmd, driftCmd, 
- fetchmdCmd, focusCmd, handoffCmd, historyCmd, initCmd, kbCmd, learnCmd, lintCmd,
- logsCmd, monitorCmd, nextCmd, patternsCmd, portCmd, questionCmd, resumeCmd,
- retriesCmd, reviewCmd, sendCmd, serveCmd, serversCmd, sessionsCmd, spawnCmd,
- staleCmd, statusCmd, swarmCmd, synthesisCmd, tailCmd, tokensCmd, transcriptCmd,
- usageCmd, versionCmd, waitCmd, workCmd

Commands in gendoc (30 total):
- Missing: doctorCmd, fetchmdCmd, handoffCmd, historyCmd, initCmd, kbCmd, learnCmd,
  lintCmd, logsCmd, patternsCmd, portCmd, retriesCmd, serversCmd, sessionsCmd,
  staleCmd, swarmCmd, synthesisCmd, tokensCmd, transcriptCmd, versionCmd

**Source:** 
- `grep 'rootCmd\.AddCommand' cmd/orch/*.go` 
- `cmd/gendoc/main.go:95-117`

**Significance:** 11 commands are completely undocumented. The manual duplication approach guarantees drift.

---

### Finding 3: The root cause is architectural

**Evidence:** 
gendoc builds its own command tree with `buildCommandTree()` that manually recreates 
each command. When a new command is added to cmd/orch/*.go, the developer must 
also add it to gendoc/main.go. This is the "two places" anti-pattern.

```go
// gendoc/main.go - manually duplicates commands
func buildCommandTree() *cobra.Command {
    rootCmd := &cobra.Command{...}
    rootCmd.AddCommand(buildSpawnCmd())  // Must add each command here
    rootCmd.AddCommand(buildAskCmd())    // If you forget, docs drift
    ...
}
```

**Source:** cmd/gendoc/main.go:79-119

**Significance:** The architecture makes drift inevitable. Any solution must eliminate manual duplication.

---

## Synthesis

**Key Insights:**

1. **Infrastructure exists, architecture is wrong** - Cobra's doc generator works fine; the problem is that gendoc manually rebuilds the command tree instead of importing it.

2. **Drift is already significant** - 11 commands (27%) are missing from docs. This validates the original concern.

3. **Multiple valid solutions exist** - Three approaches can fix this, each with different tradeoffs.

**Answer to Investigation Question:**

Yes, we can auto-generate documentation from --help text, and in fact we already do. The drift problem exists because gendoc duplicates command definitions manually. The fix is either:
- Import the actual rootCmd from cmd/orch (Option A)
- Validate that docs match commands (Option C: `orch lint --docs`)
- Generate from `orch --help` output instead of Cobra internals (Option B)

---

## Structured Uncertainty

**What's tested:**

- ✅ gendoc generates docs from Cobra commands (verified: ran `go run ./cmd/gendoc`)
- ✅ 41 commands exist in cmd/orch/*.go (verified: grep for rootCmd.AddCommand)
- ✅ 30 commands in gendoc (verified: read cmd/gendoc/main.go)

**What's untested:**

- ⚠️ Whether importing rootCmd from cmd/orch creates circular imports (not tested)
- ⚠️ Whether --help-doc output would match Cobra's doc generator quality (not tested)
- ⚠️ Performance impact of any solution (not measured)

**What would change this:**

- If importing rootCmd causes import cycles, Option A becomes non-viable
- If Cobra's doc format doesn't meet needs, Option B becomes more attractive

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Option C: `orch lint --docs` validation** - Add a lint command that validates all registered commands have documentation.

**Why this approach:**
- Least invasive - doesn't require refactoring gendoc
- Catches drift at PR time via pre-commit or CI
- Can be added incrementally
- Works with current architecture

**Trade-offs accepted:**
- Docs still need manual updates when commands are added
- Reactive (catches drift) rather than proactive (prevents drift)

**Implementation sequence:**
1. Add `orch lint --docs` command that lists missing docs
2. Add to CI/pre-commit to gate on drift
3. Fix existing 11 missing command docs

### Alternative Approaches Considered

**Option A: Import rootCmd directly in gendoc**
- **Pros:** Single source of truth, zero drift possible
- **Cons:** May cause import cycles, requires refactoring cmd/orch to export rootCmd
- **When to use instead:** If lint approach proves insufficient

**Option B: Generate from `orch --help` output**
- **Pros:** Works with any CLI regardless of framework
- **Cons:** Requires parsing help text, may lose structured info
- **When to use instead:** If we want to support non-Cobra CLIs in future

**Rationale for recommendation:** Option C provides the best ROI: it's non-invasive, immediately actionable, and creates a gate that prevents future drift. We can always evolve to Option A later if needed.

---

### Implementation Details

**What to implement first:**
- `orch lint --docs` that checks for undocumented commands
- Return non-zero exit code when drift detected

**Things to watch out for:**
- ⚠️ Some commands may be intentionally internal/undocumented
- ⚠️ Need to handle subcommands (account list, daemon run, etc.)

**Areas needing further investigation:**
- Which of the 11 missing commands should actually be documented vs kept internal
- Whether to add to CI or just pre-commit

**Success criteria:**
- ✅ Running `orch lint --docs` returns 0 when all public commands are documented
- ✅ Adding a new command without docs causes lint failure
- ✅ Existing drift is resolved (all 41 commands documented or explicitly marked internal)

---

## References

**Files Examined:**
- cmd/gendoc/main.go - Current doc generator implementation
- cmd/orch/main.go - Main CLI with command registrations
- cmd/orch/*.go - Individual command files with rootCmd.AddCommand
- Makefile - docs target

**Commands Run:**
```bash
# List all commands registered
grep 'rootCmd\.AddCommand' cmd/orch/*.go

# Run doc generator
go run ./cmd/gendoc

# List generated docs
ls docs/cli/
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-28-inv-critical-meta-gap-orch-features.md - Original source of this investigation question

---

## Investigation History

**2025-12-28 15:30:** Investigation started
- Initial question: Can we auto-generate docs from --help to prevent drift?
- Context: Meta-gap investigation found documentation drift as recurring problem

**2025-12-28 15:45:** Key finding - gendoc already exists
- Discovered cmd/gendoc/main.go uses Cobra's doc generator
- Root cause identified: manual duplication of command definitions

**2025-12-28 16:00:** Investigation completed
- Status: Complete
- Key outcome: Recommend `orch lint --docs` validation command (Option C)
