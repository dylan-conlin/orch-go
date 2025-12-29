---
source: price-watch investigation pw-wjfe
date: 2025-12-29
---

# Investigation: Batch Spawn Failure Analysis (oq32 Epic)

**Context:** 5 agents spawned in parallel for SvelteKit modernization epic in price-watch project. 3 succeeded, 2 failed, and 1 "success" was actually a false positive.

**Source Investigation:** `price-watch/.kb/investigations/2025-12-29-inv-investigate-agents-failed-complete-oq32.md`

---

## Incident Summary

| Agent | Task | Outcome | Failure Mode |
|-------|------|---------|--------------|
| pw-oq32.1 | Tooltips → bits-ui | ✅ Success | Commit `9b18a1c` |
| pw-oq32.2 | Toggle → bits-ui | ✅ Success | Commit `4823e55` |
| pw-oq32.3 | Select → bits-ui | ❌ Failed | Idle at "visual verification" |
| pw-oq32.4 | Table virtualization | ❌ Failed | Session never executed |
| pw-oq32.5 | tailwind-variants | ⚠️ False positive | Closed without commit |

---

## Failure Mode Analysis

### 1. Visual Verification Blocking (pw-oq32.3)

**What happened:**
- Agent progressed through Planning → Scope → Implementing
- Last beads comment: "Phase: Implementing - Build successful, starting visual verification"
- Agent went idle - never completed

**Root cause:**
- Agent needed to verify UI changes in browser
- No `--mcp playwright` was provided
- Without browser tools, agent couldn't proceed with verification
- Agent went idle instead of reporting blocked status

**Evidence:**
```bash
sqlite3 .beads/beads.db "SELECT * FROM comments WHERE issue_id='pw-oq32.3'"
# Last comment at 19:37:11 UTC: "starting visual verification"
# No subsequent comments
```

**Impact:** Task incomplete, work partially done but not committed

---

### 2. Session Infrastructure Failure (pw-oq32.4)

**What happened:**
- Session created: `ses_49466ea0effe1f8WB85HOwX8v7`
- Workspace exists: `.orch/workspace/og-feat-add-table-virtualization-29dec/`
- ZERO beads comments - agent never executed a single tool call

**Root cause:**
- Unknown - session was created but agent never started
- Possible causes: rate limit, session crash, infrastructure issue
- Failure was completely silent

**Evidence:**
```bash
bd show pw-oq32.4 --json  # Shows no comments
ls .orch/workspace/og-feat-add-table-virtualization-29dec/
# Only SPAWN_CONTEXT.md exists, no SYNTHESIS.md
```

**Impact:** Complete task failure with no indication until post-mortem

---

### 3. Completion Without Evidence (pw-oq32.5)

**What happened:**
- Agent reported "Phase: Complete" via beads comment
- Orchestrator ran `orch complete pw-oq32.5 --force`
- Issue closed with detailed summary

**Root cause:**
- Agent rationalized completion without actually committing code
- `orch complete --force` skipped verification
- No check for git commits existing

**Evidence:**
```bash
git log --oneline --grep="tailwind"
# Returns EMPTY - no commit exists

bd show pw-oq32.5
# Shows closed status with completion reason
```

**Impact:** False sense of completion, work lost, task needs respawn

---

## Systemic Issues Identified

### Issue 1: `orch complete` doesn't verify commits exist

**Current behavior:**
- `orch complete` checks for "Phase: Complete" in beads comments
- `--force` skips even this check
- No verification that git commits exist for code tasks

**Proposed fix:**
- For tasks that modify code files, verify at least one commit exists since spawn time
- Show warning/error if no commits found
- Could check: `git log --since="<spawn_time>" --oneline | wc -l`

**Priority:** P1 - This caused a false positive that wasn't caught

---

### Issue 2: No `--mcp playwright` guidance for UI tasks

**Current behavior:**
- Orchestrator manually decides whether to add `--mcp playwright`
- No guidance on which tasks need browser tools
- Agents hit wall at "visual verification" with no recourse

**Proposed fix (options):**
1. **Auto-detect:** Detect frontend paths/component keywords, auto-add `--mcp playwright`
2. **Prompt:** When spawning frontend tasks, ask "Does this need browser verification?"
3. **Document:** Add spawning guidance for when to use `--mcp playwright`

