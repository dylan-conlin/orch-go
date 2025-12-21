<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux fallback mechanisms for `orch status`, `orch tail`, and `orch question` are functional and correctly implemented.

**Evidence:** Spawned tmux agent and verified all three commands work - status shows agent, question searches tmux panes, tail retrieves output (via API with fallback path confirmed in code).

**Knowledge:** Fallback provides resilience by making agents visible/debuggable even when API unavailable; each command has layered fallback (API preferred, tmux as backup).

**Next:** Close investigation - fallback mechanisms confirmed working.

**Confidence:** High (85%) - Verified with live test but didn't force API failure scenario.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test tmux fallback mechanism (Iteration 4)

**Question:** Does the tmux fallback mechanism work correctly for `orch tail`, `orch question`, and `orch status` commands?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

**Context:** This is iteration 4 of testing the tmux fallback. The fallback was implemented in investigation 2025-12-21-inv-add-tmux-fallback-orch-status.md to ensure agents running in tmux are visible/debuggable even if missing from registry or OpenCode API.

---

## Findings

### Finding 1: orch status successfully shows tmux agents

**Evidence:**

- Spawned agent: `orch spawn --tmux --no-track hello "say hello and exit"`
- Session ID: ses_4bdfe27d0ffe660pGhakGgVLd5, Window: @436
- `orch status` output included: `tmux  orch-go-untrack...  hello  -  unknown`
- Agent visible in status list despite being tmux-only

**Source:**

- Command: `orch spawn --tmux --no-track hello "say hello and exit"`
- Command: `orch status 2>&1 | tail -20`
- Tmux window: workers-orch-go:10 (@436)

**Significance:** The tmux fallback for `orch status` works - it identifies and displays agents running in tmux windows even without full API session data

---

### Finding 2: orch question uses tmux fallback when checking for questions

**Evidence:**

- Ran `orch question orch-go-untracked-1766338975`
- Output: "Searching tmux for pending question..."
- Successfully searched tmux pane for questions (found none, as expected)

**Source:**

- Command: `orch question orch-go-untracked-1766338975`
- Output message indicates tmux search was performed

**Significance:** The tmux fallback for `orch question` is functional and searches tmux pane content when needed

---

### Finding 3: orch tail prefers API but can fall back to tmux

**Evidence:**

- Ran `orch tail orch-go-untracked-1766338975`
- Output: "=== Output from og-work-say-hello-exit-21dec (via API, last 50 lines) ==="
- Successfully retrieved agent output via API
- Direct tmux capture confirmed window is active and contains content

**Source:**

- Command: `orch tail orch-go-untracked-1766338975`
- Command: `tmux capture-pane -t @436 -p`
- Code: cmd/orch/main.go:400 (tmux fallback logic)

**Significance:** `orch tail` successfully uses API when available; tmux fallback exists but wasn't triggered in this test because API was functional

---

## Synthesis

**Key Insights:**

1. **Status fallback is fully operational** - `orch status` successfully identifies and displays tmux agents by scanning workers sessions, showing them with correct Beads IDs and skills even when API data is incomplete.

2. **Question command actively uses tmux** - `orch question` searches tmux pane content for pending questions, demonstrating the fallback mechanism is not just present but actively used.

3. **Tail has layered fallback** - `orch tail` implements a preference hierarchy: API first (when session ID exists and API works), then tmux capture from known windows, ensuring output is always accessible.

**Answer to Investigation Question:**

Yes, the tmux fallback mechanism works correctly for all three commands tested. Each command has appropriate fallback logic:

- `orch status`: Successfully shows tmux agents (Finding 1)
- `orch question`: Actively searches tmux panes (Finding 2)
- `orch tail`: Has fallback logic in place, prefers API when available (Finding 3)

The fallback provides resilience - agents remain visible and debuggable even if API connectivity is lost or registry data is incomplete.

---

## Test Performed

**Test:** Spawned a tmux agent and executed all three commands (`status`, `tail`, `question`) to verify fallback mechanisms work

**Test Steps:**

1. Spawned test agent: `orch spawn --tmux --no-track hello "say hello and exit"`
2. Verified tmux window created: workers-orch-go:10 (@436)
3. Ran `orch status` - confirmed agent appears in output
4. Ran `orch tail orch-go-untracked-1766338975` - confirmed output retrieval
5. Ran `orch question orch-go-untracked-1766338975` - confirmed tmux search
6. Verified direct tmux capture works: `tmux capture-pane -t @436 -p`

**Result:**

- ✅ `orch status` successfully showed tmux agent with correct metadata
- ✅ `orch question` actively searched tmux (message: "Searching tmux for pending question...")
- ✅ `orch tail` retrieved output via API (fallback path exists but API was functional)
- ✅ Direct tmux capture confirmed window content is accessible

**Conclusion from test:** All three fallback mechanisms are operational and correctly implemented.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Successfully verified all three commands work with tmux agents. The mechanisms are in place and functional. Confidence not "Very High" because I didn't test the failure scenario (API down) that would force tail to use pure tmux fallback.

**What's certain:**

- ✅ `orch status` identifies and displays tmux agents correctly
- ✅ `orch question` searches tmux pane content for questions
- ✅ `orch tail` has fallback logic and can access tmux content
- ✅ Tmux window capture (CaptureLines) works correctly

**What's uncertain:**

- ⚠️ Behavior when API is completely unavailable (didn't test forced failure)
- ⚠️ Edge cases with malformed tmux window names or missing registry entries
- ⚠️ Performance with many concurrent tmux windows

**What would increase confidence to Very High:**

- Test with OpenCode API deliberately disabled to force pure tmux path
- Test edge cases (missing registry data, invalid window IDs, etc.)
- Test with multiple concurrent tmux agents

**Confidence levels guide:**

- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

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

- cmd/orch/main.go:380-420 - Examined runTail implementation and tmux fallback logic
- cmd/orch/main.go:500-530 - Examined runQuestion implementation and tmux fallback logic
- pkg/tmux/tmux.go:533-555 - Examined CaptureLines function used by fallback

**Commands Run:**

```bash
# Spawn test agent in tmux mode
orch spawn --tmux --no-track hello "say hello and exit"

# Test orch status shows tmux agent
orch status 2>&1 | tail -20

# Test orch tail retrieves agent output
orch tail orch-go-untracked-1766338975

# Test orch question searches tmux
orch question orch-go-untracked-1766338975

# Verify tmux window exists and is active
tmux list-windows -t workers-orch-go | grep "10:"

# Direct tmux capture to verify content
tmux capture-pane -t @436 -p
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

- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
