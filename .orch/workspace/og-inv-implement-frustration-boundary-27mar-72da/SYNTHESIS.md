# Session Synthesis

**Agent:** og-inv-implement-frustration-boundary-27mar-72da
**Issue:** orch-go-6kib0
**Duration:** 2026-03-27 → 2026-03-27
**Outcome:** success

---

## TLDR

Implemented the interactive track (Track 1) of the frustration detection → session boundary system: a UserPromptSubmit hook that pattern-matches user text for frustration signals and proposes session restart at threshold. All 36 tests pass. Hook registration requires one orchestrator command (sandbox blocks worker writes to settings.json).

---

## Plain-Language Summary

When Dylan gets frustrated during a Claude Code session — typing things like "this isn't working" or "we keep going in circles" — the system should notice and offer to save the question and start a fresh session. This works because mid-conversation reframing doesn't actually work (the attention patterns are baked in), but a new session gives a genuine cognitive reset.

The hook watches what Dylan types, counts frustration signals across three categories (explicit frustration, repeated corrections, abandon intent), and at 3+ signals injects a proposal into Claude's context suggesting a boundary. Dylan decides whether to act on it. If yes, a FRUSTRATION_BOUNDARY.md captures the question and what didn't work, so the next session starts with the question but not the broken conversation.

---

## Delta (What Changed)

### Files Created
- `.claude/hooks/frustration-boundary.sh` - UserPromptSubmit hook (51 patterns across 3 categories)
- `.claude/hooks/frustration-boundary_test.sh` - 36 tests covering detection, counters, thresholds, JSON validity
- `.orch/templates/FRUSTRATION_BOUNDARY.md` - Template for boundary artifact (question + diagnosis + fresh angle)

### Files Modified
- None (settings.json modification deferred to orchestrator)

---

## Evidence (What Was Observed)

- Hook follows comprehension-queue-count.sh pattern exactly — no new infrastructure needed
- 13 frustration patterns detected with 100% accuracy, 0 false positives on 5 clean messages
- Sandbox blocks worker writes to `.claude/settings.json` (EPERM on both Edit and cp)
- Counter scoping via tmux window name works correctly with emoji/bracket chars in window names

### Tests Run
```bash
bash .claude/hooks/frustration-boundary_test.sh
# Results: 36 passed, 0 failed
```

---

## Architectural Choices

### Pattern matching vs LLM analysis
- **What I chose:** Bash string matching against keyword arrays
- **What I rejected:** Using a prompt hook (LLM classification of frustration)
- **Why:** ~5ms latency vs ~500ms+ for LLM call, zero false positives in testing, follows design doc recommendation. LLM analysis is overkill for user text patterns.
- **Risk accepted:** May miss novel frustration expressions not in the pattern list. Mitigated by conservative threshold (3 signals).

### Counter scoping via tmux window name
- **What I chose:** Tmux window name for session isolation with 4-hour expiry
- **What I rejected:** PID-based tracking, session ID tracking
- **Why:** Tmux window naturally maps to interactive session scope. 4-hour expiry handles session turnover without needing explicit reset.
- **Risk accepted:** Multiple sessions in same tmux window share counter. Acceptable because the product philosophy is "surface signals, don't enforce."

---

## Knowledge (What Was Learned)

### Constraints Discovered
- `.claude/settings.json` is sandbox-protected from worker writes (not just governance-protected)

### Decisions Made
- Used `FRUSTRATION_WINDOW_NAME` env override for test isolation (testability > minimalism)
- 51 total patterns across 3 categories (explicit: 20, correction: 20, abandon: 11)

---

## Next (What Should Happen)

**Recommendation:** close (after orchestrator registers hook)

### Remaining Step
Orchestrator runs in direct session:
```bash
cat .claude/settings.json | jq '.hooks.UserPromptSubmit += [{"hooks": [{"command": "bash \"$CLAUDE_PROJECT_DIR\"/.claude/hooks/frustration-boundary.sh", "type": "command"}], "matcher": ""}]' > /tmp/s.json && mv /tmp/s.json .claude/settings.json
```

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — automated tests pass, two manual steps pending (hook registration + end-to-end).

---

## Unexplored Questions

- Real-world false positive rate (needs production observation)
- Session resume protocol discovery of FRUSTRATION_BOUNDARY.md (Track 1 design says this works but not verified end-to-end)
- Track 2 (headless workers via coaching plugin) — separate implementation scope

---

## Friction

- `ceremony`: Settings.json sandbox protection required reporting constraint and deferring registration to orchestrator. Net cost: ~5 minutes of investigation + one beads comment.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-implement-frustration-boundary-27mar-72da/`
**Investigation:** `.kb/investigations/2026-03-27-inv-implement-frustration-boundary-interactive-hook.md`
**Beads:** `bd show orch-go-6kib0`
