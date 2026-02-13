<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Entropy spiral was caused by autonomous agents operating without human review gates, producing 1163 commits (5.4M LOC churn) with zero human intervention over 26 days.

**Evidence:** Git history shows 1162/1163 commits by "Test User" (agent identity), 5244 files created then deleted, zero human commits in period.

**Knowledge:** Missing circuit breakers: no human-in-the-loop review, no churn metrics, no LOC gates, no "stop and ask human" triggers.

**Next:** Implement mandatory human review gates, churn monitoring, and automatic halt on runaway metrics.

**Authority:** strategic - This involves irreversible architectural decisions about human oversight

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Entropy Spiral Postmortem

**Question:** What patterns led to 1163 agent commits churning 5.4M LOC with zero human oversight, and what mitigations would prevent recurrence?

**Started:** 2026-02-12
**Updated:** 2026-02-12
**Owner:** Investigation Worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md | deepens | yes | Root causes identified but not addressed |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** Earlier post-mortem (Dec 27-Jan 2) identified same root causes: agents fixing agent infrastructure, investigations replacing testing, no human verification loop. These mitigations were not implemented, causing repeat spiral.

---

## Findings

### Finding 1: Massive Scale - 1163 commits, 5.4M LOC churn, 26 days

**Evidence:**
- Date range: Jan 18 2026 to Feb 12 2026 (26 days)
- Total commits: 1163
- Authors: 1162 "Test User" (agent), 1 "Claude"
- Total churn: 3,568,921 lines added, 1,845,816 lines deleted = 5,414,737 LOC touched
- Net change: +1,723,643 lines (not -16K as initially stated)
- Files created: 15,432
- Files deleted: 5,957
- Files created then deleted: 5,244 (33% of created files were abandoned)

**Source:** 
```bash
git log --numstat 0bca3dec..entropy-spiral-feb2026
git shortlog -sn 0bca3dec..entropy-spiral-feb2026
```

**Significance:** Zero human commits in 26 days indicates complete automation with no review gates. The 5.4M LOC churn (vs 1.7M net) shows massive rework/reversal of work.

---

### Finding 2: Churned Artifact Categories

**Evidence:**
Top churn by category:
1. `.beads/issues.jsonl` - 66,118 LOC churn (issue tracking metadata)
2. Session transcripts (`session-ses_*.md`) - 20K-18K LOC each
3. Workspace ACTIVITY.json files - 10K-19K LOC each (agent activity logs)
4. `cmd/orch/spawn_cmd.go` - 9,880 LOC churn
5. `pkg/daemon/daemon.go` - 7,218 LOC churn

Churned files by directory:
- `.kb/investigations` - 407 files created then deleted
- `.kb/archive/investigations` - 185 files
- `cmd/orch` - 175 files created then deleted
- `.kb/decisions` - 79 files
- `pkg/daemon` - 46 files (more deleted than created: 77 created, 84 deleted)

**Source:** `git log --numstat` analysis

**Significance:** Agents were creating investigations, making decisions, then abandoning them. Core code (`cmd/orch`, `pkg/daemon`) had more files deleted than created - indicating work was being built then torn down.

---

### Finding 3: Commit Type Distribution Reveals Patterns

**Evidence:**
- 168 feat commits (new features)
- 161 fix commits (fixing breakage)
- 133 bd sync commits (automated issue tracking)
- 95 investigation commits
- 91 chore commits
- 60 architect commits
- 45 inv commits
- 32 refactor commits
- 29 test commits

Fix:Feat ratio = 0.96:1 (nearly equal fixes to features)

**Source:** `git log --format="%s" | sed 's/:.*//' | sort | uniq -c`

**Significance:** High fix:feat ratio suggests features were being introduced without adequate testing, requiring immediate fixes. The 133 `bd sync` commits (11% of total) indicate significant overhead from automated tracking.

---

### Finding 4: 24/7 Autonomous Operation with No Human Checkpoints

**Evidence:**
Commits occurred at all hours including overnight:
- 2026-01-22 04:00 (4 commits)
- 2026-01-22 05:00 (1 commit)
- 2026-01-22 06:00 (1 commit)
- 2026-01-23 02:00 (1 commit)
- 2026-01-23 03:00 (2 commits)
- 2026-01-23 05:00 (1 commit)

Average 45 commits/day sustained for 26 days.
Zero human commits in the entire period.
Commit authorship: 1162 "Test User" (agent), 1 "Claude"

**Source:** `git log --format="%ad" --date=format:"%Y-%m-%d %H"` analysis

**Significance:** The system ran continuously without human oversight. No circuit breaker existed to halt autonomous operation or require human confirmation.

---

### Finding 5: Repeat Spiral - Same Root Causes as Dec 27-Jan 2

**Evidence:**
Earlier post-mortem (.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md) identified:

