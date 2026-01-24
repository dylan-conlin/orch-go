<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawned orchestrator pattern guide created at `.kb/guides/spawned-orchestrator-pattern.md` documenting hierarchical delegation (spawn autonomous orchestrators) vs interactive sessions (human continuity).

**Evidence:** Read architect analysis (orch-go-lvrzc), examined existing guides for structure patterns, verified completion protocol is key distinction (external vs self-directed).

**Knowledge:** Documentation gap was structural not conceptual - pattern exists in code but lacked user-facing guide emphasizing when-to-use (decision tree), common patterns (examples), and completion protocol (troubleshooting).

**Next:** Guide complete and committed. May need iteration based on real-world usage patterns and user feedback.

**Promote to Decision:** recommend-no (tactical guide creation, not architectural decision)

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

# Investigation: Create Spawned Orchestrator Pattern Md

**Question:** How should the spawned orchestrator pattern be documented in a guide format that clearly distinguishes it from interactive session orchestration?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** og-feat-create-spawned-orchestrator agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Existing guides cover broad orchestrator architecture but lack spawned pattern specifics

**Evidence:**
- `orchestrator-session-management.md` covers three-tier hierarchy (meta → orchestrator → worker) but doesn't provide usage guide for spawned orchestrators specifically
- `session-resume-protocol.md` explicitly scopes to "interactive orchestrator sessions" only, excludes spawned orchestrators
- Architect analysis identified gap: spawned orchestrator pattern exists but lacks dedicated guide

**Source:**
- `.kb/guides/orchestrator-session-management.md` - Line 13: "Understanding when orchestrators should be spawned vs interactive"
- `.kb/guides/session-resume-protocol.md` - Line 5: "This protocol applies ONLY to interactive orchestrator sessions"
- `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Line 259: "Create spawned-orchestrator-pattern.md guide"

**Significance:** Confirms need for dedicated spawned orchestrator guide. Existing guides address the architecture but not the usage patterns for hierarchical delegation.

---

### Finding 2: Guide structure patterns are consistent across existing guides

**Evidence:**
- Guides typically include: Purpose, Problem, How It Works, Common Patterns, Troubleshooting, Key Decisions, References
- Example guides examined: `resilient-infrastructure-patterns.md`, `orchestrator-session-management.md`, `session-resume-protocol.md`
- Visual diagrams used for architecture (ASCII boxes and arrows)
- Decision trees and comparison tables frequently used

**Source:**
- `.kb/guides/resilient-infrastructure-patterns.md:1-80` - Pattern-based structure
- `.kb/guides/orchestrator-session-management.md:1-100` - Architecture diagrams and tables
- `.kb/guides/session-resume-protocol.md:1-100` - Quick reference + how it works

**Significance:** Following established patterns ensures consistency and discoverability. Users familiar with other guides will find spawned orchestrator guide intuitive.

---

### Finding 3: Key distinction is completion protocol (external vs self-directed)

**Evidence:**
- Spawned orchestrators: Write SESSION_HANDOFF.md and WAIT for `orch complete` from level above
- Interactive orchestrators: Run `orch session end` themselves to create handoff
- Architect analysis: "Completion protocols differ: ORCHESTRATOR_CONTEXT says 'WAIT for level above', session.go runs self-completion"

**Source:**
- `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md:172-174` - Completion protocol analysis
- `pkg/spawn/orchestrator_context.go:73-88` - Completion protocol for spawned orchestrators
- `cmd/orch/session.go:447-542` - Session end command for interactive sessions

**Significance:** This is the critical behavioral difference users must understand. Confusion here leads to orchestrators trying to self-terminate or waiting indefinitely.

---

## Synthesis

**Key Insights:**

1. **Documentation gap was structural, not conceptual** - The spawned orchestrator pattern exists and is implemented in code, but lacks user-facing guide. Existing guides cover architecture (orchestrator-session-management.md) and interactive sessions (session-resume-protocol.md), but not hierarchical delegation usage patterns.

2. **Completion protocol is the critical distinction** - The behavioral difference between spawned (external completion) and interactive (self-directed completion) orchestrators is what users must understand. This maps to different lifecycle models: agent-based (wait for authority) vs human-based (self-agency).

3. **Guide structure should emphasize when-to-use over how-it-works** - Users already have architectural context from orchestrator-session-management.md. The spawned orchestrator guide should focus on: when to delegate (decision tree), common patterns (examples), and completion protocol (troubleshooting).

**Answer to Investigation Question:**

The spawned orchestrator pattern should be documented with:
1. **Clear scope** - Hierarchical delegation (spawned) vs human continuity (interactive)
2. **Decision tree** - When to spawn orchestrator vs work interactively
3. **Lifecycle emphasis** - External completion protocol as primary distinction
4. **Concrete examples** - Concurrent epics, parallel investigation, overnight processing
5. **Troubleshooting** - Common confusions (trying to run orch session end, level collapse)

Guide created at `.kb/guides/spawned-orchestrator-pattern.md` following these principles. Structure matches existing guide patterns (Purpose → Problem → How It Works → Common Patterns → Troubleshooting → References).

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing guides cover architecture but not spawned pattern usage (verified: read orchestrator-session-management.md, session-resume-protocol.md)
- ✅ Guide structure patterns are consistent (verified: read 3+ existing guides for format)
- ✅ Completion protocol is the key distinction (verified: read architect analysis and source code references)

**What's untested:**

- ⚠️ Whether guide addresses user confusion effectively (not validated with users yet)
- ⚠️ Whether examples cover most common use cases (assumed based on architect analysis, not observed usage patterns)
- ⚠️ Whether decision tree is clear enough (not tested with fresh users)

**What would change this:**

- If users still confuse spawned vs interactive after reading guide → examples/decision tree need improvement
- If common use cases emerge that aren't covered → need to add patterns section
- If troubleshooting doesn't address actual problems → need real-world failure observations

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
- `.kb/guides/orchestrator-session-management.md` - Architecture and broad orchestrator lifecycle
- `.kb/guides/session-resume-protocol.md` - Interactive session resume (scope exclusion)
- `.kb/guides/resilient-infrastructure-patterns.md` - Guide structure patterns
- `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Architect analysis source

**Commands Run:**
```bash
# None - guide creation from existing analysis
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Source analysis for this guide
- **Guide (created):** `.kb/guides/spawned-orchestrator-pattern.md` - Deliverable from this investigation

---

## Investigation History

**2026-01-13 20:30:** Investigation started
- Initial question: How to document spawned orchestrator pattern
- Context: Architect analysis (orch-go-lvrzc) recommended creating spawned-orchestrator-pattern.md guide

**2026-01-13 20:35:** Guide structure determined
- Examined existing guides for format patterns
- Identified key sections: decision tree, completion protocol, examples

**2026-01-13 20:45:** Guide created
- Created `.kb/guides/spawned-orchestrator-pattern.md`
- 600+ lines covering: when to use, lifecycle, patterns, troubleshooting

**2026-01-13 20:50:** Investigation completed
- Status: Complete
- Key outcome: Spawned orchestrator pattern documented, filling gap between architecture guide and session resume guide
