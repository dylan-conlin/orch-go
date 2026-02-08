<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb reflect exists with robust synthesis detection (threshold 3+, issue creation at 10+), and daemon already runs periodic reflection saving to ~/.orch/reflect-suggestions.json. The gap is surfacing this data at session start.

**Evidence:** `orch session start` outputs only "Session started" + workspace path. No call to `daemon.LoadSuggestions()`. Dashboard exposes /api/reflect but interactive CLI doesn't consume it.

**Knowledge:** Infrastructure for proactive consolidation detection exists but is dormant for CLI users. The orchestrator-session.ts plugin handles session.created events for skill injection but not reflect surfacing. This is a small implementation gap, not a design gap.

**Next:** Add reflect suggestion surfacing to `runSessionStart()` in session.go when suggestions exist and have high-severity items.

**Promote to Decision:** recommend-no (tactical feature addition, not architectural)

---

# Investigation: Gate kb reflect to Surface Consolidation Opportunities Proactively

**Question:** How should kb reflect surface consolidation opportunities proactively, rather than reactively after 10-44 investigations accumulate?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** Implement surfacing in session.go
**Status:** Complete

---

## Findings

### Finding 1: kb reflect has mature synthesis detection with configurable thresholds

**Evidence:** 
- Synthesis candidates trigger at 3+ investigations on same topic (`findSynthesisCandidates` line 443: `if len(files) >= 3`)
- Issue auto-creation at 10+ investigations (`SynthesisIssueThreshold = 10` line 388)
- Daemon runs periodic reflection via `--reflect-interval` flag (default 60 minutes, line 137)
- All categories tracked: synthesis, promote, stale, drift, open, refine, skill-candidate

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:388-477`

**Significance:** The detection is working. Current orch-go has 54 dashboard investigations, 37 orchestrator, 37 spawn. The problem isn't detection - it's surfacing.

---

### Finding 2: Daemon saves reflect-suggestions.json but session start doesn't read it

**Evidence:**
- `daemon.go` line 136: `daemonReflect = true` (enabled by default on daemon exit)
- `daemon.go` line 195-197: `defer runReflectionAnalysis(daemonVerbose)` 
- `reflect.go` saves to `~/.orch/reflect-suggestions.json`
- `session.go` `runSessionStart()` has NO call to `daemon.LoadSuggestions()`
- Current `orch session start "test"` output: only "Session started: test" + workspace path

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/daemon.go:136,195-197`
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/session.go:82-131`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/reflect.go:176-198`

**Significance:** The data pipeline exists (daemon → file → LoadSuggestions) but session.go doesn't consume it. Small implementation gap.

---

### Finding 3: OpenCode plugin handles session.created but not reflect surfacing

**Evidence:**
- `orchestrator-session.ts` line 189-215: handles `session.created` event
- Plugin actions: (1) inject orchestrator skill, (2) auto-start `orch session start`
- NO interaction with reflect-suggestions.json or kb reflect

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts:189-215`

**Significance:** The plugin is a viable hook point but adding complexity there is unnecessary when session.go can simply check at start time.

---

### Finding 4: Dashboard already exposes /api/reflect endpoint

**Evidence:**
- `serve_learn.go` line 117-175: `handleReflect()` returns suggestions as JSON
- Reads from `~/.orch/reflect-suggestions.json`
- Dashboard has visibility but CLI orchestrator doesn't see the same data

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve_learn.go:117-175`

**Significance:** Web users get visibility via dashboard; CLI-only orchestrators don't. Session start is the natural injection point.

---

## Synthesis

**Key Insights:**

1. **Detection is solved, surfacing is the gap** - kb reflect has sophisticated clustering (semantic topic detection, threshold-based alerting, auto-issue creation). The problem is that interactive orchestrator sessions never see this data.

2. **Session start is the right hook point** - The orchestrator-session.ts plugin already auto-starts `orch session start`, making session.go the convergence point for both direct CLI usage and plugin-triggered starts.

3. **Severity-based gating is already implemented** - `SynthesisIssueThreshold = 10` and the daemon's issue creation logic show the pattern: low counts = informational, high counts = actionable. Session start should follow this pattern: show warning for topics with N+ investigations.

**Answer to Investigation Question:**

The proactive surfacing mechanism should be:

1. **On `orch session start`**: Check `~/.orch/reflect-suggestions.json` for synthesis candidates above warning threshold (suggest 10+ as "warning", matching existing issue threshold)
2. **Display summary**: Show "54 dashboard investigations need synthesis - consider `kb create guide dashboard`" 
3. **Non-blocking**: Just informational, doesn't gate session start
4. **Freshness check**: Skip if suggestions older than 24h (daemon should refresh)

