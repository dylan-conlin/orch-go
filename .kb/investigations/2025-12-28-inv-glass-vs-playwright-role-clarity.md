<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Glass and Playwright serve distinct roles: Glass for collaborative/shared browser (human + agent see same Chrome), Playwright for headless verification and E2E testing. Skills should NOT migrate to Glass - both are needed for different use cases.

**Evidence:** Glass connects to Dylan's Chrome via CDP on port 9222 (shared browser session); Playwright spawns isolated browser instances for headless testing. Constraint kn-3c7aaf requires Glass for dashboard interactions. Playwright MCP still needed for workers doing E2E tests and visual regression.

**Knowledge:** The confusion stems from overlapping tool names (`glass_*` vs `browser_*`) but different architectural purposes. Glass is for "see what the agent sees" collaborative UX; Playwright is for "verify without human watching" automation.

**Next:** Document clear boundaries in CLAUDE.md, update skills to clarify when to use each, keep both MCP servers configured.

---

# Investigation: Glass vs Playwright Role Clarity

**Question:** When should Glass vs Playwright be used in the orch ecosystem? Should skills migrate from Playwright to Glass?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - decision ready for implementation
**Status:** Complete

---

## Problem Framing

### Design Question
Glass was introduced for collaborative browser work (human + orchestrator see same browser). Playwright MCP is used for worker agent verification. Current confusion:
1. When should each be used?
2. Glass MCP not working in this session while Playwright loaded
3. Skills reference Playwright for verification but Glass exists now

### Success Criteria
- Clear boundaries for when to use Glass vs Playwright
- Understanding of whether skills should migrate to Glass
- Decision on MCP configuration (both enabled, one preferred, etc.)
- Guidance that future agents can follow without confusion

