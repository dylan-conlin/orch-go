<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Validation gates should exist in BOTH skill (feature-impl validation phase) AND orch complete (evidence gate prompt), using defense-in-depth with tier-based validation levels.

**Evidence:** Current test_evidence.go gate only checks for "tests ran" patterns, not "behavior observed working"; feature-impl validation phase exists but lacks "demonstrate it works" requirement; Verification Bottleneck principle requires human observation, not just agent claims.

**Knowledge:** Three test types (unit/integration/behavioral) provide different confidence levels. Unit tests with mocks can pass while behavior is broken. The gap is distinguishing "tests pass" from "behavior verified working locally."

**Next:** (1) Add behavioral validation tier to feature-impl validation phase, (2) Add validation_evidence prompt to orch complete for behavior-changing work, (3) Create three-tier validation evidence taxonomy.

**Promote to Decision:** Actioned - decision exists (verification-bottleneck-principle)

---

# Investigation: Validation Gate Design for Practical Verification

**Question:** Where should validation gates live (skill, orch complete, or both) and what evidence structure is needed to prevent "tests pass but behavior unverified" gaps?

**Started:** 2026-01-26
**Updated:** 2026-01-26
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implement validation evidence tier and orch complete prompt
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Context: The Original Problem

**Incident (Jan 24, 2026):** During price-watch re-scrape fix, orchestrator reviewed agent work for Redis distributed lock. Agent reported "tests pass." Orchestrator was ready to deploy without suggesting local validation. Dylan caught this gap.

**Root cause analysis:**
1. Over-indexed on "tests pass" as sufficient validation
2. Momentum toward completion skipped "does it actually work?" step
3. Todo list framed "verify in production" as next step, skipping local
4. Verification Bottleneck principle not applied at review time

**Core question:** How do we distinguish between "tests pass" and "behavior verified working"?

---

## Findings

### Finding 1: Current test_evidence gate checks test execution, not behavioral verification

**Evidence:** `pkg/verify/test_evidence.go:78-116` defines `testEvidencePatterns` that match:
- "go test ./... - PASS"
- "ok package/name 0.123s"
- "15 passing, 0 failing"
- etc.

The patterns verify that *tests ran*, not that *behavior was observed working*. A test suite with all mocks can pass while actual behavior is broken.

**Source:** `pkg/verify/test_evidence.go:78-116`

**Significance:** The existing gate catches "agent didn't run tests" but cannot catch "tests pass but behavior not verified." These are different failure modes requiring different gates.

---

### Finding 2: feature-impl validation phase lacks "demonstrate behavior" requirement

**Evidence:** From `~/.claude/skills/worker/feature-impl/SKILL.md:266-297`, the validation phase supports four levels:
- `none` - just commit
- `tests` - run tests, verify pass
- `smoke-test` - tests + manual verification + evidence capture
- `multi-phase` - tests + smoke + orchestrator approval

The gap: `smoke-test` is the only level that includes "manual verification" but:
1. It's not the default for behavior-changing work
2. The guidance says "manual verification + evidence capture" but doesn't define what evidence demonstrates behavior working
3. There's no distinction between "I clicked the button" and "I observed the expected outcome"

**Source:** `~/.claude/skills/worker/feature-impl/SKILL.md:266-297`

**Significance:** The skill has infrastructure for behavioral validation but doesn't require or define it clearly. Agents can pass `smoke-test` with vague claims like "visually verified."

---

### Finding 3: Three test types provide different confidence levels

**Evidence:** Standard testing taxonomy:

