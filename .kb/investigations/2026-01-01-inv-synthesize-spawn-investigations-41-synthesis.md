## Summary (D.E.K.N.)

**Delta:** The spawn system evolved from a simple tmux-based launcher (Dec 19) to a comprehensive orchestration layer with headless mode, kb context injection, behavioral pattern warnings, pre-spawn gates, and tiered tracking - but accumulated 15+ test-only investigations that can be archived.

**Evidence:** Analyzed 41 spawn investigations spanning Dec 19-30, 2025; identified 5 major evolution phases, 15 test/verification artifacts, and 4 key architectural decisions (headless default, tiered tracking, kb context, behavioral patterns).

**Knowledge:** The spawn system has matured significantly - core infrastructure is stable. Remaining evolution is in spawn CONTEXT (what agents see) rather than spawn MECHANISM (how agents start). Future work should focus on context quality over mechanism changes.

**Next:** Archive 15 test-only investigations to archived/, mark this synthesis complete.

---

# Investigation: Synthesize 41 Spawn Investigations (Dec 2025)

**Question:** What patterns, contradictions, and consolidation opportunities exist across 41 spawn-related investigations from December 2025?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Agent og-feat-synthesize-spawn-investigations-01jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## The Evolution Story (5 Phases)

### Phase 1: Initial Implementation (Dec 19)

**Key Investigation:** `2025-12-19-inv-cli-orch-spawn-command.md`

The spawn command was created with:
- Skill loader (`pkg/skills/`) discovering skills from `~/.claude/skills/`
- SPAWN_CONTEXT.md template generation (`pkg/spawn/context.go`)
- Beads integration for tracking (`bd create`)
- tmux-based agent spawning as the default mode

**Outcome:** Working spawn command with skill loading and workspace creation.

---

### Phase 2: Multi-Mode Support (Dec 20-22)

**Key Investigations:**
- `2025-12-20-inv-implement-headless-spawn-mode-add.md` - Added `--headless` flag
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Flipped default to headless
- `2025-12-21-inv-add-tmux-flag-orch-spawn.md` - Made tmux opt-in

**Evolution:**
1. Dec 20: Added headless mode as opt-in (`--headless`)
2. Dec 21: Added `--tmux` flag for explicit tmux mode
3. Dec 22: Flipped default from tmux to headless

**Decision:** Headless is now the default spawn mode; use `--tmux` for visual monitoring.

---

### Phase 3: Context Enrichment (Dec 22-28)

**Key Investigations:**
- `2025-12-22-inv-fix-pre-spawn-kb-context.md` - Added `--global` flag for cross-repo knowledge
- `2025-12-28-inv-spawn-context-include-related-prior.md` - Include D.E.K.N. Delta from prior investigations
- `2025-12-28-inv-expand-orchecosystemrepos-pkg-spawn-kbcontext.md` - Ecosystem repos context
- `2025-12-29-inv-inject-behavioral-patterns-into-spawn.md` - Action-log pattern injection

**Evolution:**
- Spawn context now includes kb context (decisions, constraints, prior investigations)
- Limited to 3 most recent investigations with D.E.K.N. Delta summaries
- Behavioral patterns from action-log.jsonl injected to prevent futile actions
- Ecosystem repos listed for cross-project context

**Insight:** Context quality is more important than spawn mechanism. Agents succeed or fail based on what they see in SPAWN_CONTEXT.md.

---

### Phase 4: Failure Prevention (Dec 25-30)

**Key Investigations:**
- `2025-12-25-inv-pre-spawn-phase-complete-check.md` - Check if work already complete
- `2025-12-25-inv-pre-spawn-token-estimation-prevent.md` - Prevent context overflow
- `2025-12-30-inv-add-pre-spawn-check-verifies.md` - Block stale bugs (Gate Over Remind)

**Evolution:**
- Pre-spawn checks can now block spawns (not just warn)
- Gate Over Remind pattern: block by default, explicit bypass with `--skip-*` flags
- Stale bug detection via git log analysis

**Insight:** "Gate Over Remind" is more effective than warnings. Blocking saves more agent time than polite warnings.

---

### Phase 5: Observability (Dec 29-30)

