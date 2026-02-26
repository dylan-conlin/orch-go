<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Designed a 3-tier UI validation gate: automatic detection (file patterns + skill type), evidence verification (Glass CLI or Playwright), and manual approval fallback.

**Evidence:** Existing visual.go has infrastructure for skill-aware detection and evidence patterns but gaps for Glass integration and automated verification.

**Knowledge:** UI verification needs both automated evidence (browser tools) AND human approval to prevent agents from self-certifying visual correctness.

**Next:** Implement the 3-tier system - add Glass CLI to visual patterns, create glass snap --verify command, integrate with orch complete --approve.

---

# Investigation: UI Validation Gate System Design

**Question:** How should orch complete verify UI work before allowing completion, and how does Glass CLI integrate?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Architect
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Problem Framing

### Design Question
Agent og-feat-fix-polling-05dec completed UI dashboard work without browser testing (only ran build/typecheck). The agent self-documented "not browser-tested" but no gate stopped completion. How do we design a UI validation gate that:
1. Automatically detects UI work
2. Requires browser verification evidence
3. Integrates with Glass CLI (once built)
4. Has a manual approval fallback

### Success Criteria
- UI changes cannot slip through without *either* automated verification *or* explicit human approval
- False positive rate is low (non-UI skills touching web/ files shouldn't be blocked)
- System integrates with existing `orch complete` and `pkg/verify/` infrastructure
- Works with both Glass CLI and Playwright MCP

### Constraints
- Must work with existing spawn/complete workflow
- Cannot require Glass CLI immediately (it's still being built, issue orch-go-l1is)
- Skill-aware detection already exists (don't break it)
- Prior decision kn-cc1c45: "MCP for agent-internal use, CLI for orchestrator/scripts/humans"

### Scope
**In scope:**
- Detection triggers (file paths, skill type)
- Verification evidence (what counts as valid proof)
- Integration points (pkg/verify/, orch complete)
- Glass CLI integration design

**Out of scope:**
- Glass MCP server implementation (separate issue)
- Playwright integration (already exists)
- UI testing framework selection

---

## Findings

### Finding 1: Existing Visual Verification Infrastructure

**Evidence:** pkg/verify/visual.go implements skill-aware visual verification with:
- `skillsRequiringVisualVerification` map (currently only feature-impl)
- `skillsExcludedFromVisualVerification` map (architect, investigation, systematic-debugging, etc.)
- `visualEvidencePatterns` regex list for screenshot/browser mentions
- `humanApprovalPatterns` regex list for explicit human approval
- `VerifyVisualVerification()` function called from `VerifyCompletionFull()`

The current flow at pkg/verify/visual.go:282-354:
1. Check if web/ files were modified via git diff
2. Extract skill name from SPAWN_CONTEXT.md
3. If skill not requiring visual verification → skip (no gate)
4. Check beads comments for evidence patterns
5. Check beads comments for human approval patterns
6. If evidence found BUT no human approval → block with NeedsApproval flag
7. If no evidence → block with error

**Source:** pkg/verify/visual.go:282-367, pkg/verify/check.go:439-449

**Significance:** The infrastructure exists but has gaps:
- Glass tools (`glass_*`) not in evidence patterns
- No automated verification (only pattern matching in comments)
- Manual approval flow exists (`--approve` flag) but requires evidence first

### Finding 2: Glass Integration Status

**Evidence:** From Glass investigation synthesis (.orch/workspace/og-inv-glass-integration-status-27dec/SYNTHESIS.md):
- Glass is production-ready MCP with 5 tools (snap, actions, actions-json, tabs, url, title)
- Binary at `/Users/dylanconlin/Documents/personal/glass/glass` works
- Chrome daemon connects via WebSocket at port 9222
- Visual verification patterns at pkg/verify/visual.go:79-104 detect playwright but NOT glass

Glass CLI commands (from glass --help):
- `glass snap` - Take screenshot
- `glass actions` - Parse page for actions
- `glass url` - Get current URL
- `glass title` - Get page title

**Source:** .orch/workspace/og-inv-glass-integration-status-27dec/SYNTHESIS.md, kn-cc1c45 decision

**Significance:** Glass CLI can provide `glass snap --verify` for automated screenshot capture that proves browser was opened. This is the missing piece for automated verification.

### Finding 3: The Self-Certification Problem

**Evidence:** Post-mortem og-inv-post-mortem-two-27dec states:
- Agent completed UI work without browser testing
- Only ran build/typecheck
- Self-documented "not browser-tested" 
- No gate stopped completion

Current visual.go flow requires evidence patterns in comments. Agents can simply not mention screenshots and pass (no gate triggers if no evidence AND no web changes detected, or if agent modifies only non-web files).

**Source:** SPAWN_CONTEXT.md context (post-mortem reference)

**Significance:** The current system is opt-in (agents must provide evidence) rather than opt-out (evidence is required). This is the root cause of the failure.

---

## Synthesis

**Key Insights:**

1. **Skill-aware detection works** - The existing infrastructure correctly identifies when feature-impl modifies web/ files and excludes architect/investigation from verification requirements.

2. **Evidence requirement is the gap** - Current system checks FOR evidence patterns but doesn't REQUIRE them. An agent can complete without mentioning screenshots at all.

3. **Two-phase verification needed** - First, detect UI work (already works). Second, require either automated evidence OR human approval (needs strengthening).

**Answer to Investigation Question:**

The UI validation gate should work as a 3-tier system:

**Tier 1: Automatic Detection** (exists, working)
- web/ file changes + feature-impl skill → triggers UI verification requirement

**Tier 2: Evidence Verification** (needs enhancement)
- Glass CLI: `glass snap --verify` outputs structured JSON with screenshot path
- Playwright: existing patterns for `browser_take_screenshot`, etc.
- Pattern matching in beads comments (current approach, but should be secondary)

**Tier 3: Manual Approval Fallback** (exists, needs connection)
- `orch complete --approve` adds approval marker
- For cases where Glass/Playwright can't be used

The key change: when Tier 1 triggers but Tier 2 fails to find evidence, **block completion** instead of just warning. Human approval (Tier 3) is the escape hatch, not the default.

---

## Structured Uncertainty

**What's tested:**

- Skill-aware detection correctly identifies feature-impl + web/ changes (verified: read pkg/verify/visual.go)
- Human approval patterns work with `--approve` flag (verified: read cmd/orch/main.go:3025-3035)
- Visual evidence patterns detect playwright mentions (verified: read pkg/verify/visual.go:79-104)

**What's untested:**

- Glass CLI `glass snap` output format (not benchmarked)
- Performance impact of adding Glass pattern matching (not profiled)
- False positive rate after adding Glass patterns (not measured)

**What would change this:**

- If Glass CLI structure changes before issue orch-go-l1is completes, the integration design may need adjustment
- If a better automated verification approach emerges (e.g., screenshot comparison), the evidence model could shift

---

## Implementation Recommendations

### Recommended Approach: 3-Tier UI Validation Gate

**Why this approach:**
- Builds on existing infrastructure rather than replacing it
- Separates concerns: detection, evidence, approval
- Glass CLI integration is additive (doesn't break existing Playwright support)
- Aligns with principle: "Skills own domain behavior, spawn owns orchestration infrastructure"

**Trade-offs accepted:**
- Agents will be blocked if they forget to run browser verification
- Orchestrators must use `--approve` for legitimate edge cases
- Glass CLI must be ready before full automation works

**Implementation sequence:**

1. **Add Glass patterns to visual.go** (immediate, no dependencies)
   - Add `glass_*` tool patterns to visualEvidencePatterns
   - Add `glass snap` CLI output pattern

2. **Create `glass snap --verify` command** (depends on orch-go-l1is)
   - Outputs structured JSON: `{"screenshot": "path", "url": "...", "timestamp": "..."}`
   - Agents can run this and include output in beads comment

3. **Strengthen the gate in verify.go** (after step 1)
   - Change from "warn if no evidence" to "block if no evidence AND no approval"
   - Make `--approve` the explicit escape hatch

### Alternative Approaches Considered

**Option B: Screenshot comparison service**
- **Pros:** Could detect visual regressions automatically
- **Cons:** Requires baseline screenshots, infrastructure complexity, maintenance burden
- **When to use instead:** When team has dedicated QA resources and visual regression is a recurring problem

**Option C: Required browser session before completion**
- **Pros:** Guarantees browser was opened
- **Cons:** Breaks CLI-only workflows, some UI changes don't need visual verification (CSS tweaks)
- **When to use instead:** For high-stakes production deployments with zero tolerance

**Rationale for recommendation:** Option A balances automation with pragmatism. It improves the current system incrementally without requiring new infrastructure or breaking existing workflows.

---

### Implementation Details

**What to implement first:**
1. Add Glass patterns to pkg/verify/visual.go (30 min, no dependencies)
2. Test pattern detection with mock beads comments (unit test)
3. Wait for Glass CLI (orch-go-l1is) before full integration

**File targets:**
- `pkg/verify/visual.go` - Add glass patterns to visualEvidencePatterns (lines 79-104)
- `pkg/verify/visual.go` - Add glass CLI output pattern (new pattern set)
- `pkg/verify/visual_test.go` - Add test cases for Glass patterns

**Pattern additions:**
```go
// Glass tool mentions
regexp.MustCompile(`(?i)glass_snap`),
regexp.MustCompile(`(?i)glass snap`),
regexp.MustCompile(`(?i)glass_screenshot`),
// Glass CLI structured output
regexp.MustCompile(`(?i)"screenshot":\s*".+\.png"`),
```

**Things to watch out for:**
- Glass CLI is still in progress (orch-go-l1is) - don't block on it
- Pattern matching can have false positives - test thoroughly
- Some web/ changes are legitimate without browser testing (e.g., package.json dependency updates)

**Areas needing further investigation:**
- How Glass CLI will output verification proof (JSON format TBD)
- Whether to exclude certain file patterns (package.json, config files)
- Performance of git diff check for large repos

**Success criteria:**
- Feature-impl agents with web/ changes are blocked unless evidence found
- Glass patterns match when agents use glass tools
- `orch complete --approve` works as escape hatch
- Non-UI skills (architect, investigation) not blocked

---

## References

**Files Examined:**
- pkg/verify/visual.go - Visual verification infrastructure
- pkg/verify/check.go - Main verification entry point (VerifyCompletionFull)
- pkg/verify/phase_gates.go - Phase gate pattern for reference
- pkg/verify/constraint.go - Constraint verification pattern for reference
- pkg/verify/skill_outputs.go - Skill name extraction (ExtractSkillNameFromSpawnContext)
- cmd/orch/main.go - Complete command with --approve flag
- .orch/workspace/og-inv-glass-integration-status-27dec/SYNTHESIS.md - Glass investigation findings

**Commands Run:**
```bash
# Report phase to beads
bd comment orch-go-o93n "Phase: Planning - Designing UI validation gate for orch complete"

# Create investigation artifact
kb create investigation design-ui-validation-gate-system

# Search for Glass issues
bd search glass
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md - Glass status
- **Issue:** orch-go-l1is - Add CLI commands to Glass for orchestrator use
- **Decision:** kn-cc1c45 - MCP for agent-internal use, CLI for orchestrator/scripts/humans

---

## Investigation History

**2025-12-27 [initial]:** Investigation started
- Initial question: How to design UI validation gate for orch complete
- Context: Post-mortem og-inv-post-mortem-two-27dec identified agent self-certification gap

**2025-12-27 [exploration]:** Analyzed existing infrastructure
- Found pkg/verify/visual.go has skill-aware detection
- Found patterns exist but don't include Glass tools
- Found --approve flag exists but needs connection to evidence requirement

**2025-12-27 [synthesis]:** Completed design
- Proposed 3-tier system: detection, evidence, approval
- Identified implementation sequence
- Documented file targets and success criteria
