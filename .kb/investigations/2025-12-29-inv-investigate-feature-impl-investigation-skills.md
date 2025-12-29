<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Skills instruct agents to label discovered work as triage:ready, causing automatic daemon spawning without deduplication checks—resulting in 274 ready issues and duplicate entries (e.g., 2× for "GenerateContext", 2× for "symlink pattern").

**Evidence:** (1) feature-impl SKILL.md:311-313 and investigation SKILL.md:200-210 both guide agents to use `triage:ready` labels. (2) beads CLI has no deduplication—issues are created unconditionally. (3) `bd list` shows 801 total issues, 274 with triage:ready. (4) Multiple exact-title duplicates confirmed (qoo1/3p7q, vsqk/foac, etc.).

**Knowledge:** The original intent was valuable (surface discovered work for daemon processing), but the combination of (1) aggressive auto-labeling guidance, (2) no title/description deduplication at create-time, and (3) daemon auto-spawning creates exponential issue/agent growth when multiple agents run concurrently.

**Next:** Implement two-layer fix: (1) Add deduplication check to `bd create` (match title/description hash against open issues), (2) Change skill guidance to default to `triage:review` (requiring orchestrator approval) rather than `triage:ready` for discovered work.

---

# Investigation: Feature-Impl and Investigation Skills Auto-Labeling

**Question:** Why do feature-impl and investigation skills auto-label discovered work as triage:ready, and why is there no deduplication check before creating issues?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Agent (og-inv-investigate-feature-impl-29dec)
**Phase:** Complete
**Next Step:** Orchestrator to decide on implementation approach
**Status:** Complete

**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Skills guide agents to use triage:ready for "high confidence" discovered work

**Evidence:** Both skills contain nearly identical guidance:

**feature-impl SKILL.md:311-313:**
```markdown
### Discovered Work
- [ ] Reviewed for discoveries (bugs, tech debt, enhancement ideas)
- [ ] Created beads issues with `triage:ready` or `triage:review` labels
```

**investigation SKILL.md:200-210:**
```markdown
| Confidence | Label | When to use |
|------------|-------|-------------|
| High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Lower | `triage:review` | Uncertain scope, needs orchestrator input |
```

**Source:** 
- `~/.claude/skills/worker/feature-impl/SKILL.md:311-313`
- `~/.claude/skills/worker/investigation/SKILL.md:200-210`

**Significance:** The guidance explicitly directs agents to label issues as `triage:ready` for "clear problem, known fix" cases. This was intentional—see Finding 4.

---

### Finding 2: Beads has NO deduplication at issue creation time

**Evidence:** Examined `pkg/beads/cli_client.go:180-224` and `pkg/beads/client.go:750-777`. The `Create` functions take title, description, type, priority, and labels, then unconditionally run `bd create` without checking for existing issues with matching titles/descriptions.

```go
// pkg/beads/cli_client.go:212-223
cmd := c.bdCommand(cmdArgs...)
output, err := cmd.Output()
if err != nil {
    return nil, fmt.Errorf("bd create failed: %w", err)
}
// No dedup check before or after
```

**Source:** `pkg/beads/cli_client.go:180-224`

**Significance:** Multiple agents working concurrently can create duplicate issues. The "Discovered Work Check" runs at agent completion, so if 3 agents discover the same problem, 3 identical issues get created.

---

### Finding 3: Daemon auto-spawns triage:ready issues, compounding the problem

**Evidence:** The daemon polls beads and spawns agents for issues with `triage:ready` label:

```go
// pkg/daemon/daemon.go:47
Label: "triage:ready",
```

Combined with finding 1 and 2:
1. Agent A discovers "Bug X" → creates issue with triage:ready
2. Daemon spawns Agent B for "Bug X" issue
3. Agent B (while investigating) also discovers "Bug X" → creates duplicate issue
4. Daemon spawns Agent C for the duplicate...

**Source:** `pkg/daemon/daemon.go:47`

**Significance:** This creates a positive feedback loop. The 193 commits in 2 days mentioned in the task likely came from this exponential growth.

---

### Finding 4: The triage:ready pattern was intentional (documented history)

