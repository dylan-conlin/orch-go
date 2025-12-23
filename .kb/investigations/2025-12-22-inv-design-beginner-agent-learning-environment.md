# Design: Beginner-Friendly Agent Learning Environment

**Date:** 2025-12-22
**Status:** Complete
**Triggered by:** Setting up Lea (graphic designer) to use AI coding agents

## Context

Lea is a graphic designer who has been using ChatGPT to build Google Apps Script projects via copy-paste. She wants to learn to use proper coding agents (Claude Code, OpenCode, Cursor). The goal was to design an appropriate learning environment that sets her up for success without overwhelming her.

## Key Constraints

- No prior terminal/git experience
- Already experiencing session amnesia frustration ("ChatGPT doesn't remember decisions")
- Already discovered system prompt updates as a workaround
- Visual/design-oriented thinker
- Working with familiar domain data (SendCutSend products)

## Decisions Made

### 1. Editor: Cursor over VS Code + Claude Code

**Decision:** Start with Cursor, graduate to Claude Code/OpenCode later.

**Rationale:**
- Visual-first interface suits designer mindset
- AI built into editor (no separate terminal window)
- `Cmd+K` inline editing is intuitive
- Familiar VS Code-like UI for later transition
- Claude Code/OpenCode require terminal comfort she doesn't have yet

**Graduation path:** After 1-3 months, when terminal becomes comfortable.

### 2. Stack: SvelteKit 5 + shadcn-svelte + Tailwind + Supabase + Fly.io

**Decision:** Full modern stack with good agent support.

**Rationale:**
- SvelteKit: Less boilerplate than React, closer to HTML mental model
- shadcn-svelte: Copy-paste components (familiar pattern from her ChatGPT workflow)
- Tailwind: Inline styling, no context-switching
- Supabase: Spreadsheet-like table UI, familiar from Google Sheets
- Fly.io: CLI-driven deploys, agent-operable

### 3. Tools: Minimal initial set, add on pain

**Included now:**
- Homebrew, Bun, Git, gh, Fly, Go
- Cursor
- kn (because she's already feeling session amnesia pain)

**Explicitly deferred:**
| Tool | Why Deferred |
|------|--------------|
| kb-cli | Multi-agent orchestration, not needed for single-person projects |
| skillc | She won't be writing reusable agent procedures |
| playwright-mcp | Add when she needs UI testing |
| Handoff/synthesis templates | Solves orchestrator problems, not learner problems |

**Decision rule:** Add tools when she hits the problem they solve, not before.

### 4. kn: Include because she's already feeling the pain

**Decision:** Include kn despite "minimal tools" principle.

**Rationale:**
- She already complains about ChatGPT not remembering decisions
- She already discovered updating system prompts as a workaround
- She's independently reinventing what kn solves
- Low cognitive overhead (4 commands: decide, tried, question, context)

### 5. Documentation: Comprehensive CLAUDE.md

**Decision:** Single CLAUDE.md with full onboarding path.

**Sections included:**
1. Machine setup (brew, tools)
2. Cursor installation + usage
3. Quick start
4. kn for persisting decisions
5. Git workflow for beginners
6. Project-specific guidance

**Rationale:** Agents read CLAUDE.md, so comprehensive docs help both her AND the agents she works with.

## Risks Considered

### Risks of adding too many tools early:
1. **Cognitive overload** - Learning 5+ things simultaneously
2. **Premature abstraction** - Tools for problems she doesn't have
3. **Wrong habits** - Documenting out of obligation, not value
4. **Dependency on scaffolding** - Can't start projects without full toolkit
5. **Complexity blame** - "Coding is hard" vs "this specific thing broke"

### Mitigation:
- Start minimal, add on pain
- Each tool added when she expresses the specific frustration it solves
- kn is the exception because she already expressed the frustration

## Artifacts Produced

**Location:** `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/scs-explorer/`

| File | Purpose |
|------|---------|
| CLAUDE.md | Complete onboarding guide + agent instructions |
| README.md | Project overview + setup instructions |
| src/lib/scs-api.ts | Typed SCS API client |
| src/lib/supabase.ts | Supabase auth client |
| src/routes/* | Materials, hardware, finishes pages |
| Dockerfile + fly.toml | Deployment ready |
| .env.example | Required secrets |

## Follow-up Recommendations

1. **Pair session** - Walk through the setup together, let her drive
2. **First project idea** - "Add a favorites button to the materials page"
3. **Check-in after 2 weeks** - See if she's hitting any walls
4. **Add playwright-mcp** - When she wants to test UI or scrape data
5. **Graduate to Claude Code** - When she outgrows Cursor's capabilities

## Summary

Designed a beginner-friendly agent learning environment prioritizing:
- Familiar tools (Cursor looks like VS Code, Supabase looks like spreadsheets)
- Minimal initial complexity (add tools on pain, not preemptively)
- Exception for kn (she's already feeling session amnesia)
- Comprehensive CLAUDE.md (helps both her and her agents)
- Clear graduation path (Cursor → Claude Code when ready)
