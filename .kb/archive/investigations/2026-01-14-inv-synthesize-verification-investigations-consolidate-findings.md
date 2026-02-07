<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Synthesized 25 verification-related investigations spanning Dec 2025 - Jan 2026, revealing 4 distinct verification layers: (1) Verification Bottleneck Principle (meta-level human-AI pace constraint), (2) Visual Verification System (UI change gates), (3) Declarative Skill Verification (constraints + phases), (4) Completion Verification Architecture (already documented in completion.md guide).

**Evidence:** Read and analyzed 25 investigations covering: trace verification bottleneck (462 commits lost), visual verification scoping, risk-based categorization, constraint/phase extraction, screenshot file detection, completion workflows, and CLI verification testing.

**Knowledge:** Verification is multi-layered by design - from meta-principle (pace changes to verification bandwidth) to concrete gates (phase complete + evidence + approval). Visual verification uses spawn-time-based scoping, not HEAD~5. Constraint verification uses declarative HTML blocks. Most completion verification is already in `.kb/guides/completion.md`.

**Next:** Recommend creating `.kb/guides/verification.md` to consolidate visual verification and declarative constraint patterns not covered in completion guide. Archive 18 implementation-complete investigations, keep 7 as current reference.

**Promote to Decision:** recommend-yes - Establish verification guide as authoritative reference for visual/constraint/phase verification, complementing existing completion guide.

---

# Investigation: Synthesize Verification Investigations - Consolidate Findings

**Question:** What patterns, decisions, and reusable knowledge exist across 25 verification-related investigations, and how should they be consolidated?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-synthesize-verification-investigations-14jan-8877
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Investigations Analyzed

| # | Date | Investigation | Key Finding |
|---|------|---------------|-------------|
| 1 | 2026-01-10 | Trace Verification Bottleneck Story | "System cannot change faster than humans verify" - 462 commits lost |
| 2 | 2026-01-10 | Phase 2 Completion Verification | Verified Phase 2 complete; launchd issue separate |
| 3 | 2026-01-10 | Verify Lagging Understanding | Meta: understanding lags behind system changes |
| 4 | 2026-01-09 | Verification Test OpenCode Run Attach | `opencode run --attach` works for sending messages |
| 5 | 2026-01-08 | Verification Risk-Based Visual | LOW/MEDIUM/HIGH risk categorization for web/ changes |
| 6 | 2026-01-08 | Feature Add Screenshot File Verification | `HasScreenshotFilesInWorkspace()` checks actual image files |
| 7 | 2026-01-08 | Synthesize Completion Investigations | 10 completion investigations → completion.md guide |
| 8 | 2025-12-23 | Implement Phase Gates Verification | `<!-- SKILL-PHASES -->` block extraction |
| 9 | 2025-12-23 | Implement Skill Constraint Verification | Constraint verification already complete |
| 10 | 2025-12-23 | Implement Constraint Extraction | `ExtractConstraints()` from SPAWN_CONTEXT.md |
| 11 | 2026-01-03 | Visual Verification Checks Project Git | Duplicate - fix already existed |
| 12 | 2026-01-02 | Debug Visual Verification Scope | Root cause: HEAD~5 → spawn-time-based filtering |
| 13 | 2026-01-03 | Test Verify Daemon Skip | Validation test for daemon functionality |
| 14 | 2026-01-03 | Recover Priority Verification Gates Git | Git recovery for verification gates |
| 15 | 2026-01-03 | Verify Launchd Documentation | launchd documentation verified |
| 16 | 2026-01-04 | Design Analyze Pkg Verify Check | pkg/verify architecture analysis |
| 17 | 2025-12-25 | Migrate Verify GetComments | API migration for comments |
| 18 | 2025-12-25 | Integrate Skillc Verify | skillc integration with orch |
| 19 | 2025-12-25 | Verify Gap Detection Wired | Gap detection verification |
| 20 | 2025-12-24 | Fix Orch Clean Verify OpenCode | Clean command verification |
| 21 | 2025-12-23 | Verify Spawn Works | Basic spawn verification |
| 22 | 2025-12-21 | Post Install Verify | Post-install verification |
| 23 | 2025-12-21 | Implement Verify OpenCode Disk Session | OpenCode session verification |
| 24 | 2026-01-14 | Orch Doctor Verify Dashboard Fetch | Dashboard fetch verification |
| 25 | 2026-01-14 | Synthesize Verification (empty) | Template only |

---

## Findings

### Finding 1: Four Distinct Verification Layers

**Evidence:** The 25 investigations organize into 4 distinct layers:

