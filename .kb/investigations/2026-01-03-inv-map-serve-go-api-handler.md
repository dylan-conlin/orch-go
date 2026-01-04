<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [To be filled at completion]

**Evidence:** [To be filled at completion]

**Knowledge:** [To be filled at completion]

**Next:** [To be filled at completion]

---

# Investigation: Map Serve Go Api Handler

**Question:** How should serve.go (2921 lines) be split into handler groupings with shared middleware/utilities, and what phases should the refactoring follow?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Agent (og-inv-map-serve-go-03jan)
**Phase:** Investigating
**Next Step:** Categorize handlers and map dependencies
**Status:** In Progress

---

## Findings

### Finding 1: serve.go is 2921 lines (not 4125 as initially stated)

**Evidence:** `wc -l` shows serve.go is 2921 lines. Still substantial, but smaller than expected.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go`

**Significance:** Slightly reduces scope of refactoring. Based on main.go learnings (~500-800 lines per phase), this is roughly 4-5 phases of work.

---

### Finding 2: Initial handler groupings identified

**Evidence:** From analyzing the file, handlers fall into these logical groupings:

1. **Agents/Sessions** (~440 lines: 560-999)
   - `handleAgents` - Core agent list with OpenCode integration
   - `handleEvents` - SSE event proxy
   - `handleAgentlog`, `handleAgentlogSSE`, `handleAgentlogJSON` - Agent lifecycle events
   - Helper: `workspaceCache`, `buildWorkspaceCache`, `buildMultiProjectWorkspaceCache`

2. **Beads** (~230 lines: 1392-1622)
   - `handleBeads` - Stats endpoint
   - `handleBeadsReady` - Ready issues queue
   - `handleIssues` - Create issue POST endpoint
   - Related types: `BeadsAPIResponse`, `BeadsReadyAPIResponse`, `ReadyIssueResponse`, `CreateIssueRequest`, `CreateIssueResponse`

3. **Usage/Focus/Config** (~250 lines: 1268-1391, 2786-2882)
   - `handleUsage` - Claude Max usage stats
   - `handleFocus` - Focus/drift status
   - `handleConfig`, `handleConfigGet`, `handleConfigPut` - User config CRUD
   - Helper: `lookupAccountName`

4. **Servers/Daemon** (~180 lines: 1523-1700)
   - `handleServers` - Project server status
   - `handleDaemon` - Daemon status
   - Helper: `formatDurationAgo`
   - Related types: `ServerPortInfo`, `ServerProjectInfo`, `ServersAPIResponse`, `DaemonAPIResponse`

5. **Gaps/Reflect/Learn** (~270 lines: 1800-2110)
   - `handleGaps` - Gap tracker stats
   - `handleReflect` - Reflect suggestions
   - `getGapAnalysisFromEvents`, `extractGapAnalysisFromEvent`
   - Related types: `GapsAPIResponse`, `GapSuggestionSummary`, `GapAPIResponse`, `ReflectAPIResponse`, etc.

6. **Errors** (~170 lines: 2112-2381)
   - `handleErrors` - Error pattern analysis
   - Helpers: `extractSkillFromAgentID`, `normalizeErrorMessage`, `suggestRemediation`, `containsString`
   - Related types: `ErrorEvent`, `ErrorPattern`, `ErrorsAPIResponse`

7. **Pending Reviews** (~400 lines: 2382-2784)
   - `handlePendingReviews` - Synthesis review queue
   - `handleDismissReview` - Dismiss recommendations
   - Helpers: `isLightTierWorkspace`, `isLightTierComplete`, `contains`
   - Related types: `PendingReviewItem`, `PendingReviewAgent`, `PendingReviewsAPIResponse`, `DismissReviewRequest`, `DismissReviewResponse`

8. **Changelog** (~40 lines: 2884-2921)
   - `handleChangelog` - Aggregated changelog
   - Defers to `GetChangelog()` in changelog.go

9. **Server Setup/Shared** (~560 lines: 1-560)
   - `runServe`, `runServeStatus` - Server startup
   - `corsHandler` - CORS middleware
   - Route registration
   - Response types: `AgentAPIResponse`, `SynthesisResponse`
   - Shared helpers: `checkWorkspaceSynthesis`, `extractDateFromWorkspaceName` (imported from status_cmd.go)

**Source:** Line-by-line analysis of `cmd/orch/serve.go:1-2921`

**Significance:** These 9 groupings form natural file boundaries. Some helpers are already external (in shared.go, review.go, wait.go).

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
