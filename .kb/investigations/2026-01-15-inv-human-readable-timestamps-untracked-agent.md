<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Human Readable Timestamps Untracked Agent

**Question:** How can we make untracked agent IDs (e.g., orch-go-untracked-1768090360) more human-readable without losing uniqueness?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** Agent orch-go-ni18f
**Phase:** Investigating
**Next Step:** Analyze findings and recommend approach
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Untracked IDs are generated with Unix timestamps for uniqueness

**Evidence:** In spawn_cmd.go line 1851, untracked agent IDs are created with: `fmt.Sprintf("%s-untracked-%d", projectName, time.Now().Unix())`. Example: orch-go-untracked-1768090360.

**Source:** cmd/orch/spawn_cmd.go:1851

**Significance:** The Unix timestamp ensures uniqueness but is not human-readable at a glance. Changing the generation format would impact all code that parses these IDs.

---

### Finding 2: Multiple locations check for untracked IDs using string matching

**Evidence:** Found functions like isUntrackedBeadsID() in multiple files (pkg/daemon/active_count.go:155, cmd/orch/shared.go:91, cmd/orch/stats_cmd.go:53) that detect untracked agents by checking if ID contains "-untracked-".

**Source:** grep results showing 10 matches for isUntrackedBeadsID functions

**Significance:** The parsing logic is distributed across the codebase. Changing ID format could break these checks unless they're updated or remain backwards-compatible.

---

### Finding 3: Status display happens in status_cmd.go printAgentsWideFormat

**Evidence:** The status command displays agent information in table format with columns for BEADS ID, MODE, MODEL, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS. Beads IDs are displayed directly without transformation.

**Source:** cmd/orch/status_cmd.go:956-1025 (printAgentsWideFormat function)

**Significance:** The display layer is where we can intercept and format the ID for human readability without changing the underlying ID generation or storage.

---

## Synthesis

**Key Insights:**

1. **ID format is intentional for uniqueness** - The Unix timestamp ensures no two untracked agents can have the same ID, even if spawned in rapid succession. Changing this would require alternative uniqueness guarantees.

2. **Display layer is the safest transformation point** - Multiple parts of the codebase parse untracked IDs by looking for "-untracked-" substring. Changing the ID format itself would require updating all these locations and testing thoroughly.

3. **Status display already has formatting helpers** - Functions like formatModelForDisplay() show the codebase already transforms values for display. Adding formatBeadsID() follows this established pattern.

**Answer to Investigation Question:**

Transform untracked IDs only at the display layer (in status_cmd.go) by extracting the Unix timestamp, converting it to human-readable format (e.g., Jan14-1823), and displaying as "orch-go-untracked-Jan14-1823". This preserves the underlying ID uniqueness (Finding 1), avoids breaking existing parsing logic (Finding 2), and can be implemented as a simple display transformation (Finding 3).

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

**Display-layer transformation** - Keep Unix timestamp in IDs, add formatBeadsIDForDisplay() helper that transforms untracked IDs to human-readable format only when displaying to users.

**Why this approach:**
- Preserves uniqueness guarantee of Unix timestamps (Finding 1)
- No changes needed to ID generation or parsing logic across codebase (Finding 2)
- Follows existing pattern of display-specific formatters like formatModelForDisplay() (Finding 3)
- Minimal risk - display-only changes can't break core functionality

**Trade-offs accepted:**
- Internal logs and debugging still show Unix timestamp (requires using formatted display)
- Need to add formatting in all display locations (status, logs, etc.) - but this is a small surface area

**Implementation sequence:**
1. Add formatBeadsIDForDisplay() helper in cmd/orch/shared.go (already has helper functions)
2. Update printAgentsWideFormat(), printAgentsNarrowFormat(), printAgentsCardFormat() to use formatter
3. Test with actual untracked agents to verify readability

### Alternative Approaches Considered

**Option B: Change ID generation to use human-readable format**
- **Pros:** All locations automatically get human-readable IDs
- **Cons:** Risk of collision if multiple agents spawn in same minute; requires updating all parsing logic (Finding 2); need to ensure uniqueness another way
- **When to use instead:** If uniqueness can be guaranteed through other means (e.g., random suffix)

**Option C: Add AGE column showing relative time**
- **Pros:** Preserves existing IDs completely, adds useful info
- **Cons:** Doesn't solve the core problem (IDs still unreadable); makes already-wide table even wider
- **When to use instead:** As a complementary feature after making IDs readable

**Rationale for recommendation:** Option A (display transformation) provides immediate readability improvement with minimal risk. It doesn't require changing ID generation (avoiding collision concerns), doesn't break existing parsing (Finding 2), and can be rolled back easily if issues arise. The display layer is already designed for transformation (Finding 3).

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