**Layer 1: Verification Bottleneck Principle (Meta-Level)**
- Established after two rollbacks totaling 462 commits
- Key insight: "System cannot change faster than humans can verify"
- Local correctness ≠ global correctness
- Applies to code changes AND understanding changes
- **Source:** `2026-01-10-inv-trace-verification-bottleneck-story-system.md`

**Layer 2: Visual Verification System (UI Changes)**
- Spawn-time-based scoping (not HEAD~5)
- Risk-based categorization (LOW ≤10 lines CSS, MEDIUM, HIGH)
- Screenshot file detection in workspace/screenshots/
- Three gates: Phase + Evidence + Approval
- **Source:** Multiple: `visual-verification-scope.md`, `risk-based-visual-verification.md`, `screenshot-file-verification.md`

**Layer 3: Declarative Skill Verification (Constraints + Phases)**
- `<!-- SKILL-CONSTRAINTS -->` block embedded in SPAWN_CONTEXT.md
- `<!-- SKILL-PHASES -->` block for phase gate enforcement
- Pattern-to-glob conversion with variable substitution
- Required vs optional semantics
- **Source:** `implement-phase-gates-verification.md`, `implement-constraint-extraction-verification.md`

**Layer 4: Completion Verification Architecture**
- Already well-documented in `.kb/guides/completion.md`
- Three verification layers: Phase gate, Evidence gate, Approval gate
- Skill-type-specific paths (worker vs orchestrator)
- Cross-project completion with auto-detection
- **Source:** `synthesize-completion-investigations.md`

**Significance:** Verification is designed as defense-in-depth, not a single gate. Each layer catches different failure modes.

---

### Finding 2: Verification Bottleneck is Foundational Principle

**Evidence:** The trace verification bottleneck investigation reveals a meta-principle applicable across all verification:

> "The system cannot change faster than a human can verify behavior."

Key supporting evidence:
- First spiral (Dec 21): 115 commits in 24h → rollback
- Second spiral (Dec 27-Jan 2): 347 commits in 6 days → rollback
- All sampled "fix:" commits were individually correct
- Problem was compositional, not individual

This principle applies to:
1. **Code changes** - Changes outpacing verification creates incoherence
2. **Understanding changes** - Observability improvements misinterpreted as degradation
3. **Documentation** - Can be correct but overwhelming

**Source:** `2026-01-10-inv-trace-verification-bottleneck-story-system.md:8-16`

**Significance:** This is the foundational constraint for human-AI collaboration. All other verification gates exist to enforce this principle.

---

### Finding 3: Visual Verification Uses Spawn-Time Scoping

**Evidence:** Visual verification evolved from broken (HEAD~5) to correct (spawn-time-based):

**Before (broken):**
```go
cmd := exec.Command("git", "diff", "--name-only", "HEAD~5..HEAD")
```
This checked all recent commits, creating false positives when prior agents modified web/.

**After (correct):**
```go
func HasWebChangesForAgent(projectDir, workspacePath string) bool {
    spawnTime := spawn.ReadSpawnTime(workspacePath)
    return hasWebChangesSinceTime(projectDir, spawnTime)
}
```

The fix uses:
1. Workspace's `.spawn_time` file
2. Git's `--since` flag for time-based filtering
3. Fallback to HEAD~5 for legacy workspaces

**Source:** `2026-01-02-debug-visual-verification-scope.md`, `2026-01-03-inv-visual-verification-checks-project-git.md`

**Significance:** Agent-scoped verification must use spawn time, not fixed commit counts. This pattern is consistent with constraint verification.

---

### Finding 4: Risk-Based Verification Reduces Friction

**Evidence:** Risk-based categorization was implemented for web/ file changes:

| Risk Level | Criteria | Verification |
|------------|----------|--------------|
| LOW | ≤10 lines CSS, ≤5 lines component | Skip visual verification |
| MEDIUM | 6-30 lines changes | Standard visual verification |
| HIGH | New routes, layouts, >30 lines | Required visual verification |

**Implementation:**
- `pkg/verify/visual.go` implements `categorizeChangeRisk()`
- Takes max risk across all changed files
- New routes/layouts always HIGH risk

**Source:** `2026-01-08-inv-verification-risk-based-visual-verification.md`

**Significance:** Not all web/ changes require the same verification level. Risk categorization reduces friction for trivial CSS changes while maintaining gates for structural changes.

---

### Finding 5: Screenshot File Verification Provides Concrete Evidence

**Evidence:** Visual verification now checks for actual image files, not just keyword mentions:

