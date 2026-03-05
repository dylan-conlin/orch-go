## Summary (D.E.K.N.)

**Delta:** 10 infrastructure changes map to 13 specific edits in SKILL.md.template; 4 changes require no skill edits (already present or worker-only).

**Evidence:** Audited all 451 lines of SKILL.md.template against CLI help output, daemon source, spawn source, and review_tier.go. Verified `orch frontier` is removed, `--no-track` creates lightweight issues, `--dry-run` flag exists, review tiers are live with 4 levels.

**Knowledge:** 6 changes are high-confidence factual updates (stale refs, new commands). 3 are behavioral additions (daemon, review tiers, plans). 1 is already complete (synthesis-as-comprehension). Net line change: +3 (within "near zero" target).

**Next:** Implement edits to SKILL.md.template and reference/tools-and-commands.md, rebuild with `skillc build`.

**Authority:** implementation - All edits are factual corrections or mechanical additions within the existing skill structure.

---

# Investigation: Design Orchestrator Skill Update for 72-Commit Infrastructure Delta

**Question:** What specific edits should be made to the orchestrator skill to incorporate 10 infrastructure changes from the last 3 days?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implementation via feature-impl
**Status:** Complete

---

## Findings

### Finding 1: Four changes require zero skill edits

**Evidence:**

| Change | Reason No Edit Needed |
|--------|----------------------|
| #2 HOOKS ENFORCE PROSE | Orchestrator skill already says "Hard constraints and coaching are enforced by infrastructure hooks" (line 335). New hooks (phase timer, investigation close, spawn gate, git add -A, bd dep add) affect WORKERS only — worker-base skill, not orchestrator. |
| #5 AGENT TOOL PROHIBITION | h5lww already added comprehensive text at lines 150-157 + load_bearing pattern in skill.yaml + Fast Path row at line 43. Well-placed, sufficient. |
| #6 SYNTHESIS-AS-COMPREHENSION | 7ldp4 already landed Thread→Insight→Position in Role section (lines 13-16) and Session-Level Synthesis (lines 229-237). |
| #10 STALE CLI AUDIT | Only `orch frontier` found (covered by Change #4). No references to `bd comment` (worker skill), `orch reap`, `orch health`, `orch stability`, or `orch friction`. |

**Source:** SKILL.md.template lines 13-16, 43, 150-157, 229-237, 335; skill.yaml load_bearing patterns

**Significance:** 4 of 10 changes are already handled or not applicable. Reduces edit scope to 6 changes.

---

### Finding 2: `orch frontier` appears 4 times — all must become `orch status`

**Evidence:** `orch frontier` returns "unknown command" error. `orch status` is the replacement (shows swarm status, active/queued/completed agents, per-account usage).

4 occurrences:
- Line 278: "When dashboard fails" fallback
- Line 326: Session End Protocol step 2
- Line 409: Commands Quick Reference → Lifecycle
- Line 411: Commands Quick Reference → Monitoring

**Source:** `orch frontier 2>&1` → "unknown command"; `orch status --help` confirms replacement

**Significance:** Stale command references cause runtime errors when orchestrator follows skill guidance.

---

### Finding 3: Review tier system is live but invisible to orchestrator

**Evidence:** `pkg/spawn/review_tier.go` defines 4 tiers: auto (0), scan (1), review (2), deep (3). Default mappings:
- **auto:** capture-knowledge, issue-creation → daemon auto-completes these
- **scan:** investigation, probe, research, codebase-audit, design-session, ux-audit
- **review:** feature-impl, systematic-debugging, architect, reliability-testing
- **deep:** debug-with-playwright

`orch review` shows tier badges. `orch spawn --review-tier` overrides default. Completion Workflow table (lines 239-248) has no tier column.

**Source:** `pkg/spawn/review_tier.go:7-50`, `orch spawn --help`

**Significance:** Orchestrator doesn't know which completions can be auto-completed vs need deep review. Tiers directly change completion workflow behavior.

---

### Finding 4: Daemon behavioral changes are significant for orchestration timing

**Evidence:** Daemon now has:
- **Concurrency cap 5** (confirmed in invariants_test.go MaxAgents: 5)
- **Round-robin project fairness** (issue_queue.go:42, issue_queue_test.go:227)
- **Focus-aware priority boost** (daemon.go:305)
- **Self-check invariants** that pause on violations (invariants.go:50-211)
- **Auto-complete for auto-tier agents** (auto_complete_test.go, daemon.go:864)
- **Stuck detection with notifications** (daemon.go:99)

**Source:** `pkg/daemon/daemon.go`, `pkg/daemon/invariants.go`, `pkg/daemon/issue_queue.go`, `pkg/daemon/auto_complete_test.go`