| Root Cause (Jan 2) | Evidence in Current Spiral |
|-------------------|----------------------------|
| "Agents fixing agent infrastructure" | 175 cmd/orch files churned, 84 pkg/daemon files deleted |
| "Investigations replaced testing" | 407 investigation files created then deleted |
| "No human verification loop" | Zero human commits in 26 days |
| "Velocity over correctness" | 1163 commits = 45/day vs 58/day in first spiral |
| "Complexity as solution" | pkg/attention built across 18+ commits then deleted |

The earlier post-mortem recommended:
1. "Human verifies behavior, not just output, before next change" - NOT IMPLEMENTED
2. "Agents don't modify agent infrastructure without manual review" - NOT IMPLEMENTED
3. "One change at a time with a pause to confirm it worked" - NOT IMPLEMENTED
4. "I don't know if this is working halts progress" - NOT IMPLEMENTED
5. "Limit self-modification velocity" - NOT IMPLEMENTED

**Source:** git show entropy-spiral-feb2026:.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md

**Significance:** The system repeated the exact same failure pattern because mitigations from the first post-mortem were never implemented. This is the critical finding: **known root causes with documented mitigations were ignored.**

---

### Finding 6: Failed Stabilization Attempts Continued Without Human Intervention

**Evidence:**
One explicit stabilization commit on Feb 9:
```
21ed501d stabilize: abandon contaminated agents, strip triage:ready from blocked issues, restore clean tree
```

However, activity continued immediately afterward:
- Feb 9-10: 39 more commits after stabilization
- Feb 10-12: 150 more commits
- Zombie process fixes, zombie detection, reaper commands
- Memory pressure investigation (OpenCode killed at 8.4GB)

Problems identified but not halted:
- Zombie bun processes (25cfd6d8: "3 compounding integration mismatches")
- OpenCode stack overflows
- 8.4GB memory usage causing jetsam kills
- Test failures requiring fixes (ff84a29a: "7 test failures")

**Source:** git log analysis around stabilization commit

**Significance:** Even when instability was detected, the system continued without human intervention. The stabilization was performed by agents, not humans, and was insufficient to halt the spiral.

---

## Synthesis

**Key Insights:**

1. **Self-Modifying Systems Require External Verification** - Agents modified the very infrastructure that tracks and manages agents (Finding 2: 175 cmd/orch files, 46 pkg/daemon files churned). Each change altered the ground truth, making it impossible for subsequent agents to verify correctness.

2. **Documented Lessons Not Implemented = Repeated Failure** - The Dec 27-Jan 2 post-mortem identified 5 specific mitigations. None were implemented (Finding 5). The current spiral repeated the exact same failure pattern at 3x the duration.

3. **Velocity Without Gates Compounds Errors** - 45 commits/day for 26 days with no human verification (Finding 4). The 0.96:1 fix:feat ratio (Finding 3) shows each feature introduced nearly one bug, creating a churn cycle.

4. **Agent Self-Stabilization Fails** - The Feb 9 stabilization attempt (Finding 6) was performed by agents, not humans. It failed to halt the spiral, proving that the system cannot stabilize itself.

**Answer to Investigation Question:**

**What patterns led to 1163 agent commits churning 5.4M LOC with zero human oversight?**

1. **No human-in-the-loop gates** - The system had no mechanism to require human approval before continuing after N commits or detecting anomalies.

2. **Ignored prior post-mortem** - Known mitigations from Jan 2 post-mortem were never implemented.

3. **Self-modifying system without external verification** - Agents changed the code that observes agents, making self-diagnosis impossible.

4. **Velocity incentives without correctness gates** - The system rewarded shipping (commits, completion) without verifying the ship was sound.

**What mitigations would prevent recurrence?**

See Implementation Recommendations below.

---

## Structured Uncertainty

**What's tested:**

- ✅ Commit counts, dates, authors verified via git log (1163 commits, 26 days, 1162 by "Test User")
- ✅ LOC churn verified via git log --numstat (5.4M total, 1.7M net)
- ✅ Prior post-mortem existence verified (read full content from entropy-spiral-feb2026 branch)
- ✅ Churned file counts verified via git diff-filter analysis (5244 files created then deleted)

**What's untested:**

- ⚠️ Whether proposed mitigations would actually prevent recurrence (hypothesis)
- ⚠️ Whether velocity was the root cause vs a symptom (could be direction, not speed)
- ⚠️ Whether human intervention would have been effective (assumes human attention was available)

**What would change this:**

- Finding would be wrong if the 1163 commits represent parallel feature development rather than churn (would need to verify features shipped to production)
- Finding would be wrong if human review was occurring but not tracked via commits (would need to check Slack/email/verbal)
- Mitigations would be wrong if the real issue is agent capability, not process (would need to test with same process but better agents)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Human approval gate after N commits | strategic | Changes fundamental autonomy model |
| Daily churn metrics dashboard | architectural | Cross-component visibility |
| Infrastructure change review requirement | architectural | Changes agent workflow |
| Commit velocity throttle | implementation | Reversible rate limit |

### Recommended Approach ⭐

**Circuit Breaker Architecture** - Implement hard gates that halt autonomous operation and require human confirmation.