```go
func HasScreenshotFilesInWorkspace(workspacePath string) bool {
    // Scans {workspace}/screenshots/ for PNG, JPG, JPEG, WEBP, GIF
}
```

**Three evidence sources for visual verification:**
1. Beads comments (keyword mentions like "screenshot")
2. SYNTHESIS.md content (Evidence section)
3. Actual screenshot files in workspace

**Why this matters:** An agent claiming "screenshot captured" might have failed to save. File existence proves the screenshot was actually captured.

**Source:** `2026-01-08-inv-feature-add-screenshot-file-verification.md`

**Significance:** Evidence hierarchy: files > claims. Concrete artifacts are stronger evidence than text mentions.

---

### Finding 6: Declarative Constraint System is Complete

**Evidence:** The constraint verification system is fully implemented:

**Layer 1 (skillc):**
```markdown
<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation file -->
<!-- optional: .kb/decisions/{date}-*.md | Promoted decision -->
<!-- /SKILL-CONSTRAINTS -->
```

**Layer 2 (orch-go):**
- `ExtractConstraints()` - Parses SPAWN_CONTEXT.md
- `PatternToGlob()` - Converts {date}, {workspace}, {beads} to wildcards
- `VerifyConstraints()` - Checks files match patterns
- `VerifyCompletionFull()` - Integrates with completion flow

**Phase gates follow same pattern:**
- `<!-- SKILL-PHASES -->` block
- Extract reported phases from beads comments
- Verify required phases were reported

**Source:** `2025-12-23-inv-implement-constraint-extraction-verification.md`, `2025-12-23-inv-implement-phase-gates-verification.md`

**Significance:** Skills can declaratively define deliverables that are automatically verified at completion time.

---

## Synthesis

**Key Insights:**

1. **Verification is Defense-in-Depth** - Four layers (meta-principle, visual, declarative, completion) each catch different failure modes. No single gate is sufficient.

2. **Verification Bottleneck is Foundational** - All other verification patterns exist to enforce the meta-principle: "pace changes to verification bandwidth." This emerged from 462 lost commits.

3. **Spawn-Time Scoping is Universal** - Both visual verification and constraint verification use spawn time to scope what counts as "agent's work." Fixed commit counts are wrong.

4. **Evidence Hierarchy Exists** - Files > claims. Screenshot files are stronger evidence than text mentions. Test results are stronger than "tests pass" claims.

5. **Completion Guide Already Covers Most** - The existing `.kb/guides/completion.md` covers completion verification thoroughly. A separate verification guide should focus on visual and declarative verification patterns not covered there.

**Answer to Investigation Question:**

The 25 verification investigations reveal a mature, multi-layered verification system:

- **Meta-level:** Verification Bottleneck principle (pace changes to bandwidth)
- **Visual:** Spawn-time scoping, risk categorization, file verification
- **Declarative:** Constraint and phase extraction from skill definitions
- **Completion:** Well-documented in existing completion.md guide

**Consolidation recommendation:**
- Create `.kb/guides/verification.md` for visual + declarative verification patterns
- Archive 18 implementation-complete investigations
- Keep 7 recent/diagnostic investigations as current reference
- Reference completion.md for completion-specific workflows

---

## Structured Uncertainty

**What's tested:**

- ✅ Visual verification uses spawn-time scoping (verified: code in visual.go, tests pass)
- ✅ Constraint extraction parses SKILL-CONSTRAINTS blocks (verified: 22 tests pass)
- ✅ Screenshot file detection works for PNG, JPG, etc. (verified: unit tests)
- ✅ Risk categorization implemented (verified: code review)

**What's untested:**

- ⚠️ Whether Verification Bottleneck principle is formally documented (recommend decision document)
- ⚠️ Whether all skills use constraint/phase blocks (adoption rate unknown)
- ⚠️ Whether risk thresholds are well-calibrated (no production validation)

**What would change this:**

- Finding would be wrong if spawn-time scoping doesn't work in production (tests say it does)
- Finding would be wrong if declarative constraints aren't being enforced (VerifyCompletionFull calls them)
- Consolidation recommendation would change if significant gaps found in analysis

---

## Implementation Recommendations

### Recommended Approach ⭐

**Create `.kb/guides/verification.md`** - Complement completion.md with visual and declarative verification patterns.

**Why this approach:**
- Avoids duplicating completion.md content
- Provides single reference for visual verification workflow
- Documents declarative constraint/phase system
- Follows established guide consolidation pattern