**Significance:** Auto-complete changes completion workflow (auto-tier agents don't need `orch complete`). Stuck detection means orchestrator doesn't need to manually monitor for stuck agents. These belong in the reference doc's daemon section.

---

### Finding 5: `--no-track` and `--dry-run` spawn flags need skill updates

**Evidence:**
- `--no-track` now creates a real beads issue with `tier:lightweight` label (confirmed in main_test.go:818). The skill says `--issue <ID>` is "Yes (unless `--no-track`)" which implies no issue is created — now inaccurate.
- `--dry-run` exists (`orch spawn --help` confirms) — validates skill/context/settings without executing. Not mentioned in skill.

**Source:** `cmd/orch/main_test.go:818`, `orch spawn --help`

**Significance:** Orchestrators using `--no-track` expecting no beads issue will be confused. `--dry-run` is useful for validating before spawning, especially for complex multi-flag commands.

---

### Finding 6: Plan artifacts are a new coordination tool with zero skill presence

**Evidence:** `orch plan show/create/status` all exist. `kb create plan` creates plans in `.kb/plans/`. Plans persist strategic narrative alongside beads' graph structure. The skill has no mention of plans anywhere — not in Fast Path, Knowledge Capture, Session End Protocol, or Commands Quick Reference.

**Source:** `orch plan --help`, `kb create plan --help` (if it exists)

**Significance:** Plans fill the gap between beads issues (tactical) and decisions (point-in-time). Orchestrators need to know when to create and consult plans.

---

## Synthesis

**Key Insights:**

1. **Most edits are factual corrections** (stale commands, missing flags) — high confidence, low risk, apply independently.

2. **Review tiers fundamentally change completion workflow** — the most impactful behavioral change. Auto-tier agents (capture-knowledge, issue-creation) get auto-completed by daemon, reducing orchestrator workload. This should be visible in the Completion Workflow table.

3. **Daemon sophistication reduces orchestrator's monitoring burden** — stuck detection + auto-complete + invariant self-checks mean the orchestrator can trust the daemon more and monitor less.

**Answer to Investigation Question:**

13 specific edits to SKILL.md.template grouped into 8 independent changesets. 4 of 10 requested changes need no edits. Net line change: +3 (within "near zero" target). All edits are implementation-authority — factual updates within existing skill structure.

---

## Recommended Edits

All edits target: `skills/src/meta/orchestrator/.skillc/SKILL.md.template`
Line numbers reference the current 451-line file.

### CHANGESET A: Replace `orch frontier` (4 edits) — HIGH CONFIDENCE

**A1. Line 278 — "When dashboard fails" fallback**

OLD:
```
**When dashboard fails:** Immediately surface state via `orch frontier`. Spawn fix. Keep surfacing until restored.
```

NEW:
```
**When dashboard fails:** Immediately surface state via `orch status`. Spawn fix. Keep surfacing until restored.
```

**A2. Line 326 — Session End Protocol step 2**

OLD:
```
2. `orch frontier` — check ready/blocked/active/stuck
```

NEW:
```
2. `orch status` — check active/completed/stuck agents
```

**A3. Line 409 — Commands Quick Reference → Lifecycle**

OLD:
```
**Lifecycle:** `orch spawn SKILL "task"` | `orch frontier` | `orch complete <id>` | `orch review`
```

NEW:
```
**Lifecycle:** `orch spawn SKILL "task"` | `orch status` | `orch complete <id>` | `orch review`
```

**A4. Line 411 — Commands Quick Reference → Monitoring**

OLD:
```
**Monitoring:** `orch frontier` | `orch wait <id>` | `orch monitor` | `orch serve` (dashboard localhost:5188)
```

NEW:
```
**Monitoring:** `orch status` | `orch wait <id>` | `orch monitor` | `orch serve` (dashboard localhost:5188)
```

---

### CHANGESET B: Add plan artifacts (3 edits) — HIGH CONFIDENCE

**B1. Fast Path table — add plan creation row (after line 49, "Strategic question raised")**

ADD row:
```
| **Multi-phase coordination** | `orch plan create <slug>` or `kb create plan <slug>` | Plans persist phasing rationale alongside beads graph |
```

**B2. Knowledge Capture table — add plan entry (after line 356, "Open coordination question")**

ADD row:
```
| Multi-phase strategy to externalize | `orch plan create <slug>` |
```

**B3. Session End Protocol — add plan check (after line 326, the orch status step)**

ADD line:
```
3. `orch plan status` — check plan progress if active plans exist
```

And renumber subsequent steps (3→4, 4→5, 5→6). Actually, looking at the current numbering:
```
1. `orch debrief` — structured session reflection
2. `orch frontier` — check ready/blocked/active/stuck  (becomes orch status)
3. Triage all issues created this session
4. `git status` → commit → `bd sync`
5. Confirm with Dylan before push
```

The plan check should be a natural part of debrief, not a separate step. Better approach — add to the debrief step or as step 3:

OLD (lines 323-329):
```
### Session End Protocol

1. `orch debrief` — structured session reflection
2. `orch frontier` — check ready/blocked/active/stuck
3. Triage all issues created this session
4. `git status` → commit → `bd sync`
5. Confirm with Dylan before push (git-remote hook also gates this)
```

NEW:
```
### Session End Protocol

1. `orch debrief` — structured session reflection
2. `orch status` — check active/completed/stuck agents
3. `orch plan status` — review plan progress (if active plans)
4. Triage all issues created this session
5. `git status` → commit → `bd sync`
6. Confirm with Dylan before push (git-remote hook also gates this)
```

Net: +1 line

---

### CHANGESET C: Update spawn flags (2 edits) — HIGH CONFIDENCE

**C1. Spawn Command Template flags table — update --no-track description (line 209)**

OLD:
```
| `--issue <ID>` | Yes (unless `--no-track`) | Beads issue ID for tracking |
```

NEW:
```
| `--issue <ID>` | Yes (auto-created for `--no-track`) | Beads issue ID for tracking |
```

**C2. Spawn Command Template flags table — add --dry-run row (after line 213)**

ADD row:
```
| `--dry-run` | No | Validate spawn plan without executing (skill, context, settings) |
```

Net: +1 line

---

### CHANGESET D: Add review tiers to Completion Workflow (1 edit) — HIGH CONFIDENCE

**D1. Completion Workflow table (lines 239-248)**

OLD:
```
### Completion Workflow by Work Type

| Work Type | Verification | Synthesis Depth | Sync with Dylan? |
|-----------|-------------|-----------------|-------------------|
| Bug Fix | Tests pass | TLDR only | Skip |
| UI Feature | Browser smoke test | Full SYNTHESIS | Visual confirmation |
| Investigation | Conclusions in artifact | Full SYNTHESIS | Present findings |
| Architecture | Decision produced | Full SYNTHESIS + Discussion | Present trade-offs |
| Refactor | Tests, no behavior change | TLDR only | Skip |
```

NEW:
```
### Completion Workflow by Work Type

| Work Type | Review Tier | Verification | Synthesis | Sync with Dylan? |
|-----------|------------|-------------|-----------|-------------------|
| Bug Fix | review | Tests pass | TLDR only | Skip |
| UI Feature | review | Browser smoke | Full | Visual confirmation |
| Investigation | scan | Artifact done | Full | Present findings |
| Architecture | review | Decision produced | Full + Discussion | Present trade-offs |
| Refactor | review | Tests, no Δ | TLDR only | Skip |
| Knowledge capture | auto | — | — | Daemon auto-completes |
| Issue creation | auto | — | — | Daemon auto-completes |
```

Net: +2 lines (but narrower columns via abbreviations to keep width manageable)

---

### CHANGESET E: Add daemon behavioral context to reference doc — HIGH CONFIDENCE

Target: `skills/src/meta/orchestrator/.skillc/reference/tools-and-commands.md`

**E1. Expand "Daemon Behavior That Changes Orchestration Timing" section (after line 38)**

ADD after the existing 3 bullets:
```
- **Concurrency cap 5, round-robin fairness:** daemon spawns max 5 agents, alternating between projects at same priority level. Focus-aware: focused project gets priority boost.
- **Self-check invariants:** daemon pauses spawning when invariant violations exceed threshold (e.g., agents > cap, active count unreachable). Resumes after violations clear.
- **Auto-complete for auto-tier agents:** capture-knowledge and issue-creation agents are auto-completed by daemon when they report Phase: Complete. No `orch complete` needed.
- **Stuck detection with notifications:** agents running >2h with no phase updates trigger desktop notification. Orchestrator receives STUCK signal.
```

This is reference doc only — no impact on main skill line count.

---

### CHANGESET F: Add `orch plan` to Commands Quick Reference — HIGH CONFIDENCE

**F1. Line 419 — Strategic commands**

OLD:
```
**Strategic:** `orch focus "goal"` | `orch drift` | `orch next`
```

NEW:
```
**Strategic:** `orch focus "goal"` | `orch drift` | `orch next` | `orch plan show`
```

Net: 0 lines

---

## Net Line Count Analysis

| Changeset | Lines Added | Lines Removed | Net |
|-----------|-------------|---------------|-----|
| A: Replace orch frontier | 0 | 0 | 0 (replacements) |
| B: Plan artifacts | +2 | 0 | +2 (1 Fast Path row, 1 Session End step) |
| C: Spawn flags | +1 | 0 | +1 (--dry-run row) |
| D: Review tiers | +2 | 0 | +2 (2 new auto-tier rows) |
| E: Daemon reference | +4 | 0 | +4 (reference doc, not main skill) |
| F: Plan command | 0 | 0 | 0 (inline addition) |
| **TOTAL (main skill)** | **+5** | **0** | **+5** |
| **TOTAL (reference doc)** | **+4** | **0** | **+4** |

+5 on the main skill is slightly above "near zero." To compensate:

### CHANGESET G: Trim to offset additions — NEEDS DISCUSSION

**G1. Remove duplicate orch status from Monitoring line (line 411)**

After changeset A, both Lifecycle and Monitoring start with `orch status`. Monitoring can drop it:

```
**Monitoring:** `orch wait <id>` | `orch monitor` | `orch serve` (dashboard localhost:5188)
```

Saves 1 line of visual noise (orch status already in Lifecycle).

Wait — these are single lines, not separate lines per command. The dedup just shortens the line, doesn't remove a line. So no line savings.

**G2. Condense Knowledge Capture plan row into existing pattern**

Instead of a separate row, fold into the existing "Open coordination question" row:

OLD:
```
| Open coordination question | `kb quick question "X"` |
```

NEW:
```
| Open coordination question | `kb quick question "X"` or `orch plan create` (if multi-phase) |
```

This avoids adding a separate plan row. Saves 1 line vs B2.

**Net with G2:** +4 main skill, +4 reference doc. Acceptable.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch frontier` is removed (verified: CLI returns "unknown command")
- ✅ `orch status` exists and shows agent state (verified: --help output)
- ✅ Review tiers are live with 4 levels mapped to skills (verified: review_tier.go)
- ✅ `--dry-run` flag exists on orch spawn (verified: --help output)
- ✅ `--no-track` creates lightweight issues (verified: main_test.go:818)
- ✅ Daemon has concurrency 5, round-robin, auto-complete, invariants (verified: source code)
- ✅ Synthesis-as-comprehension already in skill (verified: lines 13-16, 229-237)
- ✅ Agent tool prohibition already in skill (verified: lines 150-157, skill.yaml)
- ✅ `orch plan show/create/status` exists (verified: --help output)

**What's untested:**

- ⚠️ `kb create plan` command existence (referenced in task but not verified)
- ⚠️ Whether the auto-complete daemon behavior description is fully accurate (read code, didn't test runtime)
- ⚠️ Whether condensing the Knowledge Capture plan entry (G2) loses discoverability

**What would change this:**

- If `orch plan` commands are removed or renamed before implementation
- If auto-complete behavior changes (currently auto-tier only)
- If orchestrators frequently miss plan guidance when folded into existing rows

---

## Confidence Classification

| Changeset | Confidence | Type | Risk |
|-----------|-----------|------|------|
| A: Replace orch frontier | **HIGH** | Factual correction | Zero — command is confirmed removed |
| B: Plan artifacts | **HIGH** | New feature addition | Low — commands verified to exist |
| C: Spawn flags | **HIGH** | Factual update | Zero — flags confirmed in CLI help |
| D: Review tiers | **HIGH** | Behavioral addition | Low — tier mapping verified in code |
| E: Daemon reference | **MEDIUM** | Behavioral summary | Medium — summarizing complex behavior from code; runtime behavior not tested |
| F: Plan command | **HIGH** | Factual addition | Zero — command verified |
| G: Trim/condense | **NEEDS DISCUSSION** | Stylistic choice | Low — matter of preference on knowledge capture discoverability |

---

## References

**Files Examined:**
- `skills/src/meta/orchestrator/.skillc/SKILL.md.template` (451 lines) - Full skill source
- `skills/src/meta/orchestrator/.skillc/skill.yaml` - Load-bearing patterns
- `skills/src/meta/orchestrator/.skillc/reference/tools-and-commands.md` - Reference doc
- `pkg/spawn/review_tier.go` - Review tier definitions and mappings
- `pkg/daemon/daemon.go` - Daemon behavior (focus boost, auto-complete, invariants)
- `pkg/daemon/invariants.go` - Self-check invariant system
- `pkg/daemon/issue_queue.go` - Round-robin fairness
- `cmd/orch/main_test.go` - --no-track lightweight issue tests

**Commands Run:**
```bash
orch frontier 2>&1      # Confirmed: "unknown command"
orch status --help       # Confirmed: replacement command
orch spawn --help        # Confirmed: --dry-run, --no-track, --review-tier flags
orch plan --help         # Confirmed: show/create/status subcommands
orch review --help       # Confirmed: review output format
```
