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

# Investigation: Meta Failure Decision Documentation Gap

**Question:** What is the pattern of decision documentation gaps using the Jan 7 follow-orchestrator case as an example, and what process failures enabled it?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-inv-meta-failure-decision-14jan-00a2
**Phase:** Investigating
**Next Step:** Document findings and test hypotheses
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Empty investigation template archived instead of cleaned up

**Evidence:** Found empty investigation template at `.kb/investigations/archived/2026-01-07-inv-implement-follow-orchestrator-dashboard-filtering.md` - 225 lines of template with no content filled. This was archived alongside a completed investigation on the same topic.

**Source:**
- `.kb/investigations/archived/2026-01-07-inv-implement-follow-orchestrator-dashboard-filtering.md` (empty template)
- `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` (completed investigation)
- Found 3 total archived investigations from Jan 7 via `ls -la .kb/investigations/archived/ | grep 2026-01-07 | wc -l`

**Significance:** Empty templates indicate process failure - agent likely died/restarted and created new investigation instead of continuing the original. Archiving instead of deleting preserves noise.

---

### Finding 2: Significant feature not documented in model Evolution section

**Evidence:** The completed investigation `2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` implemented "dashboard beads follow orchestrator context via project_dir parameter" - a significant architectural change. However, the `dashboard-architecture.md` model's Evolution section for Jan 7, 2026 only mentions "Two-Mode Design" and doesn't mention the follow-orchestrator beads feature.

**Source:**
- `.kb/investigations/2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md:1-10` (D.E.K.N. summary shows completed feature)
- `.kb/models/dashboard-architecture.md:219-223` (Evolution section Jan 7 entry)
- Investigation says "Promote to Decision: recommend-no" even though it adds project_dir parameter support

**Significance:** Model Evolution sections should capture significant changes. The follow-orchestrator work added cross-project capability but wasn't documented in the model, making the model incomplete as an understanding artifact.

---

### Finding 3: Systemic pattern of empty investigation templates in archive

**Evidence:** Found 10+ archived investigations with 36-89 placeholders (mostly unfilled templates) from Dec 19, 2025 through Jan 7, 2026. Examples include:
- `2025-12-21-inv-implement-failure-report-md-template.md` (233 lines, 89 placeholders)
- `2025-12-21-inv-implement-orch-init-command-project.md` (233 lines, 89 placeholders)
- `2025-12-21-inv-implement-session-handoff-md-template.md` (233 lines, 89 placeholders)
- `2026-01-07-inv-implement-follow-orchestrator-dashboard-filtering.md` (225 lines, 86 placeholders)

**Source:** Searched archived investigations for high placeholder counts using `grep -c '\[.*\]'` to identify unfilled templates

**Significance:** This is not an isolated incident - it's a systemic process failure. Agents are creating investigation files, dying/restarting, creating new files instead of resuming, and the empty templates are being archived instead of deleted. The archive directory is accumulating noise rather than valuable historical context.

---

### Finding 4: "Promote to Decision" field rarely triggers actual promotion

**Evidence:** Of all non-archived investigations:
- 107 with "Promote to Decision: recommend-no"
- ~10 with "Promote to Decision: recommend-yes"
- 0 with "Promote to Decision: unclear"
- Many still have template placeholder text

Examples of recommend-yes without corresponding decisions:
- `2026-01-08-inv-synthesize-model-investigations-11-synthesis-work.md` recommends decision on "synthesis completion recognition pattern" - no decision file found
- `2026-01-09-inv-explore-opencode-github-issue-7410.md` recommends decision on "spoofing-based auth pattern" - no decision file found

**Source:** Searched all investigations for "Promote to Decision" field values

**Significance:** The "Promote to Decision" field was added to create a forcing function for decision capture, but it's not working - even when investigations recommend promotion, decisions aren't being created. The field has become performative documentation rather than actionable signal.

---

## Synthesis

**Key Insights:**

1. **Process failure pattern**: Empty investigation templates → death/restart → new file creation → archive instead of delete → noise accumulation

2. **Missing feedback loop**: "Promote to Decision: recommend-yes" doesn't trigger decision creation - no one is reading these flags or acting on them

3. **Model staleness**: Significant architectural changes (like follow-orchestrator beads support) aren't being documented in model Evolution sections, even when investigations are complete

### Finding 5: "Promote to Decision" field has no tooling support

**Evidence:**
- Investigation template includes "Promote to Decision: recommend-yes | recommend-no | unclear" field
- `kb reflect --type promote` exists but only searches kb quick entries, NOT investigation files
- Ran `kb reflect --type promote` and got "No promote opportunities found" despite having ~10 investigations with "recommend-yes"
- Orchestrator skill references `kb reflect --type promote` but for kb quick entries only

**Source:**
- `kb reflect --help` output shows it searches "kn entries worth promoting to kb decisions"
- `kb reflect --type promote` test run
- Orchestrator skill at `~/.claude/skills/meta/orchestrator/reference/orch-commands.md:58`

**Significance:** The "Promote to Decision" field in investigation templates is performative - there's no tool that reads it. Orchestrators would need to manually grep for "recommend-yes" flags, which doesn't happen. The field creates the illusion of a process without actual workflow support.

---

**Answer to Investigation Question:**

The Jan 7 follow-orchestrator case exemplifies a systemic pattern of decision documentation gaps caused by four interconnected process failures:

1. **Empty template accumulation** (Finding 1, 3): Agents create investigation files, die/restart, create new files instead of resuming, and empty templates get archived instead of deleted - creating 10+ unfilled templates as noise.

2. **Missing feedback loop** (Finding 2, 4): The "Promote to Decision" field exists in templates but has no tooling support - even when investigations recommend promotion, no one acts on these flags.

3. **Model staleness** (Finding 2): Significant architectural changes (like cross-project beads support via project_dir parameter) aren't documented in model Evolution sections, even when investigations are complete.

4. **Tooling-process mismatch** (Finding 5): `kb reflect --type promote` only checks kb quick entries, not investigation "Promote to Decision" fields, creating two disconnected promotion paths.

The follow-orchestrator case shows all four failures: empty template archived, "recommend-no" on architectural work, Evolution section incomplete, no decision created despite adding cross-project capability.

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
