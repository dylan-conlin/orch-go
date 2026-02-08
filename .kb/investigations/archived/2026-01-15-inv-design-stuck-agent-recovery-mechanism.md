<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Tiered recovery with advisory-first principle: Auto-resume works for rate limits, auto-respawn risks duplicate work, auto-abandon loses progress. Best approach is resume + escalation to human.

**Evidence:** Ghost visibility decision shows reversibility matters (filter > delete). Existing `orch resume` works for manual recovery. Daemon auto-completion works because completion is low-risk (agent explicitly said done). Recovery is higher-risk (guessing).

**Knowledge:** Different failure modes need different recovery: rate limit → resume works; server restart → resume may work (session on disk); context exhaustion → resume fails; infinite loop → resume perpetuates. One-size-fits-all recovery is wrong.

**Next:** Implement tiered recovery in daemon poll loop: (1) detect stuck agents, (2) attempt resume with rate limiting, (3) surface in Needs Attention if resume fails.

**Promote to Decision:** Actioned - patterns in agent lifecycle guide (advisory recovery)

---

# Investigation: Design Stuck Agent Recovery Mechanism

**Question:** When OpenCode workers get stuck after server restart or rate limit, should we auto-resume (send continue message), auto-respawn (new session same context), auto-abandon + re-queue, or use a tiered approach?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** Agent og-work-design-stuck-agent-15jan-faa3
**Phase:** Complete
**Next Step:** None - design ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Existing Resume Mechanism is Manual and Works

**Evidence:** `orch resume` command exists in `cmd/orch/resume.go`. It can:
- Resume by beads ID: `orch resume proj-123`
- Resume by workspace: `orch resume --workspace meta-orch-xyz`
- Resume by session: `orch resume --session ses_abc123`

The resume prompt tells the agent to re-read its SPAWN_CONTEXT.md and continue work.

**Source:** `cmd/orch/resume.go:90-100` (GenerateResumePrompt function)

**Significance:** The machinery for resume already exists. The question is whether to automate it.

---

### Finding 2: Stalled Detection Already Designed (Advisory-Only)

**Evidence:** Investigation `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` designed phase-based stalled detection:
- Signal: Phase unchanged for 15+ minutes
- Action: Surface in Needs Attention (advisory only)
- No auto-abandon, no auto-restart
- Single threshold (15 min), single signal (phase unchanged)

The design explicitly rejected auto-actions due to complexity spiral risk (Dec 27-Jan 2 rollback).

**Source:** `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md:44-49` (Success Criteria: "Advisory only - Surface in Needs Attention, don't auto-abandon")

**Significance:** Prior investigation established "advisory over automatic" as the right approach. Recovery design should follow this pattern.

---

### Finding 3: Ghost Visibility Decision Shows Reversibility Matters

**Evidence:** Decision `.kb/decisions/2026-01-15-ghost-visibility-over-cleanup.md` chose filtering over cleanup:
- "Ghosts can exist indefinitely. The fix is how we count and display them, not whether they exist."
- "Reversible? Yes (--all shows everything) vs No (deleted = gone)"

| Aspect | Cleanup | Filtering |
|--------|---------|-----------|
| Reversible? | No | Yes |
| Edge cases | Many | Few |
| Fights architecture? | Yes | No |

**Source:** `.kb/decisions/2026-01-15-ghost-visibility-over-cleanup.md:44-52`

**Significance:** Auto-respawn and auto-abandon are destructive (irreversible). Following the same logic, recovery should prefer non-destructive actions (resume, surface) over destructive ones (respawn, abandon).

---

### Finding 4: Different Failure Modes Need Different Recovery

**Evidence:** Analysis of failure modes reveals divergent recovery needs:

| Failure Mode | Session State | Recovery Strategy | Why |
|--------------|---------------|-------------------|-----|
| **Rate limit** | Valid, paused | Resume works | Just waiting for rate limit to clear |
| **Server restart** | On-disk, not in memory | Resume may work | Session loads from disk on first message |
| **Context exhaustion** | Valid but depleted | Resume fails | Agent can't process more tokens |
| **Infinite loop** | Valid but stuck | Resume perpetuates | Bad state is persistent |
| **Crash** | Lost | Respawn needed | No session to resume |

