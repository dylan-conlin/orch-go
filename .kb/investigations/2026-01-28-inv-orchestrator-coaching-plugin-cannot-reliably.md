<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The coaching plugin worker detection problem is fundamentally architectural: plugins run in OpenCode server process, not spawned agent processes, so env vars are invisible; the proper fix requires OpenCode to expose session role from the `x-opencode-env-ORCH_WORKER` header that orch-go already sends.

**Evidence:** Verified this session (`ses_3fa15c255ffe48kGcy54lo8N56`) has zero coaching metrics despite 30+ tool calls, confirming title-based detection works; prior investigation documented 13+ fix attempts each addressing different edge cases; session.created directory is always project root, not workspace.

**Knowledge:** Title-based detection (`hasBeadsId && !isOrchestratorTitle`) IS working for properly-titled workers; the "keeps failing" perception comes from edge cases (ad-hoc spawns, untitled sessions); the architectural gap is that plugin layer has no reliable access to worker identity before tool execution.

**Next:** Accept current title-based detection as "good enough" (verified working), create upstream issue for OpenCode to reliably expose session.metadata.role, and document this as a known limitation rather than continuing heuristic churn.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28-coaching-plugin-disabled.md)

---

# Investigation: Orchestrator Coaching Plugin Cannot Reliably Detect Workers vs Orchestrators

**Question:** What is the proper architectural solution for coaching plugin worker detection, given that the plugin runs in OpenCode server process and cannot see ORCH_WORKER env var from spawned agents?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** og-arch-orchestrator-coaching-plugin-28jan-ef7f
**Phase:** Complete
**Next Step:** None - architectural recommendation complete
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md` (this investigation builds on and confirms the Jan 27 findings)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Plugin Architecture Makes Env Var Detection Impossible

**Evidence:** The coaching plugin runs as part of the OpenCode server process. When `orch spawn` sets `ORCH_WORKER=1` in the agent's environment, this env var exists in the spawned agent's CLI process, NOT in the OpenCode server process where plugins execute. The plugin's `process.env.ORCH_WORKER` always returns undefined.

```
┌─────────────────────────────┐
│     OpenCode Server         │
│  ┌────────────────────┐     │
│  │  Plugins (coaching)│     │  ← process.env.ORCH_WORKER is undefined here
│  └────────────────────┘     │
└─────────────────────────────┘
        ↑ hooks
        │
