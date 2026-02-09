# Decision: Agent Completion - Three Orthogonal Lifecycle Dimensions

**Date:** 2026-02-07
**Status:** Accepted
**Authority:** Architectural (beads, orch, daemon boundaries)

---

## Context

Four investigations into agent completion issues (Dec 2025 - Feb 2026) converged on the same insight: what looks like "agents not completing" is actually three separate state machines being conflated.

| Investigation | Key Finding |
|---------------|-------------|
| 2025-12-22 | 41 "active" agents are ghost tmux windows — `orch status` counts windows not running agents |
| 2026-01-03 | SSE idle ≠ Phase: Complete (by design); cross-project `bd close` fails silently due to wrong directory context |
| 2026-01-08 | 25-28% "not completing" is a metrics artifact; true rate is ~89% after dedup and accounting for missing events |
| 2026-02-04 | Issue lifecycle and agent lifecycle are separate state machines; Phase: Complete is the mapping event |

The root cause across all four: codepaths assume a single "completion" concept, when actually three independent dimensions exist.

## Decision

**Agent completion involves three orthogonal state dimensions that must remain independent:**

### 1. Work Status (Agent Progress)
```
not_started → planning → implementing → testing → done
```
- **Tracked via:** Phase comments in beads (`Phase: Planning`, `Phase: Complete`)
- **Actor:** Agent
- **Authority:** Agent owns this dimension entirely

### 2. Verification Status (Quality Gates)
```
unverified → verification_passed | verification_failed → human_verified
```
- **Tracked via:** `orch complete` result, verification events
- **Actor:** Orchestrator (automated gates) + Human (subjective gates)
- **Authority:** System for automated, human for subjective

### 3. Issue Status (Beads Lifecycle)
```
open → in_progress → blocked → closed
```
- **Tracked via:** Beads issue status field
- **Actor:** Orchestrator (via `orch complete` → `bd close`)
- **Authority:** Orchestrator or daemon

### Mapping Between Dimensions

```
Phase: Complete     →  no issue state change  (agent declares done)
orch complete       →  verification_passed + issue closed  (orchestrator verifies + closes)
orch abandon        →  issue closed with reason  (abort path)
bd close (direct)   →  issue closed, event gap  (bypass path - needs fix)
```

### Specific Rules

1. **Phase: Complete is declaration, not closure** — Agents report `Phase: Complete` via `bd comment`. This is the agent's done signal. It does NOT close the beads issue.

2. **Verification is the bottleneck by design** — The orchestrator step between Phase: Complete and issue closure is intentional. It runs quality gates (SYNTHESIS.md, test evidence, build, git diff). This bottleneck is a feature.

3. **All close paths must emit events** — Currently, `bd close` (direct), zombie reconciliation, and force-close bypass `orch complete` and emit no events. This creates the metrics gap where true completion is ~89% but reported is ~72%.

4. **Stats must deduplicate by beads_id** — Completion events are counted per-event, not per-unique-completion. 26+ duplicates in a 7-day window inflate counts.

5. **Tmux windows ≠ active agents** — `orch status` must distinguish "tmux window exists" from "agent is running". Ghost windows accumulate when completion doesn't kill the window.

## Consequences

**Positive:**
- Clear mental model: "Which dimension am I looking at?"
- Prevents conflation bugs (e.g., SSE idle treated as completion)
- Enables separate optimization of each dimension
- UI can show each dimension independently (Work Graph badges)

**Negative:**
- More complex than "one completion flag"
- Three things to track instead of one
- Verification bottleneck remains (queue buildup when orchestrator is busy)

**Open items (from investigations, not yet implemented):**
- Fix `FallbackClose` to set `cmd.Dir = beads.DefaultDir`
- Fix `CloseIssue` to pass `WithCwd` to RPC client
- Add event emission to all close paths (not just `orch complete`)
- Stats deduplication by beads_id
- `orch clean --windows` to kill ghost tmux windows

## Synthesized From

- `.kb/investigations/2025-12-22-inv-40-agents-showing-as-active.md`
- `.kb/investigations/2026-01-03-inv-agents-going-idle-without-phase.md`
- `.kb/investigations/2026-01-08-inv-25-28-agents-not-completing.md`
- `.kb/investigations/archived/2026-02-04-inv-agents-own-declaration-via-bd.md`
