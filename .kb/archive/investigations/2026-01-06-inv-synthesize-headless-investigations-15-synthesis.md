## Summary (D.E.K.N.)

**Delta:** Synthesized 15 headless investigations into authoritative guide covering implementation, bugs fixed, common issues, and architecture decisions.

**Evidence:** Read and analyzed all 15 investigations spanning Dec 20, 2025 - Jan 6, 2026; identified 6 major bug fixes, 5 key decisions, and common troubleshooting patterns.

**Knowledge:** Headless mode evolved from experimental feature to production-ready default through iterative bug fixing (model format, beads lookup, phantom status, project directory, token bloat).

**Next:** Guide created at `.kb/guides/headless.md`; original investigations can be archived.

**Promote to Decision:** no (consolidation, not new decision)

---

# Investigation: Synthesize 15 Headless Investigations

**Question:** What patterns and knowledge can be consolidated from 15 headless-related investigations into a single authoritative guide?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-feat-synthesize-headless-investigations-06jan-eb35
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Six major bugs were fixed during headless development

**Evidence:** 

| Bug | Root Cause | Fix | Investigation |
|-----|------------|-----|---------------|
| Model ignored | OpenCode expects model as object, not string | Added `parseModelSpec()` | 2025-12-23-debug-headless-spawn-model-format.md |
| Agent not findable | Naive `strings.Contains(dir, beadsID)` lookup | Use `findWorkspaceByBeadsID()` | 2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md |
| Phantom status | Checked beads status instead of session existence | `isPhantom = false` for OpenCode agents | 2025-12-23-inv-orch-status-shows-headless-agents.md |
| Wrong project | Directory passed in JSON body, not header | Use `x-opencode-directory` header | 2025-12-22-inv-headless-spawn-registers-wrong-project.md |
| Prompts not sent | Outdated binary still using tmux | Rebuild from source | 2025-12-21-inv-headless-spawn-not-sending-prompts.md |
| Model not threaded | SendMessageAsync missing model parameter | Add model to SendPrompt signature | 2025-12-23-inv-headless-spawn-does-not-pass.md |

**Source:** All 6 debug/fix investigations

**Significance:** Most bugs stemmed from API contract mismatches (OpenCode expects different formats than assumed) or duplicate code paths diverging from correct implementations.

---

### Finding 2: Key architectural decisions established during development

**Evidence:**

| Decision | kn ID | Rationale |
|----------|-------|-----------|
| Headless is default | kn-6f7dd1 | Optimizes for automation, reduces TUI overhead |
| Tmux is opt-in | kn-318507 | Visual monitoring available when needed |
| Per-message model | kn-a485c6 | OpenCode design constraint |
| ORCH_WORKER=1 for workers | kn-56f594 | Prevents double skill loading (saves ~37k tokens) |
| Beads comments for phases | (design decision) | Spawn-mode agnostic |

**Source:** Chronicle output and individual investigations

**Significance:** These decisions form the foundation of headless mode's current design and should be preserved.

---

### Finding 3: Token limit explosion was a significant production issue

**Evidence:**

- Investigation `2025-12-23-inv-token-limit-explosion-headless-spawn.md` found 207k token failure
- Causes: KB context explosion (60k+), double skill loading (37k), OpenCode overhead (40-60k)
- Solutions: `ORCH_WORKER=1`, KB token limits, `--skip-artifact-check`

**Source:** 2025-12-23-inv-token-limit-explosion-headless-spawn.md

**Significance:** Token management remains important for headless spawns; the guide documents prevention strategies.

---

### Finding 4: Headless mode was thoroughly tested before becoming default

**Evidence:**

Four test investigations in archived folder:
- `2025-12-22-inv-test-headless-mode.md` - E2E functionality (85% confidence)
- `2025-12-22-inv-test-headless-spawn-list-files.md` - Filesystem ops (90% confidence)
- `2025-12-22-inv-test-headless-spawn.md` - Default mode verification (95% confidence)
- `2025-12-23-inv-test-headless-spawn-after-fix.md` - Post-fix validation

All tests confirmed headless mode works correctly for session creation, filesystem operations, KB integration, and artifact production.

**Source:** Archived test investigations

**Significance:** High confidence that headless mode is production-ready.

---

### Finding 5: Production readiness assessment was done before default flip

**Evidence:**

Investigation `2025-12-22-inv-headless-spawn-mode-readiness-what.md` verified all five requirements:
- ✅ Status detection (unified agent list)
- ✅ Monitoring (SSE monitor)
- ✅ Completion detection (beads comments + SSE)
- ✅ Error handling (HTTP API errors propagate)
- ✅ User visibility (spawn output, status, monitor)