This matches the "Gate Over Remind" principle - but here the "gate" is attention-grabbing output, not a blocker, because knowledge consolidation is advisory not mandatory.

---

## Structured Uncertainty

**What's tested:**

- ✅ kb reflect produces synthesis candidates correctly (ran `kb reflect --format json`, saw 54 dashboard investigations)
- ✅ Daemon periodic reflection saves to file (checked `~/.orch/reflect-suggestions.json`, timestamp 2026-01-08T00:37:47Z)
- ✅ Session start currently shows no suggestions (ran `orch session start "test"`, only got session started message)

**What's untested:**

- ⚠️ Implementation of LoadSuggestions call in session.go (proposed, not implemented)
- ⚠️ User impact of showing warnings at session start (might feel spammy)
- ⚠️ Integration with OpenCode plugin path (should work since plugin calls `orch session start`)

**What would change this:**

- If users find session-start warnings annoying, make it opt-in via `--reflect/--no-reflect` flag
- If freshness check causes false negatives, daemon could refresh suggestions more frequently

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Session-Start Surfacing

**Add reflect suggestions to `runSessionStart()` in session.go**

**Why this approach:**
- Single convergence point for both CLI and plugin-triggered sessions
- Non-invasive: add ~20 lines to existing function
- Uses existing infrastructure (daemon.LoadSuggestions)

**Trade-offs accepted:**
- Suggestions shown every session start (could be repetitive)
- Requires daemon to have run at least once to populate file

**Implementation sequence:**
1. Add `daemon.LoadSuggestions()` call at start of `runSessionStart()`
2. If suggestions exist and have high-count items, print warning summary
3. Add optional `--no-reflect` flag to suppress

### Alternative Approaches Considered

**Option B: OpenCode plugin event hook**
- **Pros:** Runs before session even starts, more "gate-like"
- **Cons:** Plugin complexity, harder to maintain, stdout issues with TUI
- **When to use instead:** If we wanted to block session start (we don't)

**Option C: Periodic CLI prompt (like `orch learn`)**
- **Pros:** Separate command, user-initiated
- **Cons:** Requires user to remember to run it, reactive not proactive
- **When to use instead:** Already exists via `orch daemon reflect`

**Rationale for recommendation:** Session-start surfacing is proactive (user doesn't need to remember), lightweight (just output), and uses existing infrastructure.

---

### Implementation Details

**What to implement first:**
- Add call to `daemon.LoadSuggestions()` in `runSessionStart()`
- Format synthesis warnings: "53 dashboard investigations need synthesis"
- Only show topics with 10+ investigations (match issue threshold)

**Things to watch out for:**
- ⚠️ Don't spam with all categories - focus on synthesis (actionable guide creation)
- ⚠️ Handle missing/stale file gracefully (no error, just skip)
- ⚠️ Keep output concise - max 3-5 lines

**Areas needing further investigation:**
- Should we add a `--verbose-reflect` to show all categories?
- Should freshness be configurable or fixed at 24h?

**Success criteria:**
- ✅ Running `orch session start "goal"` shows high-count synthesis warnings
- ✅ No regression in session start speed (<100ms added)
- ✅ Warning is actionable (tells user what to do)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go` - kb reflect implementation
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/session.go` - session start implementation
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/daemon.go` - daemon reflect integration
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/reflect.go` - reflect suggestions storage
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/orchestrator-session.ts` - OpenCode plugin

**Commands Run:**
```bash
# Test kb reflect output
kb reflect --format json | head -200

# Test session start behavior
orch session start "test"

# Check reflect-suggestions existence
ls -la ~/.orch/reflect-suggestions.json
cat ~/.orch/reflect-suggestions.json | head -50
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-daemon-hook-integration-kb-reflect.md` - Prior work on daemon integration
- **Decision:** Existing thresholds (10 for issues) established in kb-cli

---

## Investigation History

**2026-01-07 16:40:** Investigation started
- Initial question: How to gate kb reflect for proactive surfacing
- Context: Task from beads issue orch-go-ckum1

**2026-01-07 17:00:** Key findings documented
- Found synthesis detection is mature (3+ threshold, 10+ for issues)
- Found daemon saves to file but session.go doesn't load
- Identified session.go as convergence point

**2026-01-07 17:15:** Investigation completed
- Status: Complete
- Key outcome: Simple implementation - add LoadSuggestions call to runSessionStart()
