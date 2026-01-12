<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The coaching plugin has 8 bugs and 2 abandonments because injection logic is architecturally coupled to metric collection - injection requires ephemeral session state but should depend on persistent metrics, creating restart brittleness and an entire class of "injection doesn't fire after X" bugs.

**Evidence:** Verified that metrics persist to JSONL file (plugins/coaching.ts:391-399), session state is in-memory Map (line 942), and injection only fires from flushMetrics within tool.execute.after hook (lines 1287, 1297), meaning injection cannot run independently of active observation despite metrics showing problems.

**Knowledge:** This is a canonical "Coherence Over Patches" violation - the 8 tactical fixes attempted to patch symptoms without addressing the fundamental design flaw that observation (passive) and intervention (active) are conflated into a single coupled code path instead of being separate concerns with separate lifecycles.

**Next:** Recommend architectural separation: extract injection into independent daemon that reads persistent metrics file and injects via OpenCode API, completely decoupled from plugin's observation code path - this eliminates the entire class of restart/state bugs and aligns with "Coherence Over Patches" principle.

**Promote to Decision:** recommend-yes - This establishes an architectural pattern: "separate observation from intervention" that should apply to any future behavioral monitoring/coaching features, preventing similar coupling bugs from emerging elsewhere in the system.

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

# Investigation: Review Design Coaching Plugin Injection

**Question:** Why does the coaching plugin injection system have 8 bugs in the 'serve' area, 2 abandoned attempts, and injection not firing after server restart despite metrics conditions being met?

**Started:** 2026-01-11 18:36
**Updated:** 2026-01-11 18:45
**Owner:** Agent og-arch-review-design-coaching-11jan-f74a
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: State/Behavior Coupling - Persistent Metrics + Ephemeral Injection

**Evidence:**
- Metrics are **persistent** (written to `~/.orch/coaching-metrics.jsonl`, survive restart)
- Session state is **in-memory** (`sessions` Map, lost on restart at plugins/coaching.ts:942)
- Injection logic is **coupled to metric collection** (only happens via `flushMetrics` at lines 1287, 1297)
- `flushMetrics` only called when: (1) tool calls happen AND (2) session state exists
- After server restart: metrics file shows "poor" status, but no session state exists yet, so injection never fires

**Source:**
- plugins/coaching.ts:942 (sessions Map - in-memory)
- plugins/coaching.ts:391-399 (writeMetric - persistent to JSONL)
- plugins/coaching.ts:1280-1299 (periodic flush triggers injection)
- plugins/coaching.ts:490-565 (flushMetrics function with injection logic)

**Significance:** The fundamental problem is architectural - injection is implemented as a side effect of metric collection, not as a separate concern that can operate independently. This means injection can only happen when the plugin is actively observing behavior, not when it should be reacting to already-observed problems.

---

### Finding 2: Two Responsibilities, One Code Path

**Evidence:**
- **Passive observation**: Track metrics (action_ratio, analysis_paralysis) via tool.execute.after hook
- **Active intervention**: Inject coaching messages via client.session.prompt()
- Both responsibilities share the same code path (flushMetrics function)
- Intervention is triggered by observation thresholds, but can't run without active observation
- Dashboard (serve_coaching.go) reads persistent metrics, but plugin injection reads ephemeral state

**Source:**
- plugins/coaching.ts:1100-1302 (tool.execute.after hook - observation)
- plugins/coaching.ts:546-562 (injection triggers within flushMetrics)
- serve_coaching.go:36-76 (reads metrics file independent of plugin state)

**Significance:** The architecture conflates two concerns that should be separate. This creates the "observer effect" problem - the act of observing enables intervention, but intervention should happen based on what was observed (persistent), not whether we're currently observing (ephemeral).

---

### Finding 3: High Churn Area - 8 Bugs from Treating Symptoms

**Evidence:**
- 2 abandoned debugging attempts (investigation files are empty templates, never filled)
- Multiple commits fixing specific symptoms: worker detection, message content detection, etc.
- Git log shows 8+ commits in coaching area over 2 days (Jan 10-11)
- Commits are tactical fixes ("fix: move worker detection", "fix: detect workers from message content") not architectural changes
- Issue abandoned 2x suggests agents hit complexity wall trying to debug without understanding design problem

**Source:**
- git log --grep="coaching" shows: 1bca4628, e9328a37, 4320188f, f6679954, 05859f38, 6e6503ae, baae3615, ddca8a36
- .kb/investigations/2026-01-11-inv-coaching-plugin-injection-*.md (3 files, all empty templates)
- bd show orch-go-rcah9 (2 abandoned attempts)

---

## Synthesis

**Key Insights:**

