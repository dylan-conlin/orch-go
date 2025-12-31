<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Three systemic failures caused session chaos: (1) Agents were completing without Phase: Complete comments, (2) Issues were being respawned despite being already-fixed, (3) bd comments returns empty even when JSONL has comments (sync issue).

**Evidence:** gxwu and lsrj both have no visible comments via `bd comments` but JSONL shows 5+ comments; both issues had "already fixed" close reasons but kept being respawned; SYNTHESIS.md files claimed success but no commits were made.

**Knowledge:** The verification chain has gaps: agents can complete without reporting Phase, orchestrator can close issues manually, and respawning doesn't check if issues were already addressed.

**Next:** Implement: (1) Gate completion on Phase: Complete comment existing, (2) Add "stale issue" check before spawning bugs, (3) Fix bd comments sync issue.

---

# Investigation: What Went Wrong in Dec 30 Evening Session

**Question:** Why did multiple agents claim success without making changes, and why did we keep respawning already-fixed bugs (gxwu, lsrj)?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Issues lsrj and gxwu Have No Visible Comments But JSONL Shows Them

**Evidence:** 
- `bd comments lsrj` returns "No comments on lsrj"
- `bd comments gxwu` returns "No comments on gxwu"
- BUT `.beads/issues.jsonl` shows lsrj has 5 comments with Phase: Complete
- AND git diff b2b19b4a shows gxwu had 6+ comments including Phase: Planning and Phase: Implementing

**Source:** 
- `bd comments lsrj` / `bd comments gxwu` - returned empty
- `grep "orch-go-lsrj" .beads/issues.jsonl | python3 ...` - shows 5 comments
- `git diff b2b19b4a -- .beads/issues.jsonl` - shows gxwu comment history

**Significance:** There's a sync issue between `bd comments` CLI and the JSONL file. This would cause completion verification to fail to find Phase: Complete even when agents reported it correctly.

---

### Finding 2: Both Issues Were "Already Fixed" But Kept Being Respawned

**Evidence:**
- lsrj close reason: "Investigation found bug was already fixed by Dec 28 commits (3a834ac0, 784c2703). No code changes needed."
- gxwu close reason: "Already fixed in commit b2b19b4a - daemon now skips failing issues and continues processing queue"
- Both issues had workspaces with SYNTHESIS.md claiming investigation/fix
- Yet the same bugs were spawned 5+ times according to the task description

**Source:**
- `bd show lsrj` / `bd show gxwu` - shows close reasons
- `.orch/workspace/og-debug-dashboard-shows-active-30dec/SYNTHESIS.md` - claims success for lsrj
- `.orch/workspace/og-debug-daemon-blocked-cross-30dec/SYNTHESIS.md` - claims success for gxwu

**Significance:** There's no "stale bug check" before spawning. Issues can be spawned for bugs that were already fixed, wasting agent time investigating non-bugs.

---

### Finding 3: Agents Complete Without Reporting Phase Via bd comment

**Evidence:**
- Spawn context for lsrj explicitly says: "Report via `bd comment orch-go-lsrj "Phase: Complete - [summary]"`"
- Yet `bd comments lsrj` shows "No comments" 
- Meanwhile SYNTHESIS.md exists and claims outcome: success
- Close was performed manually (close reason matches SYNTHESIS.md content)

**Source:**
- `.orch/workspace/og-debug-dashboard-shows-active-30dec/SPAWN_CONTEXT.md` lines with "bd comment" instructions
- `bd comments lsrj` - empty
- `.orch/workspace/og-debug-dashboard-shows-active-30dec/SYNTHESIS.md` - claims success

**Significance:** The completion flow allowed closing issues without Phase: Complete being verified via beads comments. This bypasses the VerifyCompletionFull check.

---

### Finding 4: Git Commit Verification Gate Exists But May Not Be Triggering

**Evidence:**
- `pkg/verify/git_commits.go` contains `VerifyGitCommitsForCompletion()` which checks for commits since spawn time
- For code-producing skills (feature-impl, systematic-debugging), it should block completion if no commits exist
- Both lsrj and gxwu used investigation/systematic-debugging skills
- Investigation skill is in `artifactProducingSkills` map - exempt from commit check
- SYNTHESIS.md for gxwu says "### Commits: To be committed after this synthesis" - claimed code changes but no actual commit

**Source:**
- `pkg/verify/git_commits.go:27-43` - skill classification maps
- `.orch/workspace/og-debug-daemon-blocked-cross-30dec/SYNTHESIS.md` line 24-25 - claims commits

**Significance:** The gate exists but:
1. Investigation skill is exempt (correctly)
2. But agents can claim code changes in SYNTHESIS.md without actually committing
3. The verification trusts skill type, not actual SYNTHESIS.md claims

---

### Finding 5: Manual Closure Bypasses Verification Chain

**Evidence:**
- lsrj was closed with close_reason matching investigation findings
- gxwu close_reason says "Already fixed in commit b2b19b4a" but issue is still marked "open" in some contexts
- Commit 916363a9 shows `.beads/issues.jsonl` was modified to add close information
- This appears to be manual `bd close` without going through `orch complete` verification

**Source:**
- `git show 916363a9` - commit that closed lsrj
- `bd show lsrj` vs `bd show gxwu` - different statuses despite similar "already fixed" claims

**Significance:** Manual `bd close` bypasses all verification gates. Orchestrator (or agent) can close issues directly without VerifyCompletionFull running.

---

## Synthesis

**Key Insights:**

1. **Beads Comments Sync Issue** - The `bd comments` command doesn't reflect the actual JSONL state. This breaks any verification that depends on parsing beads comments for Phase status.

