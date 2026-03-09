## Summary (D.E.K.N.)

**Delta:** Governance health should be a single 0-100 score composed of 5 weighted signals that measure whether the harness is actually constraining agent behavior, not just whether the codebase is healthy.

**Evidence:** Analyzed existing entropy/health systems (pkg/entropy 619 lines, pkg/health 299 lines, 7 entropy signals, 6 health metrics). Current `orch entropy` measures codebase entropy (fix:feat ratio, bloat, velocity) but not harness effectiveness. The gap: a project can have healthy entropy metrics while its harness is completely missing.

**Knowledge:** The key insight is the distinction between "is the codebase healthy?" (what entropy currently measures) and "is the harness working?" (what governance health should measure). A harness that's never tested might as well not exist. The score must capture: (1) structural presence of enforcement, (2) operational evidence that gates fire, (3) trend direction, (4) absence of bypass accumulation, (5) control plane integrity.

**Next:** Implement `GovernanceScore` in pkg/entropy (or new pkg/governance), add `--governance` flag to `orch entropy`, create initial test suite. Route to architect for placement decision (pkg/entropy extension vs new package).

**Authority:** architectural - Cross-component design affecting entropy, health, harness, and control packages; involves new signal aggregation that reaches across multiple existing systems.

---

# Investigation: Design Governance Health Metric

**Question:** What should `orch entropy` measure as a single indicator of harness effectiveness, and how should the signals be weighted?

