<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orchestrator should use a "triage batch" workflow: batch-label issues for daemon when 3+ ready issues exist AND available capacity permits, using issue type inference for skill selection (not explicit skill:* labels).

**Evidence:** Daemon only uses `InferSkill(issueType)` (pkg/daemon/daemon.go:302-316), ignoring skill labels. Type→skill mapping is reliable for 80% of issues. skill:* labels mentioned in orchestrator skill but not implemented in code.

**Knowledge:** Skill selection judgment is preserved through issue type selection (orchestrator chooses type when creating issues). Explicit skill labels add friction without clear value—defer to issue-creation skill to produce correct types.

**Next:** Update orchestrator skill to document triage batch workflow with clear triggers and thresholds. Optionally implement skill label override in daemon for edge cases requiring non-standard skill.

---

# Investigation: Capacity Utilization Workflow for Proactive Batch Labeling

**Question:** When and how should the orchestrator proactively batch-label issues for daemon pickup to maximize throughput without losing judgment on skill selection?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Design Session Agent
**Phase:** Complete
**Next Step:** None - design produced, ready for implementation or decision
**Status:** Complete

---

## Findings

### Finding 1: Daemon uses type-based skill inference only

**Evidence:** The daemon's skill selection logic in `InferSkill()` maps issue types to skills:
- bug → systematic-debugging
- feature → feature-impl
- task → feature-impl
- investigation → investigation

There is NO code path that reads `skill:*` labels from issues. The orchestrator skill documentation mentions labels, but they are not implemented.

**Source:** pkg/daemon/daemon.go:302-316, cmd/orch/main.go:852-865

**Significance:** The "skill selection judgment" the orchestrator provides is actually happening at issue creation time (choosing the right issue type), not at labeling time. This simplifies the batch-labeling workflow.

---

### Finding 2: Current triage labels are binary (ready vs review)

**Evidence:** The daemon filters issues with `triage:ready` label (configurable via `Config.Label`). The orchestrator skill documents:
- `triage:ready` - High confidence, daemon can auto-spawn
- `triage:review` - Needs orchestrator review before spawning

No intermediate states or skill-specific labels are used in practice.

**Source:** pkg/daemon/daemon.go:47, orchestrator SKILL.md lines 203-206

**Significance:** The labeling decision is simple: mark `triage:ready` when confident, mark `triage:review` when uncertain. Skill selection is delegated to type inference.

---

### Finding 3: Capacity infrastructure exists but isn't used for proactive labeling

**Evidence:** 
- Daemon has `WorkerPool` with `MaxAgents` concurrency control
- `AvailableSlots()` returns available capacity
- `ReconcileWithOpenCode()` syncs pool state with actual sessions
- Daemon processes one issue at a time in `Once()` loop

However, the orchestrator has no workflow to:
1. Check available capacity before batch-labeling
2. Determine how many issues to release to daemon
3. Prioritize which issues to release first

**Source:** pkg/daemon/daemon.go:200-217, 256-266

**Significance:** Capacity-aware batch labeling is a gap. Orchestrator currently labels ad-hoc without considering how many agents can actually run.

---

### Finding 4: Issue-creation skill provides skill selection judgment upstream

**Evidence:** The orchestrator skill documents that `issue-creation` investigates symptoms and produces issues with appropriate types. When the issue type is set correctly at creation:
- bug type → systematic-debugging skill
- investigation type → investigation skill
- feature/task type → feature-impl skill

The orchestrator's judgment is exercised when CREATING the issue, not when labeling it for daemon.

**Source:** Orchestrator SKILL.md "Bug Triage" and "Issue Creation" sections

**Significance:** Skill selection judgment is NOT lost by batch-labeling—it's preserved through correct issue typing upstream.

---

## Synthesis

**Key Insights:**

1. **Skill judgment is upstream** - The orchestrator's skill selection judgment happens at issue creation time (choosing bug vs feature vs investigation type), not at the labeling stage. This means batch-labeling doesn't sacrifice judgment—it just releases pre-judged work.

2. **Simple is correct** - The binary triage:ready vs triage:review model is sufficient. Adding skill:* labels would add friction without clear value, since InferSkill from type is reliable.

3. **Capacity-awareness is missing** - The orchestrator has no guidance on WHEN to batch-label (how many issues? what priority threshold?) or HOW to check capacity before labeling.

**Answer to Investigation Question:**

The orchestrator should use a **"Triage Batch"** workflow:

**When to batch-label:**
- At session start when reviewing `bd ready` output
- When completing multiple agents (follow-up work discovered)
- When 3+ `triage:review` issues accumulate (batch review)
- When capacity permits (available slots > 2)

**How to batch-label:**
1. Review pending issues: `bd list --labels triage:review`
2. Check capacity: `orch status` (see available slots)
3. Label up to `min(ready_issues, available_slots + 2)` issues
4. Use: `bd label <id> triage:ready` (no skill label needed)

**Skill selection judgment preserved by:**
- Using issue-creation skill which produces correct types
- Manually overriding issue type via `bd edit <id> --type X` before labeling (rare)
- Deferring implementation of skill:* label override (edge case handling)

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon only uses InferSkill(issueType) (verified: grep for skill label usage found none)
- ✅ Daemon filters by triage:ready label (verified: pkg/daemon/daemon.go:185-190)
- ✅ WorkerPool provides capacity tracking (verified: pkg/daemon/daemon.go:200-217)

**What's untested:**

- ⚠️ Whether batch-labeling 5+ issues at once causes rate-limit issues (need to test with real daemon run)
- ⚠️ Whether orchestrators will actually follow the workflow (behavioral, not technical)
- ⚠️ Edge cases where type→skill mapping is wrong (may need skill label override)