**Why this approach:**
- Directly addresses Finding 4: No mechanism existed to pause and ask human
- Addresses Finding 5: Jan 2 mitigations recommended same approach but weren't implemented
- Prevents self-stabilization loops (Finding 6): Human must explicitly continue

**Trade-offs accepted:**
- Reduced velocity (acceptable: 45 commits/day with 0.96:1 fix:feat ratio isn't productive velocity)
- Requires human attention (acceptable: that's the point - human-in-loop)

**Implementation sequence:**
1. **Daily commit limit with human override** - System halts after 20 commits/day; human must explicitly continue
2. **Churn monitoring** - Alert when files created then deleted exceeds threshold (e.g., 10%)
3. **Infrastructure change gate** - Any change to cmd/orch, pkg/daemon, pkg/spawn requires human review
4. **Fix:Feat ratio monitor** - Alert when ratio exceeds 0.5:1 (50% of features causing bugs)

### Alternative Approaches Considered

**Option B: Improve agent quality (better testing, better prompts)**
- **Pros:** Preserves autonomy, addresses root capability
- **Cons:** Doesn't address self-modification problem (Finding 2); agents can't verify changes to observation infrastructure
- **When to use instead:** After circuit breakers are in place; defense in depth

**Option C: Disable autonomous operation entirely (human approves every commit)**
- **Pros:** Maximum control
- **Cons:** Eliminates agent value; overkill
- **When to use instead:** During active crisis recovery

**Rationale for recommendation:** Circuit breakers balance autonomy with oversight. Finding 5 shows known mitigations were ignored - circuit breakers FORCE human involvement rather than relying on process discipline.

---

### Implementation Details

**What to implement first:**
1. **Daily commit limit (orch config)** - Add `max_commits_per_day: 20` to orch config; daemon halts and notifies human when reached
2. **Churn ratio monitoring** - Track created/deleted file ratio per day; alert at >10%
3. **Infrastructure gate list** - Define `protected_paths: [cmd/orch, pkg/daemon, pkg/spawn]` requiring human review

**Things to watch out for:**
- ⚠️ Agents may work around limits (e.g., batch changes into single commits) - monitor commit size
- ⚠️ Human may rubber-stamp approvals without reviewing - make approval effortful
- ⚠️ Legitimate parallel work may hit daily limits - allow per-issue exemptions

**Areas needing further investigation:**
- What caused the Dec 27-Jan 2 mitigations to not be implemented? Process failure or prioritization?
- Are there legitimate high-velocity periods (e.g., initial development) that need different rules?
- How to distinguish productive velocity from churn velocity automatically?

**Success criteria:**
- ✅ No period >7 days without human commit
- ✅ Churn ratio (created+deleted/net) stays below 2:1
- ✅ Infrastructure changes require explicit human approval in beads comments
- ✅ Fix:feat ratio stays below 0.5:1 over rolling 7-day window

---

## References

**Files Examined:**
- `.kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md` (via git show) - Prior post-mortem with same root causes
- Git history on entropy-spiral-feb2026 branch - Primary evidence source

**Commands Run:**
```bash
# Commit statistics
git log --oneline 0bca3dec..entropy-spiral-feb2026 | wc -l  # 1163
git shortlog -sn 0bca3dec..entropy-spiral-feb2026  # 1162 Test User, 1 Claude

# LOC churn
git log --numstat 0bca3dec..entropy-spiral-feb2026 | awk 'NF==3 {plus+=$1; minus+=$2} END {print plus, minus, plus+minus}'

# File churn
git log --diff-filter=A --name-only --format="" 0bca3dec..entropy-spiral-feb2026 | sort -u > created.txt
git log --diff-filter=D --name-only --format="" 0bca3dec..entropy-spiral-feb2026 | sort -u > deleted.txt
comm -12 created.txt deleted.txt | wc -l  # 5244

# Commit type distribution
git log --format="%s" 0bca3dec..entropy-spiral-feb2026 | sed 's/:.*//' | sort | uniq -c | sort -rn
```

**Related Artifacts:**
- **Post-mortem:** .kb/post-mortems/2026-01-02-system-spiral-dec27-jan02.md - Earlier spiral with identical root causes
- **Branch:** entropy-spiral-feb2026 - Preserved evidence of the spiral

---

## Investigation History

**2026-02-12 22:40:** Investigation started
- Initial question: What patterns led to entropy spiral with 772+ agent commits?
- Context: Orchestrator requested post-mortem analysis

**2026-02-12 22:45:** Scale discovery
- Found 1163 commits (not 772), 5.4M LOC churn, zero human commits

**2026-02-12 23:00:** Prior post-mortem discovery
- Found Dec 27-Jan 2 post-mortem documenting same root causes
- Critical finding: mitigations were documented but never implemented

**2026-02-12 23:15:** Investigation completed
- Status: Complete
- Key outcome: Repeat spiral caused by unimplemented mitigations from prior post-mortem
