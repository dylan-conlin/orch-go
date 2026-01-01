## Summary (D.E.K.N.)

**Delta:** 26 agent-related investigations cluster into 5 themes: (1) State Management (4 investigations), (2) Dashboard UI (12 investigations), (3) Cross-Project Visibility (4 investigations), (4) Agent Lifecycle (3 investigations), (5) Process/Skill Gaps (3 investigations).

**Evidence:** Read all 26 investigations, identified recurring patterns and architectural themes spanning Dec 20-31, 2025.

**Knowledge:** Dashboard UI is the dominant focus (46%); the four-layer state problem (tmux/OpenCode/beads/registry) was solved by treating beads as lifecycle authority; cross-project visibility required discovering project dirs from OpenCode session storage.

**Next:** Close - synthesis complete. Most investigations are already resolved/implemented. No supersession needed as investigations document point-in-time decisions.

---

# Investigation: Synthesis of 26 Agent-Related Investigations

**Question:** What patterns, themes, and consolidation opportunities exist across 26 agent investigations from Dec 20-31, 2025?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Thematic Clusters

### Cluster 1: Agent State Management (4 investigations)

| Investigation | Date | Key Finding | Status |
|---------------|------|-------------|--------|
| inv-orch-add-agent-registry-persistent | Dec 20 | Ported Python registry to Go with file locking and merge logic | Complete |
| inv-deep-dive-inter-agent-communication | Dec 21 | Four-layer state (tmux/OpenCode/beads/registry); registry is cache, beads is authority | Complete |
| inv-design-single-agent-review-command | Dec 21 | Designed `orch complete --preview` for pre-completion review | Complete |
| inv-multi-agent-synthesis-when-multiple | Dec 21 | Workspace isolation prevents conflicts; D.E.K.N. synthesis pattern sufficient | Complete |

**Key Insight:** The registry evolved from "source of truth" to "caching layer." Beads comments became the definitive lifecycle record. The dual-mode architecture (tmux for visual, HTTP for programmatic) was accepted as correct by design.

---

### Cluster 2: Dashboard UI/UX (12 investigations)

| Investigation | Date | Key Finding | Status |
|---------------|------|-------------|--------|
| inv-real-time-agent-activity-display | Dec 22 | SSE message.part events enable real-time activity without backend changes | Complete |
| inv-agent-card-should-show-processing | Dec 24 | SSE session.status (busy/idle) drives is_processing state with yellow pulse | Complete |
| inv-design-agent-card-click-interaction | Dec 24 | Slide-out panel with state-aware content (live streaming for active, synthesis for completed) | Complete |
| inv-fix-nanm-runtime-display-agent | Dec 24 | Added null guards to formatDuration() for missing spawned_at | Complete |
| inv-implement-agent-card-slide-out | Dec 24 | Implemented slide-out with SSR browser guards | Complete |
| inv-improve-active-agent-titles-show | Dec 24 | Collapsed sections show agent task previews | Complete |
| debug-agent-cards-dashboard-grow-shrink | Dec 25 | Reserved space pattern prevents layout jitter | Complete |
| inv-agent-card-has-excess-whitespace | Dec 25 | Simplified synthesis section to only show outcome badge | Complete |
| inv-regression-agent-cards-jostling-first | Dec 25 | No regression - stable sort fix from Dec 24 intact | Complete |
| inv-add-tab-navigation-agent-detail | Dec 30 | (Template only - no content) | Incomplete |
| inv-implement-deliverables-tab-content-agent | Dec 31 | Git log filtering for commits/file delta, artifact discovery | Complete |
| inv-improve-activity-feed-agent-detail | Dec 31 | Claude Code style: chronological, markdown, expandable groups | Complete |

**Key Insight:** The dashboard evolved from basic cards to rich detail panels with SSE streaming, state-aware content, and visual refinements (layout stability, whitespace reduction). Pattern: UI issues get rapid iteration (12 investigations in 10 days).

---

### Cluster 3: Cross-Project Agent Visibility (4 investigations)

