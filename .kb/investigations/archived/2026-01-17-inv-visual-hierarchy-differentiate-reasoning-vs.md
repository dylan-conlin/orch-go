<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Visual Hierarchy Differentiate Reasoning Vs

**Question:** How to apply visual hierarchy to differentiate reasoning text, tool calls, and results in the activity feed?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Worker Agent
**Phase:** Implementing
**Next Step:** Apply typography and spacing changes to activity-tab.svelte
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Activity feed currently has minimal visual differentiation

**Evidence:** All event types (text, tool, reasoning, steps) use similar styling with only emoji icons differentiating them. Tool calls have blue color for tool name, but reasoning and text messages appear identical.

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte:565-573 (non-tool events), 513-563 (tool events)

**Significance:** Users cannot quickly scan the feed to distinguish between agent reasoning (thinking process) and tool calls (actions taken), reducing the feed's utility for monitoring.

---

### Finding 2: Design system provides four-level contrast hierarchy

**Evidence:** CSS variables define: --foreground (primary), --secondary-foreground (secondary), --muted-foreground (muted), and opacity can create faint level. Current activity feed uses muted-foreground for all event text.

**Source:** web/src/app.css:19 (muted-foreground), activity-tab.svelte:567 (text-muted-foreground)

**Significance:** The design system already provides the tokens needed to implement the hierarchy - we just need to apply them correctly to different event types.

---

### Finding 3: Parent container uses monospace font, requiring explicit font-sans for reasoning

**Evidence:** Activity feed container (line 504) applies `font-mono text-xs` to all children. Reasoning text needs explicit `font-sans` class to override inheritance and display in standard font as per design requirements.

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte:504

**Significance:** Without explicit font override, reasoning text will render in monospace (JetBrains Mono) instead of standard font (Inter), reducing visual differentiation from tool calls.

---

### Finding 4: Tailwind spacing classes follow 4px grid system

**Evidence:** Current spacing uses gap-1 (4px), gap-2 (8px), py-1 (4px vertical), etc. Tailwind's default spacing scale is 4px-based (1 unit = 0.25rem = 4px), matching design-principles requirement.

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte (multiple lines), tailwind.config.js

**Significance:** Existing spacing implementation already complies with 4px grid requirement; no changes needed to spacing.

---

### Finding 5: Text events and reasoning events have insufficient differentiation

**Evidence:** Both text events (line 577) and reasoning events (line 569) use muted-foreground color. Text events use opacity-60 for icon, reasoning uses /70 opacity for text. The only visual difference is icon type (emoji vs bullet).

**Source:** web/src/lib/components/agent-detail/activity-tab.svelte:569, 577

**Significance:** Users cannot quickly distinguish between agent text output and internal reasoning without reading content or relying solely on small icon differences.

---

## Synthesis

**Key Insights:**

1. **Most requirements already met** - Tool calls already use monospace font, colored labels, and bold styling (Finding 1, 3). Tool results already nested, muted, and monospace (Finding 1). Spacing follows 4px grid (Finding 4). Only reasoning text needs font adjustment.

2. **Font inheritance is the blocker** - Parent container's font-mono (Finding 3) causes all children to inherit monospace unless explicitly overridden. Reasoning text needs font-sans to achieve standard font requirement.

3. **Contrast hierarchy underutilized** - Design system provides four levels (foreground → secondary → muted → faint) but current implementation only uses muted variants (Finding 2, 5). Text events should use higher contrast than reasoning to establish visual hierarchy.

**Answer to Investigation Question:**

Apply visual hierarchy by: (1) Add font-sans to reasoning text to override parent's monospace inheritance, (2) Differentiate text events from reasoning by using foreground color instead of muted, (3) Verify tool calls and results maintain current styling (already correct). Spacing already complies with 4px grid. No changes needed to tool styling which already meets all requirements.

---

## Structured Uncertainty

**What's tested:**

- ✅ Parent container uses font-mono (verified: read activity-tab.svelte:504)
- ✅ Tool calls already use monospace and colored labels (verified: read lines 521-537)
- ✅ Tool results already nested and muted (verified: read lines 543-547)
- ✅ Tailwind spacing uses 4px grid (verified: Tailwind default scale is 4px-based)
- ✅ Design system has four-level contrast hierarchy (verified: read app.css:19, tailwind.config.js)

**What's untested:**

- ⚠️ Reasoning text will be readable in both themes with font-sans (will verify in visual test)
- ⚠️ Text events with foreground color won't be too bright (will verify in visual test)
- ⚠️ Bullet prefix still visible with font-sans (will verify in visual test)

**What would change this:**

- Finding would be wrong if parent container already used font-sans (but verified it uses font-mono)
- Finding would be wrong if reasoning already had font-sans override (but verified it doesn't)
- Finding would be wrong if text events already differentiated from reasoning (but verified they use same muted color)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Minimal font-family override with contrast adjustment** - Add font-sans to reasoning text and increase text event contrast to foreground color.

**Why this approach:**
- Leverages existing correct implementation for tool calls and results (no changes needed)
- Minimal code changes - only two class modifications required
- Uses design system's existing contrast hierarchy (Finding 2)
- Directly addresses font inheritance issue (Finding 3) and differentiation gap (Finding 5)

**Trade-offs accepted:**
- Not creating new design tokens - using existing Tailwind classes
- Not changing parent container's font (keeps monospace default for terminal-style feed)
- Text events will be more prominent than reasoning (acceptable hierarchy: text output > reasoning thoughts)

**Implementation sequence:**
1. Add `font-sans` to reasoning text element (line 569) - establishes standard font vs monospace distinction
2. Change text events from `text-muted-foreground` to `text-foreground` (line 577) - creates visual hierarchy between output and reasoning
3. Visual verification via Playwright screenshot - confirm hierarchy visible in actual rendering

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
- Add `font-sans` class to reasoning text div (line 569) - highest impact for meeting core requirement
- Change text events to use `text-foreground` (line 577) - establishes proper hierarchy
- Visual verification via Playwright screenshot - mandatory gate for web/ changes

**Things to watch out for:**
- ⚠️ Don't change tool call styling (lines 521-537) - already meets all requirements
- ⚠️ Don't change tool result styling (lines 543-547) - already correct
- ⚠️ Verify reasoning text is readable in both light and dark themes
- ⚠️ Check that bullet prefix (opacity-60) still visible with new font

**Areas needing further investigation:**
- None - requirements are clear and implementation is straightforward
- Future consideration: Should step events (related events in tool groups) have different hierarchy?

**Success criteria:**
- ✅ Reasoning text renders in Inter font (not JetBrains Mono monospace)
- ✅ Text events are more prominent than reasoning (visual hierarchy established)
- ✅ Tool calls remain monospace with colored labels
- ✅ Tool results remain nested, muted, monospace
- ✅ Spacing remains on 4px grid (no changes needed)
- ✅ Screenshot evidence shows clear visual differentiation between all three types

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
