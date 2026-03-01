# Design: Daemon-Agreements Integration

**Status:** In Progress
**Phase:** Complete
**Date:** 2026-02-28
**Issue:** orch-go-mdb1
**Type:** Architect design

---

## Problem Statement

There is a gap between agreement failure detection and correction. Today:
- `kb agreements check` runs at spawn time (Phase 3, warning-only)
- Failed agreements surface warnings, but someone must manually notice, create an issue, and let the daemon spawn a fix agent
- The daemon polls `bd ready` every 15s and spawns `triage:ready` issues, but has no mechanism to detect agreement violations and create work

**Goal:** Close the loop: agreement fails → issue auto-created → daemon spawns fix → agreement passes. No human needed for mechanical corrections.

**Success Criteria:**
1. Failed agreements automatically produce beads issues
2. No duplicate issues for the same failing agreement
3. Severity-appropriate gating (not all failures warrant auto-fix)
4. Cross-project awareness
5. Natural verification via re-check on next cycle
6. Follows existing daemon periodic task patterns

---

## Design Forks

### Fork 1: Where in the daemon loop?

**Options:**
- A: New periodic task (like knowledge_health) with its own interval
- B: Piggyback on existing task (e.g., reflection)
- C: Run on every poll cycle

**Substrate says:**
- **Pattern:** Every independent concern in the daemon gets its own periodic task with Enable/Interval config (8 existing tasks all follow this)
- **Constraint:** `daemon.go runDaemonLoop must be extracted before adding new subsystems` — but this adds to the already-extracted `daemon_periodic.go`, not to `runDaemonLoop` itself
- **Principle (Gate Over Remind):** Agreement checking should be gated into the workflow, not a reminder