1. **Coherence Over Patches principle violation** - The 8 bugs in this area are not independent problems requiring 8 fixes. They are symptoms of a fundamental design flaw: injection is coupled to observation. When the ~/.kb/principles.md principle "Coherence Over Patches" says "If 5+ fixes hit the same area, recommend redesign not another patch," this is the canonical example.

2. **State/Behavior mismatch creates restart brittleness** - The plugin has THREE different lifecycles with no coordination: (1) Metrics file (persistent across restarts), (2) Plugin state (ephemeral, resets on restart), (3) OpenCode sessions (persistent until user closes). Injection depends on #2 but should depend on #1. This mismatch means "coaching broken after restart" is not a bug to fix - it's a design inevitability.

3. **The Observer Effect Problem** - In physics, observation affects the observed system. In this code, observation *enables* intervention. The plugin can only inject coaching when it's actively observing tool calls, not when it should be responding to already-observed patterns. This creates gaps: restart, plugin crash, OpenCode server restart, etc. all break the intervention loop despite metrics showing problems.

**Answer to Investigation Question:**

The coaching plugin has 8 bugs and 2 abandonments because the architecture conflates "observing behavior" with "responding to behavior" into a single coupled code path. Specifically: (Finding 1) metrics are persistent but injection requires ephemeral session state, (Finding 2) intervention is a side effect of metric collection not a separate responsibility, and (Finding 3) agents keep fixing symptoms (worker detection bugs, timing issues, etc.) without addressing the structural problem that injection can't run independently of observation. The correct fix is architectural separation: decouple metric collection (passive observation) from coaching injection (active intervention based on persistent metrics), not another tactical patch to make the current design work in one more edge case.

---

## Structured Uncertainty

**What's tested:**

- ✅ Metrics persist across restarts (verified: read plugins/coaching.ts:391-399, write to JSONL file)
- ✅ Session state is ephemeral (verified: plugins/coaching.ts:942, Map not persisted anywhere)
- ✅ Injection only fires from flushMetrics (verified: grepped for injectCoachingMessage calls, only in flushMetrics)
- ✅ flushMetrics requires session state (verified: called at lines 1287, 1297 within tool.execute.after hook)

**What's untested:**

- ⚠️ Whether separation would actually fix restart brittleness (architectural hypothesis, not implemented)
- ⚠️ Whether metrics file could be used as source of truth for injection (would need to handle duplicate injection prevention)
- ⚠️ Whether separate polling loop would have acceptable performance impact (no benchmark)

**What would change this:**

- Finding would be wrong if injection fired after restart without session state (testable: restart OpenCode server, check for injection)
- Finding would be wrong if there's a separate code path for injection we didn't find (verified via grep, only one path exists)
- Recommendation would be wrong if performance of polling metrics file exceeded acceptable latency (needs benchmarking if implemented)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Separate Coaching Injection from Metric Collection** - Extract injection logic into an independent daemon that reads persistent metrics file and injects coaching messages based on thresholds, completely decoupled from the observation/collection code path.

**Why this approach:**
- **Addresses Finding 1**: Injection runs based on persistent metrics, survives restart
- **Addresses Finding 2**: Two responsibilities get two code paths - observation (plugin) and intervention (daemon)
- **Addresses Finding 3**: Eliminates the class of bugs caused by coupling - no more "injection doesn't fire after X" bugs
- **Principle alignment**: Coherence Over Patches - redesign the architecture, don't patch the symptoms

**Trade-offs accepted:**
- Adds operational complexity (new daemon process to manage)
- Potential for duplicate injections if daemon doesn't track what it already injected (needs state)
- Slight latency between metric threshold breach and injection (polling interval vs real-time)
- These trade-offs are acceptable because they trade implementation complexity for architectural clarity

**Implementation sequence:**
1. **Create injection daemon** - New Go program that reads coaching-metrics.jsonl, calculates current session health, injects via OpenCode API when thresholds breached
2. **Add injection tracking** - Daemon maintains state of what it already injected (to prevent spam), either in memory or separate JSONL
3. **Extract injection logic from plugin** - Remove injectCoachingMessage calls from flushMetrics, keep metric writing only
4. **Integration test** - Restart OpenCode server with poor metrics, verify daemon injects on first tool call (shows decoupling works)
5. **Cleanup** - Remove now-unused injection code from plugin, update documentation

### Alternative Approaches Considered

**Option B: Add session state persistence to plugin**
- **Pros:** Smaller change, keeps injection in plugin
- **Cons:** Doesn't fix the fundamental coupling, adds complexity to plugin for state management, still breaks if plugin crashes mid-session
- **When to use instead:** If operational complexity of daemon is too high and restart brittleness is acceptable

