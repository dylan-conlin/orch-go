## Summary (D.E.K.N.)

**Delta:** session-start.sh was injecting session resume content to spawned workers/orchestrators who don't need it.

**Evidence:** Tested fix - CLAUDE_CONTEXT=worker/orchestrator exits immediately (2 lines); unset CLAUDE_CONTEXT runs full resume logic.

**Knowledge:** CLAUDE_CONTEXT env var (worker|orchestrator|meta-orchestrator) is the established pattern for spawn detection in Claude Code hooks.

**Next:** Merge change; continue with epic Phase 1 (dedup beads guidance, lazy orchestrator skill).

**Promote to Decision:** recommend-no (tactical fix applying existing pattern, not architectural)

---

# Investigation: Add CLAUDE_CONTEXT Check to session-start.sh

**Question:** How to prevent session-start.sh from injecting session resume to spawned agents?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker agent (og-feat-add-claude-context-16jan-909e)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: session-start.sh had no spawn detection

**Evidence:** Lines 8-24 of original session-start.sh ran session resume injection unconditionally:
```bash
if command -v orch >/dev/null 2>&1; then
  if orch session resume --check 2>/dev/null; then
    HANDOFF=$(orch session resume --for-injection 2>/dev/null)
    # ... outputs JSON with session handoff
  fi
fi
```

**Source:** `/Users/dylanconlin/.claude/hooks/session-start.sh:8-24`

**Significance:** Workers and orchestrators spawned via `orch spawn` were receiving session resume context they don't need. This adds ~1-4KB of irrelevant context and can cause confusion about which session to continue.

---

### Finding 2: load-orchestration-context.py has established pattern

**Evidence:** Lines 436-457 show the pattern:
```python
def is_spawned_agent():
    ctx = os.environ.get('CLAUDE_CONTEXT', '')
    return ctx in ('worker', 'orchestrator', 'meta-orchestrator')

def main():
    if is_spawned_agent():
        sys.exit(0)
```

**Source:** `/Users/dylanconlin/.orch/hooks/load-orchestration-context.py:436-457`

**Significance:** CLAUDE_CONTEXT env var is already the standard mechanism for spawn detection. Using the same pattern maintains consistency.

---

### Finding 3: Fix verified through testing

**Evidence:** Test results:
- `CLAUDE_CONTEXT=worker bash session-start.sh` → exits after case match (2 trace lines)
- `CLAUDE_CONTEXT=orchestrator bash session-start.sh` → exits after case match (2 trace lines)
- `unset CLAUDE_CONTEXT; bash session-start.sh` → runs full resume logic (5+ trace lines)

**Source:** Manual test via `bash -x session-start.sh`

**Significance:** Fix correctly discriminates between manual sessions (need resume) and spawned sessions (don't need resume).

---

## Synthesis

**Key Insights:**

1. **Pattern already exists** - CLAUDE_CONTEXT is the established spawn detection mechanism in the Claude Code hook ecosystem. No new patterns needed.

2. **Context waste eliminated** - Spawned agents no longer receive ~1-4KB of session resume content that was causing confusion.

3. **Simple fix** - 7-line case statement addition, following existing bash style in the hook.

**Answer to Investigation Question:**

Adding a case statement at the start of session-start.sh that checks `CLAUDE_CONTEXT` env var and exits for worker/orchestrator/meta-orchestrator values. This follows the exact pattern from load-orchestration-context.py.

---

## Structured Uncertainty

**What's tested:**

- ✅ Script exits immediately for CLAUDE_CONTEXT=worker (verified: bash -x trace shows 2 lines)
- ✅ Script exits immediately for CLAUDE_CONTEXT=orchestrator (verified: bash -x trace shows 2 lines)
- ✅ Script runs full resume logic when CLAUDE_CONTEXT unset (verified: bash -x trace shows resume check)

**What's untested:**

- ⚠️ Real spawned agent sessions will get correct behavior (manual test env, not real spawn)
- ⚠️ CLAUDE_CONTEXT=meta-orchestrator path (not tested, but follows same pattern)

**What would change this:**

- Finding would be wrong if CLAUDE_CONTEXT env var isn't set correctly by orch spawn (verified it is in spawn code)

---

## Implementation Recommendations

### Recommended Approach (IMPLEMENTED)

**Early exit via case statement** - Add CLAUDE_CONTEXT check at script start

**Why this approach:**
- Matches load-orchestration-context.py pattern exactly
- Uses bash case statement which is idiomatic for multi-value checks
- Zero execution cost for spawned agents (immediate exit)

**Implementation sequence:**
1. Add case statement after initial comments
2. Test with different CLAUDE_CONTEXT values
3. Commit change

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/hooks/session-start.sh` - Hook being modified
- `/Users/dylanconlin/.orch/hooks/load-orchestration-context.py` - Reference pattern

**Commands Run:**
```bash
# Test hook behavior
CLAUDE_CONTEXT=worker bash -x session-start.sh
CLAUDE_CONTEXT=orchestrator bash -x session-start.sh
unset CLAUDE_CONTEXT; bash -x session-start.sh
```

**Related Artifacts:**
- **Epic:** `.orch/epics/context-injection-architecture.md` - Phase 1 quick win
- **Investigation:** `.kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md` - Hook audit that identified this issue

---

## Investigation History

**2026-01-16 ~20:00:** Investigation started
- Initial question: How to add CLAUDE_CONTEXT check to session-start.sh
- Context: Epic identified this as Phase 1 quick win (~5 min fix)

**2026-01-16 ~20:10:** Fix implemented and tested
- Pattern copied from load-orchestration-context.py
- Verified via bash -x tracing

**2026-01-16 ~20:15:** Investigation completed
- Status: Complete
- Key outcome: 7-line fix eliminates session resume injection for spawned agents