**Source:** Derived from `.kb/guides/agent-lifecycle.md` (four-layer state model) and daemon operation model

**Significance:** One-size-fits-all recovery is wrong. A tiered approach that tries the cheapest action first and escalates on failure is the right architecture.

---

### Finding 5: Daemon Already Has Completion Loop Architecture

**Evidence:** Daemon has separate completion loop (every 60s) that:
1. Polls for Phase: Complete comments
2. Verifies completion (check artifacts)
3. Closes beads issues
4. Releases pool slots

This is parallel to the spawn loop, not blocking it.

**Source:** `.kb/guides/daemon.md:259-299` (Completion Detection section), `.kb/models/daemon-autonomous-operation.md:46-58`

**Significance:** Recovery could fit into this architecture as a third loop: "Check for stuck agents → Attempt resume → Surface if stuck persists". The daemon is the right place for automated recovery.

---

### Finding 6: Principles Constrain Recovery Design

**Evidence:** Relevant principles from `~/.kb/principles.md`:

1. **Gate Over Remind** (but gates must be passable): Recovery should be automatic where safe, but destructive actions need human decision.

2. **Verification Bottleneck**: "The system cannot change faster than a human can verify behavior." Auto-respawn would create new sessions faster than humans can verify the original session's state.

3. **Session Amnesia**: Agents forget between sessions. Resume prompt must provide context. Current resume prompts point back to SPAWN_CONTEXT.md (good).

4. **Friction is Signal**: Stuck agents are friction - capture why they got stuck, don't just restart them.

**Source:** `~/.kb/principles.md` (Gate Over Remind:166-183, Verification Bottleneck:293-326, Session Amnesia:45-68, Friction is Signal:493-539)

**Significance:** Recovery automation has limits. Destructive recovery (respawn, abandon) requires human verification. Non-destructive recovery (resume) can be automated with rate limiting.

---

## Synthesis

**Key Insights:**

1. **Advisory-First is the Pattern** - Both stalled detection (Jan 8) and ghost visibility (Jan 15) chose advisory/visibility over automatic action. Recovery should follow this pattern: try non-destructive resume, then surface for human decision.

2. **Failure Mode Determines Recovery** - Rate limits recover via resume. Server restarts might recover via resume. Context exhaustion and infinite loops don't recover via resume. The system can't distinguish these reliably, so it must try resume first and escalate on failure.

3. **Daemon is the Right Location** - Recovery fits naturally into the daemon's poll-based architecture. A recovery loop parallel to spawn and completion loops provides the automation while keeping the system simple.

**Answer to Investigation Question:**

**Use a tiered approach** with advisory-first principle:

| Tier | Action | Condition | Destructive? | Automatic? |
|------|--------|-----------|--------------|------------|
| 1 | Resume | Idle >10min, no Phase: Complete | No | Yes (rate-limited) |
| 2 | Surface | Resume didn't help after 15min | No | Yes (visibility) |
| 3 | Human decision | Surfaced agent | Varies | No |

**Why not other options:**