**Started:** 2026-03-08
**Updated:** 2026-03-08
**Owner:** Agent (orch-go-ycdbr)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** `.kb/plans/2026-03-08-harness-publication.md` Phase 1

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/harness-engineering/model.md` | extends | yes | - |
| `.kb/models/entropy-spiral/model.md` | extends | yes | - |
| `.kb/investigations/2026-03-05-inv-design-dual-authority-detection-scan.md` | extends | pending | - |
| `.kb/plans/2026-03-08-harness-publication.md` | implements | yes | - |

---

## Findings

### Finding 1: Current entropy measures codebase health, not harness health

**Evidence:** `pkg/entropy/entropy.go` aggregates 7 signals:
- Fix:Feat commit ratio (lines 141-157)
- Commit velocity (lines 159-169)
- Bloated file count (lines 188-230)
- Duplicate pair count (lines 552-585)
- Architecture lint results (lines 588-612)
- Override/bypass trend (lines 295-350)
- Agent event stats (lines 248-292)

The health level determination (`healthLevel()` at line 176) is based solely on fix:feat ratio. A project with zero harness (no deny rules, no hooks, no control plane lock, no pre-commit gates) would show "healthy" if it simply had more feat: than fix: commits.

**Source:** `pkg/entropy/entropy.go:29-42` (Report struct), `pkg/entropy/entropy.go:176-185` (healthLevel function)

**Significance:** The current entropy command answers "is the codebase degrading?" not "is the harness working?" These are fundamentally different questions. A codebase can be degrading despite having a harness (gates miscalibrated), or healthy despite having no harness (early project, low velocity). Governance health needs to measure the harness itself.

---

### Finding 2: Existing infrastructure provides 5 queryable harness signals

**Evidence:** The orch codebase already has infrastructure to query each component of harness health:

1. **Control plane integrity** — `pkg/control/` can check lock state of all files via `VerifyLocked()`, deny rule presence via `DenyRules()`, and file existence. Currently exposed via `orch harness status` and `orch control deny`. Current state: 12/12 locked, but 0/6 deny rules present.

2. **Gate activity** — `pkg/events/logger.go` tracks 27 event types including `verification.bypassed`, `spawn.hotspot_bypassed`, `verification.failed`, `verification.auto_skipped`. Currently aggregated by `pkg/entropy/` into EventStats (spawns, completions, abandonments, bypasses, reworks).

3. **Pre-commit enforcement** — `orch precommit accretion` runs via `.git/hooks/pre-commit`. Its presence/absence is queryable by checking the hook file. Its firing rate is not currently tracked.

4. **Structural test coverage** — `architecture_lint_test.go` has 4 tests. Pass/fail queryable via `entropy.RunArchLintTests()`.

5. **Override trend** — `pkg/entropy/entropy.go:295-350` already computes window-over-window bypass direction.

**Source:** `pkg/control/`, `pkg/events/logger.go`, `pkg/entropy/entropy.go`, `cmd/orch/harness_cmd.go`

**Significance:** No new data collection is needed. The governance health score can be computed entirely from existing infrastructure. The design challenge is weighting and composition, not instrumentation.

---

### Finding 3: The governance health score must distinguish "present" from "working"

**Evidence:** Current `orch harness status` shows LOCKED (12/12 files), but `orch control deny` shows 0/6 deny rules present. This means the control plane files are immutable... but the deny rules that prevent agents from editing control plane files were never added. The lock protects files that don't contain the protection rules.

Similarly, a pre-commit hook file can exist but contain only `exit 0`. Architecture lint tests can exist but not be in CI.

**Source:** Running `orch harness status` (LOCKED 12/12) and `orch control deny` (MISS 0/6) — the contradiction.

**Significance:** The score must measure at two levels: (1) Is the mechanism present? (structural) and (2) Is there evidence the mechanism fires? (operational). A gate that never fires is either perfectly calibrated (nothing to catch) or completely ineffective (not wired in). Operational evidence disambiguates.

---

### Finding 4: The harness publication plan requires a portable, explainable score

**Evidence:** From `.kb/plans/2026-03-08-harness-publication.md` Phase 1: "Exit criteria: `orch entropy` reports a single governance health score." The score must be:
- **Single number** — not a dashboard of 15 metrics
- **Portable** — computable in non-orch projects (Phase 4 goal)
- **Explainable** — each sub-signal visible for diagnosis
- **Directional** — trend-able over time (for 30-day trajectory measurement in orch-go-1ittt)

**Source:** `.kb/plans/2026-03-08-harness-publication.md:50-52`, `.kb/threads/2026-03-08-harness-engineering-as-strategic-position.md`

**Significance:** The score design must balance simplicity (single number) with diagnosability (breakdowns). The model is like a credit score: one number, but backed by component scores you can drill into.

---

### Finding 5: Five sub-scores map to the harness taxonomy

**Evidence:** The harness engineering model (`.kb/models/harness-engineering/model.md`) defines the harness taxonomy:

| Harness Layer | What It Measures | Queryable? |
|--------------|-----------------|------------|
| Control plane immutability | Are governance files locked + deny rules present? | Yes — `pkg/control/` |
| Hard gates (spawn, completion, pre-commit) | Do gates exist and fire? | Partial — events track bypasses, not all fires |
| Structural tests | Do architecture lint tests pass? | Yes — `entropy.RunArchLintTests()` |
| Soft harness coverage | Do skills/CLAUDE.md/kb exist? | Yes — file existence checks |
| Operational health | Are agents succeeding? Is bypass rate low? | Yes — `pkg/events/` |

These map naturally to 5 sub-scores that compose into a governance health score.

**Source:** `.kb/models/harness-engineering/model.md`, Sections 1, 6, and Critical Invariants

**Significance:** The 5-layer harness taxonomy provides the scoring framework. Each layer contributes a sub-score; the weighted composite is the governance health score.

---

## Synthesis

**Key Insights:**

1. **Governance health != codebase health** — Current `orch entropy` measures whether the codebase is degrading (fix:feat ratio, bloat, velocity). Governance health measures whether the harness that prevents degradation is in place and working. A project can score healthy on entropy with zero governance (luck/low velocity), or degrading on entropy with strong governance (the gates are catching problems, which means fixes are being generated).

2. **Present vs working is the core distinction** — A harness component can be structurally present (file exists, config set) without operationally working (never fires, no events logged). The score must capture both dimensions. The deny rules finding (0/6 present despite 12/12 lock) exemplifies this perfectly.

3. **The score must be time-series capable** — For the 30-day trajectory measurement (orch-go-1ittt), the score needs to be snapshot-able and trend-able. This aligns with the existing `pkg/health/` Store pattern (JSONL append, trend computation).

**Answer to Investigation Question:**

`orch entropy` should report a **Governance Health Score (0-100)** composed of 5 weighted sub-scores:

### Sub-Score 1: Control Plane Integrity (0-25, weight: 25%)

Measures: Are the governance files locked and protected?

| Check | Points | Source |
|-------|--------|--------|
| settings.json exists | 3 | File check |
| All control plane files locked (uchg) | 8 | `control.VerifyLocked()` |
| All 6 deny rules present | 8 | `control.DenyRules()` check |
| Pre-commit hook installed | 3 | `.git/hooks/pre-commit` exists + contains `orch` |
| Beads close hook installed | 3 | `.beads/hooks/on_close` exists |

**Why highest weight:** The harness engineering model's invariant #6: "Mutable hard harness is soft harness with extra steps." If the control plane isn't locked, everything else is theater. This is the foundation.

### Sub-Score 2: Gate Coverage (0-20, weight: 20%)

Measures: Are hard gates installed and configured?

| Check | Points | Source |
|-------|--------|--------|
| Spawn hotspot gate exists | 5 | Code check or config check |
| Completion verification enabled | 5 | `pkg/verify/` presence |
| Pre-commit accretion gate active | 5 | Hook content check |
| Architecture lint tests exist + pass | 5 | `entropy.RunArchLintTests()` |

**Why this weight:** Gates are the enforcement layer. Without gates, conventions are suggestions.

### Sub-Score 3: Operational Evidence (0-25, weight: 25%)

Measures: Is there evidence the harness is actively working?

| Check | Points | Source |
|-------|--------|--------|
| Events logging active (>0 events in 7 days) | 5 | `events.jsonl` check |
| Completion rate > 50% (completions/spawns) | 5 | Event stats |
| Abandonment rate < 30% | 5 | Event stats |
| Bypass rate < 10% of completions | 5 | Event stats |
| Override trend flat or down | 5 | `calculateOverrideTrend()` |

**Why tied for highest weight:** The harness engineering model's insight: "agent failure is harness failure." Operational evidence tells you whether the harness is actually producing good outcomes, not just installed.

### Sub-Score 4: Structural Hygiene (0-15, weight: 15%)

Measures: Is the codebase structured to resist accretion?

| Check | Points | Source |
|-------|--------|--------|
| CLAUDE.md exists with accretion boundaries | 3 | File check |
| .kb/ directory exists | 3 | File check |
| < 10 bloated files (>800 lines) | 5 | `countBloatedFiles()` — graded: 0 files = 5pts, 1-5 = 4pts, 6-10 = 3pts, 11-20 = 2pts, 21-50 = 1pt, >50 = 0pts |
| Fix:feat ratio < 0.5 | 4 | Commit classification — graded: <0.3 = 4pts, <0.5 = 3pts, <0.9 = 1pt, ≥0.9 = 0pts |

**Why lower weight:** This overlaps with existing entropy metrics. It's a hygiene check, not a governance check. Important but not the primary signal.

### Sub-Score 5: Knowledge Infrastructure (0-15, weight: 15%)

Measures: Does the governance learning loop exist?

| Check | Points | Source |
|-------|--------|--------|
| .kb/models/ directory with ≥1 model | 3 | File check |
| .kb/guides/ directory with ≥1 guide | 3 | File check |
| .kb/decisions/ directory with ≥1 decision | 3 | File check |
| Skills deployed (≥1 skill in ~/.claude/skills/) | 3 | File check |
| Event schema registered (events.jsonl has ≥3 event types) | 3 | Event diversity check |

**Why included:** The harness engineering model's failure mode #3: "Documentation without implementation." But the inverse is also true: implementation without documentation means the governance rationale is lost. Knowledge infrastructure ensures the system can learn from operations.

### Composite Score

```
GovernanceScore = ControlPlaneIntegrity + GateCoverage + OperationalEvidence + StructuralHygiene + KnowledgeInfra
```

Score is 0-100. Health bands:

| Score | Band | Interpretation |
|-------|------|----------------|
| 80-100 | Strong | Harness operational, gates active, control plane locked |
| 60-79 | Adequate | Most governance in place, some gaps |
| 40-59 | Weak | Significant governance gaps, harness partially effective |
| 20-39 | Minimal | Harness mostly absent, project at risk of entropy spiral |
| 0-19 | None | No governance infrastructure detected |

### Output Format

Human-readable (default):
```
Governance Health Score: 72/100 (Adequate)
  Control Plane Integrity:  22/25  (deny rules missing: 3pts lost)
  Gate Coverage:            18/20  (arch lint tests not in CI: 2pts lost)
  Operational Evidence:     20/25  (bypass rate elevated: 5pts lost)
  Structural Hygiene:       7/15   (52 bloated files: 8pts lost)
  Knowledge Infrastructure: 5/15   (no deployed skills detected: 10pts lost)
