## Summary (D.E.K.N.)

**Delta:** Headless spawn evolved from initial implementation (Dec 20) to production-ready default (Dec 22-23), with 17 investigations documenting bugs fixed, stability improvements, and remaining edge cases.

**Evidence:** 17 investigations across 12 days covering: initial implementation, mode flip, 8 bug fixes (model format, project registration, beads discovery, phantom status, etc.), error visibility improvements, and 1 remaining race condition.

**Knowledge:** Headless spawn is production-ready with one known issue: rapid sequential spawns can hit race condition where SendPrompt fires before session is ready; fix recommendation exists (message verification with retry).

**Next:** Archive investigations superseded by this synthesis; fix remaining race condition (silent failures on rapid spawns) per 2025-12-30 investigation.

---

# Investigation: Headless Spawn Evolution Synthesis

**Question:** What is the consolidated state of headless spawn after 17 investigations, and what remains to be done?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** 
- .kb/investigations/2025-12-20-inv-implement-headless-spawn-mode-add.md
- .kb/investigations/2025-12-20-inv-make-headless-mode-default-deprecate.md
- .kb/investigations/2025-12-20-inv-scope-out-headless-swarm-implementation.md
- .kb/investigations/2025-12-21-inv-headless-spawn-not-sending-prompts.md
- .kb/investigations/2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md
- .kb/investigations/2025-12-22-inv-headless-spawn-mode-readiness-what.md
- .kb/investigations/2025-12-22-inv-headless-spawn-registers-wrong-project.md
- .kb/investigations/2025-12-22-inv-flip-default-spawn-mode-headless.md
- .kb/investigations/2025-12-23-debug-headless-spawn-model-format.md
- .kb/investigations/2025-12-23-inv-headless-spawn-does-not-pass.md
- .kb/investigations/2025-12-23-inv-orch-status-shows-headless-agents.md
- .kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md
- .kb/investigations/2025-12-27-inv-enhanced-error-visibility-headless-spawns.md
- .kb/investigations/2025-12-30-inv-headless-spawn-silent-failures-agents.md
- .kb/investigations/archived/2025-12-22-inv-test-headless-mode.md

---

## Evolution Timeline

### Phase 1: Initial Implementation (Dec 20, 2025)

**Implementation:** `--headless` flag added, uses HTTP API (CreateSession + SendPrompt) instead of tmux.
- Source: `2025-12-20-inv-implement-headless-spawn-mode-add.md`
- Key: Agents registered with `window_id='headless'` for tracking
- Confidence: 95%

**Default Flip:** Changed default from tmux to headless, added `--tmux` for opt-in.
- Source: `2025-12-20-inv-make-headless-mode-default-deprecate.md`
- Key: Logic inverted - headless is default, `--headless` deprecated (no-op)
- Confidence: 95%

**Swarm Scoping:** Defined "headless swarm" = batch execution with rate-limit management.
- Source: `2025-12-20-inv-scope-out-headless-swarm-implementation.md`
- Key: Created epic orch-go-bdd with 6 child tasks

---

### Phase 2: Bug Fixes (Dec 21-23, 2025)

| Bug | Root Cause | Fix | Source |
|-----|-----------|-----|--------|
| Prompts not sending | Outdated orch-go binary | Rebuild binary | 2025-12-21-inv-headless-spawn-not-sending-prompts.md |
| Not discoverable by beads ID | `runTail()` used naive lookup | Use `findWorkspaceByBeadsID()` | 2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md |
| Wrong project registered | Missing `--workdir` flag | Added `--workdir` flag | 2025-12-22-inv-headless-spawn-registers-wrong-project.md |
| Model format error (400) | Model passed as string, API expects object | Added `parseModelSpec()` | 2025-12-23-debug-headless-spawn-model-format.md |
| Model flag ignored | SendPrompt missing model param | Thread model through API | 2025-12-23-inv-headless-spawn-does-not-pass.md |
| Status shows phantom | Checked beads instead of session | OpenCode agents always non-phantom | 2025-12-23-inv-orch-status-shows-headless-agents.md |
| Token limit explosion (207k) | KB context bloat + double skill loading | Set ORCH_WORKER=1, limit KB tokens | 2025-12-23-inv-token-limit-explosion-headless-spawn.md |

---

### Phase 3: Readiness Assessment (Dec 22, 2025)

**Production Readiness:** Confirmed all 5 requirements met:
- Source: `2025-12-22-inv-headless-spawn-mode-readiness-what.md`
- Status detection: runStatus handles both modes
- Monitoring: SSE monitor tracks completions
- Completion detection: Beads comments + SSE
- Error handling: HTTP API errors propagate
- User visibility: spawn output, status, beads integration
- Confidence: 90%

**Testing:** End-to-end test confirmed agents spawn via API and produce artifacts.
- Source: `archived/2025-12-22-inv-test-headless-mode.md`
- Confidence: 85%

---

### Phase 4: Hardening (Dec 27-30, 2025)

**Error Visibility:** Added stderr capture and `--verbose` flag for debugging.
- Source: `2025-12-27-inv-enhanced-error-visibility-headless-spawns.md`
- Key: Background processes now log errors to events.jsonl

**Silent Failures (REMAINING ISSUE):** Race condition when spawning rapidly.
- Source: `2025-12-30-inv-headless-spawn-silent-failures-agents.md`
- Problem: SendPrompt fires immediately after CreateSession with no delay
- Evidence: 4th agent of 4 spawned in 45s had 0 messages
- Fix: Message verification with retry (recommended, not implemented)

---

## Findings

### Finding 1: Headless is Production-Ready with One Known Edge Case