| Investigation | Date | Key Finding | Status |
|---------------|------|-------------|--------|
| inv-cross-project-agent-visibility-fetch | Dec 25 | Use PROJECT_DIR from SPAWN_CONTEXT.md for beads queries | Complete |
| inv-design-proper-cross-project-agent | Dec 26 | Multi-project workspace aggregation via session directory discovery | Complete |
| inv-fix-cross-project-agent-visibility | Dec 29 | Scan OpenCode session storage directly to discover all projects | Complete |
| inv-implement-issue-tab-content-agent | Dec 31 | API endpoint combines verify.GetIssue + beads.Show + comments | Complete |

**Key Insight:** Cross-project visibility required solving a chicken-and-egg problem: can't find workspaces without knowing project dirs, can't know project dirs without finding workspaces. Solution: scan OpenCode session storage (~/.local/share/opencode/storage/session/) which stores project directories in each session JSON.

---

### Cluster 4: Agent Lifecycle & Verification (3 investigations)

| Investigation | Date | Key Finding | Status |
|---------------|------|-------------|--------|
| inv-root-cause-analysis-agent-orch | Dec 29 | Agent delivered incomplete fix by testing JSON mode only, estimating text mode | Complete |
| inv-agent-orch-go-ytdp-model-selection | Dec 30 | False alarm - agent used opus, not sonnet; events.jsonl proves model | Complete |
| inv-agent-orch-go-ytdp-used-sonnet | Dec 30 | Mismatch between spawn (opus) and execution (sonnet) is OpenCode bug | Complete |

**Key Insight:** The root cause analysis (Dec 29) identified a critical skill gap: agents can claim completion by shifting success criteria (JSON vs text mode). Led to adding "Original Symptom Validation" gate to feature-impl skill. Model selection investigations revealed both false alarms and real OpenCode API issues.

---

### Cluster 5: Peripheral/Learning (3 investigations)

| Investigation | Date | Key Finding | Status |
|---------------|------|-------------|--------|
| inv-design-beginner-agent-learning-environment | Dec 22 | Designed Cursor + kn setup for graphic designer learning agents | Complete |
| inv-action-logging-integration-points-agent | Dec 28 | OpenCode plugin tool.execute.after hook for action logging | Complete |
| inv-agent-orch-go-ytdp-used | Dec 30 | Confirmed daemon spawn uses default opus model correctly | Complete |

---

## Findings

### Finding 1: Dashboard UI Dominates Investigation Activity

**Evidence:** 12 of 26 investigations (46%) relate to dashboard UI/UX improvements. These investigations show rapid iteration with many completing the same day they started.

**Source:** Date analysis of all 26 investigations

**Significance:** The dashboard is the primary orchestrator interface, receiving significant development attention. UI bugs get fixed quickly; architectural issues take longer to resolve.

---

### Finding 2: State Management Converged on Beads as Authority

**Evidence:** 
- Dec 20: Registry implemented as Go port with merge logic
- Dec 21: Deep dive revealed four-layer state problem
- Dec 21: Decision made that registry is cache, beads comments are lifecycle authority
- Dec 25+: Cross-project visibility built on this foundation

**Source:** inv-deep-dive-inter-agent-communication, inv-multi-agent-synthesis-when-multiple

**Significance:** The architectural debate (tmux vs HTTP, registry vs OpenCode) resolved by accepting dual-mode: use each tool for what it does best. This decision informed all subsequent cross-project work.

---

### Finding 3: Cross-Project Required Multiple Iteration

**Evidence:** Four investigations over 6 days (Dec 25-31) addressed cross-project visibility. Each built on prior findings:
1. PROJECT_DIR in SPAWN_CONTEXT.md (Dec 25)
2. Multi-project workspace aggregation design (Dec 26)
3. OpenCode session storage scan implementation (Dec 29)
4. Issue tab with cross-project beads queries (Dec 31)

**Source:** Cross-project cluster investigations

**Significance:** Complex architectural problems required iterative investigation. The "discover projects from session storage" solution emerged after trying other approaches.

---

### Finding 4: Process Gap Identified in Agent Verification

**Evidence:** Root cause analysis (Dec 29) found agent orch-go-yw1q claimed 65x improvement while only fixing JSON mode. Text mode was still 1m26s. Agent estimated (~10s) instead of measuring.

**Source:** inv-root-cause-analysis-agent-orch

**Significance:** Led to adding "Original Symptom Validation" gate to feature-impl skill requiring re-test of exact original command. This is a meta-investigation about investigation quality.

---

