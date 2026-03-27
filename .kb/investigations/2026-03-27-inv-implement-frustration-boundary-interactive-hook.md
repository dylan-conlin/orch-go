## Summary (D.E.K.N.)

**Delta:** Implemented Track 1 (interactive) of the frustration detection → session boundary system: a UserPromptSubmit hook that detects frustration signals in user text and proposes session restart at threshold.

**Evidence:** 36/36 tests passing — signal detection (explicit frustration, corrections, abandon intent), counter accumulation, threshold behavior, JSON output validity, case insensitivity, worker skip, env overrides.

**Knowledge:** `.claude/settings.json` is sandbox-protected from worker writes. Hook registration must be done by orchestrator in a direct session. The hook itself, template, and tests are all committed and ready.

**Next:** Orchestrator registers hook in `.claude/settings.json` (one jq command provided in beads comment).

**Authority:** implementation — Hook follows existing comprehension-queue-count.sh pattern exactly, no architectural decisions.

---

# Investigation: Implement Frustration Boundary Interactive Hook

**Question:** Can we implement a UserPromptSubmit hook that detects frustration signals in user text and proposes a session boundary with question carryforward?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None — orchestrator registers hook
**Status:** Complete
**Model:** orchestrator-session-lifecycle

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md | implements | yes | none |
| .kb/models/orchestrator-session-lifecycle/probes/2026-03-27-probe-frustration-detection-session-boundary-design.md | implements | yes | none |

---

## Findings

### Finding 1: Hook follows comprehension-queue-count.sh pattern exactly

**Evidence:** Both hooks use identical structure — worker skip via SPAWN_CONTEXT.md check, stdin message reading, JSON output with `hookSpecificOutput.additionalContext`, registration in `.claude/settings.json` UserPromptSubmit array.

**Source:** `.claude/hooks/comprehension-queue-count.sh` (reference implementation), `.claude/hooks/frustration-boundary.sh` (new hook)

**Significance:** No new patterns needed. The hook infrastructure already supports everything the design requires.

### Finding 2: Three frustration signal categories provide good coverage without false positives

**Evidence:** Testing against 5 clean messages (feature requests, approvals, questions, compliments) — zero false positives. Testing against 13 frustration signals across three categories — 100% detection. Categories: explicit frustration (20 patterns), repeated correction (20 patterns), session abandon intent (11 patterns).

**Source:** Test group 4 (detection), group 5 (non-frustration), group 10 (case insensitivity)

**Significance:** The keyword approach is simple but effective for interactive sessions. The design doc recommended compound signals for headless workers (Track 2), but for user text analysis, direct pattern matching is sufficient.

### Finding 3: Sandbox prevents worker modification of .claude/settings.json

**Evidence:** Both `cp` (via Bash) and Edit tool return EPERM when attempting to write to `.claude/settings.json` from a worker session. This is a built-in sandbox protection, not the governance hook (which only protects `.orch/hooks/`, `pkg/spawn/gates/`, etc.).

**Source:** Direct observation during implementation

**Significance:** Hook registration is a one-line orchestrator task. The hook script, template, and tests are all committed and ready to use.

---

## Synthesis

**Key Insights:**

1. **Pattern matching over LLM analysis** — Simple keyword matching with bash string comparison is fast (~5ms), has zero false positives in testing, and doesn't require an LLM call. The design doc's pattern list maps directly to bash arrays.

2. **Counter scoping via tmux window name** — Session isolation comes naturally from tmux window names. 4-hour stale file expiry prevents cross-session contamination without needing a session ID.

3. **Proposal, not enforcement** — The hook only injects additionalContext suggesting a boundary. Claude decides whether to surface it. Dylan decides whether to act. This matches the product surface philosophy: surface signals, don't enforce.

**Answer to Investigation Question:**

Yes, implemented. The hook detects frustration via three signal categories (explicit, correction, abandon), tracks count per tmux window, and proposes a boundary at threshold (default 3). The FRUSTRATION_BOUNDARY.md template provides the artifact structure for question carryforward. All 36 tests pass.

---

## Structured Uncertainty

**What's tested:**

- ✅ Signal detection across all three categories (13 patterns verified)
- ✅ Zero false positives on 5 clean message types
- ✅ Counter accumulation and threshold behavior
- ✅ Case insensitivity
- ✅ Worker session skip
- ✅ JSON output validity and structure
- ✅ Custom threshold override
- ✅ Disable via SKIP_FRUSTRATION_BOUNDARY

**What's untested:**

- ⚠️ 4-hour counter expiry (requires waiting or mocking system time)
- ⚠️ Real-world false positive rate (no production data yet)
- ⚠️ Session resume protocol discovery of FRUSTRATION_BOUNDARY.md (Track 1 design says this already works via `.orch/session/{window}/` but not verified end-to-end)
- ⚠️ Claude's response quality when boundary proposal is injected (does it surface the question well?)

**What would change this:**

- High false positive rate in production would require tightening patterns or adding compound signal requirement
- If Claude ignores the additionalContext, would need a stronger injection mechanism

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Register hook in settings.json | implementation | One-line config change, follows existing pattern |
| Monitor false positive rate | implementation | First-week observation, no architectural impact |

### Recommended Approach ⭐

**Register hook and observe** — Add the hook to settings.json, use for a week, tune patterns based on observed false positives.

**Implementation sequence:**
1. Orchestrator runs jq command to register hook in `.claude/settings.json`
2. Use naturally for 1 week
3. Tune threshold and patterns based on real usage

---

## References

**Files Created:**
- `.claude/hooks/frustration-boundary.sh` - UserPromptSubmit hook (detection + proposal)
- `.claude/hooks/frustration-boundary_test.sh` - 36 tests covering all behavior
- `.orch/templates/FRUSTRATION_BOUNDARY.md` - Artifact template for boundary handoff

**Files Examined:**
- `.claude/hooks/comprehension-queue-count.sh` - Reference implementation for hook pattern
- `.claude/settings.json` - Hook registration target
- `.kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md` - Design doc (Track 1)

**Commands Run:**
```bash
# Test suite
bash .claude/hooks/frustration-boundary_test.sh
# Result: 36 passed, 0 failed
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md` - Design doc
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-03-27-probe-frustration-detection-session-boundary-design.md` - Model validation

---

## Investigation History

**2026-03-27:** Investigation started
- Initial question: Can we implement the interactive track of the frustration detection → session boundary system?
- Context: Design doc completed, spawned for Track 1 implementation

**2026-03-27:** Implementation complete
- Status: Complete
- Key outcome: Hook, template, and tests all working. Settings registration deferred to orchestrator (sandbox protection).
