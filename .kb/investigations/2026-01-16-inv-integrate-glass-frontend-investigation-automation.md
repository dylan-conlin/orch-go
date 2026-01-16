<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Glass integration required documentation updates, not code changes - skill files now recommend Glass as primary browser automation tool with Playwright as fallback.

**Evidence:** Updated 3 skill source files (systematic-debugging, feature-impl, orchestrator), added 9 Glass tool patterns to pkg/verify/visual.go, deployed via skillc, committed to both repos.

**Knowledge:** Agents follow skill documentation literally - changing documentation from "USE Playwright" to "USE Glass MCP / FALLBACK Playwright" shifts agent behavior without infrastructure work.

**Next:** Close - integration complete, future agents will use Glass by default for frontend investigations.

**Promote to Decision:** recommend-no - This completes existing strategic direction (kn-3c7aaf), not a new architectural choice.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Integrate Glass Frontend Investigation Automation

**Question:** How can Glass become the default browser automation tool for frontend investigations, replacing Playwright as the recommended option?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Glass is production-ready but not integrated as default

**Evidence:** 
- Glass MCP server works with 5 tools: glass_page_state, glass_elements, glass_click, glass_type, glass_navigate
- Glass binary is symlinked at ~/bin/glass and functional
- Glass CLI has `glass assert` command for validation gates with exit codes
- Connects to Dylan's actual Chrome tabs via DevTools Protocol (not headless)

**Source:** 
- `.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md` - Complete Glass status
- `.kb/investigations/2026-01-06-inv-glass-browser-automation-not-working.md` - Symlink fix
- `.kb/investigations/2025-12-27-inv-add-cli-commands-glass-orchestrator.md` - CLI assert commands

**Significance:** Glass is ready for use but agents don't know to use it. Infrastructure exists but defaults to Playwright.

---

### Finding 2: Playwright is currently recommended in skill documentation

**Evidence:**
- systematic-debugging skill (line 197-204) recommends "Playwright MCP" for browser automation
- Lists decision flow: "Need browser automation? → Playwright MCP"
- No mention of Glass in skill guidance
- Agents follow skill documentation literally

**Source:** 
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md:197-204`

**Significance:** Agents will continue using Playwright unless skill documentation explicitly recommends Glass instead.

---

### Finding 3: Visual verification only detects Playwright tools

**Evidence:**
- pkg/verify/visual.go contains patterns: `playwright`, `browser_take_screenshot`, `browser_navigate`
- No `glass_*` patterns in detection
- Visual verification system is the gate that validates UI changes

**Source:**
- Glass investigation mentions: "pkg/verify/visual.go:79-104 - visualEvidencePatterns"
- "Currently only detects Playwright tools explicitly"

**Significance:** Even if agents use Glass, the verification system won't recognize it, causing completion failures.

---

### Finding 4: KB constraint exists for Glass-only dashboard interaction

**Evidence:**
- kn constraint kn-3c7aaf: "Dylan doesn't interact with dashboard directly - orchestrator uses Glass for all browser interactions"
- Reason: "Pressure Over Compensation: forces gaps in Glass tooling to surface"
- Strategic decision, not experimental

**Source:**
- `.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md` Finding 4

**Significance:** There's explicit architectural intent for Glass to be primary. This task aligns with existing strategic direction.

---

### Finding 5: orch spawn supports --mcp flag but Glass not configured

**Evidence:**
- `orch spawn --mcp playwright` flag exists
- Infrastructure for MCP integration is in place (pkg/spawn/config.go)
- Glass could be added as `--mcp glass` option
- No current shorthand for spawning with Glass

**Source:**
- Glass investigation Finding 2: "orch spawn has MCP support but not Glass-specific"
- cmd/orch/main.go:172,247,274 - spawnMCP flag

**Significance:** Adding Glass as spawn option would enable `orch spawn --mcp glass feature-impl "UI task"` pattern.

---

## Synthesis

**Key Insights:**

1. **Glass is ready but invisible to agents** - Glass infrastructure is production-ready (MCP server, CLI commands, Chrome integration), but agents don't know it exists because skill documentation recommends Playwright. This is a documentation/configuration gap, not a technical limitation.

2. **Multiple integration points need updating** - Making Glass default requires coordinated changes: (1) skill documentation to recommend Glass, (2) visual verification to detect glass_* tools, (3) possibly spawn configuration for automatic MCP setup, (4) guides/references for consistency.

3. **Aligns with existing strategic intent** - The kn constraint for Glass-only dashboard interaction shows this isn't a new direction - it's completing an existing migration. The system wants to use Glass but documentation hasn't caught up.

**Answer to Investigation Question:**

Glass can become the default for frontend investigations through three coordinated changes:

1. **Skill documentation** - Update systematic-debugging, investigation, and feature-impl skills to recommend Glass over Playwright for browser automation
2. **Visual verification** - Add glass_* tool patterns to pkg/verify/visual.go so verification gates recognize Glass usage
3. **Optional spawn config** - Consider adding `--mcp glass` shorthand to orch spawn for easier agent configuration

The core issue is discoverability: Glass works but agents follow Playwright recommendations in skill docs. Updating documentation to reflect the strategic intent (Glass-first) will shift agent behavior without infrastructure changes.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Documentation-first migration** - Update skill documentation and verification patterns to make Glass the recommended default, with Playwright as fallback for specific scenarios.

**Why this approach:**
- Agents follow skill documentation literally (Finding 2) - changing docs changes behavior
- Glass infrastructure already works (Finding 1) - no code changes needed for core functionality
- Verification system needs pattern updates (Finding 3) - must detect glass_* tools
- Aligns with strategic intent (Finding 4) - completing existing architectural direction

**Trade-offs accepted:**
- Playwright remains available for fallback scenarios (CI, headless testing)
- Requires Chrome to be running with remote debugging (Dylan's normal workflow)
- Glass is Dylan-specific workflow, not general-purpose like Playwright

**Implementation sequence:**
1. **Update visual verification first** - Add glass_* patterns to pkg/verify/visual.go so agents can complete after using Glass
2. **Update systematic-debugging skill** - Change browser automation recommendation from Playwright to Glass
3. **Update feature-impl skill** - Add Glass as recommended tool for UI visual verification
4. **Update orchestrator skill references** - Change spawning guidance to recommend --mcp glass for UI work
5. **Smoke test** - Verify changes work by checking that recommendations load correctly in skills

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
