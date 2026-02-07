## Summary (D.E.K.N.)

**Delta:** Daemon should auto-spawn reworks, but `triage:ready` should be cleared on reopen to force orchestrator review of failure classification.

**Evidence:** Analyzed daemon polling logic (checks label + status), spawn context comment surfacing (already includes non-Phase comments), and skill inference (supports `skill:*` label override). Reopened issues with `triage:ready` would auto-spawn immediately without review.

**Knowledge:** The pause point between failure detection and respawn is valuable - it lets orchestrator classify the failure mode and route to appropriate skill. Auto-spawn without review would repeat the same skill that failed.

**Next:** Implement orch-go-21240 (spawn context surfaces POST-COMPLETION-FAILURE prominently), add `failure:*` label handling, update `orch rework` to clear `triage:ready` and add failure classification labels.

**Authority:** architectural - Establishes cross-component pattern affecting daemon, spawn context, and completion verification systems.

---

# Investigation: Daemon Interaction with Reopened/Rework Issues

**Question:** How should the daemon interact with reopened issues from the rework pattern? Specifically: auto-spawn behavior, failure context surfacing, triage:ready handling, and failure classification -> skill inference.

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - recommendations ready for orchestrator review
**Status:** Complete

**Follow-up to:** `.kb/investigations/2026-02-04-inv-design-retry-rework-pattern-completed.md`

---

## Context

The retry/rework pattern (orch-go-21237) established:
1. Reopen original issue + add `POST-COMPLETION-FAILURE` comment
2. Classify failure (verification/implementation/spec/integration)
3. Spawn new attempt with failure context

This investigation addresses the **daemon interaction** gaps:
- Should daemon auto-spawn reopened issues?
- How does spawn context surface failure comments?
- Should `triage:ready` be cleared on reopen?
- How does failure classification affect skill inference?

---

## Findings

### Finding 1: Daemon Would Auto-Spawn Reopened Issues Immediately

**Evidence:** Current daemon behavior from `pkg/daemon/daemon.go:421-502`:
1. Daemon polls `bd ready` which returns issues with status `open` or `in_progress`
2. Filters for `triage:ready` label (`Config.Label = "triage:ready"`)
3. Checks issue type is spawnable (`IsSpawnableType` in skill_inference.go)
4. Spawns if capacity available

When `bd reopen` transitions an issue from `closed` -> `open`, if it still has `triage:ready` label:
- Status = `open` (check)
- Label = `triage:ready` (check)
- Type = original type (bug/feature/task) (check)

**Result:** Daemon would spawn it on next poll cycle (within 60 seconds).

**Source:** `pkg/daemon/daemon.go:421-502`, `pkg/daemon/skill_inference.go:9-42`

**Significance:** Without intervention, reworked issues auto-spawn with the **same skill** that failed. This may be undesirable - a verification failure might not need the same skill, a spec issue needs investigation first.

---

### Finding 2: Spawn Context Already Surfaces Non-Phase Comments

**Evidence:** From `cmd/orch/spawn_validation.go:330-361`:

```go
func fetchIssueCommentsForSpawn(beadsID string) []spawn.IssueComment {
    // ...
    for _, c := range beadsComments {
        // Skip Phase: comments (progress tracking, not guidance)
        if strings.HasPrefix(c.Text, "Phase:") {
            continue
        }
        // Skip empty comments
        if strings.TrimSpace(c.Text) == "" {
            continue
        }
        comments = append(comments, spawn.IssueComment{...})
    }
    return comments
}
```

These comments appear in SPAWN_CONTEXT.md as "ORCHESTRATOR NOTES".

**Result:** `POST-COMPLETION-FAILURE` comments would be surfaced, but:
1. Not prominently (buried in general notes section)
2. No special formatting to highlight failure context
3. No suggested diagnostic focus based on failure type

**Source:** `cmd/orch/spawn_validation.go:330-361`, `pkg/spawn/context.go:105-116`

**Significance:** Partial solution exists. Issue orch-go-21240 correctly identifies need for prominent surfacing with failure-type-specific guidance.

---

### Finding 3: Skill Inference Supports Label Overrides

**Evidence:** From `pkg/daemon/skill_inference.go:81-100`:

