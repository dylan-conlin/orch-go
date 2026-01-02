## Summary (D.E.K.N.)

**Delta:** Fixed agent phase reporting order - agents now report Phase: Complete BEFORE their final commit, preventing race condition where agent dies after commit but before phase report.

**Evidence:** Updated SpawnContextTemplate in orch-go (3 locations), and 6 skill completion templates in orch-knowledge. All tests pass.

**Knowledge:** The completion order matters: Phase report → Commit → Exit. If agent dies between commit and phase report, orchestrator cannot detect completion.

**Next:** Close - implementation complete, tests passing.

---

# Investigation: Fix Agent Phase Reporting Order

**Question:** How to fix the race condition where agents report Phase: Complete AFTER their final commit, causing agents to die after commit but before phase report?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** og-feat-fix-agent-phase-02jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current completion order causes race condition

**Evidence:** All skill templates and SPAWN_CONTEXT say "After your final commit, BEFORE typing anything else: report Phase: Complete". This means:
1. Agent does work
2. Agent commits
3. Agent tries to report Phase: Complete
4. Agent dies (context exhaustion) BEFORE Phase: Complete is reported

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:87`
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/completion.md:3`

**Significance:** This is a systemic issue affecting all spawned agents - completion detection relies on Phase: Complete, which may never be reported.

---

### Finding 2: Multiple locations need updating

**Evidence:** Found "After your final commit" pattern in:
- `orch-go/pkg/spawn/context.go` - 3 occurrences (NoTrack, tracked, final step sections)
- `orch-knowledge/skills/src/shared/worker-base/.skillc/completion.md`
- `orch-knowledge/skills/src/worker/investigation/.skillc/completion.md`
- `orch-knowledge/skills/src/worker/systematic-debugging/.skillc/completion.md`
- `orch-knowledge/skills/src/worker/issue-creation/.skillc/completion.md`
- `orch-knowledge/skills/src/worker/reliability-testing/.skillc/completion.md`
- `orch-knowledge/skills/src/worker/kb-reflect/.skillc/completion.md`

**Source:** grep search across skill templates

**Significance:** All locations need consistent ordering to prevent the race condition.

---

### Finding 3: Correct order is Phase → Commit → Exit

**Evidence:** The correct order should be:
1. Report Phase: Complete FIRST (before commit)
2. Commit any final changes
3. Run /exit to close session

This way:
- If agent dies after phase report but before commit, orchestrator knows it's complete but can see uncommitted work
- If agent dies after commit but before exit, that's fine - work is saved
- Phase report is the critical visibility signal

**Source:** Logical analysis of failure modes

**Significance:** This ordering ensures orchestrator visibility regardless of when agent dies.

---

## Synthesis

**Key Insights:**

1. **Phase report is the critical signal** - The orchestrator monitors for "Phase: Complete" to know when agents finish. This must happen before commit because commits are less likely to fail than context exhaustion.

2. **Consistent enforcement across all skills** - All completion templates needed updating to enforce the same order. The shared worker-base and individual skill completion files all had the wrong order.

3. **Test expectation updated** - The context_test.go expected the old text "SYNTHESIS.md is created and committed" which was changed to just "SYNTHESIS.md is created" (with commit as a separate step).

**Answer to Investigation Question:**

The fix is to reverse the order: agents should report Phase: Complete BEFORE committing their final changes. This was implemented by updating the SpawnContextTemplate in orch-go/pkg/spawn/context.go (3 locations) and 6 skill completion templates in orch-knowledge. The new wording explicitly states "(report phase FIRST - before commit)" and includes rationale for why this order matters.

---

## Structured Uncertainty

**What's tested:**

- ✅ All spawn package tests pass (verified: ran `go test ./pkg/spawn/...`)
- ✅ Template changes compile correctly (no template parsing errors)

**What's untested:**

- ⚠️ skillc build not run (orch-knowledge skills need rebuild to propagate to deployed SKILL.md files)
- ⚠️ End-to-end agent spawn not tested (would require full spawn cycle)

**What would change this:**

- Finding would be wrong if agents can reliably report Phase: Complete even with very low context (but context exhaustion is unpredictable)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Phase-first completion protocol** - Report Phase: Complete before final commit

**Why this approach:**
- Ensures orchestrator visibility even if agent dies after phase report
- Commits can still succeed even without context (git operations are lightweight)
- Simple change to existing templates

**Trade-offs accepted:**
- If agent dies after phase report but before commit, orchestrator sees "complete" but work is uncommitted
- This is better than the alternative (work committed but invisible to orchestrator)

**Implementation sequence:**
1. Update SpawnContextTemplate in orch-go (done)
2. Update skill completion templates in orch-knowledge (done)
3. Run skillc deploy to propagate changes to deployed skills (needed)

---

## References

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - Main template, 3 locations
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context_test.go` - Test expectation update
- `/Users/dylanconlin/orch-knowledge/skills/src/shared/worker-base/.skillc/completion.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc/completion.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/systematic-debugging/.skillc/completion.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/issue-creation/.skillc/completion.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/reliability-testing/.skillc/completion.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/kb-reflect/.skillc/completion.md`

**Commands Run:**
```bash
# Verify tests pass
go test ./pkg/spawn/... -v
```

---

## Investigation History

**2026-01-02 08:28:** Investigation started
- Initial question: How to fix agent phase reporting order?
- Context: Agents dying after commit but before phase report

**2026-01-02 08:35:** Implementation complete
- Updated all templates with correct order
- Tests passing
- Ready for commit