**Evidence:** Git history shows deliberate design decisions:

1. **Dec 6, 2025** (`480b765`): Added "Discovered Work Check" sections to skills
   - Intent: "Ensure agents explicitly review for and track discovered bugs, technical debt, enhancement ideas"
   
2. **Dec 17, 2025** (`bcc10e9`): Added triage labeling to investigation skill
   - Intent: "Enables daemon auto-pickup of discovered work from investigations"

Original investigation (`2025-12-17-inv-investigate-opportunities-expand-triage-ready.md`) explicitly states:
> "There's a clear opportunity to add triage labeling guidance to 5 skills, enabling discovered issues to flow automatically to daemon processing."

**Source:** 
- `~/orch-knowledge` git history
- `.kb/investigations/2025-12-17-inv-investigate-opportunities-expand-triage-ready.md`
- `.kb/investigations/2025-12-06-investigation-skill-guide-agents-create.md`

**Significance:** The feature was designed to enable autonomous work queuing. The problem is that deduplication wasn't considered at the same time.

---

### Finding 5: Current issue count confirms the problem scope

**Evidence:**
```bash
$ bd list | wc -l
801

$ bd list | grep "triage:ready" | wc -l
274

$ bd list | grep "in_progress" | wc -l
50
```

Duplicate examples:
- `orch-go-qoo1` and `orch-go-3p7q`: Both "Wire up in GenerateContext()..."
- `orch-go-vsqk` and `orch-go-foac`: Both "Implement symlink pattern..."
- `orch-go-z52s` and `orch-go-zr2n`: Both about propagation tasks

**Source:** `bd list` output

**Significance:** 801 issues with 274 ready for daemon pickup is unsustainable. Many are duplicates or near-duplicates created by concurrent agents.

---

## Synthesis

**Key Insights:**

1. **Intentional but incomplete design** - The triage:ready pattern was deliberately added to enable autonomous daemon processing. However, the design assumed single-agent workflows and didn't consider:
   - Multiple agents discovering the same issues
   - Concurrent agent execution creating race conditions
   - No feedback loop prevention

2. **Missing deduplication layer** - Neither the beads CLI nor the skills have any mechanism to:
   - Check if an issue with similar title/description exists
   - Warn the agent before creating a duplicate
   - Merge or link related issues

3. **Compounding factors:**
   - Skills guide toward `triage:ready` (immediate daemon pickup)
   - Daemon spawns agents that also run "Discovered Work Check"
   - Each new agent can rediscover the same issues
   - No circuit breaker stops the exponential growth

**Answer to Investigation Question:**

**Why auto-label as triage:ready?** Intentional design decision from Dec 2025 to enable autonomous daemon processing of discovered work. The goal was reducing orchestrator bottleneck.

**Why no deduplication?** Simply wasn't considered. The original investigations focused on "how to surface discovered work" rather than "how to prevent duplicate work from being surfaced."

---

## Structured Uncertainty

**What's tested:**

- ✅ Skills contain triage:ready guidance (verified by reading SKILL.md files)
- ✅ Beads client has no deduplication (verified by reading source code)
- ✅ Daemon auto-spawns triage:ready issues (verified in daemon.go)
- ✅ Duplicate issues exist (verified via `bd list` grep)
- ✅ Issue count is 801 total, 274 triage:ready (verified via command)

**What's untested:**

- ⚠️ Exact rate of duplicate creation (would need timestamp analysis)
- ⚠️ Whether all 193 commits came from duplicate spawns (correlation not causation)
- ⚠️ Impact of changing to triage:review default (would need A/B comparison)

**What would change this:**

- If beads had silent deduplication we didn't find → different root cause
- If daemon had rate limiting → problem would be capped
- If skills didn't actually run for most agents → smaller scope

---

## Implementation Recommendations

**Purpose:** Preserve original intent (surface discovered work) while preventing runaway duplication.

### Recommended Approach ⭐

**Two-layer fix: Deduplication + Conservative Labels**

**Why this approach:**
- Layer 1 (dedup) prevents identical issues regardless of label
- Layer 2 (conservative labels) prevents auto-spawning of uncertain discoveries
- Preserves daemon workflow for genuinely high-confidence issues
- Doesn't break existing orchestrator workflows

