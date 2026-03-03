# Self-Review (Mandatory)

Before completing the audit, verify quality of findings and recommendations.

---

## Audit-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence concrete** | Each finding has file:line reference | Add specific locations |
| **Reproducible** | Pattern searches documented | Add grep/glob commands used |
| **Prioritized** | Recommendations ranked by ROI | Add severity/effort matrix |
| **Actionable** | Each recommendation has clear next step | Make specific |
| **Baseline captured** | Metrics for re-audit comparison | Add counts/percentages |

---

## Self-Review Checklist

### 1. Findings Quality

- [ ] **Each finding has evidence** - Concrete file:line references, not "there are issues"
- [ ] **Pattern searches documented** - grep/glob commands that found issues
- [ ] **False positives filtered** - Reviewed results, removed non-issues
- [ ] **Severity assessed** - Each finding has impact level (critical/high/medium/low)

### 2. Recommendations Quality

- [ ] **Prioritized by ROI** - High impact, low effort items first
- [ ] **Actionable** - Each recommendation specifies what to do
- [ ] **Scoped** - Recommendations are achievable (not "rewrite everything")
- [ ] **Linked to findings** - Each recommendation traces to specific findings

### 3. Documentation Quality

- [ ] **Investigation file complete** - All sections filled
- [ ] **Baseline metrics** - Numbers for future comparison
- [ ] **Reproduction commands** - Someone can re-run the audit
- [ ] **NOT DONE claims verified** - For each 'NOT DONE' or 'NOT IMPLEMENTED' finding, confirmed with file/code search (not just artifact reading)

### 4. Commit Hygiene

- [ ] Conventional format (`audit:` or `chore:`)
- [ ] Investigation file committed

### 5. Discovered Work Check

*Audits typically discover actionable work. Track it in beads so it doesn't get lost.*

| Type | Examples | Action |
|------|----------|--------|
| **Security bugs** | Vulnerabilities, injection risks | `bd create "SECURITY: description" --type bug` |
| **Architecture issues** | God objects, tight coupling, tech debt | `bd create "ARCHITECTURE: description" --type task` |
| **Performance issues** | N+1 queries, missing indexes | `bd create "PERFORMANCE: description" --type bug` |
| **Missing tests** | Coverage gaps, critical paths untested | `bd create "TESTING: description" --type task` |
| **Strategic Unknowns** | Architectural/premise questions discovered | `bd create "description" --type question` |

**Triage labeling for daemon processing:**

After creating issues, apply triage labels based on finding severity:

| Severity | Label | When to use |
|----------|-------|-------------|
| Critical/High | `triage:ready` | Clear problem, known fix approach, well-scoped |
| Medium/Low | `triage:review` | Needs orchestrator review before work starts |

Example:
```bash
bd create "SECURITY: SQL injection in api.py:123" --type bug
bd label <issue-id> triage:ready  # Critical severity, clear fix
```

**Why this matters:** Issues labeled `triage:ready` are automatically picked up by the work daemon for autonomous processing. Critical/High severity issues have clear scope and can be worked immediately; Medium/Low issues benefit from orchestrator review first.

**Checklist:**
- [ ] **Reviewed recommendations** - Checked audit recommendations for actionable items
- [ ] **Tracked if applicable** - Created beads issues for high-priority items (or noted "No actionable items")
- [ ] **Included in summary** - Completion comment mentions tracked issues (if any)

**If no actionable items:** Note "No beads issues created - recommendations are informational only" in completion comment.

**Why this matters:** Audits produce recommendations that often require follow-up work. Beads issues ensure these surface in SessionStart context rather than getting buried in audit files.

---

## Report via Beads

**If self-review finds issues:**
1. Fix them before proceeding
2. Report: `bd comment <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report: `bd comment <beads-id> "Self-review passed - ready for completion"`

**Checklist summary (verify mentally, report issues only):**
- Findings: Evidence with file:line, pattern searches documented, false positives filtered, severity assessed
- Recommendations: Prioritized by ROI, actionable, scoped, linked to findings
- Documentation: Investigation file complete, baseline metrics, reproduction commands
- Discovered work: Reviewed for actionable items, tracked in beads or noted "No actionable items"

**Only proceed to completion after self-review passes.**

---

## Completion Criteria

Before marking complete:

- [ ] Self-review passed
- [ ] **Leave it Better completed:** At least one `kb quick` command run OR noted as not applicable
- [ ] Investigation file complete with all findings
- [ ] Recommendations prioritized and actionable
- [ ] Baseline metrics documented for re-audit
- [ ] Pattern search commands documented (reproducibility)
- [ ] Discovered work reviewed and tracked (or noted "No actionable items")
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported via beads: `bd comment <beads-id> "Phase: Complete - [findings summary]"`

**If ANY box unchecked, audit is NOT complete.**

---

**After completing all criteria:**

1. Verify all checkboxes marked
2. Report completion: `bd comment <beads-id> "Phase: Complete - Audit findings: [count], Recommendations: [count]"`
3. Call /exit to close agent session