**Evidence:** 
- 8 bugs identified and fixed across Dec 21-27
- Readiness assessment (Dec 22) confirmed all 5 requirements met
- Only remaining issue: race condition on rapid sequential spawns

**Source:** All 17 investigations synthesized above

**Significance:** Headless can be used confidently for most workloads. Only rapid batch spawns (4+ in <1 minute) risk silent failures.

---

### Finding 2: OpenCode API Has Undocumented Timing Requirements

**Evidence:**
- Tmux mode waits 1s (`PostReadyDelay`) before sending prompt
- Headless API mode has no delay - fires immediately
- Model must be object `{providerID, modelID}`, not string
- Directory passed via HTTP header, not JSON body

**Source:** 
- 2025-12-30-inv-headless-spawn-silent-failures-agents.md (timing)
- 2025-12-23-debug-headless-spawn-model-format.md (model format)
- 2025-12-22-inv-headless-spawn-registers-wrong-project.md (directory header)

**Significance:** OpenCode API contract is discovered through debugging, not documentation. Each API quirk required a separate investigation.

---

### Finding 3: Workspace Lookup Evolved to Handle Headless Agents

**Evidence:**
- Initial: `strings.Contains(entry.Name(), beadsID)` - failed for headless
- Fixed: `findWorkspaceByBeadsID()` scans SPAWN_CONTEXT.md for beads ID
- OpenCode agents now always non-phantom (they have running sessions)

**Source:**
- 2025-12-22-debug-headless-spawns-not-discoverable-by-beads-id.md
- 2025-12-23-inv-orch-status-shows-headless-agents.md

**Significance:** Beads ID is in SPAWN_CONTEXT.md, not directory name. Lookup logic unified across tmux/headless.

---

## Synthesis

**Key Insights:**

1. **Iterative Debugging Pattern** - Headless spawn required 8 bug fixes across 10 days. Each fix was targeted (model format, project registration, beads discovery, etc.) and independently verified. This pattern suggests API integration benefits from incremental testing.

2. **Tmux Parity Achieved** - All capabilities that worked in tmux mode now work in headless: status tracking, monitoring, completion detection, error handling, beads integration. The mode flip was justified.

3. **One Remaining Gap** - Race condition on rapid spawns (4+ in <1 min) is documented but not fixed. Recommended fix: `WaitForMessage()` polling after SendPrompt, with retry on timeout.

**Answer to Investigation Question:**

Headless spawn is production-ready. The 17 investigations document:
- Initial implementation and mode flip (Dec 20)
- 8 bug fixes covering model format, project registration, lookup, status display, token limits (Dec 21-27)
- Readiness confirmation with all 5 requirements met (Dec 22)
- One remaining edge case: rapid sequential spawns can silently fail (Dec 30)

The remaining fix (message verification with retry) is well-understood and scoped. Headless is safe for typical usage patterns; only batch spawning needs caution until the race condition is fixed.

---

## Structured Uncertainty

**What's tested:**

- Status detection works for headless agents (verified via `orch status`)
- Model format fix works (verified via smoke test)
- Beads ID lookup works via SPAWN_CONTEXT.md scan (verified via tests)
- End-to-end spawn produces artifacts (verified via test spawn)

**What's untested:**

- Message verification retry pattern (recommended but not implemented)
- Daemon integration with headless spawns (designed for it, not directly tested)
- Behavior under rate limiting with multiple Max accounts

**What would change this:**

- Finding would be wrong if OpenCode adds breaking API changes
- Recommendation would change if message verification has significant latency

---

## Remaining Work

### Priority 1: Fix Silent Failure Race Condition

**Issue:** SendPrompt fires immediately after CreateSession; rapid spawns can fail silently.

**Fix:** Add `WaitForMessage(sessionID, timeout, interval)` after SendPrompt:
1. Poll GetMessages until count > 0 or timeout (5s)
2. If timeout, retry SendPrompt once
3. Log failure if still empty

**Source:** 2025-12-30-inv-headless-spawn-silent-failures-agents.md

### Priority 2: Archive Superseded Investigations

These 15 investigations are superseded by this synthesis and can be archived:
- All investigations listed in "Supersedes" section above

---

## References

**Investigations Synthesized (17 total):**

| Date | Type | Topic | Status |
|------|------|-------|--------|
| 2025-12-20 | impl | Implement headless spawn mode | Complete, superseded |
| 2025-12-20 | impl | Make headless mode default | Complete, superseded |
| 2025-12-20 | scope | Scope headless swarm | Complete, superseded |
| 2025-12-21 | debug | Prompts not sending | Complete, fixed (rebuild binary) |
| 2025-12-22 | debug | Not discoverable by beads ID | Complete, fixed |
| 2025-12-22 | assess | Readiness assessment | Complete, superseded |
| 2025-12-22 | debug | Wrong project registered | Complete, fixed (--workdir) |
| 2025-12-22 | impl | Flip default to headless | Complete, superseded |
| 2025-12-22 | test | Test headless mode | Complete, archived |
| 2025-12-23 | debug | Model format error | Complete, fixed |
| 2025-12-23 | debug | Model flag ignored | Complete, fixed |
| 2025-12-23 | debug | Status shows phantom | Complete, fixed |
| 2025-12-23 | debug | Token limit explosion | Complete, fixed |
| 2025-12-27 | impl | Enhanced error visibility | Complete, fixed |
| 2025-12-30 | debug | Silent failures | Complete, fix recommended |

---

## Investigation History

**2026-01-01:** Synthesis started
- Initial question: What is the consolidated state of headless spawn after 17 investigations?
- Context: kb chronicle identified 17 headless-related investigations for consolidation

**2026-01-01:** Synthesis completed
- Status: Complete
- Key outcome: Headless is production-ready with one remaining edge case (race condition on rapid spawns)
