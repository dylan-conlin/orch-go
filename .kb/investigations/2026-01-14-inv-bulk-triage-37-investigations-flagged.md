<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Of 37 investigations flagged for promotion, 7 warrant decisions (architectural patterns), 12 are already addressed (stale flags), and 18 are tactical (skip).

**Evidence:** Read each investigation's D.E.K.N. summary and "Promote to Decision" field; cross-referenced with existing `.kb/decisions/` to identify already-documented patterns.

**Knowledge:** Investigation promotion requires fresh triage - many "recommend-yes" flags become stale when work completes but flag isn't cleared; kb reflect correctly identifies candidates but can't detect completion.

**Next:** Create 7 decision documents in priority order: Verification Bottleneck, Two-Tier Cleanup, Schema Migration, Observation/Intervention, Registry Contract, Trust Calibration, Understanding Lag.

**Promote to Decision:** recommend-no - This is a triage report, not an architectural pattern itself.

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

# Investigation: Bulk Triage 37 Investigations Flagged for Promotion

**Question:** Of 37 investigations flagged by kb reflect --type investigation-promotion, which should be promoted to decisions vs addressed/skipped?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Triage Agent
**Phase:** Complete
**Next Step:** None - triage complete, hand off to orchestrator
**Status:** Complete

---

## Classification Summary Table

| # | Investigation | Classification | Reason | Decision to Create |
|---|---------------|----------------|--------|--------------------|
| 1 | `2026-01-10-inv-trace-verification-bottleneck-story-system.md` | **PROMOTE** | Foundational principle for human-AI collaboration; tested across 3 cases with 462 lost commits | "Verification Bottleneck Principle" |
| 2 | `2026-01-11-design-opencode-session-cleanup-mechanism.md` | **PROMOTE** | Establishes two-tier cleanup pattern (event-based + periodic background) for resource management | "Two-Tier Cleanup Pattern" |
| 3 | `2026-01-11-inv-registry-abandonment-workflow-validate-simple.md` | **PROMOTE** | Documents registry's actual contract (spawn-cache not lifecycle tracker) - affects how commands interact | "Registry Contract: Spawn-Cache Only" |
| 4 | `2026-01-11-inv-review-design-coaching-plugin-injection.md` | **PROMOTE** | Separation of observation from intervention pattern; prevents coupling bugs across behavioral monitoring | "Separate Observation from Intervention" |
| 5 | `2026-01-13-design-session-resume-discovery-failure.md` | **PROMOTE** | Schema migration pattern: backward-compatible discovery + optional migration tooling | "Schema Migration Pattern" |
| 6 | `2026-01-13-inv-implement-backward-compatible-session-resume.md` | ADDRESSED | Implementation of schema migration pattern already in Finding 5 | N/A - merged with above |
| 7 | `2026-01-10-inv-clean-up-mac-dev-environment.md` | ADDRESSED | Decision already created: `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md` | N/A - exists |
| 8 | `2026-01-09-inv-trust-calibration-meta-pattern.md` | **PROMOTE** | Meta-pattern about human-AI trust calibration; Dylan defers when he has relevant knowledge | "Trust Calibration: Assert Relevant Knowledge" |
| 9 | `2026-01-09-inv-recovery-mode-spawn-system-failure.md` | ADDRESSED | Escape hatch pattern already documented in CLAUDE.md (--mode claude --tmux) | N/A - in CLAUDE.md |
| 10 | `2026-01-09-inv-design-observability-infrastructure-overmind-docker.md` | ADDRESSED | Covered by dev-vs-prod architecture decision | N/A - covered |
| 11 | `2026-01-08-inv-design-stalled-agent-detection-agents.md` | SKIP | Tactical investigation for specific feature, not architectural pattern | N/A |
| 12 | `2026-01-08-inv-synthesize-completion-investigations-10-synthesis.md` | SKIP | Meta-synthesis document, not decision-worthy | N/A |
| 13 | `2026-01-08-inv-synthesize-model-investigations-11-synthesis.md` | SKIP | Meta-synthesis document, not decision-worthy | N/A |
| 14 | `2026-01-07-design-dashboard-activity-feed-persistence.md` | SKIP | Feature design, not cross-cutting architectural pattern | N/A |
| 15 | `2026-01-07-design-screenshot-artifact-storage-decision.md` | SKIP | Specific feature decision, not broadly applicable | N/A |
| 16 | `2026-01-07-inv-cross-project-agents-show-wrong.md` | ADDRESSED | Bug fix, already implemented | N/A |
| 17 | `2026-01-07-design-post-synthesis-investigation-archival.md` | SKIP | Specific to kb workflow, not architectural pattern | N/A |
| 18 | `2026-01-07-inv-comprehensive-template-audit-canonical-sources.md` | SKIP | Audit, not decision | N/A |
| 19 | `2026-01-08-inv-bug-session-checkpoint-alert-miscalibrated.md` | ADDRESSED | Bug fix, already implemented | N/A |
| 20 | `2026-01-08-inv-design-config-code-orch-ecosystem.md` | SKIP | Configuration design, not broad pattern | N/A |
| 21 | `2026-01-08-inv-epic-mechanize-principles-via-opencode.md` | SKIP | Epic/planning document, not decision | N/A |
| 22 | `2026-01-07-inv-address-340-active-workspaces-completion.md` | ADDRESSED | Cleanup task completed | N/A |
| 23 | `2026-01-09-inv-deepseek-reasoner-orchestrator-evaluation.md` | SKIP | Model evaluation, not architectural pattern | N/A |
| 24 | `2026-01-09-research-investigate-model-landscape-agent-tasks.md` | SKIP | Research, not decision | N/A |
| 25 | `2026-01-09-inv-add-model-mode-auto-selection.md` | ADDRESSED | Feature implemented | N/A |
| 26 | `2026-01-10-inv-debug-worker-filtering-coaching-ts.md` | ADDRESSED | Bug fix implemented | N/A |
| 27 | `2026-01-10-inv-verify-lagging-understanding-hypothesis-dec.md` | **PROMOTE** | Understanding lag meta-pattern: systems add observability faster than humans understand new visibility | "Understanding Lag Pattern" |
| 28 | `2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md` | SKIP | Session-specific checkpoint discipline, not broad pattern | N/A |
| 29 | `2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` | SKIP | Specific kb workflow issue | N/A |
| 30 | `2026-01-07-inv-orch-serve-path-issue-server.md` | ADDRESSED | Bug fix | N/A |
| 31 | `2026-01-08-inv-data-model-load-bearing.md` | SKIP | Data model design, not pattern | N/A |
| 32 | `2026-01-09-inv-explore-opencode-github-issue-7410.md` | SKIP | External issue exploration | N/A |
| 33 | `2026-01-09-inv-investigate-actual-failure-distribution-across.md` | SKIP | Analysis, not decision | N/A |
| 34 | `2026-01-08-inv-kb-cli-fix-reflect-dedup.md` | ADDRESSED | Bug fix implemented | N/A |
| 35 | `2026-01-08-inv-synthesize-synthesis-investigations-26-synthesis.md` | SKIP | Meta-synthesis | N/A |
| 36 | `2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` | SKIP | Architecture analysis, covered by session resume pattern | N/A |
| 37 | `2026-01-14-inv-meta-failure-decision-documentation-gap.md` | SKIP | Meta-observation, not actionable pattern | N/A |