Confidence: High (90%) - "Headless is already production-ready"

**Source:** 2025-12-22-inv-headless-spawn-mode-readiness-what.md

**Significance:** Formal verification that headless was ready before becoming default.

---

## Synthesis

**Key Insights:**

1. **Iterative bug fixing matured the feature** - Headless mode evolved from initial implementation (Dec 20) to production-ready default through 6+ bug fixes over 3 days. Each bug revealed an API contract or design assumption that needed correction.

2. **Documentation-code mismatch was recurring theme** - Multiple investigations found CLAUDE.md claiming headless was default when tmux actually was (until the flip). This highlights the importance of keeping docs synchronized with implementation.

3. **Fire-and-forget design enables scalability** - The core design decision (return immediately, agent runs asynchronously) enables parallel spawning and daemon automation. All the complexity is in monitoring and completion detection.

4. **Token management is a production concern** - The 207k token explosion revealed that KB context and skill loading can easily exceed Claude's limits. Worker spawns need `ORCH_WORKER=1` to prevent double skill loading.

**Answer to Investigation Question:**

The 15 investigations consolidate into:

1. **How headless works** - HTTP API (CreateSession + SendPrompt), fire-and-forget, returns immediately
2. **When to use it** - Daemon, batch, overnight work (default); tmux for debugging
3. **Common issues** - Model format, beads lookup, phantom status, token limits
4. **Architecture** - Per-message model selection, workspace/session/tmux layers
5. **Key decisions** - Headless default, ORCH_WORKER=1, beads-based phase tracking

All captured in `.kb/guides/headless.md`.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 15 investigations read and analyzed
- ✅ Bug fixes verified as implemented (code references in investigations)
- ✅ Guide covers all major topics from investigations

**What's untested:**

- ⚠️ Whether guide is sufficient for new users (no user testing)
- ⚠️ Whether any bugs regressed since original fixes (not re-verified)

**What would change this:**

- User feedback indicating missing information
- New headless-related bugs revealing gaps in documentation

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide created** - The authoritative guide is at `.kb/guides/headless.md`.

**Why this approach:**
- Consolidates scattered knowledge into single reference
- Enables future agents to start here instead of reading 15 investigations
- Provides troubleshooting patterns for common issues

**Next steps:**
1. Consider archiving the 4 test investigations (already in archived/)
2. Keep debug investigations for historical context (they document why things are the way they are)
3. Update guide when new headless issues arise

---

## References

**Files Examined:**

- `.kb/investigations/2025-12-20-inv-implement-headless-spawn-mode-add.md`
- `.kb/investigations/2025-12-20-inv-make-headless-mode-default-deprecate.md`
- `.kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md`
- `.kb/investigations/2025-12-21-inv-headless-spawn-not-sending-prompts.md`
- `.kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md`
- `.kb/investigations/2025-12-22-inv-headless-spawn-mode-readiness-what.md`
- `.kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md`
- `.kb/investigations/2025-12-23-debug-headless-spawn-model-format.md`
- `.kb/investigations/2025-12-23-inv-headless-spawn-does-not-pass.md`
- `.kb/investigations/2025-12-23-inv-orch-status-shows-headless-agents.md`
- `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md`
- `.kb/investigations/archived/2025-12-22-inv-test-headless-mode.md`
- `.kb/investigations/archived/2025-12-22-inv-test-headless-spawn-list-files.md`
- `.kb/investigations/archived/2025-12-22-inv-test-headless-spawn.md`
- `.kb/investigations/archived/2025-12-23-inv-test-headless-spawn-after-fix.md`

**Commands Run:**

```bash
# Get chronicle timeline
kb chronicle "headless"

# Check existing guides
ls .kb/guides/
```

**Related Artifacts:**

- **Guide:** `.kb/guides/headless.md` - Output of this synthesis
- **Guide:** `.kb/guides/spawn.md` - Related spawn documentation

---

## Investigation History

**2026-01-06 16:40:** Investigation started
- Initial question: What patterns can be consolidated from 15 headless investigations?
- Context: `kb reflect --type synthesis` identified opportunity

**2026-01-06 16:50:** Read all 15 investigations
- Identified 6 major bugs fixed
- Identified 5 key decisions
- Mapped common issues and solutions

**2026-01-06 17:00:** Created authoritative guide
- `.kb/guides/headless.md` created with synthesized knowledge
- Status: Complete
- Key outcome: Single authoritative reference replaces 15 scattered investigations
