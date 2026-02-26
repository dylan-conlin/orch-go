<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Completion verification churn stems from 4 root causes: agent-scoping ambiguity (spawn time ≠ agent identity), evidence-vs-claim conflation, 12-gate proliferation with independent edge cases, and legacy workspace compatibility burden.

**Evidence:** Analyzed 26+ completion/verification investigations (Jan 8 + Jan 14 syntheses), read pkg/verify/check.go showing 12 distinct gates, identified 4 recurring bug patterns: concurrent agent pollution, missing spawn_time, cross-project scoping, evidence keyword vs file gaps.

**Knowledge:** Current architecture lacks a canonical agent identity; gates independently implement workspace/project resolution, causing inconsistent behavior. 55% of completions used --force bypass, indicating gate reliability issues.

**Next:** Recommend Agent Manifest pattern (spawn-time metadata snapshot), git-commit-based change detection, and evidence collection phase before gate verification. Update model at `.kb/models/completion-verification/model.md`.

**Promote to Decision:** recommend-yes - This identifies architectural patterns that should guide future verification work and prevent churn.

---

# Investigation: Synthesize 26 Completion Investigations - Architectural Analysis

**Question:** Why does completion verification generate so much churn/investigation, and what architectural improvements would reduce friction?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** og-arch-synthesize-26-investigations-17jan-f765
**Phase:** Complete
**Next Step:** None - recommendations ready for architectural decision
**Status:** Complete

---

## Investigations Analyzed

### Prior Synthesis Work

| Date | Investigation | Key Findings |
|------|---------------|--------------|
| Jan 8, 2026 | Synthesize Completion Investigations (10) | 4 evolution phases, verification architecture stabilized, guide created |
| Jan 14, 2026 | Synthesize Verification Investigations (25) | 4 verification layers identified, verification bottleneck principle, visual/declarative patterns |

### Recent Investigations (Jan 8-17, 2026)

| Date | Investigation | Root Cause Pattern |
|------|---------------|-------------------|
| Jan 14 | fix-git-diff-verification-false | Missing spawn_time → wrong git command |
| Jan 8 | bug-test-evidence-gate-triggers | Concurrent agents pollute spawn-time-based checks |
| Jan 14 | implement-targeted-skip-gate-flags | 55% of completions used --force bypass |
| Jan 8 | 25-28-agents-not-completing | Metrics artifact, true rate ~89% |
| Jan 15 | verify-test-first-gate (4 investigations) | Gate verification fragility |
| Jan 15 | support-cross-project-agent-completion | Wrong directory detection |
| Jan 14 | exempt-non-code-work-test | Gate triggers incorrectly for markdown-only |
| Jan 14 | detect-cross-repo-file-changes | File detection outside project |
| Jan 14 | add-verification-metrics-orch-stats | Tracking and metrics gaps |

---

## Findings

### Finding 1: Agent-Scoping Ambiguity is the Primary Churn Source

**Evidence:** Multiple investigations trace back to "what files did THIS agent change?" question:
- **Git diff false positive** (Jan 14): Missing `.spawn_time` causes `git diff HEAD` which only shows uncommitted changes, not agent's commits
- **Test evidence false positive** (Jan 8): `git log --since=<spawn_time>` includes concurrent agents' commits
- **Cross-project completion** (Jan 15): Workspace path != project dir, verification runs in wrong directory

All three bugs stem from imperfect proxies for agent identity:
- Spawn time: Fails for concurrent agents, missing files
- Workspace path: Fails for cross-project work
- Beads ID: Not linked to git commits

**Source:**
- `2026-01-14-inv-fix-git-diff-verification-false.md:37-48`
- `2026-01-08-inv-bug-test-evidence-gate-triggers.md:35-50`
- `2026-01-15-inv-support-cross-project-agent-completion.md`

**Significance:** Without a canonical agent identity, each gate independently implements agent-scoping logic, leading to inconsistent behavior and edge case bugs.

---

### Finding 2: Gate Proliferation Creates Multiplicative Complexity

**Evidence:** Current architecture has 12 distinct gates in `pkg/verify/check.go`:

| Gate | Purpose | Common Failure Mode |
|------|---------|---------------------|
| GatePhaseComplete | Agent reported completion | Agent forgets to report |
| GateSynthesis | SYNTHESIS.md exists | Light tier confusion |
| GateSessionHandoff | Orchestrator handoff | Placeholder content |
| GateHandoffContent | Handoff has actual content | Empty template |
| GateConstraint | Skill constraints met | Pattern matching edge cases |
| GatePhaseGate | Required phases reported | Missing phase comments |
| GateSkillOutput | Skill outputs exist | Wrong skill detection |
| GateVisualVerify | UI changes verified | False positives on web/ |
| GateTestEvidence | Tests were run | Concurrent agent pollution |
| GateGitDiff | Claims match git | Missing spawn_time |
| GateBuild | Project builds | Unrelated build failures |
| GateDecisionPatchLimit | Patch count limits | Threshold calibration |

**Key insight:** 55% of completions used `--force` to bypass gates (per Jan 14 investigation), indicating systematic reliability issues.

