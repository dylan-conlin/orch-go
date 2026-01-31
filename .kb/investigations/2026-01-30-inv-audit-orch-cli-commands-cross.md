<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Most orch CLI commands use `os.Getwd()` without cross-project support; only 7/30+ commands have `--workdir` or `--cross-project` flags.

**Evidence:** Code audit of cmd/orch/*.go found 22 commands using hardcoded `os.Getwd()` vs 7 with explicit cross-project flags (daemon, spawn, complete, work, abandon, status, serve).

**Knowledge:** The constraint kb-d29e8a ("Cross-project orchestration must work") is violated by most commands. Critical gaps: `review`, `frontier`, `tail`, `question`, `clean`, `resume`, `attach`.

**Next:** Implement `--workdir` flag on P0/P1 commands (review, frontier, tail, question, clean, resume, attach) following the pattern from complete_cmd.go.

**Authority:** architectural - Cross-project support spans multiple commands and requires consistent UX pattern across the CLI.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Audit Orch CLI Commands for Cross-Project Support

**Question:** Which orch CLI commands support cross-project operations via --workdir or --project flags, and which have gaps?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Worker agent (orch-go-21096)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A
**Related-Constraint:** kb-d29e8a (Cross-project orchestration must work)

---

## Findings

### Finding 1: Seven commands have explicit cross-project support

**Evidence:** Commands with `--workdir`, `--project`, or `--cross-project` flags:

| Command | Flag | Implementation |
|---------|------|----------------|
| `daemon` | `--cross-project` | Polls all kb-registered projects (daemon.go:160-169) |
| `spawn` | `--workdir` | Spawns in different project directory (spawn_cmd.go:196) |
| `complete` | `--workdir` | Cross-project completion with auto-detection (complete_cmd.go:145) |
| `work` | `--workdir` | Cross-project work command (spawn_cmd.go:243) |
| `abandon` | `--workdir` | Cross-project abandonment (abandon_cmd.go:61) |
| `status` | `--project` | Filter by project + tracks cross-project agents (status_cmd.go:59) |
| `serve` | N/A | Has cross-project workspace scanning for dashboard (serve_agents.go:337-352) |

**Source:** cmd/orch/daemon.go:160-169, spawn_cmd.go:196, complete_cmd.go:145, abandon_cmd.go:61, status_cmd.go:59

**Significance:** These commands demonstrate the correct pattern: accept `--workdir` for explicit override, with auto-detection fallback from workspace SPAWN_CONTEXT.md.

---

### Finding 2: Twenty-two commands hardcode os.Getwd() without cross-project support

**Evidence:** Commands using hardcoded `os.Getwd()` without `--workdir` flag:

| Command | os.Getwd() Location | Severity |
|---------|---------------------|----------|
| `frontier` | frontier.go (implicit via CalculateFrontier) | P0 - Critical |
| `review` | review.go:140, 460, 792 | P0 - Critical |
| `tail` | tail_cmd.go:96 | P1 - High |
| `question` | question_cmd.go:37 | P1 - High |
| `clean` | clean_cmd.go:299, 522 | P1 - High |
| `resume` | resume.go:142, 223, 325 | P1 - High |
| `attach` | attach.go:40 | P1 - High |
| `wait` | wait.go:155 | P2 - Medium |
| `reconcile` | reconcile.go:73 | P2 - Medium |
| `handoff` | handoff.go:272 | P2 - Medium |
| `history` | history.go:101 | P2 - Medium |
| `doctor` | doctor.go:924, 1118, 1223 | P3 - Low |
| `init` | init.go:102 | P3 - Low |
| `focus` | focus.go:263 | P3 - Low |
| `hotspot` | hotspot.go:102 | P3 - Low |
| `claim` | claim_cmd.go:47 | P3 - Low |
| `config` | config_cmd.go:102, 150 | P3 - Low |
| `deploy` | deploy.go:186, 483 | P3 - Low |
| `test_report` | test_report_cmd.go:101 | P3 - Low |
| `kb` | kb.go:491, 563, 698 | P3 - Low |
| `serve_approve` | serve_approve.go:124 | P3 - Low |
| `serve_hotspot` | serve_hotspot.go:23 | P3 - Low |

**Source:** Grep for `os\.Getwd` across cmd/orch/*.go

**Significance:** The P0/P1 commands are frequently needed for cross-project orchestration. Orchestrators running from one project need to review/debug/clean agents in other projects.

---

### Finding 3: complete_cmd.go provides the reference implementation pattern

**Evidence:** The `complete` command (complete_cmd.go:310-475) implements the full cross-project pattern:

1. **Explicit flag**: `--workdir` for user override (line 145)
2. **Auto-detection**: Extracts PROJECT_DIR from workspace SPAWN_CONTEXT.md (lines 438-447)
3. **Beads integration**: Sets `beads.DefaultDir` before any beads operations (lines 452-455)
4. **Helpful errors**: Suggests `--workdir` when cross-project is detected but fails (lines 468-475)

```go
// Auto-detect from workspace SPAWN_CONTEXT.md
projectDirFromWorkspace := extractProjectDirFromWorkspace(workspacePath)
if projectDirFromWorkspace != "" && projectDirFromWorkspace != currentDir {
    // Cross-project agent detected
    beadsProjectDir = projectDirFromWorkspace
    fmt.Printf("Auto-detected cross-project: %s\n", filepath.Base(beadsProjectDir))
}
```

**Source:** cmd/orch/complete_cmd.go:310-475, especially lines 420-455 for the cross-project logic

**Significance:** This pattern should be replicated to the P0/P1 commands. The auto-detection from workspace is especially important for ergonomics.

---

## Synthesis

**Key Insights:**

1. **Pattern exists but not applied consistently** - The complete_cmd.go implementation shows the correct pattern (--workdir flag + auto-detection from workspace), but 22 commands don't implement it.

2. **Critical workflow gaps** - P0 commands (`review`, `frontier`) are essential for orchestrators working across projects. Without cross-project support, orchestrators must `cd` to each project to run these commands.

3. **The constraint kb-d29e8a is violated** - The constraint states "Cross-project orchestration must work - orch commands must not assume cwd equals target project." This is violated by most commands.

**Answer to Investigation Question:**

Only 7 of 30+ orch CLI commands support cross-project operations. The supported commands are: `daemon` (--cross-project), `spawn` (--workdir), `complete` (--workdir), `work` (--workdir), `abandon` (--workdir), `status` (--project), and `serve` (implicit via workspace scanning).

22 commands use hardcoded `os.Getwd()` without cross-project support. The most critical gaps are:
- **P0 Critical:** `review`, `frontier` - Orchestrators need these to review agents and see decidability state across projects
- **P1 High:** `tail`, `question`, `clean`, `resume`, `attach` - Frequently needed for debugging/managing agents from other projects

The constraint kb-d29e8a is violated by most commands. The fix is to implement `--workdir` flags following the pattern in complete_cmd.go.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code audit confirmed os.Getwd() usage in 22 commands (verified: grep across cmd/orch/*.go)
- ✅ complete_cmd.go has --workdir flag with auto-detection (verified: read lines 145, 420-455)
- ✅ daemon.go has --cross-project flag (verified: read lines 160-169)

**What's untested:**

- ⚠️ Actual runtime behavior of commands when run from project A targeting project B (code review only)
- ⚠️ Whether auto-detection from SPAWN_CONTEXT.md works reliably in all edge cases
- ⚠️ Dashboard visibility for cross-project agents (serve_agents.go has code but not tested)

**What would change this:**

- Finding would be wrong if commands accept positional project argument instead of flag
- Finding would be wrong if os.Getwd() is overridden later in the call chain (checked, it's not)
- Severity ratings could change based on actual orchestrator workflow frequency

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add --workdir to P0/P1 commands | architectural | Spans 8 commands, requires consistent UX pattern |
| Create shared helper for cross-project logic | implementation | Stays within pkg/shared.go |

### Recommended Approach ⭐

**Incremental --workdir Implementation** - Add `--workdir` flag to P0 commands first, following complete_cmd.go pattern.

**Why this approach:**
- complete_cmd.go proves the pattern works (auto-detection + explicit override)
- P0 commands (`review`, `frontier`) are blocking orchestrator workflows
- Incremental rollout allows validation before wider adoption

**Trade-offs accepted:**
- P2/P3 commands remain project-local (acceptable: rarely needed cross-project)
- Some code duplication across commands (can extract to shared helper later)

**Implementation sequence:**
1. Extract cross-project logic from complete_cmd.go to shared helper in shared.go
2. Add --workdir to `review` command (most critical for orchestrator workflow)
3. Add --workdir to `frontier` command (completes P0)
4. Add --workdir to P1 commands: `tail`, `question`, `clean`, `resume`, `attach`

### Alternative Approaches Considered

**Option B: Global --workdir flag on rootCmd**
- **Pros:** Single implementation, consistent across all commands
- **Cons:** May not make sense for all commands (init, config); harder to test
- **When to use instead:** If most commands need cross-project support

**Option C: Environment variable ORCH_PROJECT_DIR**
- **Pros:** No flag repetition for multi-command workflows
- **Cons:** Implicit, easy to forget; doesn't work for one-off cross-project calls
- **When to use instead:** For automated scripts that always target one project

**Rationale for recommendation:** Option A (incremental --workdir) balances implementation effort with immediate value. The complete_cmd.go pattern is proven and can be extracted for reuse.

---

### Implementation Details

**What to implement first:**
- `review` command --workdir support (most critical gap)
- `frontier` command --workdir support (P0 complete)
- Extract `resolveProjectDir(workdir string) (string, error)` helper to shared.go

**Things to watch out for:**
- ⚠️ beads.DefaultDir must be set BEFORE any beads operations
- ⚠️ Auto-detection from workspace SPAWN_CONTEXT.md requires PROJECT_DIR field
- ⚠️ Cross-project agents may have workspaces in different .orch/workspace/ directories

**Areas needing further investigation:**
- How to handle commands that don't take a beads ID (e.g., `frontier` shows all issues)
- Whether serve_agents.go cross-project scanning is working correctly

**Success criteria:**
- ✅ `orch review --workdir ~/other-project` shows agents from other-project
- ✅ `orch frontier --workdir ~/other-project` shows decidability state for other-project
- ✅ Orchestrators can manage agents across projects without `cd`

---

## References

**Files Examined:**
- cmd/orch/daemon.go - Cross-project flag implementation
- cmd/orch/spawn_cmd.go - --workdir flag for spawn
- cmd/orch/complete_cmd.go - Reference implementation for cross-project pattern
- cmd/orch/status_cmd.go - --project filter flag
- cmd/orch/frontier.go - P0 gap, uses os.Getwd() implicitly
- cmd/orch/review.go - P0 gap, uses os.Getwd() at lines 140, 460, 792
- cmd/orch/tail_cmd.go - P1 gap, uses os.Getwd() at line 96
- cmd/orch/question_cmd.go - P1 gap, uses os.Getwd() at line 37
- cmd/orch/work_cmd.go - Has --workdir support
- cmd/orch/abandon_cmd.go - Has --workdir support

**Commands Run:**
```bash
# Find all os.Getwd() usage across commands
grep -n "os\.Getwd\|--workdir\|--project\|crossProject\|cross-project" cmd/orch/*.go

# List all command files
ls cmd/orch/*.go | grep -v _test.go
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Constraint:** kb-d29e8a - "Cross-project orchestration must work - orch commands must not assume cwd equals target project"

---

## Investigation History

**2026-01-30:** Investigation started
- Initial question: Which orch CLI commands support cross-project operations?
- Context: Beads issue orch-go-21096, constraint kb-d29e8a states cross-project must work

**2026-01-30:** Code audit completed
- Identified 7 commands with cross-project support
- Identified 22 commands using hardcoded os.Getwd()
- Documented reference implementation in complete_cmd.go

**2026-01-30:** Investigation completed
- Status: Complete
- Key outcome: Most commands violate constraint kb-d29e8a; P0/P1 commands need --workdir flags