2. **Missing Stale Bug Detection** - When a bug is spawned, there's no check for whether it was already fixed. This leads to wasted agent time investigating non-bugs and finding "already fixed" repeatedly.

3. **Manual Close Bypass** - The verification chain (`VerifyCompletionFull`) only runs when using `orch complete`. Manual `bd close` bypasses all verification, allowing issues to be closed without Phase: Complete or other gates.

4. **SYNTHESIS.md Claims vs Reality Mismatch** - Agents can claim "commits pending" or "code changes made" in SYNTHESIS.md without actually committing. Verification trusts skill type, not synthesis content.

**Answer to Investigation Question:**

The session chaos had THREE root causes:

1. **Respawning already-fixed bugs (gxwu, lsrj)**: No "stale issue" check exists. Bug reports can be spawned even when the bug was fixed hours ago. Agents waste time discovering "this was already fixed."

2. **False success claims without commits**: Agents completed SYNTHESIS.md with "outcome: success" but the git commit verification gate only applies to code-producing skills. Investigation skill is exempt, so agents can claim success without any verifiable output.

3. **Verification bypass via manual close**: Issues were closed with `bd close` directly (visible in git history), bypassing the `orch complete` verification chain. This allowed closing without Phase: Complete being verified.

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd comments lsrj` returns empty despite JSONL having 5 comments (verified: ran both commands)
- ✅ Git history shows manual close commits modifying .beads/issues.jsonl directly (verified: git show 916363a9)
- ✅ Both issues have "already fixed" close reasons (verified: bd show)
- ✅ SYNTHESIS.md files exist with success claims (verified: read files)

**What's untested:**

- ⚠️ Whether beads daemon was running during the session (might explain sync issue)
- ⚠️ Whether agents actually called `bd comment` but it failed silently
- ⚠️ The exact sequence of events that led to 5+ respawns

**What would change this:**

- Finding would be wrong if agents DID report Phase: Complete but a different sync issue prevents reading
- Finding would be wrong if "stale bug" spawning was intentional for regression testing

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Multi-Gate Verification

**Three changes needed:**

1. **Fix beads comments sync** - Investigate why `bd comments` returns empty when JSONL has data. Likely beads daemon issue or socket communication problem.

2. **Add stale bug check before spawn** - Before spawning a bug issue, check git history for commits mentioning the issue ID or related keywords since issue creation. Warn if potentially stale.

3. **Gate `bd close` on Phase: Complete** - Either route all closes through `orch complete`, or add verification to `bd close --reason` that checks for Phase: Complete in comments.

**Why this approach:**
- Addresses each root cause independently
- Maintains backward compatibility (stale check is warning, not block)
- Follows existing verification pattern in pkg/verify

**Trade-offs accepted:**
- Stale check may have false positives for long-running bugs
- bd close gating adds friction for legitimate manual closes

**Implementation sequence:**
1. Fix beads sync issue first (most impactful, unblocks verification)
2. Add stale bug warning (prevents waste)
3. Gate bd close (prevents bypass)

### Alternative Approaches Considered

**Option B: Trust SYNTHESIS.md claims**
- **Pros:** Would catch false claims in synthesis
- **Cons:** Complex NLP parsing, agents could still lie
- **When to use instead:** If beads sync is unfixable

**Option C: Require git commits for all skills**
- **Pros:** Simple, verifiable
- **Cons:** Investigation skill genuinely produces artifacts, not commits
- **When to use instead:** Never - would block valid investigations

---

### Implementation Details

**What to implement first:**
- Investigate beads sync issue (run `bd daemon status`, check socket)
- Add logging to `bd comments` to trace data flow

**Things to watch out for:**
- ⚠️ Beads daemon might need restart after changes
- ⚠️ Stale check should use issue creation time, not spawn time
- ⚠️ bd close gating may break scripts that close programmatically

**Areas needing further investigation:**
- Why did beads daemon not sync comments? Was it down?
- What's the full event sequence for the 5+ respawns?

**Success criteria:**
- ✅ `bd comments <id>` returns same data as JSONL
- ✅ Spawning stale bugs shows warning
- ✅ Manual close without Phase: Complete shows error

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Completion verification logic
- `pkg/verify/git_commits.go` - Git commit verification gate
- `.orch/workspace/og-debug-*/SYNTHESIS.md` - Agent completion claims
- `.beads/issues.jsonl` - Beads issue database

**Commands Run:**
```bash
# Check beads comments vs JSONL
bd comments lsrj  # Empty
grep "orch-go-lsrj" .beads/issues.jsonl | python3 -c "..." # 5 comments

# Check git history for manual closes
git show 916363a9

# Check issue states
bd show lsrj
bd show gxwu
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-30-inv-dashboard-shows-0-active-agents-while-cli-shows-2.md` - The lsrj investigation that found "already fixed"

---

## Self-Review

- [x] Real test performed (ran bd comments, checked JSONL)
- [x] Conclusion from evidence (traced three root causes)
- [x] Question answered (why chaos, why respawning)
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-30 17:38:** Investigation started
- Initial question: Why did agents claim success without changes? Why respawn fixed bugs?
- Context: Evening session had multiple failures

**2025-12-30 17:45:** Found beads sync issue
- bd comments returns empty but JSONL has data

**2025-12-30 17:55:** Found manual close bypass
- Git history shows direct JSONL modifications

**2025-12-30 18:05:** Investigation completed
- Status: Complete
- Key outcome: Three root causes identified - beads sync, stale bugs, verification bypass
