<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session handoffs are now scoped by tmux window name (.orch/session/{window-name}/latest) to prevent concurrent orchestrator sessions from clobbering each other's context.

**Evidence:** Implemented GetCurrentWindowName with filesystem-safe sanitization, updated session end/resume commands to use window-scoped paths, tested that window "🏗️ og-feat-implement-tmux-window-13jan-4191 [orch-go-uwo6p]" sanitizes to "og-feat-implement-tmux-window-13jan-4191-orch-go-uwo6p", and verified separate handoff directories per window.

**Knowledge:** Window names containing emojis, spaces, and special characters must be sanitized for filesystem compatibility - keep only alphanumeric, dash, and underscore characters.

**Next:** Update session-resume-protocol.md documentation updated, hooks will automatically use window-scoped discovery.

**Promote to Decision:** recommend-no (implementation detail, not architectural decision)

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

# Investigation: Implement Tmux Window Scoped Session

**Question:** How can we scope session handoffs by tmux window name to prevent concurrent orchestrator sessions from clobbering each other's context?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** og-feat-implement-tmux-window agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Window-scoped session directories prevent context clobbering

**Evidence:**
- Changed structure from `.orch/session/latest` to `.orch/session/{window-name}/latest`
- Each tmux window now has independent handoff storage
- Window names are sanitized for filesystem safety (emojis and special chars removed)
- Test shows window "🏗️ og-feat-implement-tmux-window-13jan-4191 [orch-go-uwo6p]" becomes "og-feat-implement-tmux-window-13jan-4191-orch-go-uwo6p"

**Source:**
- `pkg/tmux/tmux.go:63-123` - GetCurrentWindowName with sanitization
- `cmd/orch/session.go:666-761` - createSessionHandoffDirectory with window scoping
- `cmd/orch/session.go:614-672` - discoverSessionHandoff with window scoping

**Significance:** Multiple orchestrator sessions in different windows can now run concurrently without overwriting each other's context. Window name sanitization ensures compatibility with all filesystems.

---

### Finding 2: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

## Synthesis

**Key Insights:**

1. **Window-scoped directories solve the multi-session problem** - By using `.orch/session/{window-name}/latest` instead of `.orch/session/latest`, each tmux window maintains independent session context without interference.

2. **Filesystem safety requires sanitization** - Tmux window names can contain emojis, brackets, and special characters that are invalid in directory names. Sanitization (keeping only alphanumeric, dash, underscore) ensures cross-platform compatibility.

3. **"default" fallback enables non-tmux usage** - When not in a tmux session, using "default" as the window name maintains backward compatibility and allows interactive sessions outside tmux.

**Answer to Investigation Question:**

Session handoffs can be scoped by tmux window name using the structure `.orch/session/{window-name}/latest`. This prevents concurrent orchestrator sessions from clobbering context by giving each window independent handoff storage. Window names are sanitized to remove emojis and special characters for filesystem safety. The implementation automatically detects the current window (or uses "default" outside tmux) and scopes all handoff operations accordingly.

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
