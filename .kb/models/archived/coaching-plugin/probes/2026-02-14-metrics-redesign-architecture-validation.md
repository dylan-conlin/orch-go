# Probe: Coaching Metrics Redesign Architecture Validation

**Model:** Coaching Plugin
**Date:** 2026-02-14
**Status:** Complete

---

## Question

Do the coaching plugin model's invariants and constraints hold under the proposed metrics redesign? Specifically:
1. Can killed metrics be fully removed without breaking the model's detection mechanism?
2. Does the new `session.created` event hook for auto-surfacing violate the "observation coupled to intervention" anti-pattern?
3. Can Go-side metric writing (completion_backlog) coexist with plugin-side writing to the same JSONL?

## What I Tested

### Test 1: Metric removal impact on behavioral proxy coverage
- Read coaching.ts (1831 lines) and mapped which invariants each metric serves
- Analyzed 1022 real metrics by type: action_ratio (381), analysis_paralysis (355), behavioral_variation (145), compensation_pattern (79), context_ratio (41), frame_collapse (13), circular_pattern (7), context_usage (1)
- Cross-referenced against model's "Detection Patterns (8 Behavioral Proxies)" table

### Test 2: Plugin hook availability for session.created
- Searched OpenCode fork plugin interface (~/Documents/personal/opencode/packages/plugin/src/index.ts)
- Confirmed `event` hook exists and receives all Bus events including `session.created`
- Verified `session.created` event includes `{ info: SessionInfo }` with `metadata` field

### Test 3: JSONL concurrent write safety
- coaching-metrics.jsonl is append-only with pruning to 1000 lines
- JSONL format inherently supports concurrent appenders (each line is independent)
- readCoachingMetrics() in serve_coaching.go reads line-by-line, tolerates malformed lines

## What I Observed

### Observation 1: Killed metrics are ALL behavioral proxies with high false-positive rates
- `action_ratio` and `analysis_paralysis` account for 72% of all events (736/1022) — volume indicates noise, not signal
- The model's "Behavioral proxies are the only detection mechanism" invariant (invariant #2) still holds — we're replacing noisy proxies with more specific ones, not changing the mechanism
- The 3 kept metrics (frame_collapse, circular_pattern, behavioral_variation) plus 2 new ones (spawn_without_context, completion_backlog) maintain behavioral proxy coverage while reducing noise

### Observation 2: session.created hook is available but has a subtlety
- The `event` hook fires for ALL events — plugin must filter by type
- `session.created` fires for both orchestrator AND worker sessions
- Worker detection at session creation time is possible IF `session.metadata.role` is set before the event fires
- **Critical finding:** The OpenCode server sets `metadata.role = 'worker'` during session creation (commit 459a1bfba), so the metadata IS available in the `session.created` event

### Observation 3: session.created injection is NOT the "observation coupled to intervention" anti-pattern
- The model documents this anti-pattern (Failure Mode 3) as: "Injection only fires from flushMetrics within tool.execute.after hook"
- Session-start injection is a DIFFERENT pattern: one-time injection at session creation, not coupled to continuous observation
- It's more like `config.instructions` (one-time setup) than `flushMetrics` (continuous observation)
- This EXTENDS the model — adds a third injection mechanism (session-start health summary)

### Observation 4: Go-side JSONL writing is safe
- appendFileSync in TypeScript and os.OpenFile with O_APPEND in Go both use atomic append on POSIX
- The pruneMetrics() function in the plugin runs at startup (not continuously), so no concurrent truncation risk
- readCoachingMetrics() skips malformed lines, so a partially-written line won't crash readers

## Model Impact

### Confirms:
1. **Invariant #2 (Behavioral proxies are the only detection mechanism)** — New metrics continue this pattern. spawn_without_context uses tool call sequence as proxy. completion_backlog uses phase state as proxy.
2. **Invariant #3 (Metrics persist, session state doesn't)** — No changes to persistence model. JSONL remains the durable layer.
3. **Invariant #5 (Worker detection caching is one-way)** — Unchanged. New session.created handler uses same metadata.role check.

### Extends:
1. **Injection mechanisms become 3, not 2.** Model currently documents two: `config.instructions` (static) and `client.session.prompt(noReply: true)` (runtime). Session-start injection adds a third pattern: one-time injection at session creation. Model should be updated to document this.
2. **Metric writers become 2, not 1.** Model currently assumes plugin is sole writer to coaching-metrics.jsonl. Go backend writing completion_backlog creates a second writer. Model's "Metrics persist" invariant still holds, but the constraint "JSONL file, pruned to last 1000 lines" needs clarification — pruning should only happen in one writer (plugin).
3. **Detection pattern count changes: 8 → 5 orchestrator + 4 worker.** Model's detection table needs update.

### Does NOT Contradict:
- None of the 6 critical invariants are violated by the redesign.
- The "observation coupled to intervention" anti-pattern is NOT reintroduced — session-start injection is a one-time event, not coupled to continuous observation.