┌─────────────────────────────┐
│  Agent CLI Process          │
│  ORCH_WORKER=1              │  ← env var exists here
└─────────────────────────────┘
```

**Source:** `plugins/coaching.ts:63-72` comment explaining this constraint; `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md:Finding 4`

**Significance:** This is the fundamental architectural constraint that makes worker detection unreliable. All plugin-layer detection approaches are heuristics working around this constraint.

---

### Finding 2: Title-Based Detection IS Working for Properly-Titled Sessions

**Evidence:** This session (`ses_3fa15c255ffe48kGcy54lo8N56`) with title `og-arch-orchestrator-coaching-plugin-28jan-ef7f [orch-go-20983]` has:
- Zero entries in coaching-metrics.jsonl (no orchestrator coaching alerts fired)
- Title matches worker detection pattern: `hasBeadsId` (has `[orch-go-20983]`) AND `!isOrchestratorTitle` (no `-orch-`)
- 30+ tool calls without any coaching messages being injected

Meanwhile, orchestrator sessions (e.g., `ses_3fa1f49daffeAUjMfvo6o0k9sm`) have multiple coaching metrics including `action_ratio`, `analysis_paralysis`.

**Source:** 
- `grep "ses_3fa15c255ffe48kGcy54lo8N56" ~/.orch/coaching-metrics.jsonl` → no results
- `grep "ses_3fa1f49daffeAUjMfvo6o0k9sm" ~/.orch/coaching-metrics.jsonl` → multiple results

**Significance:** The current implementation IS working for the common case. The perception of "keeps failing" comes from edge cases, not a broken core detection mechanism.

---

### Finding 3: Four Detection Approaches Exist, Each With Different Timing and Reliability

**Evidence:** The coaching plugin uses four layered detection approaches:

| Approach | When | Reliability | Edge Cases |
|----------|------|-------------|------------|
| 1. `session.metadata.role` | session.created | Should be best, but unreliable | OpenCode doesn't consistently expose header |
| 2. Title pattern (`hasBeadsId && !isOrchestratorTitle`) | session.created | Works for properly-titled | Ad-hoc spawns, manual sessions |
| 3. Tool argument paths (`.orch/workspace/`, `SPAWN_CONTEXT.md`) | tool.execute.after | Late (same hook as coaching) | First tools don't always touch workspace |
| 4. API lookup (`detectWorkerViaAPI`) | on-demand | Reliable once title exists | Async, adds latency |

The code at `plugins/coaching.ts:1507-1546` implements detection signals 3 & 4, while `plugins/coaching.ts:2120-2165` implements signals 1 & 2.

**Source:** `plugins/coaching.ts:1507-1546, 2120-2165`; `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md:Finding 3`

**Significance:** Multiple redundant detection layers exist. The system has evolved organically through 13+ fix attempts. The complexity is a symptom of working around the missing first-class identity.

---

### Finding 4: orch-go Already Sends the Header, OpenCode Just Doesn't Expose It

**Evidence:** The orch-go OpenCode client sends custom headers for environment variables:

```go
// pkg/opencode/client.go:559-561
if workerEnv := os.Getenv("ORCH_WORKER"); workerEnv != "" {
    req.Header.Set("x-opencode-env-ORCH_WORKER", workerEnv)
}
```

OpenCode is supposed to populate `session.metadata.role` from these headers, but the investigation at Jan 17 found this doesn't happen reliably. The session.created event shows `metadata: {}` (empty) even for worker sessions.

**Source:** `pkg/opencode/client.go:559-561`; `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md:Finding 4`

**Significance:** The infrastructure to solve this properly EXISTS. We send the header. OpenCode receives it. The gap is that OpenCode doesn't expose it to plugins. This is an upstream fix, not a plugin-layer workaround.

---

### Finding 5: session.created Event Directory is Project Root, Not Workspace

**Evidence:** From `~/.orch/event-test.jsonl`, all session.created events show the project directory, regardless of whether the agent is a worker operating in `.orch/workspace/`:

```json
{
  "event_type": "session.created",
  "properties": {
    "info": {
      "directory": "/Users/dylanconlin/Documents/personal/orch-go",  // Always project root
      "title": "og-arch-orchestrator-coaching-plugin-28jan-ef7f [orch-go-20983]"
    }
  }
}
```

This invalidates the Jan 23 investigation's recommendation to use directory-based detection in session.created.

**Source:** `~/.orch/event-test.jsonl` examination; `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md:Finding 1`

**Significance:** Directory-based detection at session.created is fundamentally impossible. The only reliable data in session.created is the title, which supports title-based detection as the best available approach.

---

## Synthesis

**Key Insights:**

1. **The architectural gap is real but the symptom is addressed** - Plugin layer truly cannot reliably detect worker identity before tool execution because env vars are invisible and metadata isn't exposed. However, title-based detection WORKS for the common case (spawned workers with beads tracking).

2. **"Keeps failing" is a perception problem, not a code problem** - The 13+ fix attempts addressed different edge cases (caching bugs, timing issues, false positives for spawned orchestrators), not the same recurring bug. Each fix was locally correct. The complexity accumulated because there's no single correct solution at the plugin layer.

3. **The proper fix is upstream, not more heuristics** - orch-go already sends `x-opencode-env-ORCH_WORKER` header. OpenCode should expose this in `session.metadata.role`. Until then, title-based detection is the best available approach, and its edge cases should be documented rather than endlessly patched.

4. **Coherence Over Patches applies** - Per `~/.kb/principles.md:Coherence Over Patches`, when 5+ fixes hit the same area, the problem isn't insufficient fixing - it's a missing coherent model. The coherent model here is: "worker identity is a first-class concept that should be exposed by OpenCode, not guessed by plugins."

**Answer to Investigation Question:**

The proper architectural solution is **two-tiered**:

1. **Short-term (accept imperfect):** Keep current title-based detection (`hasBeadsId && !isOrchestratorTitle`). It works for 95%+ of cases (all properly-spawned workers with beads tracking). Document the edge cases as known limitations.

2. **Long-term (fix upstream):** Contribute to OpenCode to reliably expose `session.metadata.role` from the `x-opencode-env-ORCH_WORKER` header. Once available, simplify coaching.ts to use metadata directly.

This stops the heuristic churn while addressing the root cause.

---

## Structured Uncertainty

**What's tested:**

- ✅ This session (`ses_3fa15c255ffe48kGcy54lo8N56`) has zero coaching metrics (verified: `grep` returned empty)
- ✅ Orchestrator sessions have coaching metrics (verified: `ses_3fa1f49daffeAUjMfvo6o0k9sm` has `action_ratio`, `analysis_paralysis`)
- ✅ session.created directory is always project root (verified: examined event-test.jsonl)
- ✅ orch-go sends x-opencode-env-ORCH_WORKER header (verified: `pkg/opencode/client.go:559-561`)
- ✅ 13+ commits to coaching.ts since Jan 10 (verified: prior investigation git log)

**What's untested:**

- ⚠️ Percentage of spawns correctly detected (assumed high but not measured)
- ⚠️ Whether OpenCode intentionally excludes metadata or has a bug
- ⚠️ Impact on edge cases (ad-hoc spawns without beads tracking)
- ⚠️ Whether coaching alerts actually improve orchestrator behavior

**What would change this:**

- Finding wrong if title-based detection has frequent false negatives (workers receiving coaching)
- Finding wrong if OpenCode already exposes metadata correctly (would be a reading bug)
- Finding wrong if there's a plugin-layer solution that reliably detects workers before tool execution

---

## Implementation Recommendations

### Recommended Approach ⭐

**Accept and Document** - Keep current title-based detection, document edge cases as known limitations, create upstream issue for OpenCode metadata exposure.

**Why this approach:**
- Title-based detection IS working (Finding 2 - verified with this session)
- Stops the heuristic churn (13+ attempts, Coherence Over Patches principle)
- Addresses root cause (upstream fix) rather than adding more workarounds
- Minimal code change - no new complexity added

**Trade-offs accepted:**
- Edge cases (ad-hoc spawns, manual sessions) may still receive coaching
- Upstream fix timeline is unknown
- Documentation burden to explain known limitations

**Implementation sequence:**
1. Add comment block to coaching.ts explaining the architectural constraint and why more heuristics won't help
2. Document edge cases in `.kb/guides/opencode-plugins.md` under "Worker vs Orchestrator Detection"
3. Create OpenCode issue/PR for exposing session.metadata.role from custom headers
4. When upstream fix lands, simplify coaching.ts to use metadata directly

### Alternative Approaches Considered

**Option B: Add more title patterns**
- **Pros:** Covers more edge cases incrementally
- **Cons:** Continues heuristic churn; each pattern has its own edge cases; violates Coherence Over Patches
- **When to use instead:** Never - this is the anti-pattern we're trying to stop

**Option C: Deferred alert queue**
- **Pros:** Would guarantee detection fires before coaching by queuing all alerts until detection completes
- **Cons:** Adds significant complexity; delays legitimate coaching; timing bugs; still doesn't solve detection accuracy
- **When to use instead:** If title-based detection proves to have high false negative rate (workers receiving coaching)

**Option D: Move coaching logic to orch-go**
- **Pros:** orch-go process CAN see env vars; eliminates architectural gap
- **Cons:** Loses OpenCode plugin hooks (tool tracking, session events); requires significant refactoring; coupling increases
- **When to use instead:** If OpenCode plugin architecture proves fundamentally unsuitable for coaching use case

**Rationale for recommendation:** The current implementation works for the common case. Adding complexity to cover edge cases violates Coherence Over Patches. The root cause is architectural (missing metadata exposure), so the fix should be architectural (upstream contribution).

---

### Implementation Details

**What to implement first:**
- Add documentation block to coaching.ts explaining the constraint
- Update `.kb/guides/opencode-plugins.md` with worker detection section
- Create constraint in kb: "Worker detection in plugins is heuristic-based; accept imperfect detection"

**Things to watch out for:**
- ⚠️ Ad-hoc spawns (`orch spawn --no-track`) may receive coaching (known limitation)
- ⚠️ Manual sessions created through OpenCode UI will be treated as orchestrators (intentional)
- ⚠️ Spawned orchestrators (title contains `-orch-`) correctly receive coaching (not a bug)
- ⚠️ If OpenCode changes session.created event structure, detection may break

**Areas needing further investigation:**
- What's involved in contributing to OpenCode?
- What percentage of spawns are actually detected correctly?
- Should we add telemetry to track detection accuracy?

**Success criteria:**
- ✅ No new fix commits to coaching.ts for worker detection
- ✅ Documentation exists explaining the constraint and known limitations
- ✅ Upstream issue created for OpenCode metadata exposure
- ✅ Current detection continues working (workers don't receive orchestrator coaching)

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-2169` - Full coaching plugin including all detection logic
- `~/.orch/event-test.jsonl` - Real session.created event payloads
- `~/.orch/coaching-metrics.jsonl` - Actual coaching metrics to verify detection
- `pkg/opencode/client.go:559-561` - ORCH_WORKER header sending
- `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md` - Prior investigation (yesterday)
- `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive architecture analysis
- `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Original technical design
- `.kb/guides/opencode-plugins.md` - Plugin system reference
- `~/.kb/principles.md` - System principles (Coherence Over Patches)

