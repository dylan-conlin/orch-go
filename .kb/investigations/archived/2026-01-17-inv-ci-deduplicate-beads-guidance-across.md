<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** bd prime was outputting redundant beads guidance in spawned contexts where SPAWN_CONTEXT.md already provides authoritative beads tracking instructions.

**Evidence:** Added isSpawnedContext() detection to bd prime; verified with tests showing 0 lines output in spawned context vs 80 lines in regular context; unit tests pass for spawn detection logic.

**Knowledge:** SPAWN_CONTEXT.md file presence is the reliable signal for spawn context detection; simple file existence check prevents cross-repo dependencies and maintains bd prime's silent-exit pattern.

**Next:** Close - fix implemented, tested, and committed to beads repo.

**Promote to Decision:** recommend-no (tactical bug fix implementing existing constraint from context injection model)

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

# Investigation: CI Deduplicate Beads Guidance Across Injection Sources

**Question:** Where is beads guidance being duplicated across injection sources, and how can we prevent `bd prime` from outputting redundant guidance when running in spawned agent contexts?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker Agent (orch-go-8dhhg)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: bd prime outputs beads guidance unconditionally

**Evidence:** The `outputPrimeContext` function in `/Users/dylanconlin/Documents/personal/beads/cmd/bd/prime.go` runs for all Claude Code sessions (SessionStart and PreCompact hooks) without checking whether the agent is running in a spawned context. It only checks: (1) if we're in a beads project, (2) MCP mode, and (3) stealth mode.