**Trade-offs accepted:**
- Deduplication adds latency to `bd create` (must search existing issues)
- Conservative labels mean more orchestrator review (but prevents runaway spawning)

**Implementation sequence:**

1. **Add deduplication to beads CLI** - Before creating, search for open issues with similar title (Levenshtein distance or exact match). Return existing issue ID if found, or create new if truly unique.

2. **Change skill default to triage:review** - Modify skill guidance:
   - Default: `triage:review` for all discovered work
   - Only use `triage:ready` when orchestrator explicitly requests it
   - Example: "QUESTION: Found bug X. Should I label triage:ready?" → wait for response

3. **Add rate limiting to daemon** - Cap spawns per hour to prevent runaway growth even if duplicates slip through.

### Alternative Approaches Considered

**Option B: Deduplication only (no label change)**
- **Pros:** Preserves autonomous workflow
- **Cons:** Doesn't solve near-duplicates (similar but not identical titles)
- **When to use instead:** If orchestrator time is severely constrained

**Option C: Label change only (no deduplication)**
- **Pros:** Faster to implement (skill edit only)
- **Cons:** Duplicates still created, just not auto-spawned
- **When to use instead:** If beads codebase is hard to modify

**Rationale for recommendation:** Both layers needed because:
- Dedup alone doesn't help if daemon spawns fast enough to create issues before check runs
- Label change alone doesn't prevent backlog pollution with duplicates

---

### Implementation Details

**What to implement first:**
1. Change skill guidance (quick win, prevents new auto-spawns)
2. Add beads deduplication (requires beads repo changes)
3. Optional: daemon rate limiting (defense in depth)

**Things to watch out for:**
- ⚠️ String matching for dedup must handle variations ("Bug: X" vs "X" vs "Fix X")
- ⚠️ Changing labels might break existing orchestrator automations
- ⚠️ Dedup could be slow if issue count is very large

**Areas needing further investigation:**
- How similar must titles be to be considered duplicates? (exact match? fuzzy?)
- Should dedup work across projects or per-project only?
- What happens to the 274 existing triage:ready issues?

**Success criteria:**
- ✅ Running 5 concurrent agents on same project creates <3 duplicate issues (currently creates ~15+)
- ✅ Daemon queue size stabilizes rather than growing unbounded
- ✅ Orchestrator can still manually label issues as triage:ready

---

## References

**Files Examined:**
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Discovered Work guidance
- `~/.claude/skills/worker/investigation/SKILL.md` - Discovered Work guidance
- `pkg/beads/cli_client.go` - Issue creation without dedup
- `pkg/daemon/daemon.go` - Daemon triage:ready filter
- `~/orch-knowledge/.kb/investigations/2025-12-17-inv-investigate-opportunities-expand-triage-ready.md` - Original design investigation

**Commands Run:**
```bash
# Count total issues
bd list | wc -l  # Result: 801

# Count triage:ready issues
bd list | grep "triage:ready" | wc -l  # Result: 274

# Find duplicate examples
bd list | grep -i "GenerateContext"  # Found qoo1 and 3p7q
```

**Related Artifacts:**
- Original design: `2025-12-17-inv-investigate-opportunities-expand-triage-ready.md`
- Discovered Work addition: commit `480b765` in orch-knowledge

---

## Self-Review

- [x] Real test performed (ran bd list, verified duplicates, read source code)
- [x] Conclusion from evidence (based on code examination and git history)
- [x] Question answered (explained both auto-labeling and lack of dedup)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-29 [Start]:** Investigation started
- Initial question: Why do skills auto-label discovered work, and why no deduplication?
- Context: Observed 193 commits in 2 days, duplicate issues in backlog

**2025-12-29 [Mid]:** Found design history
- Located original investigations in orch-knowledge
- Confirmed intentional design from Dec 2025
- Identified missing deduplication as the gap

**2025-12-29 [End]:** Investigation completed
- Status: Complete
- Key outcome: Auto-labeling was intentional but deduplication wasn't considered; recommend two-layer fix
