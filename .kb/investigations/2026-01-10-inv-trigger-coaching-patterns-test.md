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

---

# Investigation: Trigger Coaching Patterns Test

**Question:** Do the coaching plugin patterns trigger correctly when appropriate behavioral patterns occur?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent og-inv-trigger-coaching-patterns-10jan-87fd
**Phase:** Investigating
**Next Step:** Test each coaching pattern detection mechanism
**Status:** In Progress

---

## Findings

### Finding 1: Understanding the Coaching Plugin Architecture

**Evidence:** Read `/Users/dylanconlin/.config/opencode/plugin/coaching.ts` - 1139 lines implementing behavioral pattern detection

**Source:** coaching.ts:1-1139

**Significance:** The plugin tracks three categories of patterns:
1. **Behavioral Variation** (Phase 1) - 3+ consecutive attempts in same semantic group without 30s pause
2. **Circular Patterns** (Phase 2) - decisions contradicting prior investigation recommendations
3. **Dylan Patterns** (Phase 3.5) - explicit prefixes, priority uncertainty, compensation patterns

All patterns write to `~/.orch/coaching-metrics.jsonl` and can stream to a coach session via `ORCH_COACH_SESSION_ID`.

---

### Finding 2: Semantic Command Classification

**Evidence:** Plugin classifies bash commands into semantic groups (coaching.ts:101-164):
- `process_mgmt`: overmind, tmux, launchd, launchctl, systemctl
- `git`: git commands
- `build`: make, go build, npm, bun
- `test`: go test, npm test, jest, pytest
- `knowledge`: kb, bd commands
- `orch`: orch spawn, orch status
- `file_ops`: ls, cat, mkdir, cp, mv, rm, find, rg
- `network`: curl, wget, nc, ssh, http
- `other`: fallback

**Source:** SEMANTIC_PATTERNS array in coaching.ts:117-164

**Significance:** Commands are grouped semantically so the plugin can detect when an orchestrator keeps retrying variations of the same operation (e.g., trying multiple process managers). This is the foundation for behavioral variation detection.

---

### Finding 3: Behavioral Variation Threshold

**Evidence:** Plugin triggers `behavioral_variation` metric when:
- 3+ consecutive commands in same semantic group (VARIATION_THRESHOLD = 3)
- No strategic pause (30s+ without tools)
- Tracks last 20 commands in variation history

**Source:** coaching.ts:56, 1041-1068

**Significance:** This detects analysis paralysis or thrashing - when orchestrator tries multiple variations without stepping back to think strategically.

---

## Test Plan

**Test Strategy:** Since coaching.ts is an OpenCode plugin that hooks into tool execution, I'll test the pattern detection logic directly by:

1. Verifying the classification functions work correctly
2. Testing the detection thresholds
3. Checking metric writing behavior
4. Validating the streaming mechanism (if coach session exists)

**Tests to Run:**

1. **Semantic Classification Test** - verify bash commands are classified into correct groups
2. **Behavioral Variation Test** - trigger 3+ process_mgmt commands without pause
3. **Dylan Signal Prefix Test** - detect explicit prefixes (frame-collapse:, compensation:, etc.)
4. **Priority Uncertainty Test** - detect "what's next?" type questions
5. **Circular Pattern Test** - detect decisions contradicting investigation recommendations

---

## Test Performed

**Test 1: Verify Coaching Plugin Is Installed**

```bash
ls -la ~/.config/opencode/plugin/coaching.ts
```