- **Auto-resume only**: Doesn't handle cases where resume fails (context exhaustion, infinite loops)
- **Auto-respawn**: Irreversible, risks duplicate work if original session has partial commits
- **Auto-abandon + re-queue**: Irreversible, loses any uncommitted progress
- **No automation**: Manual resume works but doesn't scale; stuck agents accumulate

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch resume` sends continuation prompt (verified: read resume.go source, command exists)
- ✅ Daemon completion loop operates parallel to spawn loop (verified: daemon.md guide)
- ✅ Ghost visibility filtering implemented for concurrency and display (verified: decision document)

**What's untested:**

- ⚠️ 10-minute threshold for stuck detection (educated guess, may need tuning)
- ⚠️ Resume success rate for different failure modes (unknown without production data)
- ⚠️ Rate limiting (1 resume/hour) is the right frequency (untested)

**What would change this:**

- If resume has >90% success rate → could be more aggressive with auto-resume
- If resume causes problems (infinite loops worse after resume) → need smarter detection
- If stuck agents are rare (<5%) → manual resume may be sufficient

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Tiered Recovery with Advisory First

**Why this approach:**
- Follows established patterns (advisory over automatic)
- Non-destructive first (resume), escalate to human for destructive
- Fits into existing daemon architecture
- Preserves reversibility (no session deletion)

**Trade-offs accepted:**
- Some stuck agents won't recover (context exhaustion, infinite loops)
- Human intervention still required for complex cases
- ~15 minute latency before surfacing in dashboard

**Implementation sequence:**
1. **Stuck detection** - Add to daemon poll loop: identify agents idle >10min without Phase: Complete
2. **Auto-resume** - Attempt `orch resume` for stuck agents (rate-limited: 1/hour per agent)
3. **Surface in dashboard** - If still stuck after resume + 15min, add to Needs Attention

### Alternative Approaches Considered

**Option B: Full Automation (auto-resume → auto-respawn)**
- **Pros:** No human intervention needed, handles all failure modes
- **Cons:** Auto-respawn risks duplicate work, irreversible, violates reversibility principle
- **When to use instead:** If stuck agents become >30% of spawns and human capacity is limited

**Option C: Manual Only (visibility + dashboard)**
- **Pros:** Simplest, no automation risk, human always decides
- **Cons:** Doesn't scale, stuck agents accumulate, requires constant monitoring
- **When to use instead:** If stuck agents are <5% and automation causes problems

**Rationale for Tier 1 Recommendation:** Option A (tiered with advisory-first) provides automation where safe (resume is non-destructive) while preserving human decision for destructive actions (respawn, abandon). This matches established patterns from stalled detection and ghost visibility decisions.

---

### Implementation Details

**What to implement first:**
1. Stuck detection in daemon (highest impact, enables everything else)
2. Resume tracking (prevent infinite resume loops)
3. Needs Attention integration (surface stuck agents)

**Things to watch out for:**
- ⚠️ Resume may wake agents in bad state (infinite loop) - limit to 1 resume/hour
- ⚠️ Server restart may invalidate all sessions temporarily - don't mass-resume immediately
- ⚠️ Rate limit recovery may need delay (wait for rate limit to clear before resume)

**Areas needing further investigation:**
- What's the actual success rate of resume for different failure modes?
- Should resume include diagnostic message ("check if you're stuck in a loop")?
- Should recovery respect business hours (don't wake agents at 3am)?

**Success criteria:**
- ✅ Stuck agents surface in Needs Attention within 25 minutes of becoming stuck
- ✅ Resume automatically attempted for idle agents (rate-limited)
- ✅ No increase in duplicate work or lost progress
- ✅ Human can still manually resume/abandon/respawn from dashboard

---

## References

**Files Examined:**
- `cmd/orch/resume.go` - Existing resume command implementation
- `.kb/decisions/2026-01-15-ghost-visibility-over-cleanup.md` - Reversibility principle
- `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` - Stalled detection design
- `.kb/guides/daemon.md` - Daemon architecture and completion loop
- `.kb/guides/agent-lifecycle.md` - Four-layer state model
- `.kb/models/daemon-autonomous-operation.md` - Poll-spawn-complete cycle
- `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch architecture
- `~/.kb/principles.md` - Gate Over Remind, Verification Bottleneck, Session Amnesia, Friction is Signal

**Commands Run:**
```bash
# Find related knowledge
kb context "stuck agent recovery daemon"

# Find existing resume implementation
glob "**/resume*.go"

# Check for related issues
bd list --status=open | grep -i "stuck\|recovery\|resume"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-15-ghost-visibility-over-cleanup.md` - Filtering over cleanup establishes reversibility preference
- **Investigation:** `.kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md` - Stalled detection established advisory-only pattern
- **Issue:** `orch-go-vwjle` - Add stuck-agent detection and monitoring (open)

---

## Investigation History

**2026-01-15 10:00:** Investigation started
- Initial question: Design stuck agent recovery mechanism
- Context: Server restarts and rate limits cause agents to get stuck

**2026-01-15 10:30:** Context gathering complete
- Found existing resume command, stalled detection design, ghost visibility decision
- Identified four-layer state model and failure mode analysis

**2026-01-15 11:00:** Investigation completed
- Status: Complete
- Key outcome: Tiered recovery with advisory-first principle - auto-resume (non-destructive), then surface for human decision
