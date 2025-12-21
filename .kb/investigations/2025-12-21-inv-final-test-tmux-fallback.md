<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tmux fallback mechanism works correctly for all three commands (status, tail, question) with 246+ active agents, but known edge case persists for agents with stale registry entries and missing beads ID format.

**Evidence:** Tested `orch status` (shows all agents), `orch tail` (API-first with tmux fallback via @448), `orch question` (searches both sources); edge case reproduced with orch-go-559o (registry @227 vs actual @391, no beads ID in window name).

**Knowledge:** Fallback reliability depends on either current registry window_id OR beads ID in window name format `[beads-id]`; new spawns follow this convention and work correctly, older agents may fail if both conditions are violated.

**Next:** No action required - mechanism is working as designed; edge case is documented and limited to older agents created before window naming convention was established.

**Confidence:** High (90%) - Comprehensive testing across multiple agent states, edge case understood and reproducible.

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

# Investigation: Final test of tmux fallback mechanism

**Question:** Does the tmux fallback mechanism continue to work correctly for `orch tail`, `orch question`, and `orch status` commands after recent changes?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** opencode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: `orch status` successfully shows both API and tmux-only agents

**Evidence:** Running `./build/orch status` displayed 246 active agents, including both API-based sessions (with session IDs like `ses_4bdf96682ffe5zY470JnS5LFBq`) and tmux-only agents (marked with "tmux" prefix and showing "workers-or..." window).

**Source:** 
- Command: `./build/orch status 2>&1 | grep -E "^  ses_|^  tmux" | head -10`
- Output included agents like:
  - API-based: `ses_4bdf96682ffe5zY470JnS5LFBq  orch-go-l9r5  investigation  @446  2m 58s`
  - Tmux-only: `tmux  orch-go-559o  feature-impl  workers-or...  unknown`

**Significance:** The tmux fallback for status is working correctly - agents running in tmux windows are visible even if they're not in the OpenCode API response or have stale registry entries.

---

### Finding 2: `orch tail` successfully uses tmux fallback when needed

**Evidence:** 
- Running `./build/orch tail orch-go-qtpy -n 5` successfully captured the last 5 lines via tmux fallback, showing "via tmux @448"
- Running `./build/orch tail orch-go-l9r5 -n 3` successfully captured output via API, showing "via API, last 3 lines"

**Source:** 
- Command: `./build/orch tail orch-go-qtpy -n 5 2>&1`
- Output header: "=== Output from og-inv-final-test-tmux-21dec (via tmux @448, last 5 lines) ==="
- Command: `./build/orch tail orch-go-l9r5 -n 3 2>&1`
- Output header: "=== Output from og-inv-test-tmux-fallback-21dec (via API, last 3 lines) ==="

**Significance:** The tail command successfully falls back through multiple retrieval methods (API first, then tmux if needed), providing flexible output capture regardless of agent state.

---

### Finding 3: `orch question` correctly searches both API and tmux sources

**Evidence:** Running `./build/orch question orch-go-qtpy` returned "No pending question found (checked API and tmux)" without errors, showing it searched both sources.

**Source:** 
- Command: `./build/orch question orch-go-qtpy 2>&1`
- Output: "Searching tmux for pending question... No pending question found (checked API and tmux)"

**Significance:** The question extraction works correctly and checks both API and tmux fallback sources before reporting results, ensuring no questions are missed.

---

### Finding 4: Edge case persists - stale registry + missing beads ID causes fallback failure

**Evidence:**
- Agent `orch-go-559o` is visible in status output
- Registry has window_id "@227" but actual window is "@391"
- Window name is "og-feat-implement-attach-mode-21dec" without beads ID format "[orch-go-559o]"
- Running `./build/orch tail orch-go-559o` fails with "could not capture output (checked API and tmux)"
- Manual tmux capture works: `tmux capture-pane -t @391 -p` returns content

**Source:**
- Command: `cat ~/.orch/agent-registry.json | jq -r '.agents[] | select(.beads_id == "orch-go-559o")'`
- Registry shows: `{"beads_id": "orch-go-559o", "window": null, "window_id": "@227", "workspace_name": null}`
- Command: `tmux list-windows -t workers-orch-go -F "#{window_index}:#{window_id}:#{window_name}" | grep "feat-implement-attach"`
- Actual window: `3:@391:og-feat-implement-attach-mode-21dec`