**Key Investigations:**
- `2025-12-29-inv-batch-spawn-failure-analysis.md` - Analyzed 5-agent batch spawn failures
- `2025-12-29-inv-detect-sessions-spawn-but-never.md` - Detect zombie sessions
- `2025-12-30-inv-implement-spawntelemetry-event-observability-mvp.md` - Telemetry events

**Problems Discovered:**
1. Sessions can spawn but never execute (silent failure)
2. Agents can report "complete" without commits (false positive)
3. Agents go idle instead of reporting blocked status
4. No health check between spawn and first tool call

**Solutions Implemented:**
- Spawn telemetry for tracking spawn → execution → completion
- `orch complete` should verify git commits exist

---

## Test/Verification Investigations (Candidates for Archive)

These 15 investigations were spawn verification tests, not architectural discoveries. They should be moved to `archived/`:

| Investigation | Purpose | Status |
|--------------|---------|--------|
| `2025-12-22-inv-test-spawn-context.md` | Verify context generation | Can archive |
| `2025-12-22-inv-test-spawn-tracking-works.md` | Verify beads tracking | Can archive |
| `2025-12-22-inv-test-spawn-verify-investigation-skill.md` | Verify skill loading | Can archive |
| `2025-12-22-inv-test-spawn-verify-pre-spawn.md` | Verify pre-spawn checks | Can archive |
| `2025-12-22-inv-test-spawn-works-after-phantom.md` | Verify after bug fix | Can archive |
| `2025-12-23-inv-test-spawn-after-fix.md` | Verify after fix | Can archive |
| `2025-12-23-inv-test-spawn-fresh-build.md` | Verify fresh build | Can archive |
| `2025-12-23-inv-test-spawn-functionality.md` | General verification | Can archive |
| `2025-12-23-inv-test-spawn.md` | General verification | Can archive |
| `2025-12-23-inv-verify-spawn-works.md` | General verification | Can archive |
| `2025-12-24-inv-test-spawn-24dec.md` | Daily verification | Can archive |
| `2025-12-26-inv-test-orch-spawn-context.md` | Context verification | Can archive |
| `2025-12-26-inv-test-spawn-context.md` | Context verification | Can archive |
| `2025-12-28-inv-test-spawn-debugging.md` | Debugging verification | Can archive |
| `2025-12-28-inv-test-spawn-fix-say-hello.md` | Hello test | Can archive |
| `2025-12-28-inv-test-spawn-say-hello-immediately.md` | Hello test | Can archive |

---

## Key Findings

### Finding 1: Mode evolution settled on headless default

**Evidence:** The spawn mode evolved: tmux default (Dec 19) → headless opt-in (Dec 20) → headless default (Dec 22). This is now stable - no investigations after Dec 22 questioned the default.

**Source:** 
- `2025-12-20-inv-implement-headless-spawn-mode-add.md`
- `2025-12-22-inv-flip-default-spawn-mode-headless.md`

**Significance:** Mode selection is a solved problem. Future work is on context, not mechanism.

---

### Finding 2: Context enrichment is the primary evolution vector

**Evidence:** 6 investigations (Dec 22-30) focused on adding context:
- kb context with `--global` for cross-repo knowledge
- D.E.K.N. Delta extraction for prior investigation summaries
- Behavioral patterns from action-log.jsonl
- Ecosystem repos listing
- Server context for project-specific servers

**Source:** 
- `pkg/spawn/kbcontext.go` - 560+ lines of context generation
- `pkg/spawn/context.go` - Template with 10+ conditional sections

**Significance:** SPAWN_CONTEXT.md quality determines agent success. Investment in context enrichment pays dividends across all spawns.

---

### Finding 3: Gate Over Remind pattern established

**Evidence:** Three pre-spawn gates now block (not warn):
1. Failure report unfilled → blocks spawn
2. Stale bug detected → blocks spawn (Dec 30)
3. Phase: Complete already exists → can warn or block

Each has explicit bypass flag (`--skip-failure-review`, `--skip-stale-check`).

**Source:** `2025-12-30-inv-add-pre-spawn-check-verifies.md`

**Significance:** Blocking prevents more wasted agent time than warnings. The system is evolving toward "fail fast" rather than "proceed with caution."

---

### Finding 4: Silent failures are the worst failures

