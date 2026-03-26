## Summary (D.E.K.N.)

**Delta:** The stall tracker only marks a session stalled when two unchanged token samples are separated by at least the threshold, so normal 30 second polling does not accumulate toward the configured 3 minute stall window.

**Evidence:** `pkg/daemon/stall_tracker.go` resets the stored snapshot on every `Update`, and an empirical `go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go` probe showed repeated 1 second unchanged updates never stall while a single 4 second gap does.

**Knowledge:** The token-based tracker is advisory only and currently feeds dashboard/status surfacing rather than remediation, while a separate 15 minute phase-timeout path also sets the same `IsStalled` flag.

**Next:** Escalate to architect review for the hotspot daemon/status path so stall semantics and the current `execution` vs `opencode` token type split can be corrected together.

**Authority:** architectural - the findings cross `pkg/daemon`, `cmd/orch`, and dashboard attention behavior in a hotspot area.

---

# Investigation: Stall Tracker Pkg Daemon Stall

**Question:** How does `pkg/daemon/stall_tracker.go` decide an agent is stalled, and what code path turns that detection into visible or actionable behavior?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Stall detection only watches input plus output token growth

**Evidence:** `Update`, `IsStalled`, and `GetStallDuration` all reduce the session state to `InputTokens + OutputTokens`; `ReasoningTokens`, `CacheReadTokens`, and `TotalTokens` are ignored. The first sample for a session always returns `false`, and any later sample with a larger sum clears stall detection.

**Source:** `pkg/daemon/stall_tracker.go:43`, `pkg/daemon/stall_tracker.go:80`, `pkg/daemon/stall_tracker.go:130`

**Significance:** The tracker defines progress narrowly: an agent can be busy in other ways, but it only counts as making progress if the combined input/output token total increases.

---

### Finding 2: The threshold measures gap between samples, not sustained inactivity

**Evidence:** `Update` reads the previous snapshot, immediately overwrites it with `Timestamp: now`, then compares `now.Sub(prev.Timestamp)` only on the stale-token path. In a probe run with a 3 second threshold, unchanged updates at 1 second, 2 seconds, and 3 seconds all returned `stalled=false`, while a later unchanged update after a 4 second gap returned `stalled=true`. `GetStallDuration` immediately after each `Update` was near zero because the timestamp had already been reset.

**Source:** `pkg/daemon/stall_tracker.go:55`, `pkg/daemon/stall_tracker.go:58`, `pkg/daemon/stall_tracker.go:74`, `pkg/daemon/stall_tracker.go:151`, `./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go`, `go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go`

**Significance:** With the dashboard polling `/api/agents` every 30 seconds and the global threshold set to 3 minutes, the current implementation does not accumulate six idle polls into a stall; it only trips if a single poll gap exceeds 3 minutes.

---

### Finding 3: Detection currently drives advisory surfacing, not automatic recovery

**Evidence:** The shared tracker is created once with `daemon.NewStallTracker(3 * time.Minute)`. Both `orch status` and the dashboard `/api/agents` handler call `globalStallTracker.Update(...)` and set `agent.IsStalled = true` when it returns true. Status formatting renders this as `⚠️ STALLED`. Separately, the dashboard also marks agents stalled after 15 minutes without a phase update, and the attention collector only escalates stuck agents for human review; it does not auto-kill or restart them.

**Source:** `cmd/orch/serve_agents_status.go:222`, `cmd/orch/status_cmd.go:349`, `cmd/orch/serve_agents_handlers.go:183`, `cmd/orch/serve_agents_handlers.go:436`, `cmd/orch/status_format.go:334`, `pkg/attention/stuck_collector.go:121`

**Significance:** The code path from detection to action ends in observability and prioritization. Today the tracker changes what humans see and what enters attention queues, but it does not trigger remediation on its own.

---

## Synthesis

**Key Insights:**

1. **Stall timing is edge-triggered, not cumulative** - Findings 1 and 2 together show that the tracker treats a stall as one long gap between samples rather than a run of unchanged samples, which is why normal polling fails to cross the 3 minute threshold.

