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

# Investigation: Pivot Coaching Plugin Two Frame

**Question:** How should we pivot the coaching plugin from passive dashboard metrics to active AI injection + simplified health indicator?

**Started:** 2026-01-11
**Updated:** 2026-01-11
**Owner:** Agent og-feat-pivot-coaching-plugin-11jan-be2c
**Phase:** Investigating
**Next Step:** Document current implementation findings
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Current Plugin Architecture (Metrics-Only)

**Evidence:** 
- Plugin tracks metrics via tool.execute.after hook: reads, actions, context_checks, spawns
- Calculates action_ratio, context_ratio, analysis_paralysis metrics
- Writes to ~/.orch/coaching-metrics.jsonl (append-only, pruned to 1000 lines)
- NO active intervention - only writes data for dashboard consumption

**Source:** 
- plugins/coaching.ts:1019-1220 (tool.execute.after hook)
- plugins/coaching.ts:489-538 (flushMetrics function)
- plugins/coaching.ts:391-399 (writeMetric function)

**Significance:** Current design is passive observation only. To implement Frame 1 (AI injection), we need to add client.session.prompt() calls when patterns detected, not just metric writing.

---

### Finding 2: Reference Pattern for AI Injection

**Evidence:**
- agentlog-inject.ts demonstrates the injection pattern:
  - Hooks on session.created event
  - Uses client.session.prompt({ path: { id: sessionId }, body: { noReply: true, parts: [...] } })
  - Returns immediately without blocking user interaction
- This is the exact pattern needed for Frame 1

**Source:** ~/.config/opencode/plugin/agentlog-inject.ts:118-131

**Significance:** We have a working reference implementation. Can adapt flushMetrics() to inject coaching messages using this exact pattern when thresholds exceeded.

---

### Finding 3: Dashboard Shows Detailed Metrics Grid

**Evidence:**
- API returns full metrics object with value/label/status for each metric type
- UI renders 3-column grid showing action_ratio, context_ratio, analysis_paralysis
- Each metric shows numeric value with color coding (green/yellow/red)
- Coaching messages shown as bullet list below metrics
- 25+ lines of Svelte markup for rendering

**Source:**
- serve_coaching.go:30-36 (API response structure)
- web/src/routes/+page.svelte:410-468 (coaching section UI)
- web/src/lib/stores/coaching.ts:4-20 (data model)

**Significance:** Frame 2 requires simplifying this to single health indicator. Can replace entire metrics grid with one emoji + status line + timestamp.

---

## Synthesis

**Key Insights:**

1. **Plugin already has detection logic, needs injection capability** - The coaching plugin correctly detects patterns (low action_ratio, high analysis_paralysis) but only writes metrics to JSONL. Adding client.session.prompt() calls to flushMetrics() will enable real-time AI coaching.

2. **agentlog-inject.ts provides the exact pattern needed** - The reference implementation shows how to inject messages with noReply:true, preventing blocking while still surfacing coaching to the orchestrator.

3. **Dashboard complexity can be dramatically reduced** - Current UI shows 3 detailed metrics + coaching list. Frame 2 design collapses this to single health indicator (🟢 good / 🟡 warning / 🔴 poor) with optional timestamp, reducing cognitive load.

**Answer to Investigation Question:**

The pivot requires two parallel changes: (1) Add client.session.prompt() injection to plugins/coaching.ts flushMetrics() function using agentlog-inject.ts pattern, (2) Replace dashboard metrics grid with single health indicator derived from aggregated status. Both changes are straightforward - plugin already has pattern detection, dashboard already has data structure with status field. Main implementation risk is ensuring injection doesn't fire for worker sessions (already has detectWorkerSession logic).

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

**Parallel Implementation of Frame 1 (Injection) and Frame 2 (Simplified UI)** - Implement AI coaching injection in plugin first, then simplify dashboard UI to match new "health indicator" mental model.

**Why this approach:**
- Frame 1 is higher value (active intervention vs passive display) so implement first
- Frame 2 depends on understanding what coaching looks like in practice
- Reference pattern (agentlog-inject.ts) de-risks Frame 1 implementation
- Simplified UI reduces maintenance burden and cognitive load

**Trade-offs accepted:**
- Dashboard will temporarily show old metrics UI while injection is being built
- Losing detailed metric breakdown (but that's the point - signal over noise)
- No A/B testing of effectiveness (building on hypothesis that active coaching > passive metrics)

**Implementation sequence:**
1. **Add injection to flushMetrics()** - Use client.session.prompt() pattern when thresholds exceeded (action_ratio < 0.5, analysis_paralysis >= 3)
2. **Test injection with debug session** - Verify messages appear, don't block, skip workers
3. **Simplify dashboard** - Replace metrics grid with single health indicator + last coaching timestamp
4. **Simplify API response** - Collapse serve_coaching.go to return overall_status + last_coaching_time instead of detailed metrics
5. **Sync source and deployed plugin** - Keep plugins/coaching.ts and ~/.config/opencode/plugin/coaching.ts identical

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
- Add injection logic to flushMetrics() - highest value change
- Inject message format: "You've done 6 reads without acting. Consider spawning an agent instead of investigating yourself."
- Use noReply:true to avoid blocking orchestrator workflow

**Things to watch out for:**
- ⚠️ Ensure injection only fires for orchestrator sessions (detectWorkerSession check already exists)
- ⚠️ Avoid injection loops if orchestrator is reading coaching messages (check if action is reading coaching context)
- ⚠️ Message formatting needs to be actionable, not just "low action ratio detected"
- ⚠️ Dashboard polling at 30s intervals - ensure timestamp shows recency accurately

**Areas needing further investigation:**
- None identified - implementation is straightforward based on existing patterns

**Success criteria:**
- ✅ Orchestrator receives coaching message in session when pattern detected (test: trigger low action_ratio)
- ✅ Dashboard shows single health indicator instead of metrics grid
- ✅ Worker sessions do NOT receive coaching injections
- ✅ Message includes actionable recommendation (not just metric values)

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