**Evidence:** Batch spawn analysis (Dec 29) found:
- Session created but never executed (0 beads comments)
- Agent reported "complete" without any git commits
- Agent went idle at "visual verification" without reporting blocked

**Source:** `2025-12-29-inv-batch-spawn-failure-analysis.md`

**Significance:** Observability gaps cause more damage than mechanism failures. Future work should focus on:
- Health checks between spawn and first tool call
- Commit verification before allowing completion
- BLOCKED status pattern for stuck agents

---

## Synthesis

**Key Insights:**

1. **Spawn mechanism is stable** - Mode selection (headless/tmux/inline), skill loading, workspace creation, and beads integration are mature. No mechanism bugs reported after Dec 23.

2. **Context is the leverage point** - Agent success correlates with SPAWN_CONTEXT.md quality. Adding kb context, behavioral patterns, and prior investigation summaries improved agent effectiveness.

3. **Gates beat warnings** - The Gate Over Remind pattern (block by default, explicit bypass) prevents more waste than polite warnings. Apply to more pre-spawn checks.

4. **Observability is the next frontier** - Silent failures (session never executes, completion without commits, idle without blocked status) are the current gap.

**Answer to Investigation Question:**

The 41 spawn investigations reveal a mature, stable spawn mechanism with ongoing evolution in context enrichment and observability. Key consolidation opportunities:
- **Archive 15 test-only investigations** - They verified fixes but contain no reusable knowledge
- **No contradictions found** - The evolution was additive (new features) not corrective (changing decisions)
- **Future work is context/observability** - Mechanism changes are complete; focus on what agents see and how we detect failures

---

## Structured Uncertainty

**What's tested:**

- ✅ Headless is stable default mode (verified: no issues since Dec 22)
- ✅ Context enrichment improves agent success (verified: kb context, behavioral patterns in production)
- ✅ Gate Over Remind prevents wasted time (verified: stale bug gate blocks spawns)

**What's untested:**

- ⚠️ Whether archived investigations contain any hidden gems (quick scan only)
- ⚠️ Long-term stability of spawn telemetry (recently implemented)
- ⚠️ Whether commit verification will catch all false positives

**What would change this:**

- Finding would be wrong if mechanism bugs resurface requiring architectural changes
- Finding would be wrong if archived tests contained unique insights beyond verification

---

## Implementation Recommendations

### Recommended Approach ⭐

**Archive test investigations and focus future spawn work on observability**

**Why this approach:**
- Reduces investigation noise (15 fewer files to search)
- Clear signal that mechanism work is complete
- Focuses future effort on highest-value area (observability)

**Implementation sequence:**
1. Move 15 test investigations to `archived/` directory
2. Document in CLAUDE.md that spawn mechanism is stable
3. Create observability epic for spawn health checks

### Archival Script

```bash
# Archive test investigations
cd .kb/investigations
mv 2025-12-2*-inv-test-spawn*.md archived/
mv 2025-12-23-inv-verify-spawn-works.md archived/
```

---

## References

**Files Examined:**
- 41 spawn-related investigations in `.kb/investigations/`
- 13 archived spawn investigations in `.kb/investigations/archived/`
- `kb chronicle "spawn"` output (584 entries)

**Key Investigations:**
- `2025-12-19-inv-cli-orch-spawn-command.md` - Initial implementation
- `2025-12-20-inv-implement-headless-spawn-mode-add.md` - Headless mode
- `2025-12-22-inv-flip-default-spawn-mode-headless.md` - Default flip
- `2025-12-28-inv-spawn-context-include-related-prior.md` - D.E.K.N. Delta
- `2025-12-29-inv-batch-spawn-failure-analysis.md` - Failure analysis
- `2025-12-30-inv-add-pre-spawn-check-verifies.md` - Stale bug gate

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-31-inv-synthesize-22-daemon-investigations-dec.md` - Similar synthesis for daemon

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: What patterns exist across 41 spawn investigations?
- Context: Topic accumulation flagged by kb knowledge review

**2026-01-01:** Analysis complete
- Identified 5 evolution phases
- Found 15 archivable test investigations
- Confirmed mechanism stability, context enrichment focus

**2026-01-01:** Investigation completed
- Status: Complete
- Key outcome: Spawn mechanism is stable; future work is context/observability
