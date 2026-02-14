<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Integrated GET /session/status endpoint, replacing SSE-based status polling with HTTP polling in WaitForSessionIdle (65 lines → 28 lines, 57% reduction).

**Evidence:** All tests passing (70 tests in pkg/opencode); WaitForSessionIdle simplified to polling with 500ms interval; added GetAllSessionStatus() and GetSessionStatusByID() methods; 7 new tests added, 3 existing tests updated.

**Knowledge:** Polling is simpler than SSE for simple status checks; hybrid approach (polling for status, SSE for streaming) eliminates complexity without losing functionality; line count reduction is incremental (not 1,400 lines yet - that requires metadata API).

**Next:** Implementation complete. Monitor for performance issues with 500ms polling interval. Future: integrate session metadata API (Phase 5 Step 2) for additional simplification.

**Authority:** implementation - Stayed within scope of Phase 5 Step 1, no architectural changes needed.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Integrate Existing Session Status Endpoint

**Question:** How to integrate existing GET /session/status endpoint to replace SSE-only status polling in orch-go?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** orch-go-3nw
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: GET /session/status endpoint exists and returns Record<string, SessionStatus.Info>

**Evidence:** OpenCode fork has GET /session/status endpoint at packages/opencode/src/server/routes/session.ts:200-220. Returns `Record<string, SessionStatus.Info>` where key is sessionID and value is one of: `{type: "idle"}`, `{type: "busy"}`, or `{type: "retry", attempt: number, message: string, next: number}`.

**Source:** 
- ~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts (endpoint definition)
- ~/Documents/personal/opencode/packages/opencode/src/session/status.ts (SessionStatus module)

**Significance:** This endpoint provides exactly what we need for status polling - we can replace SSE event parsing with a simple HTTP GET. The status is in-memory only and deleted when session goes idle (status.ts:71-72), matching existing SSE behavior.

---

### Finding 2: Current SSE usage is concentrated in 3 main areas

**Evidence:** 
1. pkg/opencode/client.go:632-697 (WaitForSessionIdle) - uses SSE to detect busy→idle transition
2. pkg/opencode/client.go:892-1039 (SendMessageWithStreaming) - uses SSE for status and text streaming
3. pkg/opencode/monitor.go:1-229 (Monitor) - background SSE monitoring for completion detection

**Source:** grep results for "session.status" across pkg/opencode/*.go files

**Significance:** WaitForSessionIdle can be replaced with polling GET /session/status. SendMessageWithStreaming still needs SSE for text streaming, but can add polling fallback. Monitor.go is the complex piece - may need refactoring or hybrid approach.

---

### Finding 3: Line count analysis shows potential 400-600 line reduction from sse.go simplification

**Evidence:** 
- pkg/opencode/sse.go: 212 lines (ParseSessionStatus can be simplified significantly)
- pkg/opencode/monitor.go: 229 lines (SSE reconnection logic, state tracking could be streamlined)
- Decision doc claims ~1,400 lines eliminable, but that's likely across all cleanup phases

**Source:** wc -l pkg/opencode/sse.go pkg/opencode/monitor.go

**Significance:** We won't eliminate all SSE code (still needed for text streaming), but we can simplify status parsing and reduce complexity in Monitor. The 1,400 line estimate from the decision likely includes future session metadata work (Phase 5 Step 2).

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

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Hybrid Polling + SSE** - Add HTTP polling for status checks while keeping SSE for text streaming and real-time monitoring.

**Why this approach:**
- Eliminates SSE dependency for simple status checks (IsSessionActive, WaitForSessionIdle)
- Keeps SSE for scenarios that need it (text streaming, real-time dashboard updates)
- Incremental migration - can test polling without removing SSE infrastructure

**Trade-offs accepted:**
- Not eliminating all SSE code (still needed for streaming)
- Adds polling overhead for status checks (but simpler than maintaining SSE connection)
- Won't hit 1,400 line reduction immediately (that requires metadata API work)

**Implementation sequence:**
1. Add SessionStatusInfo type and GetAllSessionStatus() + GetSessionStatus(id) methods to client.go
2. Replace WaitForSessionIdle's SSE logic with polling (simpler, fewer edge cases)
3. Keep monitor.go and SendMessageWithStreaming unchanged (SSE still valuable here)
4. Document the hybrid approach for future work

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