```go
func InferSkillFromIssue(issue *Issue) (string, error) {
    // First, check for explicit skill:* label
    if skill := InferSkillFromLabels(issue.Labels); skill != "" {
        return skill, nil
    }
    // Check for title-based patterns
    if skill := InferSkillFromTitle(issue.Title); skill != "" {
        return skill, nil
    }
    // Fall back to type-based inference
    return InferSkill(issue.IssueType)
}
```

**Priority order:**
1. `skill:*` label (explicit override)
2. Title pattern matching
3. Issue type inference

**Result:** Failure classification can influence skill via labels:
- Add `skill:investigation` for spec issues
- Add `skill:reliability-testing` for integration issues
- Keep type-based inference for implementation bugs

**Source:** `pkg/daemon/skill_inference.go:81-100`, `InferSkillFromLabels` at :47-54

**Significance:** Infrastructure exists for failure-type -> skill mapping. Just need to connect `orch rework` classification to daemon label handling.

---

### Finding 4: beads Reopen Preserves Labels

**Evidence:** From `bd reopen --help`:
```
Reopen closed issues by setting status to 'open' and clearing the closed_at timestamp.
This is more explicit than 'bd update --status open' and emits a Reopened event.
```

Labels are NOT automatically modified on reopen - only status and closed_at.

**Result:** If issue had `triage:ready` before closure, it retains `triage:ready` after reopen.

**Source:** `bd reopen --help`, beads state machine behavior

**Significance:** Without explicit label management in `orch rework`, reopened issues become daemon-spawnable immediately. The "pause for review" must be implemented via label clearing.

---

## Decision Forks

### Fork 1: Should Daemon Auto-Spawn Reworks?

**Options:**
- A: Yes, auto-spawn reworks (reopened issues with triage:ready)
- B: No, require orchestrator re-triage
- C: Conditional based on failure type

**Substrate says:**
- **Session Amnesia:** Agents need failure context to avoid repeating mistakes
- **Gate Over Remind:** Enforcement through gates, not reminders
- **Decidability:** Unknown decision forks need human judgment

**RECOMMENDATION:** Option B - Require orchestrator re-triage

**Reasoning:**
- The failure classification (verification/implementation/spec/integration) affects the appropriate skill
- Auto-spawning with same skill may repeat the failure
- The pause between failure detection and respawn is valuable for orchestrator review
- Gate enforced by clearing `triage:ready` on reopen

**Trade-off accepted:** Slightly slower rework cycle (requires orchestrator action). This is a feature - forces conscious decision about remediation approach.

---

### Fork 2: How Does Spawn Context Surface Failure Comments?

**Options:**
- A: Current approach (buried in ORCHESTRATOR NOTES)
- B: Prominent section with failure-specific guidance (orch-go-21240)
- C: Dedicated SPAWN_CONTEXT section with skill suggestion

**Substrate says:**
- **Progressive Disclosure:** TLDR first, details available
- **Surfacing Over Browsing:** Bring relevant state to the agent

**RECOMMENDATION:** Option B - Implement orch-go-21240 with enhancement

When spawning for an issue with `POST-COMPLETION-FAILURE` comment:

```markdown
## REWORK ATTEMPT - Previous attempt failed

**Prior Failure:** [extracted from POST-COMPLETION-FAILURE comment]
**Failure Type:** [verification/implementation/spec/integration]
**Attempt Number:** [2, 3, etc. - from attempt tracking]

**Diagnostic Focus:**
- [Type-specific guidance based on failure classification]
```

**Trade-off accepted:** More complex spawn context generation. Worth it for agent effectiveness.

---

### Fork 3: Should triage:ready Be Cleared on Reopen?

**Options:**
- A: Clear on reopen (require explicit re-labeling)
- B: Keep on reopen (auto-spawn continues)
- C: Clear only for certain failure types

**Substrate says:**
- **Gate Over Remind:** Gates must be passable by the gated party
- **Decidability:** Unknown forks need human judgment

**RECOMMENDATION:** Option A - Clear `triage:ready` on reopen

**Implementation:** `orch rework` command:
1. Removes `triage:ready` label
2. Adds `failure:*` label (classification)
3. Orchestrator must re-add `triage:ready` to release to daemon

**Why this pattern:**
- Forces review of failure classification
- Prevents immediate re-spawn with potentially wrong skill
- Allows orchestrator to spawn manually with context if urgent
- Gate is easy to pass (just `bd label <id> triage:ready`)

