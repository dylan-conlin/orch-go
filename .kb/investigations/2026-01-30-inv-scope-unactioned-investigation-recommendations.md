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

# Investigation: Scope Unactioned Investigation Recommendations

**Question:** Which investigation recommendations in .kb/investigations/ remain unactioned, and what work is needed to action them?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Investigation Worker (og-inv-scope-unactioned-investigation-30jan-bab5)
**Phase:** Investigating
**Next Step:** Search all investigation files for recommendations, identify unactioned items
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Approach - Identify All Investigation Files

**Evidence:** Will search .kb/investigations/ for all investigation files, then examine each for recommendations in "Next:" fields and Implementation Recommendations sections.

**Source:** Starting with `find .kb/investigations/ -name "*.md"` and systematic review

**Significance:** Establishes baseline of what investigations exist and provides corpus for identifying unactioned recommendations.

---

### Finding 2: 237 Investigations Have Actionable Recommendations

**Evidence:** Search for investigations with actionable "Next:" fields (starting with Implement, Add, Create, Fix, Build) found 237 matches out of 702 total investigation files. Sample shows patterns like:
- "Implement `orch tail --session` flag"
- "Add usage caching (30-60s TTL)"
- "Implement in 3 phases: prompt-based action space restriction..."
- "Create issues for both optimizations"

**Source:** `rg "^\*\*Next:\*\* (Implement|Add|Create|Fix|Build)" .kb/investigations/ --type md -l | wc -l` returned 237 files

**Significance:** Large corpus of recommendations exists, but need to distinguish between:
1. Recommendations that were acted upon (implementation completed)
2. Recommendations that spawned tracked issues/work
3. Recommendations that remain completely unactioned

---

### Finding 3: Investigation Status Doesn't Indicate Action Taken

**Evidence:** Examined three sample investigations:
1. 2026-01-29-inv-orch-cannot-inspect-opencode-sessions.md - Status: Complete, Next: "Implement...", Investigation History shows "Implementation completed and verified" - recommendation WAS acted upon
2. 2026-01-28-inv-analyze-memory-usage-patterns.md - Status: Complete, Next: "Create issues for both optimizations" - unclear if issues were created
3. 2026-01-27-inv-design-information-hiding-tool-restriction.md - Status: Complete, Next: "Implement in 3 phases" - unclear if implementation happened

**Source:** Direct file reading of sample investigations

**Significance:** Investigation Status: Complete doesn't mean recommendation was acted upon - it only means the investigation concluded. Some investigations include implementation (like #1), while others just provide recommendations (like #2, #3). Need to cross-reference with beads issues, git commits, or subsequent work artifacts to determine which recommendations remain unactioned.

---

### Finding 4: Some Recommendations Become Beads Issues, Others Don't

**Evidence:** Cross-referenced investigation recommendations with open beads issues. Found matches:
- 2026-01-23-inv-gastown investigation recommended "Create beads issue to evaluate GUPP-style hooks" → became orch-go-0ns2e
- Investigation recommended "Investigate Strategic Center dashboard" → became orch-go-21022
- Investigation recommended "Auto-resume agents after OpenCode/server restart" → became orch-go-21032

However, many recommendations do NOT have corresponding beads issues. Example recommendations without visible tracking:
- "Add usage caching (30-60s TTL)" from 2026-01-28-inv-analyze-memory-usage-patterns.md
- "Implement in 3 phases: prompt-based action space restriction..." from 2026-01-27-inv-design-information-hiding-tool-restriction.md
- "Implement `orch servers` subcommands" from 2025-12-23-inv-explore-options-centralized-server-management.md

**Source:** `bd list --status open --limit 0 | rg -i "GUPP|Strategic Center|auto-resume"` found 3 matches; manual review of other recommendations found no corresponding issues

**Significance:** There's an inconsistent pattern of converting investigation recommendations into tracked work. Some get issues created, others remain as recommendations in completed investigation files. Need systematic approach to identify untracked recommendations.

---

### Finding 5: Verification Shows Mixed Implementation Status

**Evidence:** Tested three specific recommendations from investigations:
1. "Implement `orch servers` subcommands" (2025-12-23-inv-explore-options-centralized-server-management.md) - IMPLEMENTED: `orch servers --help` shows working command with all recommended subcommands
2. "Implement action space restriction" (2026-01-27-inv-design-information-hiding-tool-restriction.md) - IMPLEMENTED: `~/.claude/skills/meta/orchestrator/SKILL.md` contains exact "You CAN (meta-actions)" and "You CANNOT (primitive actions)" sections
3. "Add usage caching (30-60s TTL)" (2026-01-28-inv-analyze-memory-usage-patterns.md) - NOT IMPLEMENTED: `rg -i "cache.*usage" pkg/usage/` returns no results

**Source:** 
- `orch servers --help` command execution
- `rg "You CAN \(meta-actions\)" ~/.claude/skills/meta/orchestrator/` search
- `rg -i "cache.*usage" pkg/usage/` search

**Significance:** This confirms that investigation recommendations have three possible states:
1. Implemented directly (without beads issue tracking)
2. Tracked via beads issue (may be open or in-progress)
3. Completely unactioned (no implementation, no issue)

Need methodology to identify category 3 (unactioned) recommendations systematically.

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

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

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