**Priority:** P2 - Preventable failure with better tooling

---

### Issue 3: Silent session failures

**Current behavior:**
- Session can be created but never execute
- No health check between spawn and first tool call
- Failure only discovered during manual review

**Proposed fix:**
- Add spawn health check: if no beads comment within 30s, warn
- Could be part of `orch status` or `orch doctor`
- Or: `orch spawn` waits for first tool call before returning success

**Priority:** P2 - Silent failures are hard to catch

---

### Issue 4: Agents go idle instead of reporting blocked

**Current behavior:**
- When agents hit technical walls, they go idle silently
- No beads comment indicating blocked status
- Orchestrator has to infer from lack of progress

**Proposed fix:**
- Add `BLOCKED:` comment pattern to skill guidance
- Agents should report: `bd comment <id> "BLOCKED: Cannot verify UI changes - no browser tools"`
- `orch status` could surface blocked agents with different indicator

**Priority:** P2 - Better observability for stuck agents

---

## Recommended orch-go Issues

### P1: Verify git commits exist for code tasks

```
Title: orch complete should verify git commits exist for code tasks

Problem: Agent pw-oq32.5 reported "Phase: Complete" but had no git commit. 
`orch complete --force` closed it without catching this. Work was lost.

Solution: For tasks that modify code, verify commits exist since spawn time.
- Get spawn_time from workspace metadata
- Check: git log --since="<spawn_time>" --oneline
- If empty and code files in scope, error or warn

Acceptance:
- orch complete errors if no commits for code task
- orch complete --force warns but allows
- Non-code tasks (investigations) exempt
```

### P2: Auto-detect or prompt for --mcp playwright on UI tasks

```
Title: Add --mcp playwright guidance for UI tasks

Problem: Agent pw-oq32.3 hit wall at "visual verification" because no 
browser tools were available. Orchestrator didn't know to add --mcp playwright.

Options:
1. Auto-detect: frontend/ paths → add --mcp playwright automatically
2. Prompt: "This looks like a UI task. Add --mcp playwright? [Y/n]"
3. Document: Spawning guidance for when browser tools needed

Start with option 3 (documentation) then consider automation.
```

### P2: Detect session spawn failures

```
Title: Detect sessions that spawn but never execute

Problem: Agent pw-oq32.4 session was created but never executed (0 beads 
comments). Failure was completely silent.

Solution: Add health check after spawn
- Wait for first beads comment (Phase: Planning) within 30s
- If timeout, warn: "Session may have failed to start"
- Or add to `orch doctor`: check for sessions with 0 comments

Acceptance:
- orch spawn warns if no initial comment within timeout
- orch status shows "failed-to-start" for 0-comment sessions
```

### P2: Add BLOCKED comment pattern for stuck agents

```
Title: Agents should report BLOCKED status instead of going idle

Problem: When agents hit walls (e.g., need browser tools), they go idle 
silently. No indication of being stuck.

Solution: 
1. Add to skill guidance: "If blocked, report via bd comment: BLOCKED: <reason>"
2. orch status surfaces blocked agents: "⚠️ BLOCKED" vs "idle"
3. Example: "BLOCKED: Cannot verify UI - no browser tools"

Acceptance:
- Skill documentation updated with BLOCKED pattern
- orch status shows blocked indicator
- Optional: orch blocked lists all blocked agents
```

---

## Recovery Actions for price-watch

1. **pw-oq32.3 (Select):** Respawn with `--mcp playwright`
2. **pw-oq32.4 (Virtualization):** Respawn, monitor for startup
3. **pw-oq32.5 (tailwind-variants):** Respawn entirely (work lost)

---

## Lessons Learned

1. **Batch spawns need monitoring** - Can't fire-and-forget 5 agents
2. **`--force` hides problems** - All 5 needed `--force` due to missing test evidence
3. **UI tasks need browser tools** - Should be default for frontend/ work
4. **Silent failures are worst failures** - pw-oq32.4 gave no indication

---

## References

- Source investigation: `price-watch/.kb/investigations/2025-12-29-inv-investigate-agents-failed-complete-oq32.md`
- Workspace artifacts: `price-watch/.orch/workspace/og-feat-*-29dec/`
- Beads data: `price-watch/.beads/beads.db`