**Source:** `pkg/verify/check.go:12-26`

**Significance:** Each gate has its own edge cases and failure modes. 12 gates × N edge cases = multiplicative investigation churn.

---

### Finding 3: Evidence vs Claim Conflation

**Evidence:** Gates inconsistently check claims vs evidence:

| Gate | Checks Claims | Checks Evidence | Issue |
|------|---------------|-----------------|-------|
| Phase gate | ✅ ("Phase: Complete") | ❌ | Agent can lie |
| Evidence gate | ✅ ("screenshot" keyword) | ❌ | Keyword ≠ actual screenshot |
| Git diff gate | ❌ | ✅ (actual files) | Scoped incorrectly |
| Test evidence | ✅ ("test output" keyword) | ❌ | No actual test verification |
| Visual verification | ✅ ("APPROVED" keyword) | Partial (screenshot files) | Keyword false positives |

The screenshot file verification was added specifically because keyword matching produced false positives ("agent claims 'screenshot captured' without saving file").

**Source:**
- `2026-01-08-inv-feature-add-screenshot-file-verification.md`
- `2026-01-08-inv-verification-risk-based-visual-verification.md`

**Significance:** Claim-based gates are inherently unreliable. Evidence-based gates are more reliable but harder to implement and scope correctly.

---

### Finding 4: Legacy Workspace Compatibility Burden

**Evidence:** Multiple investigations reveal "skip if metadata missing" patterns:

```go
// From git_diff.go - zero spawn time handling
if spawnTime.IsZero() {
    result.Warnings = append(result.Warnings,
        "spawn time unavailable (workspace may predate spawn time tracking) - skipping git diff verification")
    return result
}
```

28 workspaces lack `.spawn_time` files. Legacy workspaces also lack:
- `.tier` files (fall back to "full")
- SPAWN_CONTEXT.md with PROJECT_DIR
- Consistent skill metadata

**Source:** `2026-01-14-inv-fix-git-diff-verification-false.md:52-59`

**Significance:** Legacy compatibility creates unpredictable verification behavior. Gates behave differently for old vs new workspaces.

---

### Finding 5: Verification Bottleneck is the Meta-Principle

**Evidence:** The Jan 10 investigation established the foundational principle:

> "The system cannot change faster than a human can verify behavior."

This was discovered after two major rollbacks:
- First spiral (Dec 21): 115 commits in 24h → rollback
- Second spiral (Dec 27-Jan 2): 347 commits in 6 days → rollback

All sampled commits were individually correct. The problem was compositional.

**Source:** `2026-01-10-inv-trace-verification-bottleneck-story-system.md:8-16`

**Significance:** Verification exists to enforce the pace constraint. But current implementation creates false positives (blocking good work) and false negatives (passing bad work), undermining the purpose.

---

## Synthesis

**Key Insights:**

1. **Agent Identity is Distributed and Inconsistent** - There is no single source of truth for "what did this agent do." Gates independently resolve workspace path, spawn time, project directory, and beads ID, leading to inconsistent scoping.

2. **Gates Are Individually Correct but Compositionally Fragile** - Each gate was designed for a specific failure mode. But interactions between gates, concurrent agents, and legacy workspaces create emergent bugs.

3. **Claims Are Easier to Check but Less Reliable Than Evidence** - Claim-based gates (keyword matching) are simple but unreliable. Evidence-based gates (file existence, git diff) are reliable but harder to scope correctly.

4. **Legacy Compatibility Degrades Verification Quality** - "Skip if missing" patterns create two verification regimes: strict for new workspaces, permissive for old ones.

5. **High Bypass Rate Indicates Systematic Issues** - 55% force-bypass rate means gates are failing legitimate completions, not just blocking bad ones.

**Answer to Investigation Question:**

Completion verification generates churn because:

1. **No canonical agent identity** forces each gate to implement its own agent-scoping logic (Finding 1)
2. **12 gates with independent edge cases** creates multiplicative failure modes (Finding 2)
3. **Claim vs evidence conflation** makes gates unreliable (Finding 3)
4. **Legacy compatibility** degrades verification quality (Finding 4)
5. **These issues compound** when multiple agents run concurrently or cross-project

The architectural improvements proposed in the Recommendations section address these root causes.

---

## Structured Uncertainty

**What's tested:**

- ✅ 12 gates exist in pkg/verify/check.go (verified: read source)
- ✅ 55% force bypass rate mentioned in investigations (verified: Jan 14 investigation)
- ✅ Concurrent agent pollution occurs (verified: Jan 8 test_evidence investigation)
- ✅ Missing spawn_time causes wrong git command (verified: Jan 14 git_diff investigation)

**What's untested:**

- ⚠️ Whether Agent Manifest would actually reduce churn (architectural proposal, not implemented)
- ⚠️ Whether git commit-based scoping works in practice (needs implementation testing)
- ⚠️ Whether reducing gate count would maintain verification quality

**What would change this:**

