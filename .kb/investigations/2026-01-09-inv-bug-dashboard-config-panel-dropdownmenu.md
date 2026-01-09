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

# Investigation: Bug Dashboard Config Panel Dropdownmenu

**Question:** Why does the DropdownMenu component not render content when clicked in the dashboard config panel and settings panel?

**Started:** 2026-01-09
**Updated:** 2026-01-09  
**Owner:** og-debug-bug-dashboard-config-09jan-3269
**Phase:** Investigating
**Next Step:** Test Portal-less rendering approach or browser verification
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Tooltip works without Portal, DropdownMenu uses Portal

**Evidence:** 
- `tooltip-content.svelte` renders directly to TooltipPrimitive.Content (no Portal wrapper)
- `dropdown-menu-content.svelte` wraps DropdownMenuPrimitive.Content in DropdownMenuPrimitive.Portal
- Tooltip components function correctly in the dashboard
- DropdownMenu components do not render visible content

**Source:** 
- `web/src/lib/components/ui/tooltip/tooltip-content.svelte` (lines 13-21)
- `web/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte` (lines 13-23)

**Significance:** Portal is the key difference between working (Tooltip) and broken (DropdownMenu) components, suggesting Portal rendering issue with bits-ui + Svelte 5.

---

### Finding 2: Both Settings and Daemon config dropdowns are broken

**Evidence:**
- Prior investigation (og-feat-dashboard-config-editing-08jan-13ee) noted "Same behavior observed for Settings dropdown"
- Both SettingsPanel and DaemonConfigPanel use DropdownMenu.Root wrapper
- Both use identical {#snippet child({ props })} pattern for triggers

**Source:**
- `web/src/lib/components/settings-panel/settings-panel.svelte` (lines 19-36)
- `web/src/lib/components/stats-bar/stats-bar.svelte` (lines 245-286)
- `.orch/workspace/og-feat-dashboard-config-editing-08jan-13ee/SYNTHESIS.md` (line 58)

**Significance:** This is a systemic issue affecting all DropdownMenu components, not specific to the daemon config panel.

---

### Finding 3: Z-index conflict with header

**Evidence:**
- Header has `z-50` (sticky top-0)
- DropdownMenu.Content has `z-50` by default
- Same z-index level could cause rendering conflicts

**Source:**
- `web/src/routes/+layout.svelte` (line 52)
- `web/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte` (line 18)

**Significance:** Z-index increased to 100 to ensure dropdown renders above header, but this doesn't solve the core Portal issue.

---

### Finding 4: bits-ui version and Svelte 5 compatibility

**Evidence:**
- bits-ui: ^2.11.0
- svelte: ^5.43.8
- Portal component from bits-ui may have compatibility issues with Svelte 5 runes

**Source:**
- `web/package.json` (dependencies)

**Significance:** Version compatibility may be root cause, but cannot verify without web access to bits-ui GitHub/docs.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

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

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

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
