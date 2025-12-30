<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Behavioral patterns from action-log.jsonl are now automatically injected into SPAWN_CONTEXT.md, surfacing futile action patterns to agents.

**Evidence:** Tests verify patterns are detected from log, formatted correctly, and included in spawn context template.

**Knowledge:** Pattern detection uses existing action.Tracker.FindPatterns() with threshold of 3+ occurrences; patterns are limited to top 5 to avoid context bloat.

**Next:** Close - implementation is complete with tests passing.

---

# Investigation: Inject Behavioral Patterns Into Spawn

**Question:** How should we inject behavioral patterns (from action-log.jsonl) into SPAWN_CONTEXT.md so agents avoid repeating futile actions?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Worker agent (og-feat-inject-behavioral-patterns-29dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Pattern detection already exists in pkg/action

**Evidence:** The action.Tracker type has FindPatterns() method that returns []ActionPattern with Count, Tool, Target, Outcome fields. Threshold is 3 occurrences within 7 days.

**Source:** pkg/action/action.go:395-472

**Significance:** No new pattern detection logic needed - just need to format existing patterns for spawn context.

---

### Finding 2: Spawn context uses template-based generation

**Evidence:** SpawnContextTemplate is a Go template with conditional sections for ecosystem context, server context, etc. BehavioralPatterns field can be added similarly.

**Source:** pkg/spawn/context.go:19-257 (template), pkg/spawn/context.go:404-460 (GenerateContext)

**Significance:** Adding behavioral patterns follows established pattern - add field to contextData, add section to template, generate in GenerateContext().

---

### Finding 3: Existing context injection follows opt-in pattern

**Evidence:** Server context and ecosystem context are conditionally included based on skill type or project. Behavioral patterns should always be included when detected (no opt-out needed).

**Source:** pkg/spawn/context.go:414-426 (server/ecosystem context generation)

**Significance:** Behavioral patterns can auto-generate without needing configuration since they only appear when patterns exist.

---

## Synthesis

**Key Insights:**

1. **Minimal new code needed** - The pattern detection infrastructure already exists in pkg/action. Implementation primarily involves formatting and template integration.

2. **Automatic injection** - Unlike server/ecosystem context which are conditionally included, behavioral patterns should always appear when detected since they directly prevent agent failure.

3. **Context budget awareness** - Patterns are limited to top 5 with a "more patterns" indicator to avoid context bloat while still surfacing critical warnings.

**Answer to Investigation Question:**

Inject behavioral patterns by: (1) Adding GenerateBehavioralPatternsContext() function that calls action.LoadTracker() and formats patterns; (2) Adding BehavioralPatterns field to Config and contextData; (3) Adding template section with warning header and explanation. The implementation surfaces patterns as warnings with icons indicating severity (🚫 for 5+ occurrences, ⚠️ for 3-4).

---

## Structured Uncertainty

**What's tested:**

- ✅ GenerateBehavioralPatternsContext returns empty when no log exists (test passes)
- ✅ GenerateBehavioralPatternsContext returns patterns when futile actions detected (test passes)
- ✅ GenerateContext includes patterns section when patterns exist (test passes)
- ✅ GenerateContext excludes patterns section when no patterns (test passes)
- ✅ Provided BehavioralPatterns field overrides auto-generation (test passes)

**What's untested:**

- ⚠️ Real-world effectiveness (do agents actually avoid warned patterns?)
- ⚠️ Performance impact when action-log.jsonl is very large
- ⚠️ Workspace-specific pattern filtering effectiveness

**What would change this:**

- If agents ignore the warning section, may need more prominent placement
- If log files become too large, may need indexing or pre-filtering

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Direct integration into GenerateContext** - Call action.LoadTracker().FindPatterns() during context generation.

**Why this approach:**
- Uses existing pattern detection infrastructure
- Automatic - no user action required
- Patterns surfaced at spawn time when most useful

**Trade-offs accepted:**
- Small I/O overhead reading action log on every spawn
- Not filtering patterns to workspace-specific (global patterns shown)

**Implementation sequence:**
1. Add BehavioralPatterns field to Config struct (config.go)
2. Add GenerateBehavioralPatternsContext() function (context.go)
3. Add template section with warning formatting
4. Call generation in GenerateContext() with fallback to auto-generate

### Alternative Approaches Considered

**Option B: Pre-compute patterns periodically**
- **Pros:** No I/O on spawn, patterns cached
- **Cons:** Complexity of caching, stale patterns possible
- **When to use instead:** If performance becomes issue

**Option C: CLI command to show patterns**
- **Pros:** User-controlled, no context overhead
- **Cons:** Agents don't see patterns automatically
- **When to use instead:** Never - defeats the purpose

**Rationale for recommendation:** Direct integration is simplest and ensures agents always see patterns.

---

### Implementation Details

**What to implement first:**
- ✅ BehavioralPatterns field in Config (config.go:122-127)
- ✅ GenerateBehavioralPatternsContext function (context.go:934-990)
- ✅ Template section for patterns (context.go:37-47)
- ✅ Integration in GenerateContext (context.go:438-442)
- ✅ Tests for all cases (context_test.go:1335-1445)

**Things to watch out for:**
- ⚠️ Pattern target normalization (file paths become *.ext patterns)
- ⚠️ Need to import action package in spawn package

**Areas needing further investigation:**
- Workspace-specific pattern filtering could be added later
- May want orch patterns command for CLI visibility

**Success criteria:**
- ✅ Tests pass for pattern detection and context generation
- ✅ No regression in existing spawn tests
- ✅ Build compiles without errors

---

## References

**Files Examined:**
- pkg/action/action.go - Pattern detection implementation
- pkg/spawn/context.go - Context generation template and logic
- pkg/spawn/config.go - Config struct definition

**Commands Run:**
```bash
# Run tests
go test ./pkg/spawn/... -v -run "Behavioral" -count=1
go test ./pkg/spawn/... ./pkg/action/... -count=1
```

**Related Artifacts:**
- **Issue:** orch-go-o84w - Inject behavioral patterns into SPAWN_CONTEXT.md
- **Epic:** orch-go-4oh7 - Patterns are detected but not surfaced to agents

---

## Investigation History

**2025-12-29 17:24:** Investigation started
- Initial question: How to inject behavioral patterns from action-log.jsonl into SPAWN_CONTEXT.md
- Context: Epic orch-go-4oh7 - patterns detected but not surfaced to agents automatically

**2025-12-29 17:30:** Implementation complete
- Added BehavioralPatterns field, GenerateBehavioralPatternsContext(), template section
- All tests passing

**2025-12-29 17:35:** Investigation completed
- Status: Complete
- Key outcome: Behavioral patterns now auto-injected into spawn context