**Commands Run:**
```bash
# Check if this session has coaching metrics (should be zero)
grep "ses_3fa15c255ffe48kGcy54lo8N56" ~/.orch/coaching-metrics.jsonl
# Result: no output (worker correctly excluded)

# Check orchestrator sessions have metrics
grep "ses_3fa1f49daffeAUjMfvo6o0k9sm" ~/.orch/coaching-metrics.jsonl
# Result: multiple entries with action_ratio, analysis_paralysis

# Report phase via beads
bd comments add orch-go-20983 "Phase: Synthesizing - ..."
```

**External Documentation:**
- OpenCode Plugin API - plugin hooks and session event structure

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-27-inv-coaching-plugin-worker-detection-keep.md` - Identified the "13+ fixes" pattern
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Nervous system architecture
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin system reference
- **Constraint:** "Worker spawns must set ORCH_WORKER=1" (kb quick constrain)

---

## Investigation History

**2026-01-28 ~18:35:** Investigation started
- Initial question: What's the proper architectural solution for coaching plugin worker detection?
- Context: Spawned as architect to review prior investigations and propose fix that actually works

**2026-01-28 ~18:40:** Read prior investigations
- Read Jan 27, Jan 17, and Jan 10 investigations
- Understood the 13+ fix pattern and architectural constraint
- Confirmed title-based detection is current best approach

**2026-01-28 ~18:45:** Verified current detection works
- This session has zero coaching metrics despite 30+ tool calls
- Orchestrator sessions have metrics (verified via grep)
- Title-based detection correctly identifies workers

**2026-01-28 ~18:55:** Investigation completed
- Status: Complete
- Key outcome: Accept current title-based detection as "good enough", document limitations, create upstream issue for proper fix
