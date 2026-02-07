TASK: Opus gate latest status - is it still active, any announcements about lifting, workarounds, community discussion

SPAWN TIER: full

📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.



## PRIOR KNOWLEDGE (from kb context)

**Query:** "opus"

### Constraints (MUST respect)
- orch-go DefaultModel should be Opus (claude-opus-4-5-20251101), not Gemini
  - Reason: Orchestrator guidance expects Opus for complex work, current Gemini default conflicts with operational practice
- orch-go DefaultModel should be Opus (claude-opus-4-5-20251101), not Gemini
  - Reason: Orchestrator guidance expects Opus for complex work, current Gemini default conflicts with operational practice
- Never spawn agents with Opus model until auth gate bypassed
  - Reason: Opus 4.5 requires Claude Code binary auth. Spawning with Opus via opencode creates zombie agents that never start but consume concurrency slots. Use Gemini Flash (default) or Sonnet instead.
- Opus 4.5 blocked via OAuth for opencode
  - Reason: Anthropic actively fingerprinting and updating enforcement. Cat and mouse game not worth chasing. Use Sonnet/Gemini.
- Do not use Opus 4.5 via OpenCode
  - Reason: Server-side fingerprinting gates it to official CLI
- Only two viable spawn paths: claude+opus or opencode+sonnet
  - Reason: Flash TPM exceeded, Opus blocked via API. Claude mode gives Max subscription Opus. OpenCode mode only works with Sonnet (pay-per-token).

### Prior Decisions
- Opus default, Gemini escape hatch
  - Reason: 2 Claude Max subscriptions (covered), Gemini is pay-per-token + tier 2 TPM. Escalation: 1) Opus default, 2) account switch, 3) --model flash
- Update opencode plugin to 0.0.7
  - Reason: Restores Opus 4.5 auth gate bypass via community fix
- Model restrictions are primary failure cause
  - Reason: Analysis of 19 FAILURE_REPORT.md and 4.8M daemon log failures shows Opus restrictions dominate
- Use claude+opus+tmux for critical infrastructure work
  - Reason: OpenCode server crashes kill agents. Claude CLI is independent, agents survive crashes. Tmux provides visibility. Worth Max quota for breaking death spiral. Applied to: orch doctor, Phase 2 dashboard, overmind supervision.
- Use opencode-anthropic-auth@0.0.7 to bypass Opus auth gate
  - Reason: Opus 4.5 performance worth the ban risk. Community plugin approach isolated to Anthropic requests, unlike failed manual header injection.
- Create model for Opus gate and spawn paths
  - Reason: System-wide ripple effects (escape hatch architecture, infrastructure detection, cost tradeoffs) warrant synthesized understanding. Makes constraints explicit for strategic decisions.
- Opus default, Gemini escape hatch
  - Reason: 2 Claude Max subscriptions (covered), Gemini is pay-per-token + tier 2 TPM. Escalation: 1) Opus default, 2) account switch, 3) --model flash
- Escape hatch for P0/P1 infrastructure work
  - Reason: Infrastructure work risks destabilizing the system used to fix it. OpenCode crashes kill headless agents. P0/P1 infrastructure = use --backend claude --opus for crash resistance (tmux survives crashes). Quality is secondary - reliability is primary.
- Abandon Claude Max OAuth, Use Gemini Flash as Primary Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md
- Dual Spawn Mode Architecture
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md
- Cancel Second Claude Max Subscription
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md
- Opus 4.5 fingerprint spoofing
- Direct Opus 4.5 auth gate spoofing via header injection
- Should orch spawn auto-select mode based on model? opus→claude, sonnet→opencode, flash→error