**Trade-offs accepted:**
- Some overlap with completion guide inevitable
- Doesn't cover verification bottleneck meta-principle (recommend separate decision doc)

**Implementation sequence:**
1. Create guide skeleton with visual + declarative sections
2. Archive 18 implementation-complete investigations
3. Add cross-references between guides

### Archive Candidates

| Investigation | Status | Reason |
|---------------|--------|--------|
| `2025-12-21-inv-post-install-verify.md` | Archive | Implementation complete |
| `2025-12-21-inv-implement-verify-opencode-disk-session.md` | Archive | Implementation complete |
| `2025-12-23-inv-verify-spawn-works.md` | Archive | Validation test |
| `2025-12-23-inv-implement-constraint-extraction-verification.md` | Archive | Implementation complete |
| `2025-12-23-inv-implement-skill-constraint-verification.md` | Archive | Implementation complete |
| `2025-12-23-inv-implement-phase-gates-verification.md` | Archive | Implementation complete |
| `2025-12-24-inv-fix-orch-clean-verify-opencode.md` | Archive | Fix complete |
| `2025-12-25-inv-migrate-verify-getcomments.md` | Archive | Migration complete |
| `2025-12-25-inv-integrate-skillc-verify-into-orch.md` | Archive | Integration complete |
| `2025-12-25-inv-verify-gap-detection-wired-into.md` | Archive | Implementation complete |
| `2026-01-02-debug-visual-verification-scope.md` | Archive | Fix complete |
| `2026-01-03-inv-visual-verification-checks-project-git.md` | Archive | Duplicate of above |
| `2026-01-03-inv-test-verify-daemon-skip-functionality.md` | Archive | Validation test |
| `2026-01-03-inv-verify-launchd-documentation.md` | Archive | Documentation verified |
| `2026-01-03-inv-recover-priority-verification-gates-git.md` | Archive | Recovery complete |
| `2026-01-08-inv-feature-add-screenshot-file-verification.md` | Archive | Implementation complete |
| `2026-01-08-inv-verification-risk-based-visual-verification.md` | Archive | Implementation complete |
| `2026-01-09-inv-verification-test-opencode-run-attach.md` | Archive | Validation test |

### Keep as Reference

| Investigation | Reason |
|---------------|--------|
| `2026-01-10-inv-trace-verification-bottleneck-story-system.md` | Foundational principle, blog-ready narrative |
| `2026-01-10-inv-phase-2-completion-verification.md` | Recent completion verification example |
| `2026-01-10-inv-verify-lagging-understanding-hypothesis.md` | Meta-level insight |
| `2026-01-04-inv-design-analyze-pkg-verify-check.md` | Architecture reference |
| `2026-01-08-inv-synthesize-completion-investigations.md` | Synthesis reference |
| `2026-01-14-inv-orch-doctor-verify-dashboard-fetch.md` | Recent/active |
| `2026-01-14-inv-synthesize-verification-investigations.md` | This synthesis |

---

## References

**Files Examined:**
- 25 verification-related investigations in `.kb/investigations/`
- `.kb/guides/completion.md` - Existing completion guide (overlaps with verification)
- `pkg/verify/visual.go` - Visual verification implementation
- `pkg/verify/constraint.go` - Constraint verification implementation
- `pkg/verify/phase_gates.go` - Phase gate verification implementation

**Key Code References:**
- `pkg/verify/visual.go:159-186` - `HasWebChangesForAgent()` with spawn-time scoping
- `pkg/verify/constraint.go:44-113` - `ExtractConstraints()` parsing
- `pkg/verify/phase_gates.go:32-85` - Phase extraction
- `pkg/verify/check.go:330-382` - `VerifyCompletionFull()` integration

**Related Artifacts:**
- **Guide:** `.kb/guides/completion.md` - Completion workflow (overlaps verification)
- **Decision (proposed):** `.kb/decisions/verification-bottleneck-principle.md`

---

## Investigation History

**2026-01-14 11:55:** Investigation started
- Initial question: Consolidate 25 verification investigations
- Context: kb reflect flagged synthesis opportunity

**2026-01-14 12:15:** Read and categorized all 25 investigations
- Identified 4 distinct verification layers
- Found significant overlap with existing completion.md guide
- Found Verification Bottleneck principle as foundational

**2026-01-14 12:30:** Synthesis complete
- 18 archive candidates identified
- 7 keep as reference
- Recommend new verification.md guide for visual + declarative patterns

**2026-01-14 12:45:** Investigation completed
- Status: Complete
- Key outcome: Four-layer verification architecture documented; recommend new guide for visual/declarative patterns