---

## Summary Statistics

- **PROMOTE:** 7 investigations (worth creating decisions)
- **ADDRESSED:** 12 investigations (already implemented or documented)
- **SKIP:** 18 investigations (not decision-worthy)

---

## Decisions to Create

| Priority | Decision Title | Source Investigation | Pattern Type |
|----------|---------------|---------------------|--------------|
| 1 | Verification Bottleneck Principle | `2026-01-10-inv-trace-verification-bottleneck-story-system.md` | Foundational principle |
| 2 | Two-Tier Cleanup Pattern | `2026-01-11-design-opencode-session-cleanup-mechanism.md` | Resource management |
| 3 | Schema Migration Pattern | `2026-01-13-design-session-resume-discovery-failure.md` | Data evolution |
| 4 | Separate Observation from Intervention | `2026-01-11-inv-review-design-coaching-plugin-injection.md` | Behavioral monitoring |
| 5 | Registry Contract: Spawn-Cache Only | `2026-01-11-inv-registry-abandonment-workflow-validate-simple.md` | Component contract |
| 6 | Trust Calibration: Assert Relevant Knowledge | `2026-01-09-inv-trust-calibration-meta-pattern.md` | Human-AI collaboration |
| 7 | Understanding Lag Pattern | `2026-01-10-inv-verify-lagging-understanding-hypothesis-dec.md` | Observability |

---

## Findings

### Finding 1: Seven investigations recommend promotion to decisions

**Evidence:** Read and classified 37 investigations. 7 contain patterns that are architectural, cross-cutting, or foundational enough to warrant formal decision documents.

**Source:** Classification table above

**Significance:** These 7 patterns represent reusable knowledge that should be preserved for future sessions and team members.

---

### Finding 2: Twelve investigations already addressed

**Evidence:** 12 investigations describe work that was already implemented, bug fixes already merged, or decisions already documented in `.kb/decisions/`.

**Source:** Classification table - items marked ADDRESSED

**Significance:** These can be closed/archived without additional work. The "recommend-yes" flag was stale - work completed but promotion flag wasn't cleared.

---

### Finding 3: Eighteen investigations are tactical, not architectural

**Evidence:** 18 investigations describe specific bug fixes, feature implementations, meta-syntheses, or analyses that don't establish reusable patterns.

**Source:** Classification table - items marked SKIP

**Significance:** These provide historical context but don't warrant promotion. Investigation artifacts serve their purpose - decision promotion is for patterns worth preserving.

---

## Synthesis

**Key Insights:**

1. **Verification Bottleneck is the most important pattern** - Tested across 462 lost commits, three distinct triggers, same failure mode. This is foundational for human-AI collaboration and should be promoted first.

2. **Two-tier cleanup and schema migration are implementation patterns** - Both establish reusable approaches for common problems (resource management, data evolution). These are concrete, actionable patterns.

3. **Observation/Intervention separation and Trust Calibration are meta-patterns** - These address how the system should work at a higher level, affecting multiple components and workflows.

4. **Many "recommend-yes" flags were stale** - 12 investigations had work already completed but weren't marked as addressed. The kb reflect system correctly identified candidates but couldn't detect when promotion became moot.

**Answer to Investigation Question:**

Of 37 investigations flagged for promotion, **7 should be promoted to decisions** (architectural patterns worth preserving), **12 are already addressed** (work completed or documented elsewhere), and **18 should be skipped** (tactical fixes not decision-worthy). The highest priority decisions to create are Verification Bottleneck Principle and Two-Tier Cleanup Pattern, as these establish foundational constraints for the entire orch-go system.

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
