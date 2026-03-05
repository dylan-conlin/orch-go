# Self-Review (Mandatory)

Before completing the audit, verify quality of findings and documentation.

---

## Audit-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence concrete** | Each finding has screenshot ref or snapshot excerpt | Add specific evidence |
| **Reproducible** | Playwright commands documented | Add browser commands used |
| **Severity justified** | Each severity matches definition table | Re-assess against criteria |
| **Actionable** | Each recommendation has specific fix | Make concrete |
| **Baseline captured** | Metrics table filled with counts | Add axe-core + finding counts |

---

## Self-Review Checklist

### 1. Findings Quality

- [ ] **Each finding has evidence** — Screenshot reference, snapshot excerpt, or computed style value
- [ ] **Viewports specified** — Each finding lists affected viewports
- [ ] **Severity assessed** — Each finding has Blocker/Major/Minor/Cosmetic
- [ ] **Severity matches definition** — Blockers are truly task-blocking, Cosmetic is truly invisible to users
- [ ] **False positives filtered** — Reviewed findings, removed non-issues or design choices

### 2. Recommendations Quality

- [ ] **Actionable** — Each recommendation specifies what to change
- [ ] **Scoped** — Recommendations are achievable (not "redesign everything")
- [ ] **Linked to findings** — Each recommendation traces to a specific finding

### 3. Evidence Quality

- [ ] **Screenshots captured** — At minimum, 5 baseline screenshots (one per viewport)
- [ ] **Screenshot index present** — Table listing all screenshots with viewport and state
- [ ] **axe-core results included** — Violation count, pass count, top violations with impact

### 4. Documentation Quality

- [ ] **Investigation file complete** — All required sections filled
- [ ] **Baseline metrics table** — Finding counts by severity + axe-core metrics
- [ ] **What Works Well section** — At least 2 positive findings (audits should acknowledge strengths)
- [ ] **Comparison section** — Prior audit comparison OR "First audit of this page"
- [ ] **Reproducibility section** — Auth method and re-audit schedule documented

### 5. Commit Hygiene

- [ ] Conventional format: `audit: UX audit — {page name} ({beads-id})`
- [ ] Investigation file committed
- [ ] Screenshots committed (or noted as excluded per policy)

### 6. Discovered Work Check

*UX audits typically discover actionable work. Track it in beads.*

| Severity | Action |
|----------|--------|
| **Blocker** | `bd create "UX BLOCKER: description" --type bug --priority 1` |
| **Major** | `bd create "UX: description" --type bug --priority 2` |
| **Minor** | `bd create "UX: description" --type task --priority 3` |
| **Cosmetic** | Include in investigation only (no separate issue unless part of a pattern) |

**Triage labeling:**

| Finding Severity | Label | Reason |
|-----------------|-------|--------|
| Blocker | `triage:ready` | Clear problem, needs immediate fix |
| Major | `triage:review` | Needs prioritization against other work |
| Minor/Cosmetic | No issue | Tracked in investigation file |

**Checklist:**
- [ ] **Reviewed findings** — Checked all Blocker/Major findings for beads issue creation
- [ ] **Tracked if applicable** — Created beads issues for high-severity items
- [ ] **Included in summary** — Completion comment mentions tracked issues (if any)

---

## Report via Beads

**If self-review finds issues:**
1. Fix them before proceeding
2. Report: `bd comments add <beads-id> "Self-review: Fixed [issue summary]"`

**If self-review passes:**
- Report: `bd comments add <beads-id> "Self-review passed - ready for completion"`

---

## Completion Criteria

Before marking complete:

- [ ] Self-review passed
- [ ] **Leave it Better completed** (see next section)
- [ ] Investigation file complete with all findings
- [ ] Baseline metrics documented
- [ ] Screenshots captured and indexed
- [ ] Discovered work reviewed and tracked
- [ ] All changes committed: `git status` shows "nothing to commit"
- [ ] Reported: `bd comments add <beads-id> "Phase: Complete - UX audit: {N} findings ({blocker}B/{major}M/{minor}m/{cosmetic}C), axe-core: {N} violations"`

**If ANY box unchecked, audit is NOT complete.**

---

**After completing all criteria:**

1. Verify all checkboxes marked
2. Report completion: `bd comments add <beads-id> "Phase: Complete - UX audit: [summary]"`
3. Call /exit to close agent session