### Related Investigations
- Legacy Artifacts Synthesis Protocol Alignment
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-audit-legacy-artifacts-synthesis-protocol.md
- Synthesis Protocol Design for Agent Handoffs
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-design-synthesis-protocol-schema.md
- Enhance orch review to parse and display SYNTHESIS.md
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-enhance-orch-review-parse-display.md
- Synthesis Protocol Implementation Verification
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-implement-synthesis-protocol-create-orch.md
- Port model flexibility and arbitrage to orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-investigate-model-flexibility-arbitrage-orch.md
- Research: Claude 4.5 and Claude Max Pricing (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-research-claude-claude-max-pricing.md
- Scope Out Headless Swarm Implementation
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md
- Model Arbitrage and API vs Max Math (Late 2025)
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-20-research-model-arbitrage-api-vs-max.md
- Deep Post-Mortem on 24 Hours of Development Chaos
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-deep-post-mortem-last-24.md
- Model Handling Conflicts Between orch-go and opencode
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md
- Workspace Lifecycle in orch-go
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md
- Model Flexibility Phase Expand Model
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-model-flexibility-phase-expand-model.md
- Analyze Nate Jones Article Llm
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-analyze-nate-jones-article-llm.md
- Model Selection Issue Architect Agent
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md
- Add Api Usage Endpoint Serve
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-add-api-usage-endpoint-serve.md
- Fix NaNm Runtime Display Agent
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-fix-nanm-runtime-display-agent.md
- Model Provider Architecture - orch vs OpenCode Auth Responsibility
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-24-inv-model-provider-architecture-orch-vs.md
- Evaluate Building API Proxy Layer for Claude Max Account Sharing
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-evaluate-building-api-proxy-layer.md
- Detect Agents Exhausting Context with Uncommitted Work
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-03-inv-detect-agents-exhausting-context-uncommitted.md
- Synthesize Model Investigations Into Guide
  - See: /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-06-inv-synthesize-model-investigations-10-synthesis.md

**IMPORTANT:** The above context represents existing knowledge and decisions. Do not contradict constraints. Reference investigations for prior findings.





📋 AD-HOC SPAWN (--no-track):
This is an ad-hoc spawn without beads issue tracking.
Progress tracking via bd comment is NOT available.

🚨 SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE typing anything else:

1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `/exit` to close the agent session



CONTEXT: [See task description]

PROJECT_DIR: /Users/dylanconlin/Documents/personal/orch-go

SESSION SCOPE: Medium (estimated [1-2h / 2-4h / 4-6h+])
- Default estimation
- Recommend checkpoint after Phase 1 if session exceeds 2 hours


AUTHORITY:
**You have authority to decide:**
- Implementation details (how to structure code, naming, file organization)
- Testing strategies (which tests to write, test frameworks to use)
- Refactoring within scope (improving code quality without changing behavior)
- Tool/library selection within established patterns (using tools already in project)
- Documentation structure and wording

**You must escalate to orchestrator when:**
- Architectural decisions needed (changing system structure, adding new patterns)
- Scope boundaries unclear (unsure if something is IN vs OUT scope)
- Requirements ambiguous (multiple valid interpretations exist)
- Blocked by external dependencies (missing access, broken tools, unclear context)
- Major trade-offs discovered (performance vs maintainability, security vs usability)
- Task estimation significantly wrong (2h task is actually 8h)

**When uncertain:** Err on side of escalation. Document question in workspace, set Status: QUESTION, and wait for orchestrator response. Better to ask than guess wrong.

**Full criteria:** See `.kb/guides/decision-authority.md` for the complete decision tree and examples.

**Surface Before Circumvent:**
Before working around ANY constraint (technical, architectural, or process):
1. Document it in your investigation file: "CONSTRAINT: [what constraint] - [why considering workaround]"
2. Include the constraint and your reasoning in SYNTHESIS.md
3. The accountability is a feature, not a cost

This applies to:
- System constraints discovered during work (e.g., API limits, tool limitations)
- Architectural patterns that seem inconvenient for your task
- Process requirements that feel like overhead
- Prior decisions (from `kb context`) that conflict with your approach

**Why:** Working around constraints without surfacing them:
- Prevents the system from learning about recurring friction
- Bypasses stakeholders who should know about the limitation
- Creates hidden technical debt

DELIVERABLES (REQUIRED):
1. **FIRST:** Verify project location: pwd (must be /Users/dylanconlin/Documents/personal/orch-go)
2. **SET UP investigation file:** Run `kb create investigation opus-gate-latest-status-still` to create from template
   - This creates: `.kb/investigations/simple/YYYY-MM-DD-opus-gate-latest-status-still.md`
   - This file is your coordination artifact (replaces WORKSPACE.md)
   - If command fails, report to orchestrator immediately

3. **UPDATE investigation file** as you work:
   - Add TLDR at top (1-2 sentence summary of question and finding)
   - Fill sections: What I tried → What I observed → Test performed
   - Only fill Conclusion if you actually tested (this is the key discipline)
4. Update Status: field when done (Active → Complete)
5. [Task-specific deliverables]

6. **CREATE SYNTHESIS.md:** Before completing, create `SYNTHESIS.md` in your workspace: /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-research-opus-gate-latest-13jan-5a4c/SYNTHESIS.md
   - Use the template from: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md
   - This is CRITICAL for the orchestrator to review your work.


STATUS UPDATES:
Update Status: field in your investigation file:
- Status: Active (while working)
- Status: Complete (when done and committed) → then call /exit to close agent session

Signal orchestrator when blocked:
- Add '**Status:** BLOCKED - [reason]' to investigation file
- Add '**Status:** QUESTION - [question]' when needing input



## SKILL GUIDANCE (research)

**IMPORTANT:** You have been spawned WITH this skill context already loaded.
You do NOT need to invoke this skill using the Skill tool.
Simply follow the guidance provided below.

---

---
name: research
skill-type: procedure
description: Web-based research producing structured recommendations with uncertainty assessment. Use for technology comparisons, best practices research, and option evaluation. Distinct from investigation skill (codebase exploration) - this is for external sources (docs, articles, tutorials).
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: 2f1c3352c71a -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/research/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/research/SKILL.md -->
<!-- To modify: edit files in /Users/dylanconlin/orch-knowledge/skills/src/worker/research/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-10 00:50:38 -->

## Summary

**Use when the user says:**
- "Compare [options] for [use case]" - Technology/library/approach comparison
- "Which approach for [problem]?" - Evaluating multiple solutions
- "Evaluate [technology/library/pattern]" - Assessing a specific option
- "Research best practices for [topic]" - Finding recommended approaches
- "What are the options for [decision]?" - Identifying and comparing alternatives

---

# Research Skill

## When to Use This Skill

**Use when the user says:**
- "Compare [options] for [use case]" - Technology/library/approach comparison
- "Which approach for [problem]?" - Evaluating multiple solutions
- "Evaluate [technology/library/pattern]" - Assessing a specific option
- "Research best practices for [topic]" - Finding recommended approaches
- "What are the options for [decision]?" - Identifying and comparing alternatives

**Use research skill when:**
- Need to compare external options (technologies, libraries, patterns)
- Research findings should be durable (referenced in future decisions)
- Question requires web research (docs, articles, comparisons)
- Will inform architectural or technology decisions
- Need structured recommendation with uncertainty assessment

**Don't use research skill for:**
- Codebase exploration (use `investigation` skill instead)
- Debugging/fixing bugs (use `systematic-debugging` instead)
- Quick questions with obvious answers (direct response)
- Implementation tasks (use `implement-feature` instead)
- One-off questions unlikely to need future reference

## Key Distinction: Research vs Other Skills

| Skill | When to Use | Example | Output |
|-------|-------------|---------|--------|
| **research** | Compare external options | "Compare static site generators (Gatsby, Next.js, Astro)" | Research file with options + recommendation + structured uncertainty |
| **investigation** | Understand internal system | "Investigate auth flow before adding 2FA" | Investigation file with findings + synthesis |
| **systematic-debugging** | Find root cause of bug | "Auth is broken, investigate why" | Investigation file with hypothesis + fix |
| **architect -i** | Explore design space | "Design approaches for rate limiting" | Investigation with recommendations (interactive collaboration) |

**Key insight:**
- **Research** = External sources (docs, articles) → Recommendation with uncertainty assessment
- **Investigation** = Internal sources (code, logs) → Understanding how system works
- **Systematic-debugging** = Problem-driven analysis → Root cause + fix
- **Architect -i** = Design exploration → Options with trade-off analysis (interactive mode)

## Workflow

### 1. Create Research Template Immediately

**Before starting research**, create research file from template:

```bash
# Create investigation using kb CLI command
# Update SLUG based on your research topic
# Use research/ prefix for research investigations
kb create investigation "research/topic-in-kebab-case"
```

**Important:**
- The `kb create investigation` command auto-detects project directory and creates the investigation in the appropriate subdirectory.
- The investigation file includes Resolution-Status field (Unresolved/Resolved/Recurring/Synthesized/Mitigated) which is critical for the synthesis workflow. Always fill this field when completing the research.

**Critical:** Create template at START, not at end. This forces you to document progressively.

### 2. Fill Question and Metadata

Edit the research file:

```markdown
# Research: [Specific topic, e.g., "Static Site Generators for Documentation"]

**Question:** [Precise research question, e.g., "Which static site generator should we use for technical documentation with versioning?"]

**Started:** 2025-11-09
**Updated:** 2025-11-09
**Status:** In Progress
```

### 3. Evaluate Options Progressively (As You Research)

**After researching each option**, add an option section:

```markdown
### Option 1: Gatsby

**Overview:** React-based static site generator with rich plugin ecosystem and GraphQL data layer.

**Pros:**
- Rich plugin ecosystem (1000+ plugins)
- Excellent React integration
- GraphQL for data querying
- Strong community support

**Cons:**
- Complex build process
- Slower build times for large sites
- GraphQL learning curve for simple use cases
- Heavier runtime bundle size

**Evidence:**
- Official docs: https://www.gatsbyjs.com/docs/
- Build time benchmarks: ~45s for 100 pages (Source: benchmarks.example.com)
- Bundle size: ~200KB min+gzip for basic site
- Active development: 50K+ GitHub stars, recent commits
```

**Pattern:** Overview → Pros → Cons → Evidence (with sources)

**Don't wait to write everything at the end.** Add options as you research them.

### 4. Update Recommendation After Evaluating Options

**Once you've evaluated all options**, write your recommendation:

```markdown
## Recommendation

**I recommend Astro** because it provides the best balance of simplicity and performance for documentation sites. Key factors:

1. **Performance:** Zero JavaScript by default (fastest page loads)
2. **Flexibility:** Support for React, Vue, or plain HTML (future-proof)
3. **Simplicity:** No GraphQL layer, straightforward file-based routing
4. **Versioning:** Built-in support via content collections

**Trade-offs I'm accepting:**
- Smaller plugin ecosystem than Gatsby (but sufficient for docs)
- Less mature than Next.js (but stable enough for production)

**When this recommendation might change:**
- If you need a rich plugin ecosystem → Gatsby
- If you need full React SSR/ISR → Next.js
- If you need maximum simplicity → Plain HTML/CSS
```

**Clear recommendation > hedge your bets.** State what you're recommending and why.

### 5. Update Structured Uncertainty

**After completing research**, assess uncertainty using structured sections:

```markdown
## Structured Uncertainty

**What's tested:**
- Astro has zero JavaScript by default (verified in official docs)
- Build performance is significantly faster than Gatsby (benchmarked: 10s vs 45s for 100 pages)
- All three options are production-ready (used by major companies)
- Astro supports versioning via content collections (tested in demo)

**What's untested:**
- Long-term maintenance commitment (Astro is newer than alternatives)
- Plugin ecosystem growth rate (current ecosystem sufficient but smaller)
- Migration difficulty if we need to switch later (haven't tested)

**What would change this:**
- Major security vulnerability in recommended option
- Astro maintenance stops or slows significantly
- Team has strong React preference (would favor Next.js)
- Versioning implementation proves complex in practice
```

**Honest uncertainty > false certainty.** State what you haven't tested.

### 6. Mark Complete and Commit

**When research is done:**

1. Update status:
   ```markdown
   **Status:** Complete
   **Updated:** 2025-11-09
   ```

2. Add completion entry to Research History:
   ```markdown
   **2025-11-09:** Research completed
   - Status: Complete
   - Options evaluated: 3 (Gatsby, Next.js, Astro)
   - Recommendation: Astro for documentation sites
   ```

3. Commit to project repository (do NOT push - orchestrator handles remote operations):
   ```bash
   cd ${PROJECT}
   git add .kb/investigations/${DATE}-research-${SLUG}.md
   git commit -m "research: ${SLUG}"
   # Do NOT git push - orchestrator decides when to push/deploy
   ```

4. Call /exit to close agent session

### 7. Link from Decision (If Applicable)

If making a decision based on this research:

**From decision document:**
```markdown
## Research

This decision is based on research documented in:
- {project}/.kb/investigations/2025-11-09-research-static-site-generators.md
```

**Report completion via beads:**

## Key Principles

| Principle | Application |
|-----------|-------------|
| **Progressive documentation** | Create template first, fill options as you research (not at end) |
| **Evidence-based** | Every option needs concrete evidence (docs, benchmarks, examples) |
| **Honest uncertainty** | State what you haven't tested - gaps are valuable information |
| **Durable artifacts** | Commit to project .orch/ for future reference |
| **Clear sourcing** | Always include URLs and specific sources for evidence |
| **Synthesis over list** | Connect options into clear recommendation, don't just list facts |

## Common Mistakes to Avoid

**Don't:**
- Research everything then write artifact at end (loses context)
- Skip evidence/sources for claims (unverifiable assertions)
- Claim certainty when gaps exist (false certainty)
- Create research for trivial questions (quick Google search = direct response)
- Forget to commit to .orch/ (research lost if not durable)
- Recommend "it depends" without stating your recommendation (hedge your bets)

**Do:**
- Create template immediately before researching
- Add options progressively (after researching each)
- Include specific evidence with URLs/sources
- Be explicit about gaps in understanding (what's uncertain)
- Make clear recommendation even if there are trade-offs
- Link research from workspace/decisions/knowledge files

## Integration with Other Skills

**Before decisions:**
```
User: "Should we use Gatsby or Next.js for our docs?"
You: "Let me research both options and compare them."
[Use research skill]
[Create research file, evaluate options progressively, commit findings]
[Research informs record-decision skill]
```

**Comparing with investigation:**
```
User: "Investigate our authentication system"
You: "I'll use investigation skill to explore the codebase."
[NOT research skill - this is internal exploration]

User: "Compare OAuth providers (Auth0, Okta, Firebase)"
You: "I'll use research skill to evaluate these options."
[NOT investigation skill - this is external research]
```

**Building knowledge base:**
```
[After research is complete]
You: "Research complete. This is foundational knowledge."
[Use capture-knowledge skill to externalize understanding from research]
```

## Examples

### Good Example: Technology Comparison

```markdown
# Research: Static Site Generators for Documentation

**Question:** Which static site generator should we use for technical documentation with versioning?

**Started:** 2025-11-09
**Updated:** 2025-11-09
**Status:** Complete

## Question

We need to choose a static site generator for our technical documentation. Requirements:
- Fast page loads (performance critical)
- Version support (docs for multiple product versions)
- Developer-friendly (team knows React)
- SEO optimized
- Active maintenance

## Options Evaluated

### Option 1: Gatsby
[Full option details with pros, cons, evidence]

### Option 2: Next.js
[Full option details with pros, cons, evidence]

### Option 3: Astro
[Full option details with pros, cons, evidence]

## Recommendation

**I recommend Astro** because it provides the best balance of simplicity and performance...
[Clear recommendation with reasoning]

## Structured Uncertainty

**What's tested:**
- Performance benchmarks verified
- All options production-ready

**What's untested:**
- Long-term maintenance
- Migration difficulty

**What would change this:**
- Major issues in production scenario
- Team preference for different framework

## Research History

**2025-11-09:** Research started
- Question defined
- Evaluated 3 options

**2025-11-09:** Research completed
- Recommendation: Astro
```

### Bad Example: Missing Structured Uncertainty

```markdown
# Research: Static Site Generators

I looked at Gatsby, Next.js, and Astro.

Gatsby is good but slow. Next.js is popular. Astro is new but fast.

I think Astro is probably best but it depends on your needs.
```

**Why bad:**
- No structured uncertainty (what's tested vs untested)
- Vague evidence ("good", "slow", "popular" - no sources)
- No clear recommendation ("probably best but it depends")
- No structured options evaluation
- Not durable or referenceable

## Template Location

Template: `~/.claude/skills/worker/research/templates/research.md`

Use this template for all research to maintain consistency.

## Self-Review (Mandatory)

**Before completing, verify research quality.**

### Research-Specific Checks

| Check | Verification | If Failed |
|-------|--------------|-----------|
| **Evidence sourced** | Each claim has URL/source | Add sources |
| **Options compared** | 2+ options with pros/cons | Add comparison |
| **Recommendation clear** | Not "it depends" | Make specific recommendation |
| **Uncertainty assessed** | What's tested/untested documented | Add assessment |
| **No speculation** | Claims backed by evidence | Remove or verify |

### Self-Review Checklist

#### 1. Evidence Quality

- [ ] **Each option has evidence** - URLs, benchmarks, docs cited
- [ ] **Sources are authoritative** - Official docs, not random blogs
- [ ] **Claims are verifiable** - Someone could check your sources
- [ ] **No unsourced assertions** - Every claim backed by evidence

#### 2. Recommendation Quality

- [ ] **Clear recommendation** - Not "it depends" or "either works"
- [ ] **Reasoning explained** - Why this option over others
- [ ] **Trade-offs acknowledged** - What you're sacrificing
- [ ] **When recommendation changes** - Conditions that would change answer

#### 3. Structured Uncertainty

- [ ] **What's tested** - Facts verified with evidence
- [ ] **What's untested** - Gaps acknowledged honestly
- [ ] **What would change this** - Conditions that would invalidate recommendation
- [ ] **No false certainty** - Don't claim certainty without testing

#### 4. Documentation Quality

- [ ] **Question stated clearly** - Precise, answerable
- [ ] **Options documented progressively** - Not all at end
- [ ] **Research file complete** - All sections filled
- [ ] **Committed to repository** - In `.kb/investigations/` (with `research-` prefix)

### Document in Research File

At the end of your research file, add:

```markdown
## Self-Review

- [ ] Each option has evidence with sources
- [ ] Clear recommendation (not "it depends")
- [ ] Structured uncertainty documented
- [ ] Research file complete and committed

**Self-Review Status:** PASSED / FAILED

---




CONTEXT AVAILABLE:
- Global: ~/.claude/CLAUDE.md
- Project: /Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md

🚨 FINAL STEP - SESSION COMPLETE PROTOCOL:
After your final commit, BEFORE doing anything else:


1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. `/exit`


⚠️ Your work is NOT complete until you run these commands.