**Option C: Inject on session.created hook instead of periodic flush**
- **Pros:** Catches sessions immediately on creation, simpler than daemon
- **Cons:** Doesn't solve restart problem (still requires session state), only helps with "new session after restart" not "existing session after restart"
- **When to use instead:** If the restart use case is rare enough to ignore and timing of injection matters more than reliability

**Rationale for recommendation:**

Option A (daemon) is the only approach that fundamentally decouples observation from intervention. Options B and C are patches that reduce symptoms without addressing the architectural problem identified in Finding 2. The principle "Coherence Over Patches" explicitly says this is the situation requiring redesign, not incremental fixes. The daemon approach has higher upfront cost but eliminates an entire class of bugs (8 fixed bugs + unknown future bugs in this area) and makes the system conceptually simpler: plugin observes, daemon intervenes.

---

### Implementation Details

**What to implement first:**
- **Injection daemon skeleton** - Go program with metrics file parsing (reuse serve_coaching.go logic)
- **Daemon lifecycle** - Integration with overmind (dev) and systemd (prod) for auto-start
- **OpenCode client** - HTTP client to call client.session.prompt() API (similar to orch spawn)
- **Worker session detection** - Don't inject into worker sessions (copy detectWorkerSession logic)

**Things to watch out for:**
- ⚠️ **Duplicate injection prevention** - Daemon needs state to track what sessions it already injected into (use injection-state.jsonl similar to metrics file)
- ⚠️ **OpenCode API availability** - Daemon should gracefully handle OpenCode server being down (retry with backoff)
- ⚠️ **Metrics file corruption** - Handle partial JSON lines, empty file, file doesn't exist yet
- ⚠️ **Session ID resolution** - Daemon needs to determine "current orchestrator session" - use most recent non-worker session from metrics
- ⚠️ **Polling frequency** - Too frequent = waste CPU, too slow = delayed coaching (recommend: 30s interval matching dashboard)

**Areas needing further investigation:**
- **Should daemon inject once per pattern or continuously until resolved?** (once = less spam, continuous = persistent reminder)
- **How to handle multiple orchestrator sessions running simultaneously?** (inject into all, or only "active" one based on recency?)
- **Should daemon have its own dashboard section showing injection history?** (useful for debugging, but adds scope)

**Success criteria:**
- ✅ After OpenCode server restart, daemon injects coaching within 30s when metrics show poor status
- ✅ No duplicate injections (daemon tracks injection state correctly)
- ✅ Worker sessions never receive coaching injections
- ✅ Dashboard shows coaching messages appearing in orchestrator session
- ✅ Zero crashes during 24-hour continuous run test
- ✅ Graceful degradation when OpenCode server is down (daemon doesn't crash, retries)

---

## References

**Files Examined:**
- plugins/coaching.ts - Plugin implementation with coupled observation/injection logic
- cmd/orch/serve_coaching.go - Dashboard API that reads metrics file independent of plugin state
- web/src/lib/stores/coaching.ts - Frontend state for coaching display
- .kb/investigations/2026-01-11-inv-pivot-coaching-plugin-two-frame.md - Implementation investigation showing two-frame design
- .kb/investigations/2026-01-11-inv-coaching-plugin-injection-*.md (3 files) - Empty templates from abandoned debugging attempts
- ~/.kb/principles.md - Referenced "Coherence Over Patches" principle

**Commands Run:**
```bash
# Find coaching-related commits
git log --oneline --grep="coaching" | head -20

# Show recent commits context
git log --oneline --since="2 days ago" | head -30

# Find beads issue details
bd show orch-go-rcah9

# Find serve-related files
glob "**/*serve*.go"

# Search for flushMetrics calls
grep -n "flushMetrics\(" plugins/coaching.ts
```

**External Documentation:**
- None referenced

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-11-inv-pivot-coaching-plugin-two-frame.md` - Shows two-frame design that was implemented
- **Workspace:** `.orch/workspace/og-feat-pivot-coaching-plugin-11jan-be2c/` - Workspace from implementation
- **Beads Issue:** `orch-go-rcah9` - Issue tracking this architectural review

---

## Investigation History

**[2026-01-11 18:36]:** Investigation started
- Initial question: Why does the coaching plugin injection system have 8 bugs in the 'serve' area, 2 abandoned attempts, and injection not firing after server restart?
- Context: Issue has been abandoned 2x without completion, suggesting agents hitting complexity wall without understanding fundamental problem

**[2026-01-11 18:40]:** Root cause identified
- Discovered state/behavior coupling: metrics persistent, session state ephemeral, injection coupled to observation code path
- Found that restart brittleness is a design inevitability, not a fixable bug

**[2026-01-11 18:45]:** Investigation completed
- Status: Complete
- Key outcome: Recommended architectural separation of observation (plugin) from intervention (daemon) to eliminate class of bugs caused by coupling