```

JSON (`--json`):
```json
{
  "governance_score": 72,
  "band": "adequate",
  "sub_scores": {
    "control_plane": {"score": 22, "max": 25, "details": [...]},
    "gate_coverage": {"score": 18, "max": 20, "details": [...]},
    "operational": {"score": 20, "max": 25, "details": [...]},
    "structural": {"score": 7, "max": 15, "details": [...]},
    "knowledge": {"score": 5, "max": 15, "details": [...]}
  }
}
```

---

## Structured Uncertainty

**What's tested:**

- ✅ Current `orch entropy` output produces fix:feat ratio 0.68:1 and "degrading" status (verified: ran `orch entropy --skip-dupdetect --skip-lint`)
- ✅ Control plane lock state queryable (verified: `orch harness status` returns 12/12 locked)
- ✅ Deny rule presence queryable (verified: `orch control deny` returns 0/6 MISS — demonstrates the present-vs-working gap)
- ✅ Events.jsonl provides bypass/abandonment/completion data (verified: `tail` on events.jsonl shows event types flowing)
- ✅ All 5 sub-scores computable from existing infrastructure (verified: traced each check to existing code or simple file checks)

**What's untested:**

- ⚠️ Whether the 25/20/25/15/15 weight distribution produces useful differentiation across projects (need cross-project testing in Phase 2)
- ⚠️ Whether the score meaningfully correlates with actual entropy spiral risk (need 30-day trajectory data from orch-go-1ittt)
- ⚠️ Whether sub-score thresholds (e.g., "bypass rate < 10%") are calibrated correctly (need operational data)
- ⚠️ Whether "operational evidence" sub-score is meaningful for new projects with < 7 days of history (may need grace period)
- ⚠️ Portability to non-Go, non-orch projects (Phase 2/4 concern)

**What would change this:**

- If 30-day data shows governance score doesn't correlate with accretion trajectory, the weight distribution is wrong
- If cross-project testing reveals most checks are orch-specific (e.g., beads hooks), need to identify the portable subset
- If a simpler 3-signal model (control plane + gates + operational) proves equally predictive, drop structural hygiene and knowledge infra

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Governance score design (this document) | architectural | Crosses entropy, health, control, and events packages; multiple valid approaches; needs orchestrator synthesis |
| Package placement (new pkg vs extend pkg/entropy) | architectural | Cross-component decision about package boundaries |
| Weight calibration from 30-day data | implementation | Tuning within established framework |
| Integration with `orch entropy` output | implementation | UI change within existing command |

### Recommended Approach: Extend pkg/entropy with GovernanceScore

**Why this approach:**
- pkg/entropy already aggregates cross-package signals (events, git, file system, arch lint)
- Avoids new package proliferation — governance score is conceptually "a new dimension of entropy analysis"
- The entropy Report struct already has a natural extension point

**Trade-offs accepted:**
- pkg/entropy grows by ~200-300 lines (still well under 800-line threshold at current 619)
- Conceptual overloading of "entropy" (codebase entropy vs governance entropy) — mitigated by clear naming (GovernanceScore vs HealthLevel)

**Implementation sequence:**
1. Add `GovernanceScore` struct to `pkg/entropy/` with 5 sub-score types
2. Implement `ComputeGovernanceScore()` that queries control, events, file system
3. Add `governance_score` field to `Report` struct
4. Update `FormatText()` to include governance score section
5. Add `--governance` flag to `orch entropy` for governance-only output
6. Add governance score to health JSONL snapshots for trending

### Alternative Approaches Considered

**Option B: New pkg/governance package**
- **Pros:** Clean separation, focused responsibility, clear ownership
- **Cons:** Creates a new package for ~200 lines of code; duplicates signal aggregation patterns already in pkg/entropy; adds another import to entropy_cmd.go
- **When to use instead:** If governance score grows beyond 500 lines or gains its own CLI command (`orch governance` instead of `orch entropy --governance`)

**Option C: Extend pkg/health package**
- **Pros:** Health package already has time-series, trends, alerts
- **Cons:** pkg/health currently tracks issue-level health (open/blocked/stale issues), not harness-level health; mixing concerns would make both harder to understand
- **When to use instead:** Never — health and governance are genuinely different domains

**Rationale for recommendation:** Option A (extend pkg/entropy) minimizes new code while leveraging existing aggregation patterns. The entropy command is already the "system health dashboard" — adding governance makes it complete.

---

### Implementation Details

**What to implement first:**
- GovernanceScore struct and ComputeGovernanceScore() — the core calculation
- Control plane integrity sub-score — most straightforward, uses existing pkg/control
- FormatText() update — make it visible immediately

**Things to watch out for:**
- ⚠️ The `control.VerifyLocked()` function expects specific settings.json path — make it configurable for portability
- ⚠️ Operational evidence sub-score requires events.jsonl — score should degrade gracefully (not error) when events file is absent
- ⚠️ File existence checks for .kb/, .beads/, etc. should use the project directory, not hardcoded paths
- ⚠️ Test files should NOT count toward bloated file metric — need to separate in scoring (currently they're 70% of the bloated files list)

**Areas needing further investigation:**
- Should the governance score distinguish test file bloat from source file bloat? (Current entropy doesn't)
- Should there be a "grace period" for new projects where operational evidence is N/A?
- How does this interact with the 30-day trajectory measurement (orch-go-1ittt)?

**Success criteria:**
- ✅ `orch entropy` shows governance score alongside existing metrics
- ✅ `orch entropy --json` includes governance_score object with sub-scores
- ✅ Score differentiates between a governed project (orch-go) and an ungoverned one
- ✅ Score correctly penalizes the current missing deny rules (should lose 8 points in control plane)
- ✅ Unit tests cover all 5 sub-score calculations

---

## References

**Files Examined:**
- `pkg/entropy/entropy.go` — Existing entropy analysis (619 lines, 7 signals)
- `pkg/entropy/save.go` — Report persistence
- `pkg/health/health.go` — Time-series health monitoring (299 lines)
- `pkg/events/logger.go` — Event types and logging (617 lines)
- `cmd/orch/entropy_cmd.go` — Entropy CLI command (141 lines)
- `cmd/orch/harness_cmd.go` — Harness lock/unlock/status/verify (198 lines)
- `cmd/orch/control_cmd.go` — Control plane deny rules
- `.kb/models/harness-engineering/model.md` — Harness taxonomy (5 layers, hard/soft)
- `.kb/models/entropy-spiral/model.md` — Spiral mechanism, 3 spirals, 1,625 lost commits
- `.kb/guides/minimum-viable-harness.md` — MVH checklist (3 tiers)
- `.kb/plans/2026-03-08-harness-publication.md` — Publication roadmap

**Commands Run:**
```bash
# Current entropy output
orch entropy --skip-dupdetect --skip-lint