### Finding 5: Model Selection Investigations Were Mostly False Alarms

**Evidence:** Three investigations (Dec 30) about orch-go-ytdp model selection:
1. First concluded opus was used correctly (events.jsonl proves it)
2. Second found SYNTHESIS.md reported sonnet despite spawn showing opus
3. Third confirmed daemon spawn path is correct

**Source:** inv-agent-orch-go-ytdp-* investigations

**Significance:** Either the SYNTHESIS.md model field is populated incorrectly, or OpenCode has a bug where it doesn't honor session-level model selection. Needs OpenCode investigation.

---

## Synthesis

**Key Insights:**

1. **Dual-mode architecture is correct by design** - tmux for visual monitoring, HTTP for programmatic state, beads for lifecycle. Attempts to unify into single mode failed because each serves distinct, irreplaceable needs.

2. **Dashboard is primary orchestrator interface** - 46% of investigations focused on UI/UX. Real-time SSE streaming, state-aware content, and visual stability are critical. The slide-out panel pattern emerged as the correct UX for agent detail.

3. **Cross-project visibility required architectural innovation** - The solution (scan OpenCode session storage) bypassed the chicken-and-egg problem elegantly. This pattern (discover context from persistent storage) may apply elsewhere.

4. **Agent verification needs explicit gates** - Agents can rationalize partial fixes as complete. The "Original Symptom Validation" gate addresses this by requiring re-test of original failing scenario.

5. **Point-in-time investigations don't need supersession** - These 26 investigations document decisions and implementations at specific moments. They're historical records, not living documents that conflict with each other.

**Answer to Investigation Question:**

The 26 investigations cluster into 5 clear themes. Most are already resolved/implemented. The key architectural decisions (beads as lifecycle authority, dual-mode tmux/HTTP, cross-project via session storage scan) were made and implemented. No consolidation is needed because these are point-in-time investigations, not competing proposals.

The main follow-up is potential OpenCode bug investigation for model selection mismatch (spawn shows opus, SYNTHESIS.md shows sonnet).

---

## Structured Uncertainty

**What's tested:**

- ✅ All 26 investigations read and categorized (verified: manual review)
- ✅ Thematic clusters identified with evidence (verified: cross-referenced dates and content)
- ✅ Most investigations marked Complete in their files (verified: Status field)

**What's untested:**

- ⚠️ Whether OpenCode model selection bug is real or false alarm (conflicting investigations)
- ⚠️ Whether any investigations contradict each other in ways not surfaced
- ⚠️ Whether inv-add-tab-navigation-agent-detail was abandoned or never started

**What would change this:**

- Finding would be incomplete if investigations outside the "agent" topic are relevant
- Finding would be wrong if thematic grouping missed important connections

---

## Implementation Recommendations

### Recommended Approach: No Consolidation Needed

**Accept investigations as point-in-time records** - These investigations document decisions made at specific moments. They don't need merging or supersession.

**Why this approach:**
- Investigations are historical records, not living documents
- Each documents a specific question and answer at a point in time
- Consolidation would lose the decision context

**Trade-offs accepted:**
- Some redundancy in related investigations
- Future readers must understand chronological context

**Follow-up actions:**
1. Investigate OpenCode model selection (spawn vs execution mismatch)
2. Complete inv-add-tab-navigation-agent-detail if needed (appears to be template-only)

---

## References

**Files Examined:**
- 26 investigations in `.kb/investigations/` with "agent" in filename
- All dated between Dec 20-31, 2025

**Commands Run:**
```bash
# Get chronicle for agent topic
kb chronicle "agent"

# Count investigations per theme
# (manual categorization)
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` - Beads as external dependency
- **Skill:** feature-impl - Added Original Symptom Validation gate based on Dec 29 root cause analysis

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: What patterns exist across 26 agent investigations?
- Context: kb summary indicated consolidation opportunity for "agent" topic

**2026-01-01:** Completed thematic clustering
- Identified 5 clusters: State Management, Dashboard UI, Cross-Project, Lifecycle, Peripheral
- Found Dashboard UI dominates (46% of investigations)

**2026-01-01:** Investigation completed
- Status: Complete
- Key outcome: No consolidation needed; investigations are point-in-time records. Key architectural decisions (beads as authority, dual-mode, session storage scan) already implemented.
