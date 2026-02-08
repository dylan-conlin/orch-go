<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented doc debt tracking for new CLI commands via `~/.orch/doc-debt.json` with `orch doctor --docs` integration.

**Evidence:** Created DocDebt struct in userconfig, detection logic in complete_cmd.go, and surfacing via orch doctor.

**Knowledge:** Passive tracking is sufficient - aggressive blocking would interrupt workflows. Pattern follows config drift detection precedent.

**Next:** Use `orch doctor --docs` to surface undocumented commands; update docs manually or spawn doc-update agent.

**Promote to Decision:** recommend-no (implementation detail, not architectural)

---

# Investigation: Drift Prevention Auto Track CLI

**Question:** How can we automatically track new CLI commands and surface when documentation drifts out of sync?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** Agent og-feat-drift-prevention-auto-08jan-c18c
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing CLI Detection Already Present

**Evidence:** `detectNewCLICommands()` in `cmd/orch/complete_cmd.go:758-826` already:
- Scans last 5 commits for added files in `cmd/orch/`
- Checks for `cobra.Command` and `rootCmd.AddCommand` patterns
- Displays advisory message suggesting doc updates

**Source:** `cmd/orch/complete_cmd.go:605-620` (display), `cmd/orch/complete_cmd.go:758-826` (detection)

**Significance:** Foundation exists - we need to add persistence and surfacing, not rebuild detection logic.

---

### Finding 2: Config Drift Detection Pattern Exists

**Evidence:** `orch doctor --config` already implements drift detection:
- `ConfigDrift` struct stores field/expected/actual
- `ConfigDriftReport` tracks healthy status and drifts
- Pattern can be reused for doc debt tracking

**Source:** `cmd/orch/doctor.go:885-1020`

**Significance:** Establishes design pattern: separate check flag, report struct, surfacing via doctor.

---

### Finding 3: ~/.orch/config.yaml is Central Config Location

**Evidence:** User config stored in `~/.orch/config.yaml` with:
- YAML format
- Helper methods for defaults
- Load/Save functions in `pkg/userconfig/userconfig.go`

**Source:** `pkg/userconfig/userconfig.go:1-377`

**Significance:** Doc debt should be stored in `~/.orch/doc-debt.json` alongside config to follow existing patterns.

---

## Synthesis

**Key Insights:**

1. **Reuse existing detection** - `detectNewCLICommands()` already does the hard work of finding new commands.

2. **Follow config drift pattern** - `orch doctor --config` is the model: separate flag, dedicated check, clear report.

3. **JSON for debt tracking** - config.yaml is for settings; doc-debt.json is for state tracking (like gap-tracker.json).

**Answer to Investigation Question:**

The solution involves:
1. Persist detected new commands to `~/.orch/doc-debt.json` (command name, date added, documented: bool)
2. Add `orch doctor --docs` to surface undocumented commands
3. Provide mechanism to mark commands as documented
4. Weekly digest is implicit (run `orch doctor --docs` anytime)

Blocking completion is rejected as too aggressive - advisory approach is sufficient.

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing detection logic works (verified: read detectNewCLICommands implementation)
- ✅ Config drift pattern is reusable (verified: reviewed doctor --config implementation)
- ✅ ~/.orch directory exists and is writable (verified: ls -la ~/.orch)

**What's untested:**

- ⚠️ Performance impact of checking doc-debt.json on every completion (likely negligible)
- ⚠️ Edge cases for command naming (e.g., subcommands in same file)

**What would change this:**

- If orch complete becomes too slow, consider debouncing debt tracking
- If users need finer control, add per-command ignore list

---

## Implementation Recommendations

### Recommended Approach ⭐

**Passive Doc Debt Tracking with Doctor Integration** - Track new commands in JSON, surface via `orch doctor --docs`, allow manual marking as documented.

**Why this approach:**
- Non-blocking: doesn't interrupt workflows
- Follows existing patterns: mirrors config drift detection
- Low overhead: simple JSON state file
- Easy to extend: can add weekly digest later via daemon

**Trade-offs accepted:**
- No enforcement: relies on human discipline to update docs
- Manual marking: no automatic detection of doc updates

**Implementation sequence:**
1. Add DocDebt types to pkg/userconfig (JSON structs)
2. Modify complete_cmd.go to persist new commands to doc-debt.json
3. Add `orch doctor --docs` flag and check function
4. Add `orch docs mark-documented <command>` subcommand

### Alternative Approaches Considered

**Option B: Block completion if undocumented**
- **Pros:** Enforces documentation discipline
- **Cons:** Too aggressive, interrupts legitimate workflows
- **When to use instead:** If doc drift becomes critical problem

**Option C: Integrate with kb reflect**
- **Pros:** Unified drift detection
- **Cons:** kb reflect is for CLAUDE.md constraints, not CLI docs
- **When to use instead:** If we unify all drift detection

---

## References

**Files Examined:**
- cmd/orch/complete_cmd.go:605-826 - CLI command detection and advisory
- cmd/orch/doctor.go:1-1103 - Config drift pattern and doctor infrastructure
- pkg/userconfig/userconfig.go:1-377 - Config management patterns

**Related Artifacts:**
- **Beads Issue:** orch-go-edcy - This implementation task
- **Prior Decision:** kb reflect Command Interface - Shows drift detection precedent