2. **One flag mixes two meanings** - Finding 3 shows `IsStalled` can mean either token stagnation or 15 minute phase stagnation, so downstream consumers cannot tell which signal actually fired.

3. **The tracker is currently observational** - Even when stall detection fires, the system only surfaces a warning or attention item; there is no direct recovery path in the traced code.

**Answer to Investigation Question:**

`pkg/daemon/stall_tracker.go` declares a session stalled when a new sample has the same `InputTokens + OutputTokens` total as the prior sample and the time gap between those two samples is at least `stallThreshold` (default 3 minutes). The primary callers are `orch status` and `/api/agents`, which use `Update` to set `agent.IsStalled`; that flag is then rendered as `⚠️ STALLED` and contributes to stuck-attention collection. The important limitation is that the current implementation resets its timestamp every sample, so steady 30 second polling does not accumulate into a 3 minute stall window.

---

## Structured Uncertainty

**What's tested:**

- ✅ Repeated unchanged samples under the threshold do not become stalled cumulatively (verified with `go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go`)
- ✅ A single unchanged sample gap longer than the threshold returns `stalled=true` (verified by the same probe's 4 second gap)
- ✅ The attention collector consumes `IsStalled` as an advisory escalation signal rather than a recovery action (verified with `go test ./pkg/attention` and code inspection of `pkg/attention/stuck_collector.go`)

**What's untested:**

- ⚠️ End-to-end dashboard behavior with a live OpenCode session was not exercised; conclusions about `/api/agents` come from the handler code path rather than browser interaction.
- ⚠️ No production daemon logs were sampled, so frequency of false negatives in real usage is inferred from the documented 30 second dashboard poll interval.
- ⚠️ The compile failures from `go test ./pkg/daemon ./cmd/orch ./pkg/attention` were observed, but the broader migration intent behind the `execution` type split was not investigated.

**What would change this:**

- Finding 2 would be wrong if another caller preserves an older snapshot timestamp instead of calling `Update` on every unchanged poll.
- The action-path conclusion would change if another code path outside the traced files auto-restarts or abandons stalled agents based on `IsStalled`.
- The normal-polling inference would change if the dashboard no longer polls `/api/agents` on the documented 30 second cadence.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Redesign stall detection around last token change semantics and untangle the current token-type split before changing dashboard behavior | architectural | The fix crosses tracker logic, command/dashboard callers, and hotspot observability paths |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Architect the stall signal as a cross-component contract** - Redefine the tracker around last observed token change and split token-stall vs phase-stall into distinct signals before any implementation patch.

**Why this approach:**
- It fixes the core semantic bug from Finding 2 instead of layering more thresholds onto the current sample-gap behavior.
- It gives downstream consumers a clear distinction between token stagnation and missing phase updates.
- It creates one coordinated place to resolve the observed `execution` vs `opencode` token type mismatch that currently breaks relevant builds.

**Trade-offs accepted:**
- Immediate tactical patching is deferred.
- That delay is acceptable because the affected files are in a hotspot area and the warning signal already feeds only advisory surfaces.

**Implementation sequence:**
1. Define the desired stall contract: what counts as progress, what timestamp persists, and which caller owns sampling cadence.
2. Separate token-stall from phase-stall in the API/status model so one flag does not carry two meanings.
3. Update tests and type plumbing together so the tracker, callers, and assertions compile against the same token/status types.

### Alternative Approaches Considered

**Option B: Patch only `Update` to preserve last-change timestamps**
- **Pros:** Smallest behavior fix, likely restores cumulative stall timing quickly.
- **Cons:** Leaves the shared `IsStalled` ambiguity and the current build/type mismatch unresolved.
- **When to use instead:** If an emergency hotfix is needed before broader status-signal cleanup.

**Option C: Keep current tracker and tune thresholds higher or lower**
- **Pros:** No structural code change.
- **Cons:** Does not address the sample-gap bug from Finding 2, so threshold tuning only changes how long a poll outage must last before the warning appears.
- **When to use instead:** Only if the intended behavior really is detect missing polls rather than detect no token progress.

**Rationale for recommendation:** Option A is the only approach that resolves both the incorrect semantics and the overloaded downstream signal in a hotspot path.

---

### Implementation Details

**What to implement first:**
- Decide whether the source of truth should be `execution.TokenStats` or `opencode.TokenStats` for runtime stall detection.
- Add tests that simulate repeated sub-threshold polling to prove cumulative no-progress detection.
- Audit every `IsStalled` consumer before changing field meaning or shape.

**Things to watch out for:**
- ⚠️ `GetStallDuration` currently reports near-zero immediately after `Update` because the same timestamp reset affects both methods.
- ⚠️ Dashboard and CLI status paths both mutate the shared tracker, so caller cadence changes will change perceived stall behavior.
- ⚠️ A silent behavior change could alter attention queue volume if token-stall starts firing under normal polling.

**Areas needing further investigation:**
- Determine when the `execution` package became the tracker dependency while `cmd/orch` and tests still return `opencode` types.
- Check whether any other endpoints or collectors serialize `is_stalled` with assumptions about its current mixed semantics.
- Measure real-world `/api/agents` poll cadence and whether SSE paths could support a better stall clock.

**Success criteria:**
- ✅ Repeated unchanged sub-threshold polls eventually mark a session stalled once cumulative idle time exceeds the configured threshold.
- ✅ Token-stall and phase-stall can be distinguished in status output and API responses.
- ✅ `go test ./pkg/daemon ./cmd/orch ./pkg/attention` compiles and passes the relevant stall-tracking assertions.

---

## References

**Files Examined:**
- `pkg/daemon/stall_tracker.go` - core stall detection logic, thresholds, and cleanup behavior
- `pkg/daemon/stall_tracker_test.go` - intended unit-test coverage and current compile state
- `cmd/orch/status_cmd.go` - CLI status caller that updates the tracker
- `cmd/orch/serve_agents_handlers.go` - dashboard `/api/agents` caller and phase-based stall logic
- `cmd/orch/serve_agents_status.go` - global tracker configuration and documented dashboard poll cadence
- `cmd/orch/status_format.go` - rendering path from `IsStalled` to user-visible status
- `pkg/attention/stuck_collector.go` - downstream attention action path
- `pkg/execution/types.go` - current tracker type dependency
- `pkg/opencode/client_tokens.go` - runtime token type returned by callers

**Commands Run:**
```bash
# Verify working directory
pwd

# Create investigation artifact
kb create investigation stall-tracker-pkg-daemon-stall --orphan

# Run empirical probe against stall tracker behavior
go run ./.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/stall_probe.go

# Run relevant tests/builds
go test ./pkg/daemon ./cmd/orch ./pkg/attention

# Confirm attention collector package still passes independently
go test ./pkg/attention
```

**External Documentation:**
- None

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-stall-tracker-pkg-daemon-stall.md` - primary artifact for this investigation
- **Workspace:** `.orch/workspace/og-inv-stall-tracker-pkg-26mar-0ab2/` - session workspace with synthesis, brief, verification spec, and probe program

---

## Investigation History

**[2026-03-26 00:00]:** Investigation started
- Initial question: How the stall tracker decides an agent is stalled and what happens after detection.
- Context: Orchestrator requested threshold and code-path documentation for `pkg/daemon/stall_tracker.go`.

**[2026-03-26 00:00]:** Core logic and callers traced
- Confirmed that `Update` resets the snapshot timestamp on every sample and that both CLI status and dashboard handlers call it.

**[2026-03-26 00:00]:** Empirical probe run
- A small Go probe demonstrated that repeated unchanged sub-threshold polls never trip the threshold, but one longer gap does.

**[2026-03-26 00:00]:** Build validation run
- `go test ./pkg/daemon ./cmd/orch ./pkg/attention` exposed a current `execution` vs `opencode` type mismatch in the tracker tests and callers; `go test ./pkg/attention` still passed.