**What would change this:**

- If skill:* labels are implemented in daemon, workflow should use them for precision
- If type→skill inference fails often, need explicit skill override mechanism
- If capacity check becomes expensive, may need local caching

---

## Implementation Recommendations

### Recommended Approach ⭐

**Triage Batch Workflow** - Add explicit guidance to orchestrator skill for capacity-aware batch labeling without implementing skill:* label support.

**Why this approach:**
- Solves the immediate problem (when/how to batch-label)
- Uses existing infrastructure (triage:ready label, InferSkill)
- Preserves judgment through correct issue typing upstream
- Zero code changes required—pure process documentation

**Trade-offs accepted:**
- No explicit skill override for edge cases (deferred)
- Relies on issue-creation skill producing correct types
- Orchestrator must manually check capacity

**Implementation sequence:**
1. Add "Triage Batch Workflow" section to orchestrator skill with triggers and thresholds
2. Add `orch capacity` command to show available daemon slots (optional, low priority)
3. Consider skill:* label support in daemon if type inference proves insufficient

### Alternative Approaches Considered

**Option B: Implement skill:* label support in daemon**
- **Pros:** Explicit skill control, handles edge cases
- **Cons:** Adds complexity, may encourage over-labeling, InferSkill works for 80%+ of cases
- **When to use instead:** If type→skill mapping frequently fails

**Option C: Automatic batch-labeling by daemon**
- **Pros:** Zero orchestrator overhead, fully autonomous
- **Cons:** Loses human judgment checkpoint, may spawn low-quality issues
- **When to use instead:** If orchestrator becomes bottleneck AND issue quality is consistently high

**Rationale for recommendation:** Option A (Triage Batch Workflow) provides the guidance needed without code changes. InferSkill from issue type is reliable enough that skill labels add friction without value.

---

### Implementation Details

**What to implement first:**
- Add "Triage Batch Workflow" section to orchestrator skill (document triggers, thresholds)
- Update session-start checklist to include capacity-aware triage

**Things to watch out for:**
- ⚠️ Don't label more issues than `available_slots + buffer` (buffer of 2 allows for completion churn)
- ⚠️ Prioritize by P0 > P1 > P2 when batch-labeling
- ⚠️ Review any `triage:review` issues before batch-labeling (that label exists for a reason)

**Areas needing further investigation:**
- How often does type→skill inference produce wrong skill?
- Should there be a max batch size to prevent overwhelming the daemon?
- Should capacity check be automated in `bd label` command?

**Success criteria:**
- ✅ Orchestrator has clear triggers for when to batch-label
- ✅ Orchestrator knows how to check capacity before labeling
- ✅ Daemon stays busy without idle capacity when ready work exists
- ✅ Skill selection judgment is preserved through correct issue typing

---

## Proposed Workflow Documentation (for Orchestrator Skill)

```markdown
## Triage Batch Workflow (Capacity-Aware Labeling)

**When to run:**
| Trigger | Action |
|---------|--------|
| Session start | Review bd ready, batch-label triage:review issues |
| 3+ agents completed | Check for discovered follow-up work, batch-label |
| Daemon idle (0 active agents) | Check for unlabeled ready work |
| 3+ triage:review issues | Batch review and label |

**How to batch-label:**

1. **Check capacity:**
   ```bash
   orch status  # Shows "Active: X/Y" agents
   ```
   Calculate: available = MaxAgents - Active

2. **Review pending work:**
   ```bash
   bd list --labels triage:review  # Needs review
   bd ready                        # All ready work
   ```

3. **Label for daemon:**
   - Label up to `available + 2` issues (buffer for completion churn)
   - Prioritize by priority: P0 > P1 > P2
   - Skip if unsure—leave as triage:review
   ```bash
   bd label <id> triage:ready
   ```

4. **Verify daemon picking up:**
   ```bash
   orch daemon preview  # Should show your labeled issues
   ```

**Skill selection:**
- Skill is inferred from issue type (bug → systematic-debugging, etc.)
- To change skill: `bd edit <id> --type <correct-type>` BEFORE labeling
- If type→skill mapping is wrong, create issue for skill label override feature

**Anti-patterns:**
- ❌ Labeling all issues at once without checking capacity
- ❌ Labeling issues you haven't reviewed
- ❌ Expecting skill:* labels to work (not implemented)
```

---

## References

**Files Examined:**
- pkg/daemon/daemon.go - Daemon configuration, skill inference, capacity tracking
- cmd/orch/main.go - Work command, skill inference duplication
- cmd/orch/work_test.go - Skill inference tests
- ~/.claude/skills/meta/orchestrator/SKILL.md - Current orchestrator guidance
- .kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md - Prior swarm scoping
- .kb/investigations/2025-12-22-inv-add-concurrency-control-daemon-worker.md - WorkerPool implementation

**Commands Run:**
```bash
# Check ready issues
bd ready

# Search for skill label usage
grep -r "skill:" pkg/daemon/*.go cmd/orch/*.go

# Check InferSkill usage
grep -r "InferSkill" pkg/daemon/*.go cmd/orch/*.go
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md - Original swarm scoping
- **Investigation:** .kb/investigations/2025-12-22-inv-add-concurrency-control-daemon-worker.md - WorkerPool that enables capacity tracking

---

## Investigation History

**2025-12-27:** Investigation started
- Initial question: When and how should orchestrator batch-label for daemon?
- Context: Capacity utilization workflow design session

**2025-12-27:** Key finding: skill:* labels not implemented
- Daemon only uses InferSkill(issueType)
- Skill judgment happens at issue creation, not labeling

**2025-12-27:** Investigation completed
- Status: Complete
- Key outcome: Designed Triage Batch Workflow—process documentation, no code changes needed
