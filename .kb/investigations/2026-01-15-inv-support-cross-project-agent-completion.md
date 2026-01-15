<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch complete` fails on cross-project agents because beads ID resolution happens before --workdir processing, looking in wrong project's .beads database.

**Evidence:** Tested `orch complete pw-ed7h --workdir ~/path/to/price-watch` fails with "beads issue 'pw-ed7h' not found" even with correct --workdir.

**Knowledge:** resolveShortBeadsID (complete_cmd.go:360) uses current dir's beads database before beadsProjectDir is determined (line 369-405); auto-detection from beads ID prefix can make cross-project completion "just work".

**Next:** Implement auto-detection of project from beads ID prefix before resolution, using existing findProjectDirByName pattern from status_cmd.go.

**Promote to Decision:** recommend-no - tactical fix for cross-project workflow, uses existing patterns

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

# Investigation: Support Cross Project Agent Completion

**Question:** Why does `orch complete` fail on cross-project agents that appear in `orch status`, and how should we fix it?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** og-feat-support-cross-project-15jan-acb3
**Phase:** Investigating
**Next Step:** Implement auto-detection solution
**Status:** Active

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Cross-project agents visible in status but not completable

**Evidence:**
- `orch status --json` shows 3 price-watch agents: pw-ed7h, pw-jq31, pw-jfxr.2
- Running `orch complete pw-ed7h` from orch-go directory fails with "beads issue 'pw-ed7h' not found"
- Even with `--workdir ~/path/to/price-watch`, the command still fails with same error

**Source:** Tested `orch status --json | jq '.agents[] | select(.project != "orch-go")'` and `orch complete pw-ed7h --workdir ~/path/to/price-watch`

**Significance:** Users can see cross-project work in status but cannot act on it, creating asymmetry in the UX. The --workdir flag doesn't help because the error occurs before that flag is processed.

---

### Finding 2: Beads ID resolution happens before workdir processing

**Evidence:**
- Line 360 in complete_cmd.go: `resolveShortBeadsID(identifier)` called to resolve beads ID
- Lines 369-405: beadsProjectDir determination (including --workdir) happens AFTER resolution
- resolveShortBeadsID uses `beads.FindSocketPath("")` which defaults to current directory
- Result: Cross-project beads IDs are resolved against wrong project's database

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/complete_cmd.go:358-405`

**Significance:** Timing mismatch - we need to know the project before resolving the ID, but we determine the project after resolution fails. This is a sequencing bug in the command flow.

---

### Finding 3: Beads ID prefix contains project information

**Evidence:**
- Beads IDs follow format: `{project}-{short-id}` (e.g., pw-ed7h, orch-go-nqgjr, kb-cli-abc1)
- Project prefix is extractable: `extractProjectFromBeadsID()` already exists
- `findProjectDirByName()` in status_cmd.go:1342 can locate project directories by name
- Pattern: Search ~/Documents/personal/{project}, ~/{project}, ~/projects/{project}, ~/src/{project}

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/status_cmd.go:1342-1369`, beads ID analysis

**Significance:** The solution is self-contained - beads ID itself tells us which project to use. We can auto-detect the project directory before resolution, making cross-project completion "just work" without flags.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Auto-detect project from beads ID prefix** - Extract project name from beads ID (e.g., "pw" from "pw-ed7h"), locate project directory, and set beads.DefaultDir before resolution.

**Why this approach:**
- Makes cross-project completion "just work" - no flags needed when agent is visible in status
- Uses existing infrastructure (`extractProjectFromBeadsID`, `findProjectDirByName`)
- Aligns with user expectations - status shows agent, complete should work on it
- Beads ID is self-describing - contains project information we should use

**Trade-offs accepted:**
- Relies on project naming conventions (beads IDs have project prefix)
- Requires projects to be in standard locations (~/Documents/personal/{name}, etc.)
- Won't work for non-standard project structures (acceptable - can still use --workdir)

**Implementation sequence:**
1. Extract project name from beads ID before resolution (uses existing extractProjectFromBeadsID)
2. Auto-detect project directory using findProjectDirByName pattern
3. Set beads.DefaultDir early if cross-project detected
4. Continue with existing resolution logic (now looks in correct project)

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

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
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

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