# Current harness status
orch harness status
# Result: LOCKED (12/12 files)

# Current deny rule status
orch control deny
# Result: MISS (0/6 rules) — the present-vs-working gap

# Recent events
tail -5 ~/.orch/events.jsonl
```

**Related Artifacts:**
- **Plan:** `.kb/plans/2026-03-08-harness-publication.md` — This investigation is Phase 1 deliverable
- **Model:** `.kb/models/harness-engineering/model.md` — Provides the scoring framework
- **Model:** `.kb/models/entropy-spiral/model.md` — Provides the "why" for governance measurement
- **Thread:** `.kb/threads/2026-03-08-open-questions-harness-as-governance.md` — Lists open questions this partially answers

---

## Investigation History

**2026-03-08 20:45:** Investigation started
- Initial question: What should `orch entropy` measure as single indicator of harness effectiveness?
- Context: Phase 1 of harness publication plan (orch-go-ycdbr)

**2026-03-08 21:00:** Analyzed existing infrastructure
- Mapped all existing signals (entropy, health, events, control, harness)
- Discovered present-vs-working gap (12/12 locked, 0/6 deny rules)
- Confirmed no new instrumentation needed — all signals already queryable

**2026-03-08 21:20:** Designed 5-sub-score governance health metric
- Mapped harness taxonomy to scorable dimensions
- Defined point allocation and health bands
- Specified output format (human-readable + JSON)

**2026-03-08 21:30:** Investigation completed
- Status: Complete
- Key outcome: Governance health = 0-100 composite of 5 weighted sub-scores measuring harness presence AND effectiveness