### Constraints
- Existing constraint kn-3c7aaf: "Dylan doesn't interact with dashboard directly - orchestrator uses Glass for all browser interactions"
- Prior decision kn-cc1c45: "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- Glass CLI exists for orchestrator validation (assert command)
- Playwright used in existing E2E tests (web/tests/*.spec.ts)

### Scope
**In scope:**
- Role clarity between Glass and Playwright
- When each tool is appropriate
- MCP configuration recommendations
- Skill guidance

**Out of scope:**
- Detailed Playwright test migration
- Glass feature development (separate issues)
- Performance benchmarking

---

## Findings

### Finding 1: Glass is for Collaborative/Shared Browser UX

**Evidence:**
- Glass connects to existing Chrome via CDP on port 9222 (`--remote-debugging-port=9222`)
- Requires Chrome to be launched with debugging enabled (primary profile or isolated)
- Agent and human see the SAME browser window and tabs
- MCP config: `/Users/dylanconlin/bin/glass mcp` (always available to orchestrator)
- Tools: glass_tabs, glass_page_state, glass_elements, glass_click, glass_type, glass_navigate, glass_focus, glass_enable_user_tracking, glass_recent_actions
- CLI commands: snap, screenshot, assert (for orchestrator validation)

**Source:**
- Prior investigation: .kb/investigations/2025-12-27-inv-dogfood-glass-browser-automation-effectively.md
- Prior investigation: .kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md
- OpenCode config: ~/.config/opencode/opencode.jsonc lines 36-40

**Significance:** Glass is purpose-built for Dylan's workflow where the orchestrator operates within Dylan's actual browser session. This enables:
- "See what the agent sees" collaborative UX
- Dashboard interaction without Dylan touching the mouse
- Real browser context with extensions, login sessions, bookmarks

---

### Finding 2: Playwright is for Headless/Isolated Verification

**Evidence:**
- Playwright spawns isolated browser instances (not Dylan's Chrome)
- MCP config uses npx: `["npx", "@playwright/mcp@latest", "--viewport-size=1440x900"]`
- Tools: browser_navigate, browser_take_screenshot, browser_click, browser_type, etc.
- Used in existing E2E tests: web/tests/*.spec.ts (6 test files)
- Designed for CI/CD and automated testing
- Does NOT require Chrome to be manually launched

**Source:**
- OpenCode config: ~/.config/opencode/opencode.jsonc lines 41-47
- Test files: web/tests/mode-toggle.spec.ts, agent-detail.spec.ts, filtering.spec.ts, etc.
- Playwright config: web/playwright.config.ts

**Significance:** Playwright is for workers doing verification where:
- Human doesn't need to see the browser
- Tests need isolation (fresh state each run)
- CI/CD pipelines need headless execution
- Visual regression testing needs consistent viewport

---

### Finding 3: The Tools Are Complementary, Not Competing

**Evidence:**
From existing orch spawn documentation (cmd/orch/main.go:256-257):
```
orch-go spawn --mcp playwright feature-impl "add UI feature" # With Playwright MCP (full browser)
orch-go spawn --mcp glass feature-impl "verify dashboard"    # With Glass MCP (shared Chrome)
```

Visual verification patterns in pkg/verify/visual.go detect BOTH:
- Lines 93-95: Playwright patterns (`playwright`, `browser_take_screenshot`, `browser_navigate`)
- Lines 97-98: Glass patterns (`glass_*`, `glass screenshot|navigate|click|type`)

**Source:**
- pkg/verify/visual.go:79-107
- cmd/orch/main.go:256-257, 285

**Significance:** The system was designed to support BOTH tools. The `--mcp` flag lets you choose which browser automation approach to use for a given spawn. Skills don't need to migrate - they need clarity on when to use which.

---

### Finding 4: Current Confusion Sources

**Evidence:**
From SPAWN_CONTEXT.md task description:
1. "Glass MCP not working in this session while Playwright loaded" - This was a binary corruption issue (resolved, see .kb/investigations/2025-12-27-inv-orchestrator-see-playwright-browser-tools.md)
2. "Skills reference Playwright for verification" - True, but skills don't need to change
3. "Should skills migrate to glass" - No, see Finding 3

The Vibium research (.kb/investigations/2025-12-27-research-compare-vivium-selenium-glass-automation.md) added confusion by stating "Glass as described doesn't exist" - this was referring to useglass.ai (a different product), not Dylan's Glass project.

**Source:**
- Task SPAWN_CONTEXT.md
- .kb/investigations/2025-12-27-inv-orchestrator-see-playwright-browser-tools.md
- .kb/investigations/2025-12-27-research-compare-vivium-selenium-glass-automation.md

**Significance:** The confusion is documentation/clarity issue, not an architectural problem. Both tools work and have distinct purposes.

---

## Synthesis

**Key Insights:**

1. **Different Browser Models** - Glass shares Dylan's actual Chrome; Playwright spawns isolated instances. This is the fundamental distinction that determines when to use each.

2. **Complementary Use Cases** - Glass for collaborative work (orchestrator helping Dylan), Playwright for automated verification (workers testing in isolation). Neither replaces the other.

3. **MCP Configuration is Correct** - Both should remain enabled. Glass for orchestrator, Playwright for workers with `--mcp playwright` flag.

**Answer to Investigation Question:**

| Dimension | Glass | Playwright |
|-----------|-------|------------|
| **Browser** | Dylan's Chrome (shared) | Isolated instance |
| **Launch** | Requires `--remote-debugging-port=9222` | Auto-spawned |
| **Primary User** | Orchestrator | Worker agents |
| **Use Case** | Dashboard interaction, collaborative browsing | E2E tests, visual verification |
| **When** | "I want to see what happens" | "Verify this works correctly" |
| **MCP Flag** | Default (always available) | `--mcp playwright` |
| **CLI** | glass assert (for validation gates) | npx playwright test |

**Should skills migrate to Glass?** No. Skills should clarify WHEN to use each tool:
- Use Glass when orchestrator needs to interact with Dylan's browser session
- Use Playwright when worker needs to verify UI in isolation
- Both count as valid visual verification evidence

---

## Structured Uncertainty

**What's tested:**

- ✅ Glass binary works and exposes 9 MCP tools (verified: ran `glass --help`)
- ✅ Playwright MCP spawns isolated browser (verified: existing E2E tests work)
- ✅ Visual verification patterns detect both tools (verified: read pkg/verify/visual.go)
- ✅ Both MCPs can be configured simultaneously (verified: opencode.jsonc has both)

**What's untested:**

- ⚠️ Whether workers effectively use `--mcp playwright` flag (no recent spawn logs examined)
- ⚠️ Performance comparison between Glass and Playwright for same tasks (not benchmarked)
- ⚠️ Edge cases where both tools are needed in same session

**What would change this:**

- If Glass gains headless mode → could replace Playwright for some use cases
- If Playwright MCP adds shared-browser support → distinction becomes less clear
- If workers frequently need to see Dylan's actual browser state → Glass for workers too

---

## Implementation Recommendations

### Recommended Approach: Document Boundaries, Keep Both ⭐

**Why this approach:**
- Both tools serve valid, distinct purposes
- No code changes needed - only documentation clarity
- Aligns with existing architecture (--mcp flag, visual verification patterns)
- Follows principle of "Evolve by distinction" - the tools have evolved to serve different needs

**Trade-offs accepted:**
- Two browser automation tools to maintain (but they're different enough to justify)
- Potential confusion for new agents (mitigated by clear documentation)
- Glass requires Chrome to be running (operational requirement)

**Implementation sequence:**
1. Update orchestrator CLAUDE.md with Glass vs Playwright guidance
2. Add brief comment to skills that mention browser verification
3. Document in orch spawn help which MCP to use when
4. Keep both MCP servers enabled in opencode.jsonc

### Alternative Approaches Considered

**Option B: Deprecate Playwright, Glass for everything**
- **Pros:** Single tool, simpler mental model
- **Cons:** Glass requires manual Chrome launch; can't run in CI/CD; loses test isolation
- **When to use instead:** If Dylan's workflow never needs isolated testing

**Option C: Deprecate Glass, Playwright for everything**
- **Pros:** More mature tool, better documentation
- **Cons:** Loses "see what agent sees" collaborative UX; violates constraint kn-3c7aaf
- **When to use instead:** If collaborative browsing becomes unnecessary

**Rationale for recommendation:** The tools serve genuinely different needs. Collapsing to one would either lose collaborative UX (killing Glass) or lose CI/CD capability (killing Playwright).

---

### Implementation Details

**Documentation updates needed:**

1. **CLAUDE.md (global)** - Add section:
```markdown
## Browser Automation
- **Glass:** Use for collaborative browser work where Dylan sees what you're doing.
  Connects to Dylan's Chrome on port 9222. Default for orchestrator.
- **Playwright:** Use for headless verification. Spawns isolated browser.
  Add `--mcp playwright` when spawning workers that need browser testing.
```

2. **Skill guidance** - Feature-impl skill should mention:
```markdown
## Visual Verification
Use Glass (glass_*) or Playwright (browser_*) tools to verify UI changes.
- Glass: When orchestrator needs to interact with shared browser
- Playwright: When worker needs isolated browser for E2E tests
```

**Things to watch out for:**
- ⚠️ Glass requires Chrome launched with `--remote-debugging-port=9222`
- ⚠️ Playwright MCP requires npx in PATH (or use absolute path)
- ⚠️ Don't spawn with both --mcp flags - choose one per spawn

**Success criteria:**
- ✅ Future agents know which tool to use without confusion
- ✅ Visual verification gate accepts both tool types
- ✅ Orchestrator uses Glass for dashboard interactions
- ✅ Workers use Playwright for E2E tests

---

## References

**Files Examined:**
- pkg/verify/visual.go - Visual verification patterns
- cmd/orch/main.go - Spawn command with --mcp flag
- ~/.config/opencode/opencode.jsonc - MCP configuration
- web/tests/*.spec.ts - Existing Playwright tests

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md
- **Investigation:** .kb/investigations/2025-12-27-inv-dogfood-glass-browser-automation-effectively.md
- **Investigation:** .kb/investigations/2025-12-27-inv-orchestrator-see-playwright-browser-tools.md
- **Investigation:** .kb/investigations/2025-12-27-inv-design-ui-validation-gate-system.md
- **Constraint:** kn-3c7aaf - Glass for dashboard interactions

---

## Self-Review

- [x] Problem framing with success criteria defined
- [x] 2+ approaches compared (Glass vs Playwright)
- [x] Clear recommendation with reasoning
- [x] Trade-offs acknowledged
- [x] Principle cited (Evolve by distinction)
- [x] Investigation file complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-28:** Investigation started
- Initial question: Glass vs Playwright role clarity
- Context: Confusion about when to use each tool, whether skills should migrate

**2025-12-28:** Exploration phase
- Read prior Glass investigations (3 related)
- Examined visual verification patterns
- Reviewed MCP configurations
- Understood existing --mcp spawn flag

**2025-12-28:** Synthesis completed
- Recommendation: Keep both, document boundaries
- Key insight: Different browser models (shared vs isolated)
- No skill migration needed - clarify usage, don't change tools
