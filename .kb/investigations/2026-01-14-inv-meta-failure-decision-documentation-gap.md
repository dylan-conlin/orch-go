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

- ✅ Empty templates exist in archive (verified: counted 10+ files with 36-89 placeholders via grep)
- ✅ Model Evolution doesn't mention follow-orchestrator beads feature (verified: read dashboard-architecture.md:219-223)
- ✅ "Promote to Decision: recommend-yes" exists but rare (verified: found ~10 cases vs 107 recommend-no via grep)
- ✅ kb reflect --type promote doesn't check investigation files (verified: ran command, got "No promote opportunities found")
- ✅ Follow-orchestrator investigation is complete and filled (verified: read full D.E.K.N. summary with concrete evidence)

**What's untested:**

- ⚠️ Whether orchestrators are expected to manually grep for "recommend-yes" flags (process documentation doesn't specify)
- ⚠️ Whether follow-orchestrator work truly warranted a decision (inferred from adding cross-project capability and per-project caching pattern)
- ⚠️ Root cause of agent death/restart pattern (didn't investigate why agents die mid-investigation)
- ⚠️ Whether there are other promotion workflows besides kb reflect (didn't search all orchestrator guidance)

**What would change this:**

- If orchestrator skill explicitly documents manual promotion workflow, findings about "missing feedback loop" would be wrong
- If kb reflect had a separate --type for investigation promotion, the tooling gap claim would be incorrect
- If follow-orchestrator investigation had clear "tactical only" reasoning, the "recommend-no seems wrong" claim would be invalid

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add kb reflect --type investigation-promotion to make "Promote to Decision" field actionable** - Extend kb reflect to search investigation files for "recommend-yes" flags and surface them for orchestrator review.

**Why this approach:**
- Closes the feedback loop - the field becomes actionable instead of performative (addresses Finding 4, 5)
- Reuses existing kb reflect pattern that orchestrators already know (--type synthesis, --type stale work well)
- Makes promotion a proactive check rather than hoping orchestrators manually grep
- Surfaces investigations that recommended promotion but were ignored (like the synthesis completion and auth spoofing cases)

**Trade-offs accepted:**
- Doesn't fix the root cause of empty templates (agent death/restart) - that needs separate investigation
- Doesn't automatically update model Evolution sections - still requires orchestrator judgment
- Adds another kb reflect type to check (but that's better than nothing)

**Implementation sequence:**
1. Add `kb reflect --type investigation-promotion` that greps `.kb/investigations/**/*.md` for "Promote to Decision: recommend-yes"
2. Format output to show investigation path, date, and recommendation reason (from D.E.K.N. summary)
3. Update orchestrator skill to reference this in completion workflow: "After completing investigation agent, run `kb reflect --type investigation-promotion` to check for decisions to create"
4. Add to SessionStart hook suggestions: if investigation-promotion count > 0, surface as hygiene item

### Alternative Approaches Considered

**Option B: Remove "Promote to Decision" field from investigation template**
- **Pros:** Eliminates performative documentation, reduces template complexity
- **Cons:** Loses the explicit forcing function - orchestrators might forget to consider promotion entirely
- **When to use instead:** If we shift to "kb quick decide" during investigation as primary decision capture mechanism

**Option C: Document manual promotion workflow in orchestrator skill**
- **Pros:** No code changes needed, clarifies existing process
- **Cons:** Relies on orchestrator discipline - easy to skip during busy sessions, same problem as current state
- **When to use instead:** As a stopgap until kb reflect can be extended

**Option D: Use kb quick decide during investigation instead of post-investigation promotion**
- **Pros:** Captures decisions in the moment when context is fresh, already has tooling (kb reflect --type promote)
- **Cons:** Requires changing investigation skill guidance, doesn't help with existing backlog of recommend-yes investigations
- **When to use instead:** For future investigations - still need to address existing backlog

**Rationale for recommendation:** Option A (kb reflect --type investigation-promotion) addresses the immediate backlog of ~10 investigations flagged for promotion while creating a proactive check for future cases. Options B-D either abandon the field entirely (losing the forcing function) or rely on discipline (proven insufficient). Extending kb reflect is the only approach that makes the field actionable without changing investigation workflow.

---

### Implementation Details

**What to implement first:**
- Add `kb reflect --type investigation-promotion` to kb-cli (immediate value, surfaces existing backlog)
- Clean up empty templates from `.kb/investigations/archived/` (reduces noise, one-time cleanup)
- Create beads issues for the ~10 investigations with "recommend-yes" that need decision promotion (address backlog)
- Update dashboard-architecture.md Evolution section with Jan 7 follow-orchestrator entry (fix identified gap)

**Things to watch out for:**
- ⚠️ "Promote to Decision" field might have variations in wording (recommend-yes vs Recommend-yes vs recommend_yes) - grep needs to be case-insensitive and flexible
- ⚠️ Investigation template has evolved - older investigations might not have D.E.K.N. summary section, need fallback to extract recommendation reason
- ⚠️ Some investigations might be archived but still have "recommend-yes" - decide whether to include archived in the search or skip them
- ⚠️ Root cause of agent death/restart pattern unknown (Finding 3) - cleaning up empty templates is a symptom fix, not root cause fix

**Areas needing further investigation:**
- Why do agents die mid-investigation and create new files instead of resuming? (affects 10+ investigations)
- What criteria should determine "recommend-yes" vs "recommend-no"? (107 recommend-no suggests possible over-use of that flag)
- Should model Evolution sections be auto-updated from investigation completions? (or remain manual curation)
- Is there a way to validate "tactical vs architectural" classification automatically?

**Success criteria:**
- ✅ `kb reflect --type investigation-promotion` surfaces the ~10 known recommend-yes investigations
- ✅ Running the command takes <1s (should be simple grep, not complex analysis)
- ✅ Output includes enough context to decide whether to act (investigation path, date, reason from D.E.K.N.)
- ✅ Orchestrator skill documents when to run this command (completion workflow, session hygiene)
- ✅ Empty templates removed from archive directory (reduces from 10+ to 0)
- ✅ Dashboard model Evolution section updated with follow-orchestrator entry

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