**Source:** 
- `/Users/dylanconlin/.claude/settings.json:152,222` - Hook invocations
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/prime.go:40-68` - Main execution logic
- `.kb/models/context-injection.md:46-48` - Documents this as Failure Mode 2

**Significance:** This causes redundant beads tracking instructions to appear in spawned agent contexts where SPAWN_CONTEXT.md already includes comprehensive beads guidance (lines 252-282 of my spawn context).

---

### Finding 2: SPAWN_CONTEXT.md is the authoritative source for spawned agents

**Evidence:** The context injection model establishes that "For spawned agents, SPAWN_CONTEXT.md is the source of truth. Hooks must back off to avoid duplication." SPAWN_CONTEXT.md already includes comprehensive beads tracking guidance in the "BEADS PROGRESS TRACKING" section.

**Source:** 
- `.kb/models/context-injection.md:59` - Constraint 2: Authoritative Spawn Context
- My own SPAWN_CONTEXT.md:252-282 - Contains full beads tracking instructions

**Significance:** bd prime should detect when running in a spawned context and skip output to respect SPAWN_CONTEXT.md as the authoritative source.

---

### Finding 3: SPAWN_CONTEXT.md is the reliable detection signal

**Evidence:** Every spawned agent workspace contains a SPAWN_CONTEXT.md file in its working directory. Found 5 active workspaces, all containing SPAWN_CONTEXT.md. No environment variables are set to indicate spawned contexts, making file presence the most reliable detection mechanism.

**Source:**
- Command: `find /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace -name "SPAWN_CONTEXT.md"`
- Result: All 5 active workspaces contain SPAWN_CONTEXT.md
- Command: `env | grep -i spawn` - No spawn-related environment variables

**Significance:** A simple file existence check in the current working directory provides reliable spawn context detection without requiring cross-repo coordination or environment variable setup.

---

## Synthesis

**Key Insights:**

1. **SPAWN_CONTEXT.md as authoritative source** - The context injection model already established that SPAWN_CONTEXT.md is the authoritative source for spawned contexts, but bd prime hook was not respecting this constraint, causing token waste.

2. **File existence is sufficient detection** - No environment variables or complex detection needed; SPAWN_CONTEXT.md presence in PWD is a reliable signal since spawned agents always have PWD set to their workspace directory.

3. **Silent exit pattern** - bd prime already uses silent exit for "not in beads project" scenarios; applying the same pattern for spawned contexts maintains consistency and prevents hook errors.

**Answer to Investigation Question:**

Beads guidance was duplicated between bd prime hook output (SessionStart and PreCompact) and SPAWN_CONTEXT.md content in spawned agent workspaces. The fix adds a simple spawn context detection (checking for SPAWN_CONTEXT.md file) that causes bd prime to silently skip output when in a spawned context, implementing the "Authoritative Spawn Context" constraint from the context injection model (Finding 2). Testing confirms 0 lines of output in spawned contexts while maintaining normal output (80 lines) in regular contexts.

---

## Structured Uncertainty

**What's tested:**

- ✅ bd prime produces 0 lines output in spawned context (verified: `cd workspace && bd prime | wc -l` returned 0)
- ✅ bd prime produces 80 lines output in regular context (verified: `cd orch-go && bd prime | wc -l` returned 80)
- ✅ isSpawnedContext() correctly detects file presence/absence (verified: unit tests pass in both scenarios)
- ✅ Implementation builds without errors (verified: `make install` succeeded)

**What's untested:**

- ⚠️ Behavior when SPAWN_CONTEXT.md exists but is empty (assumed same as file exists)
- ⚠️ Behavior when SPAWN_CONTEXT.md is a symlink (assumed os.Stat follows symlinks)
- ⚠️ Impact on PreCompact hook specifically (only tested SessionStart hook manually)

**What would change this:**

- Finding would be wrong if bd prime output still appears in spawned agent sessions after hook runs
- Finding would be wrong if regular (non-spawned) sessions show no bd prime output
- Finding would be wrong if isSpawnedContext() returns true outside workspace directories

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**SPAWN_CONTEXT.md detection in bd prime** - Add spawn context detection to bd prime's main execution flow, silently skipping output when SPAWN_CONTEXT.md exists in the current directory.

**Why this approach:**
- Directly implements the "Authoritative Spawn Context" constraint from the context injection model
- Simple file existence check with no cross-repo dependencies
- Maintains bd prime's existing silent-exit pattern (same as "not in beads project")
- Zero token overhead for spawned agents (no output at all)

**Trade-offs accepted:**
- Only detects when PWD is the workspace directory (doesn't traverse parent directories)
- This is acceptable because spawned agents always have PWD set to their workspace

**Implementation sequence:**
1. Add `isSpawnedContext()` function that checks for SPAWN_CONTEXT.md existence
2. Call this function early in the Run handler (after beadsDir check)
3. Silent exit (exit 0) if spawn context detected
4. Test with spawned agent to verify no output

### Alternative Approaches Considered

**Option B: Check PWD for `.orch/workspace/` path pattern**
- **Pros:** Simple string check
- **Cons:** Less reliable if workspace directories are moved or renamed; still requires verifying SPAWN_CONTEXT.md exists anyway
- **When to use instead:** Never - file existence is more authoritative

**Option C: Set environment variable in orch spawn**
- **Pros:** Explicit signal from spawn mechanism
- **Cons:** Requires changes in orch-go repo; cross-repo coordination complexity; environment inheritance issues
- **When to use instead:** If we need more metadata about the spawn (e.g., spawn tier, skill name)

**Rationale for recommendation:** Option A (file existence check) is the simplest and most reliable. SPAWN_CONTEXT.md is the authoritative source per the context injection model, so checking for its presence directly addresses the constraint. No cross-repo changes needed.

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/prime.go:40-68` - Main execution logic for bd prime
- `/Users/dylanconlin/Documents/personal/beads/cmd/bd/prime_test.go` - Existing test structure for reference
- `/Users/dylanconlin/.claude/settings.json:152,222` - Hook invocation points (SessionStart, PreCompact)
- `.kb/models/context-injection.md:46-62` - Context injection model with constraints

**Commands Run:**
```bash
# Build and install updated bd binary
cd /Users/dylanconlin/Documents/personal/beads && make install

# Test in spawned context (should be silent)
cd /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-arch-ci-deduplicate-beads-17jan-d747
bd prime | wc -l  # Result: 0

# Test in regular context (should output)
cd /Users/dylanconlin/Documents/personal/orch-go
bd prime | wc -l  # Result: 80

# Run unit tests
cd /Users/dylanconlin/Documents/personal/beads
go test -v ./cmd/bd/... -run TestIsSpawnedContext  # PASS
```

**External Documentation:**
- None required - internal bug fix

**Related Artifacts:**
- **Model:** `.kb/models/context-injection.md` - Established "Authoritative Spawn Context" constraint
- **Workspace:** `og-arch-ci-deduplicate-beads-17jan-d747` - Current spawned workspace used for testing

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