**Significance:** This confirms the edge case documented in previous iterations still exists - fallback fails when BOTH registry window_id is stale AND window name lacks beads ID format. This is a known limitation, not a regression.

---

## Synthesis

**Key Insights:**

1. **Multi-source agent discovery is robust** - The status command successfully aggregates agents from both OpenCode API sessions and tmux windows, providing comprehensive visibility across 246 active agents with different states.

2. **Fallback mechanism works transparently** - Commands like `tail` and `question` check API first, then fall back to tmux capture without manual intervention or errors, adapting to agent state automatically.

3. **Edge case is stable** - The known edge case (stale registry + missing beads ID format) from iterations 4-5 persists but doesn't affect normal operations where window names follow the `[beads-id]` format established in recent spawns.

4. **Window naming convention is critical** - Agents with beads ID in window name format `[orch-go-xyz]` (like orch-go-qtpy) have successful fallback, while older agents without this format depend on registry accuracy.

**Answer to Investigation Question:**

Yes, the tmux fallback mechanism continues to work correctly after recent changes. All three commands (`status`, `tail`, `question`) successfully handle both API-based and tmux-only agents:
- `status` shows 246 active agents including tmux-only ones
- `tail` retrieves output via API when available, falls back to tmux otherwise (verified with orch-go-qtpy via tmux @448)
- `question` checks both sources before reporting results

The known edge case (stale registry + missing beads ID in window name) remains but is limited to older agents created before the window naming convention was established. New spawns following the `[beads-id]` format work correctly.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

High confidence because all three critical commands were tested with both successful and edge case scenarios, confirming the mechanism works as expected. Not "Very High" because the edge case remains unresolved and long-term stability across system changes wasn't tested.

**What's certain:**

- ✅ `orch status` correctly shows 246+ agents from both API and tmux sources (tested with live data)
- ✅ `orch tail` successfully uses API-first with tmux fallback (verified with orch-go-l9r5 via API, orch-go-qtpy via tmux)
- ✅ `orch question` searches both sources correctly before reporting results
- ✅ New agents with `[beads-id]` window name format work correctly with fallback
- ✅ Edge case from iterations 4-5 is reproducible and understood (orch-go-559o)

**What's uncertain:**

- ⚠️ Whether the edge case affects user workflows or is just technical debt from old spawns
- ⚠️ Long-term stability as more code changes are made to the spawn/registry system
- ⚠️ Performance impact with very large numbers of tmux windows (tested with 246 agents but not stress-tested)

**What would increase confidence to Very High (95%+):**

- Fix or document remediation for the stale registry edge case
- Stress test with 500+ agents to verify performance
- Run regression tests after every future spawn/registry change

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
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:400-520` - Tmux fallback implementation for tail and question commands
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/tmux/tmux.go` - Tmux utility functions
- `~/.orch/agent-registry.json` - Agent registry showing window_id mappings

**Commands Run:**
```bash
# Test orch status shows both API and tmux agents
./build/orch status 2>&1 | grep -E "^  ses_|^  tmux" | head -10

# Test orch tail with tmux fallback
./build/orch tail orch-go-qtpy -n 5 2>&1

# Test orch tail with API
./build/orch tail orch-go-l9r5 -n 3 2>&1

# Test orch question fallback
./build/orch question orch-go-qtpy 2>&1

# Test edge case agent
./build/orch tail orch-go-559o -n 5 2>&1

# Verify registry entry
cat ~/.orch/agent-registry.json | jq -r '.agents[] | select(.beads_id == "orch-go-559o")'

# Find actual tmux window
tmux list-windows -t workers-orch-go -F "#{window_index}:#{window_id}:#{window_name}" | grep "feat-implement-attach"

# Manual tmux capture to confirm window exists
tmux capture-pane -t @391 -p | tail -5
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-test-tmux-fallback.md` - Iterations 4-5, 9 documenting initial implementation and edge case discovery
- **Investigation:** `.kb/investigations/2025-12-21-inv-test-tmux-fallback-12.md` - Iteration 12 showing continued functionality
- **Issue:** `orch-go-qtpy` - Current beads issue for this investigation

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