**RECOMMENDATION:** Option A — New periodic task. Interval: **30 minutes** (agreements don't change frequently; command-type checks can be expensive). This follows the established pattern exactly and keeps the concern isolated.

**Trade-off accepted:** Another periodic task adds ~3 config fields and ~20 lines to daemon structs. At 8 existing tasks, one more is tolerable but we're approaching the point where the periodic task system itself needs abstraction.

### Fork 2: Dedup — Preventing duplicate issues

**Options:**
- A: Label-based dedup via `bd list --status=open -l agreement:<id>`
- B: In-memory tracker (like spawn_tracker)
- C: Both

**Substrate says:**
- **Pattern (knowledge_health):** Uses `bd list --status=open -l area:knowledge` + title matching for dedup
- **Constraint (daemon dedup):** Content-aware dedup was the fix for broken daemon dedup — need to match on agreement ID, not just title
- **Constraint (extraction convergence):** "Never create extraction for a file if extraction was already attempted" — same principle applies: never create agreement-fix issue if one already exists

**RECOMMENDATION:** Option A — Label-based dedup with `agreement:<agreement-id>` label. Each agreement gets a unique label on its auto-created issue. Before creating, check `bd list --status=open -l agreement:<id>`. This is simpler than in-memory tracking because agreement checks run every 30 minutes (no race condition window like spawn dedup's 15s cycle).

**Trade-off accepted:** One extra `bd list` call per failing agreement per check cycle. At 1-10 agreements, this is negligible.

### Fork 3: Severity gating — What triggers auto-creation?

**Options:**
- A: Only severity:error auto-creates
- B: Both error and warning auto-create
- C: New agreement YAML field `auto_fix: true/false` for opt-in
- D: Severity-based default with agreement-level override

**Substrate says:**
- **Principle (Gate Over Remind):** If we can gate fixes into the workflow, we should
- **Principle (Pressure Over Compensation):** Don't compensate for staleness by pasting answers — let maintenance touchpoints create natural pressure
- **Existing behavior:** Spawn-time agreement checks are WARNING-ONLY (never block). Auto-fix is a step beyond warning.
- **Agreement schema:** No `auto_fix` field currently exists

**RECOMMENDATION:** Option D — Severity-based default with agreement-level override.

Default behavior:
- `severity: error` → auto-creates issue (errors are contract violations that need fixing)
- `severity: warning` → no auto-creation (informational, may not warrant agent work)
- `severity: info` → no auto-creation

Override via new optional YAML field:
```yaml
auto_fix: true    # warning agreement opts into auto-fix
auto_fix: false   # error agreement opts out of auto-fix
```

When `auto_fix` is not specified, severity determines behavior. When specified, it overrides severity default.

**Trade-off accepted:** Requires schema change to agreement YAML (additive, backwards-compatible). Agreement authors need to understand the default behavior.

### Fork 4: Issue shape — What context does the fix agent get?

**Options:**
- A: Minimal issue (title + agreement ID)
- B: Rich issue (contract + check output + parties + fix guidance)
- C: Issue + dedicated SPAWN_CONTEXT supplement

**Substrate says:**
- **Agreement YAML fields:** `contract` (what should be true), `check.run` (how to verify), `description` (why it matters), `failure_mode` (what kind of drift), `parties` (source/consumer)
- **Daemon skill inference:** type=task → investigation. Agent gets standard spawn context + issue description
- **Principle (Session Amnesia):** The fix agent starts fresh — it needs enough context to understand the problem and fix it

**RECOMMENDATION:** Option B — Rich issue description with structured context.

Issue template:
```
Title: Agreement violation: {title} ({id})
Type: task
Priority: 2 (error) or 3 (warning with auto_fix:true)
Labels: triage:ready, agreement:{id}, area:agreements

Description:
## Agreement Violation: {title}

**Severity:** {severity}
**Failure Mode:** {failure_mode}
**Agreement ID:** {id}

### Contract
{contract field content}

### Check Output (Failure Details)
{message from check result}

### Parties
- **Source:** {parties.source.project} — {parties.source.artifact}
- **Consumer:** {parties.consumer.project} — {parties.consumer.artifact}

### Fix Guidance
Fix the code or documentation to satisfy the contract above.
After fixing, verify with: kb agreements check
```

The `contract` field is the most valuable piece — it tells the agent WHAT should be true in plain language. The check output tells WHAT is wrong. Together, they provide sufficient context.

**Trade-off accepted:** Longer issue descriptions increase `bd create` command complexity. Worth it for agent context quality.

### Fork 5: Cross-project — How does auto-fix work when fix is in different repo?

**Options:**
- A: Only create issues for same-project agreements
- B: Create issue in the project where the agreement YAML lives
- C: Create issue in the source project (where the fix likely needs to happen)

**Substrate says:**
- **Constraint:** "Agreement checks for cross-project data contracts must run against consumer-facing interface (API endpoint), not source of truth"
- **Pattern (worker-base):** Cross-repo issue handoff via `CROSS_REPO_ISSUE` blocks
- **Project groups:** Daemon supports cross-project operation, but `kb agreements check` runs per-project

**RECOMMENDATION:** Option B — Create issue in the project where the agreement YAML lives.

**Rationale:**
1. `kb agreements check` runs in a specific project directory
2. The agreement YAML is in that project
3. The check failure is detected in that project
4. The spawned agent works in that project's directory
5. If the fix is in a different repo, the agent discovers this and uses the cross-repo issue handoff protocol (existing pattern)

This avoids the complexity of the daemon trying to figure out which repo needs the fix. The agreement's `parties` field documents the source, but the daemon shouldn't try to cross project boundaries for issue creation — that's the agent's job.

**Trade-off accepted:** Fix agents for cross-project agreements may need to escalate or use handoff protocol instead of fixing directly. This adds a step but keeps the daemon simple.

### Fork 6: Fix verification — Re-run check after fix completes?

**Options:**
- A: No verification (trust the fix agent)
- B: Re-run agreement check after issue closed (natural cycle)
- C: Special completion-time verification gate

**Substrate says:**
- **Daemon completion constraint:** "Daemon completion path labels daemon:ready-review but does NOT close issues — orchestrator must still run orch complete"
- **Existing flow:** Daemon detects Phase: Complete → marks ready-review → orchestrator runs orch complete → closes issue
- **Agreement check cycle:** Runs every 30 minutes regardless

**RECOMMENDATION:** Option B — Natural re-check via the periodic cycle.

The verification is free: the agreement check runs every 30 minutes. After the fix agent completes and the issue is closed:
- Next agreement check runs → if agreement passes, nothing happens (expected)
- Next agreement check runs → if agreement still fails, dedup check finds no open issue → creates new issue

This creates a natural feedback loop with zero additional implementation. The self-healing property emerges from the combination of periodic checking + dedup + auto-creation.

**Trade-off accepted:** Up to 30-minute delay between fix completion and verification. Acceptable for maintenance work.

---

## Recommended Design

### Architecture

```
┌──────────────────────────────────────────────────────┐
│ Daemon Periodic Tasks                                 │
│                                                       │
│  ... existing tasks ...                               │
│                                                       │
│  ┌───────────────────────────────────────────────┐   │
│  │ Agreement Check (every 30 min)                 │   │
│  │                                                │   │
│  │  1. Run kb agreements check --json             │   │
│  │  2. Filter failures by severity + auto_fix     │   │
│  │  3. For each actionable failure:               │   │
│  │     a. Dedup: bd list -l agreement:<id>        │   │
│  │     b. If no open issue: bd create with        │   │
│  │        rich context from agreement YAML        │   │
│  │                                                │   │
│  └───────────────────────────────────────────────┘   │
│                                                       │
│  ... rest of daemon loop ...                          │
│                                                       │
│  Poll bd ready → finds triage:ready agreement issues  │
│  Daemon spawns fix agent → agent fixes → completes    │
│  Next agreement check verifies fix                    │
│                                                       │
└──────────────────────────────────────────────────────┘
```

### Implementation Plan

#### Phase 1: Agreement YAML Schema Extension

**File:** kb-cli agreements (schema change)

Add optional `auto_fix` field to agreement YAML:

```yaml
auto_fix: true   # Override severity default for auto-creation
auto_fix: false  # Prevent auto-creation even for errors
# Omitted: use severity default (error=true, warning/info=false)
```

This is a **cross-project change** (kb-cli owns the agreement schema). The orch-go daemon consumes the existing JSON output, which already includes severity. The `auto_fix` field needs to be added to:
1. kb-cli agreement YAML parser
2. kb-cli `kb agreements check --json` output
3. orch-go daemon consumer

#### Phase 2: Daemon Periodic Task (orch-go)

**New files:**
- `pkg/daemon/agreement_check.go` — Check logic, result types, issue creation

**Modified files:**
- `pkg/daemonconfig/config.go` — Add AgreementCheck config fields
- `pkg/daemon/daemon.go` — Add state field (`lastAgreementCheck time.Time`)
- `cmd/orch/daemon_periodic.go` — Add handler
- `cmd/orch/daemon.go` — Add CLI flags (if needed) and snapshot to status

**Config fields:**
```go
// AgreementCheckEnabled controls periodic agreement checking.
AgreementCheckEnabled bool

// AgreementCheckInterval is how often to check agreements (default: 30 minutes).
AgreementCheckInterval time.Duration
```

**Result type:**
```go
type AgreementCheckResult struct {
    Total        int                // Total agreements checked
    Passed       int                // Agreements passing
    Failed       int                // Agreements failing
    IssuesCreated int               // New issues created this cycle
    Skipped      int                // Failures skipped (open issue exists)
    Failures     []AgreementFailureDetail
    Error        error
    Message      string
}

type AgreementFailureDetail struct {
    AgreementID string
    Title       string
    Severity    string
    AutoFix     *bool   // nil = use severity default
    Message     string  // Check output
    Contract    string  // From agreement YAML
    IssueCreated bool   // Whether issue was created this cycle
    SkipReason  string  // Why issue wasn't created (dedup, severity, etc.)
}
```

#### Phase 3: Reuse Existing AgreementsChecker

The `AgreementsChecker` function type already exists in `pkg/spawn/gates/agreements.go`. The daemon should reuse `buildAgreementsChecker()` from `cmd/orch/kb.go` to get the checker function, then extend the result parsing to extract `contract`, `auto_fix`, and `parties` for issue creation.

**Key insight:** The current `--json` output from `kb agreements check` includes `agreement_id`, `title`, `severity`, `pass`, `message` — but NOT `contract`, `auto_fix`, or `parties`. The daemon needs the full agreement YAML context, not just the check result.

**Two approaches:**
1. **Extend `kb agreements check --json`** to include `contract`, `auto_fix`, `parties` in output
2. **Parse agreement YAMLs directly** in addition to running checks

**Recommendation:** Approach 1 — extend the JSON output. This keeps the daemon as a consumer of kb-cli's output (consistent with existing architecture). The kb-cli already parses the YAMLs; it just needs to include more fields in the JSON output.

#### Phase 4: Issue Creation with Dedup

```go
func (d *Daemon) createAgreementIssue(failure AgreementFailureDetail) error {
    // Dedup check: look for open issue with this agreement's label
    listCmd := exec.Command("bd", "list", "--status=open", "-l",
        fmt.Sprintf("agreement:%s", failure.AgreementID))
    listOutput, err := listCmd.Output()
    if err == nil {
        lines := strings.Split(string(listOutput), "\n")
        for _, line := range lines {
            if strings.TrimSpace(line) != "" {
                return nil // Already have an open issue
            }
        }
    }
    // If bd list fails, proceed with creation (fail-open)

    // Determine priority from severity
    priority := "2" // error default
    if failure.Severity == "warning" {
        priority = "3"
    }

    // Build rich description
    description := buildAgreementIssueDescription(failure)

    cmd := exec.Command("bd", "create",
        "--title", fmt.Sprintf("Agreement violation: %s (%s)", failure.Title, failure.AgreementID),
        "--type", "task",
        "--priority", priority,
        "-l", "triage:ready",
        "-l", fmt.Sprintf("agreement:%s", failure.AgreementID),
        "-l", "area:agreements",
    )
    // Note: bd create may need description via stdin or flag
    ...
}
```

### Defaults

| Config | Default | Rationale |
|--------|---------|-----------|
| `AgreementCheckEnabled` | `true` | Gate Over Remind — if agreements exist, check them |
| `AgreementCheckInterval` | `30 * time.Minute` | Balance between responsiveness and check cost |

### Event Tracking

New event type: `daemon.agreement_check`

```json
{
  "type": "daemon.agreement_check",
  "timestamp": 1709150400,
  "data": {
    "total": 6,
    "passed": 5,
    "failed": 1,
    "issues_created": 1,
    "skipped": 0,
    "message": "Agreement check: 5/6 passed, 1 issue created"
  }
}
```

---

## Blocking Questions

### Q1: Should `auto_fix` field be added to kb-cli agreement schema?

- **Authority:** architectural (cross-project schema change)
- **Subtype:** judgment
- **What changes based on answer:**
  - Yes → kb-cli schema change needed first (dependency)
  - No → severity is the only gating mechanism (simpler, less flexible)

**Recommendation:** Yes. The field is additive (backward-compatible), and gives agreement authors control over which violations warrant automatic agent intervention.

### Q2: Should the daemon run agreement checks for all projects in the group or only the current project?

- **Authority:** architectural (cross-project scope)
- **Subtype:** judgment
- **What changes based on answer:**
  - All projects → daemon needs to iterate `kb agreements check` per project directory (more complex, more comprehensive)
  - Current only → simpler but misses agreements in other repos

**Recommendation:** Current project only (where daemon runs). Multi-project agreement checking can be added later when the daemon's cross-project operation matures. The daemon already has the `--group` flag for multi-project polling — agreement checking can follow the same pattern in a future iteration.

### Q3: Should `kb agreements check --json` be extended to include `contract`, `auto_fix`, and `parties` fields?

- **Authority:** architectural (cross-project API change)
- **Subtype:** factual (depends on whether daemon needs this data)
- **What changes based on answer:**
  - Yes → kb-cli change needed, daemon gets rich context for issue creation
  - No → daemon parses agreement YAMLs directly (duplicates kb-cli's parsing)

**Recommendation:** Yes. This follows the existing pattern where orch-go consumes kb-cli's structured output rather than reimplementing parsing.

---

## Implementation Issues

These issues should be created after this design is accepted:

### Issue 1: Add `auto_fix` field to kb-cli agreement schema
- **Type:** feature
- **Project:** kb-cli
- **Priority:** 2
- **Description:** Add optional `auto_fix: true|false` field to agreement YAML schema. Update `kb agreements check --json` output to include `auto_fix`, `contract`, and `parties` fields for each agreement result.

### Issue 2: Implement daemon agreement check periodic task
- **Type:** feature
- **Project:** orch-go
- **Priority:** 2
- **Depends on:** Issue 1
- **Description:** Add `AgreementCheckEnabled` and `AgreementCheckInterval` config, create `pkg/daemon/agreement_check.go` with check/dedup/create logic, add handler to `daemon_periodic.go`. Follow knowledge_health pattern.

### Issue 3: Add agreement check CLI flags to daemon
- **Type:** task
- **Project:** orch-go
- **Priority:** 3
- **Depends on:** Issue 2
- **Description:** Add `--agreement-check` / `--no-agreement-check` and `--agreement-interval` flags to `orch daemon run`. Update daemon.md guide and CLAUDE.md.

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision establishes the pattern for daemon self-healing (future: test failures, lint violations)
- Future agents building similar detection→fix loops should reference this design

**Suggested blocks keywords:**
- daemon agreements
- agreement auto-fix
- daemon self-healing
- daemon periodic task

---

## Recommendations

**RECOMMENDED:** Daemon periodic agreement check with severity-based auto-fix and agreement-level override

- **Why:** Closes the detection→correction gap with zero human intervention for mechanical fixes. Follows established daemon patterns exactly. Natural verification via periodic re-check.
- **Trade-off:** Adds another periodic task to an already-busy daemon. Schema change to kb-cli required.
- **Expected outcome:** Failed agreements auto-create `triage:ready` issues → daemon spawns fix agents → agreements pass. Self-healing property emerges from the combination of periodic checking + dedup + auto-creation.

**Alternative: Manual agreement triage**
- **Pros:** No code changes, human reviews all violations
- **Cons:** Defeats the purpose of agreements — they become reminders, not gates
- **When to choose:** If agreement violations are too nuanced for agent auto-fix

**Alternative: Spawn-time blocking (upgrade agreements from warning to gate)**
- **Pros:** Prevents spawning when agreements are violated
- **Cons:** Blocks all work, not just the fix. Too aggressive for most violations.
- **When to choose:** For critical error-severity agreements where spawning would make things worse

---

## Prerequisites

1. **Daemon.go extraction constraint:** The constraint says `cmd/orch/daemon.go runDaemonLoop must be extracted before adding new daemon subsystems`. This design adds to the already-extracted `daemon_periodic.go`, not to `runDaemonLoop` itself. However, the constraint should be reviewed — at 1174 lines, daemon.go is approaching the 1500-line threshold and any implementation that touches it needs architectural awareness.

2. **kb-cli schema change:** The `auto_fix` field and extended JSON output need to land in kb-cli before orch-go can consume them. This is Issue 1 above.
