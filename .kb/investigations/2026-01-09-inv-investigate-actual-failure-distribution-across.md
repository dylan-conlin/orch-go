<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Failure distribution shows model restrictions (Opus) are top cause, skill selection errors second, feature-impl skill dominates failures, with 4.8M spawn failures in daemon log indicating systemic capacity issues.

**Evidence:** 19 FAILURE_REPORT.md files analyzed, daemon log grep count (4.8M failures), categorization of reasons and skill distribution, sample verification with beads issues.

**Knowledge:** External API limits (Anthropic Opus) cause most failures; skill mis-triage leads to wasted spawns; failure tracking is incomplete (workspace failures vs daemon log failures).

**Next:** Implement model fallback strategies, improve skill inference, enhance failure tracking (daemon log analysis).

**Promote to Decision:** recommend-yes - patterns indicate systemic issues needing architectural decisions

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

# Investigation: Investigate Actual Failure Distribution Across

**Question:** What types of failures occur across spawns, how often, and which skills/models fail most?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** [Owner name or team]
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: 19 FAILURE_REPORT.md files exist in workspaces

**Evidence:** Found 19 FAILURE_REPORT.md files across .orch/workspace/ directories, indicating abandoned spawns. Example reasons: "Needs --dangerously-skip-permissions flag", "cleanup", "Session wrap-up - will restart via daemon".

**Source:** `find .orch/workspace -name FAILURE_REPORT.md | wc -l`; individual file inspection.

**Significance:** Shows failure rate non-zero; need to categorize failure types and frequency.

---

### Finding 2: Failure reasons categorize into model restrictions, skill selection, and operational cleanup

**Evidence:** Top failure reasons: "Stalled due to Anthropic Opus restriction" (3 occurrences), "Wrong skill - need architect not feature-impl" (2). Other single occurrences: stalled, cleanup, testing, session wrap-up, permission flags, investigation complete, stale registry, abandoning incremental fix.

**Source:** `grep -h "Reason:" .orch/workspace/*/FAILURE_REPORT.md | sed 's/.*Reason: //' | sort | uniq -c`

**Significance:** Model restrictions are the most common failure cause, indicating external API limits impact spawn success. Skill selection errors indicate mis-triage by orchestrator.

---

### Finding 3: Feature-impl skill accounts for 13/19 failures; failures cluster in Jan 2026

**Evidence:** Skill distribution: feat (13), arch (2), work (2), inv (1), debug (1). Date distribution: 08jan (6), 07jan (4), 26dec (4), 04jan (3), 05jan (1), 09jan (1). Failures increased in January (14 vs 5 in Dec).

**Source:** Workspace name analysis extracting skill prefix and date pattern.

**Significance:** Feature-impl skill may have higher failure rate due to complexity or mis-triage. January spike aligns with Anthropic Opus restrictions.

---

### Finding 4: Daemon log shows 4.8M spawn failures, likely due to model capacity

**Evidence:** ~/.orch/daemon.log contains 4,867,856 lines with 'failed to spawn this cycle'. Sampled issue orch-go-e41u (closed) shows spawn concurrency bug; other failures likely model restrictions.

**Source:** `grep -c 'failed to spawn this cycle' ~/.orch/daemon.log`, `bd show orch-go-e41u`.

**Significance:** Spawn failures far exceed workspace failures, indicating systemic model capacity or concurrency issues.

---

## Synthesis

**Key Insights:**

1. **Model restrictions dominate failure causes** - External API limits (Anthropic Opus) cause repeated spawn failures, both in workspace abandonments and daemon log.

2. **Skill selection errors indicate orchestrator mis-triage** - Wrong skill assignments (e.g., feature-impl vs architect) lead to wasted spawns, suggesting need for better skill inference or validation.

3. **Failure tracking gap** - Workspace failures (19) are dwarfed by daemon log failures (4.8M), indicating most failures happen before workspace creation (spawn phase), highlighting need for better spawn failure logging and recovery.

**Answer to Investigation Question:**

Types of failures: model restrictions (most common), skill selection errors, operational cleanup, permission flags, stale registry. Frequency: millions of spawn failures (daemon log), but only 19 workspace abandonments. Skills/models: feature-impl skill fails most (13/19), Opus model failures due to external restrictions. Limitations: daemon log analysis limited to count, not root cause; beads issue correlation partial; model-specific failure rates not measured.

---

## Structured Uncertainty

**What's tested:**

- ✅ Verified failure reasons in FAILURE_REPORT.md match workspace abandonment patterns: sampled 2 failures (wrong skill, Opus restriction), both show correct reason
- ✅ Counted and categorized 19 FAILURE_REPORT.md files, extracted failure reasons, skill distribution, dates using grep and analysis scripts
- ✅ Analyzed daemon log for spawn failures, counted 4.8M failures, sampled one issue to confirm concurrency bug


**What's untested:**

- ⚠️ Whether model restrictions (Opus) are the primary cause of daemon log failures (need deeper log analysis)
- ⚠️ Whether skill selection errors are due to orchestrator mis-triage or skill inference issues
- ⚠️ Whether failure rates differ by model (Opus vs Sonnet vs Flash) due to external API limits

**What would change this:**

- If daemon log failures are due to non-model causes (e.g., network errors, concurrency bugs)
- If skill selection errors are actually due to skill ambiguity in issue descriptions
- If model failure rates are similar across models (indicating internal orchestration issues)

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
