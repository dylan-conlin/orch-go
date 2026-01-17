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

# Investigation: Analyze Orch Status Observability Gaps

**Question:** What observability gaps exist in `orch status` for tracking escape hatch spawns vs normal spawns, and how can we improve visibility for critical infrastructure work?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-inv-analyze-orch-status-17jan-cf9d
**Phase:** Investigating
**Next Step:** Explore how escape hatch spawns are tracked vs normal spawns
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Mode field exists and is displayed in orch status

**Evidence:** 
- Registry stores `Mode` field: "claude" (escape hatch/tmux) or "opencode" (headless API) (`pkg/registry/registry.go:47`)
- `orch status` includes MODE column in wide format output (`cmd/orch/status_cmd.go:1049, 1052`)
- MODE displayed alongside MODEL, STATUS, PHASE for each agent
- JSON output includes `"mode"` field in AgentInfo struct (`status_cmd.go:94`)

**Source:** 
- `pkg/registry/registry.go:33-36, 47` - Mode constants and Agent struct
- `cmd/orch/status_cmd.go:1049-1110` - Wide format display with MODE column
- Terminal test: `orch status` shows column headers including MODE

**Significance:** Basic tracking infrastructure exists - mode is captured at spawn time and displayed in status output. This provides visibility into which backend was used for each agent.

---

### Finding 2: Escape hatch statistics are tracked and displayed

**Evidence:**
- `orch stats` includes dedicated "ESCAPE HATCH" section showing:
  - Total spawns: 165 (all time)
  - Last 7 days: 147 spawns
  - Last 30 days: 165 spawns
  - Escape hatch rate: 50.0% of spawns
- Stats identify escape hatch by checking `spawn_mode = "claude"` in spawn events
- Infrastructure detection events logged: `spawn.infrastructure_detected` type

**Source:**
- `cmd/orch/stats_cmd.go:171-179` - EscapeHatchStats struct definition
- `cmd/orch/stats_cmd.go:322-379` - Escape hatch tracking logic
- Terminal test: `orch stats` output shows escape hatch metrics
- Event log: `~/.orch/events.jsonl` contains `spawn.infrastructure_detected` events

**Significance:** Aggregate visibility exists - users can see escape hatch usage patterns over time. This helps track when the escape hatch is being used frequently (potential signal of infrastructure instability).

---

### Finding 3: Infrastructure work auto-detection applies escape hatch

**Evidence:**
- Auto-detection logic at spawn time checks if task/beads ID indicates infrastructure work
- When detected, automatically applies `--backend claude --tmux` flags
- User sees message: "🔧 Infrastructure work detected - auto-applying escape hatch"
- Infrastructure detection event logged with task, beads_id, skill metadata

**Source:**
- `cmd/orch/spawn_cmd.go:1115-1136` - `isInfrastructureWork()` check and escape hatch application
- Logged event type: `spawn.infrastructure_detected` with structured metadata
- Auto-application happens after explicit flags but before config defaults

**Significance:** System proactively applies escape hatch for critical work, reducing the chance of agents killing themselves while fixing infrastructure. This automation increases resilience without requiring users to remember the flags.

---

### Finding 4: Narrow format omits MODE column (observability gap)

**Evidence:**
- Wide format (>120 chars): Shows SOURCE, BEADS ID, MODE, MODEL, STATUS, PHASE, TASK, SKILL, RUNTIME, TOKENS
- Narrow format (80-100 chars): Shows SOURCE, BEADS ID, MODEL, STATUS, PHASE, SKILL, RUNTIME, TOKENS
- MODE column dropped in narrow format to fit smaller terminals
- No explicit filter for `--mode` or `--backend` in status command

**Source:**
- `cmd/orch/status_cmd.go:1049` - Wide format header with MODE
- `cmd/orch/status_cmd.go:1132` - Narrow format header WITHOUT MODE
- `orch status --help` - No mode/backend filter flag documented

**Significance:** Users on smaller terminals or when condensed output is needed cannot see which spawn backend was used. This reduces visibility into escape hatch usage at the individual agent level. There's also no way to filter `orch status` to show only escape hatch spawns.

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