- If spawn_time-based scoping is actually sufficient (would need to disprove concurrent agent bugs)
- If bypass rate is actually lower than 55% (would need fresh metrics)
- If there's an existing canonical agent identity system I missed

---

## Implementation Recommendations

**Purpose:** Reduce verification churn by addressing root causes, not symptoms.

### Recommended Approach ⭐

**Agent Manifest + Git-Based Scoping + Evidence Collection Phase**

**Why this approach:**
- Agent Manifest provides canonical identity (addresses Finding 1)
- Git commit-based scoping eliminates concurrent agent pollution (addresses Finding 1)
- Evidence collection before gates enables reliable evidence-based checking (addresses Finding 3)

**Trade-offs accepted:**
- Requires spawn-time changes to create manifest
- Git commit scoping requires author/committer metadata discipline
- Evidence collection phase adds verification time

**Implementation sequence:**

1. **Agent Manifest at Spawn Time**
   - Create `.orch/workspace/{name}/AGENT_MANIFEST.json` at spawn
   - Contains: spawn_time, project_dir, beads_id, skill, git_base_commit
   - All gates read from manifest, not individual files

2. **Git Commit-Based Change Detection**
   - Record current git commit SHA in manifest at spawn
   - At verification, diff from base_commit to HEAD
   - Filter commits by workspace modifications (agent-specific)
   - Eliminates time-based scoping issues

3. **Evidence Collection Phase**
   - Before running gates, collect evidence:
     - Files changed (from git diff)
     - Screenshots (from workspace/screenshots/)
     - Test output (from beads comments + actual files)
     - Phase claims (from beads comments)
   - Gates operate on collected evidence, not raw sources

4. **Gate Consolidation (Future)**
   - Consider merging related gates (test_evidence + git_diff → code_changes)
   - Make gates skill-aware from start, not post-hoc exceptions
   - Remove "skip if missing" branches after migration period

### Alternative Approaches Considered

**Option B: Fix Individual Gate Bugs**
- **Pros:** Lower risk, incremental progress
- **Cons:** Doesn't address root causes, churn will continue
- **When to use instead:** When architectural change is blocked by resources

**Option C: Simplify to 3 Gates (Phase + Artifact + Build)**
- **Pros:** Dramatically reduces complexity
- **Cons:** Loses nuanced verification (visual, test evidence, constraints)
- **When to use instead:** If false positive rate is unacceptably high

**Rationale for recommendation:** Option A addresses root causes while preserving verification depth. Options B and C either don't solve the problem (B) or sacrifice too much (C).

---

### Implementation Details

**What to implement first:**
- Agent Manifest is foundational (gates need it to work correctly)
- Git-based scoping is highest impact for reducing false positives
- Evidence collection enables reliable evidence-based gates

**Things to watch out for:**
- ⚠️ Migration path for existing workspaces without manifests
- ⚠️ Performance of git diff for large repositories
- ⚠️ Author/committer metadata reliability (depends on git config)

**Areas needing further investigation:**
- How to handle cross-project spawns in manifest
- Whether git author filtering works for all spawn modes
- What evidence format enables reliable gate checking

**Success criteria:**
- ✅ Force bypass rate drops from 55% to <10%
- ✅ Gate failure investigations drop from ~2/week to <1/month
- ✅ No concurrent agent pollution bugs

---

## References

**Files Examined:**
- `pkg/verify/check.go` - Main verification logic, gate constants
- `.kb/models/completion-verification/model.md` - Current architecture model
- `.kb/guides/completion.md` - Completion workflow guide

**Investigations Analyzed:**
- `2026-01-08-inv-synthesize-completion-investigations-10-synthesis.md`
- `2026-01-14-inv-synthesize-verification-investigations-consolidate-findings.md`
- `2026-01-14-inv-fix-git-diff-verification-false.md`
- `2026-01-08-inv-bug-test-evidence-gate-triggers.md`
- `2026-01-14-inv-implement-targeted-skip-gate-flags.md`
- `2026-01-08-inv-25-28-agents-not-completing.md`
- `2026-01-15-inv-verify-test-first-gate-implementation.md`

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification/model.md` - Should be updated with these findings
- **Guide:** `.kb/guides/completion.md` - Reference for current workflow
- **Decision (proposed):** `.kb/decisions/verification-agent-manifest.md` - If recommendations accepted

---

## Investigation History

**2026-01-17 00:55:** Investigation started
- Initial question: Why does completion verification generate churn?
- Context: 26+ investigations on completion/verification indicate architectural friction

**2026-01-17 01:15:** Prior syntheses reviewed
- Found Jan 8 completion synthesis (10 investigations)
- Found Jan 14 verification synthesis (25 investigations)
- Identified 4-layer verification architecture and evolution phases

**2026-01-17 01:30:** Root causes identified
- Agent-scoping ambiguity as primary cause
- Gate proliferation as complexity multiplier
- Evidence vs claim conflation
- Legacy compatibility burden

**2026-01-17 01:45:** Investigation completed
- Status: Complete
- Key outcome: 4 root causes identified; Agent Manifest + Git-based scoping + Evidence Collection recommended to reduce churn