**Trade-off accepted:** Extra step for orchestrator. Intentional friction that prevents blind retry loops.

---

### Fork 4: How Does Failure Classification Affect Skill Inference?

**Options:**
- A: No effect (use original issue type)
- B: `failure:*` labels map to skill overrides
- C: `orch rework` adds explicit `skill:*` label

**Substrate says:**
- **Skill inference priority:** skill:* label > title pattern > issue type
- **Existing infrastructure:** InferSkillFromLabels already checks for skill:* labels

**RECOMMENDATION:** Option C - `orch rework` adds explicit `skill:*` label based on classification

**Mapping:**

| Failure Type | Skill Label | Rationale |
|--------------|-------------|-----------|
| `--verification-failure` | (none - keep type inference) | Same skill, just needs to actually verify |
| `--implementation-bug` | (none - keep type inference) | Try again with same approach |
| `--spec-issue` | `skill:investigation` | Understand what's wrong before fixing |
| `--integration-issue` | `skill:reliability-testing` | Systematic validation needed |

**Implementation:** `orch rework` prompts for classification, adds appropriate label:
- `failure:verification` - no skill override
- `failure:implementation` - no skill override  
- `failure:spec` + `skill:investigation` - force investigation first
- `failure:integration` + `skill:reliability-testing` - force testing skill

**Trade-off accepted:** Orchestrator must classify correctly. Better than automated guessing.

---

## Synthesis

**Key Insights:**

1. **The pause is the feature** - The time between failure detection and respawn is valuable. It forces orchestrator to review, classify, and consciously decide on remediation approach. Auto-spawning with same skill that failed is likely to fail again.

2. **Infrastructure mostly exists** - Spawn context already surfaces comments (just not prominently). Skill inference already supports label overrides. `bd reopen` already works. The work is connecting these pieces, not building new primitives.

3. **Classification drives routing** - The failure type determines the appropriate skill. Not all failures need the same approach. Spec issues need investigation before implementation. Integration issues need systematic testing. This classification must be explicit, not inferred.

4. **Labels are the control plane** - `triage:ready` controls daemon spawning. `skill:*` controls skill inference. `failure:*` provides audit trail. The rework flow manipulates labels to achieve desired behavior.

**Answer to Investigation Question:**

The daemon should NOT auto-spawn reopened issues. Instead:

1. **`orch rework` clears `triage:ready`** - Forces orchestrator review
2. **`orch rework` adds `failure:X` label** - Records classification
3. **`orch rework` adds `skill:Y` label** - When failure type indicates different skill
4. **Spawn context surfaces failure prominently** - POST-COMPLETION-FAILURE shown at top
5. **Orchestrator re-labels `triage:ready`** - Explicit decision to release to daemon

This creates a controlled rework cycle:
```
Feature fails validation
    |
Orchestrator runs: orch rework <id> --spec-issue
    |
triage:ready removed, failure:spec + skill:investigation added
POST-COMPLETION-FAILURE comment added
Issue reopened
    |
Orchestrator reviews, adds triage:ready
    |
Daemon spawns investigation skill (not original feature-impl)
    |
Investigation completes, creates child task for fix
```

---

## Structured Uncertainty

**What's tested:**

- Daemon filters by `triage:ready` label (verified: read pkg/daemon/daemon.go:491-502)
- Spawn context includes non-Phase comments (verified: read cmd/orch/spawn_validation.go:330-361)
- Skill inference checks `skill:*` labels first (verified: read pkg/daemon/skill_inference.go:81-100)
- `bd reopen` preserves labels (verified: bd reopen --help shows only status/closed_at affected)

**What's untested:**

- Performance impact of additional label operations (not benchmarked)
- Edge case: issue reopened manually without `orch rework` (would retain triage:ready)
- Attempt tracking increments correctly on reopened issues (assumed based on events.jsonl design)

**What would change this:**

- If beads adds automatic label management on reopen (unlikely)
- If orchestrator wants fully autonomous retry cycles (would need different pattern)
- If failure classification proves too coarse (would need finer-grained skill mapping)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Clear triage:ready on reopen | architectural | Affects daemon behavior, spawn workflow, orchestrator UX |
| Spawn context POST-COMPLETION-FAILURE surfacing | implementation | Enhancement within existing spawn context system |
| Failure type -> skill mapping | architectural | Creates new cross-component pattern |
| Add failure:* labels | implementation | Extends existing labeling patterns |

