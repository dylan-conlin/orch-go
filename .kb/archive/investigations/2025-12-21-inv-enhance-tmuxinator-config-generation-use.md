<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented tmuxinator config generation with port registry integration - orch spawn now auto-updates workers-{project}.yml with allocated ports.

**Evidence:** Tests pass (8 new tests), CLI command works (`orch port tmuxinator snap /path`), generated config includes correct port-specific dev server commands.

**Knowledge:** Tmuxinator configs can be auto-generated with port allocations; integration point is EnsureWorkersSession for automatic updates on spawn, plus manual CLI command for explicit regeneration.

**Next:** Close issue - implementation complete with tests and CLI command.

**Confidence:** High (90%) - Tested with unit tests and manual verification; edge cases for custom server command formats may need enhancement.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Enhance Tmuxinator Config Generation with Port Registry

**Question:** How can we generate tmuxinator workers-{project}.yml configs that use ports from the port registry?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent (feature-impl)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Port registry provides ListByProject API

**Evidence:** `pkg/port/port.go` has `ListByProject(project string) []Allocation` method that returns all port allocations for a given project.

**Source:** `pkg/port/port.go:258-269`

**Significance:** This provides the data needed to generate tmuxinator configs with correct ports per project.

---

### Finding 2: Existing tmuxinator configs use consistent format

**Evidence:** Examined `~/.tmuxinator/workers-*.yml` files - all follow same pattern with `servers` window and panes array.

**Source:** `~/.tmuxinator/workers-opencode.yml`, `~/.tmuxinator/workers-snap.yml`

**Significance:** The format is well-defined and can be generated programmatically.

---

### Finding 3: EnsureWorkersSession is the ideal hook point

**Evidence:** All tmux spawns call `EnsureWorkersSession(projectName, projectDir)` which creates the session. This is the natural point to also update tmuxinator config.

**Source:** `pkg/tmux/tmux.go:197-223`, `cmd/orch/main.go:981`

**Significance:** Hooking here means configs are auto-updated on every spawn without requiring explicit user action.

---

## Synthesis

**Key Insights:**

1. **Automatic config updates on spawn** - By hooking into EnsureWorkersSession, tmuxinator configs are automatically updated whenever a worker is spawned, ensuring the config always reflects current port allocations.

2. **Port-purpose mapping for commands** - Different port purposes (vite, api) require different command formats. Vite ports get `bun run dev --port N`, API ports get placeholder comments.

3. **CLI command for manual control** - The `orch port tmuxinator` command provides explicit control for regenerating configs after port allocations change.

**Answer to Investigation Question:**

Tmuxinator config generation was implemented via:
1. New `pkg/tmux/tmuxinator.go` with `GenerateTmuxinatorConfig()` that queries port registry
2. Hook in `EnsureWorkersSession` for automatic updates
3. CLI command `orch port tmuxinator <project> <dir>` for manual regeneration
4. Template-based YAML generation with conditional pane layouts

---

## Confidence Assessment

**Current Confidence:** [Level] ([Percentage])

**Why this level?**

[Explanation of why you chose this confidence level - what evidence supports it, what's strong vs uncertain]

**What's certain:**

- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]
- ✅ [Thing you're confident about with supporting evidence]

**What's uncertain:**

- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]
- ⚠️ [Area of uncertainty or limitation]

**What would increase confidence to [next level]:**

- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]
- [Specific additional investigation or evidence needed]

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

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
- Final confidence: [Level] ([Percentage])
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