| Test Type | What It Validates | Failure Mode |
|-----------|-------------------|--------------|
| **Unit (mocked)** | Logic correctness | Behavior broken despite tests pass (mocks don't match reality) |
| **Integration** | Component interaction | Edge cases, timing, real dependencies not tested |
| **Behavioral/E2E** | Actual user-visible behavior | None for tested paths (but expensive, slow) |

The Redis distributed lock example: Unit tests could mock the Redis client and pass. Integration tests could use a test Redis and pass. Only behavioral testing (actually running the scraper with a real lock) would reveal if the lock works in production conditions.

**Source:** Industry standard test pyramid; Verification Bottleneck principle from `~/.kb/principles.md:343-369`

**Significance:** "Tests pass" is ambiguous. Different test types provide different confidence levels. The validation system should distinguish between them and require appropriate evidence for the risk level.

---

### Finding 4: Verification Bottleneck principle requires human observation

**Evidence:** From `~/.kb/principles.md:343-369`:
> "The system cannot change faster than a human can verify behavior. Velocity without verification is regression with extra steps."
> "Has a human observed this working, or just read that it works?"

The principle explicitly distinguishes "observed" from "read that it works." Agent claims like "tests pass" are reading, not observing.

**Source:** `~/.kb/principles.md:343-369`

**Significance:** This is the foundational constraint. Any solution must result in human observation of behavior for risky changes. Gates without human observation don't satisfy the principle.

---

### Finding 5: Defense in depth is the correct architecture

**Evidence:** From `.kb/models/completion-verification.md:17-28`, the system uses three independent gates (Phase, Evidence, Approval) precisely because each catches different failure modes:
- Phase gate catches "agent didn't finish"
- Evidence gate catches "agent didn't run tests"
- Approval gate catches "agent didn't verify UI visually"

Similarly, from `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md`, the recommended approach for bloat control uses two layers: spawn-time surfacing + CI gate enforcement.

**Source:** `.kb/models/completion-verification.md:17-28`, `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md:123-140`

**Significance:** The precedent is clear: defense in depth with multiple gates. Validation should exist at skill level (agent must produce evidence) AND at orch complete level (orchestrator must review evidence).

---

### Finding 6: Trivial changes shouldn't require behavioral validation

**Evidence:** The spawn context mentions "avoiding ceremony for trivial changes." Examples of trivial changes:
- Documentation updates (markdown only)
- Code formatting/style fixes
- Dependency updates with no behavior change
- Configuration changes with test coverage

These don't require behavioral validation - test evidence suffices.

**Source:** Issue description, general software engineering practice

**Significance:** The gate must be smart enough to not require behavioral validation for changes where tests are sufficient. Over-gating creates workarounds.

---

## Synthesis

**Key Insights:**

1. **Tests ≠ Behavior Verification** - The gap in the Jan 24 incident was not that tests weren't run, but that behavior wasn't observed. The current test_evidence gate addresses a different failure mode.

2. **Defense in Depth via Layers** - Following the completion verification architecture, validation should exist at:
   - **Skill level (feature-impl):** Agent must produce validation evidence appropriate to change type
   - **orch complete level:** Orchestrator must review validation evidence before closing

3. **Tier-Based Validation** - Not all changes need the same validation level:
   - **Trivial (tests-only):** Refactoring, docs, config with test coverage → test evidence sufficient
   - **Standard (integration):** New features with integration tests → test evidence + integration verification
   - **Behavioral (demonstrated):** Behavior changes, external integrations → must demonstrate behavior working locally

4. **Evidence Structure Matters** - The key insight: validation evidence must answer "What behavior did you observe?" not just "Did you run tests?" This requires structured evidence capture.

**Answer to Investigation Question:**

Validation gates should exist in BOTH:

1. **feature-impl skill:** Enhanced validation phase that requires tier-appropriate evidence
2. **orch complete:** Validation evidence prompt that asks "Did you observe behavior working?" for behavioral-tier changes

The evidence structure should distinguish test types and require demonstrated behavior for risky changes. This applies the Verification Bottleneck principle while avoiding ceremony for trivial changes.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current test_evidence.go only checks test execution patterns (verified: read source code)
- ✅ feature-impl validation phase exists with four levels (verified: read skill source)
- ✅ Verification Bottleneck principle requires human observation (verified: read principles.md)
- ✅ Defense in depth is the established pattern (verified: read completion-verification model)

**What's untested:**

- ⚠️ Whether agents will produce meaningful behavioral evidence vs gaming the gate (not observed in production)
- ⚠️ Optimal trigger for behavioral vs tests-only tier (heuristics not validated)
- ⚠️ Whether orch complete prompt will be effective vs skipped (not deployed)

**What would change this:**

- If behavioral evidence is consistently gamed → need stricter gate (Playwright screenshot requirement)
- If tier detection has too many false positives → reduce scope to explicit flag
- If orchestrators consistently skip prompt → need blocking gate instead of prompt

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Two-Layer Validation with Tier-Based Evidence**

**Why this approach:**
- Applies defense in depth (skill layer + completion layer)
- Respects Gate Over Remind (orch complete gate, not just reminder)
- Follows Verification Bottleneck (requires human observation for risky changes)
- Avoids over-gating (tier-based, not blanket requirement)

**Trade-offs accepted:**
- Some agent burden to produce validation evidence
- Orchestrator must review evidence (cannot fully automate)
- Tier detection heuristics may have false positives/negatives

**Implementation sequence:**

1. **Define validation evidence taxonomy** - Three tiers with clear evidence requirements
2. **Add behavioral validation tier to feature-impl** - "Demonstrated behavior" requirement for risky changes
3. **Add validation_evidence prompt to orch complete** - "What behavior did you observe?" for behavioral-tier work
4. **Update spawn context generation** - Inject validation tier based on change type heuristics

### Alternative Approaches Considered

**Option B: Blocking gate on orch complete only**
- **Pros:** Single point of enforcement, simpler
- **Cons:** Agent doesn't know validation is required until completion → late feedback, wasted work
- **When to use instead:** If skill-level changes prove too complex

**Option C: Playwright/screenshot requirement for all**
- **Pros:** Unambiguous evidence, hard to game
- **Cons:** Massive overhead for non-UI changes, false requirement for backend work
- **When to use instead:** If behavioral evidence is consistently gamed

**Option D: Trust agent claims, no additional gates**
- **Pros:** No ceremony, fast completion
- **Cons:** Doesn't solve the problem (Jan 24 incident will recur)
- **When to use instead:** Never for behavior-changing work

**Rationale for recommendation:** Two-layer validation with tiers balances thoroughness (catches the Jan 24 gap) with pragmatism (doesn't over-gate trivial changes). It follows established patterns (completion verification model) and principles (Verification Bottleneck, Gate Over Remind).

---

### Implementation Details

**What to implement first:**

1. **Validation Evidence Taxonomy** - Define the three tiers clearly:

```markdown
## Validation Evidence Tiers

| Tier | Trigger | Required Evidence | Example |
|------|---------|-------------------|---------|
| **Tests-Only** | Docs, refactoring, config | Test output (existing gate) | "go test ./... - PASS (47 tests in 2.1s)" |
| **Integration** | New features with tests | Test output + integration verification | "Tests pass + API responds correctly" |
| **Behavioral** | Behavior changes, external integrations | Demonstrated behavior working | "Ran locally: lock acquired, second process blocked as expected" |
```

2. **feature-impl Skill Update** - Add `validation: behavioral` level:

```markdown
### Validation Phase

| Level | Workflow |
|-------|----------|
| `none` | Commit, report complete |
| `tests` | Run test suite, verify pass, commit |
| `integration` | Tests + verify integration points work |
| `behavioral` | Tests + **demonstrate behavior locally** + capture evidence |
| `multi-phase` | Tests + behavioral + STOP for orchestrator approval |

**⚠️ Behavioral Validation (MANDATORY if behavior changes):**

Before completing, you must DEMONSTRATE the behavior works:
1. Run the feature/fix locally (not just tests)
2. Observe the expected behavior occurs
3. Document what you observed: `bd comment <id> "Behavior verified: [what you did] → [what you observed]"`

Example: "Behavior verified: Ran scraper with Redis lock → second instance waited until first completed"
```

3. **orch complete Prompt** - Add validation evidence check:

```go
// In pkg/verify/behavioral.go (new file)

// BehavioralValidationResult captures whether behavioral evidence was found
type BehavioralValidationResult struct {
    Passed bool
    RequiresBehavioral bool // Based on change type heuristics
    HasBehavioralEvidence bool
    Evidence []string
    Errors []string
    Warnings []string
}

// behavioralEvidencePatterns match actual behavior demonstration
var behavioralEvidencePatterns = []*regexp.Regexp{
    regexp.MustCompile(`(?i)behavior\s+verified:`),           // "Behavior verified: ..."
    regexp.MustCompile(`(?i)ran\s+locally:.*→`),              // "Ran locally: X → Y"
    regexp.MustCompile(`(?i)observed\s+expected\s+behavior`), // "Observed expected behavior"
    regexp.MustCompile(`(?i)demonstrated:.*working`),         // "Demonstrated: X working"
    regexp.MustCompile(`(?i)manual\s+test:.*passed`),         // "Manual test: X passed"
}

// Heuristics for when behavioral validation is required
func RequiresBehavioralValidation(changedFiles []string, commitMessages []string) bool {
    // External integrations (Redis, APIs, databases)
    // Behavior changes (not pure refactoring)
    // User-visible changes (UI, CLI output)
    // Concurrency changes (locks, async)
    // ...
}
```

4. **orch complete Prompt** (in complete command):

```go
// If behavioral validation required but no evidence found
if result.RequiresBehavioral && !result.HasBehavioralEvidence {
    fmt.Println("⚠️  VALIDATION CHECK: This appears to be a behavior-changing work.")
    fmt.Println("")
    fmt.Println("Did you observe the behavior working locally, or just see tests pass?")
    fmt.Println("")
    fmt.Println("To proceed, confirm observation:")
    fmt.Println("  orch complete <id> --observed \"[what you observed]\"")
    fmt.Println("")
    fmt.Println("Or skip with reason:")
    fmt.Println("  orch complete <id> --skip-behavioral \"[reason]\"")
}
```

**Things to watch out for:**

- ⚠️ Behavioral tier detection heuristics need tuning - start conservative (explicit flag or keywords)
- ⚠️ Don't block on "behavioral evidence" for changes that genuinely can't be demonstrated locally (infra, CI)
- ⚠️ Evidence patterns must distinguish real observation from templated claims

**Areas needing further investigation:**

- Optimal heuristics for behavioral tier detection (could use commit message keywords, file paths, skill type)
- Whether to require Playwright/screenshot for UI behavioral evidence
- How to handle "can't demonstrate locally" cases (cloud-only, production-only features)

**Success criteria:**

- ✅ Jan 24 scenario prevented: Agent completing behavior-changing work without demonstration gets prompted
- ✅ Trivial changes not over-gated: Docs/refactoring flows through tests-only tier
- ✅ Evidence distinguishable: "Tests pass" vs "Behavior verified: X → Y"
- ✅ Defense in depth: Skill warns, orch complete gates

---

## Implementation Recommendations (Fork Navigation)

### Fork 1: Where does the gate live?

**Options:**
- A: Skill only (feature-impl validation phase)
- B: orch complete only
- C: Both (defense in depth)

**Substrate says:**
- Model `.kb/models/completion-verification.md`: Uses multiple independent gates
- Principle Gate Over Remind: Requires gates, not reminders
- Decision bloat-control: Uses two-layer approach (surfacing + enforcement)

**RECOMMENDATION:** Option C (both) - Defense in depth with skill surfacing the requirement and orch complete enforcing it.

**Trade-off accepted:** More implementation work, but catches failure at two points.

---

### Fork 2: How to distinguish change types?

**Options:**
- A: Explicit flag (`--validation behavioral`)
- B: Heuristic detection (file paths, keywords, skill type)
- C: Always require behavioral evidence

**Substrate says:**
- Principle Session Amnesia: Future orchestrators need clear signals
- Finding 6: Trivial changes shouldn't require behavioral validation

**RECOMMENDATION:** Option B (heuristics) with Option A (explicit flag) as override - Use heuristics for auto-detection, allow explicit flag to override.

**Trade-off accepted:** Heuristics may have false positives/negatives, but explicit flag provides escape hatch.

---

### Fork 3: What evidence structure?

**Options:**
- A: Free-form text ("Behavior verified: ...")
- B: Structured format (command → observation pattern)
- C: Screenshot/recording requirement

**Substrate says:**
- Model completion-verification: Uses pattern matching for evidence detection
- Principle Verification Bottleneck: Requires human observation

**RECOMMENDATION:** Option B (structured format) - Pattern "Ran locally: [what I did] → [what I observed]" is parseable and meaningful.

**Trade-off accepted:** Agents can still game it with generic claims, but structured format is harder to fake than free-form.

---

## References

**Files Examined:**
- `pkg/verify/test_evidence.go` - Current test evidence gate implementation
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Current validation phase guidance
- `~/.kb/principles.md` - Verification Bottleneck principle
- `.kb/models/completion-verification.md` - Completion verification architecture
- `.kb/guides/completion-gates.md` - Gate reference documentation
- `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md` - Precedent for two-layer approach

**Commands Run:**
```bash
# Check beads issue for full context
bd show orch-go-sryez

# Review kb context for prior decisions
kb context "design validation gate"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification.md` - Architecture for completion gates
- **Guide:** `.kb/guides/completion-gates.md` - Gate reference
- **Investigation:** `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md` - Similar two-layer design

---

## Investigation History

**2026-01-26 21:35:** Investigation started
- Initial question: Where should validation gates live and what evidence structure is needed?
- Context: Jan 24 incident where "tests pass" was treated as sufficient for behavior-changing work

**2026-01-26 21:50:** Findings gathered
- Found current test_evidence.go only checks test execution
- Found feature-impl has validation phase but lacks behavioral requirement
- Found defense in depth is the established pattern

**2026-01-26 22:15:** Investigation completed
- Status: Complete
- Key outcome: Two-layer validation (skill + orch complete) with three-tier evidence taxonomy