### Recommended Approach

**Controlled Rework Flow** - `orch rework` manages labels to gate daemon spawning while routing to appropriate skill.

**Why this approach:**
- Uses existing infrastructure (labels, skill inference, spawn context)
- Enforces review without blocking urgent manual spawns
- Enables classification-driven skill routing
- Creates audit trail via failure:* labels

**Trade-offs accepted:**
- Requires orchestrator action for each rework (intentional gate)
- Manual spawn still possible without rework flow (escape hatch exists)

**Implementation sequence:**

1. **Update `orch rework` to manage labels** (orch-go-21239 enhancement)
   - Clear `triage:ready` label before reopening
   - Add `failure:X` label based on classification
   - Add `skill:Y` label when failure type indicates different skill
   - This blocks daemon auto-spawn while preserving context

2. **Implement orch-go-21240** (spawn context enhancement)
   - Detect POST-COMPLETION-FAILURE comments
   - Surface prominently in SPAWN_CONTEXT.md
   - Include failure-type-specific diagnostic guidance
   - Show attempt number from attempt tracking

3. **Add failure:* label documentation**
   - Update daemon guide with failure labels
   - Document skill mapping in decision record

### Alternative Approaches Considered

**Option B: Auto-spawn with enhanced skill inference**
- **Pros:** Faster rework cycle, less orchestrator overhead
- **Cons:** May repeat failures, no human review checkpoint
- **When to use instead:** If rework success rate is high and review overhead proves costly

**Option C: Conditional auto-spawn based on failure type**
- **Pros:** Auto-spawn for simple failures (verification), gate for complex (spec)
- **Cons:** Complexity in classification, partial review coverage
- **When to use instead:** If initial approach proves too slow for simple retries

---

### Implementation Details

**What to implement first:**
1. Update `orch rework` command to clear `triage:ready` (orch-go-21239)
   - This immediately gates daemon auto-spawn
2. Then spawn context enhancement (orch-go-21240)
   - Surfaces failure context for manually spawned agents immediately

**Things to watch out for:**
- Edge case: Issue reopened via `bd reopen` directly (without `orch rework`) - would retain labels
- Ensure POST-COMPLETION-FAILURE format is documented for consistency
- Test that attempt tracking sees reopened issues correctly

**Success criteria:**
- Reopened issues do NOT auto-spawn (daemon logs show "missing label triage:ready")
- Spawn context shows prominent failure section for rework attempts
- `orch daemon preview` shows correct skill inference from failure:* + skill:* labels
- Orchestrator can release to daemon via `bd label <id> triage:ready`

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go:421-502` - Issue filtering and spawn logic
- `pkg/daemon/skill_inference.go` - Skill inference from type/labels
- `cmd/orch/spawn_validation.go:330-361` - Comment fetching for spawn context
- `pkg/spawn/context.go:105-116` - ORCHESTRATOR NOTES section in template
- `.kb/investigations/2026-02-04-inv-design-retry-rework-pattern-completed.md` - Prior rework pattern design

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-04-inv-design-retry-rework-pattern-completed.md` - Parent investigation
- **Issue:** orch-go-21239 - Implement orch rework command
- **Issue:** orch-go-21240 - Spawn context surfaces POST-COMPLETION-FAILURE comments
- **Issue:** orch-go-21241 - Add ReopenedCount to attempt tracking

---

## Investigation History

**2026-02-04 10:45:** Investigation started
- Initial question: How does daemon interact with reopened/rework issues?
- Context: Follow-up to orch-go-21237 rework pattern design

**2026-02-04 11:15:** Codebase exploration
- Read daemon polling logic
- Read spawn context comment handling
- Read skill inference priority

**2026-02-04 11:30:** Fork navigation
- Identified 4 decision forks
- Consulted substrate (Session Amnesia, Gate Over Remind, Decidability)
- Synthesized recommendations

**2026-02-04 11:45:** Investigation completed
- Status: Complete
- Key outcome: Daemon should NOT auto-spawn reworks; triage:ready cleared on reopen creates review gate; failure classification drives skill routing via labels
